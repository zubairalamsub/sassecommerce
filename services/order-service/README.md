# Order Service

The Order Service manages the complete order lifecycle in the e-commerce platform using Event Sourcing and CQRS patterns.

## Architecture

### Event Sourcing
- All state changes are captured as immutable events
- Events are stored in an append-only event store
- Aggregate state is reconstructed by replaying events
- Provides complete audit trail of all order changes

### CQRS (Command Query Responsibility Segregation)
- **Write Model**: Commands modify state and generate events
- **Read Model**: Denormalized projections optimized for queries
- Separate database tables for writes (events) and reads (projections)

### Saga Pattern
- Orchestrates distributed transactions across services
- Three main steps:
  1. Reserve Inventory
  2. Process Payment
  3. Confirm Order
- Automatic compensation on failure

### Kafka Event Streaming
- Events published to Kafka for inter-service communication
- Event consumers update read model projections
- Enables event-driven architecture

## Domain Model

### Order Aggregate
The Order is the aggregate root managing:
- Order items
- Order status transitions
- Payment tracking
- Inventory reservations
- Shipping information

### Order Lifecycle
```
Pending → Confirmed → Shipped → Delivered
    ↓
Cancelled
```

### Domain Events
- OrderCreated
- OrderItemAdded
- OrderItemRemoved
- OrderConfirmed
- OrderCancelled
- OrderShipped
- OrderDelivered
- PaymentProcessed
- PaymentFailed
- InventoryReserved
- InventoryReleased

## API Endpoints

### Commands (Write Operations)

#### Create Order
```http
POST /api/v1/orders
Content-Type: application/json

{
  "tenant_id": "tenant-123",
  "customer_id": "customer-123",
  "shipping_address": {
    "street": "123 Main St",
    "city": "Springfield",
    "state": "IL",
    "postal_code": "62701",
    "country": "USA"
  },
  "billing_address": {
    "street": "456 Oak Ave",
    "city": "Springfield",
    "state": "IL",
    "postal_code": "62702",
    "country": "USA"
  }
}
```

#### Add Order Item
```http
POST /api/v1/orders/{id}/items
Content-Type: application/json

{
  "product_id": "product-123",
  "variant_id": "variant-123",
  "sku": "SKU-001",
  "name": "Test Product",
  "quantity": 2,
  "unit_price": 99.99
}
```

#### Remove Order Item
```http
DELETE /api/v1/orders/{id}/items/{itemId}
```

#### Confirm Order
```http
POST /api/v1/orders/{id}/confirm
Content-Type: application/json

{
  "confirmed_by": "admin@example.com"
}
```

#### Cancel Order
```http
POST /api/v1/orders/{id}/cancel
Content-Type: application/json

{
  "reason": "Customer request",
  "cancelled_by": "customer@example.com"
}
```

#### Ship Order
```http
POST /api/v1/orders/{id}/ship
Content-Type: application/json

{
  "tracking_number": "TRACK123",
  "carrier": "FedEx",
  "shipped_by": "warehouse@example.com"
}
```

#### Deliver Order
```http
POST /api/v1/orders/{id}/deliver
Content-Type: application/json

{
  "received_by": "customer@example.com"
}
```

### Queries (Read Operations)

#### Get Order by ID
```http
GET /api/v1/orders/{id}
```

#### Get Orders by Customer
```http
GET /api/v1/customers/{customerId}/orders?limit=10&offset=0
```

#### Get Orders by Tenant
```http
GET /api/v1/tenants/{tenantId}/orders?limit=10&offset=0
```

## Configuration

Environment variables (see `.env.example`):

- `SERVER_HOST` - Server bind address (default: 0.0.0.0)
- `SERVER_PORT` - Server port (default: 8080)
- `DB_HOST` - PostgreSQL host
- `DB_PORT` - PostgreSQL port
- `DB_USER` - PostgreSQL username
- `DB_PASSWORD` - PostgreSQL password
- `DB_NAME` - Database name
- `KAFKA_ENABLED` - Enable Kafka (default: true)
- `KAFKA_BROKERS` - Kafka broker addresses
- `KAFKA_TOPIC` - Kafka topic for events
- `KAFKA_CONSUMER_GROUP` - Consumer group ID
- `INVENTORY_SERVICE_URL` - Inventory service URL
- `PAYMENT_SERVICE_URL` - Payment service URL
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

## Running the Service

### Local Development
```bash
# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=order_db
export KAFKA_BROKERS=localhost:9092

# Run the service
go run cmd/server/main.go
```

### Docker
```bash
# Build image
docker build -t order-service:latest .

# Run container
docker run -p 8080:8080 \
  -e DB_HOST=postgres \
  -e KAFKA_BROKERS=kafka:9092 \
  order-service:latest
```

## Testing

### Run Unit Tests
```bash
go test ./internal/domain/aggregates/... -v
```

### Run Integration Tests
```bash
# Requires PostgreSQL running
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/order_test?sslmode=disable"
go test ./internal/eventstore/... -v
```

### Run All Tests
```bash
go test ./... -v
```

## Database Schema

### Event Store Table
```sql
CREATE TABLE events (
    id VARCHAR(36) PRIMARY KEY,
    aggregate_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    version INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    INDEX idx_aggregate_id (aggregate_id),
    INDEX idx_event_type (event_type),
    INDEX idx_timestamp (timestamp),
    UNIQUE (aggregate_id, version)
);
```

### Read Model Tables
```sql
CREATE TABLE order_read_model (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    customer_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    total_amount DECIMAL(15, 2) NOT NULL,
    -- ... more fields
    INDEX idx_order_tenant (tenant_id),
    INDEX idx_order_customer (customer_id),
    INDEX idx_order_status (status)
);

CREATE TABLE order_item_read_model (
    id VARCHAR(100) PRIMARY KEY,
    order_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    -- ... more fields
    FOREIGN KEY (order_id) REFERENCES order_read_model(id)
);
```

## Dependencies

- Go 1.21+
- PostgreSQL 14+
- Kafka (optional, can be disabled)

## Project Structure

```
order-service/
├── cmd/
│   └── server/           # Application entry point
│       └── main.go
├── internal/
│   ├── api/              # HTTP handlers and routing
│   │   ├── command_handlers.go
│   │   ├── query_handlers.go
│   │   ├── router.go
│   │   └── dtos.go
│   ├── config/           # Configuration management
│   │   └── config.go
│   ├── domain/           # Domain layer
│   │   ├── aggregates/   # Aggregate roots
│   │   │   └── order.go
│   │   ├── commands/     # Command definitions and handlers
│   │   │   ├── commands.go
│   │   │   └── handler.go
│   │   ├── events/       # Domain events
│   │   │   └── events.go
│   │   └── queries/      # Query models
│   │       └── models.go
│   ├── eventstore/       # Event persistence
│   │   ├── eventstore.go
│   │   └── eventstore_with_kafka.go
│   ├── messaging/        # Kafka integration
│   │   ├── kafka_publisher.go
│   │   └── kafka_consumer.go
│   ├── projection/       # Read model projections
│   │   └── projection.go
│   └── saga/             # Saga orchestration
│       └── order_saga.go
├── Dockerfile
├── .env.example
└── README.md
```

## Key Design Decisions

1. **Event Sourcing**: Provides complete audit trail and enables temporal queries
2. **CQRS**: Optimizes read and write operations independently
3. **Saga Pattern**: Manages distributed transactions without 2PC
4. **Optimistic Concurrency**: Version-based conflict detection prevents lost updates
5. **Kafka Integration**: Enables event-driven architecture and service decoupling

## Future Enhancements

- Event snapshots for performance optimization
- Read model caching with Redis
- Dead letter queue for failed events
- Event replay capabilities
- GraphQL API support
