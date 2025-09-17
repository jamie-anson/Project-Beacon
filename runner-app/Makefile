.PHONY: help setup dev build test clean docker-up docker-down run-runner cover cover-html race cover-check port addr base yagna-start yagna-stop yagna-status e2e

# Use bash for advanced env handling (indirect expansion)
SHELL := /bin/bash

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
	@echo "  docker-dev - Docker mode: services + runner (uses port 8090)"
	@echo "  run-runner - Start runner container with auto port failover (8090->8091)"
	@echo "  port       - Print active HTTP port from .runner-http.addr"
	@echo "  addr       - Print active bound addr from .runner-http.addr"
	@echo "  base       - Print base URL using .runner-http.addr (http://localhost:<port>)"
	@echo "  e2e        - Run end-to-end sign→submit test (requires Postgres & Redis running, API on :8090)"
	@echo "  sign       - Sign a JobSpec JSON with dev key (usage: make sign IN=examples/foo.json [OUT=...])"
	@echo "  verify     - Verify a signed JobSpec (usage: make verify IN=examples/foo.json.signed)"
	@echo "  submit     - Submit a (signed) JobSpec to API (usage: make submit FILE=examples/foo.json.signed [URL=http://localhost:8090])"
	@echo "  extract-pubkey - Print public key from signed file (usage: make extract-pubkey IN=...json.signed)"
	@echo "  regen-examples - Regenerate signed example JobSpecs using sigtool"
	@echo "  validate-examples - Validate example .signed files are parseable JSON"
	@echo "  install-pre-commit - Install git pre-commit hook to validate examples"
	@echo ""
	@echo "Terminal-specific targets:"
	@echo "  yagna-start - Terminal A: Start Yagna daemon"
	@echo "  yagna-stop  - Terminal A: Stop Yagna daemon"
	@echo "  yagna-status- Terminal A: Check Yagna daemon status"

# Setup development environment
setup:
	@echo "Setting up Project Beacon Runner..."
	go mod tidy
	go mod download
	@echo "Setup complete!"

# Run development server (loads .env if present)
dev:
	@echo "Starting development server..."
	@echo "Ensuring no Docker runner conflicts..."
	@docker compose stop runner 2>/dev/null || true
	@if [ -f .env ]; then \
		echo "Loading .env (overriding shell env for dev)..."; \
		while IFS= read -r line; do \
			case "$$line" in ''|'#'*) continue ;; esac; \
			key=$$(printf '%s' "$$line" | cut -d= -f1 | xargs); \
			val=$$(printf '%s' "$$line" | cut -d= -f2-); \
			if [ -z "$$key" ]; then continue; fi; \
			export "$$key=$$val"; \
		done < .env; \
		go run cmd/runner/main.go; \
	else \
		DATABASE_URL="postgres://postgres:password@localhost:5433/beacon_runner?sslmode=disable" \
		REDIS_URL="redis://localhost:6379" \
		go run cmd/runner/main.go; \
	fi

# Build the application
build:
	@echo "Building runner application..."
	go build -o bin/runner cmd/runner/main.go
	@echo "Building sigtool..."
	go build -o sigtool cmd/sigtool/main.go

# Paths and defaults for signing utilities
SIGTOOL := $(CURDIR)/sigtool
DEV_PRIV_KEY_FILE ?= $(CURDIR)/.dev/keys/private.key
DEV_PRIV_KEY_B64 := $(shell tr -d '\n\r\t ' < "$(DEV_PRIV_KEY_FILE)" 2>/dev/null)
API_URL ?= http://localhost:8090

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
	go test -race -timeout=$(TIMEOUT) -tags ci -coverpkg=$$PKGCSV -coverprofile=coverage.out $$PKGS

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

# Docker mode: services + runner (stops local dev conflicts)
docker-dev:
	@echo "Starting Docker development mode..."
	@echo "This will use Docker runner on port 8090"
	docker compose up -d postgres redis ipfs
	docker compose up --build runner

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

# Print the active HTTP port (reads .runner-http.addr written by the server)
port:
	@if [ -f .runner-http.addr ]; then \
	  PORT=$$(grep -oE '[0-9]+$$' .runner-http.addr); \
	  if [ -n "$$PORT" ]; then echo $$PORT; else echo "could not parse port from .runner-http.addr"; fi; \
	else \
	  echo ".runner-http.addr not found. Start the server first."; \
	fi

# Print the active addr (host:port)
addr:
	@if [ -f .runner-http.addr ]; then \
	  cat .runner-http.addr; \
	else \
	  echo ".runner-http.addr not found. Start the server first."; \
	fi

# Print the base URL using helper script
base:
	@./scripts/runner-port.sh --base

# End-to-end sign→submit test (requires API on :8090, Postgres & Redis running)
e2e:
	@echo "Running E2E: sign→submit..."
	@chmod +x scripts/e2e-sign-submit.sh
	@./scripts/e2e-sign-submit.sh

# Sign a JobSpec JSON using dev private key (requires IN, optional OUT)
# Usage: make sign IN=examples/jobspec-who-are-you.json [OUT=examples/jobspec-who-are-you.json.signed]
sign: build
	@if [ -z "$(IN)" ]; then echo "Error: IN=<input.json> is required"; exit 1; fi
	@OUT_FILE=$$( [ -n "$(OUT)" ] && echo "$(OUT)" || echo "$(IN).signed" ); \
	  if [ ! -f "$(DEV_PRIV_KEY_FILE)" ]; then echo "Error: dev private key not found at $(DEV_PRIV_KEY_FILE)"; exit 1; fi; \
	  if [ ! -x "$(SIGTOOL)" ]; then echo "Error: sigtool not built"; exit 1; fi; \
	  TMP=$$(mktemp); \
	  jq '(.metadata.timestamp? |= empty) | (.metadata.nonce? |= empty)' "$(IN)" > "$$TMP"; \
	  "$(SIGTOOL)" sign --private-key "$(DEV_PRIV_KEY_B64)" --input "$$TMP" --output "$$OUT_FILE"; \
	  rm -f "$$TMP"; \
	  echo "Signed -> $$OUT_FILE"

# Verify a signed JobSpec
# Usage: make verify IN=examples/jobspec-who-are-you.json.signed
verify: build
	@if [ -z "$(IN)" ]; then echo "Error: IN=<input.signed> is required"; exit 1; fi
	@if [ ! -x "$(SIGTOOL)" ]; then echo "Error: sigtool not built"; exit 1; fi
	@"$(SIGTOOL)" verify --input "$(IN)"

# Extract public key from a signed JobSpec
# Usage: make extract-pubkey IN=examples/jobspec-who-are-you.json.signed
extract-pubkey: build
	@if [ -z "$(IN)" ]; then echo "Error: IN=<input.signed> is required"; exit 1; fi
	@if [ ! -x "$(SIGTOOL)" ]; then echo "Error: sigtool not built"; exit 1; fi
	@"$(SIGTOOL)" extract-pubkey --input "$(IN)"

# Submit a (signed) JobSpec to the API
# Usage: make submit FILE=examples/jobspec-who-are-you.json.signed [URL=http://localhost:8090]
submit:
	@if [ -z "$(FILE)" ]; then echo "Error: FILE=<path to .json or .signed> is required"; exit 1; fi
	@URL=$$( [ -n "$(URL)" ] && echo "$(URL)" || echo "$(API_URL)" ); \
	  echo "Submitting $(FILE) -> $$URL/api/v1/jobs"; \
	  curl -sS -D >(grep -E 'HTTP/|location' >&2) -H 'Content-Type: application/json' \
	    --data-binary @"$(FILE)" "$$URL/api/v1/jobs" | jq .

# Regenerate signed examples using current signing rules
regen-examples:
	@chmod +x scripts/regenerate_signed_examples.sh
	@./scripts/regenerate_signed_examples.sh

# Validate that signed examples are valid JSON
validate-examples:
	@echo "Validating example JSON..."
	@for f in examples/*.signed; do \
	  if [ -f "$$f" ]; then echo "Checking $$f"; jq -e . "$$f" >/dev/null; fi; \
	done

# Install git pre-commit hook
install-pre-commit:
	@echo "Installing pre-commit hook..."
	@mkdir -p .git/hooks
	@cp scripts/pre-commit-validate-examples.sh .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Installed .git/hooks/pre-commit"

# Terminal A: Yagna daemon management
yagna-start:
	@echo "Starting Yagna daemon (Terminal A)..."
	@if command -v yagna >/dev/null 2>&1; then \
		yagna service run --api-allow-origin='*' & \
		echo "Yagna daemon started. Check status with 'make yagna-status'"; \
	else \
		echo "Error: yagna not found. Install with: curl -sSf https://join.golem.network/as-requestor | bash"; \
		exit 1; \
	fi

yagna-stop:
	@echo "Stopping Yagna daemon..."
	@pkill -f "yagna service" || echo "No Yagna daemon running"

yagna-status:
	@echo "Yagna daemon status:"
	@if pgrep -f "yagna service" >/dev/null; then \
		echo "✓ Yagna daemon is running"; \
		if command -v yagna >/dev/null 2>&1; then \
			echo "API endpoint: http://localhost:7465"; \
			yagna app-key list 2>/dev/null || echo "Run 'yagna app-key create requestor' to create API key"; \
		fi; \
	else \
		echo "✗ Yagna daemon is not running"; \
	fi
