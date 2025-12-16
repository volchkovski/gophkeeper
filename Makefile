.PHONY: build build-server build-client build-all test lint migrate-up migrate-down clean run-server run-client docker-up docker-down generate

# Version info
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)"

# Database
DB_DSN ?= postgres://gophkeeper:secret@localhost:5432/gophkeeper?sslmode=disable

# Build targets
build: build-server build-client

build-server:
	@echo "Building server..."
	go build $(LDFLAGS) -o bin/gophkeeper-server ./cmd/server

build-client:
	@echo "Building client..."
	go build $(LDFLAGS) -o bin/gophkeeper ./cmd/client

build-all:
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/gophkeeper-linux-amd64 ./cmd/client
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/gophkeeper-darwin-amd64 ./cmd/client
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/gophkeeper-darwin-arm64 ./cmd/client
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/gophkeeper-windows-amd64.exe ./cmd/client

# Test targets
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

test-coverage:
	@echo "Running tests with HTML coverage report..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

test-unit:
	@echo "Running unit tests..."
	go test -v -race -short ./...

test-integration:
	@echo "Running integration tests..."
	go test -v -race -run Integration ./...

# Lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Database migrations
migrate-up:
	@echo "Running migrations up..."
	migrate -path migrations -database "$(DB_DSN)" up

migrate-down:
	@echo "Running migrations down..."
	migrate -path migrations -database "$(DB_DSN)" down

migrate-create:
	@echo "Creating new migration..."
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# Run targets
run-server:
	@echo "Starting server..."
	go run ./cmd/server

run-client:
	@echo "Starting client..."
	go run ./cmd/client

# Docker
docker-up:
	@echo "Starting docker containers..."
	docker-compose up -d

docker-down:
	@echo "Stopping docker containers..."
	docker-compose down

docker-build:
	@echo "Building docker images..."
	docker-compose build

# Clean
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Generate mocks
generate:
	@echo "Generating mocks..."
	go generate ./...

# Install tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang/mock/mockgen@latest

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build server and client"
	@echo "  build-server   - Build server only"
	@echo "  build-client   - Build client only"
	@echo "  build-all      - Build client for all platforms"
	@echo "  test           - Run all tests with coverage"
	@echo "  test-coverage  - Run tests and generate HTML coverage report"
	@echo "  test-unit      - Run unit tests only"
	@echo "  lint           - Run linter"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback database migrations"
	@echo "  run-server     - Run server locally"
	@echo "  run-client     - Run client locally"
	@echo "  docker-up      - Start docker containers"
	@echo "  docker-down    - Stop docker containers"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  install-tools  - Install development tools"

