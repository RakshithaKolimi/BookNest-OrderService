package events

import "time"

const EventOrderCreated = "order.created"

type OrderCreatedEvent struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	OccurredAt  time.Time `json:"occurred_at"`
	OrderID     string    `json:"order_id"`
	OrderNumber string    `json:"order_number"`
	UserID      string    `json:"user_id"`
	TotalPrice  float64   `json:"total_price"`
}
