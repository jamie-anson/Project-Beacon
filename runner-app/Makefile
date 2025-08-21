.PHONY: help setup dev build test clean docker-up docker-down run-runner cover cover-html race cover-check

# Default test timeout (can be overridden: `make test TIMEOUT=1m`)
TIMEOUT ?= 10m

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
	go test -v -timeout=$(TIMEOUT) ./...

# Run tests with race detector
race:
	@echo "Running tests with -race..."
	go test -race -timeout=$(TIMEOUT) ./...

# Generate coverage profile and HTML report
cover:
	@echo "Running coverage across all packages (excluding ./cmd/)..."
	@PKGS=$$(go list ./... | grep -v "^github.com/jamie-anson/project-beacon-runner/cmd/"); \
	PKGCSV=$$(echo $$PKGS | tr ' ' ','); \
	go test -race -timeout=$(TIMEOUT) -coverpkg=$$PKGCSV -coverprofile=coverage.out $$PKGS

cover-html: cover
	@echo "Generating coverage HTML report (coverage.html)..."
	go tool cover -html=coverage.out -o coverage.html

# Check coverage thresholds (overall, core, and non-core packages)
# Usage: make cover-check OVERALL_MIN=0.85 CORE_MIN=0.90 NONCORE_MIN=0.80
cover-check: cover cover-html
	@OVERALL_MIN=${OVERALL_MIN}; \
	CORE_MIN=${CORE_MIN}; \
	NONCORE_MIN=${NONCORE_MIN}; \
	[ -z "$$OVERALL_MIN" ] && OVERALL_MIN=0.51 || true; \
	[ -z "$$CORE_MIN" ] && CORE_MIN=0.86 || true; \
	[ -z "$$NONCORE_MIN" ] && NONCORE_MIN=0.53 || true; \
	echo "Enforcing thresholds: overall >= $$OVERALL_MIN, core >= $$CORE_MIN, non-core >= $$NONCORE_MIN"; \
	overall_pct=$$(go tool cover -func=coverage.out | awk '/total:/ {print $$3}' | sed 's/%//'); \
	echo "Overall coverage: $$overall_pct%"; \
	core_regex='(pkg/models|pkg/crypto|pkg/merkle|internal/worker|internal/service|internal/jobspec)/'; \
	core_pct=$$(go tool cover -func=coverage.out | awk -v r="$$core_regex" 'BEGIN{sum=0; n=0} $$1 ~ r {gsub(/%/, "", $$3); sum+= $$3; n++} END{ if(n==0){print 0} else { printf "%.2f", sum/n } }'); \
	echo "Core coverage (avg): $$core_pct%"; \
	noncore_pct=$$(go tool cover -func=coverage.out | awk -v r="$$core_regex" 'BEGIN{sum=0; n=0} /\/(internal|pkg)\// { if ($$1 !~ r) { gsub(/%/, "", $$3); sum+= $$3; n++ } } END{ if(n==0){print 0} else { printf "%.2f", sum/n } }'); \
	echo "Non-core coverage (avg): $$noncore_pct%"; \
	ov_pass=$$(awk -v p="$$overall_pct" -v min="$$OVERALL_MIN" 'BEGIN{ if (p/100 >= min) print 1; else print 0 }'); \
	core_pass=$$(awk -v p="$$core_pct" -v min="$$CORE_MIN" 'BEGIN{ if (p/100 >= min) print 1; else print 0 }'); \
	noncore_pass=$$(awk -v p="$$noncore_pct" -v min="$$NONCORE_MIN" 'BEGIN{ if (p/100 >= min) print 1; else print 0 }'); \
	if [ "$$ov_pass" -ne 1 ]; then echo "FAIL: overall coverage ($$overall_pct%) < threshold ($$OVERALL_MIN)"; exit 1; fi; \
	if [ "$$core_pass" -ne 1 ]; then echo "FAIL: core coverage ($$core_pct%) < threshold ($$CORE_MIN)"; exit 1; fi; \
	if [ "$$noncore_pass" -ne 1 ]; then echo "FAIL: non-core coverage ($$noncore_pct%) < threshold ($$NONCORE_MIN)"; exit 1; fi; \
	echo "Coverage thresholds satisfied."

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
