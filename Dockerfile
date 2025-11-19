# Volnix Protocol Production Dockerfile
# Multi-stage build for minimal image size

# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Set working directory
WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the standalone binary
RUN make build-standalone

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates bash curl

# Create non-root user
RUN addgroup -g 1000 volnix && \
    adduser -D -u 1000 -G volnix volnix

# Create directories
RUN mkdir -p /home/volnix/.volnix && \
    chown -R volnix:volnix /home/volnix

# Copy binary from builder
COPY --from=builder /app/build/volnixd-standalone /usr/local/bin/

# Switch to non-root user
USER volnix
WORKDIR /home/volnix

# Expose ports
# P2P port
EXPOSE 26656
# RPC port
EXPOSE 26657
# API port (if needed)
EXPOSE 1317
# gRPC port (if needed)
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:${VOLNIX_RPC_PORT:-26657}/health || exit 1

# Default environment variables
ENV VOLNIX_HOME=/home/volnix/.volnix
ENV VOLNIX_RPC_PORT=26657
ENV VOLNIX_P2P_PORT=26656

# Volume for data persistence
VOLUME ["/home/volnix/.volnix"]

# Entry point
ENTRYPOINT ["volnixd-standalone"]
CMD ["start"]

