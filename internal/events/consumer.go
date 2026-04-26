package events

import (
	"context"
	"log/slog"
)

const orderCreatedLogQueue = "order.created.log"

func ConsumeOrderCreated(ctx context.Context, rabbitMQURL string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	logger = logger.With(slog.String("component", "order_created_consumer"))

	conn, ch, err := ConnectRabbitMQ(rabbitMQURL)
	if err != nil {
		logger.Error("failed to connect to rabbitmq", slog.Any("error", err))
		return err
	}
	defer conn.Close()
	defer ch.Close()

	deliveries, err := ConsumeEvent(ch, orderCreatedLogQueue, EventOrderCreated)
	if err != nil {
		logger.Error(
			"failed to start event consumer",
			slog.Any("error", err),
			slog.String("queue", orderCreatedLogQueue),
			slog.String("routing_key", EventOrderCreated),
		)
		return err
	}

	logger.Info(
		"listening for events",
		slog.String("queue", orderCreatedLogQueue),
		slog.String("routing_key", EventOrderCreated),
	)

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping consumer", slog.Any("error", ctx.Err()))
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				logger.Warn("delivery channel closed", slog.String("queue", orderCreatedLogQueue))
				return nil
			}

			logger.Info(
				"received event",
				slog.String("routing_key", delivery.RoutingKey),
				slog.String("queue", orderCreatedLogQueue),
				slog.String("body", string(delivery.Body)),
			)
		}
	}
}
