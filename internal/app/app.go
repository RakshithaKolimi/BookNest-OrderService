package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"

	"booknest-order-service/internal/events"
	ordergrpc "booknest-order-service/internal/grpc"
	"booknest-order-service/internal/repository/postgres"
	"booknest-order-service/internal/service"
)

type App struct {
	DB            *pgxpool.Pool
	RabbitConn    *amqp.Connection
	RabbitChannel *amqp.Channel
	OrderRepo     *postgres.OrderRepository
	OrderService  *service.OrderService
}

// New initializes the application dependencies and returns an App instance.
func New(ctx context.Context, rabbitMQURL string) (*App, error) {
	db, err := postgres.NewPool(ctx)
	if err != nil {
		return nil, err
	}

	conn, ch, err := events.ConnectRabbitMQ(rabbitMQURL)
	if err != nil {
		db.Close()
		return nil, err
	}

	orderRepo := postgres.NewOrderRepository(db)
	publisher := service.NewPublisherService(ch)
	orderService := service.NewOrderService(orderRepo, publisher)

	return &App{
		DB:            db,
		RabbitConn:    conn,
		RabbitChannel: ch,
		OrderRepo:     orderRepo,
		OrderService:  orderService,
	}, nil
}

func (a *App) AttachGRPCServer(server *grpc.Server) {
	ordergrpc.Register(server, a.OrderService)
}

func (a *App) Close() {
	if a.RabbitChannel != nil {
		_ = a.RabbitChannel.Close()
	}
	if a.RabbitConn != nil {
		_ = a.RabbitConn.Close()
	}
	if a.DB != nil {
		a.DB.Close()
	}
}
