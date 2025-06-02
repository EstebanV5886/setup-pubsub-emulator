package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
	"github.com/setup-pubsub-emulator/app"
)

func main() {
	ctx := context.Background()

	// --- Dotenv initialization ---
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting pwd")
	}
	path := filepath.Join(pwd, "../setup-pusbub-emulator/.env")
	err = godotenv.Load(path)
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	emulatorHost := os.Getenv(app.ENVVAR_EMULATOR_HOST)
	projectID := os.Getenv(app.ENVVAR_GOOGLE_PROJECT_ID)
	topicID := os.Getenv(app.ENVVAR_TOPIC_ID)      // Needs to be exported in shell
	subID := os.Getenv(app.ENVVAR_SUBSCRIPTION_ID) // Needs to be exported in shell
	dltID := os.Getenv(app.ENVVAR_DLT_TOPIC_ID)    // Needs to be exported in shell

	// --- Logger Setup ---
	logger, logLevel, err := app.NewAppLogger(
		os.Getenv(app.ENVVAR_ENV),
		os.Getenv(app.ENVVAR_LOG_LEVEL),
	)

	// Set default logger and level early
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(logLevel)

	if err != nil {
		if errors.Is(err, app.ErrUnknownLogLevel) {
			errMsg := fmt.Errorf(
				"failed to parse %s envvar, using log level %s, err: %s",
				app.ENVVAR_LOG_LEVEL,
				logLevel.String(),
				err,
			)
			logger.WarnContext(ctx, "Failed to parse log level envvar", slog.Any("error", errMsg))
		} else {
			logger.ErrorContext(ctx, "Failed to configure logger", slog.Any("error", err))
			return
		}
	}
	if emulatorHost == "" {
		log.Fatalln("PUBSUB_EMULATOR_HOST environment variable not set. Please set it to e.g., 'localhost:8085'")
	}
	if projectID == "" {
		log.Fatalln("PUBSUB_PROJECT_ID environment variable not set. Please set it to your dummy project ID.")
	}
	if topicID == "" || subID == "" || dltID == "" {
		log.Fatalln("PUB_SUB_TOPIC_ID, PUB_SUB_SUBSCRIPTION_ID, or DLT_TOPIC_ID environment variable(s) not set.")
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	defer client.Close()

	logger.InfoContext(ctx, fmt.Sprintf("Connecting to Pub/Sub emulator at %s for project %s", slog.String("emulatorHost", emulatorHost), slog.String("projectID", projectID)))

	// --- 1. Create Dead-Letter Topic (DLT) ---
	logger.InfoContext(ctx, "Creating DLT topic...", slog.String("dltID", dltID))
	dltTopic := client.Topic(dltID)
	exists, err := dltTopic.Exists(ctx)
	if err != nil {
		log.Fatalf("Failed to check DLT topic existence: %v", err)
	}
	if exists {
		logger.WarnContext(ctx, fmt.Sprintf("DLT topic '%s' already exists.", slog.String("dltID", dltID)))
	} else {
		_, err = client.CreateTopic(ctx, dltID)
		if err != nil {
			log.Fatalf("Failed to create DLT topic '%s': %v", dltID, err)
		}
		logger.InfoContext(ctx, fmt.Sprintf("DLT topic '%s' created successfully.", slog.String("dltID", dltID)))
	}

	// --- 2. Create Main Topic ---
	logger.InfoContext(ctx, fmt.Sprintf("Creating main topic '%s'...", slog.String("topicID", topicID)))
	topic := client.Topic(topicID)
	exists, err = topic.Exists(ctx)
	if err != nil {
		log.Fatalf("Failed to check main topic existence: %v", err)
	}
	if exists {
		logger.WarnContext(ctx, fmt.Sprintf("Main topic '%s' already exists.", slog.String("topicID", topicID)))
	} else {
		_, err = client.CreateTopic(ctx, topicID)
		if err != nil {
			log.Fatalf("Failed to create main topic '%s': %v", topicID, err)
		}
		logger.InfoContext(ctx, fmt.Sprintf("Main topic '%s' created successfully.", slog.String("topicID", topicID)))
	}

	// --- 3. Create Pull Subscription ---
	logger.InfoContext(ctx, fmt.Sprintf("Creating PULL subscription '%s' to topic '%s'...", slog.String("subID", subID), slog.String("topicID", topicID)))

	sub := client.Subscription(subID)
	exists, err = sub.Exists(ctx)
	if err != nil {
		log.Fatalf("Failed to check subscription existence: %v", err)
	}
	if exists {
		logger.WarnContext(ctx, fmt.Sprintf("Subscription '%s' already exists. Deleting and recreating for fresh config.", slog.String("subID", subID)))
		if err := sub.Delete(ctx); err != nil {
			log.Fatalf("Failed to delete existing subscription '%s': %v", subID, err)
		}
		logger.InfoContext(ctx, fmt.Sprintf("Existing subscription '%s' deleted.", slog.String("subID", subID)))
	}

	subConfig := pubsub.SubscriptionConfig{
		Topic: topic,
		DeadLetterPolicy: &pubsub.DeadLetterPolicy{
			DeadLetterTopic:     dltTopic.String(), // Full topic name: "projects/PROJECT_ID/topics/TOPIC_ID"
			MaxDeliveryAttempts: 10,
		},
		AckDeadline: 60 * time.Second, // Max time for worker to acknowledge
	}

	_, err = client.CreateSubscription(ctx, subID, subConfig)
	if err != nil {
		log.Fatalf("Failed to create push subscription '%s': %v", subID, err)
	}
	logger.InfoContext(ctx, fmt.Sprintf("PULL subscription '%s' created successfully.", slog.String("subID", subID)))

	logger.InfoContext(ctx, "Pub/Sub emulator setup complete.")
}
