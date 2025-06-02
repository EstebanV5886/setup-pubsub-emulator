package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"go.opentelemetry.io/otel/trace"
)

// fmtErr returns a slog.GroupValue with keys "message" and "stacktrace".
// This will use the stacktrace of when logging occurred, not where the error was created.
func fmtErr(err error) slog.Value {
	var groupValues []slog.Attr

	groupValues = append(groupValues, slog.String("message", err.Error()))

	groupValues = append(groupValues, slog.Any("stacktrace", traceLines()))

	return slog.GroupValue(groupValues...)
}

var reTrace = regexp.MustCompile(`.*/slog/logger\.go.*\n`)

func traceLines() []string {
	stackInfo := make([]byte, 1024*1024)
	if stackSize := runtime.Stack(stackInfo, false); stackSize > 0 {
		traceLines := reTrace.Split(string(stackInfo[:stackSize]), -1)
		return strings.Split(traceLines[len(traceLines)-1], "\n\t")
	}

	return []string{}
}

var ErrUnknownLogLevel = errors.New("failed to parse log level")

func NewAppLogger(env, levelStr string) (*slog.Logger, slog.Level, error) {
	var err error
	logLevel := slog.LevelInfo

	if levelStr != "" {
		err = logLevel.UnmarshalText([]byte(levelStr))
		if err != nil {
			logLevel = slog.LevelInfo
			err = errors.Join(ErrUnknownLogLevel, err)
		}
	}

	logHandlerOpts := slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// Rename attribute keys to match Cloud Logging structured log format.
			switch attr.Key {
			case slog.LevelKey:
				attr.Key = "severity"
				// Map slog.Level string values to Cloud Logging LogSeverity.
				// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
				if level := attr.Value.Any().(slog.Level); level == slog.LevelWarn {
					attr.Value = slog.StringValue("WARNING")
				}
			case slog.TimeKey:
				attr.Key = "timestamp"
			case slog.MessageKey:
				attr.Key = "message"
			}

			if attr.Value.Kind() == slog.KindAny {
				switch v := attr.Value.Any().(type) {
				case error:
					attr.Value = fmtErr(v)
				}
			}

			return attr
		},
	}

	var handler slog.Handler
	if env == "local" {
		logHandlerOpts.AddSource = true

		handler = tint.NewHandler(os.Stderr, &tint.Options{
			AddSource:   logHandlerOpts.AddSource,
			Level:       logHandlerOpts.Level,
			ReplaceAttr: logHandlerOpts.ReplaceAttr,
			TimeFormat:  time.Kitchen,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stderr, &logHandlerOpts)
	}

	logger := slog.New(handlerWithSpanContext(handler))
	return logger, logLevel, err
}

func handlerWithSpanContext(handler slog.Handler) *spanContextLogHandler {
	return &spanContextLogHandler{Handler: handler}
}

// spanContextLogHandler is an slog.Handler which adds attributes from the span context
type spanContextLogHandler struct {
	slog.Handler
}

// Handle overrides slog.Handler's Handle method. This adds attributes from the span context to the slog.Record.
func (t *spanContextLogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Get the SpanContext from the golang Context.
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		// Add trace context attributes following Cloud Logging structured log format described
		// in https://cloud.google.com/logging/docs/structured-logging#special-payloads-fields
		record.AddAttrs(
			slog.Any("logging.googleapis.com/trace", s.TraceID()),
		)
		record.AddAttrs(
			slog.Any("logging.googleapis.com/spanId", s.SpanID()),
		)
		record.AddAttrs(
			slog.Bool("logging.googleapis.com/trace_sampled", s.TraceFlags().IsSampled()),
		)
	}
	return t.Handler.Handle(ctx, record)
}

func (t *spanContextLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return handlerWithSpanContext(t.Handler.WithAttrs(attrs))
}

func (t *spanContextLogHandler) WithGroup(name string) slog.Handler {
	return handlerWithSpanContext(t.Handler.WithGroup(name))
}
