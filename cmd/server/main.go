package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	"booknest-order-service/internal/app"
)

func main() {
	ctx, err := app.Bootstrap()
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	port := app.EnvOrDefault("GRPC_PORT", "50051")

	rabbitMQURL, err := app.RequireEnv("RABBITMQ_URL")
	if err != nil {
		log.Fatal(err)
	}

	application, err := app.New(ctx, rabbitMQURL)
	if err != nil {
		log.Fatalf("application setup failed: %v", err)
	}
	defer application.Close()

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}

	// Create the gRPC server process that will handle incoming RPC requests.
	server := grpc.NewServer()

	// Register transport handlers here.
	application.AttachGRPCServer(server)

	log.Printf("order service gRPC server listening on :%s", port)
	log.Printf("order repository and service initialized successfully")

	// Start serving requests and keep the process running until it fails or is stopped.
	if err := server.Serve(lis); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	}
}
