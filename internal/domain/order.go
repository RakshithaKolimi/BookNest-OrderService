package domain

import "time"

type Order struct {
	ID            string
	OrderNumber   string
	UserID        string
	TotalPrice    float64
	PaymentMethod string
	PaymentStatus string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OrderItem struct {
	OrderID       string
	BookID        string
	PurchaseCount int32
	PurchasePrice float64
	TotalPrice    float64
}

type OrderWithItems struct {
	Order Order
	Items []OrderItem
}

type CheckoutInput struct {
	PaymentMethod string
	Items         []OrderItem
}

type PaymentConfirmInput struct {
	OrderID string
	Success bool
}

type OrderCancelInput struct {
	OrderID            string
	CancellationReason string
}

type AdminOrderStatusUpdateInput struct {
	OrderID            string
	Status             string
	PaymentStatus      string
	CancellationReason string
}
