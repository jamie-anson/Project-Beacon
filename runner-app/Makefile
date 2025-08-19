.PHONY: help setup dev build test clean docker-up docker-down run-runner

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
	@echo "  run-runner - Start runner container with auto port failover (8090->8091)"

# Setup development environment
setup:
	@echo "Setting up Project Beacon Runner..."
	go mod tidy
	go mod download
	@echo "Setup complete!"

# Run development server (loads .env if present)
dev:
	@echo "Starting development server..."
	@if [ -f .env ]; then \
		set -a; \
		. ./.env; \
		set +a; \
		go run cmd/runner/main.go; \
	else \
		go run cmd/runner/main.go; \
	fi

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

# Start Redis service
redis-up:
	@echo "Starting Redis service..."
	docker compose up -d redis

# Stop Redis service
redis-down:
	@echo "Stopping Redis service..."
	docker compose stop redis

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

# Start runner container with auto port failover
run-runner:
	@echo "Starting runner container with port failover..."
	@if lsof -i :8090 >/dev/null 2>&1; then \
		echo "Port 8090 busy, using 8091..."; \
		PORT=8091 docker compose up -d runner; \
	else \
		echo "Port 8090 available, using 8090..."; \
		PORT=8090 docker compose up -d runner; \
	fi
	@echo "Runner container started!"
