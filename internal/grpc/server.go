package grpc

import (
	"context"

	grpcserver "google.golang.org/grpc"
	grpcstatus "google.golang.org/grpc/status"

	orderv1 "booknest-order-service/gen/order/v1"
	"booknest-order-service/internal/service"
)

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	orderService *service.OrderService
}

func NewServer(orderService *service.OrderService) *Server {
	return &Server{orderService: orderService}
}

func Register(server grpcserver.ServiceRegistrar, orderService *service.OrderService) {
	orderv1.RegisterOrderServiceServer(server, NewServer(orderService))
}

func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	order, items, err := mapCreateOrderRequest(req)
	if err != nil {
		return nil, err
	}

	created, err := s.orderService.CreateOrder(ctx, order, items)
	if err != nil {
		return nil, grpcstatus.Errorf(mapErrorCode(err), "create order: %v", err)
	}

	return mapCreateOrderResponse(created), nil
}

func (s *Server) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, grpcstatus.Error(mapErrorCode(errInvalidArgument), "order_id is required")
	}

	order, items, err := s.orderService.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		return nil, grpcstatus.Errorf(mapErrorCode(err), "get order: %v", err)
	}

	return mapGetOrderResponse(order, items), nil
}

func (s *Server) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	userID, limit, offset, err := mapListOrdersRequest(req)
	if err != nil {
		return nil, err
	}

	orders, err := s.orderService.ListOrdersByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, grpcstatus.Errorf(mapErrorCode(err), "list orders: %v", err)
	}

	return mapListOrdersResponse(orders), nil
}

func (s *Server) ConfirmPayment(ctx context.Context, req *orderv1.ConfirmPaymentRequest) (*orderv1.ConfirmPaymentResponse, error) {
	userID, orderID, success, err := mapConfirmPaymentRequest(req)
	if err != nil {
		return nil, err
	}
	
	order, err := s.orderService.ConfirmPayment(ctx, userID, orderID, success)
	if err != nil {
		return nil, grpcstatus.Errorf(mapErrorCode(err), "confirm payment: %v", err)
	}

	return mapConfirmPaymentResponse(order), nil
}
