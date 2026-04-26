package main

import (
	"log/slog"
	"net"

	"google.golang.org/grpc"

	"booknest-order-service/internal/app"
	"booknest-order-service/internal/logging"
)

func main() {
	ctx, logger, err := app.Bootstrap()
	if err != nil {
		slog.Error("bootstrap failed", slog.Any("error", err))
		return
	}
	logger = logging.WithComponent(logger, "grpc_server")

	port := app.EnvOrDefault("GRPC_PORT", "50051")

	rabbitMQURL, err := app.RequireEnv("RABBITMQ_URL")
	if err != nil {
		logger.Error("missing required environment variable", slog.Any("error", err))
		return
	}

	application, err := app.New(ctx, rabbitMQURL, logger)
	if err != nil {
		logger.Error("application setup failed", slog.Any("error", err))
		return
	}
	defer application.Close()

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("listen failed", slog.Any("error", err), slog.String("port", port))
		return
	}

	// Create the gRPC server process that will handle incoming RPC requests.
	server := grpc.NewServer()

	// Register transport handlers here.
	application.AttachGRPCServer(server)

	logger.Info("gRPC server listening", slog.String("port", port))
	logger.Info("application initialized")

	// Start serving requests and keep the process running until it fails or is stopped.
	if err := server.Serve(lis); err != nil {
		logger.Error("gRPC serve failed", slog.Any("error", err))
	}
}
