# Pub/Sub Emulator Setup

This project provides a Go-based utility to automate the setup of Google Cloud Pub/Sub topics and subscriptions (including Dead Letter Topics) for local development using the Pub/Sub Emulator.

## Features

- Loads environment variables from a `.env` file.
- Creates main and dead-letter topics if they do not exist.
- Creates a pull subscription with dead-letter policy.
- Cleans up existing subscriptions for a fresh configuration.
- Uses structured logging for better observability.

## Requirements

- Go 1.20+
- [Google Cloud Pub/Sub Emulator](https://cloud.google.com/pubsub/docs/emulator)
- [godotenv](https://github.com/joho/godotenv/)

## Environment Variables

All configuration is handled via environment variables, typically loaded from a `.env` file at the root of the project. Example:

```env
PUBSUB_EMULATOR_LOG_LEVEL=DEBUG
PUBSUB_EMULATOR_ENV=local
GOOGLE_APPLICATION_CREDENTIALS=path-to-credentials
PUBSUB_EMULATOR_GOOGLE_LOCATION=location
PUBSUB_EMULATOR_HOST=localhost:8085
PUBSUB_EMULATOR_GOOGLE_PROJECT_ID=your-project-id
PUBSUB_EMULATOR_TOPIC_ID=your-topic-id
PUBSUB_EMULATOR_SUBSCRIPTION_ID=your-subscription-id
PUBSUB_EMULATOR_DLT_TOPIC_ID=your-dlt-topic-id
PUBSUB_WORKER_LOCAL_URL=worker-url
```

## Usage

### 1. Start the Pub/Sub Emulator

You can start the emulator using Docker:

```bash
docker run --rm -p 8085:8085 google/cloud-sdk:emulators gcloud beta emulators pubsub start --host-port=0.0.0.0:8085
```

Or using the gcloud CLI:

```bash
gcloud beta emulators pubsub start --host-port=localhost:8085
```

### 2. Configure Environment Variables

- Place your variables in a `.env` file as shown above.
- Alternatively, export them manually in your shell.

### 3. Run the Setup Utility

From the project directory, run:

```bash
go run main.go
```

```justfile
just watch
```

If you use `direnv` or `dotenv-cli`, your environment will be loaded automatically.

### 4. Verify

Check the emulator logs and the output of the utility to ensure topics and subscriptions are created.

## Notes

- **Push subscriptions are not supported by the Pub/Sub Emulator.** Only pull subscriptions are created.
- The utility will delete and recreate the subscription if it already exists, ensuring a clean state.
- Make sure the emulator is running and accessible at the host/port specified in your `.env`.

## Project Structure

```
setup-pusbub-emulator/
├── main.go
├── .env
├── credentials/
│   └── tech-spec-logisitcs-1dbe95df92ff.json
└── app/
    └── ... (logger and constants)
```

## Troubleshooting

- If you see `context deadline exceeded`, ensure the emulator is running and accessible.
- If you see `Unsupported push_endpoint`, remember that push subscriptions are not supported in the emulator.

## License

MIT License

---

**Author:** Esteban Varela  