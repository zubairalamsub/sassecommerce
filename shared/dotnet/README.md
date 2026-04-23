# .NET Shared Libraries

Common libraries and utilities for .NET microservices in the e-commerce platform.

## 📦 Package

**Ecommerce.Shared** - Common utilities for ASP.NET Core services

## 🚀 Installation

Add package reference to your .csproj:

```xml
<ItemGroup>
  <ProjectReference Include="../../shared/dotnet/Ecommerce.Shared/Ecommerce.Shared.csproj" />
</ItemGroup>
```

## 📋 Features

### Logging

Structured logging with Serilog.

```csharp
using Ecommerce.Shared.Logging;

// In Program.cs
builder.Services.AddCustomLogging(builder.Configuration);

// Use in services
public class UserService
{
    private readonly ILogger<UserService> _logger;

    public UserService(ILogger<UserService> logger)
    {
        _logger = logger;
    }

    public void CreateUser(User user)
    {
        _logger.LogInformation("Creating user {UserId}", user.Id);
    }
}
```

**Configuration** (appsettings.json):

```json
{
  "ServiceName": "inventory-service",
  "Logging": {
    "LogLevel": {
      "Default": "Information"
    },
    "Format": "json",
    "Output": "console"
  }
}
```

### Middleware

#### Request ID Middleware

```csharp
using Ecommerce.Shared.Middleware;

// In Program.cs
app.UseRequestId();

// Get request ID in controllers
var requestId = HttpContext.GetRequestId();
```

#### Tenant Middleware

```csharp
// In Program.cs
app.UseTenant(options =>
{
    options.Required = true;
    options.AllowHeader = true;
    options.AllowSubdomain = true;
    options.AllowPath = true;
});

// Get tenant info in controllers
var tenantId = HttpContext.GetTenantId();
var tenantSlug = HttpContext.GetTenantSlug();
```

#### Error Handling Middleware

```csharp
// In Program.cs
app.UseErrorHandling();

// Throw exceptions in services
throw CommonExceptions.NotFound("Product");
throw CommonExceptions.BadRequest("Invalid input");
throw CommonExceptions.ValidationError("Validation failed", validationErrors);
```

### Exceptions

Custom application exceptions.

```csharp
using Ecommerce.Shared.Exceptions;

// Throw exceptions
throw CommonExceptions.NotFound("Product");
throw CommonExceptions.BadRequest("Invalid input");
throw CommonExceptions.Unauthorized();
throw CommonExceptions.Forbidden("Access denied");
throw CommonExceptions.Conflict("Product already exists");
throw CommonExceptions.ValidationError("Validation failed", errors);
throw CommonExceptions.InternalError();
throw CommonExceptions.ServiceUnavailable("Payment");
throw CommonExceptions.TooManyRequests();

// Custom exception
throw new AppException(
    "CUSTOM_ERROR",
    "Custom error message",
    HttpStatusCode.BadRequest,
    new { field = "value" }
);
```

### API Responses

Standardized response models.

```csharp
using Ecommerce.Shared.Models;

// Success response
return Ok(ApiResponse<Product>.SuccessResponse(product));
return Ok(ApiResponse<Product>.SuccessResponse(product, "Product created"));

// Paginated response
var response = PaginatedResponse<Product>.Create(
    products,
    page,
    pageSize,
    totalItems
);
return Ok(response);
```

### Pagination

Pagination utilities.

```csharp
using Ecommerce.Shared.Pagination;

// In controller
[HttpGet]
public async Task<IActionResult> GetProducts()
{
    var paginationParams = PaginationParams.FromQuery(Request);

    var products = await _context.Products
        .Skip(paginationParams.Offset)
        .Take(paginationParams.PageSize)
        .OrderBy(p => EF.Property<object>(p, paginationParams.SortBy))
        .ToListAsync();

    var total = await _context.Products.CountAsync();

    return Ok(PaginatedResponse<Product>.Create(
        products,
        paginationParams.Page,
        paginationParams.PageSize,
        total
    ));
}
```

### Kafka

Kafka producer and consumer.

#### Producer

```csharp
using Ecommerce.Shared.Kafka;

// Register in DI
builder.Services.AddSingleton<IKafkaProducer>(sp =>
{
    var logger = sp.GetRequiredService<ILogger<KafkaProducer>>();
    var config = new KafkaProducerConfig
    {
        Brokers = new List<string> { "localhost:9092" }
    };
    return new KafkaProducer(config, logger);
});

// Use in services
public class InventoryService
{
    private readonly IKafkaProducer _kafkaProducer;

    public async Task UpdateStock(UpdateStockEvent evt)
    {
        // Update stock...

        // Publish event
        await _kafkaProducer.PublishAsync(
            "inventory-events",
            evt.ProductId,
            evt
        );

        // With headers
        var headers = new Dictionary<string, string>
        {
            { "event_type", "stock.updated" },
            { "version", "1.0" }
        };
        await _kafkaProducer.PublishAsync(
            "inventory-events",
            evt.ProductId,
            evt,
            headers
        );
    }
}
```

#### Consumer

```csharp
// Implement message handler
public class StockEventHandler : IMessageHandler<StockUpdatedEvent>
{
    private readonly ILogger<StockEventHandler> _logger;

    public async Task HandleAsync(
        StockUpdatedEvent message,
        CancellationToken cancellationToken)
    {
        _logger.LogInformation("Processing stock update for product {ProductId}", message.ProductId);
        // Process event
    }
}

// Register consumer in DI
builder.Services.AddSingleton<IMessageHandler<StockUpdatedEvent>, StockEventHandler>();
builder.Services.AddHostedService(sp =>
{
    var config = new KafkaConsumerConfig
    {
        Brokers = new List<string> { "localhost:9092" },
        Topic = "inventory-events",
        GroupId = "payment-service-group"
    };
    var handler = sp.GetRequiredService<IMessageHandler<StockUpdatedEvent>>();
    var logger = sp.GetRequiredService<ILogger<KafkaConsumer<StockUpdatedEvent>>>();
    return new KafkaConsumer<StockUpdatedEvent>(config, handler, logger);
});
```

### Configuration

Configuration utilities.

```csharp
using Ecommerce.Shared.Configuration;

// Get required value (throws if not found)
var apiKey = configuration.GetRequiredValue("ApiKey");

// Get required section
var kafkaConfig = configuration.GetRequiredSection<KafkaConfig>("Kafka");

// Environment checks
if (configuration.IsProduction())
{
    // Production-specific logic
}

if (configuration.IsDevelopment())
{
    // Development-specific logic
}

// Get string array
var allowedOrigins = configuration.GetStringArray("Cors:AllowedOrigins");
```

## 📝 Complete Example

```csharp
using Ecommerce.Shared.Logging;
using Ecommerce.Shared.Middleware;
using Ecommerce.Shared.Kafka;
using Ecommerce.Shared.Configuration;

var builder = WebApplication.CreateBuilder(args);

// Add logging
builder.Services.AddCustomLogging(builder.Configuration);

// Add services
builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

// Add Kafka producer
builder.Services.AddSingleton<IKafkaProducer>(sp =>
{
    var logger = sp.GetRequiredService<ILogger<KafkaProducer>>();
    var config = new KafkaProducerConfig
    {
        Brokers = builder.Configuration.GetStringArray("Kafka:Brokers").ToList()
    };
    return new KafkaProducer(config, logger);
});

var app = builder.Build();

// Add middleware (order matters!)
app.UseErrorHandling();  // Must be first
app.UseRequestId();
app.UseTenant(options =>
{
    options.Required = true;
});

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.UseAuthorization();
app.MapControllers();

app.Run();
```

**Controller Example:**

```csharp
using Microsoft.AspNetCore.Mvc;
using Ecommerce.Shared.Models;
using Ecommerce.Shared.Pagination;
using Ecommerce.Shared.Exceptions;

[ApiController]
[Route("api/v1/[controller]")]
public class ProductsController : ControllerBase
{
    [HttpGet]
    public async Task<IActionResult> GetProducts()
    {
        var pagination = PaginationParams.FromQuery(Request);
        var products = await _service.GetProductsAsync(pagination);
        var total = await _service.GetTotalCountAsync();

        var response = PaginatedResponse<Product>.Create(
            products,
            pagination.Page,
            pagination.PageSize,
            total
        );

        return Ok(response);
    }

    [HttpGet("{id}")]
    public async Task<IActionResult> GetProduct(string id)
    {
        var product = await _service.GetProductByIdAsync(id);
        if (product == null)
        {
            throw CommonExceptions.NotFound("Product");
        }

        return Ok(ApiResponse<Product>.SuccessResponse(product));
    }

    [HttpPost]
    public async Task<IActionResult> CreateProduct(CreateProductRequest request)
    {
        var product = await _service.CreateProductAsync(request);
        return Created($"/api/v1/products/{product.Id}",
            ApiResponse<Product>.SuccessResponse(product, "Product created"));
    }
}
```

## 🧪 Testing

Run tests:

```bash
cd shared/dotnet
dotnet test
```

## 📄 License

MIT
