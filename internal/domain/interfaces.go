package domain

import (
	"context"

	"booknest-order-service/internal/events"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order Order, items []OrderItem) (Order, error)
	GetOrder(ctx context.Context, orderID string) (Order, []OrderItem, error)
	ListOrdersByUser(ctx context.Context, userID string, limit, offset int32) ([]OrderWithItems, error)
	ListOrders(ctx context.Context, limit, offset int32) ([]OrderWithItems, error)
	UpdateOrder(ctx context.Context, order Order) (Order, error)
}

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, event events.OrderCreatedEvent) error
}
