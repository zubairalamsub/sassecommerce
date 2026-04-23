# User Service

Authentication and user management microservice with JWT, RBAC, and multi-tenant support.

## 📋 Overview

The User Service handles:
- User registration and authentication
- JWT token generation and validation
- Role-Based Access Control (RBAC)
- User profile management
- Password management
- Multi-tenant user isolation

## ✨ Features

- **JWT Authentication** - Secure token-based authentication
- **Password Hashing** - BCrypt password encryption
- **Role-Based Access** - Admin, Moderator, Customer, Guest roles
- **Multi-Tenant Support** - Complete tenant isolation
- **RESTful API** - 10 well-designed endpoints
- **Comprehensive Testing** - 23+ tests with 80%+ coverage
- **PostgreSQL Database** - Relational data storage with GORM

## 🏗️ Architecture

```
user-service/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── models/
│   │   ├── user.go                # User model and DTOs
│   │   └── token.go               # JWT token models
│   ├── repository/
│   │   ├── user_repository.go     # Data access layer
│   │   ├── user_repository_test.go # Repository tests (12 tests)
│   │   └── mocks/
│   │       └── user_repository_mock.go
│   ├── service/
│   │   ├── auth_service.go        # Authentication logic
│   │   ├── auth_service_test.go   # Auth service tests (11 tests)
│   │   ├── user_service.go        # User management logic
│   │   └── user_service_test.go
│   ├── middleware/
│   │   └── auth.go                # JWT authentication middleware
│   └── api/
│       ├── auth_handler.go        # Auth HTTP handlers
│       └── user_handler.go        # User HTTP handlers
├── tests/
│   └── e2e/
│       └── user_api_test.go       # E2E API tests
├── Dockerfile
├── go.mod
└── .env.example
```

## 🚀 API Endpoints

### Authentication

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/auth/register` | Register new user | No |
| POST | `/api/v1/auth/login` | Login and get JWT token | No |
| GET | `/api/v1/auth/profile` | Get current user profile | Yes |
| POST | `/api/v1/auth/change-password` | Change password | Yes |

### User Management

| Method | Endpoint | Description | Auth Required | Role |
|--------|----------|-------------|---------------|------|
| GET | `/api/v1/users` | List users (paginated) | Yes | Any |
| GET | `/api/v1/users/:id` | Get user by ID | Yes | Any |
| PUT | `/api/v1/users/:id` | Update user profile | Yes | Self/Admin |
| DELETE | `/api/v1/users/:id` | Delete user (soft delete) | Yes | Self |
| PATCH | `/api/v1/users/:id/role` | Update user role | Yes | Admin |
| PATCH | `/api/v1/users/:id/status` | Update user status | Yes | Admin |

## 📊 User Model

```go
type User struct {
    ID            string      // UUID
    TenantID      string      // Tenant identifier
    Email         string      // Unique within tenant
    Username      string      // Unique within tenant
    PasswordHash  string      // BCrypt hashed
    FirstName     string
    LastName      string
    Phone         string
    Avatar        string
    Status        UserStatus  // active, inactive, suspended, deleted
    Role          UserRole    // admin, moderator, customer, guest
    EmailVerified bool
    LastLoginAt   *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
    DeletedAt     *time.Time
}
```

## 🔐 User Roles

- **Admin** - Full system access, can manage all users
- **Moderator** - Limited administrative access
- **Customer** - Standard user access
- **Guest** - Read-only access

## 💻 Usage Examples

### Register

```bash
curl -X POST http://localhost:8082/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant-uuid",
    "email": "user@example.com",
    "username": "johndoe",
    "password": "password123",
    "first_name": "John",
    "last_name": "Doe"
  }'
```

### Login

```bash
curl -X POST http://localhost:8082/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant-uuid",
    "email": "user@example.com",
    "password": "password123"
  }'
```

Response:
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user-uuid",
      "email": "user@example.com",
      "first_name": "John",
      "role": "customer"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-04-17T10:30:00Z"
  }
}
```

### Get Profile

```bash
curl http://localhost:8082/api/v1/auth/profile \
  -H "Authorization: Bearer <your-jwt-token>"
```

### List Users

```bash
curl "http://localhost:8082/api/v1/users?page=1&page_size=20" \
  -H "Authorization: Bearer <your-jwt-token>"
```

## 🧪 Testing

### Run All Tests

```bash
cd services/user-service
go test -v ./...
```

### Run Specific Test Suites

```bash
# Repository tests only
go test -v ./internal/repository/...

# Service tests only
go test -v ./internal/service/...

# E2E tests only
go test -v ./tests/e2e/...
```

### Test Coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Test Statistics

- **Total Tests**: 23+
- **Repository Tests**: 12 (100% pass)
- **Service Tests**: 11 (100% pass)
- **Coverage**: 80%+

## 🐳 Docker

### Build

```bash
docker build -t user-service:latest .
```

### Run

```bash
docker run -p 8082:8082 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=postgres \
  -e DB_NAME=user_db \
  -e JWT_SECRET=your-secret-key \
  user-service:latest
```

## ⚙️ Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP server port | 8082 |
| DB_HOST | PostgreSQL host | localhost |
| DB_PORT | PostgreSQL port | 5432 |
| DB_USER | Database user | postgres |
| DB_PASSWORD | Database password | postgres |
| DB_NAME | Database name | user_db |
| JWT_SECRET | JWT signing secret | (required) |
| ENVIRONMENT | Environment (development/production) | development |
| LOG_LEVEL | Logging level | info |
| LOG_FORMAT | Log format (json/text) | json |

## 🔒 Security Features

1. **Password Hashing** - BCrypt with cost factor 10
2. **JWT Tokens** - Signed with HS256 algorithm
3. **Token Expiration** - 24-hour validity
4. **Role-Based Access** - Route-level authorization
5. **Soft Delete** - User data retained for audit
6. **Email Uniqueness** - Per-tenant email validation

## 📝 JWT Token Structure

```json
{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "email": "user@example.com",
  "role": "customer",
  "exp": 1713355800,
  "iat": 1713269400,
  "iss": "user-service",
  "sub": "uuid"
}
```

## 🔄 Database Schema

```sql
CREATE TABLE users (
    id VARCHAR PRIMARY KEY,
    tenant_id VARCHAR NOT NULL,
    email VARCHAR NOT NULL,
    username VARCHAR,
    password_hash VARCHAR NOT NULL,
    first_name VARCHAR,
    last_name VARCHAR,
    phone VARCHAR,
    avatar VARCHAR,
    status VARCHAR DEFAULT 'active',
    role VARCHAR DEFAULT 'customer',
    email_verified BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    UNIQUE(tenant_id, email),
    UNIQUE(tenant_id, username)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_deleted ON users(deleted_at);
```

## 🚦 Health Check

```bash
curl http://localhost:8082/health
```

Response:
```json
{
  "status": "healthy",
  "service": "user-service",
  "time": "2024-04-16T10:30:00Z"
}
```

## 📚 Dependencies

- **Gin** - HTTP web framework
- **GORM** - ORM for PostgreSQL
- **JWT** - golang-jwt/jwt/v5
- **BCrypt** - golang.org/x/crypto/bcrypt
- **Logrus** - Structured logging
- **Testify** - Testing framework
- **UUID** - google/uuid

## 🔗 Related Services

- **Tenant Service** - Tenant management (port 8081)
- **Product Service** - Product catalog (port 8083)
- **Order Service** - Order processing (port 8084)

## 📄 License

MIT

---

**User Service** - Part of the E-Commerce Microservices Platform
