package logging

import (
	"log/slog"
	"os"
	"strings"
)

const (
	defaultLevel  = "info"
	defaultFormat = "json"
)

func New(service string) *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))

	handlerOptions := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch format {
	case "", defaultFormat:
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	default:
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	}

	return slog.New(handler).With(
		slog.String("service", service),
	)
}

func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}

	return logger.With(slog.String("component", component))
}

func parseLevel(raw string) slog.Leveler {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "", defaultLevel:
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}
