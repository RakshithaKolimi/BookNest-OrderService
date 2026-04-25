package app

import (
	"context"
	"fmt"
	"os"

	"booknest-order-service/internal/config"
)

func Bootstrap() (context.Context, error) {
	if err := config.LoadEnvFile(".env"); err != nil {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	return context.Background(), nil
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
