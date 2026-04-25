package events

import (
	"context"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConnectRabbitMQ(rabbitMQURL string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	if err := declareExchange(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}

func PublishEvent(ctx context.Context, ch *amqp.Channel, routingKey string, body []byte) error {
	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "booknest.events"
	}

	return ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

func ConsumeEvent(ch *amqp.Channel, queueName, routingKey string) (<-chan amqp.Delivery, error) {
	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "booknest.events"
	}

	queue, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if err := ch.QueueBind(
		queue.Name,
		routingKey,
		exchange,
		false,
		nil,
	); err != nil {
		return nil, err
	}

	return ch.Consume(
		queue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
}

func declareExchange(ch *amqp.Channel) error {
	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "booknest.events"
	}

	return ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}
