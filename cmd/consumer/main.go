package main

import (
	"log"

	"booknest-order-service/internal/app"
	"booknest-order-service/internal/events"
)

func main() {
	ctx, err := app.Bootstrap()
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	rabbitMQURL, err := app.RequireEnv("RABBITMQ_URL")
	if err != nil {
		log.Fatal(err)
	}

	if err := events.ConsumeOrderCreated(ctx, rabbitMQURL); err != nil {
		log.Fatalf("consumer failed: %v", err)
	}
}
