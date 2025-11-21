# Volnix Protocol Production Dockerfile
# Multi-stage build for minimal image size and security

# Build arguments
ARG VERSION=unknown
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Install security scanning tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install golang.org/x/vuln/cmd/govulncheck@latest

# Set working directory
WORKDIR /app

# Copy go module files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && \
    go mod verify

# Copy source code
COPY . .

# Run security checks
RUN golangci-lint run --timeout=5m || true && \
    govulncheck ./... || true

# Build the standalone binary with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o /app/build/volnixd-standalone \
    ./cmd/volnixd-standalone

# Stage 2: Runtime
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    bash \
    curl \
    jq \
    netcat-openbsd \
    tzdata && \
    update-ca-certificates

# Create non-root user
RUN addgroup -g 1000 volnix && \
    adduser -D -u 1000 -G volnix volnix

# Create directories with proper permissions
RUN mkdir -p /home/volnix/.volnix/data && \
    mkdir -p /home/volnix/.volnix/config && \
    chown -R volnix:volnix /home/volnix && \
    chmod 700 /home/volnix/.volnix

# Copy binary from builder
COPY --from=builder /app/build/volnixd-standalone /usr/local/bin/volnixd-standalone

# Copy utility scripts
COPY infrastructure/docker/entrypoint.sh /usr/local/bin/entrypoint.sh
COPY infrastructure/docker/healthcheck.sh /usr/local/bin/healthcheck.sh
COPY infrastructure/docker/node-info.sh /usr/local/bin/node-info.sh

# Set binary and script permissions
RUN chmod +x /usr/local/bin/volnixd-standalone && \
    chmod +x /usr/local/bin/entrypoint.sh && \
    chmod +x /usr/local/bin/healthcheck.sh && \
    chmod +x /usr/local/bin/node-info.sh && \
    chown -R volnix:volnix /usr/local/bin/volnixd-standalone /usr/local/bin/entrypoint.sh /usr/local/bin/healthcheck.sh /usr/local/bin/node-info.sh

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
    CMD /usr/local/bin/healthcheck.sh

# Default environment variables
ENV VOLNIX_HOME=/home/volnix/.volnix
ENV VOLNIX_RPC_PORT=26657
ENV VOLNIX_P2P_PORT=26656
ENV PATH="/usr/local/bin:${PATH}"

# Metadata labels
LABEL org.opencontainers.image.title="Volnix Protocol" \
      org.opencontainers.image.description="Volnix Protocol blockchain node" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      org.opencontainers.image.vendor="Volnix" \
      org.opencontainers.image.authors="Volnix Team"

# Volume for data persistence
VOLUME ["/home/volnix/.volnix/data", "/home/volnix/.volnix/config"]

# Entry point
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["start"]

