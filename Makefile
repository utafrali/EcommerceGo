# =============================================================================
# EcommerceGo - Root Makefile
# =============================================================================

.PHONY: help setup build test lint proto-gen docker-up docker-down docker-infra \
        docker-infra-down docker-build docker-logs docker-backend docker-ps \
        migrate migrate-down seed clean fmt vet build-all test-all \
        security-scan vuln-check security-all swagger

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
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "==> Setup complete!"

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------
SERVICES := product cart order checkout payment user inventory campaign notification search media gateway

# Build metadata embedded via ldflags
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/utafrali/EcommerceGo/pkg/health.gitCommit=$(GIT_COMMIT) \
           -X github.com/utafrali/EcommerceGo/pkg/health.buildTime=$(BUILD_TIME)

build: ## Build all Go services (output to bin/)
	@for svc in $(SERVICES); do \
		echo "==> Building $$svc..."; \
		cd services/$$svc && go build -ldflags "$(LDFLAGS)" -o ../../bin/$$svc ./cmd/server && cd ../..; \
	done
	@echo "==> All services built successfully!"

build-%: ## Build a specific service (e.g., make build-product)
	@echo "==> Building $*..."
	cd services/$* && go build -ldflags "$(LDFLAGS)" -o ../../bin/$* ./cmd/server

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
# Swagger / OpenAPI
# -----------------------------------------------------------------------------
swagger: ## Regenerate OpenAPI docs for all services (requires swag: go install github.com/swaggo/swag/cmd/swag@latest)
	@echo "==> Regenerating Swagger docs..."
	@for svc in $(SERVICES); do \
		echo "  swag init for $$svc..."; \
		(cd services/$$svc && swag init -g cmd/server/main.go -o docs --overridesFile docs/.swaggo 2>/dev/null || true); \
	done
	@echo "==> Swagger docs generated (run 'make build' to pick up changes)"

swagger-%: ## Regenerate Swagger docs for a specific service (e.g., make swagger-product)
	@echo "==> Running swag init for $*..."
	cd services/$* && swag init -g cmd/server/main.go -o docs

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
# Security Scanning
# -----------------------------------------------------------------------------
security-scan: ## Run gosec static security analysis on all Go code
	@echo "==> Running gosec security scan..."
	@failed=0; \
	for target in pkg $(SERVICES); do \
		if [ "$$target" = "pkg" ]; then \
			dir=pkg; \
		else \
			dir=services/$$target; \
		fi; \
		echo "  Scanning $$target..."; \
		(cd $$dir && gosec -quiet -exclude-dir=vendor -exclude-dir=testdata ./...) || failed=1; \
	done; \
	if [ $$failed -eq 1 ]; then \
		echo "SECURITY SCAN FAILED"; exit 1; \
	fi
	@echo "SECURITY SCAN PASSED"

vuln-check: ## Run govulncheck for dependency vulnerabilities on all Go modules
	@echo "==> Running govulncheck..."
	@failed=0; \
	for target in pkg $(SERVICES); do \
		if [ "$$target" = "pkg" ]; then \
			dir=pkg; \
		else \
			dir=services/$$target; \
		fi; \
		echo "  Checking $$target..."; \
		(cd $$dir && govulncheck ./...) || failed=1; \
	done; \
	if [ $$failed -eq 1 ]; then \
		echo "VULNERABILITY CHECK FAILED"; exit 1; \
	fi
	@echo "VULNERABILITY CHECK PASSED"

security-all: security-scan vuln-check ## Run all security checks (gosec + govulncheck)

# -----------------------------------------------------------------------------
# Clean
# -----------------------------------------------------------------------------
clean: ## Clean build artifacts
	rm -rf bin/ coverage/
	@for svc in $(SERVICES); do \
		rm -f services/$$svc/$$svc; \
	done
