package main

import (
	"log/slog"

	"booknest-order-service/internal/app"
	"booknest-order-service/internal/events"
	"booknest-order-service/internal/logging"
)

func main() {
	ctx, logger, err := app.Bootstrap()
	if err != nil {
		slog.Error("bootstrap failed", slog.Any("error", err))
		return
	}
	logger = logging.WithComponent(logger, "consumer")

	rabbitMQURL, err := app.RequireEnv("RABBITMQ_URL")
	if err != nil {
		logger.Error("missing required environment variable", slog.Any("error", err))
		return
	}

	if err := events.ConsumeOrderCreated(ctx, rabbitMQURL, logger); err != nil {
		logger.Error("consumer failed", slog.Any("error", err))
	}
}
