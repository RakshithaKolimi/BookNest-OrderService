package grpc

import (
	"context"
	"testing"

	orderv1 "booknest-order-service/gen/order/v1"
	"booknest-order-service/internal/domain"
	"booknest-order-service/internal/service"
)

type mockOrderRepository struct {
	getOrderFunc    func(ctx context.Context, orderID string) (domain.Order, []domain.OrderItem, error)
	listOrdersFunc  func(ctx context.Context, limit, offset int32) ([]domain.OrderWithItems, error)
	updateOrderFunc func(ctx context.Context, order domain.Order) (domain.Order, error)
}

func (m *mockOrderRepository) CreateOrder(ctx context.Context, order domain.Order, items []domain.OrderItem) (domain.Order, error) {
	return domain.Order{}, nil
}

func (m *mockOrderRepository) GetOrder(ctx context.Context, orderID string) (domain.Order, []domain.OrderItem, error) {
	if m.getOrderFunc != nil {
		return m.getOrderFunc(ctx, orderID)
	}
	return domain.Order{}, nil, nil
}

func (m *mockOrderRepository) ListOrdersByUser(ctx context.Context, userID string, limit, offset int32) ([]domain.OrderWithItems, error) {
	return nil, nil
}

func (m *mockOrderRepository) ListOrders(ctx context.Context, limit, offset int32) ([]domain.OrderWithItems, error) {
	if m.listOrdersFunc != nil {
		return m.listOrdersFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockOrderRepository) UpdateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	if m.updateOrderFunc != nil {
		return m.updateOrderFunc(ctx, order)
	}
	return order, nil
}

func TestServerCancelOrderMapsRequestAndResponse(t *testing.T) {
	order := domain.Order{
		ID:            "order-1",
		UserID:        "user-1",
		OrderNumber:   "BN-100",
		PaymentStatus: "COMPLETED",
		Status:        "CONFIRMED",
	}
	items := []domain.OrderItem{{BookID: "book-1", PurchaseCount: 1, PurchasePrice: 10, TotalPrice: 10}}

	repo := &mockOrderRepository{
		getOrderFunc: func(ctx context.Context, orderID string) (domain.Order, []domain.OrderItem, error) {
			return order, items, nil
		},
		updateOrderFunc: func(ctx context.Context, updated domain.Order) (domain.Order, error) {
			if updated.CancellationReason != "customer request" {
				t.Fatalf("expected cancellation reason to be persisted, got %q", updated.CancellationReason)
			}
			order = updated
			return order, nil
		},
	}

	svc := service.NewOrderService(repo, nil, nil)
	server := NewServer(svc)

	resp, err := server.CancelOrder(context.Background(), &orderv1.CancelOrderRequest{
		UserId:             "user-1",
		OrderId:            "order-1",
		CancellationReason: "customer request",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.GetStatus() != "CANCELLED" {
		t.Fatalf("expected cancelled status, got %s", resp.GetStatus())
	}
	if resp.GetCancellationReason() != "customer request" {
		t.Fatalf("expected cancellation reason to round-trip, got %q", resp.GetCancellationReason())
	}
}

func TestServerAdminUpdateOrderStatusMapsRequestAndResponse(t *testing.T) {
	order := domain.Order{
		ID:            "order-2",
		UserID:        "user-2",
		OrderNumber:   "BN-200",
		PaymentStatus: "REFUND_INITIATED",
		Status:        "PENDING",
	}

	repo := &mockOrderRepository{
		getOrderFunc: func(ctx context.Context, orderID string) (domain.Order, []domain.OrderItem, error) {
			return order, nil, nil
		},
		updateOrderFunc: func(ctx context.Context, updated domain.Order) (domain.Order, error) {
			if updated.PaymentStatus != "REFUNDED" {
				t.Fatalf("expected refunded payment status, got %s", updated.PaymentStatus)
			}
			order = updated
			return order, nil
		},
	}

	svc := service.NewOrderService(repo, nil, nil)
	server := NewServer(svc)

	resp, err := server.AdminUpdateOrderStatus(context.Background(), &orderv1.AdminUpdateOrderStatusRequest{
		OrderId:            "order-2",
		Status:             "CANCELLED",
		PaymentStatus:      "REFUNDED",
		CancellationReason: "out of stock",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.GetPaymentStatus() != "REFUNDED" {
		t.Fatalf("expected refunded payment status, got %s", resp.GetPaymentStatus())
	}
	if resp.GetCancellationReason() != "out of stock" {
		t.Fatalf("expected cancellation reason to round-trip, got %q", resp.GetCancellationReason())
	}
}

func TestServerListAllOrdersUsesRepository(t *testing.T) {
	repo := &mockOrderRepository{
		listOrdersFunc: func(ctx context.Context, limit, offset int32) ([]domain.OrderWithItems, error) {
			if limit != 20 {
				t.Fatalf("expected limit 20, got %d", limit)
			}
			if offset != 5 {
				t.Fatalf("expected offset 5, got %d", offset)
			}
			return []domain.OrderWithItems{{
				Order: domain.Order{
					ID:            "order-3",
					OrderNumber:   "BN-300",
					UserID:        "user-3",
					PaymentMethod: "UPI",
					PaymentStatus: "PENDING",
					Status:        "PENDING",
				},
				Items: []domain.OrderItem{{
					BookID:        "book-3",
					PurchaseCount: 2,
					PurchasePrice: 15,
					TotalPrice:    30,
				}},
			}}, nil
		},
	}

	svc := service.NewOrderService(repo, nil, nil)
	server := NewServer(svc)

	resp, err := server.ListAllOrders(context.Background(), &orderv1.ListAllOrdersRequest{
		Limit:  20,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.GetOrders()) != 1 {
		t.Fatalf("expected 1 order, got %d", len(resp.GetOrders()))
	}
	if resp.GetOrders()[0].GetPaymentMethod() != "UPI" {
		t.Fatalf("expected payment method to be preserved, got %s", resp.GetOrders()[0].GetPaymentMethod())
	}
}

func TestServerCancelOrderValidatesRequiredFields(t *testing.T) {
	svc := service.NewOrderService(&mockOrderRepository{}, nil, nil)
	server := NewServer(svc)

	_, err := server.CancelOrder(context.Background(), &orderv1.CancelOrderRequest{
		UserId:  "user-1",
		OrderId: "order-1",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}
