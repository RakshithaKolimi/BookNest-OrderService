# BookNest Order Service — Beginner's Guide to Microservices

This guide explains microservices from scratch using your BookNest Order Service as the real example.
Read it top to bottom once, then use the step-by-step section as a checklist.

---

## Part 1 — What is a Microservice?

A **monolith** is one big application that does everything: user auth, cart, orders, payments, notifications.

A **microservice** is one small application that does one thing well and talks to other services over a network.

### Why split?

| Monolith | Microservice |
|---|---|
| One deploy for everything | Deploy each service independently |
| One bug can crash everything | A broken payment service doesn't kill the order service |
| Hard to scale one feature | Scale only the service that needs it |
| One database for all data | Each service owns its own database |

### The rule to remember

> A microservice owns its data and its logic. It never reaches into another service's database.

---

## Part 2 — The BookNest Architecture

Your project is extracting the order responsibility out of a monolith called `BookNest-Platform`.

```
                         ┌─────────────────────────┐
                         │     BookNest-Platform    │
                         │   (existing monolith /   │
                         │       API gateway)       │
                         └────────────┬────────────┘
                                      │  gRPC call
                                      ▼
                         ┌─────────────────────────┐
                         │   BookNest-OrderService  │  ◄── this repo
                         │                         │
                         │  • Creates orders        │
                         │  • Stores in Postgres    │
                         │  • Publishes events      │
                         └────────────┬────────────┘
                                      │  publishes to RabbitMQ
                                      ▼
                         ┌─────────────────────────┐
                         │      RabbitMQ broker     │
                         └────────────┬────────────┘
                                      │  event consumed by
                                      ▼
                         ┌─────────────────────────┐
                         │  Notification Service    │
                         │  (future — logs for now) │
                         └─────────────────────────┘
```

### Two communication styles used here

| Style | Technology | When to use |
|---|---|---|
| Synchronous | gRPC | "I need an answer right now" |
| Asynchronous | RabbitMQ | "Tell others something happened, don't wait" |

A client calls the Order Service via **gRPC** and waits for the response.
After saving the order, the Order Service fires an event into **RabbitMQ** and moves on without waiting for anyone to consume it.

---

## Part 3 — How a Single Order Request Flows

Follow this when a user places a book order:

```
User clicks "Buy"
      │
      ▼
BookNest-Platform (gateway)
      │
      │  gRPC CreateOrderRequest
      │  { user_id, items: [{book_id, qty, price}] }
      ▼
OrderService.CreateOrder  (internal/grpc/server.go)
      │
      │  validates input
      │  calculates totals
      │  generates order number (BN-{timestamp})
      ▼
OrderService business layer  (internal/service/order_service.go)
      │
      ▼
OrderRepository.CreateOrder  (internal/repository/postgres/order_repository.go)
      │
      │  INSERT orders + order_items in one transaction
      ▼
PostgreSQL  (docker container, port 5433)
      │
      │  success
      ▼
EventPublisher.Publish(OrderCreatedEvent)  (not yet built)
      │
      │  JSON payload to RabbitMQ exchange "booknest.events"
      │  routing key: "order.created"
      ▼
RabbitMQ  (docker container, port 5672)
      │
      ▼
gRPC returns CreateOrderResponse to platform
      │
      ▼
Platform returns HTTP 200 to user
```

---

## Part 4 — Your Project Structure Explained

```
BookNest-OrderService/
│
├── cmd/server/main.go          ← entry point: starts the gRPC server
│
├── internal/
│   ├── domain/
│   │   ├── order.go            ← Order and OrderItem structs (your data shapes)
│   │   └── interfaces.go       ← contracts: what a repository and publisher must do
│   │
│   ├── repository/postgres/
│   │   ├── db.go               ← opens the PostgreSQL connection pool
│   │   └── order_repository.go ← SQL queries: create, get, list orders
│   │
│   ├── service/
│   │   └── order_service.go    ← business logic: call repo, call publisher
│   │
│   ├── grpc/
│   │   ├── server.go           ← receives gRPC calls, calls service layer
│   │   ├── mapper.go           ← converts proto types ↔ domain types
│   │   └── errors.go           ← maps Go errors to gRPC status codes
│   │
│   ├── events/
│   │   └── order_created.go    ← defines the event payload struct
│   │
│   └── app/
│       └── app.go              ← wires everything together (dependency injection)
│
├── proto/order/v1/order.proto  ← defines the gRPC API (the contract)
├── gen/order/v1/               ← auto-generated Go code from the proto file
│
├── migrations/                 ← SQL files to create / drop the database tables
├── docker-compose.yml          ← starts PostgreSQL and RabbitMQ locally
└── .env                        ← your local config (ports, passwords)
```

### The Clean Architecture layers (inner → outer)

```
  ┌──────────────────────────────────┐
  │  domain/  (core business models) │  ← no dependencies on anything
  └──────────────┬───────────────────┘
                 │ used by
  ┌──────────────▼───────────────────┐
  │  service/  (business logic)      │  ← depends only on domain interfaces
  └──────────────┬───────────────────┘
                 │ used by
  ┌──────────────▼───────────────────┐
  │  grpc/ + repository/             │  ← implementation details
  └──────────────────────────────────┘
```

This matters because you can swap PostgreSQL for a different database later without touching the business logic.

---

## Part 5 — What is Already Built

The entire foundation is complete. You do not need to touch these files unless something breaks.

| File | Status | What it does |
|---|---|---|
| [domain/order.go](../internal/domain/order.go) | Done | Order & OrderItem data structs |
| [domain/interfaces.go](../internal/domain/interfaces.go) | Done | Repository + EventPublisher contracts |
| [repository/postgres/db.go](../internal/repository/postgres/db.go) | Done | Connects to PostgreSQL |
| [repository/postgres/order_repository.go](../internal/repository/postgres/order_repository.go) | Done | CreateOrder, GetOrder, ListOrders SQL |
| [service/order_service.go](../internal/service/order_service.go) | Done | Calls repo + publisher |
| [grpc/server.go](../internal/grpc/server.go) | Done | CreateOrder + GetOrder handlers |
| [grpc/mapper.go](../internal/grpc/mapper.go) | Done | Proto ↔ domain conversion |
| [grpc/errors.go](../internal/grpc/errors.go) | Done | Error → gRPC status mapping |
| [app/app.go](../internal/app/app.go) | Done | Dependency injection wiring |
| [cmd/server/main.go](../cmd/server/main.go) | Done | Starts the gRPC server |
| [migrations/](../migrations/) | Done | SQL schema for orders tables |
| [docker-compose.yml](../docker-compose.yml) | Done | PostgreSQL + RabbitMQ locally |
| [proto/order/v1/order.proto](../proto/order/v1/order.proto) | Done | gRPC API contract |
| [gen/order/v1/](../gen/order/v1/) | Done | Generated gRPC Go code |

---

## Part 6 — Step-by-Step: What You Need to Build Next

Work through these in order. Each step has one clear deliverable.

---

### Step 1 — Start local infrastructure

**Goal:** PostgreSQL and RabbitMQ are running on your machine.

```bash
# Copy the environment file
cp .env.example .env

# Start the containers
docker compose up -d

# Verify they are running
docker ps
```

You should see `order-db` (PostgreSQL) and `rabbitmq` in the list.
RabbitMQ management UI: http://localhost:15672 (guest / guest)

---

### Step 2 — Apply database migrations

**Goal:** The `orders` and `order_items` tables exist in PostgreSQL.

```bash
# Connect to the database
docker exec -it booknest-order-db psql -U postgres -d booknest_orders

# Inside psql, paste and run both migration files:
\i migrations/001_create_orders.up.sql

# Verify
\dt
```

You should see `orders` and `order_items` listed.

---

### Step 3 — Run the service and make your first gRPC call

**Goal:** The server starts without errors and accepts requests.

```bash
# Start the service
make run
# or: go run ./cmd/server/main.go
```

You should see: `gRPC server listening on :50051`

To test it, use `grpcurl` or a gRPC GUI like Postman or BloomRPC:

```bash
grpcurl -plaintext -d '{
  "user_id": "user-123",
  "payment_method": "CREDIT_CARD",
  "items": [{"book_id": "book-1", "purchase_count": 1, "purchase_price": 450}]
}' localhost:50051 order.v1.OrderService/CreateOrder
```

---

### Step 4 — Implement the RabbitMQ publisher

**Goal:** After an order is saved, a JSON event arrives in RabbitMQ.

**File to create:** `internal/events/rabbitmq_publisher.go`

This is the main thing you need to build. Here is the interface you must satisfy (from [domain/interfaces.go](../internal/domain/interfaces.go)):

```go
type EventPublisher interface {
    Publish(ctx context.Context, event interface{}) error
}
```

Your implementation must:
1. Connect to RabbitMQ using `RABBITMQ_URL` from the environment.
2. Declare an exchange named `booknest.events` (type: `topic`).
3. In `Publish()`, serialize the event to JSON and publish to the exchange with routing key `order.created`.

**Wire it in** [app/app.go](../internal/app/app.go) so `OrderService` gets the publisher.

**Verify:** After `CreateOrder`, check RabbitMQ UI → Exchanges → `booknest.events` → check message rate.

---

### Step 5 — Implement the ListOrders gRPC handler

**Goal:** A client can list all orders for a user.

**File to edit:** [internal/grpc/server.go](../internal/grpc/server.go)

The proto already defines `ListOrders`. The repository already has `ListOrdersByUser`.
You only need to add the handler that connects them, similar to how `GetOrder` is done.

---

### Step 6 — Implement the ConfirmPayment gRPC handler

**Goal:** A payment service can mark an order as paid.

**Files to edit:**
- [internal/repository/postgres/order_repository.go](../internal/repository/postgres/order_repository.go) — add `UpdatePaymentStatus` SQL
- [internal/service/order_service.go](../internal/service/order_service.go) — add `ConfirmPayment` method
- [internal/grpc/server.go](../internal/grpc/server.go) — add the handler

---

### Step 7 — Build a simple event consumer

**Goal:** A separate Go program (or goroutine) reads `OrderCreated` events from RabbitMQ and logs them.

**File to create:** `internal/events/consumer.go`

This proves the full asynchronous flow:
- Order saved → event published → consumer receives it → logs it

This is the foundation for future notification sending or other reactions.

---

### Step 8 — Connect BookNest-Platform as a gateway

**Goal:** The existing monolith forwards order requests to this service instead of handling them itself.

In `BookNest-Platform`, replace direct order-database calls with gRPC calls to `localhost:50051`.
Use the generated client from `gen/order/v1/order_grpc.pb.go` as reference for the proto contract.

---

## Part 7 — What to Build After All of the Above Works

Once the basic flow (create → store → publish → consume) works end-to-end, extend one thing at a time:

| Priority | What to add | Why |
|---|---|---|
| High | Structured logging (e.g., `log/slog`) | Understand what the service is doing |
| High | Retry on RabbitMQ publish failure | Events must not silently disappear |
| High | Idempotency on the consumer | RabbitMQ can redeliver — handle duplicates |
| Medium | Unit tests for the service layer | Catch logic bugs without running containers |
| Medium | Integration tests for the repository | Catch SQL bugs against a real database |
| Medium | gRPC middleware (logging, recovery) | Every request should be observable |
| Low | Distributed tracing (OpenTelemetry) | Trace a request across services |
| Low | Metrics (Prometheus) | Monitor order throughput and error rates |
| Low | Service discovery | When you deploy to Kubernetes |

---

## Part 8 — Key Concepts Glossary

| Term | Meaning in this project |
|---|---|
| **gRPC** | A fast, typed way for services to call each other over the network |
| **Protobuf** | The schema language that defines gRPC messages (`.proto` files) |
| **RabbitMQ** | A message broker — services publish events, other services consume them |
| **Exchange** | A RabbitMQ routing point; producers send to an exchange, not directly to a queue |
| **Routing key** | A label on a message (e.g., `order.created`) that RabbitMQ uses to route it |
| **Clean Architecture** | Organizing code in layers so business logic is independent of infrastructure |
| **Repository pattern** | Wrapping all database access in one place so the rest of the code doesn't know SQL |
| **Domain model** | The core data shapes of your business (Order, OrderItem) with no framework dependencies |
| **Eventual consistency** | After an event is published, other services will catch up — just not instantly |
| **Idempotency** | Processing the same message twice produces the same result (no double-charges, no duplicate emails) |

---

## Part 9 — Quick Reference: Key Files

| What you want to do | File to open |
|---|---|
| Change the gRPC API | [proto/order/v1/order.proto](../proto/order/v1/order.proto) |
| Add a new database query | [internal/repository/postgres/order_repository.go](../internal/repository/postgres/order_repository.go) |
| Add business logic | [internal/service/order_service.go](../internal/service/order_service.go) |
| Add a new gRPC handler | [internal/grpc/server.go](../internal/grpc/server.go) |
| Wire a new dependency | [internal/app/app.go](../internal/app/app.go) |
| Change the database schema | Add a new file in [migrations/](../migrations/) |
| Add a new event type | Add a file in [internal/events/](../internal/events/) |
| Configure ports/passwords | [.env](../.env) |
