# =============================================================================
# User Service - Makefile
# =============================================================================

SERVICE_NAME := user
BINARY_NAME  := user-service
CMD_DIR      := ./cmd/server
MIGRATIONS   := ./migrations

# Database configuration (override via environment)
POSTGRES_HOST     ?= localhost
POSTGRES_PORT     ?= 5432
POSTGRES_USER     ?= ecommerce
POSTGRES_PASSWORD ?= ecommerce_secret
POSTGRES_DB       ?= user_db
POSTGRES_SSL_MODE ?= disable

DATABASE_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)

.PHONY: help build run test test-coverage lint fmt vet clean migrate migrate-down migrate-create

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------
build: ## Build the service binary
	@echo "==> Building $(SERVICE_NAME)..."
	go build -o $(BINARY_NAME) $(CMD_DIR)
	@echo "==> Built $(BINARY_NAME)"

run: ## Run the service locally
	go run $(CMD_DIR)/main.go

# -----------------------------------------------------------------------------
# Test
# -----------------------------------------------------------------------------
test: ## Run all tests
	go test ./... -v -count=1 -race

test-coverage: ## Run tests with coverage report
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "==> Coverage report: coverage.html"

# -----------------------------------------------------------------------------
# Code Quality
# -----------------------------------------------------------------------------
lint: ## Run linters
	golangci-lint run ./...

fmt: ## Format Go code
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

# -----------------------------------------------------------------------------
# Database Migrations
# -----------------------------------------------------------------------------
migrate: ## Run database migrations up
	@echo "==> Running migrations for $(SERVICE_NAME)..."
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" up

migrate-down: ## Rollback the last migration
	@echo "==> Rolling back last migration for $(SERVICE_NAME)..."
	migrate -path $(MIGRATIONS) -database "$(DATABASE_URL)" down 1

migrate-create: ## Create a new migration (usage: make migrate-create NAME=add_something)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir $(MIGRATIONS) -seq $(NAME)

# -----------------------------------------------------------------------------
# Clean
# -----------------------------------------------------------------------------
clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
