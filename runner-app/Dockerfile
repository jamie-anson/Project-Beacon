FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for go modules)
RUN apk update && apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files from runner-app directory
COPY runner-app/go.mod runner-app/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code from runner-app directory
COPY runner-app/ .

# Build the server application with cross-region API endpoints
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -a -installsuffix cgo -o server cmd/server/main.go

# Final stage
FROM alpine:3.18

# Install ca-certificates for HTTPS requests and curl for health checks
RUN apk update && apk add --no-cache ca-certificates curl

WORKDIR /app

# Create non-root user and group
RUN addgroup -S app && adduser -S app -G app

# Copy the server binary from builder
COPY --from=builder /app/server /app/server

# Copy migrations directory for database setup
COPY --from=builder /app/migrations /app/migrations

# Set ownership
RUN chown -R app:app /app

# Expose port 8080 for the server API
EXPOSE 8080

# Drop privileges
USER app

# Run the server
CMD ["/app/server"]
