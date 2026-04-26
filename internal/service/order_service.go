package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"booknest-order-service/internal/domain"
	"booknest-order-service/internal/events"
)

const (
	paymentPendingStatus   = "PENDING"
	paymentCompletedStatus = "COMPLETED"
	paymentFailedStatus    = "FAILED"
	paymentRefundedStatus  = "REFUNDED"
	paymentRefundInit      = "REFUND_INITIATED"
	orderConfirmedStatus   = "CONFIRMED"
	orderCompletedStatus   = "COMPLETED"
	orderCancelledStatus   = "CANCELLED"
	orderPaymentFailed     = "PAYMENT_FAILED"
	orderPendingStatus     = "PENDING"
)

var errOrderNotFound = errors.New("order not found")
var errInvalidPaymentRequest = errors.New("invalid argument")

type OrderService struct {
	repo      domain.OrderRepository
	publisher domain.EventPublisher
	logger    *slog.Logger
}

func NewOrderService(repo domain.OrderRepository, publisher domain.EventPublisher, logger *slog.Logger) *OrderService {
	if logger == nil {
		logger = slog.Default()
	}

	return &OrderService{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *OrderService) Checkout(ctx context.Context, userID string, input domain.CheckoutInput) (domain.OrderWithItems, error) {
	if strings.TrimSpace(userID) == "" {
		return domain.OrderWithItems{}, errInvalidPaymentRequest
	}

	order := domain.Order{
		PaymentMethod: strings.TrimSpace(input.PaymentMethod),
		UserID:        strings.TrimSpace(userID),
	}

	created, err := s.CreateOrder(ctx, order, input.Items)
	if err != nil {
		return domain.OrderWithItems{}, err
	}

	return s.getOrderWithItems(ctx, created.ID)
}

func (s *OrderService) CreateOrder(ctx context.Context, order domain.Order, items []domain.OrderItem) (domain.Order, error) {
	prepared, err := prepareOrderForCreate(order, items)
	if err != nil {
		return domain.Order{}, err
	}

	items = prepared.Items

	created, err := s.repo.CreateOrder(ctx, prepared.Order, items)
	if err != nil {
		s.logger.Error("failed to create order", slog.Any("error", err), slog.String("user_id", prepared.Order.UserID))
		return domain.Order{}, err
	}

	s.logger.Info(
		"order created",
		slog.String("order_id", created.ID),
		slog.String("order_number", created.OrderNumber),
		slog.String("user_id", created.UserID),
		slog.Int("item_count", len(items)),
	)

	if s.publisher != nil {
		event := events.OrderCreatedEvent{
			EventID:     created.ID,
			EventType:   events.EventOrderCreated,
			OccurredAt:  created.CreatedAt,
			OrderID:     created.ID,
			OrderNumber: created.OrderNumber,
			UserID:      created.UserID,
			TotalPrice:  created.TotalPrice,
		}

		if err := s.publisher.PublishOrderCreated(ctx, event); err != nil {
			s.logger.Error(
				"failed to publish order created event",
				slog.Any("error", err),
				slog.String("order_id", created.ID),
				slog.String("event_type", event.EventType),
			)
		} else {
			s.logger.Info(
				"published order created event",
				slog.String("order_id", created.ID),
				slog.String("event_type", event.EventType),
			)
		}
	}

	return created, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (domain.Order, []domain.OrderItem, error) {
	return s.repo.GetOrder(ctx, orderID)
}

func (s *OrderService) ListOrdersByUser(ctx context.Context, userID string, limit, offset int32) ([]domain.OrderWithItems, error) {
	return s.repo.ListOrdersByUser(ctx, userID, limit, offset)
}

func (s *OrderService) ListUserOrders(ctx context.Context, userID string, limit, offset int32) ([]domain.OrderWithItems, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errInvalidPaymentRequest
	}

	return s.repo.ListOrdersByUser(ctx, strings.TrimSpace(userID), normalizePageSize(limit), normalizeOffset(offset))
}

func (s *OrderService) ListAllOrders(ctx context.Context, limit, offset int32) ([]domain.OrderWithItems, error) {
	return s.repo.ListOrders(ctx, normalizePageSize(limit), normalizeOffset(offset))
}

func (s *OrderService) ConfirmPayment(ctx context.Context, userID, orderID string, success bool) (domain.Order, error) {
	if strings.TrimSpace(userID) == "" {
		return domain.Order{}, errInvalidPaymentRequest
	}

	if strings.TrimSpace(orderID) == "" {
		return domain.Order{}, errInvalidPaymentRequest
	}

	order, _, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	if order.UserID != userID {
		return domain.Order{}, errOrderNotFound
	}

	if success {
		order.PaymentStatus = paymentCompletedStatus
		order.Status = orderConfirmedStatus
	} else {
		order.PaymentStatus = paymentFailedStatus
		order.Status = orderPaymentFailed
	}

	return s.repo.UpdateOrder(ctx, order)
}

func (s *OrderService) ConfirmPaymentWithInput(ctx context.Context, userID string, input domain.PaymentConfirmInput) (domain.OrderWithItems, error) {
	updated, err := s.ConfirmPayment(ctx, userID, input.OrderID, input.Success)
	if err != nil {
		return domain.OrderWithItems{}, err
	}

	return s.getOrderWithItems(ctx, updated.ID)
}

func (s *OrderService) CancelOrder(ctx context.Context, userID string, input domain.OrderCancelInput) (domain.OrderWithItems, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(input.OrderID) == "" {
		return domain.OrderWithItems{}, errInvalidPaymentRequest
	}

	order, _, err := s.repo.GetOrder(ctx, input.OrderID)
	if err != nil {
		return domain.OrderWithItems{}, err
	}

	if order.UserID != strings.TrimSpace(userID) {
		return domain.OrderWithItems{}, errOrderNotFound
	}

	if err := validateOrderForCancellation(order, input.CancellationReason); err != nil {
		return domain.OrderWithItems{}, err
	}

	if shouldStartRefund(order.PaymentStatus) {
		order.PaymentStatus = paymentRefundInit
	}
	order.Status = orderCancelledStatus

	updated, err := s.repo.UpdateOrder(ctx, order)
	if err != nil {
		return domain.OrderWithItems{}, err
	}

	return s.getOrderWithItems(ctx, updated.ID)
}

func (s *OrderService) AdminUpdateOrderStatus(ctx context.Context, input domain.AdminOrderStatusUpdateInput) (domain.OrderWithItems, error) {
	if strings.TrimSpace(input.OrderID) == "" {
		return domain.OrderWithItems{}, errInvalidPaymentRequest
	}

	order, _, err := s.repo.GetOrder(ctx, input.OrderID)
	if err != nil {
		return domain.OrderWithItems{}, err
	}

	if err := validateAdminOrderStatusUpdate(order, input.Status, input.PaymentStatus, input.CancellationReason); err != nil {
		return domain.OrderWithItems{}, err
	}

	if status := strings.TrimSpace(input.Status); status != "" {
		order.Status = status
		if status == orderCancelledStatus && shouldStartRefund(order.PaymentStatus) {
			order.PaymentStatus = paymentRefundInit
		}
	}

	if paymentStatus := strings.TrimSpace(input.PaymentStatus); paymentStatus != "" {
		order.PaymentStatus = paymentStatus
	}

	updated, err := s.repo.UpdateOrder(ctx, order)
	if err != nil {
		return domain.OrderWithItems{}, err
	}

	return s.getOrderWithItems(ctx, updated.ID)
}

func (s *OrderService) getOrderWithItems(ctx context.Context, orderID string) (domain.OrderWithItems, error) {
	order, items, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return domain.OrderWithItems{}, err
	}

	return domain.OrderWithItems{
		Order: order,
		Items: items,
	}, nil
}

func normalizePageSize(limit int32) int32 {
	if limit <= 0 {
		return 10
	}

	return limit
}

func normalizeOffset(offset int32) int32 {
	if offset < 0 {
		return 0
	}

	return offset
}

func validateOrderForCancellation(order domain.Order, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("cancellation reason is required")
	}

	if order.Status == orderCancelledStatus {
		return errors.New("order is already cancelled")
	}

	if order.Status == orderCompletedStatus {
		return errors.New("completed orders cannot be cancelled")
	}

	return nil
}

func validateAdminOrderStatusUpdate(order domain.Order, nextStatus, nextPaymentStatus, reason string) error {
	nextStatus = strings.TrimSpace(nextStatus)
	nextPaymentStatus = strings.TrimSpace(nextPaymentStatus)

	if nextStatus == "" && nextPaymentStatus == "" {
		return errors.New("status or payment status is required")
	}

	if nextStatus != "" {
		if nextStatus != orderCompletedStatus && nextStatus != orderCancelledStatus {
			return errors.New("admin can only set order status to COMPLETED or CANCELLED")
		}

		if order.Status == orderCompletedStatus || order.Status == orderCancelledStatus {
			return errors.New("order is already finalized")
		}

		if nextStatus == orderCancelledStatus && strings.TrimSpace(reason) == "" {
			return errors.New("cancellation reason is required")
		}
	}

	if nextPaymentStatus != "" {
		if nextPaymentStatus != paymentRefundedStatus {
			return errors.New("admin can only set payment status to REFUNDED")
		}

		if order.PaymentStatus != paymentRefundInit {
			return errors.New("refund can only be completed after refund is initiated")
		}
	}

	return nil
}

func shouldStartRefund(paymentStatus string) bool {
	return paymentStatus == paymentCompletedStatus
}

func prepareOrderForCreate(order domain.Order, items []domain.OrderItem) (domain.OrderWithItems, error) {
	if strings.TrimSpace(order.UserID) == "" {
		return domain.OrderWithItems{}, errInvalidPaymentRequest
	}

	order.PaymentMethod = strings.TrimSpace(order.PaymentMethod)
	if order.PaymentMethod == "" {
		return domain.OrderWithItems{}, errInvalidPaymentRequest
	}

	if len(items) == 0 {
		return domain.OrderWithItems{}, errInvalidPaymentRequest
	}

	totalPrice := 0.0
	preparedItems := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.BookID) == "" || item.PurchaseCount <= 0 || item.PurchasePrice <= 0 {
			return domain.OrderWithItems{}, errInvalidPaymentRequest
		}

		if item.TotalPrice <= 0 {
			item.TotalPrice = float64(item.PurchaseCount) * item.PurchasePrice
		}
		totalPrice += item.TotalPrice
		preparedItems = append(preparedItems, item)
	}

	if strings.TrimSpace(order.ID) == "" {
		orderID, err := newUUID()
		if err != nil {
			return domain.OrderWithItems{}, err
		}
		order.ID = orderID
	}

	if strings.TrimSpace(order.OrderNumber) == "" {
		order.OrderNumber = fmt.Sprintf("BN-%d", time.Now().UnixNano())
	}

	if order.TotalPrice <= 0 {
		order.TotalPrice = totalPrice
	}

	if strings.TrimSpace(order.PaymentStatus) == "" {
		order.PaymentStatus = paymentPendingStatus
	}

	if strings.TrimSpace(order.Status) == "" {
		order.Status = orderPendingStatus
	}

	return domain.OrderWithItems{
		Order: order,
		Items: preparedItems,
	}, nil
}

func newUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}

	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	buf := make([]byte, 36)
	hex.Encode(buf[0:8], raw[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], raw[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], raw[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], raw[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], raw[10:16])

	return string(buf), nil
}
