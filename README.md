# BookNest Order Service Starter Pack

This is a learning-first starter pack for extracting the BookNest order flow into its own microservice.

BookNest Order Service is a Go-based microservice that manages order creation and retrieval for the BookNest platform. It stores order data in PostgreSQL, exposes synchronous gRPC APIs for order operations, and publishes asynchronous order events through RabbitMQ so other services can react without tight coupling.

The goal is to help you learn in this order:

1. Define the service boundary
2. Create a standalone order service
3. Add gRPC contracts
4. Add event publishing with RabbitMQ

## What this starter pack includes

- Go service skeleton
- First `order.proto`
- PostgreSQL migrations for `orders` and `order_items`
- Docker Compose for PostgreSQL and RabbitMQ
- `.env.example`
- Step-by-step implementation checklist

## Prerequisites

### Concept prerequisites

You should be comfortable with these ideas before going too far:

- Service boundary: what belongs to the order service and what does not
- Synchronous vs asynchronous communication
- Eventual consistency: services may not all update at the exact same time
- Idempotency: processing the same event twice should not create duplicate side effects
- Ownership: one service should own one database schema/table set

### Coding prerequisites

- Go basics: structs, interfaces, packages, context
- REST basics from your current monolith
- SQL/PostgreSQL basics
- Docker and `docker compose`
- Logging and debugging application startup issues

### Tooling prerequisites

- Go `1.24+`
- Docker Desktop
- PostgreSQL client tools or a DB viewer
- RabbitMQ management UI via Docker
- `protoc`
- `protoc-gen-go`
- `protoc-gen-go-grpc`

Example install commands for protobuf tooling:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

You also need the `protoc` binary itself installed on your machine.

## Suggested learning path

1. Read `docs/IMPLEMENTATION_CHECKLIST.md`
2. Review `proto/order/v1/order.proto`
3. Start infra with Docker Compose
4. Implement database connection and repository layer
5. Implement the gRPC handler
6. Publish the first `OrderCreated` event

## Local setup

Copy the example env file:

```bash
cp .env.example .env
```

Start infrastructure:

```bash
docker compose up -d
```

Apply migrations using your preferred migration tool, or manually run the SQL files in `migrations/`.

## Docker image builds

This repository now includes:

- A multi-stage `Dockerfile` that can build either the gRPC server or the consumer
- A GitHub Actions workflow at `.github/workflows/order-service-ci.yml`
- Test execution via `go test ./...`
- Docker image builds for `server` and `consumer` on pull requests
- Image publishing to GitHub Container Registry (`ghcr.io`) on `main` and version tags

Build the server image locally:

```bash
docker build -t booknest-order-service:server --build-arg SERVICE=server .
```

Build the consumer image locally:

```bash
docker build -t booknest-order-service:consumer --build-arg SERVICE=consumer .
```

## What to build first

Keep the first version intentionally small:

- `CreateOrder`
- `GetOrder`
- `ListOrders`
- Publish `OrderCreated`

Do not start with distributed transactions, sagas, or multiple consumers. Keep the first iteration boring and observable.
