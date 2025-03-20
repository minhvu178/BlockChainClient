# Start from a small Golang base image
FROM golang:1.20-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum (if available) and download dependencies
COPY go.mod ./
RUN go mod download

# Copy source code (all .go files)
COPY . .

# Build the Go app
RUN go build -o solana-client

# Use a minimal Alpine image for the final container
FROM alpine:latest

# Set working directory
WORKDIR /root/

# Create a non-root user for security
RUN adduser -D -g '' appuser
USER appuser

# Copy the binary from the builder stage
COPY --from=builder /app/solana-client .

# Expose API port
EXPOSE 8080

# Health check to ensure the container is running
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/latest-block || exit 1

# Run the binary
CMD ["./solana-client"]
