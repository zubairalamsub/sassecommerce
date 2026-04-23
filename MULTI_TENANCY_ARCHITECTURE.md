# Multi-Tenancy Architecture

## Overview

This document outlines the multi-tenant architecture for the e-commerce platform, enabling multiple independent businesses (tenants) to use the same infrastructure while maintaining complete data isolation and customization capabilities.

---

## Table of Contents

1. [Multi-Tenancy Models](#multi-tenancy-models)
2. [Chosen Architecture](#chosen-architecture)
3. [Tenant Isolation Strategies](#tenant-isolation-strategies)
4. [Database Design](#database-design)
5. [Authentication & Authorization](#authentication--authorization)
6. [Tenant Onboarding](#tenant-onboarding)
7. [Data Isolation & Security](#data-isolation--security)
8. [Customization & Branding](#customization--branding)
9. [Billing & Metering](#billing--metering)
10. [Scaling Strategy](#scaling-strategy)
11. [Implementation Examples](#implementation-examples)

---

## Multi-Tenancy Models

### 1. Shared Database, Shared Schema (Pool Model)

**Architecture**: All tenants share the same database and tables with a `tenant_id` column

**Pros**:
- Most cost-effective
- Easiest to maintain
- Efficient resource utilization

**Cons**:
- Risk of data leakage
- Difficult to customize per tenant
- Noisy neighbor problems
- Complex queries with tenant filtering

**Best For**: Small businesses, low-security requirements

---

### 2. Shared Database, Separate Schema (Bridge Model)

**Architecture**: Each tenant has their own schema within a shared database

**Pros**:
- Better data isolation
- Easier backup/restore per tenant
- Per-tenant customization possible
- Moderate resource efficiency

**Cons**:
- More complex management
- Schema limits in databases
- Migration complexity

**Best For**: Medium-sized businesses, moderate customization needs

---

### 3. Separate Database Per Tenant (Silo Model)

**Architecture**: Each tenant has a completely separate database

**Pros**:
- Complete data isolation
- Full customization per tenant
- Easy to scale individual tenants
- Compliance-friendly
- No noisy neighbor issues

**Cons**:
- Higher infrastructure costs
- More complex management
- Difficult cross-tenant analytics

**Best For**: Enterprise customers, high-security requirements, compliance-heavy industries

---

## Chosen Architecture

### Hybrid Multi-Tenancy Model

We'll use a **hybrid approach** combining different models based on tenant tier:

```
┌─────────────────────────────────────────────────────────────┐
│                    TENANT TIERS                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Tier 1: Free/Starter (Pool Model)                          │
│  → Shared database, shared schema with tenant_id            │
│  → Up to 1000 tenants per database                          │
│  → Limited customization                                     │
│                                                              │
│  Tier 2: Professional (Bridge Model)                        │
│  → Shared database, separate schema per tenant              │
│  → Up to 100 tenants per database                           │
│  → Moderate customization                                    │
│                                                              │
│  Tier 3: Enterprise (Silo Model)                            │
│  → Dedicated database per tenant                            │
│  → Full customization                                        │
│  → Dedicated resources                                       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Tenant Isolation Strategies

### Request Flow with Tenant Context

```
1. Client Request → API Gateway
2. API Gateway → Extract Tenant ID (from subdomain/header/token)
3. Validate Tenant → Check tenant status (active/suspended)
4. Route to Service → Pass tenant context
5. Service → Apply tenant-specific logic
6. Database → Query with tenant filter
7. Response → Return tenant-scoped data
```

### Tenant Identification Methods

#### 1. Subdomain-based (Recommended)
```
tenant1.example.com → Tenant ID: tenant1
tenant2.example.com → Tenant ID: tenant2
```

#### 2. Path-based
```
example.com/tenant1/products → Tenant ID: tenant1
example.com/tenant2/products → Tenant ID: tenant2
```

#### 3. Header-based
```
X-Tenant-ID: tenant1
```

#### 4. Token-based (JWT)
```json
{
  "sub": "user_123",
  "tenant_id": "tenant1",
  "roles": ["admin"]
}
```

---

## Database Design

### Pool Model (Tier 1) - Shared Schema

#### Tenants Table

```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(50) DEFAULT 'free', -- free, starter, professional, enterprise
    status VARCHAR(50) DEFAULT 'active', -- active, suspended, trial, cancelled

    -- Subscription
    plan_id VARCHAR(50),
    billing_email VARCHAR(255),
    trial_ends_at TIMESTAMP,
    subscription_starts_at TIMESTAMP,
    subscription_ends_at TIMESTAMP,

    -- Limits
    max_products INTEGER DEFAULT 100,
    max_orders INTEGER DEFAULT 1000,
    max_users INTEGER DEFAULT 10,
    max_storage_mb INTEGER DEFAULT 1000,

    -- Customization
    custom_domain VARCHAR(255),
    logo_url TEXT,
    primary_color VARCHAR(7),
    settings JSONB DEFAULT '{}',

    -- Contact
    owner_name VARCHAR(255),
    owner_email VARCHAR(255) NOT NULL,
    owner_phone VARCHAR(20),

    -- Address
    company_name VARCHAR(255),
    address JSONB,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_tier ON tenants(tier);
```

#### Modified Users Table (with tenant_id)

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) DEFAULT 'customer', -- owner, admin, staff, customer
    status VARCHAR(50) DEFAULT 'active',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,

    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(tenant_id, email);
CREATE INDEX idx_users_role ON users(tenant_id, role);
```

#### Modified Products Table (with tenant_id)

```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(12, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'draft',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,

    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_products_tenant_id ON products(tenant_id);
CREATE INDEX idx_products_status ON products(tenant_id, status);
CREATE INDEX idx_products_slug ON products(tenant_id, slug);
```

#### Modified Orders Table (with tenant_id)

```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_number VARCHAR(100) NOT NULL,
    user_id UUID REFERENCES users(id),
    status VARCHAR(50) DEFAULT 'pending',
    total_amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(tenant_id, order_number)
);

CREATE INDEX idx_orders_tenant_id ON orders(tenant_id);
CREATE INDEX idx_orders_user_id ON orders(tenant_id, user_id);
CREATE INDEX idx_orders_status ON orders(tenant_id, status);
CREATE INDEX idx_orders_created_at ON orders(tenant_id, created_at DESC);
```

### Row-Level Security (PostgreSQL)

```sql
-- Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

-- Create policy for tenant isolation
CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE POLICY tenant_isolation_products ON products
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE POLICY tenant_isolation_orders ON orders
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Set tenant context in application
-- SET LOCAL app.current_tenant = 'tenant-uuid';
```

---

### Bridge Model (Tier 2) - Separate Schema

#### Schema Structure

```
database: ecommerce_shared_tier2
├── tenant_abc (schema)
│   ├── users
│   ├── products
│   ├── orders
│   └── ...
├── tenant_xyz (schema)
│   ├── users
│   ├── products
│   ├── orders
│   └── ...
└── public (schema)
    └── tenants (central tenant registry)
```

#### Dynamic Schema Selection

```sql
-- Set search path per request
SET search_path TO tenant_abc, public;

-- Then all queries run in tenant context
SELECT * FROM users; -- Queries tenant_abc.users
```

#### Schema Creation for New Tenant

```sql
-- Create schema
CREATE SCHEMA tenant_abc;

-- Create tables in schema
CREATE TABLE tenant_abc.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    -- No tenant_id needed in separate schema
    created_at TIMESTAMP DEFAULT NOW()
);

-- Apply same structure to all tables
```

---

### Silo Model (Tier 3) - Separate Database

#### Database Naming Convention

```
ecommerce_tenant_abc
ecommerce_tenant_xyz
ecommerce_tenant_enterprise1
```

#### Central Tenant Registry

```sql
-- Master database: ecommerce_registry
CREATE TABLE tenant_databases (
    tenant_id UUID PRIMARY KEY,
    tenant_slug VARCHAR(100) UNIQUE NOT NULL,
    database_name VARCHAR(100) UNIQUE NOT NULL,
    database_host VARCHAR(255) NOT NULL,
    database_port INTEGER DEFAULT 5432,
    connection_pool_size INTEGER DEFAULT 10,

    -- Connection credentials (encrypted)
    db_username_encrypted TEXT,
    db_password_encrypted TEXT,

    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### Dynamic Connection Pool

```typescript
// Connection pool manager
class TenantConnectionManager {
  private pools: Map<string, Pool> = new Map();

  async getConnection(tenantId: string): Promise<PoolClient> {
    if (!this.pools.has(tenantId)) {
      const tenantDb = await this.getTenantDatabase(tenantId);

      const pool = new Pool({
        host: tenantDb.host,
        port: tenantDb.port,
        database: tenantDb.database,
        user: decrypt(tenantDb.username),
        password: decrypt(tenantDb.password),
        max: tenantDb.poolSize,
      });

      this.pools.set(tenantId, pool);
    }

    return this.pools.get(tenantId)!.connect();
  }

  private async getTenantDatabase(tenantId: string) {
    // Query central registry
    const result = await registryDb.query(
      'SELECT * FROM tenant_databases WHERE tenant_id = $1',
      [tenantId]
    );
    return result.rows[0];
  }
}
```

---

## Authentication & Authorization

### Multi-Tenant JWT Structure

```json
{
  "sub": "user_uuid",
  "email": "john@example.com",
  "tenant_id": "tenant_abc",
  "tenant_slug": "acme-corp",
  "tenant_tier": "professional",
  "roles": ["admin"],
  "permissions": [
    "products:create",
    "products:read",
    "products:update",
    "orders:read"
  ],
  "exp": 1234567890,
  "iat": 1234567890
}
```

### Tenant-Aware Middleware

```typescript
// Express middleware
export const tenantContext = async (
  req: Request,
  res: Response,
  next: NextFunction
) => {
  try {
    // 1. Extract tenant identifier
    const tenantSlug = extractTenantSlug(req);

    if (!tenantSlug) {
      return res.status(400).json({ error: 'Tenant not specified' });
    }

    // 2. Load tenant from cache or database
    const tenant = await getTenant(tenantSlug);

    if (!tenant) {
      return res.status(404).json({ error: 'Tenant not found' });
    }

    // 3. Check tenant status
    if (tenant.status !== 'active') {
      return res.status(403).json({
        error: 'Tenant suspended',
        reason: tenant.suspension_reason
      });
    }

    // 4. Attach tenant to request
    req.tenant = tenant;
    req.tenantId = tenant.id;

    // 5. Set database context (for RLS)
    if (req.db) {
      await req.db.query(
        `SET LOCAL app.current_tenant = $1`,
        [tenant.id]
      );
    }

    next();
  } catch (error) {
    next(error);
  }
};

function extractTenantSlug(req: Request): string | null {
  // Method 1: Subdomain
  const hostname = req.hostname;
  const subdomain = hostname.split('.')[0];
  if (subdomain && subdomain !== 'www' && subdomain !== 'api') {
    return subdomain;
  }

  // Method 2: Header
  const headerTenant = req.headers['x-tenant-id'] as string;
  if (headerTenant) {
    return headerTenant;
  }

  // Method 3: JWT
  if (req.user?.tenant_slug) {
    return req.user.tenant_slug;
  }

  return null;
}
```

### Tenant-Scoped Queries

```typescript
// Repository base class
export class TenantRepository<T> {
  constructor(
    private tableName: string,
    private tenantId: string
  ) {}

  async findAll(): Promise<T[]> {
    const result = await db.query(
      `SELECT * FROM ${this.tableName} WHERE tenant_id = $1`,
      [this.tenantId]
    );
    return result.rows;
  }

  async findById(id: string): Promise<T | null> {
    const result = await db.query(
      `SELECT * FROM ${this.tableName}
       WHERE id = $1 AND tenant_id = $2`,
      [id, this.tenantId]
    );
    return result.rows[0] || null;
  }

  async create(data: Partial<T>): Promise<T> {
    // Automatically inject tenant_id
    const dataWithTenant = {
      ...data,
      tenant_id: this.tenantId,
    };

    const columns = Object.keys(dataWithTenant);
    const values = Object.values(dataWithTenant);
    const placeholders = columns.map((_, i) => `$${i + 1}`);

    const result = await db.query(
      `INSERT INTO ${this.tableName} (${columns.join(', ')})
       VALUES (${placeholders.join(', ')})
       RETURNING *`,
      values
    );

    return result.rows[0];
  }

  async update(id: string, data: Partial<T>): Promise<T> {
    const updates = Object.keys(data)
      .map((key, i) => `${key} = $${i + 2}`)
      .join(', ');

    const result = await db.query(
      `UPDATE ${this.tableName}
       SET ${updates}
       WHERE id = $1 AND tenant_id = $${Object.keys(data).length + 2}
       RETURNING *`,
      [id, ...Object.values(data), this.tenantId]
    );

    return result.rows[0];
  }

  async delete(id: string): Promise<boolean> {
    const result = await db.query(
      `DELETE FROM ${this.tableName}
       WHERE id = $1 AND tenant_id = $2`,
      [id, this.tenantId]
    );

    return result.rowCount > 0;
  }
}

// Usage
const productRepo = new TenantRepository<Product>(
  'products',
  req.tenantId
);

const products = await productRepo.findAll();
```

---

## Tenant Onboarding

### Onboarding Flow

```
1. Registration
   ├── Collect: Company name, email, subdomain preference
   ├── Validate: Subdomain availability
   └── Create: Tenant record

2. Provisioning
   ├── Create database/schema (based on tier)
   ├── Run migrations
   ├── Setup default data
   └── Configure resources

3. Account Setup
   ├── Create owner user
   ├── Send verification email
   └── Generate initial API keys

4. Customization
   ├── Upload logo
   ├── Set brand colors
   └── Configure settings

5. Activation
   ├── Complete onboarding checklist
   ├── Activate subscription
   └── Enable public access
```

### Provisioning Service

```typescript
export class TenantProvisioningService {
  async provisionTenant(data: TenantRegistration): Promise<Tenant> {
    const tenant = await this.createTenant(data);

    try {
      // 1. Provision database resources
      await this.provisionDatabase(tenant);

      // 2. Run migrations
      await this.runMigrations(tenant);

      // 3. Seed default data
      await this.seedDefaultData(tenant);

      // 4. Create owner user
      await this.createOwnerUser(tenant, data);

      // 5. Setup integrations
      await this.setupIntegrations(tenant);

      // 6. Activate tenant
      await this.activateTenant(tenant.id);

      // 7. Send welcome email
      await this.sendWelcomeEmail(tenant, data);

      return tenant;
    } catch (error) {
      // Rollback on failure
      await this.rollbackProvisioning(tenant.id);
      throw error;
    }
  }

  private async provisionDatabase(tenant: Tenant): Promise<void> {
    switch (tenant.tier) {
      case 'free':
      case 'starter':
        // Pool model - no additional provisioning needed
        break;

      case 'professional':
        // Bridge model - create schema
        await db.query(`CREATE SCHEMA IF NOT EXISTS ${tenant.slug}`);
        break;

      case 'enterprise':
        // Silo model - create dedicated database
        await this.createDedicatedDatabase(tenant);
        break;
    }
  }

  private async runMigrations(tenant: Tenant): Promise<void> {
    const migrationRunner = new MigrationRunner(tenant);
    await migrationRunner.runAll();
  }

  private async seedDefaultData(tenant: Tenant): Promise<void> {
    const seeder = new TenantSeeder(tenant.id);

    // Create default categories
    await seeder.seedCategories([
      'Electronics',
      'Clothing',
      'Home & Garden',
    ]);

    // Create default settings
    await seeder.seedSettings({
      currency: 'USD',
      timezone: 'UTC',
      language: 'en',
    });

    // Create default email templates
    await seeder.seedEmailTemplates();
  }

  private async createOwnerUser(
    tenant: Tenant,
    data: TenantRegistration
  ): Promise<void> {
    const userService = new UserService(tenant.id);

    await userService.create({
      email: data.ownerEmail,
      firstName: data.ownerFirstName,
      lastName: data.ownerLastName,
      password: data.password,
      role: 'owner',
      emailVerified: false,
    });
  }
}
```

---

## Data Isolation & Security

### Security Best Practices

#### 1. Query-Level Isolation

```typescript
// BAD: Missing tenant filter
const products = await db.query('SELECT * FROM products');

// GOOD: With tenant filter
const products = await db.query(
  'SELECT * FROM products WHERE tenant_id = $1',
  [tenantId]
);

// BETTER: Using RLS
await db.query('SET LOCAL app.current_tenant = $1', [tenantId]);
const products = await db.query('SELECT * FROM products');
```

#### 2. API-Level Isolation

```typescript
// Ensure tenant context in all requests
app.use('/api', tenantContext);

// Validate tenant ownership
app.get('/api/products/:id', async (req, res) => {
  const product = await productService.findById(req.params.id);

  // Double-check tenant ownership
  if (product.tenant_id !== req.tenantId) {
    return res.status(403).json({ error: 'Access denied' });
  }

  res.json(product);
});
```

#### 3. File Storage Isolation

```typescript
// S3 structure
// s3://ecommerce-assets/
//   ├── tenant-abc/
//   │   ├── products/
//   │   ├── logos/
//   │   └── uploads/
//   ├── tenant-xyz/
//   │   ├── products/
//   │   └── ...

export class TenantFileStorage {
  async upload(
    tenantId: string,
    file: Buffer,
    path: string
  ): Promise<string> {
    const key = `${tenantId}/${path}`;

    await s3.putObject({
      Bucket: 'ecommerce-assets',
      Key: key,
      Body: file,
      // Tenant-specific ACL
      Metadata: {
        'tenant-id': tenantId,
      },
    });

    return key;
  }

  async download(tenantId: string, path: string): Promise<Buffer> {
    const key = `${tenantId}/${path}`;

    const object = await s3.getObject({
      Bucket: 'ecommerce-assets',
      Key: key,
    });

    // Verify tenant ownership
    if (object.Metadata?.['tenant-id'] !== tenantId) {
      throw new Error('Access denied');
    }

    return object.Body as Buffer;
  }
}
```

#### 4. Cache Isolation

```typescript
// Redis key naming
// tenant:{tenant_id}:products:{product_id}
// tenant:{tenant_id}:cart:{user_id}

export class TenantCache {
  private getKey(tenantId: string, key: string): string {
    return `tenant:${tenantId}:${key}`;
  }

  async get(tenantId: string, key: string): Promise<string | null> {
    return redis.get(this.getKey(tenantId, key));
  }

  async set(
    tenantId: string,
    key: string,
    value: string,
    ttl?: number
  ): Promise<void> {
    const redisKey = this.getKey(tenantId, key);

    if (ttl) {
      await redis.setex(redisKey, ttl, value);
    } else {
      await redis.set(redisKey, value);
    }
  }

  async delete(tenantId: string, key: string): Promise<void> {
    await redis.del(this.getKey(tenantId, key));
  }

  async flushTenant(tenantId: string): Promise<void> {
    const pattern = `tenant:${tenantId}:*`;
    const keys = await redis.keys(pattern);

    if (keys.length > 0) {
      await redis.del(...keys);
    }
  }
}
```

---

## Customization & Branding

### Tenant Settings Schema

```sql
CREATE TABLE tenant_settings (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,

    -- Branding
    logo_url TEXT,
    favicon_url TEXT,
    primary_color VARCHAR(7) DEFAULT '#007bff',
    secondary_color VARCHAR(7) DEFAULT '#6c757d',
    font_family VARCHAR(100) DEFAULT 'Inter',

    -- Business Info
    business_name VARCHAR(255),
    business_email VARCHAR(255),
    business_phone VARCHAR(20),
    support_email VARCHAR(255),

    -- Localization
    default_currency VARCHAR(3) DEFAULT 'USD',
    default_language VARCHAR(5) DEFAULT 'en',
    timezone VARCHAR(50) DEFAULT 'UTC',
    date_format VARCHAR(50) DEFAULT 'YYYY-MM-DD',

    -- Features
    features_enabled JSONB DEFAULT '{}',
    -- {
    --   "wishlist": true,
    --   "reviews": true,
    --   "guestCheckout": true,
    --   "multiCurrency": false
    -- }

    -- Email Settings
    email_from_name VARCHAR(255),
    email_from_address VARCHAR(255),
    smtp_settings JSONB,

    -- Payment
    payment_methods JSONB DEFAULT '[]',
    -- ["credit_card", "paypal", "stripe"]

    -- Shipping
    shipping_zones JSONB DEFAULT '[]',

    -- SEO
    seo_title VARCHAR(255),
    seo_description TEXT,
    seo_keywords TEXT,

    -- Custom Code
    custom_css TEXT,
    custom_js TEXT,
    header_scripts TEXT,
    footer_scripts TEXT,

    -- Integrations
    integrations JSONB DEFAULT '{}',
    -- {
    --   "google_analytics": "UA-XXXXX",
    --   "facebook_pixel": "XXXXX"
    -- }

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Custom Domain Support

```typescript
// Domain mapping
CREATE TABLE tenant_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    domain VARCHAR(255) UNIQUE NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    ssl_status VARCHAR(50) DEFAULT 'pending', -- pending, active, failed
    ssl_certificate TEXT,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(tenant_id, is_primary) WHERE is_primary = TRUE
);

// Resolve tenant by custom domain
export async function resolveTenantByDomain(
  domain: string
): Promise<Tenant | null> {
  const result = await db.query(
    `SELECT t.* FROM tenants t
     JOIN tenant_domains td ON t.id = td.tenant_id
     WHERE td.domain = $1 AND td.verified_at IS NOT NULL`,
    [domain]
  );

  return result.rows[0] || null;
}
```

---

## Billing & Metering

### Usage Tracking

```sql
CREATE TABLE tenant_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric VARCHAR(100) NOT NULL, -- products, orders, storage_mb, api_calls
    value BIGINT NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(tenant_id, metric, period_start)
);

CREATE INDEX idx_tenant_usage_tenant_period ON tenant_usage(tenant_id, period_start);
```

### Usage Metering Service

```typescript
export class TenantUsageService {
  async trackUsage(
    tenantId: string,
    metric: string,
    value: number
  ): Promise<void> {
    const today = new Date().toISOString().split('T')[0];

    await db.query(
      `INSERT INTO tenant_usage (tenant_id, metric, value, period_start, period_end)
       VALUES ($1, $2, $3, $4, $4)
       ON CONFLICT (tenant_id, metric, period_start)
       DO UPDATE SET value = tenant_usage.value + $3`,
      [tenantId, metric, value, today]
    );
  }

  async checkLimit(
    tenantId: string,
    metric: string
  ): Promise<{ allowed: boolean; current: number; limit: number }> {
    const tenant = await this.getTenant(tenantId);
    const limit = this.getLimit(tenant, metric);

    const usage = await this.getCurrentUsage(tenantId, metric);

    return {
      allowed: usage < limit,
      current: usage,
      limit,
    };
  }

  private getLimit(tenant: Tenant, metric: string): number {
    const limits: Record<string, Record<string, number>> = {
      free: {
        products: 100,
        orders: 1000,
        storage_mb: 1000,
        api_calls: 10000,
      },
      professional: {
        products: 10000,
        orders: 100000,
        storage_mb: 10000,
        api_calls: 1000000,
      },
      enterprise: {
        products: Infinity,
        orders: Infinity,
        storage_mb: Infinity,
        api_calls: Infinity,
      },
    };

    return limits[tenant.tier]?.[metric] || 0;
  }
}
```

### Rate Limiting Per Tenant

```typescript
export class TenantRateLimiter {
  async checkRateLimit(
    tenantId: string,
    endpoint: string
  ): Promise<boolean> {
    const tenant = await getTenant(tenantId);
    const limit = this.getRateLimit(tenant.tier);

    const key = `rate_limit:${tenantId}:${endpoint}`;
    const current = await redis.incr(key);

    if (current === 1) {
      await redis.expire(key, 60); // 1 minute window
    }

    return current <= limit;
  }

  private getRateLimit(tier: string): number {
    const limits: Record<string, number> = {
      free: 100,       // 100 requests/minute
      professional: 1000,  // 1000 requests/minute
      enterprise: 10000,   // 10000 requests/minute
    };

    return limits[tier] || 100;
  }
}
```

---

## Scaling Strategy

### Horizontal Scaling

```yaml
# Kubernetes deployment with tenant awareness
apiVersion: apps/v1
kind: Deployment
metadata:
  name: product-service
spec:
  replicas: 5
  selector:
    matchLabels:
      app: product-service
  template:
    spec:
      containers:
      - name: product-service
        image: product-service:latest
        env:
        - name: TENANT_MODE
          value: "multi"
        - name: DB_POOL_SIZE
          value: "20"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

### Database Sharding

```typescript
// Shard tenants across multiple databases
export class TenantShardManager {
  private shards: DatabaseShard[] = [
    { id: 'shard-1', host: 'db1.example.com', capacity: 1000 },
    { id: 'shard-2', host: 'db2.example.com', capacity: 1000 },
    { id: 'shard-3', host: 'db3.example.com', capacity: 1000 },
  ];

  getShardForTenant(tenantId: string): DatabaseShard {
    // Consistent hashing
    const hash = this.hashTenantId(tenantId);
    const shardIndex = hash % this.shards.length;
    return this.shards[shardIndex];
  }

  private hashTenantId(tenantId: string): number {
    // Simple hash function
    let hash = 0;
    for (let i = 0; i < tenantId.length; i++) {
      hash = ((hash << 5) - hash) + tenantId.charCodeAt(i);
      hash |= 0;
    }
    return Math.abs(hash);
  }
}
```

### Tenant Migration

```typescript
// Migrate tenant to different tier/shard
export class TenantMigrationService {
  async migrateTenant(
    tenantId: string,
    targetTier: string
  ): Promise<void> {
    const tenant = await this.getTenant(tenantId);

    // 1. Create backup
    await this.backupTenantData(tenantId);

    // 2. Provision new resources
    await this.provisionNewResources(tenant, targetTier);

    // 3. Copy data
    await this.copyTenantData(tenant, targetTier);

    // 4. Switch traffic (zero-downtime)
    await this.switchTraffic(tenantId, targetTier);

    // 5. Verify migration
    await this.verifyMigration(tenantId);

    // 6. Cleanup old resources
    await this.cleanupOldResources(tenant);
  }
}
```

---

## Implementation Examples

### Complete API Request Flow

```typescript
// 1. Request arrives
app.post('/api/products',
  tenantContext,        // Extract and validate tenant
  authenticate,         // Verify user authentication
  authorize(['admin']), // Check permissions
  validateRequest,      // Validate request body
  async (req, res) => {
    // 2. Tenant context is available
    const { tenantId, tenant } = req;

    // 3. Check usage limits
    const limitCheck = await usageService.checkLimit(
      tenantId,
      'products'
    );

    if (!limitCheck.allowed) {
      return res.status(429).json({
        error: 'Product limit exceeded',
        limit: limitCheck.limit,
        current: limitCheck.current,
      });
    }

    // 4. Create product with automatic tenant scoping
    const productService = new ProductService(tenantId);
    const product = await productService.create(req.body);

    // 5. Track usage
    await usageService.trackUsage(tenantId, 'products', 1);

    // 6. Publish event (tenant-scoped)
    await eventBus.publish({
      eventType: 'ProductCreated',
      tenantId,
      payload: product,
    });

    res.status(201).json(product);
  }
);
```

### Event Handling with Tenant Context

```typescript
// Event consumer with tenant isolation
export class TenantAwareEventHandler {
  async handleProductCreated(event: DomainEvent): Promise<void> {
    const { tenantId, payload } = event;

    // Get tenant-specific configuration
    const tenant = await getTenant(tenantId);

    // Process in tenant context
    const searchService = new SearchService(tenantId);
    await searchService.indexProduct(payload);

    // Send tenant-specific notifications
    if (tenant.settings.notifyOnNewProduct) {
      const notificationService = new NotificationService(tenantId);
      await notificationService.notifyAdmins('New product created', payload);
    }
  }
}
```

---

## Monitoring & Analytics

### Tenant-Specific Metrics

```typescript
// Prometheus metrics with tenant labels
const productCreations = new Counter({
  name: 'products_created_total',
  help: 'Total products created',
  labelNames: ['tenant_id', 'tenant_tier'],
});

const apiLatency = new Histogram({
  name: 'api_request_duration_seconds',
  help: 'API request latency',
  labelNames: ['tenant_id', 'endpoint', 'method'],
});

// Track metrics
productCreations.inc({
  tenant_id: tenantId,
  tenant_tier: tenant.tier
});

apiLatency.observe(
  { tenant_id: tenantId, endpoint: '/products', method: 'POST' },
  duration
);
```

### Tenant Health Dashboard

```sql
-- Query for tenant health metrics
SELECT
  t.id,
  t.name,
  t.tier,
  COUNT(DISTINCT o.id) as total_orders,
  SUM(o.total_amount) as total_revenue,
  COUNT(DISTINCT u.id) as total_users,
  COUNT(DISTINCT p.id) as total_products,
  MAX(o.created_at) as last_order_at
FROM tenants t
LEFT JOIN orders o ON t.id = o.tenant_id
LEFT JOIN users u ON t.id = u.tenant_id
LEFT JOIN products p ON t.id = p.tenant_id
WHERE t.status = 'active'
GROUP BY t.id
ORDER BY total_revenue DESC;
```

---

## Migration Path

### From Single-Tenant to Multi-Tenant

```sql
-- Step 1: Add tenant_id column to all tables
ALTER TABLE users ADD COLUMN tenant_id UUID;
ALTER TABLE products ADD COLUMN tenant_id UUID;
ALTER TABLE orders ADD COLUMN tenant_id UUID;

-- Step 2: Create default tenant
INSERT INTO tenants (id, slug, name, tier)
VALUES ('default-tenant-id', 'default', 'Default Tenant', 'enterprise');

-- Step 3: Backfill tenant_id
UPDATE users SET tenant_id = 'default-tenant-id';
UPDATE products SET tenant_id = 'default-tenant-id';
UPDATE orders SET tenant_id = 'default-tenant-id';

-- Step 4: Make tenant_id NOT NULL
ALTER TABLE users ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE products ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE orders ALTER COLUMN tenant_id SET NOT NULL;

-- Step 5: Add foreign keys
ALTER TABLE users ADD CONSTRAINT fk_users_tenant
  FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Step 6: Add indexes
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_products_tenant_id ON products(tenant_id);
CREATE INDEX idx_orders_tenant_id ON orders(tenant_id);

-- Step 7: Enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
-- ... create policies
```

---

## Best Practices

### 1. Always Filter by Tenant

```typescript
// ❌ BAD
const users = await db.query('SELECT * FROM users');

// ✅ GOOD
const users = await db.query(
  'SELECT * FROM users WHERE tenant_id = $1',
  [tenantId]
);
```

### 2. Use Tenant Context

```typescript
// ❌ BAD: Passing tenantId everywhere
async function getProduct(productId: string, tenantId: string) {
  return db.query('SELECT * FROM products WHERE id = $1 AND tenant_id = $2',
    [productId, tenantId]);
}

// ✅ GOOD: Use dependency injection
class ProductService {
  constructor(private tenantId: string) {}

  async getProduct(productId: string) {
    return db.query('SELECT * FROM products WHERE id = $1 AND tenant_id = $2',
      [productId, this.tenantId]);
  }
}
```

### 3. Validate Tenant Access

```typescript
// Always verify tenant ownership
async function updateProduct(productId: string, data: any, tenantId: string) {
  const product = await getProduct(productId);

  if (product.tenant_id !== tenantId) {
    throw new Error('Access denied');
  }

  // Update product...
}
```

### 4. Cache Tenant Configuration

```typescript
// Cache tenant data to reduce database queries
const tenantCache = new Map<string, Tenant>();

async function getTenant(tenantId: string): Promise<Tenant> {
  if (tenantCache.has(tenantId)) {
    return tenantCache.get(tenantId)!;
  }

  const tenant = await db.query('SELECT * FROM tenants WHERE id = $1', [tenantId]);
  tenantCache.set(tenantId, tenant);

  return tenant;
}
```

---

## Conclusion

This multi-tenancy architecture provides:

✅ **Scalability**: Support thousands of tenants
✅ **Flexibility**: Different tiers for different needs
✅ **Security**: Complete data isolation
✅ **Customization**: Per-tenant branding and configuration
✅ **Cost-Efficiency**: Shared resources for small tenants, dedicated for large ones
✅ **Compliance**: Meets data residency and privacy requirements

The hybrid approach allows you to start with the pool model for cost efficiency and upgrade tenants to dedicated resources as they grow, providing the best of both worlds.
