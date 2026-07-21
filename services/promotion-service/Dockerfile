# syntax=docker/dockerfile:1.7
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org,direct

# Copy shared module first
COPY shared/go /shared/go

# Copy go mod files
COPY services/promotion-service/go.mod services/promotion-service/go.sum ./

# Fix replace directive for Docker context
RUN sed -i 's|../../shared/go|/shared/go|' go.mod
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY services/promotion-service/ .

# Fix replace directive again after copy
RUN sed -i 's|../../shared/go|/shared/go|' go.mod

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o promotion-service ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/promotion-service .

# Expose port
EXPOSE 8091

# Run the application
CMD ["./promotion-service"]
