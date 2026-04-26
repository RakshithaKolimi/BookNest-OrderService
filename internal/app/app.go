package app

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"

	"booknest-order-service/internal/events"
	ordergrpc "booknest-order-service/internal/grpc"
	"booknest-order-service/internal/logging"
	"booknest-order-service/internal/repository/postgres"
	"booknest-order-service/internal/service"
)

type App struct {
	DB            *pgxpool.Pool
	RabbitConn    *amqp.Connection
	RabbitChannel *amqp.Channel
	OrderRepo     *postgres.OrderRepository
	OrderService  *service.OrderService
	Logger        *slog.Logger
}

// New initializes the application dependencies and returns an App instance.
func New(ctx context.Context, rabbitMQURL string, logger *slog.Logger) (*App, error) {
	appLogger := logging.WithComponent(logger, "app")

	db, err := postgres.NewPool(ctx)
	if err != nil {
		appLogger.Error("failed to initialize database pool", slog.Any("error", err))
		return nil, err
	}
	appLogger.Info("database pool initialized")

	conn, ch, err := events.ConnectRabbitMQ(rabbitMQURL)
	if err != nil {
		appLogger.Error("failed to connect to rabbitmq", slog.Any("error", err))
		db.Close()
		return nil, err
	}
	appLogger.Info("rabbitmq connection initialized")

	orderRepo := postgres.NewOrderRepository(db)
	publisher := service.NewPublisherService(ch)
	orderService := service.NewOrderService(orderRepo, publisher, logging.WithComponent(logger, "order_service"))

	return &App{
		DB:            db,
		RabbitConn:    conn,
		RabbitChannel: ch,
		OrderRepo:     orderRepo,
		OrderService:  orderService,
		Logger:        logger,
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
