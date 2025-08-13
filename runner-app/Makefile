.PHONY: help setup dev build test clean docker-up docker-down

# Default target
help:
	@echo "Project Beacon Runner - Available commands:"
	@echo "  setup      - Install dependencies and initialize project"
	@echo "  dev        - Run development server with hot reload"
	@echo "  build      - Build the application binary"
	@echo "  test       - Run all tests"
	@echo "  clean      - Clean build artifacts"
	@echo "  docker-up  - Start all services with Docker Compose"
	@echo "  docker-down- Stop all Docker services"

# Setup development environment
setup:
	@echo "Setting up Project Beacon Runner..."
	go mod tidy
	go mod download
	@echo "Setup complete!"

# Run development server
dev:
	@echo "Starting development server..."
	go run cmd/runner/main.go

# Build the application
build:
	@echo "Building runner application..."
	go build -o bin/runner cmd/runner/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	docker compose up -d postgres redis ipfs
	@echo "Services started. Waiting for health checks..."
	@sleep 10
	@echo "Services ready!"

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker compose down

# Full Docker environment (including runner)
docker-full:
	@echo "Starting full Docker environment..."
	docker compose up --build

# Database migration
migrate:
	@echo "Running database migrations..."
	go run cmd/runner/main.go -migrate

# Generate API documentation
docs:
	@echo "Generating API documentation..."
	@echo "TODO: Add swagger/openapi generation"
