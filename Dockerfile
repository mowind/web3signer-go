# Multi-stage Dockerfile for web3signer-go
# Builds a minimal and secure Docker image

# Stage 1: Builder
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for go get with git dependencies)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG BUILD_TIME=unknown

# Build with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
  -o /app/web3signer \
  ./cmd/web3signer

# Stage 2: Runtime
FROM alpine:3.19

# Install ca-certificates, tzdata and curl for health checks
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 web3signer && \
    adduser -D -G web3signer -u 1001 web3signer

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=web3signer:web3signer /app/web3signer /app/

# Copy config files if any (only if configs directory exists)
COPY --chown=web3signer:web3signer configs/ /app/configs/

# Set timezone
ENV TZ=Asia/Shanghai

# Expose HTTP port
EXPOSE 9000

# Health check using curl to check HTTP endpoints
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:9000/health || exit 1

# Run as non-root user
USER web3signer

# Use dumb-init to handle signals properly
ENTRYPOINT ["/app/web3signer"]
CMD ["--http-host", "0.0.0.0", "--http-port", "9000"]