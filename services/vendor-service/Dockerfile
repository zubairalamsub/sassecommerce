# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy shared module first
COPY shared/go /shared/go

# Copy go mod files
COPY services/vendor-service/go.mod services/vendor-service/go.sum ./

# Fix replace directive for Docker context
RUN sed -i 's|../../shared/go|/shared/go|' go.mod
RUN go mod download

# Copy source code
COPY services/vendor-service/ .

# Fix replace directive again after copy
RUN sed -i 's|../../shared/go|/shared/go|' go.mod

# Build the application
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vendor-service ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/vendor-service .

# Expose port
EXPOSE 8092

# Run the application
CMD ["./vendor-service"]
