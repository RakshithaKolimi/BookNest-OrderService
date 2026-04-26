PROTO_DIR=proto/order/v1
PROTO_FILE=$(PROTO_DIR)/order.proto

.PHONY: proto
proto:
	mkdir -p gen/order/v1
	protoc \
		--go_out=. \
		--go_opt=module=booknest-order-service \
		--go-grpc_out=. \
		--go-grpc_opt=module=booknest-order-service \
		$(PROTO_FILE)

.PHONY: infra
infra:
	docker compose up -d order-db rabbitmq

.PHONY: run
run:
	go run ./cmd/server

.PHONY: consumer
consumer:
	go run ./cmd/consumer
