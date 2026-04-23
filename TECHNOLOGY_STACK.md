# Technology Stack - Detailed Specifications

## Overview

This document specifies the exact technologies, versions, and tools chosen for each component of the multi-tenant e-commerce platform.

---

## Table of Contents

1. [Backend Technologies](#backend-technologies)
2. [Databases](#databases)
3. [Frontend Technologies](#frontend-technologies)
4. [Message Queue & Events](#message-queue--events)
5. [Caching](#caching)
6. [Search](#search)
7. [Infrastructure & DevOps](#infrastructure--devops)
8. [Monitoring & Observability](#monitoring--observability)
9. [Security](#security)
10. [Third-Party Integrations](#third-party-integrations)
11. [Development Tools](#development-tools)

---

## Backend Technologies

### Programming Languages

#### Go (Golang)

**Version**: Go 1.21+

**Use For**:
- User Service
- Product Service
- Order Service
- Notification Service
- Search Service
- Review Service
- Promotion Service
- Tenant Service
- Shipping Service

**Justification**:
- ✅ Excellent performance
- ✅ Low memory footprint
- ✅ Built-in concurrency (goroutines)
- ✅ Fast compilation
- ✅ Static typing
- ✅ Great for high-throughput services
- ✅ Strong standard library
- ✅ Simple deployment (single binary)
- ✅ Excellent for microservices

**Key Libraries**:
```go
// go.mod
module github.com/yourorg/ecommerce

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/lib/pq v1.10.9
    go.mongodb.org/mongo-driver v1.13.1
    github.com/go-redis/redis/v8 v8.11.5
    github.com/segmentio/kafka-go v0.4.47
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/stretchr/testify v1.8.4
    github.com/go-playground/validator/v10 v10.16.0
    github.com/sirupsen/logrus v1.9.3
    gorm.io/gorm v1.25.5
    gorm.io/driver/postgres v1.5.4
    golang.org/x/crypto v0.17.0
)
```

---

#### .NET (C#)

**Version**: .NET 8.0 LTS

**Use For**:
- Inventory Service
- Payment Service
- Analytics Service

**Justification**:
- ✅ Excellent performance (comparable to Go)
- ✅ Strong typing with C#
- ✅ Mature ecosystem and tooling
- ✅ Built-in async/await for concurrency
- ✅ Great for CPU-intensive tasks
- ✅ Robust dependency injection
- ✅ Excellent ORM (Entity Framework Core)
- ✅ Perfect for high-throughput services
- ✅ Strong developer familiarity

**Key Libraries**:
```xml
<!-- .csproj -->
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.OpenApi" Version="8.0.0" />
    <PackageReference Include="Microsoft.EntityFrameworkCore" Version="8.0.0" />
    <PackageReference Include="Npgsql.EntityFrameworkCore.PostgreSQL" Version="8.0.0" />
    <PackageReference Include="StackExchange.Redis" Version="2.7.10" />
    <PackageReference Include="Confluent.Kafka" Version="2.3.0" />
    <PackageReference Include="Serilog.AspNetCore" Version="8.0.0" />
    <PackageReference Include="FluentValidation" Version="11.9.0" />
    <PackageReference Include="AutoMapper" Version="12.0.1" />
  </ItemGroup>
</Project>
```

---

### Backend Frameworks

#### Gin (Go)

**Version**: 1.9.1+

**Use For**: All Go services

**Configuration**:
```go
// main.go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "github.com/gin-contrib/gzip"
    "github.com/sirupsen/logrus"
)

func main() {
    // Set release mode
    gin.SetMode(gin.ReleaseMode)

    r := gin.New()

    // Middleware
    r.Use(gin.Logger())
    r.Use(gin.Recovery())

    // CORS
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Tenant-Id"},
        AllowCredentials: true,
    }))

    // Compression
    r.Use(gzip.Gzip(gzip.DefaultCompression))

    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })

    // Routes
    v1 := r.Group("/api/v1")
    {
        v1.POST("/users", createUser)
        v1.GET("/users/:id", getUser)
    }

    r.Run(":8080")
}
```

**Alternatives Considered**:
- Fiber: Fast but less mature
- Echo: Good but smaller community
- Chi: Lightweight but minimal features

---

#### ASP.NET Core (.NET)

**Version**: ASP.NET Core 8.0

**Use For**: All .NET services

**Configuration**:
```csharp
// Program.cs
using Microsoft.AspNetCore.Builder;
using Microsoft.EntityFrameworkCore;
using InventoryService.Data;
using InventoryService.Services;
using Serilog;

var builder = WebApplication.CreateBuilder(args);

// Configure Serilog
Log.Logger = new LoggerConfiguration()
    .ReadFrom.Configuration(builder.Configuration)
    .CreateLogger();

builder.Host.UseSerilog();

// Add services
builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

// Database
builder.Services.AddDbContext<InventoryDbContext>(options =>
    options.UseNpgsql(builder.Configuration.GetConnectionString("DefaultConnection")));

// Redis
builder.Services.AddStackExchangeRedisCache(options =>
{
    options.Configuration = builder.Configuration.GetConnectionString("Redis");
});

// CORS
builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(policy =>
    {
        policy.WithOrigins(builder.Configuration["AllowedOrigins"]!)
              .AllowAnyMethod()
              .AllowAnyHeader()
              .AllowCredentials();
    });
});

// Response compression
builder.Services.AddResponseCompression();

// Custom services
builder.Services.AddScoped<IInventoryService, InventoryService>();
builder.Services.AddSingleton<IKafkaProducer, KafkaProducer>();

var app = builder.Build();

// Middleware pipeline
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.UseResponseCompression();
app.UseCors();
app.UseAuthorization();
app.MapControllers();

// Health check endpoint
app.MapGet("/health", () => Results.Ok(new { status = "healthy" }));

app.Run();
```

**Alternatives Considered**:
- NancyFx: Discontinued
- ServiceStack: Commercial licensing
- Carter: Less mature

**Example Controller**:
```csharp
// Controllers/InventoryController.cs
using Microsoft.AspNetCore.Mvc;
using InventoryService.Services;
using InventoryService.DTOs;

namespace InventoryService.Controllers;

[ApiController]
[Route("api/v1/[controller]")]
public class InventoryController : ControllerBase
{
    private readonly IInventoryService _inventoryService;
    private readonly ILogger<InventoryController> _logger;

    public InventoryController(
        IInventoryService inventoryService,
        ILogger<InventoryController> logger)
    {
        _inventoryService = inventoryService;
        _logger = logger;
    }

    [HttpPost("reserve")]
    public async Task<ActionResult<ReservationResponse>> ReserveInventory(
        [FromBody] ReserveInventoryRequest request,
        [FromHeader(Name = "X-Tenant-Id")] Guid tenantId)
    {
        try
        {
            var result = await _inventoryService.ReserveInventoryAsync(
                tenantId,
                request.Sku,
                request.Quantity);

            return Ok(result);
        }
        catch (InsufficientInventoryException ex)
        {
            _logger.LogWarning(ex, "Insufficient inventory for SKU {Sku}", request.Sku);
            return BadRequest(new { error = ex.Message });
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error reserving inventory");
            return StatusCode(500, new { error = "Internal server error" });
        }
    }

    [HttpGet("{sku}")]
    public async Task<ActionResult<InventoryItemDto>> GetInventory(
        string sku,
        [FromHeader(Name = "X-Tenant-Id")] Guid tenantId)
    {
        var item = await _inventoryService.GetInventoryAsync(tenantId, sku);

        if (item == null)
            return NotFound();

        return Ok(item);
    }
}
```

**Example Service with Kafka Integration**:
```csharp
// Services/InventoryService.cs
using Microsoft.EntityFrameworkCore;
using InventoryService.Data;
using InventoryService.Events;

namespace InventoryService.Services;

public interface IInventoryService
{
    Task<ReservationResponse> ReserveInventoryAsync(Guid tenantId, string sku, int quantity);
    Task<InventoryItemDto?> GetInventoryAsync(Guid tenantId, string sku);
}

public class InventoryService : IInventoryService
{
    private readonly InventoryDbContext _context;
    private readonly IKafkaProducer _kafkaProducer;
    private readonly ILogger<InventoryService> _logger;

    public InventoryService(
        InventoryDbContext context,
        IKafkaProducer kafkaProducer,
        ILogger<InventoryService> logger)
    {
        _context = context;
        _kafkaProducer = kafkaProducer;
        _logger = logger;
    }

    public async Task<ReservationResponse> ReserveInventoryAsync(
        Guid tenantId,
        string sku,
        int quantity)
    {
        await using var transaction = await _context.Database.BeginTransactionAsync();

        try
        {
            var item = await _context.InventoryItems
                .Where(i => i.TenantId == tenantId && i.Sku == sku)
                .FirstOrDefaultAsync();

            if (item == null)
                throw new NotFoundException($"Item with SKU {sku} not found");

            if (item.Quantity < quantity)
                throw new InsufficientInventoryException(
                    $"Insufficient inventory. Available: {item.Quantity}, Requested: {quantity}");

            // Reserve inventory
            item.Quantity -= quantity;
            item.UpdatedAt = DateTime.UtcNow;

            await _context.SaveChangesAsync();

            // Publish event
            var inventoryEvent = new InventoryReservedEvent
            {
                EventId = Guid.NewGuid(),
                TenantId = tenantId,
                Sku = sku,
                Quantity = quantity,
                RemainingQuantity = item.Quantity,
                Timestamp = DateTime.UtcNow
            };

            await _kafkaProducer.ProduceAsync("inventory-events", inventoryEvent);

            await transaction.CommitAsync();

            _logger.LogInformation(
                "Reserved {Quantity} units of {Sku} for tenant {TenantId}",
                quantity, sku, tenantId);

            return new ReservationResponse
            {
                Success = true,
                ReservedQuantity = quantity,
                RemainingQuantity = item.Quantity
            };
        }
        catch
        {
            await transaction.RollbackAsync();
            throw;
        }
    }

    public async Task<InventoryItemDto?> GetInventoryAsync(Guid tenantId, string sku)
    {
        var item = await _context.InventoryItems
            .Where(i => i.TenantId == tenantId && i.Sku == sku)
            .Select(i => new InventoryItemDto
            {
                Sku = i.Sku,
                Quantity = i.Quantity,
                UpdatedAt = i.UpdatedAt
            })
            .FirstOrDefaultAsync();

        return item;
    }
}
```

---

## Databases

### PostgreSQL

**Version**: PostgreSQL 15.x (AWS RDS, Google Cloud SQL, or self-hosted)

**Use For**:
- User data
- Orders
- Payments
- Inventory
- Shipping
- Tenant metadata

**Configuration**:
```yaml
# PostgreSQL Configuration
max_connections: 200
shared_buffers: 4GB
effective_cache_size: 12GB
maintenance_work_mem: 1GB
checkpoint_completion_target: 0.9
wal_buffers: 16MB
default_statistics_target: 100
random_page_cost: 1.1
effective_io_concurrency: 200
work_mem: 20MB
min_wal_size: 2GB
max_wal_size: 8GB

# Replication
wal_level: replica
max_wal_senders: 10
max_replication_slots: 10
```

**Extensions**:
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- Text search
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- Encryption
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements"; -- Query stats
```

**Connection Pooling**: PgBouncer
```ini
[databases]
ecommerce = host=localhost port=5432 dbname=ecommerce

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
reserve_pool_size = 5
reserve_pool_timeout = 3
```

**ORM/Query Builder**:

**Go**: GORM 1.25.x
```go
import (
    "gorm.io/gorm"
    "gorm.io/driver/postgres"
    "time"
)

// Model
type User struct {
    ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    TenantID     string    `gorm:"type:uuid;not null;index"`
    Email        string    `gorm:"type:varchar(255);not null"`
    PasswordHash string    `gorm:"type:varchar(255);not null"`
    CreatedAt    time.Time `gorm:"autoCreateTime"`
    UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// Ensure unique constraint
func (User) TableName() string {
    return "users"
}

// Database connection
dsn := "host=localhost user=postgres password=postgres dbname=ecommerce port=5432 sslmode=disable"
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

// Create index
db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email)")
```

**.NET**: Entity Framework Core 8.x
```csharp
using Microsoft.EntityFrameworkCore;

// DbContext
public class InventoryDbContext : DbContext
{
    public InventoryDbContext(DbContextOptions<InventoryDbContext> options)
        : base(options) { }

    public DbSet<InventoryItem> InventoryItems { get; set; }

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<InventoryItem>(entity =>
        {
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => new { e.TenantId, e.Sku }).IsUnique();
            entity.Property(e => e.TenantId).IsRequired();
        });
    }
}

// Model
public class InventoryItem
{
    public Guid Id { get; set; }
    public Guid TenantId { get; set; }
    public string Sku { get; set; } = string.Empty;
    public int Quantity { get; set; }
    public DateTime CreatedAt { get; set; }
    public DateTime UpdatedAt { get; set; }
}
```

**Justification**:
- ✅ ACID compliance
- ✅ Strong data consistency
- ✅ Excellent JSON support (JSONB)
- ✅ Full-text search
- ✅ Rich ecosystem
- ✅ Great for transactional data
- ✅ Battle-tested at scale

**Alternatives Considered**:
- MySQL: Less advanced features
- CockroachDB: More complex, overkill for now

---

### MongoDB

**Version**: MongoDB 7.x (Atlas, DocumentDB, or self-hosted)

**Use For**:
- Product catalog
- Product reviews
- Notifications
- Logs (short-term)

**Configuration**:
```javascript
// MongoDB Configuration
{
  "storage": {
    "wiredTiger": {
      "engineConfig": {
        "cacheSizeGB": 4
      }
    }
  },
  "replication": {
    "replSetName": "ecommerce-replica"
  }
}
```

**ODM/Driver**: MongoDB Go Driver 1.13.x
```go
import (
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/bson"
    "time"
)

// Product model
type Product struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    TenantID  string            `bson:"tenant_id"`
    Name      string            `bson:"name"`
    Slug      string            `bson:"slug"`
    Price     float64           `bson:"price"`
    Images    []ProductImage    `bson:"images"`
    Metadata  map[string]interface{} `bson:"metadata"`
    CreatedAt time.Time         `bson:"created_at"`
    UpdatedAt time.Time         `bson:"updated_at"`
}

type ProductImage struct {
    URL   string `bson:"url"`
    Alt   string `bson:"alt"`
    Order int    `bson:"order"`
}

// Connect to MongoDB
clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
client, err := mongo.Connect(context.Background(), clientOptions)

collection := client.Database("ecommerce").Collection("products")

// Create indexes
indexModels := []mongo.IndexModel{
    {
        Keys: bson.D{
            {Key: "tenant_id", Value: 1},
            {Key: "slug", Value: 1},
        },
        Options: options.Index().SetUnique(true),
    },
    {
        Keys: bson.D{{Key: "tenant_id", Value: 1}},
    },
}

collection.Indexes().CreateMany(context.Background(), indexModels)
```

**Justification**:
- ✅ Flexible schema
- ✅ Excellent for product catalogs
- ✅ Fast document queries
- ✅ Rich query language
- ✅ Easy horizontal scaling
- ✅ Good for variable product attributes

**Alternatives Considered**:
- Cassandra: Too complex
- DynamoDB: Vendor lock-in

---

### Redis

**Version**: Redis 7.x (AWS ElastiCache, Google Memorystore, or self-hosted)

**Use For**:
- Caching
- Session storage
- Cart data
- Rate limiting
- Real-time features

**Configuration**:
```conf
# redis.conf
maxmemory 4gb
maxmemory-policy allkeys-lru

# Persistence
save 900 1
save 300 10
save 60 10000

# AOF
appendonly yes
appendfsync everysec

# Cluster mode
cluster-enabled yes
cluster-config-file nodes.conf
cluster-node-timeout 5000
```

**Client (Go)**: go-redis 8.x
```go
import (
    "github.com/go-redis/redis/v8"
    "context"
    "time"
)

// Single instance
rdb := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: os.Getenv("REDIS_PASSWORD"),
    DB:       0,
})

// Cluster
rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{
        "redis-node-1:6379",
        "redis-node-2:6379",
        "redis-node-3:6379",
    },
    Password: os.Getenv("REDIS_PASSWORD"),
})

ctx := context.Background()

// Usage
err := rdb.Set(ctx, "key", "value", 3600*time.Second).Err()
val, err := rdb.Get(ctx, "key").Result()
```

**Justification**:
- ✅ Extremely fast (sub-millisecond)
- ✅ Rich data structures
- ✅ Pub/Sub support
- ✅ Lua scripting
- ✅ Persistence options
- ✅ Industry standard

**Alternatives Considered**:
- Memcached: Less features
- Hazelcast: More complex

---

### Elasticsearch

**Version**: Elasticsearch 8.x (Elastic Cloud or self-hosted)

**Use For**:
- Product search
- Full-text search
- Logs (with ELK stack)
- Analytics

**Configuration**:
```yaml
# elasticsearch.yml
cluster.name: ecommerce-search
node.name: node-1

# Memory
bootstrap.memory_lock: true

# Network
network.host: 0.0.0.0
http.port: 9200

# Discovery
discovery.seed_hosts:
  - node-1
  - node-2
  - node-3
cluster.initial_master_nodes:
  - node-1
  - node-2
  - node-3

# Index settings
index.number_of_shards: 3
index.number_of_replicas: 2
```

**Client**: @elastic/elasticsearch 8.x
```typescript
import { Client } from '@elastic/elasticsearch';

const client = new Client({
  node: process.env.ELASTICSEARCH_URL,
  auth: {
    apiKey: process.env.ELASTICSEARCH_API_KEY
  }
});

// Create index
await client.indices.create({
  index: 'products',
  body: {
    mappings: {
      properties: {
        tenantId: { type: 'keyword' },
        name: { type: 'text', analyzer: 'standard' },
        description: { type: 'text' },
        price: { type: 'float' },
        category: { type: 'keyword' },
        tags: { type: 'keyword' },
        createdAt: { type: 'date' }
      }
    }
  }
});

// Search
const result = await client.search({
  index: 'products',
  body: {
    query: {
      bool: {
        must: [
          { match: { name: 'headphones' } },
          { term: { tenantId: 'tenant_123' } }
        ]
      }
    }
  }
});
```

**Justification**:
- ✅ Powerful full-text search
- ✅ Fast faceted search
- ✅ Real-time indexing
- ✅ Scalable
- ✅ Rich query DSL
- ✅ Great for analytics

**Alternatives Considered**:
- Algolia: SaaS, expensive
- MeiliSearch: Less mature
- Typesense: Good but smaller ecosystem

---

## Frontend Technologies

### Web Application

#### Next.js

**Version**: Next.js 14.x (App Router)

**Use For**:
- Admin Dashboard
- Customer Storefront

**Configuration**:
```typescript
// next.config.js
/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  images: {
    domains: ['cdn.example.com'],
    formats: ['image/avif', 'image/webp'],
  },
  experimental: {
    serverActions: true,
  },
  env: {
    API_URL: process.env.API_URL,
  },
}

module.exports = nextConfig
```

**Core Dependencies**:
```json
{
  "dependencies": {
    "next": "^14.0.4",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "typescript": "^5.3.3",
    "@tanstack/react-query": "^5.12.2",
    "zustand": "^4.4.7",
    "axios": "^1.6.2",
    "zod": "^3.22.4",
    "react-hook-form": "^7.48.2"
  }
}
```

**Justification**:
- ✅ Server-side rendering
- ✅ Great SEO
- ✅ File-based routing
- ✅ API routes
- ✅ Image optimization
- ✅ Strong ecosystem
- ✅ Excellent performance

**Alternatives Considered**:
- Remix: Less mature
- Nuxt.js: Vue-based, smaller ecosystem
- SvelteKit: Less mature

---

#### React

**Version**: React 18.x

**State Management**: Zustand 4.x
```typescript
// store/cart.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface CartState {
  items: CartItem[];
  addItem: (item: CartItem) => void;
  removeItem: (id: string) => void;
  clearCart: () => void;
}

export const useCartStore = create<CartState>()(
  persist(
    (set) => ({
      items: [],
      addItem: (item) => set((state) => ({
        items: [...state.items, item]
      })),
      removeItem: (id) => set((state) => ({
        items: state.items.filter(i => i.id !== id)
      })),
      clearCart: () => set({ items: [] }),
    }),
    {
      name: 'cart-storage',
    }
  )
);
```

**Data Fetching**: React Query 5.x
```typescript
// hooks/useProducts.ts
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

export const useProducts = (filters?: ProductFilters) => {
  return useQuery({
    queryKey: ['products', filters],
    queryFn: () => api.products.list(filters),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
};
```

**Justification**:
- ✅ Most popular framework
- ✅ Large ecosystem
- ✅ Component reusability
- ✅ Strong typing with TypeScript
- ✅ Easy to hire developers

---

#### Styling: TailwindCSS

**Version**: TailwindCSS 3.x

**Configuration**:
```javascript
// tailwind.config.js
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          500: '#3b82f6',
          900: '#1e3a8a',
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
  ],
}
```

**Justification**:
- ✅ Utility-first approach
- ✅ Fast development
- ✅ Small bundle size (with purging)
- ✅ Highly customizable
- ✅ Great documentation

**Alternatives Considered**:
- CSS Modules: More boilerplate
- Styled Components: Runtime overhead
- Chakra UI: Less flexible

---

#### Component Library: Shadcn/ui

**Version**: Latest

**Why**: Unstyled, accessible components built with Radix UI

```bash
npx shadcn-ui@latest init
npx shadcn-ui@latest add button
npx shadcn-ui@latest add dialog
npx shadcn-ui@latest add dropdown-menu
```

**Usage**:
```typescript
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent } from '@/components/ui/dialog';

export function ProductModal() {
  return (
    <Dialog>
      <DialogContent>
        <h2>Product Details</h2>
        <Button>Add to Cart</Button>
      </DialogContent>
    </Dialog>
  );
}
```

**Justification**:
- ✅ Copy-paste components
- ✅ Full customization
- ✅ Accessible (WCAG 2.1 AA)
- ✅ No runtime overhead
- ✅ Built on Radix UI

**Alternatives Considered**:
- Material UI: Heavy, opinionated
- Ant Design: Less flexible
- Chakra UI: Runtime overhead

---

### Mobile Applications

#### React Native

**Version**: React Native 0.73.x

**Use For**: iOS and Android apps

**Key Libraries**:
```json
{
  "dependencies": {
    "react-native": "0.73.0",
    "react-navigation": "^6.0.0",
    "react-native-gesture-handler": "^2.14.0",
    "react-native-reanimated": "^3.6.0",
    "@tanstack/react-query": "^5.12.2",
    "zustand": "^4.4.7"
  }
}
```

**Justification**:
- ✅ Code sharing with web
- ✅ Large community
- ✅ Native performance
- ✅ Hot reloading
- ✅ Strong ecosystem

**Alternatives Considered**:
- Flutter: Different language (Dart)
- Native (Swift/Kotlin): Separate codebases

---

## Message Queue & Events

### Apache Kafka

**Version**: Kafka 3.6.x (AWS MSK, Confluent Cloud, or self-hosted)

**Use For**:
- Event streaming
- Microservice communication
- Event sourcing

**Configuration**:
```properties
# server.properties
broker.id=1
num.network.threads=8
num.io.threads=16

# Log settings
log.retention.hours=168
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000

# Replication
default.replication.factor=3
min.insync.replicas=2

# Performance
compression.type=snappy
```

**Topic Configuration**:
```bash
# Create topics
kafka-topics --create \
  --topic order-events \
  --partitions 10 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config compression.type=snappy

kafka-topics --create \
  --topic payment-events \
  --partitions 10 \
  --replication-factor 3
```

**Client (Go)**: segmentio/kafka-go 0.4.x
```go
import "github.com/segmentio/kafka-go"

// Producer
writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers:  []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
    Topic:    "order-events",
    Balancer: &kafka.LeastBytes{},
})

err := writer.WriteMessages(context.Background(),
    kafka.Message{
        Key:   []byte(orderID),
        Value: eventData,
        Headers: []kafka.Header{
            {Key: "tenant-id", Value: []byte(tenantID)},
        },
    },
)

// Consumer
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers: []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
    Topic:   "order-events",
    GroupID: "order-processor",
})

for {
    m, err := reader.ReadMessage(context.Background())
    if err != nil {
        break
    }
    handleEvent(m.Value)
}
```

**Client (.NET)**: Confluent.Kafka 2.x
```csharp
using Confluent.Kafka;

// Producer
var config = new ProducerConfig
{
    BootstrapServers = "kafka-1:9092,kafka-2:9092",
    SecurityProtocol = SecurityProtocol.SaslSsl,
    SaslMechanism = SaslMechanism.ScramSha256,
    SaslUsername = Environment.GetEnvironmentVariable("KAFKA_USERNAME"),
    SaslPassword = Environment.GetEnvironmentVariable("KAFKA_PASSWORD")
};

using var producer = new ProducerBuilder<string, string>(config).Build();

var message = new Message<string, string>
{
    Key = productId,
    Value = JsonSerializer.Serialize(inventoryEvent),
    Headers = new Headers
    {
        { "tenant-id", Encoding.UTF8.GetBytes(tenantId) }
    }
};

await producer.ProduceAsync("inventory-events", message);

// Consumer
var consumerConfig = new ConsumerConfig
{
    BootstrapServers = "kafka-1:9092,kafka-2:9092",
    GroupId = "inventory-processor",
    AutoOffsetReset = AutoOffsetReset.Earliest,
    EnableAutoCommit = false,
    SecurityProtocol = SecurityProtocol.SaslSsl,
    SaslMechanism = SaslMechanism.ScramSha256,
    SaslUsername = Environment.GetEnvironmentVariable("KAFKA_USERNAME"),
    SaslPassword = Environment.GetEnvironmentVariable("KAFKA_PASSWORD")
};

using var consumer = new ConsumerBuilder<string, string>(consumerConfig).Build();
consumer.Subscribe("inventory-events");

while (true)
{
    var consumeResult = consumer.Consume(CancellationToken.None);
    await HandleInventoryEvent(consumeResult.Message.Value);
    consumer.Commit(consumeResult);
}
```

**Schema Registry**: Confluent Schema Registry
```bash
# Register schema
curl -X POST http://schema-registry:8081/subjects/order-events-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"schema": "{\"type\":\"record\",\"name\":\"Order\",...}"}'
```

**Justification**:
- ✅ High throughput (millions of messages/sec)
- ✅ Event replay capability
- ✅ Exactly-once semantics
- ✅ Partitioning for scalability
- ✅ Message ordering guarantees
- ✅ Battle-tested at scale

**Alternatives Considered**:
- RabbitMQ: Lower throughput
- AWS SQS: Limited features
- Google Pub/Sub: Vendor lock-in

---

### RabbitMQ (Optional)

**Version**: RabbitMQ 3.12.x

**Use For**:
- Task queues
- Delayed jobs
- Priority queues

**Client (Node.js)**: amqplib 0.10.x
```typescript
import amqp from 'amqplib';

const connection = await amqp.connect(process.env.RABBITMQ_URL);
const channel = await connection.createChannel();

await channel.assertQueue('email-queue', { durable: true });

// Publish
channel.sendToQueue('email-queue', Buffer.from(JSON.stringify({
  to: 'user@example.com',
  subject: 'Order Confirmation',
  template: 'order-confirmation',
  data: orderData,
})));

// Consume
channel.consume('email-queue', async (msg) => {
  if (msg) {
    const job = JSON.parse(msg.content.toString());
    await sendEmail(job);
    channel.ack(msg);
  }
});
```

---

## Caching

### Redis (Already covered in Databases)

**Additional Use Cases**:

**Cache-Aside Pattern**:
```typescript
async function getProduct(productId: string): Promise<Product> {
  // Check cache
  const cached = await redis.get(`product:${productId}`);
  if (cached) {
    return JSON.parse(cached);
  }

  // Fetch from database
  const product = await db.products.findById(productId);

  // Set cache with TTL
  await redis.set(
    `product:${productId}`,
    JSON.stringify(product),
    'EX',
    3600 // 1 hour
  );

  return product;
}
```

**Rate Limiting**:
```typescript
async function checkRateLimit(tenantId: string): Promise<boolean> {
  const key = `rate:${tenantId}:${Date.now() / 60000 | 0}`;
  const current = await redis.incr(key);

  if (current === 1) {
    await redis.expire(key, 60);
  }

  const limit = await getTenantRateLimit(tenantId);
  return current <= limit;
}
```

---

## Infrastructure & DevOps

### Container Orchestration

#### Kubernetes

**Version**: Kubernetes 1.28+

**Distribution**:
- AWS: Amazon EKS
- Google Cloud: GKE
- Azure: AKS
- Self-hosted: kubeadm

**Configuration**:
```yaml
# Example deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-service
  namespace: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: user-service
  template:
    metadata:
      labels:
        app: user-service
        version: v1
    spec:
      containers:
      - name: user-service
        image: ecr.example.com/user-service:1.2.3
        ports:
        - containerPort: 3000
        env:
        - name: NODE_ENV
          value: production
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: user-service-secrets
              key: db-host
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 3000
          initialDelaySeconds: 10
          periodSeconds: 5
```

**Helm**: Kubernetes package manager
```yaml
# Chart.yaml
apiVersion: v2
name: user-service
version: 1.0.0
appVersion: "1.2.3"
```

**Justification**:
- ✅ Industry standard
- ✅ Declarative configuration
- ✅ Self-healing
- ✅ Auto-scaling
- ✅ Service discovery
- ✅ Large ecosystem

---

#### Docker

**Version**: Docker 24.x

**Dockerfile Example**:
```dockerfile
# Go service
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /user-service

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /user-service .

EXPOSE 8080

CMD ["./user-service"]
```

```dockerfile
# .NET service
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build

WORKDIR /src

COPY ["InventoryService/InventoryService.csproj", "InventoryService/"]
RUN dotnet restore "InventoryService/InventoryService.csproj"

COPY . .
WORKDIR "/src/InventoryService"
RUN dotnet build "InventoryService.csproj" -c Release -o /app/build
RUN dotnet publish "InventoryService.csproj" -c Release -o /app/publish /p:UseAppHost=false

FROM mcr.microsoft.com/dotnet/aspnet:8.0

WORKDIR /app

COPY --from=build /app/publish .

EXPOSE 8080

ENV ASPNETCORE_URLS=http://+:8080

ENTRYPOINT ["dotnet", "InventoryService.dll"]
```

---

### Infrastructure as Code

#### Terraform

**Version**: Terraform 1.6+

**Provider**: AWS Provider 5.x

**Example**:
```hcl
# terraform/main.tf
terraform {
  required_version = ">= 1.6"

  backend "s3" {
    bucket = "ecommerce-terraform-state"
    key    = "production/terraform.tfstate"
    region = "us-east-1"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

module "eks" {
  source = "./modules/eks"

  cluster_name    = "ecommerce-${var.environment}"
  cluster_version = "1.28"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.private_subnets
}
```

**Justification**:
- ✅ Declarative IaC
- ✅ State management
- ✅ Multi-cloud support
- ✅ Large provider ecosystem
- ✅ Plan before apply

**Alternatives Considered**:
- Pulumi: Less mature
- CloudFormation: AWS only
- CDK: More complex

---

### CI/CD

#### GitHub Actions

**Version**: Latest

**Workflow Example**:
```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
      - run: npm ci
      - run: npm run test
      - run: npm run lint

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ${{ secrets.ECR_REGISTRY }}
          username: ${{ secrets.AWS_ACCESS_KEY_ID }}
          password: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
      - uses: docker/build-push-action@v5
        with:
          push: true
          tags: ${{ secrets.ECR_REGISTRY }}/user-service:${{ github.sha }}

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: azure/k8s-set-context@v3
        with:
          kubeconfig: ${{ secrets.KUBE_CONFIG }}
      - run: |
          kubectl set image deployment/user-service \
            user-service=${{ secrets.ECR_REGISTRY }}/user-service:${{ github.sha }} \
            -n production
```

**Justification**:
- ✅ Native GitHub integration
- ✅ Free for public repos
- ✅ Large action marketplace
- ✅ Easy to configure

**Alternatives Considered**:
- GitLab CI: Requires GitLab
- Jenkins: Self-hosted complexity
- CircleCI: Cost

---

#### ArgoCD

**Version**: ArgoCD 2.9+

**Application Configuration**:
```yaml
# argocd/user-service.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: user-service
  namespace: argocd
spec:
  project: ecommerce
  source:
    repoURL: https://github.com/yourorg/ecommerce-platform
    targetRevision: HEAD
    path: k8s/services/user-service
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**Justification**:
- ✅ GitOps workflow
- ✅ Declarative
- ✅ Automatic sync
- ✅ Rollback capability
- ✅ Multi-cluster support

---

## Monitoring & Observability

### Metrics: Prometheus + Grafana

**Prometheus Version**: 2.48+
**Grafana Version**: 10.2+

**Prometheus Configuration**:
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
```

**Node.js Metrics**: prom-client 15.x
```typescript
import client from 'prom-client';

// Create metrics
const httpRequestDuration = new client.Histogram({
  name: 'http_request_duration_seconds',
  help: 'Duration of HTTP requests in seconds',
  labelNames: ['method', 'route', 'status_code'],
  buckets: [0.1, 0.3, 0.5, 0.7, 1, 3, 5, 7, 10],
});

const httpRequestTotal = new client.Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'route', 'status_code'],
});

// Middleware
app.use((req, res, next) => {
  const start = Date.now();

  res.on('finish', () => {
    const duration = (Date.now() - start) / 1000;
    httpRequestDuration.observe(
      { method: req.method, route: req.route?.path, status_code: res.statusCode },
      duration
    );
    httpRequestTotal.inc({ method: req.method, route: req.route?.path, status_code: res.statusCode });
  });

  next();
});

// Metrics endpoint
app.get('/metrics', async (req, res) => {
  res.set('Content-Type', client.register.contentType);
  res.end(await client.register.metrics());
});
```

---

### Logging: ELK Stack

**Elasticsearch**: 8.x
**Logstash**: 8.x
**Kibana**: 8.x
**Filebeat**: 8.x

**Winston Logger (Node.js)**:
```typescript
import winston from 'winston';
import { ElasticsearchTransport } from 'winston-elasticsearch';

const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.json()
  ),
  defaultMeta: { service: 'user-service' },
  transports: [
    new winston.transports.Console(),
    new ElasticsearchTransport({
      level: 'info',
      clientOpts: { node: process.env.ELASTICSEARCH_URL },
      index: 'logs-user-service',
    }),
  ],
});

// Usage
logger.info('User registered', {
  userId: user.id,
  tenantId: user.tenantId,
  email: user.email,
});
```

---

### Distributed Tracing: Jaeger

**Version**: Jaeger 1.52+

**OpenTelemetry (Node.js)**:
```typescript
import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { JaegerExporter } from '@opentelemetry/exporter-jaeger';

const sdk = new NodeSDK({
  traceExporter: new JaegerExporter({
    endpoint: process.env.JAEGER_ENDPOINT,
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();
```

---

### Error Tracking: Sentry

**Version**: Sentry 7.x

**Configuration**:
```typescript
import * as Sentry from '@sentry/node';

Sentry.init({
  dsn: process.env.SENTRY_DSN,
  environment: process.env.NODE_ENV,
  tracesSampleRate: 0.1,
});

// Error handler
app.use(Sentry.Handlers.errorHandler());
```

---

## Security

### Secrets Management

**Choice**: HashiCorp Vault 1.15+ or AWS Secrets Manager

**Vault Configuration**:
```hcl
# vault.hcl
storage "consul" {
  address = "127.0.0.1:8500"
  path    = "vault/"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 0
}
```

---

### SSL/TLS: Let's Encrypt + cert-manager

**cert-manager Version**: 1.13+

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
```

---

## Third-Party Integrations

### Payment Processors

**Stripe**: stripe 14.x (Node.js)
```typescript
import Stripe from 'stripe';

const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, {
  apiVersion: '2023-10-16',
});

const paymentIntent = await stripe.paymentIntents.create({
  amount: 2000,
  currency: 'usd',
  payment_method_types: ['card'],
});
```

**PayPal**: @paypal/checkout-server-sdk 1.x
```typescript
import paypal from '@paypal/checkout-server-sdk';

const environment = new paypal.core.LiveEnvironment(
  process.env.PAYPAL_CLIENT_ID,
  process.env.PAYPAL_CLIENT_SECRET
);
const client = new paypal.core.PayPalHttpClient(environment);
```

---

### Email Service

**SendGrid**: @sendgrid/mail 8.x
```typescript
import sgMail from '@sendgrid/mail';

sgMail.setApiKey(process.env.SENDGRID_API_KEY);

await sgMail.send({
  to: 'customer@example.com',
  from: 'noreply@example.com',
  subject: 'Order Confirmation',
  html: emailTemplate,
});
```

---

### SMS Service

**Twilio**: twilio 4.x
```typescript
import twilio from 'twilio';

const client = twilio(
  process.env.TWILIO_ACCOUNT_SID,
  process.env.TWILIO_AUTH_TOKEN
);

await client.messages.create({
  body: 'Your order has been shipped!',
  from: process.env.TWILIO_PHONE_NUMBER,
  to: customerPhone,
});
```

---

## Development Tools

### Code Quality

- **ESLint**: 8.x
- **Prettier**: 3.x
- **TypeScript**: 5.3+
- **Husky**: 8.x (Git hooks)
- **lint-staged**: 15.x

### Testing

- **Jest**: 29.x
- **Supertest**: 6.x
- **Playwright**: 1.40+
- **k6**: 0.48+ (Load testing)

### API Documentation

- **Swagger/OpenAPI**: 3.1
- **@nestjs/swagger**: 7.x (if using NestJS)

---

## Summary Table

| Component | Technology | Version | Justification |
|-----------|-----------|---------|---------------|
| Backend (API) | Go (Golang) | 1.21+ | High performance, concurrency, simple |
| Backend (Performance) | .NET (C#) | 8.0 LTS | High performance, mature ecosystem |
| Web Framework (Go) | Gin | 1.9+ | Fast, minimal, production-ready |
| Web Framework (.NET) | ASP.NET Core | 8.0 | Fast, robust, built-in DI |
| Primary Database | PostgreSQL | 15.x | ACID, reliability |
| Document Database | MongoDB | 7.x | Flexible schema |
| Cache/Session | Redis | 7.x | Speed, versatility |
| Search | Elasticsearch | 8.x | Full-text search |
| Message Queue | Apache Kafka | 3.6+ | High throughput |
| Frontend Framework | Next.js 14 | 14.x | SSR, SEO, performance |
| UI Library | React | 18.x | Industry standard |
| State Management | Zustand | 4.x | Simple, fast |
| Data Fetching | React Query | 5.x | Caching, optimistic updates |
| Styling | TailwindCSS | 3.x | Utility-first, fast |
| Components | Shadcn/ui | Latest | Accessible, customizable |
| Mobile | React Native | 0.73+ | Code sharing |
| Container Runtime | Docker | 24.x | Standard |
| Orchestration | Kubernetes | 1.28+ | Industry standard |
| IaC | Terraform | 1.6+ | Declarative, multi-cloud |
| CI/CD | GitHub Actions | Latest | Native integration |
| GitOps | ArgoCD | 2.9+ | Declarative deployment |
| Metrics | Prometheus + Grafana | 2.48/10.2 | Standard monitoring |
| Logging | ELK Stack | 8.x | Centralized logging |
| Tracing | Jaeger | 1.52+ | Distributed tracing |
| Error Tracking | Sentry | 7.x | Real-time errors |
| Secrets | Vault or AWS SM | 1.15+ | Secure storage |

---

**This technology stack is:**
- ✅ Production-tested
- ✅ Scalable to 100M+ users
- ✅ Industry-standard
- ✅ Well-documented
- ✅ Easy to hire for
- ✅ Cost-effective

**Ready to implement!** 🚀
