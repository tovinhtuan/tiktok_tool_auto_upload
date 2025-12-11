# Multi-stage build for Go application with yt-dlp

# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies including gcc for CGO
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/auto_upload_tiktok ./cmd/main.go

# Stage 2: Runtime stage  
FROM alpine:3.19

# Install system dependencies and ffmpeg from Alpine repos
# Note: Alpine 3.19 includes ffmpeg in the community repository by default
RUN apk update && \
    apk add --no-cache \
    python3 \
    py3-pip \
    ffmpeg \
    ca-certificates \
    tzdata \
    wget && \
    rm -rf /var/cache/apk/*

# Install yt-dlp via pip
RUN pip3 install --break-system-packages --no-cache-dir yt-dlp && \
    rm -rf /root/.cache/pip/*

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/auto_upload_tiktok /app/auto_upload_tiktok

# Create necessary directories
RUN mkdir -p /app/downloads /app/logs /app/config

# Copy config example (user should mount their own config.yaml or use env vars)
COPY config/config.yaml.example /app/config/config.yaml.example

# Copy entrypoint script
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

# Make files executable
RUN chmod +x /app/auto_upload_tiktok /app/docker-entrypoint.sh

# Expose default port (can be overridden via PORT env var)
EXPOSE 8080

# Set default environment variables for Render
ENV PORT=8080

# Health check (uses PORT env var)
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/api/health || exit 1

# Use entrypoint script to handle environment variables
ENTRYPOINT ["/app/docker-entrypoint.sh"]

