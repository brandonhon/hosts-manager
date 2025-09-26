# Multi-stage build for hosts-manager
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sudo

# Create non-root user
RUN adduser -D -s /bin/sh hostsmgr

# Copy binary from builder stage
COPY --from=builder /app/build/hosts-manager /usr/local/bin/hosts-manager

# Set permissions
RUN chmod +x /usr/local/bin/hosts-manager

# Create directories for config and data
RUN mkdir -p /home/hostsmgr/.config/hosts-manager \
             /home/hostsmgr/.local/share/hosts-manager \
             /etc/hosts-manager

# Set ownership
RUN chown -R hostsmgr:hostsmgr /home/hostsmgr

# Switch to non-root user
USER hostsmgr

# Set working directory
WORKDIR /home/hostsmgr

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD hosts-manager --help > /dev/null || exit 1

# Default command
ENTRYPOINT ["hosts-manager"]
CMD ["--help"]