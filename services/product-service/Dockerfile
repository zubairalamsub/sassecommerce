# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy shared module first
COPY shared/go /shared/go

# Copy go mod files
COPY services/product-service/go.mod services/product-service/go.sum ./

# Fix replace directive for Docker context
RUN sed -i 's|../../shared/go|/shared/go|' go.mod
RUN go mod download

# Copy source code
COPY services/product-service/ .

# Fix replace directive again after copy
RUN sed -i 's|../../shared/go|/shared/go|' go.mod

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8083

# Run the binary
CMD ["./main"]
