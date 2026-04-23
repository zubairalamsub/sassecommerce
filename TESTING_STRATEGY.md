# Testing Strategy

## Overview

This document outlines the comprehensive testing strategy for the e-commerce platform, covering both frontend and backend testing approaches to ensure high quality, reliability, and maintainability.

---

## Testing Pyramid

```
                    /\
                   /  \
                  / E2E \               5% - End-to-End Tests
                 /______\
                /        \
               /Integration\            25% - Integration Tests
              /____________\
             /              \
            /   Unit Tests   \          70% - Unit Tests
           /__________________\
```

---

## Backend Testing Strategy

### 1. Unit Testing

**Purpose**: Test individual functions, methods, and classes in isolation

**Framework**: Jest (Node.js), Go Test (Go)

**Coverage Target**: > 80%

#### Example: User Service Unit Test

```typescript
// user.service.test.ts
import { UserService } from './user.service';
import { UserRepository } from './user.repository';

describe('UserService', () => {
  let userService: UserService;
  let userRepository: jest.Mocked<UserRepository>;

  beforeEach(() => {
    userRepository = {
      findByEmail: jest.fn(),
      create: jest.fn(),
      update: jest.fn(),
    } as any;

    userService = new UserService(userRepository);
  });

  describe('registerUser', () => {
    it('should create a new user with hashed password', async () => {
      const userData = {
        email: 'test@example.com',
        password: 'password123',
        firstName: 'John',
        lastName: 'Doe',
      };

      userRepository.findByEmail.mockResolvedValue(null);
      userRepository.create.mockResolvedValue({
        id: 'usr_123',
        email: userData.email,
        firstName: userData.firstName,
        lastName: userData.lastName,
      });

      const result = await userService.registerUser(userData);

      expect(result).toHaveProperty('id');
      expect(result.email).toBe(userData.email);
      expect(userRepository.create).toHaveBeenCalledWith(
        expect.objectContaining({
          email: userData.email,
          passwordHash: expect.any(String),
        })
      );
    });

    it('should throw error if email already exists', async () => {
      const userData = {
        email: 'existing@example.com',
        password: 'password123',
      };

      userRepository.findByEmail.mockResolvedValue({
        id: 'usr_existing',
        email: userData.email,
      } as any);

      await expect(userService.registerUser(userData)).rejects.toThrow(
        'Email already exists'
      );
    });
  });

  describe('hashPassword', () => {
    it('should hash password with bcrypt', async () => {
      const password = 'mySecurePassword123';
      const hashed = await userService.hashPassword(password);

      expect(hashed).not.toBe(password);
      expect(hashed.length).toBeGreaterThan(30);
    });
  });
});
```

#### Go Unit Test Example

```go
// inventory_service_test.go
package inventory

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockInventoryRepo struct {
    mock.Mock
}

func (m *MockInventoryRepo) GetQuantity(productID string) (int, error) {
    args := m.Called(productID)
    return args.Int(0), args.Error(1)
}

func (m *MockInventoryRepo) ReserveStock(productID string, quantity int) error {
    args := m.Called(productID, quantity)
    return args.Error(0)
}

func TestReserveStock(t *testing.T) {
    mockRepo := new(MockInventoryRepo)
    service := NewInventoryService(mockRepo)

    // Test successful reservation
    t.Run("successful reservation", func(t *testing.T) {
        mockRepo.On("GetQuantity", "prd_123").Return(100, nil)
        mockRepo.On("ReserveStock", "prd_123", 10).Return(nil)

        err := service.ReserveStock("prd_123", 10)

        assert.NoError(t, err)
        mockRepo.AssertExpectations(t)
    })

    // Test insufficient stock
    t.Run("insufficient stock", func(t *testing.T) {
        mockRepo.On("GetQuantity", "prd_456").Return(5, nil)

        err := service.ReserveStock("prd_456", 10)

        assert.Error(t, err)
        assert.Equal(t, "insufficient stock", err.Error())
    })
}
```

---

### 2. Integration Testing

**Purpose**: Test interactions between components, database, external services

**Framework**: Jest + Testcontainers, Supertest

**Coverage Target**: Critical flows

#### Database Integration Test

```typescript
// user.repository.integration.test.ts
import { UserRepository } from './user.repository';
import { PostgreSqlContainer } from 'testcontainers';
import { Pool } from 'pg';

describe('UserRepository Integration Tests', () => {
  let container: any;
  let pool: Pool;
  let userRepository: UserRepository;

  beforeAll(async () => {
    // Start PostgreSQL container
    container = await new PostgreSqlContainer()
      .withDatabase('test_db')
      .start();

    pool = new Pool({
      host: container.getHost(),
      port: container.getPort(),
      database: container.getDatabase(),
      user: container.getUsername(),
      password: container.getPassword(),
    });

    // Run migrations
    await runMigrations(pool);

    userRepository = new UserRepository(pool);
  }, 60000);

  afterAll(async () => {
    await pool.end();
    await container.stop();
  });

  afterEach(async () => {
    await pool.query('DELETE FROM users');
  });

  it('should create and retrieve a user', async () => {
    const userData = {
      email: 'test@example.com',
      passwordHash: 'hashed_password',
      firstName: 'John',
      lastName: 'Doe',
    };

    const createdUser = await userRepository.create(userData);
    expect(createdUser).toHaveProperty('id');

    const retrievedUser = await userRepository.findById(createdUser.id);
    expect(retrievedUser?.email).toBe(userData.email);
  });

  it('should enforce unique email constraint', async () => {
    const userData = {
      email: 'duplicate@example.com',
      passwordHash: 'hashed_password',
      firstName: 'John',
    };

    await userRepository.create(userData);

    await expect(userRepository.create(userData)).rejects.toThrow();
  });
});
```

#### API Integration Test

```typescript
// order.api.integration.test.ts
import request from 'supertest';
import { app } from '../app';
import { setupTestDatabase, teardownTestDatabase } from './test-utils';

describe('Order API Integration Tests', () => {
  let authToken: string;
  let userId: string;

  beforeAll(async () => {
    await setupTestDatabase();

    // Create test user and get auth token
    const response = await request(app)
      .post('/api/v1/users/register')
      .send({
        email: 'test@example.com',
        password: 'password123',
        firstName: 'John',
        lastName: 'Doe',
      });

    userId = response.body.id;
    authToken = response.body.token;
  });

  afterAll(async () => {
    await teardownTestDatabase();
  });

  describe('POST /api/v1/orders', () => {
    it('should create a new order', async () => {
      const orderData = {
        items: [
          {
            productId: 'prd_123',
            quantity: 2,
            unitPrice: 99.99,
          },
        ],
        shippingAddress: {
          addressLine1: '123 Main St',
          city: 'New York',
          state: 'NY',
          postalCode: '10001',
          country: 'US',
        },
        paymentMethodId: 'pm_test_123',
      };

      const response = await request(app)
        .post('/api/v1/orders')
        .set('Authorization', `Bearer ${authToken}`)
        .send(orderData)
        .expect(201);

      expect(response.body).toHaveProperty('orderId');
      expect(response.body.status).toBe('pending');
      expect(response.body.items).toHaveLength(1);
    });

    it('should return 401 without authentication', async () => {
      await request(app)
        .post('/api/v1/orders')
        .send({})
        .expect(401);
    });

    it('should validate required fields', async () => {
      const response = await request(app)
        .post('/api/v1/orders')
        .set('Authorization', `Bearer ${authToken}`)
        .send({
          items: [],
        })
        .expect(400);

      expect(response.body.error).toContain('items');
    });
  });
});
```

---

### 3. Event-Driven Testing

**Purpose**: Test event publishing and consumption

#### Event Publishing Test

```typescript
// order.events.test.ts
import { OrderService } from './order.service';
import { EventBus } from '../infrastructure/event-bus';

describe('Order Event Publishing', () => {
  let orderService: OrderService;
  let eventBus: jest.Mocked<EventBus>;

  beforeEach(() => {
    eventBus = {
      publish: jest.fn(),
      subscribe: jest.fn(),
    } as any;

    orderService = new OrderService(eventBus);
  });

  it('should publish OrderPlaced event when order is created', async () => {
    const orderData = {
      userId: 'usr_123',
      items: [{ productId: 'prd_456', quantity: 2 }],
    };

    await orderService.createOrder(orderData);

    expect(eventBus.publish).toHaveBeenCalledWith(
      expect.objectContaining({
        eventType: 'OrderPlaced',
        payload: expect.objectContaining({
          userId: 'usr_123',
        }),
      })
    );
  });
});
```

#### Event Consumer Test

```typescript
// inventory.event-handler.test.ts
import { InventoryEventHandler } from './inventory.event-handler';
import { InventoryService } from './inventory.service';

describe('Inventory Event Handler', () => {
  let handler: InventoryEventHandler;
  let inventoryService: jest.Mocked<InventoryService>;

  beforeEach(() => {
    inventoryService = {
      reserveStock: jest.fn(),
      releaseStock: jest.fn(),
    } as any;

    handler = new InventoryEventHandler(inventoryService);
  });

  it('should reserve stock when OrderPlaced event is received', async () => {
    const event = {
      eventType: 'OrderPlaced',
      payload: {
        orderId: 'ord_123',
        items: [
          { productId: 'prd_456', quantity: 2 },
        ],
      },
    };

    inventoryService.reserveStock.mockResolvedValue(undefined);

    await handler.handleOrderPlaced(event);

    expect(inventoryService.reserveStock).toHaveBeenCalledWith(
      'prd_456',
      2,
      'ord_123'
    );
  });

  it('should publish StockReserved event on success', async () => {
    // Test implementation
  });
});
```

---

### 4. Contract Testing

**Purpose**: Ensure API contracts between services are maintained

**Framework**: Pact

#### Consumer Contract Test

```typescript
// order-service.consumer.test.ts
import { PactV3, MatchersV3 } from '@pact-foundation/pact';
import { ProductServiceClient } from './product-service.client';

const { eachLike, like } = MatchersV3;

describe('Order Service -> Product Service Contract', () => {
  const provider = new PactV3({
    consumer: 'order-service',
    provider: 'product-service',
  });

  it('should get product details', async () => {
    await provider
      .given('product prd_123 exists')
      .uponReceiving('a request for product details')
      .withRequest({
        method: 'GET',
        path: '/api/v1/products/prd_123',
      })
      .willRespondWith({
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: {
          productId: like('prd_123'),
          name: like('Wireless Headphones'),
          price: like(249.99),
          inStock: like(true),
        },
      });

    await provider.executeTest(async (mockServer) => {
      const client = new ProductServiceClient(mockServer.url);
      const product = await client.getProduct('prd_123');

      expect(product.productId).toBe('prd_123');
      expect(product.price).toBeGreaterThan(0);
    });
  });
});
```

---

### 5. Load & Performance Testing

**Purpose**: Test system under load

**Framework**: k6, Artillery

#### k6 Load Test

```javascript
// load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 100 }, // Ramp-up to 100 users
    { duration: '5m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 200 }, // Ramp-up to 200 users
    { duration: '5m', target: 200 }, // Stay at 200 users
    { duration: '2m', target: 0 },   // Ramp-down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% < 500ms, 99% < 1s
    http_req_failed: ['rate<0.01'], // Error rate < 1%
  },
};

export default function () {
  // Test product search
  const searchResponse = http.get(
    'https://api.example.com/api/v1/search/products?q=headphones'
  );

  check(searchResponse, {
    'search status is 200': (r) => r.status === 200,
    'search response time < 200ms': (r) => r.timings.duration < 200,
  });

  sleep(1);

  // Test product details
  const productResponse = http.get(
    'https://api.example.com/api/v1/products/prd_123'
  );

  check(productResponse, {
    'product status is 200': (r) => r.status === 200,
    'product response time < 100ms': (r) => r.timings.duration < 100,
  });

  sleep(1);
}
```

---

### 6. Chaos Engineering

**Purpose**: Test system resilience

**Framework**: Chaos Mesh, Gremlin

#### Example Chaos Experiments

```yaml
# chaos-experiment.yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: pod-failure-test
spec:
  action: pod-failure
  mode: one
  duration: '30s'
  selector:
    namespaces:
      - production
    labelSelectors:
      'app': 'order-service'
```

---

## Frontend Testing Strategy

### 1. Unit Testing

**Purpose**: Test individual components and functions

**Framework**: Jest + React Testing Library (React), Vitest (Vue)

**Coverage Target**: > 75%

#### React Component Unit Test

```typescript
// ProductCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { ProductCard } from './ProductCard';

describe('ProductCard', () => {
  const mockProduct = {
    id: 'prd_123',
    name: 'Wireless Headphones',
    price: 249.99,
    image: 'https://example.com/image.jpg',
    rating: 4.5,
  };

  it('should render product information', () => {
    render(<ProductCard product={mockProduct} />);

    expect(screen.getByText('Wireless Headphones')).toBeInTheDocument();
    expect(screen.getByText('$249.99')).toBeInTheDocument();
    expect(screen.getByAltText('Wireless Headphones')).toHaveAttribute(
      'src',
      mockProduct.image
    );
  });

  it('should call onAddToCart when button is clicked', () => {
    const onAddToCart = jest.fn();
    render(<ProductCard product={mockProduct} onAddToCart={onAddToCart} />);

    const addButton = screen.getByRole('button', { name: /add to cart/i });
    fireEvent.click(addButton);

    expect(onAddToCart).toHaveBeenCalledWith(mockProduct.id);
  });

  it('should show out of stock message when stock is 0', () => {
    render(<ProductCard product={{ ...mockProduct, stock: 0 }} />);

    expect(screen.getByText(/out of stock/i)).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /add to cart/i })
    ).toBeDisabled();
  });
});
```

#### Hook Testing

```typescript
// useCart.test.ts
import { renderHook, act } from '@testing-library/react';
import { useCart } from './useCart';

describe('useCart', () => {
  it('should add item to cart', () => {
    const { result } = renderHook(() => useCart());

    act(() => {
      result.current.addItem({
        productId: 'prd_123',
        quantity: 2,
        price: 99.99,
      });
    });

    expect(result.current.items).toHaveLength(1);
    expect(result.current.totalItems).toBe(2);
    expect(result.current.subtotal).toBe(199.98);
  });

  it('should remove item from cart', () => {
    const { result } = renderHook(() => useCart());

    act(() => {
      result.current.addItem({
        productId: 'prd_123',
        quantity: 2,
        price: 99.99,
      });
    });

    act(() => {
      result.current.removeItem('prd_123');
    });

    expect(result.current.items).toHaveLength(0);
    expect(result.current.totalItems).toBe(0);
  });

  it('should update item quantity', () => {
    const { result } = renderHook(() => useCart());

    act(() => {
      result.current.addItem({
        productId: 'prd_123',
        quantity: 2,
        price: 99.99,
      });
    });

    act(() => {
      result.current.updateQuantity('prd_123', 5);
    });

    expect(result.current.items[0].quantity).toBe(5);
    expect(result.current.totalItems).toBe(5);
  });
});
```

---

### 2. Component Integration Testing

**Purpose**: Test component interactions

```typescript
// Checkout.integration.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { Checkout } from './Checkout';
import { CartProvider } from '../context/CartContext';
import { mockServer } from '../mocks/server';

beforeAll(() => mockServer.listen());
afterEach(() => mockServer.resetHandlers());
afterAll(() => mockServer.close());

describe('Checkout Integration', () => {
  it('should complete checkout flow', async () => {
    render(
      <CartProvider>
        <Checkout />
      </CartProvider>
    );

    // Step 1: Shipping Address
    fireEvent.change(screen.getByLabelText(/address/i), {
      target: { value: '123 Main St' },
    });
    fireEvent.change(screen.getByLabelText(/city/i), {
      target: { value: 'New York' },
    });
    fireEvent.click(screen.getByRole('button', { name: /continue/i }));

    // Step 2: Payment
    await waitFor(() => {
      expect(screen.getByText(/payment method/i)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText(/credit card/i));
    fireEvent.change(screen.getByLabelText(/card number/i), {
      target: { value: '4242424242424242' },
    });

    fireEvent.click(screen.getByRole('button', { name: /place order/i }));

    // Step 3: Confirmation
    await waitFor(() => {
      expect(screen.getByText(/order confirmed/i)).toBeInTheDocument();
    });
  });
});
```

---

### 3. End-to-End Testing

**Purpose**: Test complete user flows

**Framework**: Playwright, Cypress

#### Playwright E2E Test

```typescript
// e2e/checkout.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Checkout Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('https://example.com');
  });

  test('should complete full checkout as guest', async ({ page }) => {
    // Search for product
    await page.fill('[data-testid="search-input"]', 'headphones');
    await page.press('[data-testid="search-input"]', 'Enter');

    // Select product
    await page.click('[data-testid="product-card"]:first-child');

    // Add to cart
    await page.click('[data-testid="add-to-cart"]');
    await expect(page.locator('[data-testid="cart-count"]')).toHaveText('1');

    // Go to cart
    await page.click('[data-testid="cart-icon"]');
    await expect(page).toHaveURL(/.*\/cart/);

    // Proceed to checkout
    await page.click('[data-testid="checkout-button"]');

    // Fill shipping information
    await page.fill('[name="email"]', 'test@example.com');
    await page.fill('[name="firstName"]', 'John');
    await page.fill('[name="lastName"]', 'Doe');
    await page.fill('[name="address"]', '123 Main St');
    await page.fill('[name="city"]', 'New York');
    await page.selectOption('[name="state"]', 'NY');
    await page.fill('[name="zipCode"]', '10001');
    await page.fill('[name="phone"]', '1234567890');

    await page.click('[data-testid="continue-to-payment"]');

    // Fill payment information
    await page.fill('[name="cardNumber"]', '4242424242424242');
    await page.fill('[name="expiry"]', '12/25');
    await page.fill('[name="cvv"]', '123');

    // Place order
    await page.click('[data-testid="place-order"]');

    // Verify order confirmation
    await expect(page).toHaveURL(/.*\/order-confirmation/);
    await expect(
      page.locator('[data-testid="confirmation-message"]')
    ).toContainText('Order Confirmed');
  });

  test('should show error for invalid payment', async ({ page }) => {
    // ... navigate to payment step ...

    // Use invalid card
    await page.fill('[name="cardNumber"]', '4000000000000002');
    await page.click('[data-testid="place-order"]');

    // Verify error message
    await expect(
      page.locator('[data-testid="payment-error"]')
    ).toBeVisible();
  });
});
```

#### Cypress E2E Test

```typescript
// cypress/e2e/product-search.cy.ts
describe('Product Search', () => {
  beforeEach(() => {
    cy.visit('/');
  });

  it('should search for products and filter results', () => {
    // Search
    cy.get('[data-testid="search-input"]').type('laptop{enter}');

    // Wait for results
    cy.get('[data-testid="product-card"]').should('have.length.gt', 0);

    // Apply price filter
    cy.get('[data-testid="price-filter-min"]').type('500');
    cy.get('[data-testid="price-filter-max"]').type('1500');
    cy.get('[data-testid="apply-filters"]').click();

    // Verify filtered results
    cy.get('[data-testid="product-price"]').each(($price) => {
      const price = parseFloat($price.text().replace('$', ''));
      expect(price).to.be.within(500, 1500);
    });

    // Sort by price
    cy.get('[data-testid="sort-select"]').select('price-asc');

    // Verify sorting
    let previousPrice = 0;
    cy.get('[data-testid="product-price"]').each(($price) => {
      const price = parseFloat($price.text().replace('$', ''));
      expect(price).to.be.gte(previousPrice);
      previousPrice = price;
    });
  });
});
```

---

### 4. Visual Regression Testing

**Purpose**: Detect unintended visual changes

**Framework**: Percy, Chromatic

```typescript
// visual-regression.test.ts
import { test } from '@playwright/test';
import percySnapshot from '@percy/playwright';

test.describe('Visual Regression Tests', () => {
  test('product page', async ({ page }) => {
    await page.goto('/products/prd_123');
    await percySnapshot(page, 'Product Page');
  });

  test('cart page - empty', async ({ page }) => {
    await page.goto('/cart');
    await percySnapshot(page, 'Empty Cart');
  });

  test('cart page - with items', async ({ page }) => {
    await page.goto('/cart');
    // Add items programmatically
    await page.evaluate(() => {
      window.localStorage.setItem('cart', JSON.stringify([
        { productId: 'prd_123', quantity: 2 }
      ]));
    });
    await page.reload();
    await percySnapshot(page, 'Cart With Items');
  });
});
```

---

### 5. Accessibility Testing

**Purpose**: Ensure WCAG 2.1 AA compliance

**Framework**: jest-axe, Lighthouse CI

```typescript
// accessibility.test.tsx
import { render } from '@testing-library/react';
import { axe, toHaveNoViolations } from 'jest-axe';
import { ProductPage } from './ProductPage';

expect.extend(toHaveNoViolations);

describe('Accessibility Tests', () => {
  it('should have no accessibility violations on product page', async () => {
    const { container } = render(<ProductPage />);
    const results = await axe(container);
    expect(results).toHaveNoViolations();
  });

  it('should have proper ARIA labels', () => {
    const { getByRole } = render(<ProductPage />);

    expect(getByRole('button', { name: /add to cart/i })).toBeInTheDocument();
    expect(getByRole('img', { name: /product image/i })).toBeInTheDocument();
  });
});
```

---

### 6. Performance Testing

**Purpose**: Measure frontend performance metrics

**Framework**: Lighthouse CI, Web Vitals

```javascript
// lighthouse-ci.config.js
module.exports = {
  ci: {
    collect: {
      numberOfRuns: 3,
      url: [
        'http://localhost:3000/',
        'http://localhost:3000/products',
        'http://localhost:3000/cart',
      ],
    },
    assert: {
      assertions: {
        'first-contentful-paint': ['error', { maxNumericValue: 2000 }],
        'largest-contentful-paint': ['error', { maxNumericValue: 2500 }],
        'cumulative-layout-shift': ['error', { maxNumericValue: 0.1 }],
        'total-blocking-time': ['error', { maxNumericValue: 300 }],
        'speed-index': ['error', { maxNumericValue: 3000 }],
        'interactive': ['error', { maxNumericValue: 3500 }],
      },
    },
  },
};
```

---

## Test Data Management

### Test Data Builders

```typescript
// test-builders.ts
export class UserBuilder {
  private user: any = {
    email: 'test@example.com',
    firstName: 'John',
    lastName: 'Doe',
    password: 'password123',
  };

  withEmail(email: string) {
    this.user.email = email;
    return this;
  }

  withName(firstName: string, lastName: string) {
    this.user.firstName = firstName;
    this.user.lastName = lastName;
    return this;
  }

  build() {
    return this.user;
  }
}

export class OrderBuilder {
  private order: any = {
    userId: 'usr_123',
    items: [],
    shippingAddress: {},
  };

  withUser(userId: string) {
    this.order.userId = userId;
    return this;
  }

  withItems(items: any[]) {
    this.order.items = items;
    return this;
  }

  build() {
    return this.order;
  }
}

// Usage
const user = new UserBuilder()
  .withEmail('custom@example.com')
  .withName('Jane', 'Smith')
  .build();
```

### Fixtures

```typescript
// fixtures/products.fixture.ts
export const productsFixture = [
  {
    id: 'prd_123',
    name: 'Wireless Headphones',
    price: 249.99,
    category: 'Electronics',
    inStock: true,
  },
  {
    id: 'prd_456',
    name: 'Bluetooth Speaker',
    price: 89.99,
    category: 'Electronics',
    inStock: true,
  },
];
```

---

## CI/CD Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Test Suite

on: [push, pull_request]

jobs:
  backend-tests:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Run unit tests
        run: npm run test:unit

      - name: Run integration tests
        run: npm run test:integration

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage/coverage-final.json

  frontend-tests:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci
        working-directory: ./frontend

      - name: Run unit tests
        run: npm run test:unit
        working-directory: ./frontend

      - name: Run E2E tests
        run: npm run test:e2e
        working-directory: ./frontend

      - name: Run Lighthouse CI
        run: npm run lighthouse:ci
        working-directory: ./frontend

  contract-tests:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Run Pact tests
        run: npm run test:contract

      - name: Publish Pacts
        run: npm run pact:publish
```

---

## Test Coverage Requirements

| Test Type | Coverage Target | Notes |
|-----------|----------------|-------|
| Backend Unit | > 80% | Critical business logic: > 90% |
| Frontend Unit | > 75% | UI components: > 70% |
| Integration | Critical flows | Payment, Order, Auth |
| E2E | Happy paths + edge cases | Major user journeys |
| API Contract | All inter-service APIs | Mandatory |

---

## Testing Best Practices

### 1. Test Naming Convention

```typescript
describe('[Component/Service Name]', () => {
  describe('[Method/Function Name]', () => {
    it('should [expected behavior] when [condition]', () => {
      // Test implementation
    });
  });
});
```

### 2. AAA Pattern (Arrange-Act-Assert)

```typescript
it('should add item to cart', () => {
  // Arrange
  const cart = new Cart();
  const item = { id: '123', price: 99.99 };

  // Act
  cart.addItem(item);

  // Assert
  expect(cart.items).toHaveLength(1);
  expect(cart.total).toBe(99.99);
});
```

### 3. Test Isolation

- Each test should be independent
- Use `beforeEach` to reset state
- Avoid shared mutable state

### 4. Mock External Dependencies

- Mock HTTP calls
- Mock database queries
- Mock third-party services
- Use dependency injection for testability

### 5. Test Data

- Use realistic test data
- Avoid magic numbers
- Use test data builders for complex objects

---

## Monitoring Test Health

### Metrics to Track

- **Test Coverage**: Overall and per-service
- **Test Execution Time**: Identify slow tests
- **Flaky Tests**: Tests that fail intermittently
- **Test Failure Rate**: Track trends

### Tools

- **SonarQube**: Code quality and coverage
- **Codecov**: Coverage visualization
- **TestRail**: Test management
- **Allure**: Test reporting

---

## Conclusion

A comprehensive testing strategy ensures:
- High code quality
- Confidence in deployments
- Early bug detection
- Better documentation through tests
- Faster development cycles

Regular review and updates to the testing strategy ensure it remains effective as the system evolves.
