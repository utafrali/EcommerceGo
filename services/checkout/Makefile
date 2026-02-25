.PHONY: build test lint run

build:
	go build -o bin/checkout-service ./cmd/server

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server
