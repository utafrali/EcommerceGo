# =============================================================================
# Gateway Service - Makefile
# =============================================================================

SERVICE_NAME := gateway
BINARY_NAME  := gateway-service
CMD_DIR      := ./cmd/server

.PHONY: help build run test test-coverage lint fmt vet clean

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
# Clean
# -----------------------------------------------------------------------------
clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
