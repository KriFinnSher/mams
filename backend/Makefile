.PHONY: help build run test lint clean docker-build docker-up docker-down migrate-create migrate-up migrate-down

BINARY_NAME=mams
BUILD_DIR=bin
CMD_PATH=cmd/server

help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run the application"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-up      - Start Docker containers"
	@echo "  make docker-down    - Stop Docker containers"
	@echo "  make migrate-create - Create new migration"
	@echo "  make migrate-up     - Run migrations"
	@echo "  make migrate-down   - Rollback migrations"

build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

run:
	$(GO) run $(CMD_PATH)

test:
	$(GO) test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf $(BUILD_DIR)
	rm -rf coverage.txt

docker-build:
	docker build -t $(BINARY_NAME):latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-create:
	@echo "Usage: make migrate-create name=migration_name"

migrate-up:
	migrate -path=migrations/postgres -database="postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" up

migrate-down:
	migrate -path=migrations/postgres -database="postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" down
