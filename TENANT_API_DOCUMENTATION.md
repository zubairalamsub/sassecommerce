# Tenant Management API Documentation

## Overview

Complete API documentation for multi-tenant e-commerce platform management, including tenant administration, configuration, and billing.

**Base URL**: `https://api.example.com/api/v1`

**Authentication**: Bearer JWT Token

---

## Table of Contents

1. [Authentication](#authentication)
2. [Tenant Management](#tenant-management)
3. [Tenant Configuration](#tenant-configuration)
4. [User Management](#user-management)
5. [Billing & Subscription](#billing--subscription)
6. [Usage & Analytics](#usage--analytics)
7. [Webhooks](#webhooks)

---

## Authentication

### Register Tenant

Create a new tenant account.

**Endpoint**: `POST /tenants/register`

**Public**: Yes

**Request Body**:
```json
{
  "slug": "acme-store",
  "name": "ACME Store",
  "tier": "professional",
  "owner": {
    "firstName": "John",
    "lastName": "Doe",
    "email": "john@acme.com",
    "password": "SecurePass123!",
    "phone": "+1234567890"
  },
  "company": {
    "name": "ACME Corporation",
    "address": {
      "line1": "123 Main St",
      "city": "New York",
      "state": "NY",
      "postalCode": "10001",
      "country": "US"
    }
  }
}
```

**Response**: `201 Created`
```json
{
  "tenant": {
    "id": "tnt_abc123",
    "slug": "acme-store",
    "name": "ACME Store",
    "tier": "professional",
    "status": "trial",
    "trialEndsAt": "2024-02-15T00:00:00Z",
    "subdomain": "acme-store.example.com",
    "createdAt": "2024-01-01T00:00:00Z"
  },
  "owner": {
    "id": "usr_xyz789",
    "email": "john@acme.com",
    "firstName": "John",
    "lastName": "Doe",
    "role": "owner"
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Errors**:
- `400` - Invalid request data
- `409` - Slug already taken
- `422` - Validation errors

---

### Login

Authenticate and get access token.

**Endpoint**: `POST /auth/login`

**Tenant Context**: Required (subdomain or header)

**Request Body**:
```json
{
  "email": "john@acme.com",
  "password": "SecurePass123!"
}
```

**Response**: `200 OK`
```json
{
  "user": {
    "id": "usr_xyz789",
    "email": "john@acme.com",
    "firstName": "John",
    "lastName": "Doe",
    "role": "owner",
    "tenantId": "tnt_abc123"
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 86400
}
```

---

## Tenant Management

### Get Current Tenant

Get details of the authenticated tenant.

**Endpoint**: `GET /tenants/current`

**Authentication**: Required

**Response**: `200 OK`
```json
{
  "id": "tnt_abc123",
  "slug": "acme-store",
  "name": "ACME Store",
  "tier": "professional",
  "status": "active",
  "subdomain": "acme-store.example.com",
  "customDomain": "shop.acme.com",
  "limits": {
    "maxProducts": 10000,
    "maxOrders": 100000,
    "maxUsers": 100,
    "maxStorageMb": 50000,
    "apiCallsPerMinute": 1000
  },
  "usage": {
    "products": 1250,
    "orders": 5430,
    "users": 15,
    "storageMb": 8920,
    "apiCallsToday": 125000
  },
  "subscription": {
    "planId": "plan_professional_monthly",
    "status": "active",
    "currentPeriodStart": "2024-01-01T00:00:00Z",
    "currentPeriodEnd": "2024-02-01T00:00:00Z",
    "cancelAtPeriodEnd": false
  },
  "branding": {
    "logoUrl": "https://cdn.example.com/tenants/tnt_abc123/logo.png",
    "faviconUrl": "https://cdn.example.com/tenants/tnt_abc123/favicon.ico",
    "primaryColor": "#FF6B00",
    "secondaryColor": "#1A1A1A",
    "fontFamily": "Inter"
  },
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-15T10:30:00Z"
}
```

---

### Update Tenant

Update tenant information.

**Endpoint**: `PATCH /tenants/current`

**Authentication**: Required (Owner/Admin)

**Request Body**:
```json
{
  "name": "ACME Super Store",
  "company": {
    "name": "ACME Corporation",
    "phone": "+1234567890",
    "email": "support@acme.com"
  }
}
```

**Response**: `200 OK`
```json
{
  "id": "tnt_abc123",
  "slug": "acme-store",
  "name": "ACME Super Store",
  "updatedAt": "2024-01-15T11:00:00Z"
}
```

---

### Delete Tenant

Permanently delete tenant account (with confirmation).

**Endpoint**: `DELETE /tenants/current`

**Authentication**: Required (Owner only)

**Query Parameters**:
- `confirm`: Must be "DELETE" (case-sensitive)

**Request**: `DELETE /tenants/current?confirm=DELETE`

**Response**: `204 No Content`

**Errors**:
- `403` - Only owner can delete tenant
- `400` - Missing confirmation
- `422` - Active subscription must be cancelled first

---

## Tenant Configuration

### Get Configuration

Get all tenant configuration settings.

**Endpoint**: `GET /tenants/current/config`

**Authentication**: Required

**Response**: `200 OK`
```json
{
  "general": {
    "businessName": "ACME Store",
    "businessEmail": "support@acme.com",
    "businessPhone": "+1234567890",
    "supportEmail": "help@acme.com",
    "timezone": "America/New_York",
    "defaultCurrency": "USD",
    "defaultLanguage": "en"
  },
  "localization": {
    "supportedCurrencies": ["USD", "EUR", "GBP"],
    "supportedLanguages": ["en", "es", "fr"],
    "dateFormat": "MM/DD/YYYY",
    "timeFormat": "12h",
    "weekStartsOn": "sunday"
  },
  "features": {
    "wishlist": true,
    "guestCheckout": true,
    "productReviews": true,
    "multiCurrency": false,
    "advancedSearch": true,
    "aiRecommendations": true,
    "socialLogin": {
      "google": true,
      "facebook": false,
      "apple": false
    }
  },
  "checkout": {
    "requirePhoneNumber": true,
    "allowPOBoxes": false,
    "enableGiftMessages": true,
    "enableOrderNotes": true,
    "termsAndConditionsUrl": "https://acme.com/terms"
  },
  "payment": {
    "methods": ["credit_card", "paypal", "apple_pay"],
    "providers": {
      "stripe": {
        "enabled": true,
        "publicKey": "pk_live_..."
      },
      "paypal": {
        "enabled": true,
        "mode": "live"
      }
    },
    "currency": "USD",
    "acceptedCards": ["visa", "mastercard", "amex"]
  },
  "shipping": {
    "zones": [
      {
        "id": "zone_us",
        "name": "United States",
        "countries": ["US"],
        "rates": [
          {
            "name": "Standard Shipping",
            "price": 5.99,
            "estimatedDays": "3-5"
          },
          {
            "name": "Express Shipping",
            "price": 14.99,
            "estimatedDays": "1-2"
          }
        ]
      }
    ],
    "freeShippingThreshold": 50.00,
    "domesticShippingDays": 5,
    "internationalShippingDays": 15
  },
  "email": {
    "fromName": "ACME Store",
    "fromEmail": "noreply@acme.com",
    "replyToEmail": "support@acme.com",
    "smtp": {
      "enabled": true,
      "host": "smtp.sendgrid.net",
      "port": 587,
      "secure": true
    },
    "templates": {
      "orderConfirmation": {
        "enabled": true,
        "subject": "Order Confirmation - {{orderNumber}}"
      },
      "orderShipped": {
        "enabled": true,
        "subject": "Your order has been shipped!"
      }
    }
  },
  "notifications": {
    "channels": {
      "email": true,
      "sms": false,
      "push": true
    },
    "events": {
      "orderPlaced": ["email", "push"],
      "orderShipped": ["email", "sms", "push"],
      "lowStock": ["email"]
    }
  },
  "seo": {
    "metaTitle": "ACME Store - Quality Products Online",
    "metaDescription": "Shop the best quality products at ACME Store",
    "metaKeywords": ["online shopping", "quality products"],
    "googleAnalyticsId": "UA-XXXXX-Y",
    "facebookPixelId": "123456789",
    "googleTagManagerId": "GTM-XXXXX"
  },
  "integrations": {
    "googleAnalytics": {
      "enabled": true,
      "trackingId": "UA-XXXXX-Y"
    },
    "facebookPixel": {
      "enabled": true,
      "pixelId": "123456789"
    },
    "intercom": {
      "enabled": false
    },
    "mailchimp": {
      "enabled": true,
      "apiKey": "...",
      "listId": "..."
    }
  },
  "security": {
    "twoFactorAuth": {
      "enabled": false,
      "required": false
    },
    "passwordPolicy": {
      "minLength": 8,
      "requireUppercase": true,
      "requireLowercase": true,
      "requireNumbers": true,
      "requireSpecialChars": true
    },
    "sessionTimeout": 3600,
    "ipWhitelist": []
  },
  "advanced": {
    "customCss": "",
    "customJs": "",
    "headerScripts": "",
    "footerScripts": "",
    "webhooks": [
      {
        "id": "wh_123",
        "url": "https://acme.com/webhooks/orders",
        "events": ["order.created", "order.updated"],
        "active": true
      }
    ]
  }
}
```

---

### Update Configuration

Update specific configuration sections.

**Endpoint**: `PATCH /tenants/current/config`

**Authentication**: Required (Admin/Owner)

**Request Body** (partial update):
```json
{
  "general": {
    "timezone": "America/Los_Angeles",
    "defaultCurrency": "USD"
  },
  "features": {
    "multiCurrency": true,
    "aiRecommendations": true
  },
  "shipping": {
    "freeShippingThreshold": 75.00
  }
}
```

**Response**: `200 OK`
```json
{
  "message": "Configuration updated successfully",
  "updatedSections": ["general", "features", "shipping"],
  "updatedAt": "2024-01-15T11:30:00Z"
}
```

---

### Get Feature Flags

Get current feature flag configuration.

**Endpoint**: `GET /tenants/current/features`

**Authentication**: Required

**Response**: `200 OK`
```json
{
  "wishlist": true,
  "guestCheckout": true,
  "productReviews": true,
  "multiCurrency": false,
  "advancedSearch": true,
  "aiRecommendations": true,
  "subscriptions": false,
  "preOrders": false,
  "giftCards": true,
  "loyaltyProgram": false
}
```

---

### Toggle Feature

Enable or disable a specific feature.

**Endpoint**: `PUT /tenants/current/features/{featureName}`

**Authentication**: Required (Admin/Owner)

**Request Body**:
```json
{
  "enabled": true
}
```

**Response**: `200 OK`
```json
{
  "feature": "multiCurrency",
  "enabled": true,
  "updatedAt": "2024-01-15T12:00:00Z"
}
```

---

## Branding & Customization

### Update Branding

Update tenant branding settings.

**Endpoint**: `PATCH /tenants/current/branding`

**Authentication**: Required (Admin/Owner)

**Request Body**:
```json
{
  "primaryColor": "#FF6B00",
  "secondaryColor": "#1A1A1A",
  "fontFamily": "Roboto",
  "logoUrl": "https://cdn.example.com/logo.png",
  "faviconUrl": "https://cdn.example.com/favicon.ico"
}
```

**Response**: `200 OK`

---

### Upload Logo

Upload tenant logo.

**Endpoint**: `POST /tenants/current/branding/logo`

**Authentication**: Required (Admin/Owner)

**Content-Type**: `multipart/form-data`

**Form Data**:
- `file`: Image file (PNG, JPG, SVG)

**Response**: `200 OK`
```json
{
  "logoUrl": "https://cdn.example.com/tenants/tnt_abc123/logo.png",
  "size": 45678,
  "dimensions": {
    "width": 200,
    "height": 80
  }
}
```

---

### Custom Domain

#### Add Custom Domain

**Endpoint**: `POST /tenants/current/domains`

**Authentication**: Required (Owner)

**Request Body**:
```json
{
  "domain": "shop.acme.com",
  "isPrimary": true
}
```

**Response**: `201 Created`
```json
{
  "id": "dom_123",
  "domain": "shop.acme.com",
  "isPrimary": true,
  "status": "pending_verification",
  "verificationMethod": "dns",
  "dnsRecords": [
    {
      "type": "CNAME",
      "name": "shop",
      "value": "acme-store.example.com",
      "ttl": 3600
    }
  ],
  "sslStatus": "pending",
  "createdAt": "2024-01-15T13:00:00Z"
}
```

---

#### Verify Domain

**Endpoint**: `POST /tenants/current/domains/{domainId}/verify`

**Authentication**: Required (Owner)

**Response**: `200 OK`
```json
{
  "domain": "shop.acme.com",
  "status": "verified",
  "verifiedAt": "2024-01-15T14:00:00Z",
  "sslStatus": "active"
}
```

---

## User Management

### List Users

Get all users in tenant.

**Endpoint**: `GET /users`

**Authentication**: Required (Admin/Owner)

**Query Parameters**:
- `role`: Filter by role (owner, admin, staff, customer)
- `status`: Filter by status (active, inactive, suspended)
- `limit`: Results per page (default: 20, max: 100)
- `offset`: Pagination offset

**Response**: `200 OK`
```json
{
  "data": [
    {
      "id": "usr_xyz789",
      "email": "john@acme.com",
      "firstName": "John",
      "lastName": "Doe",
      "role": "owner",
      "status": "active",
      "emailVerified": true,
      "lastLoginAt": "2024-01-15T10:00:00Z",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "total": 15,
    "limit": 20,
    "offset": 0,
    "hasMore": false
  }
}
```

---

### Invite User

Invite a new user to the tenant.

**Endpoint**: `POST /users/invite`

**Authentication**: Required (Admin/Owner)

**Request Body**:
```json
{
  "email": "jane@acme.com",
  "role": "staff",
  "firstName": "Jane",
  "lastName": "Smith",
  "sendEmail": true
}
```

**Response**: `201 Created`
```json
{
  "id": "usr_abc456",
  "email": "jane@acme.com",
  "role": "staff",
  "status": "invited",
  "invitationToken": "inv_token123",
  "invitationExpiresAt": "2024-01-22T00:00:00Z"
}
```

---

## Billing & Subscription

### Get Current Plan

**Endpoint**: `GET /tenants/current/billing/plan`

**Authentication**: Required

**Response**: `200 OK`
```json
{
  "current": {
    "id": "plan_professional_monthly",
    "name": "Professional",
    "tier": "professional",
    "interval": "month",
    "price": 99.00,
    "currency": "USD",
    "features": [
      "Up to 10,000 products",
      "100,000 orders/month",
      "100 team members",
      "50GB storage",
      "Priority support",
      "Custom domain",
      "Advanced analytics"
    ]
  },
  "subscription": {
    "id": "sub_123",
    "status": "active",
    "currentPeriodStart": "2024-01-01T00:00:00Z",
    "currentPeriodEnd": "2024-02-01T00:00:00Z",
    "cancelAtPeriodEnd": false,
    "canceledAt": null
  },
  "usage": {
    "products": 1250,
    "orders": 5430,
    "users": 15,
    "storageMb": 8920
  },
  "limits": {
    "maxProducts": 10000,
    "maxOrders": 100000,
    "maxUsers": 100,
    "maxStorageMb": 50000
  }
}
```

---

### List Available Plans

**Endpoint**: `GET /plans`

**Public**: Yes

**Query Parameters**:
- `interval`: monthly, yearly

**Response**: `200 OK`
```json
{
  "plans": [
    {
      "id": "plan_free",
      "name": "Free",
      "tier": "free",
      "price": 0,
      "currency": "USD",
      "interval": "month",
      "limits": {
        "maxProducts": 100,
        "maxOrders": 1000,
        "maxUsers": 5,
        "maxStorageMb": 1000
      },
      "features": [
        "Up to 100 products",
        "1,000 orders/month",
        "5 team members",
        "1GB storage",
        "Email support"
      ]
    },
    {
      "id": "plan_professional_monthly",
      "name": "Professional",
      "tier": "professional",
      "price": 99.00,
      "currency": "USD",
      "interval": "month",
      "limits": {
        "maxProducts": 10000,
        "maxOrders": 100000,
        "maxUsers": 100,
        "maxStorageMb": 50000
      },
      "features": [
        "Up to 10,000 products",
        "100,000 orders/month",
        "100 team members",
        "50GB storage",
        "Priority support",
        "Custom domain",
        "Advanced analytics"
      ]
    },
    {
      "id": "plan_professional_yearly",
      "name": "Professional (Annual)",
      "tier": "professional",
      "price": 990.00,
      "currency": "USD",
      "interval": "year",
      "savings": "Save $198 (17%)",
      "limits": {
        "maxProducts": 10000,
        "maxOrders": 100000,
        "maxUsers": 100,
        "maxStorageMb": 50000
      },
      "features": [
        "Everything in Professional",
        "Annual billing discount"
      ]
    }
  ]
}
```

---

### Upgrade/Downgrade Plan

**Endpoint**: `POST /tenants/current/billing/plan/change`

**Authentication**: Required (Owner)

**Request Body**:
```json
{
  "planId": "plan_professional_yearly",
  "paymentMethodId": "pm_123",
  "prorate": true
}
```

**Response**: `200 OK`
```json
{
  "subscription": {
    "id": "sub_123",
    "planId": "plan_professional_yearly",
    "status": "active",
    "currentPeriodStart": "2024-01-15T00:00:00Z",
    "currentPeriodEnd": "2025-01-15T00:00:00Z"
  },
  "invoice": {
    "id": "inv_456",
    "amount": 841.00,
    "currency": "USD",
    "prorateAmount": -49.50,
    "status": "paid"
  }
}
```

---

### Cancel Subscription

**Endpoint**: `POST /tenants/current/billing/subscription/cancel`

**Authentication**: Required (Owner)

**Request Body**:
```json
{
  "cancelAtPeriodEnd": true,
  "reason": "switching_to_competitor",
  "feedback": "Need more advanced features"
}
```

**Response**: `200 OK`
```json
{
  "subscription": {
    "id": "sub_123",
    "status": "active",
    "cancelAtPeriodEnd": true,
    "canceledAt": "2024-01-15T15:00:00Z",
    "currentPeriodEnd": "2024-02-01T00:00:00Z"
  },
  "message": "Subscription will be cancelled on 2024-02-01"
}
```

---

### Payment Methods

#### List Payment Methods

**Endpoint**: `GET /tenants/current/billing/payment-methods`

**Authentication**: Required (Owner)

**Response**: `200 OK`
```json
{
  "data": [
    {
      "id": "pm_123",
      "type": "card",
      "isDefault": true,
      "card": {
        "brand": "visa",
        "last4": "4242",
        "expMonth": 12,
        "expYear": 2025
      },
      "billingDetails": {
        "name": "John Doe",
        "email": "john@acme.com"
      },
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

#### Add Payment Method

**Endpoint**: `POST /tenants/current/billing/payment-methods`

**Authentication**: Required (Owner)

**Request Body**:
```json
{
  "type": "card",
  "stripeToken": "tok_123",
  "setAsDefault": true
}
```

**Response**: `201 Created`

---

### Invoices

#### List Invoices

**Endpoint**: `GET /tenants/current/billing/invoices`

**Authentication**: Required (Owner)

**Query Parameters**:
- `limit`: Results per page
- `status`: paid, pending, failed

**Response**: `200 OK`
```json
{
  "data": [
    {
      "id": "inv_123",
      "number": "INV-2024-0001",
      "amount": 99.00,
      "currency": "USD",
      "status": "paid",
      "pdfUrl": "https://api.example.com/invoices/inv_123/pdf",
      "periodStart": "2024-01-01T00:00:00Z",
      "periodEnd": "2024-02-01T00:00:00Z",
      "paidAt": "2024-01-01T00:00:00Z",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "total": 12,
    "limit": 20,
    "offset": 0
  }
}
```

---

## Usage & Analytics

### Get Usage Stats

**Endpoint**: `GET /tenants/current/usage`

**Authentication**: Required

**Query Parameters**:
- `period`: current, last_30_days, last_90_days

**Response**: `200 OK`
```json
{
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-01-31T23:59:59Z"
  },
  "metrics": {
    "products": {
      "current": 1250,
      "limit": 10000,
      "percentUsed": 12.5
    },
    "orders": {
      "current": 5430,
      "limit": 100000,
      "percentUsed": 5.43
    },
    "users": {
      "current": 15,
      "limit": 100,
      "percentUsed": 15
    },
    "storage": {
      "current": 8920,
      "limit": 50000,
      "percentUsed": 17.84,
      "unit": "MB"
    },
    "apiCalls": {
      "today": 125000,
      "thisMonth": 2450000,
      "rateLimit": 1000,
      "unit": "per_minute"
    }
  },
  "trends": {
    "orders": {
      "thisMonth": 5430,
      "lastMonth": 4820,
      "growth": 12.7
    },
    "revenue": {
      "thisMonth": 125430.50,
      "lastMonth": 98250.00,
      "growth": 27.7
    }
  }
}
```

---

### Get Analytics

**Endpoint**: `GET /tenants/current/analytics`

**Authentication**: Required

**Query Parameters**:
- `metric`: orders, revenue, customers, products
- `period`: day, week, month, year
- `startDate`: ISO date
- `endDate`: ISO date

**Response**: `200 OK`
```json
{
  "metric": "revenue",
  "period": "month",
  "data": [
    {
      "date": "2024-01-01",
      "value": 12450.50
    },
    {
      "date": "2024-01-02",
      "value": 15230.00
    }
  ],
  "summary": {
    "total": 125430.50,
    "average": 4046.79,
    "min": 8920.00,
    "max": 18450.00
  }
}
```

---

## Webhooks

### List Webhooks

**Endpoint**: `GET /tenants/current/webhooks`

**Authentication**: Required (Admin/Owner)

**Response**: `200 OK`
```json
{
  "data": [
    {
      "id": "wh_123",
      "url": "https://acme.com/webhooks/orders",
      "events": ["order.created", "order.updated", "order.cancelled"],
      "active": true,
      "secret": "whsec_...",
      "createdAt": "2024-01-01T00:00:00Z",
      "lastTriggered": "2024-01-15T10:30:00Z"
    }
  ]
}
```

---

### Create Webhook

**Endpoint**: `POST /tenants/current/webhooks`

**Authentication**: Required (Admin/Owner)

**Request Body**:
```json
{
  "url": "https://acme.com/webhooks/products",
  "events": ["product.created", "product.updated", "product.deleted"],
  "active": true
}
```

**Response**: `201 Created`
```json
{
  "id": "wh_456",
  "url": "https://acme.com/webhooks/products",
  "events": ["product.created", "product.updated", "product.deleted"],
  "active": true,
  "secret": "whsec_abc123xyz",
  "createdAt": "2024-01-15T16:00:00Z"
}
```

---

## Error Responses

### Standard Error Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request data",
    "details": [
      {
        "field": "email",
        "message": "Email is required"
      }
    ],
    "requestId": "req_abc123",
    "timestamp": "2024-01-15T16:00:00Z"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `TENANT_NOT_FOUND` | 404 | Tenant does not exist |
| `TENANT_SUSPENDED` | 403 | Tenant account suspended |
| `TRIAL_EXPIRED` | 402 | Trial period expired |
| `LIMIT_EXCEEDED` | 429 | Usage limit exceeded |
| `INVALID_CREDENTIALS` | 401 | Invalid email or password |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `PLAN_LIMIT_EXCEEDED` | 402 | Current plan limit exceeded |
| `PAYMENT_REQUIRED` | 402 | Payment required to continue |

---

## Rate Limiting

All API endpoints are rate-limited based on tenant tier:

| Tier | Requests per Minute |
|------|---------------------|
| Free | 100 |
| Starter | 500 |
| Professional | 1,000 |
| Enterprise | 10,000 |

**Rate Limit Headers**:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1705334400
```

When rate limit is exceeded:
```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "API rate limit exceeded",
    "retryAfter": 60
  }
}
```

---

## Pagination

List endpoints support cursor-based pagination:

**Query Parameters**:
- `limit`: Results per page (default: 20, max: 100)
- `offset`: Pagination offset

**Response**:
```json
{
  "data": [...],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 0,
    "hasMore": true
  }
}
```

---

## Webhooks Events

### Available Events

| Event | Description |
|-------|-------------|
| `tenant.created` | Tenant account created |
| `tenant.updated` | Tenant information updated |
| `tenant.suspended` | Tenant account suspended |
| `subscription.created` | Subscription started |
| `subscription.updated` | Subscription plan changed |
| `subscription.cancelled` | Subscription cancelled |
| `user.created` | User added to tenant |
| `user.updated` | User information updated |
| `user.deleted` | User removed from tenant |
| `order.created` | New order placed |
| `order.updated` | Order status changed |
| `order.cancelled` | Order cancelled |
| `product.created` | New product added |
| `product.updated` | Product updated |
| `product.deleted` | Product removed |
| `payment.succeeded` | Payment successful |
| `payment.failed` | Payment failed |

### Webhook Payload Example

```json
{
  "id": "evt_123",
  "type": "order.created",
  "tenantId": "tnt_abc123",
  "createdAt": "2024-01-15T17:00:00Z",
  "data": {
    "orderId": "ord_xyz789",
    "orderNumber": "ORD-20240115-000123",
    "totalAmount": 299.99,
    "currency": "USD",
    "status": "pending"
  }
}
```

---

This API documentation provides complete configuration capabilities for all tenant settings and operations!
