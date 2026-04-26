package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"booknest-order-service/internal/config"
	"booknest-order-service/internal/logging"
)

func Bootstrap() (context.Context, *slog.Logger, error) {
	if err := config.LoadEnvFile(".env"); err != nil {
		return nil, nil, fmt.Errorf("load .env: %w", err)
	}

	logger := logging.New("booknest-order-service")
	slog.SetDefault(logger)
	return context.Background(), logger, nil
}

func RequireEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%s environment variable is not set", key)
	}

	return value, nil
}

func EnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
