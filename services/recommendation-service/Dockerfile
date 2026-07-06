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
COPY services/recommendation-service/go.mod services/recommendation-service/go.sum ./

# Fix replace directive for Docker context
RUN sed -i 's|../../shared/go|/shared/go|' go.mod
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY services/recommendation-service/ .

# Fix replace directive again after copy
RUN sed -i 's|../../shared/go|/shared/go|' go.mod

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod tidy
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o recommendation-service ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/recommendation-service .

# Expose port
EXPOSE 8094

# Run the application
CMD ["./recommendation-service"]
