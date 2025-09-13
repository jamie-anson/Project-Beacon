FROM golang:1.24.6-alpine AS builder

# Install git and ca-certificates (needed for go modules)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application (smaller, reproducible binary)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -a -installsuffix cgo -o runner cmd/runner/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests, bash for init scripts, curl, and IPFS
RUN apk --no-cache add ca-certificates bash wget tar curl go-ipfs

WORKDIR /app

# Create non-root user and group
RUN addgroup -S app && adduser -S app -G app

# Copy the binary from builder and set ownership
COPY --from=builder /app/runner /app/runner
COPY scripts/init-trusted-keys.sh /app/scripts/init-trusted-keys.sh
COPY scripts/start-with-ipfs.sh /app/scripts/start-with-ipfs.sh
RUN chmod +x /app/scripts/init-trusted-keys.sh /app/scripts/start-with-ipfs.sh
RUN chown -R app:app /app

# Expose ports (8090 for runner, 5001 for IPFS API, 8080 for IPFS gateway)
EXPOSE 8090 5001 8080

# Drop privileges
USER app

# Run with IPFS
CMD ["/app/scripts/start-with-ipfs.sh"]
