# Build stage — uses project-root context so the shared module is reachable.
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Copy shared module first (referenced by go.mod replace directive).
COPY shared/go /shared/go

# Copy service module files; rewrite the replace directive to /shared/go.
COPY services/tenant-service/go.mod services/tenant-service/go.sum ./
RUN sed -i 's|../../shared/go|/shared/go|' go.mod
RUN go mod download

# Copy the rest of the service source.
COPY services/tenant-service/ .
RUN sed -i 's|../../shared/go|/shared/go|' go.mod

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tenant-service ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/
COPY --from=builder /app/tenant-service .

EXPOSE 8080
CMD ["./tenant-service"]
