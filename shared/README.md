# Shared Libraries

Common libraries and utilities for all microservices in the e-commerce platform.

## 📁 Structure

```
shared/
├── go/                          # Go shared libraries
│   ├── pkg/
│   │   ├── logger/             # Structured logging
│   │   ├── middleware/         # HTTP middleware (CORS, Request ID, Tenant, Error handling)
│   │   ├── errors/             # Application errors
│   │   ├── response/           # HTTP response helpers
│   │   ├── pagination/         # Pagination utilities
│   │   ├── database/           # PostgreSQL utilities
│   │   ├── kafka/              # Kafka producer/consumer
│   │   ├── config/             # Configuration utilities
│   │   └── validator/          # Request validation
│   ├── go.mod
│   └── README.md
│
└── dotnet/                      # .NET shared libraries
    ├── Ecommerce.Shared/
    │   ├── Logging/            # Serilog logging
    │   ├── Middleware/         # ASP.NET Core middleware
    │   ├── Exceptions/         # Application exceptions
    │   ├── Models/             # Response models
    │   ├── Pagination/         # Pagination utilities
    │   ├── Kafka/              # Kafka producer/consumer
    │   ├── Configuration/      # Configuration utilities
    │   └── Ecommerce.Shared.csproj
    └── README.md
```

## 🎯 Purpose

These shared libraries provide:

1. **Consistency** - Uniform patterns across all microservices
2. **Reusability** - DRY principle (Don't Repeat Yourself)
3. **Maintainability** - Single place to update common functionality
4. **Best Practices** - Vetted patterns and utilities
5. **Developer Experience** - Faster service development

## 🔧 Components

### Go Libraries

- **Logger**: Structured logging with logrus
- **Middleware**: CORS, Request ID, Tenant identification, Error handling, Recovery
- **Errors**: Application error types and helpers
- **Response**: Standardized HTTP responses
- **Pagination**: Query parameter parsing and pagination helpers
- **Database**: PostgreSQL connection, transactions, migrations
- **Kafka**: Producer and consumer utilities
- **Config**: Environment variable helpers
- **Validator**: Request validation with custom rules

### .NET Libraries

- **Logging**: Serilog integration with structured logging
- **Middleware**: Request ID, Tenant identification, Error handling
- **Exceptions**: Custom exception types
- **Models**: API response models (success, error, paginated)
- **Pagination**: Request parameter parsing
- **Kafka**: Confluent Kafka producer/consumer
- **Configuration**: Configuration helpers and extensions

## 📖 Usage

### Go Services

```go
import "github.com/ecommerce/shared/go/pkg/logger"
import "github.com/ecommerce/shared/go/pkg/middleware"
import "github.com/ecommerce/shared/go/pkg/database"
```

See [Go README](./go/README.md) for detailed documentation.

### .NET Services

```xml
<ProjectReference Include="../../shared/dotnet/Ecommerce.Shared/Ecommerce.Shared.csproj" />
```

See [.NET README](./dotnet/README.md) for detailed documentation.

## 🚀 Quick Start

### For Go Services

1. Add to `go.mod`:
   ```go
   replace github.com/ecommerce/shared/go => ../../shared/go
   ```

2. Import and use:
   ```go
   import (
       "github.com/ecommerce/shared/go/pkg/logger"
       "github.com/ecommerce/shared/go/pkg/middleware"
   )

   log := logger.Get()
   router.Use(middleware.RequestID())
   ```

### For .NET Services

1. Add project reference to `.csproj`:
   ```xml
   <ProjectReference Include="../../shared/dotnet/Ecommerce.Shared/Ecommerce.Shared.csproj" />
   ```

2. Use in code:
   ```csharp
   using Ecommerce.Shared.Middleware;
   using Ecommerce.Shared.Logging;

   builder.Services.AddCustomLogging(builder.Configuration);
   app.UseRequestId();
   ```

## ✨ Features

### Multi-Tenancy Support

Both libraries provide tenant identification via:
- HTTP headers (`X-Tenant-ID`, `X-Tenant-Slug`)
- Subdomains (`tenant1.api.example.com`)
- URL path parameters

### Observability

- Structured JSON logging
- Request ID tracking across services
- Tenant-aware logging
- Error tracking with context

### Standardization

- Consistent error responses
- Uniform pagination
- Common validation rules
- Shared data models

### Event-Driven

- Kafka producer/consumer utilities
- Message serialization/deserialization
- Header support for metadata

## 🧪 Testing

### Go

```bash
cd shared/go
go test ./...
```

### .NET

```bash
cd shared/dotnet
dotnet test
```

## 📝 Contributing

When adding new shared functionality:

1. Ensure it's truly common across multiple services
2. Write comprehensive tests
3. Document with examples
4. Update relevant README
5. Consider backward compatibility

## 🔄 Versioning

Shared libraries follow semantic versioning:
- **Major**: Breaking changes
- **Minor**: New features (backward compatible)
- **Patch**: Bug fixes

## 📚 Documentation

- [Go Shared Libraries](./go/README.md)
- [.NET Shared Libraries](./dotnet/README.md)

## 🎓 Examples

See the following services for usage examples:
- **Tenant Service** (Go) - `/services/tenant-service`
- **Inventory Service** (.NET) - `/services/inventory-service` *(coming soon)*
- **Payment Service** (.NET) - `/services/payment-service` *(coming soon)*

## 📄 License

MIT
