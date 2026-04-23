# Go Shared Libraries

Common packages and utilities for Go microservices in the e-commerce platform.

## 📦 Packages

### Logger

Structured logging with logrus.

```go
import "github.com/ecommerce/shared/go/pkg/logger"

// Initialize logger
log := logger.New(logger.Config{
    Level:       "info",
    Format:      "json",
    Output:      "stdout",
    ServiceName: "user-service",
})

// Use logger
log.Info("Server started")
log.WithRequestID("123").Error("Request failed")
log.WithTenantID("tenant-1").WithUserID("user-1").Info("User action")
```

### Middleware

Common HTTP middleware for Gin framework.

#### CORS

```go
import "github.com/ecommerce/shared/go/pkg/middleware"

router := gin.Default()
router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
```

#### Request ID

```go
router.Use(middleware.RequestID())

// Get request ID in handler
requestID := middleware.GetRequestID(c)
```

#### Tenant

```go
router.Use(middleware.Tenant(middleware.DefaultTenantConfig()))

// Get tenant info in handler
tenantID := middleware.GetTenantID(c)
tenantSlug := middleware.GetTenantSlug(c)
```

#### Error Handler & Recovery

```go
router.Use(middleware.Recovery(log))
router.Use(middleware.ErrorHandler(log))
```

### Errors

Application error handling.

```go
import "github.com/ecommerce/shared/go/pkg/errors"

// Create errors
err := errors.NotFound("Product")
err := errors.BadRequest("Invalid input")
err := errors.ValidationError("Email is required").WithDetails(validationErrors)

// Check error type
if errors.IsAppError(err) {
    appErr := errors.GetAppError(err)
    // Use appErr.Code, appErr.Status, etc.
}
```

### Response

Standardized HTTP responses.

```go
import "github.com/ecommerce/shared/go/pkg/response"

// Success responses
response.Success(c, product)
response.Created(c, newProduct)
response.NoContent(c)

// Paginated response
response.Paginated(c, products, page, pageSize, totalItems)

// Error responses
response.Error(c, err)
response.NotFound(c, "Product")
response.BadRequest(c, "Invalid input")
response.ValidationError(c, validationErrors)
```

### Pagination

Pagination utilities.

```go
import "github.com/ecommerce/shared/go/pkg/pagination"

// Get pagination params from request
params := pagination.GetPaginationParams(c)

// Use in database query
products, total, err := repo.List(ctx, params.Offset, params.PageSize)

// Get ORDER BY clause
orderBy := params.GetOrderBy() // "created_at desc"
```

### Database

PostgreSQL utilities with GORM.

```go
import "github.com/ecommerce/shared/go/pkg/database"

// Connect to PostgreSQL
config := database.DefaultPostgresConfig()
config.Database = "user_db"
config.Host = "localhost"

db, err := database.NewPostgresDB(config)

// Auto migrate
database.AutoMigrate(db, &User{}, &Profile{})

// Use transactions
err = database.Transaction(db, func(tx *gorm.DB) error {
    // Your transactional logic
    return nil
})

// Close connection
database.CloseDB(db)
```

### Kafka

Kafka producer and consumer.

#### Producer

```go
import "github.com/ecommerce/shared/go/pkg/kafka"

// Create producer
config := kafka.DefaultProducerConfig([]string{"localhost:9092"}, "events")
producer := kafka.NewProducer(config)
defer producer.Close()

// Publish message
err := producer.Publish(ctx, "user-123", event)

// Publish with headers
headers := map[string]string{
    "event_type": "user.created",
    "version":    "1.0",
}
err := producer.PublishWithHeaders(ctx, "user-123", event, headers)
```

#### Consumer

```go
// Create consumer
config := kafka.DefaultConsumerConfig(
    []string{"localhost:9092"},
    "events",
    "user-service-group",
)
consumer := kafka.NewConsumer(config, log)
defer consumer.Close()

// Consume messages
handler := func(ctx context.Context, msg kafka.Message) error {
    var event UserEvent
    if err := kafka.UnmarshalMessage(msg, &event); err != nil {
        return err
    }
    // Process event
    return nil
}

consumer.Consume(ctx, handler)
```

### Configuration

Environment variable utilities.

```go
import "github.com/ecommerce/shared/go/pkg/config"

// Get string
dbHost := config.GetEnv("DB_HOST", "localhost")

// Get int
dbPort := config.GetEnvAsInt("DB_PORT", 5432)

// Get bool
enableSSL := config.GetEnvAsBool("ENABLE_SSL", false)

// Get duration
timeout := config.GetEnvAsDuration("TIMEOUT", 30*time.Second)

// Get slice
brokers := config.GetEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}, ",")

// Required value (panics if not set)
apiKey := config.MustGetEnv("API_KEY")

// Environment checks
if config.IsProduction() {
    // Production-specific logic
}
```

### Validator

Request validation with go-playground/validator.

```go
import "github.com/ecommerce/shared/go/pkg/validator"

// Create validator
v := validator.New()

// Validate struct
type CreateUserRequest struct {
    Email    string `validate:"required,email"`
    Password string `validate:"required,min=8"`
    Age      int    `validate:"required,gte=18"`
    Slug     string `validate:"required,slug"`
}

req := CreateUserRequest{...}
if err := v.Validate(req); err != nil {
    errors := validator.FormatValidationErrors(err)
    // Return formatted errors to client
}
```

## 🚀 Usage in Services

Add to your service's `go.mod`:

```go
require github.com/ecommerce/shared/go v0.1.0
```

Or use replace directive for local development:

```go
replace github.com/ecommerce/shared/go => ../../shared/go
```

## 📝 Example Service

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ecommerce/shared/go/pkg/logger"
    "github.com/ecommerce/shared/go/pkg/middleware"
    "github.com/ecommerce/shared/go/pkg/database"
    "github.com/ecommerce/shared/go/pkg/response"
)

func main() {
    // Initialize logger
    log := logger.New(logger.Config{
        Level:       "info",
        Format:      "json",
        ServiceName: "user-service",
    })

    // Connect to database
    db, err := database.NewPostgresDB(database.DefaultPostgresConfig())
    if err != nil {
        log.Fatal(err)
    }
    defer database.CloseDB(db)

    // Create router
    router := gin.Default()

    // Add middleware
    router.Use(middleware.RequestID())
    router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
    router.Use(middleware.Tenant(middleware.DefaultTenantConfig()))
    router.Use(middleware.Recovery(log.Logger))
    router.Use(middleware.ErrorHandler(log.Logger))

    // Add routes
    router.GET("/users", func(c *gin.Context) {
        users := []User{...}
        response.Success(c, users)
    })

    // Start server
    log.Info("Server starting on :8080")
    router.Run(":8080")
}
```

## 🧪 Testing

All packages include comprehensive test coverage. Run tests:

```bash
cd shared/go
go test ./...
```

## 📄 License

MIT
