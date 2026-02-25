.PHONY: build test lint run

SERVICE_NAME := cart

build:
	go build -o bin/$(SERVICE_NAME) ./cmd/server

test:
	go test -v -race -cover ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server
