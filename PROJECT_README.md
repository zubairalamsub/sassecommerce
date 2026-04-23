# E-Commerce Platform - Implementation Guide

This is the implementation of the multi-tenant e-commerce platform using **Go** and **.NET** microservices.

## 🏗️ Architecture

- **Backend Services**: Go (Gin) and .NET (ASP.NET Core)
- **Databases**: PostgreSQL, MongoDB, Redis, Elasticsearch
- **Message Queue**: Apache Kafka
- **Infrastructure**: Docker, Kubernetes, Terraform

## 📁 Project Structure

```
/Volumes/D/Ecommerce/
├── services/                    # Microservices
│   ├── tenant-service/         # ✅ Tenant management (Go)
│   ├── user-service/           # User authentication (Go)
│   ├── product-service/        # Product catalog (Go)
│   ├── order-service/          # Order processing (Go)
│   ├── inventory-service/      # Inventory management (.NET)
│   ├── payment-service/        # Payment processing (.NET)
│   ├── analytics-service/      # Analytics (.NET)
│   ├── notification-service/   # Notifications (Go)
│   ├── review-service/         # Reviews (Go)
│   ├── promotion-service/      # Promotions (Go)
│   └── shipping-service/       # Shipping (Go)
├── shared/                     # Shared libraries
│   ├── go-common/             # Go shared code
│   ├── dotnet-common/         # .NET shared code
│   └── proto/                 # Protocol buffers
├── infrastructure/             # Infrastructure code
│   ├── docker/                # Docker configs
│   ├── kubernetes/            # K8s manifests
│   ├── terraform/             # IaC
│   └── monitoring/            # Monitoring configs
├── frontend/                   # Frontend applications
└── docker-compose.yml         # Local development setup
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+
- .NET 8.0 SDK
- Make (optional)

### 1. Start Infrastructure Services

```bash
# Start all infrastructure services (PostgreSQL, MongoDB, Redis, Kafka, Elasticsearch)
docker-compose up -d postgres mongodb redis zookeeper kafka elasticsearch

# Check if all services are healthy
docker-compose ps
```

### 2. Run Tenant Service

#### Option A: Using Docker

```bash
# Build and run tenant service
docker-compose up -d tenant-service

# View logs
docker-compose logs -f tenant-service
```

#### Option B: Run Locally (for development)

```bash
cd services/tenant-service

# Install dependencies
go mod download

# Copy environment variables
cp .env.example .env

# Run the service
go run cmd/server/main.go
```

The Tenant Service will be available at `http://localhost:8081`

### 3. Test the Tenant Service

```bash
# Health check
curl http://localhost:8081/health

# Create a new tenant
curl -X POST http://localhost:8081/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Store",
    "email": "store@example.com",
    "tier": "free"
  }'

# List all tenants
curl http://localhost:8081/api/v1/tenants

# Get tenant by ID
curl http://localhost:8081/api/v1/tenants/{tenant-id}
```

## 📦 Implemented Services

### ✅ Tenant Service (Go)

**Status**: Complete with Full Test Coverage & Audit Logging
**Port**: 8081
**Database**: PostgreSQL (tenant_db)

**Features**:
- ✅ Create, read, update, delete tenants
- ✅ Multi-tenancy support (Pool, Bridge, Silo models)
- ✅ Tier-based limits (Free, Starter, Professional, Enterprise)
- ✅ Tenant configuration management
- ✅ Event publishing to Kafka
- ✅ **Comprehensive Audit Logging** - All API requests logged with full context
- ✅ **Complete Test Suite** - 35+ tests (Unit, Integration, E2E)
- ✅ **80%+ Code Coverage**

**API Endpoints**:
- `POST /api/v1/tenants` - Create tenant
- `GET /api/v1/tenants` - List tenants
- `GET /api/v1/tenants/:id` - Get tenant by ID
- `GET /api/v1/tenants/slug/:slug` - Get tenant by slug
- `GET /api/v1/tenants/domain` - Get tenant by domain
- `PUT /api/v1/tenants/:id` - Update tenant
- `PATCH /api/v1/tenants/:id/config` - Update tenant config
- `DELETE /api/v1/tenants/:id` - Delete tenant

## 🛠️ Development

### Building Services

#### Go Services

```bash
cd services/tenant-service
go build -o bin/tenant-service ./cmd/server
./bin/tenant-service
```

#### .NET Services

```bash
cd services/inventory-service
dotnet restore
dotnet build
dotnet run
```

### Running Tests

```bash
# Go service tests
cd services/tenant-service
go test ./...

# .NET service tests
cd services/inventory-service
dotnet test
```

### Database Migrations

Migrations are automatically run on service startup using GORM AutoMigrate.

To manually run migrations:

```bash
cd services/tenant-service
go run cmd/server/main.go
```

## 📊 Infrastructure Services

| Service | Port | Purpose |
|---------|------|---------|
| PostgreSQL | 5432 | Primary database |
| MongoDB | 27017 | Document storage |
| Redis | 6379 | Cache & sessions |
| Kafka | 9092 | Event streaming |
| Zookeeper | 2181 | Kafka coordination |
| Elasticsearch | 9200 | Search engine |

### Database Credentials

**PostgreSQL**:
- User: `postgres`
- Password: `postgres`
- Databases: `tenant_db`, `user_db`, `product_db`, etc.

**MongoDB**:
- User: `admin`
- Password: `admin123`

**Redis**:
- Password: `redis123`

## 🔍 Monitoring & Debugging

### View Service Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f tenant-service
docker-compose logs -f kafka
```

### Access Databases

```bash
# PostgreSQL
docker exec -it ecommerce-postgres psql -U postgres -d tenant_db

# MongoDB
docker exec -it ecommerce-mongodb mongosh -u admin -p admin123

# Redis
docker exec -it ecommerce-redis redis-cli -a redis123
```

### Kafka Topics

```bash
# List topics
docker exec -it ecommerce-kafka kafka-topics --bootstrap-server localhost:9092 --list

# Consume messages from tenant-events topic
docker exec -it ecommerce-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic tenant-events \
  --from-beginning
```

## 🧪 Testing

### Running Tests

```bash
# All tests
make test-tenant

# Unit tests only
make test-tenant-unit

# E2E tests only
make test-tenant-e2e

# Generate coverage report
make test-tenant-coverage

# Race detection
make test-tenant-race

# Or use the test script
cd services/tenant-service
./scripts/run_tests.sh
```

### Test Suite Overview

| Test Type | Count | Coverage |
|-----------|-------|----------|
| Repository Unit Tests | 11 | Data access layer |
| Service Unit Tests | 11 | Business logic with mocks |
| E2E API Tests | 13 | Complete HTTP flows |
| **Total** | **35** | **80%+** |

**Test Coverage Includes**:
- ✅ Create tenant (success & validation errors)
- ✅ Get tenant by ID, slug, domain
- ✅ List tenants with pagination
- ✅ Update tenant and configuration
- ✅ Delete tenant (soft delete)
- ✅ Database operations
- ✅ Service layer business logic
- ✅ API request/response handling
- ✅ Error scenarios

## 📊 Audit Logging

Every API request is automatically logged with complete context:

### What's Logged

- **Request Details**: Method, path, query params, request body
- **Response Details**: Status code, duration in milliseconds
- **Context**: Tenant ID, user ID (when authenticated)
- **Client Info**: IP address, user agent
- **Changes**: Old and new values for UPDATE operations
- **Errors**: Full error messages and stack traces

### Audit Log Schema

```sql
CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY,
    tenant_id    UUID,
    user_id      UUID,
    action       VARCHAR(100),    -- CREATE, READ, UPDATE, DELETE
    resource     VARCHAR(100),    -- tenant, product, order, etc.
    resource_id  UUID,
    method       VARCHAR(10),     -- GET, POST, PUT, DELETE
    path         VARCHAR(500),
    ip_address   VARCHAR(45),
    user_agent   VARCHAR(500),
    request_body TEXT,
    response_code INT,
    old_value    JSONB,          -- Previous value (for updates)
    new_value    JSONB,          -- New value (for updates)
    metadata     JSONB,
    error_message TEXT,
    duration_ms  BIGINT,
    created_at   TIMESTAMP
);

-- Indexes for fast querying
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
```

### Viewing Audit Logs

```bash
# Access database
make db-psql-tenant

# Query recent audit logs
SELECT
    action,
    resource,
    method,
    path,
    response_code,
    duration_ms,
    created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT 10;

# Query by tenant
SELECT * FROM audit_logs
WHERE tenant_id = 'your-tenant-id'
ORDER BY created_at DESC;

# Query failed requests
SELECT * FROM audit_logs
WHERE response_code >= 400
ORDER BY created_at DESC;

# Query slow requests (> 1 second)
SELECT * FROM audit_logs
WHERE duration_ms > 1000
ORDER BY duration_ms DESC;
```

### Example Audit Log Entry

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "123e4567-e89b-12d3-a456-426614174000",
  "action": "UPDATE",
  "resource": "tenant",
  "resource_id": "123e4567-e89b-12d3-a456-426614174000",
  "method": "PUT",
  "path": "/api/v1/tenants/123e4567-e89b-12d3-a456-426614174000",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "request_body": "{\"name\":\"Updated Name\"}",
  "response_code": 200,
  "old_value": "{\"name\":\"Old Name\",\"status\":\"active\"}",
  "new_value": "{\"name\":\"Updated Name\",\"status\":\"active\"}",
  "duration_ms": 45,
  "created_at": "2024-04-16T10:30:00Z"
}
```

## 📝 Next Steps

1. ✅ **Tenant Service** - Complete with Tests & Audit Logging
2. **User Service** - Implement authentication & authorization
3. **Product Service** - Product catalog management
4. **Order Service** - Order processing with Event Sourcing
5. **Inventory Service (.NET)** - Stock management
6. **Payment Service (.NET)** - Payment processing
7. **Other Services** - Notification, Review, Promotion, Shipping, Analytics

## 🔗 Documentation

For detailed architecture and design documentation, see:

- [SYSTEM_DESIGN.md](./SYSTEM_DESIGN.md) - Complete system architecture
- [MULTI_TENANCY_ARCHITECTURE.md](./MULTI_TENANCY_ARCHITECTURE.md) - Multi-tenancy design
- [TECHNOLOGY_STACK.md](./TECHNOLOGY_STACK.md) - Technology choices
- [DEVELOPMENT_CHECKLIST.md](./DEVELOPMENT_CHECKLIST.md) - Development tasks

## 🤝 Contributing

1. Create a feature branch
2. Make your changes
3. Write tests
4. Submit a pull request

## 📄 License

This project is proprietary.
