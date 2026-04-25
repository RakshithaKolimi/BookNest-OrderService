package grpc

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	grpcstatus "google.golang.org/grpc/status"

	orderv1 "booknest-order-service/gen/order/v1"
	"booknest-order-service/internal/domain"
)

const (
	orderPendingStatus   = "PENDING"
	paymentPendingStatus = "PENDING"
)

func mapCreateOrderRequest(req *orderv1.CreateOrderRequest) (domain.Order, []domain.OrderItem, error) {
	if strings.TrimSpace(req.GetUserId()) == "" {
		return domain.Order{}, nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "user_id is required")
	}

	if strings.TrimSpace(req.GetPaymentMethod()) == "" {
		return domain.Order{}, nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "payment_method is required")
	}

	if len(req.GetItems()) == 0 {
		return domain.Order{}, nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "at least one order item is required")
	}

	items := make([]domain.OrderItem, 0, len(req.GetItems()))
	totalPrice := 0.0

	for _, item := range req.GetItems() {
		if item == nil {
			return domain.Order{}, nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "order item cannot be nil")
		}

		if strings.TrimSpace(item.GetBookId()) == "" {
			return domain.Order{}, nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "book_id is required")
		}

		if item.GetPurchaseCount() <= 0 {
			return domain.Order{}, nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "purchase_count must be greater than zero")
		}

		if item.GetPurchasePrice() <= 0 {
			return domain.Order{}, nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "purchase_price must be greater than zero")
		}

		lineTotal := float64(item.GetPurchaseCount()) * item.GetPurchasePrice()
		totalPrice += lineTotal

		items = append(items, domain.OrderItem{
			BookID:        item.GetBookId(),
			PurchaseCount: item.GetPurchaseCount(),
			PurchasePrice: item.GetPurchasePrice(),
			TotalPrice:    lineTotal,
		})
	}

	orderID, err := newUUID()
	if err != nil {
		return domain.Order{}, nil, grpcstatus.Errorf(mapErrorCode(err), "generate order id: %v", err)
	}

	return domain.Order{
		ID:            orderID,
		OrderNumber:   fmt.Sprintf("BN-%d", time.Now().UnixNano()),
		UserID:        req.GetUserId(),
		TotalPrice:    totalPrice,
		PaymentMethod: req.GetPaymentMethod(),
		PaymentStatus: paymentPendingStatus,
		Status:        orderPendingStatus,
	}, items, nil
}

func mapCreateOrderResponse(order domain.Order) *orderv1.CreateOrderResponse {
	return &orderv1.CreateOrderResponse{
		OrderId:       order.ID,
		OrderNumber:   order.OrderNumber,
		UserId:        order.UserID,
		TotalPrice:    order.TotalPrice,
		PaymentStatus: order.PaymentStatus,
		Status:        order.Status,
	}
}

func mapGetOrderResponse(order domain.Order, items []domain.OrderItem) *orderv1.GetOrderResponse {
	pbItems := make([]*orderv1.OrderItem, 0, len(items))
	for _, item := range items {
		pbItems = append(pbItems, &orderv1.OrderItem{
			BookId:        item.BookID,
			PurchaseCount: item.PurchaseCount,
			PurchasePrice: item.PurchasePrice,
			TotalPrice:    item.TotalPrice,
		})
	}

	return &orderv1.GetOrderResponse{
		OrderId:       order.ID,
		OrderNumber:   order.OrderNumber,
		UserId:        order.UserID,
		TotalPrice:    order.TotalPrice,
		PaymentStatus: order.PaymentStatus,
		Status:        order.Status,
		Items:         pbItems,
	}
}

func mapListOrdersRequest(req *orderv1.ListOrdersRequest) (userID string, limit, offset int32, err error) {
	userID = strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return "", 0, 0, grpcstatus.Error(mapErrorCode(errInvalidArgument), "user_id is required")
	}

	limit = req.GetLimit()
	if limit <= 0 {
		limit = 10 // default limit
	}

	// ensure non-negative offset, default to 0
	offset = max(req.GetOffset(), 0)

	return userID, limit, offset, nil
}

func mapListOrdersResponse(orders []domain.OrderWithItems) *orderv1.ListOrdersResponse {
	pbOrders := make([]*orderv1.GetOrderResponse, 0, len(orders))

	for _, order := range orders {
		pbOrders = append(pbOrders, mapGetOrderResponse(order.Order, order.Items))
	}

	return &orderv1.ListOrdersResponse{
		Orders: pbOrders,
	}
}

func mapConfirmPaymentRequest(req *orderv1.ConfirmPaymentRequest) (userID, orderID string, success bool, err error) {
	if strings.TrimSpace(req.GetUserId()) == "" {
		return "","",false, grpcstatus.Error(mapErrorCode(errInvalidArgument), "user_id is required")
	}

	if strings.TrimSpace(req.GetOrderId()) == "" {
		return "","",false, grpcstatus.Error(mapErrorCode(errInvalidArgument), "order_id is required")
	}			
	
	return req.GetUserId(), req.GetOrderId(), req.GetSuccess(), nil
}

func mapConfirmPaymentResponse(order domain.Order) *orderv1.ConfirmPaymentResponse {
	return &orderv1.ConfirmPaymentResponse{
		OrderId:       order.ID,
		PaymentStatus: order.PaymentStatus,
		Status:        order.Status,
	}
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
