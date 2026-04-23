# Event Schemas & Contracts

## Event Schema Standards

All events follow this base structure:

```typescript
interface BaseEvent {
  eventId: string;           // UUID v4
  eventType: string;         // Event name (PascalCase)
  timestamp: string;         // ISO 8601 format
  version: string;           // Semantic versioning (e.g., "1.0.0")
  payload: object;           // Event-specific data
  metadata: EventMetadata;
}

interface EventMetadata {
  source: string;            // Service name that produced the event
  correlationId: string;     // UUID to track related events across services
  causationId: string;       // ID of the event that caused this event
  userId?: string;           // Optional user context
  traceId?: string;          // Distributed tracing ID
}
```

---

## User Service Events

### UserRegistered

```json
{
  "eventId": "550e8400-e29b-41d4-a716-446655440000",
  "eventType": "UserRegistered",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "userId": "usr_7x9k2m4p",
    "email": "john.doe@example.com",
    "firstName": "John",
    "lastName": "Doe",
    "phone": "+1234567890",
    "registrationMethod": "email",
    "emailVerified": false
  },
  "metadata": {
    "source": "user-service",
    "correlationId": "550e8400-e29b-41d4-a716-446655440001",
    "causationId": "550e8400-e29b-41d4-a716-446655440001"
  }
}
```

**Consumers**: Notification Service, Analytics Service, Recommendation Service

---

### UserUpdated

```json
{
  "eventId": "uuid",
  "eventType": "UserUpdated",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "userId": "usr_7x9k2m4p",
    "updatedFields": {
      "firstName": "Jonathan",
      "phone": "+1234567899"
    },
    "previousValues": {
      "firstName": "John",
      "phone": "+1234567890"
    }
  },
  "metadata": {
    "source": "user-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

---

## Product Service Events

### ProductCreated

```json
{
  "eventId": "uuid",
  "eventType": "ProductCreated",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "productId": "prd_4k8n2x7w",
    "vendorId": "vnd_3m9p1q5r",
    "name": "Wireless Noise-Cancelling Headphones",
    "description": "Premium wireless headphones with active noise cancellation",
    "category": "Electronics > Audio > Headphones",
    "price": {
      "amount": 299.99,
      "currency": "USD"
    },
    "images": [
      "https://cdn.example.com/products/prd_4k8n2x7w/main.jpg",
      "https://cdn.example.com/products/prd_4k8n2x7w/side.jpg"
    ],
    "attributes": {
      "brand": "AudioTech",
      "color": "Black",
      "weight": "250g",
      "batteryLife": "30 hours"
    },
    "sku": "AT-WNC-300-BLK",
    "status": "pending_approval"
  },
  "metadata": {
    "source": "product-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "vnd_3m9p1q5r"
  }
}
```

**Consumers**: Search Service, Inventory Service, Analytics Service

---

### PriceChanged

```json
{
  "eventId": "uuid",
  "eventType": "PriceChanged",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "productId": "prd_4k8n2x7w",
    "oldPrice": {
      "amount": 299.99,
      "currency": "USD"
    },
    "newPrice": {
      "amount": 249.99,
      "currency": "USD"
    },
    "reason": "promotional_discount",
    "effectiveDate": "2024-01-16T00:00:00Z"
  },
  "metadata": {
    "source": "product-service",
    "correlationId": "uuid",
    "causationId": "uuid"
  }
}
```

**Consumers**: Cart Service, Search Service, Notification Service (price drop alerts)

---

## Inventory Service Events

### InventoryUpdated

```json
{
  "eventId": "uuid",
  "eventType": "InventoryUpdated",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "productId": "prd_4k8n2x7w",
    "warehouseId": "wh_us_east_1",
    "previousQuantity": 150,
    "currentQuantity": 200,
    "changeType": "restock",
    "reason": "supplier_shipment"
  },
  "metadata": {
    "source": "inventory-service",
    "correlationId": "uuid",
    "causationId": "uuid"
  }
}
```

**Consumers**: Product Service, Search Service, Notification Service

---

### StockReserved

```json
{
  "eventId": "uuid",
  "eventType": "StockReserved",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "reservationId": "rsv_9h3k5m2n",
    "orderId": "ord_6t8p2q4r",
    "items": [
      {
        "productId": "prd_4k8n2x7w",
        "quantity": 2,
        "warehouseId": "wh_us_east_1"
      }
    ],
    "expiresAt": "2024-01-15T10:45:00Z"
  },
  "metadata": {
    "source": "inventory-service",
    "correlationId": "uuid",
    "causationId": "uuid"
  }
}
```

**Consumers**: Order Service

---

### StockLevelLow

```json
{
  "eventId": "uuid",
  "eventType": "StockLevelLow",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "productId": "prd_4k8n2x7w",
    "warehouseId": "wh_us_east_1",
    "currentQuantity": 15,
    "threshold": 20,
    "recommendedReorderQuantity": 100
  },
  "metadata": {
    "source": "inventory-service",
    "correlationId": "uuid",
    "causationId": "uuid"
  }
}
```

**Consumers**: Notification Service, Vendor Service, Analytics Service

---

## Order Service Events

### OrderPlaced

```json
{
  "eventId": "uuid",
  "eventType": "OrderPlaced",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "payload": {
    "orderId": "ord_6t8p2q4r",
    "userId": "usr_7x9k2m4p",
    "items": [
      {
        "productId": "prd_4k8n2x7w",
        "vendorId": "vnd_3m9p1q5r",
        "quantity": 2,
        "unitPrice": {
          "amount": 249.99,
          "currency": "USD"
        },
        "subtotal": {
          "amount": 499.98,
          "currency": "USD"
        }
      }
    ],
    "shippingAddress": {
      "recipientName": "John Doe",
      "addressLine1": "123 Main St",
      "addressLine2": "Apt 4B",
      "city": "New York",
      "state": "NY",
      "postalCode": "10001",
      "country": "US",
      "phone": "+1234567890"
    },
    "billingAddress": {
      "addressLine1": "123 Main St",
      "city": "New York",
      "state": "NY",
      "postalCode": "10001",
      "country": "US"
    },
    "pricing": {
      "subtotal": {
        "amount": 499.98,
        "currency": "USD"
      },
      "shipping": {
        "amount": 10.00,
        "currency": "USD"
      },
      "tax": {
        "amount": 45.00,
        "currency": "USD"
      },
      "discount": {
        "amount": 50.00,
        "currency": "USD"
      },
      "total": {
        "amount": 504.98,
        "currency": "USD"
      }
    },
    "paymentMethodId": "pm_card_visa_1234",
    "shippingMethodId": "shp_standard",
    "promotionCodes": ["SUMMER50"]
  },
  "metadata": {
    "source": "order-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Inventory Service, Payment Service, Notification Service, Analytics Service, Vendor Service

---

### OrderConfirmed

```json
{
  "eventId": "uuid",
  "eventType": "OrderConfirmed",
  "timestamp": "2024-01-15T10:31:00Z",
  "version": "1.0.0",
  "payload": {
    "orderId": "ord_6t8p2q4r",
    "userId": "usr_7x9k2m4p",
    "confirmationNumber": "CONF-2024-001234",
    "estimatedDeliveryDate": "2024-01-20T18:00:00Z"
  },
  "metadata": {
    "source": "order-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Shipping Service, Notification Service, Recommendation Service

---

### OrderCancelled

```json
{
  "eventId": "uuid",
  "eventType": "OrderCancelled",
  "timestamp": "2024-01-15T11:00:00Z",
  "version": "1.0.0",
  "payload": {
    "orderId": "ord_6t8p2q4r",
    "userId": "usr_7x9k2m4p",
    "reason": "customer_request",
    "cancelledBy": "customer",
    "refundAmount": {
      "amount": 504.98,
      "currency": "USD"
    }
  },
  "metadata": {
    "source": "order-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Inventory Service, Payment Service, Notification Service, Analytics Service

---

## Payment Service Events

### PaymentInitiated

```json
{
  "eventId": "uuid",
  "eventType": "PaymentInitiated",
  "timestamp": "2024-01-15T10:30:30Z",
  "version": "1.0.0",
  "payload": {
    "paymentId": "pay_8r4n6k2m",
    "orderId": "ord_6t8p2q4r",
    "userId": "usr_7x9k2m4p",
    "amount": {
      "amount": 504.98,
      "currency": "USD"
    },
    "paymentMethod": {
      "type": "credit_card",
      "last4": "1234",
      "brand": "visa"
    },
    "paymentProvider": "stripe"
  },
  "metadata": {
    "source": "payment-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

---

### PaymentCompleted

```json
{
  "eventId": "uuid",
  "eventType": "PaymentCompleted",
  "timestamp": "2024-01-15T10:30:35Z",
  "version": "1.0.0",
  "payload": {
    "paymentId": "pay_8r4n6k2m",
    "orderId": "ord_6t8p2q4r",
    "userId": "usr_7x9k2m4p",
    "amount": {
      "amount": 504.98,
      "currency": "USD"
    },
    "transactionId": "txn_stripe_ch_3NqZ8w2eZvKYlo2C0Xk9pqrs",
    "paymentProvider": "stripe",
    "capturedAt": "2024-01-15T10:30:35Z"
  },
  "metadata": {
    "source": "payment-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Order Service, Notification Service, Analytics Service, Accounting Service

---

### PaymentFailed

```json
{
  "eventId": "uuid",
  "eventType": "PaymentFailed",
  "timestamp": "2024-01-15T10:30:35Z",
  "version": "1.0.0",
  "payload": {
    "paymentId": "pay_8r4n6k2m",
    "orderId": "ord_6t8p2q4r",
    "userId": "usr_7x9k2m4p",
    "amount": {
      "amount": 504.98,
      "currency": "USD"
    },
    "failureReason": "insufficient_funds",
    "failureCode": "card_declined",
    "paymentProvider": "stripe",
    "retryable": true
  },
  "metadata": {
    "source": "payment-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Order Service, Inventory Service, Notification Service

---

### FraudDetected

```json
{
  "eventId": "uuid",
  "eventType": "FraudDetected",
  "timestamp": "2024-01-15T10:30:32Z",
  "version": "1.0.0",
  "payload": {
    "paymentId": "pay_8r4n6k2m",
    "orderId": "ord_6t8p2q4r",
    "userId": "usr_7x9k2m4p",
    "riskScore": 0.95,
    "riskFactors": [
      "unusual_location",
      "high_order_value",
      "new_payment_method"
    ],
    "action": "block",
    "reviewRequired": true
  },
  "metadata": {
    "source": "payment-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Order Service, Notification Service (admin alert), Fraud Team Dashboard

---

## Shipping Service Events

### ShipmentCreated

```json
{
  "eventId": "uuid",
  "eventType": "ShipmentCreated",
  "timestamp": "2024-01-15T14:00:00Z",
  "version": "1.0.0",
  "payload": {
    "shipmentId": "shp_5h9m2k6n",
    "orderId": "ord_6t8p2q4r",
    "carrier": "fedex",
    "trackingNumber": "1234567890123",
    "estimatedDelivery": "2024-01-20T18:00:00Z",
    "shippingAddress": {
      "recipientName": "John Doe",
      "addressLine1": "123 Main St",
      "city": "New York",
      "state": "NY",
      "postalCode": "10001",
      "country": "US"
    },
    "items": [
      {
        "productId": "prd_4k8n2x7w",
        "quantity": 2
      }
    ]
  },
  "metadata": {
    "source": "shipping-service",
    "correlationId": "uuid",
    "causationId": "uuid"
  }
}
```

**Consumers**: Order Service, Notification Service, Inventory Service

---

### ShipmentDelivered

```json
{
  "eventId": "uuid",
  "eventType": "ShipmentDelivered",
  "timestamp": "2024-01-19T15:30:00Z",
  "version": "1.0.0",
  "payload": {
    "shipmentId": "shp_5h9m2k6n",
    "orderId": "ord_6t8p2q4r",
    "trackingNumber": "1234567890123",
    "deliveredAt": "2024-01-19T15:30:00Z",
    "signedBy": "John Doe",
    "deliveryProof": "https://cdn.example.com/delivery-proofs/shp_5h9m2k6n.jpg"
  },
  "metadata": {
    "source": "shipping-service",
    "correlationId": "uuid",
    "causationId": "uuid"
  }
}
```

**Consumers**: Order Service, Notification Service, Review Service (enable review)

---

## Review Service Events

### ReviewCreated

```json
{
  "eventId": "uuid",
  "eventType": "ReviewCreated",
  "timestamp": "2024-01-20T10:00:00Z",
  "version": "1.0.0",
  "payload": {
    "reviewId": "rev_3k8n5m2p",
    "productId": "prd_4k8n2x7w",
    "userId": "usr_7x9k2m4p",
    "orderId": "ord_6t8p2q4r",
    "rating": 5,
    "title": "Excellent headphones!",
    "comment": "Great sound quality and comfortable to wear for hours.",
    "verified": true,
    "images": [
      "https://cdn.example.com/reviews/rev_3k8n5m2p/img1.jpg"
    ]
  },
  "metadata": {
    "source": "review-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Product Service, Search Service, Notification Service (vendor), Analytics Service

---

## Promotion Service Events

### CouponApplied

```json
{
  "eventId": "uuid",
  "eventType": "CouponApplied",
  "timestamp": "2024-01-15T10:29:00Z",
  "version": "1.0.0",
  "payload": {
    "couponId": "cpn_summer50",
    "code": "SUMMER50",
    "userId": "usr_7x9k2m4p",
    "orderId": "ord_6t8p2q4r",
    "discountType": "percentage",
    "discountValue": 10,
    "discountAmount": {
      "amount": 50.00,
      "currency": "USD"
    },
    "appliedAt": "2024-01-15T10:29:00Z"
  },
  "metadata": {
    "source": "promotion-service",
    "correlationId": "uuid",
    "causationId": "uuid",
    "userId": "usr_7x9k2m4p"
  }
}
```

**Consumers**: Order Service, Analytics Service

---

## Event Versioning Strategy

### Version Compatibility

**Backward Compatible Changes**:
- Adding new optional fields to payload
- Adding new event types
- Adding new metadata fields

**Breaking Changes**:
- Removing fields
- Renaming fields
- Changing field types
- Changing event structure

### Handling Breaking Changes

1. Increment major version (1.0.0 → 2.0.0)
2. Support both versions simultaneously for 6 months
3. Consumers check `version` field and handle accordingly
4. Deprecation notice in documentation

Example:
```typescript
// Consumer handling multiple versions
function handleProductCreated(event: BaseEvent) {
  if (event.version.startsWith('1.')) {
    // Handle v1
  } else if (event.version.startsWith('2.')) {
    // Handle v2
  }
}
```

---

## Event Ordering Guarantees

### Kafka Partitioning Strategy

**Partition Key Selection**:
- User events → `userId`
- Order events → `orderId`
- Product events → `productId`
- Payment events → `orderId`

**Ordering Guarantee**: Events with the same partition key are processed in order.

---

## Event Retention

| Event Type | Retention | Reason |
|------------|-----------|--------|
| Order events | Indefinite | Audit trail |
| Payment events | 7 years | Compliance |
| User events | 2 years | GDPR compliance |
| Product events | 1 year | Historical analysis |
| Inventory events | 90 days | Operational data |
| Analytics events | 30 days | Real-time processing |

---

## Dead Letter Queue (DLQ)

Events that fail processing after 3 retries are moved to DLQ for investigation.

**DLQ Processing**:
1. Alert on-call engineer
2. Log error details
3. Attempt manual replay after fixing issue
4. Archive permanently failed events

---

## Event Monitoring

**Metrics to Track**:
- Event production rate (per topic)
- Event consumption lag
- Processing time per event type
- Error rate per consumer
- DLQ size

**Alerts**:
- Consumer lag > 1000 messages
- DLQ messages > 100
- Event processing time > 5 seconds
