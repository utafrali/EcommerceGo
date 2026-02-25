# =============================================================================
# EcommerceGo - Root Makefile
# =============================================================================

.PHONY: help setup build test lint proto-gen docker-up docker-down docker-infra \
        docker-infra-down docker-build docker-logs docker-backend docker-ps \
        migrate migrate-down seed clean fmt vet build-all test-all

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# -----------------------------------------------------------------------------
# Setup
# -----------------------------------------------------------------------------
setup: ## Initial project setup (install tools, generate proto)
	@echo "==> Installing Go tools..."
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "==> Setup complete!"

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------
SERVICES := product cart order checkout payment user inventory campaign notification search media gateway

build: ## Build all Go services (output to bin/)
	@for svc in $(SERVICES); do \
		echo "==> Building $$svc..."; \
		cd services/$$svc && go build -o ../../bin/$$svc ./cmd/server && cd ../..; \
	done
	@echo "==> All services built successfully!"

build-%: ## Build a specific service (e.g., make build-product)
	@echo "==> Building $*..."
	cd services/$* && go build -o ../../bin/$* ./cmd/server

build-all: ## Build all Go services (compile check, no output binary)
	@for svc in $(SERVICES); do \
		echo "==> Building $$svc..."; \
		(cd services/$$svc && go build ./...) || exit 1; \
	done
	@echo "ALL BUILDS OK"

# -----------------------------------------------------------------------------
# Test
# -----------------------------------------------------------------------------
test: ## Run tests for all Go services
	@for svc in $(SERVICES); do \
		echo "==> Testing $$svc..."; \
		cd services/$$svc && go test ./... -v -count=1 && cd ../..; \
	done
	cd pkg && go test ./... -v -count=1

test-%: ## Run tests for a specific service (e.g., make test-product)
	cd services/$* && go test ./... -v -count=1

test-pkg: ## Run tests for shared packages
	cd pkg && go test ./... -v -count=1

test-all: ## Run tests for all Go services with race detection
	@for svc in $(SERVICES); do \
		echo "==> Testing $$svc..."; \
		(cd services/$$svc && go test ./... -count=1 -race) || exit 1; \
	done
	@echo "ALL TESTS PASSED"

test-coverage: ## Run tests with coverage
	@for svc in $(SERVICES); do \
		echo "==> Coverage for $$svc..."; \
		cd services/$$svc && go test ./... -coverprofile=../../coverage/$$svc.out && cd ../..; \
	done

# -----------------------------------------------------------------------------
# Lint & Format
# -----------------------------------------------------------------------------
lint: ## Run linters on all Go code
	golangci-lint run ./pkg/... ./services/...

fmt: ## Format all Go code
	gofmt -s -w ./pkg/ ./services/

vet: ## Run go vet on all code
	@for svc in $(SERVICES); do \
		cd services/$$svc && go vet ./... && cd ../..; \
	done
	cd pkg && go vet ./...

# -----------------------------------------------------------------------------
# Protobuf
# -----------------------------------------------------------------------------
proto-gen: ## Generate Go and gRPC code from proto files
	@echo "==> Generating protobuf code..."
	cd proto && buf generate
	@echo "==> Protobuf generation complete!"

proto-lint: ## Lint proto files
	cd proto && buf lint

# -----------------------------------------------------------------------------
# Database Migrations
# -----------------------------------------------------------------------------
migrate: ## Run migrations for all services (requires POSTGRES_HOST etc in env)
	@for svc in $(SERVICES); do \
		if [ -d "services/$$svc/migrations" ] && [ "$$(ls -A services/$$svc/migrations 2>/dev/null)" ]; then \
			echo "==> Migrating $$svc..."; \
			migrate -path services/$$svc/migrations -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$${svc}_db?sslmode=disable" up; \
		fi; \
	done

migrate-%: ## Run migration for a specific service (e.g., make migrate-product)
	migrate -path services/$*/migrations \
		-database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$*_db?sslmode=disable" up

migrate-down-%: ## Rollback migration for a specific service
	migrate -path services/$*/migrations \
		-database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$*_db?sslmode=disable" down 1

# -----------------------------------------------------------------------------
# Docker
# -----------------------------------------------------------------------------
docker-build: ## Build all Docker images
	docker compose build

docker-up: ## Start all services
	docker compose --profile infra --profile backend --profile frontend up -d

docker-down: ## Stop all services
	docker compose --profile infra --profile backend --profile frontend down

docker-infra: ## Start only infrastructure (PostgreSQL, Redis, Kafka, ES, MinIO)
	docker compose --profile infra up -d

docker-infra-down: ## Stop infrastructure
	docker compose --profile infra down

docker-backend: ## Start infrastructure + backend services
	docker compose --profile infra --profile backend up -d

docker-frontend: ## Start infrastructure + backend + frontend services
	docker compose --profile infra --profile backend --profile frontend up -d

docker-logs: ## Tail logs for all services
	docker compose logs -f

docker-logs-%: ## Tail logs for a specific service (e.g., make docker-logs-product)
	docker compose logs -f $*

docker-ps: ## Show running services
	docker compose ps

# -----------------------------------------------------------------------------
# Seed Data
# -----------------------------------------------------------------------------
seed: ## Seed sample data
	@echo "==> Seeding data..."
	./scripts/seed.sh

# -----------------------------------------------------------------------------
# Clean
# -----------------------------------------------------------------------------
clean: ## Clean build artifacts
	rm -rf bin/ coverage/
	@for svc in $(SERVICES); do \
		rm -f services/$$svc/$$svc; \
	done
