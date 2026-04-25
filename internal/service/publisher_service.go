package service

import (
	"context"
	"encoding/json"

	"github.com/rabbitmq/amqp091-go"

	"booknest-order-service/internal/events"
)

type PublisherService struct {
	ch *amqp091.Channel
}

func NewPublisherService(ch *amqp091.Channel) *PublisherService {
	return &PublisherService{
		ch: ch,
	}
}

func (s *PublisherService) PublishOrderCreated(ctx context.Context, event events.OrderCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return events.PublishEvent(ctx, s.ch, events.EventOrderCreated, body)
}
