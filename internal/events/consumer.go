package events

import (
	"context"
	"log"
)

const orderCreatedLogQueue = "order.created.log"

func ConsumeOrderCreated(ctx context.Context, rabbitMQURL string) error {
	conn, ch, err := ConnectRabbitMQ(rabbitMQURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer ch.Close()

	deliveries, err := ConsumeEvent(ch, orderCreatedLogQueue, EventOrderCreated)
	if err != nil {
		return err
	}

	log.Printf("listening for %s events on queue %s", EventOrderCreated, orderCreatedLogQueue)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}

			log.Printf("received %s event: %s", delivery.RoutingKey, string(delivery.Body))
		}
	}
}
