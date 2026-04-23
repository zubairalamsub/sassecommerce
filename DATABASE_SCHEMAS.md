# Database Schemas

## Schema Design Principles

1. **Database per Service**: Each microservice owns its database
2. **Denormalization**: Optimize for read performance where needed
3. **Soft Deletes**: Use `deleted_at` timestamp instead of hard deletes
4. **Audit Fields**: Every table includes `created_at`, `updated_at`
5. **UUID Primary Keys**: Use UUIDs for distributed system compatibility
6. **Indexing Strategy**: Index foreign keys and frequently queried fields

---

## User Service - PostgreSQL

### users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    date_of_birth DATE,
    profile_image_url TEXT,
    registration_method VARCHAR(50) DEFAULT 'email', -- email, google, facebook
    status VARCHAR(50) DEFAULT 'active', -- active, suspended, deleted
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
```

### user_addresses
```sql
CREATE TABLE user_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address_type VARCHAR(50) NOT NULL, -- shipping, billing
    is_default BOOLEAN DEFAULT FALSE,
    recipient_name VARCHAR(200),
    phone VARCHAR(20),
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(2) NOT NULL, -- ISO 3166-1 alpha-2
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_user_addresses_user_id ON user_addresses(user_id);
CREATE INDEX idx_user_addresses_default ON user_addresses(user_id, is_default);
```

### user_sessions
```sql
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token VARCHAR(500) NOT NULL,
    device_id VARCHAR(255),
    device_type VARCHAR(50), -- web, ios, android
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_refresh_token ON user_sessions(refresh_token);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
```

### user_preferences
```sql
CREATE TABLE user_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    language VARCHAR(10) DEFAULT 'en',
    currency VARCHAR(3) DEFAULT 'USD',
    timezone VARCHAR(50) DEFAULT 'UTC',
    notification_email BOOLEAN DEFAULT TRUE,
    notification_sms BOOLEAN DEFAULT FALSE,
    notification_push BOOLEAN DEFAULT TRUE,
    marketing_emails BOOLEAN DEFAULT FALSE,
    preferences_json JSONB, -- Additional flexible preferences
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## Product Catalog Service - MongoDB

### products
```javascript
{
  "_id": "prd_4k8n2x7w",
  "vendorId": "vnd_3m9p1q5r",
  "name": "Wireless Noise-Cancelling Headphones",
  "slug": "wireless-noise-cancelling-headphones-audiotech",
  "description": "Premium wireless headphones with active noise cancellation",
  "shortDescription": "Premium ANC headphones",
  "category": {
    "primary": "electronics",
    "secondary": "audio",
    "tertiary": "headphones",
    "path": "Electronics > Audio > Headphones"
  },
  "brand": "AudioTech",
  "sku": "AT-WNC-300-BLK",
  "status": "active", // draft, pending_approval, active, out_of_stock, discontinued
  "pricing": {
    "basePrice": 299.99,
    "salePrice": 249.99,
    "currency": "USD",
    "costPrice": 150.00,
    "onSale": true,
    "saleStartDate": ISODate("2024-01-01T00:00:00Z"),
    "saleEndDate": ISODate("2024-02-01T00:00:00Z")
  },
  "images": [
    {
      "url": "https://cdn.example.com/products/prd_4k8n2x7w/main.jpg",
      "alt": "AudioTech headphones front view",
      "order": 1,
      "isPrimary": true
    },
    {
      "url": "https://cdn.example.com/products/prd_4k8n2x7w/side.jpg",
      "alt": "AudioTech headphones side view",
      "order": 2,
      "isPrimary": false
    }
  ],
  "attributes": {
    "color": "Black",
    "weight": "250g",
    "dimensions": "20cm x 15cm x 8cm",
    "batteryLife": "30 hours",
    "connectivity": "Bluetooth 5.0",
    "warranty": "2 years"
  },
  "specifications": [
    {
      "name": "Driver Size",
      "value": "40mm"
    },
    {
      "name": "Frequency Response",
      "value": "20Hz - 20kHz"
    }
  ],
  "variants": [
    {
      "variantId": "var_001",
      "sku": "AT-WNC-300-BLK",
      "attributes": {
        "color": "Black"
      },
      "price": 249.99,
      "stockQuantity": 150
    },
    {
      "variantId": "var_002",
      "sku": "AT-WNC-300-WHT",
      "attributes": {
        "color": "White"
      },
      "price": 249.99,
      "stockQuantity": 75
    }
  ],
  "seo": {
    "metaTitle": "AudioTech Wireless Noise-Cancelling Headphones",
    "metaDescription": "Premium wireless headphones with 30-hour battery life",
    "keywords": ["wireless headphones", "noise cancelling", "bluetooth"]
  },
  "tags": ["wireless", "bluetooth", "noise-cancelling", "premium"],
  "ratings": {
    "average": 4.7,
    "count": 1250,
    "distribution": {
      "5": 850,
      "4": 300,
      "3": 70,
      "2": 20,
      "1": 10
    }
  },
  "metrics": {
    "viewCount": 25000,
    "purchaseCount": 1500,
    "wishlistCount": 450,
    "conversionRate": 0.06
  },
  "shippingInfo": {
    "weight": 0.5, // kg
    "dimensions": {
      "length": 20,
      "width": 15,
      "height": 8
    },
    "freeShipping": true,
    "shippingClass": "standard"
  },
  "createdAt": ISODate("2024-01-01T00:00:00Z"),
  "updatedAt": ISODate("2024-01-15T10:30:00Z"),
  "deletedAt": null
}

// Indexes
db.products.createIndex({ "vendorId": 1 });
db.products.createIndex({ "sku": 1 }, { unique: true });
db.products.createIndex({ "slug": 1 }, { unique: true });
db.products.createIndex({ "status": 1 });
db.products.createIndex({ "category.primary": 1, "category.secondary": 1 });
db.products.createIndex({ "brand": 1 });
db.products.createIndex({ "tags": 1 });
db.products.createIndex({ "pricing.salePrice": 1 });
db.products.createIndex({ "ratings.average": -1 });
db.products.createIndex({ "createdAt": -1 });
db.products.createIndex({ "name": "text", "description": "text", "brand": "text" });
```

### categories
```javascript
{
  "_id": "cat_electronics",
  "name": "Electronics",
  "slug": "electronics",
  "parentId": null,
  "level": 1,
  "path": "Electronics",
  "description": "Electronic devices and accessories",
  "image": "https://cdn.example.com/categories/electronics.jpg",
  "isActive": true,
  "displayOrder": 1,
  "seo": {
    "metaTitle": "Electronics - Shop Latest Gadgets",
    "metaDescription": "Browse our collection of electronic devices"
  },
  "productCount": 15000,
  "createdAt": ISODate("2024-01-01T00:00:00Z"),
  "updatedAt": ISODate("2024-01-15T10:30:00Z")
}

db.categories.createIndex({ "slug": 1 }, { unique: true });
db.categories.createIndex({ "parentId": 1 });
db.categories.createIndex({ "level": 1 });
```

---

## Inventory Service - PostgreSQL

### inventory
```sql
CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id VARCHAR(50) NOT NULL,
    variant_id VARCHAR(50),
    warehouse_id VARCHAR(50) NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reserved_quantity INTEGER NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    available_quantity INTEGER GENERATED ALWAYS AS (quantity - reserved_quantity) STORED,
    reorder_point INTEGER DEFAULT 20,
    reorder_quantity INTEGER DEFAULT 100,
    last_restocked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(product_id, variant_id, warehouse_id)
);

CREATE INDEX idx_inventory_product_id ON inventory(product_id);
CREATE INDEX idx_inventory_warehouse_id ON inventory(warehouse_id);
CREATE INDEX idx_inventory_available_quantity ON inventory(available_quantity);
CREATE INDEX idx_inventory_reorder ON inventory(product_id)
    WHERE available_quantity <= reorder_point;
```

### inventory_reservations
```sql
CREATE TABLE inventory_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id VARCHAR(50) UNIQUE NOT NULL,
    order_id VARCHAR(50) NOT NULL,
    product_id VARCHAR(50) NOT NULL,
    variant_id VARCHAR(50),
    warehouse_id VARCHAR(50) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status VARCHAR(50) DEFAULT 'active', -- active, released, fulfilled
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_reservations_order_id ON inventory_reservations(order_id);
CREATE INDEX idx_reservations_status ON inventory_reservations(status);
CREATE INDEX idx_reservations_expires_at ON inventory_reservations(expires_at)
    WHERE status = 'active';
```

### inventory_movements
```sql
CREATE TABLE inventory_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id VARCHAR(50) NOT NULL,
    variant_id VARCHAR(50),
    warehouse_id VARCHAR(50) NOT NULL,
    movement_type VARCHAR(50) NOT NULL, -- restock, sale, return, adjustment, transfer
    quantity INTEGER NOT NULL,
    reference_id VARCHAR(50), -- order_id, transfer_id, etc.
    reference_type VARCHAR(50),
    reason VARCHAR(255),
    performed_by VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_movements_product_id ON inventory_movements(product_id);
CREATE INDEX idx_movements_warehouse_id ON inventory_movements(warehouse_id);
CREATE INDEX idx_movements_type ON inventory_movements(movement_type);
CREATE INDEX idx_movements_created_at ON inventory_movements(created_at);
```

### warehouses
```sql
CREATE TABLE warehouses (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    address_line1 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(2),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    is_active BOOLEAN DEFAULT TRUE,
    capacity INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## Order Service - PostgreSQL + Event Store

### orders
```sql
CREATE TABLE orders (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    order_number VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending, payment_pending, confirmed, processing, shipped, delivered, cancelled, refunded
    payment_status VARCHAR(50) DEFAULT 'pending',
    -- pending, authorized, captured, failed, refunded
    fulfillment_status VARCHAR(50) DEFAULT 'unfulfilled',
    -- unfulfilled, partially_fulfilled, fulfilled

    -- Pricing
    subtotal_amount DECIMAL(12, 2) NOT NULL,
    shipping_amount DECIMAL(12, 2) DEFAULT 0,
    tax_amount DECIMAL(12, 2) DEFAULT 0,
    discount_amount DECIMAL(12, 2) DEFAULT 0,
    total_amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    -- Shipping
    shipping_method_id VARCHAR(50),
    tracking_number VARCHAR(100),
    estimated_delivery_date TIMESTAMP,
    actual_delivery_date TIMESTAMP,

    -- Addresses (denormalized for historical accuracy)
    shipping_address JSONB NOT NULL,
    billing_address JSONB NOT NULL,

    -- Payment
    payment_method_id VARCHAR(50),
    payment_provider VARCHAR(50),
    transaction_id VARCHAR(100),

    -- Metadata
    notes TEXT,
    customer_notes TEXT,
    ip_address INET,
    user_agent TEXT,

    -- Timestamps
    placed_at TIMESTAMP DEFAULT NOW(),
    confirmed_at TIMESTAMP,
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);
CREATE INDEX idx_orders_order_number ON orders(order_number);
CREATE INDEX idx_orders_placed_at ON orders(placed_at DESC);
CREATE INDEX idx_orders_user_placed ON orders(user_id, placed_at DESC);
```

### order_items
```sql
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id VARCHAR(50) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id VARCHAR(50) NOT NULL,
    variant_id VARCHAR(50),
    vendor_id VARCHAR(50) NOT NULL,

    -- Product snapshot (denormalized)
    product_name VARCHAR(255) NOT NULL,
    product_sku VARCHAR(100),
    product_image_url TEXT,
    product_attributes JSONB,

    -- Pricing
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(12, 2) NOT NULL,
    subtotal DECIMAL(12, 2) NOT NULL,
    discount_amount DECIMAL(12, 2) DEFAULT 0,
    tax_amount DECIMAL(12, 2) DEFAULT 0,
    total_amount DECIMAL(12, 2) NOT NULL,

    -- Fulfillment
    fulfillment_status VARCHAR(50) DEFAULT 'unfulfilled',
    warehouse_id VARCHAR(50),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_vendor_id ON order_items(vendor_id);
```

### order_events (Event Sourcing)
```sql
CREATE TABLE order_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(50) UNIQUE NOT NULL,
    order_id VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_version VARCHAR(10) NOT NULL,
    payload JSONB NOT NULL,
    metadata JSONB,
    sequence_number BIGSERIAL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_order_events_order_id ON order_events(order_id, sequence_number);
CREATE INDEX idx_order_events_type ON order_events(event_type);
CREATE INDEX idx_order_events_created_at ON order_events(created_at);
```

### order_status_history
```sql
CREATE TABLE order_status_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id VARCHAR(50) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    old_status VARCHAR(50),
    new_status VARCHAR(50) NOT NULL,
    changed_by VARCHAR(50),
    reason TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_order_status_history_order_id ON order_status_history(order_id, created_at DESC);
```

---

## Payment Service - PostgreSQL

### payments
```sql
CREATE TABLE payments (
    id VARCHAR(50) PRIMARY KEY,
    order_id VARCHAR(50) NOT NULL,
    user_id VARCHAR(50) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending, authorized, captured, failed, cancelled, refunded, partially_refunded

    -- Payment method
    payment_method_id VARCHAR(50) NOT NULL,
    payment_method_type VARCHAR(50), -- credit_card, debit_card, paypal, apple_pay, etc.
    payment_provider VARCHAR(50) NOT NULL, -- stripe, paypal, square

    -- Provider details
    provider_payment_id VARCHAR(255),
    provider_transaction_id VARCHAR(255),

    -- Card details (tokenized/masked)
    card_last4 VARCHAR(4),
    card_brand VARCHAR(50),
    card_exp_month INTEGER,
    card_exp_year INTEGER,

    -- Fraud detection
    fraud_score DECIMAL(3, 2),
    fraud_status VARCHAR(50), -- safe, review, block
    risk_factors JSONB,

    -- Timestamps
    authorized_at TIMESTAMP,
    captured_at TIMESTAMP,
    failed_at TIMESTAMP,
    failure_reason TEXT,
    failure_code VARCHAR(100),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX idx_payments_provider_payment_id ON payments(provider_payment_id);
```

### refunds
```sql
CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id VARCHAR(50) NOT NULL REFERENCES payments(id),
    order_id VARCHAR(50) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    reason VARCHAR(50) NOT NULL, -- customer_request, fraud, duplicate, etc.
    reason_details TEXT,
    status VARCHAR(50) DEFAULT 'pending', -- pending, processed, failed
    provider_refund_id VARCHAR(255),
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_refunds_payment_id ON refunds(payment_id);
CREATE INDEX idx_refunds_order_id ON refunds(order_id);
```

### payment_methods
```sql
CREATE TABLE payment_methods (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    type VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_method_id VARCHAR(255) NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,

    -- Card info (masked)
    card_last4 VARCHAR(4),
    card_brand VARCHAR(50),
    card_exp_month INTEGER,
    card_exp_year INTEGER,
    card_fingerprint VARCHAR(255),

    -- Billing address
    billing_address JSONB,

    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id);
CREATE INDEX idx_payment_methods_default ON payment_methods(user_id, is_default);
```

---

## Cart Service - Redis

### Cart Data Structure (Redis Hash)

Key pattern: `cart:{userId}`

```json
{
  "userId": "usr_7x9k2m4p",
  "items": [
    {
      "productId": "prd_4k8n2x7w",
      "variantId": "var_001",
      "quantity": 2,
      "unitPrice": 249.99,
      "addedAt": "2024-01-15T10:00:00Z"
    }
  ],
  "subtotal": 499.98,
  "itemCount": 2,
  "lastUpdated": "2024-01-15T10:30:00Z",
  "expiresAt": "2024-01-16T10:30:00Z"
}
```

**TTL**: 24 hours from last update

**Commands**:
```redis
# Set cart
HSET cart:usr_7x9k2m4p data '{json}'
EXPIRE cart:usr_7x9k2m4p 86400

# Get cart
HGET cart:usr_7x9k2m4p data

# Delete cart
DEL cart:usr_7x9k2m4p
```

---

## Review & Rating Service - MongoDB

### reviews
```javascript
{
  "_id": "rev_3k8n5m2p",
  "productId": "prd_4k8n2x7w",
  "userId": "usr_7x9k2m4p",
  "orderId": "ord_6t8p2q4r",
  "rating": 5,
  "title": "Excellent headphones!",
  "comment": "Great sound quality and comfortable to wear for hours.",
  "pros": ["Excellent sound", "Comfortable", "Long battery life"],
  "cons": ["A bit expensive"],
  "verified": true, // purchased product
  "images": [
    {
      "url": "https://cdn.example.com/reviews/rev_3k8n5m2p/img1.jpg",
      "thumbnailUrl": "https://cdn.example.com/reviews/rev_3k8n5m2p/img1_thumb.jpg"
    }
  ],
  "status": "published", // pending, published, rejected
  "moderatedBy": "admin_001",
  "moderatedAt": ISODate("2024-01-20T11:00:00Z"),
  "helpfulCount": 45,
  "notHelpfulCount": 2,
  "reportCount": 0,
  "vendorResponse": {
    "comment": "Thank you for your feedback!",
    "respondedAt": ISODate("2024-01-21T09:00:00Z"),
    "respondedBy": "vnd_3m9p1q5r"
  },
  "createdAt": ISODate("2024-01-20T10:00:00Z"),
  "updatedAt": ISODate("2024-01-20T11:00:00Z")
}

db.reviews.createIndex({ "productId": 1, "createdAt": -1 });
db.reviews.createIndex({ "userId": 1 });
db.reviews.createIndex({ "rating": 1 });
db.reviews.createIndex({ "status": 1 });
db.reviews.createIndex({ "verified": 1 });
```

---

## Notification Service - MongoDB

### notifications
```javascript
{
  "_id": "ntf_8k3m5n2p",
  "userId": "usr_7x9k2m4p",
  "type": "order_shipped",
  "channel": "email", // email, sms, push
  "status": "sent", // pending, sent, failed, delivered, read
  "priority": "normal", // low, normal, high, urgent
  "subject": "Your order has been shipped!",
  "content": {
    "template": "order_shipped",
    "variables": {
      "orderNumber": "ORD-2024-001234",
      "trackingNumber": "1234567890123",
      "estimatedDelivery": "2024-01-20"
    }
  },
  "recipient": {
    "email": "john.doe@example.com",
    "phone": "+1234567890"
  },
  "metadata": {
    "orderId": "ord_6t8p2q4r",
    "shipmentId": "shp_5h9m2k6n"
  },
  "sentAt": ISODate("2024-01-15T14:05:00Z"),
  "deliveredAt": ISODate("2024-01-15T14:05:30Z"),
  "readAt": null,
  "failureReason": null,
  "retryCount": 0,
  "createdAt": ISODate("2024-01-15T14:05:00Z"),
  "updatedAt": ISODate("2024-01-15T14:05:30Z")
}

db.notifications.createIndex({ "userId": 1, "createdAt": -1 });
db.notifications.createIndex({ "status": 1 });
db.notifications.createIndex({ "type": 1 });
db.notifications.createIndex({ "channel": 1 });
```

---

## Analytics Service - Data Warehouse Schema (Snowflake/BigQuery)

### fact_orders
```sql
CREATE TABLE fact_orders (
    order_id VARCHAR(50) PRIMARY KEY,
    order_date DATE NOT NULL,
    order_timestamp TIMESTAMP NOT NULL,
    user_id VARCHAR(50) NOT NULL,

    -- Metrics
    subtotal DECIMAL(12, 2),
    shipping_amount DECIMAL(12, 2),
    tax_amount DECIMAL(12, 2),
    discount_amount DECIMAL(12, 2),
    total_amount DECIMAL(12, 2),
    item_count INTEGER,

    -- Dimensions
    status VARCHAR(50),
    payment_method VARCHAR(50),
    shipping_method VARCHAR(50),
    country VARCHAR(2),
    device_type VARCHAR(50),

    -- Timestamps
    placed_at TIMESTAMP,
    confirmed_at TIMESTAMP,
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,

    loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
);

CREATE INDEX idx_fact_orders_date ON fact_orders(order_date);
CREATE INDEX idx_fact_orders_user ON fact_orders(user_id);
```

### fact_product_views
```sql
CREATE TABLE fact_product_views (
    id VARCHAR(50) PRIMARY KEY,
    view_date DATE NOT NULL,
    view_timestamp TIMESTAMP NOT NULL,
    user_id VARCHAR(50),
    session_id VARCHAR(50),
    product_id VARCHAR(50) NOT NULL,
    category VARCHAR(100),
    price DECIMAL(12, 2),
    device_type VARCHAR(50),
    country VARCHAR(2),
    referrer_source VARCHAR(100),
    loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
);

CREATE INDEX idx_fact_views_date ON fact_product_views(view_date);
CREATE INDEX idx_fact_views_product ON fact_product_views(product_id);
```

### dim_users
```sql
CREATE TABLE dim_users (
    user_id VARCHAR(50) PRIMARY KEY,
    email VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    registration_date DATE,
    country VARCHAR(2),
    customer_segment VARCHAR(50), -- new, regular, vip
    lifetime_value DECIMAL(12, 2),
    order_count INTEGER,
    loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
);
```

### dim_products
```sql
CREATE TABLE dim_products (
    product_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255),
    category VARCHAR(100),
    brand VARCHAR(100),
    vendor_id VARCHAR(50),
    current_price DECIMAL(12, 2),
    cost_price DECIMAL(12, 2),
    loaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP()
);
```

---

## Data Relationships

### Cross-Service References

Services reference entities from other services using IDs only (not foreign keys):

```
User Service (userId)
    ↓
Order Service (order.userId)
    ↓
Payment Service (payment.userId, payment.orderId)
```

**No direct database joins across services** - use events or API calls for data aggregation.

---

## Backup & Recovery Strategy

### PostgreSQL
- Continuous archiving with WAL
- Daily full backups
- Point-in-time recovery (PITR)
- Retention: 30 days

### MongoDB
- Daily snapshots
- Oplogs for point-in-time recovery
- Retention: 30 days

### Redis
- RDB snapshots every hour
- AOF for durability
- Retention: 7 days

---

## Performance Optimization

### Connection Pooling
- PgBouncer for PostgreSQL (100-500 connections)
- MongoDB connection pool (50-200 connections)
- Redis connection pool (20-50 connections)

### Query Optimization
- Analyze slow queries (> 100ms)
- Add indexes for common query patterns
- Use EXPLAIN ANALYZE for query planning
- Implement read replicas for read-heavy workloads

### Caching Strategy
- Cache frequently accessed data in Redis
- Cache invalidation via events
- Cache-aside pattern

---

## Data Migrations

### Schema Evolution Strategy

1. **Backward-compatible changes first**
   - Add new nullable columns
   - Deploy application code
   - Backfill data
   - Make column non-nullable if needed

2. **Breaking changes**
   - Blue-green deployment
   - Run both old and new schemas temporarily
   - Gradual migration

3. **MongoDB schema changes**
   - Flexible schema allows gradual migration
   - Use schema validation for critical fields
