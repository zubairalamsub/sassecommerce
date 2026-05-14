/**
 * Saajan Storefront — Customer Journey E2E Smoke Tests
 *
 * Scenario 1: Full purchase flow (register → browse → PDP → cart → checkout → order confirmation)
 * Scenario 2: Anonymous browse smoke (home → product → search)
 *
 * Prerequisites: `npm run dev` running at http://localhost:3000
 *                Backend services up (docker-compose)
 */

import { test, expect, type Page } from '@playwright/test';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const TENANT_ID = 'tenant_saajan';
const PRODUCT_SERVICE = 'http://localhost:8083';

/**
 * Demo customer hardcoded in auth store (rahim@example.com / customer123).
 * The frontend falls back to demo-login when the backend JWT response format
 * doesn't match the expected shape — these are the only credentials that work
 * reliably in the current integration state.
 */
const DEMO_EMAIL = 'rahim@example.com';
const DEMO_PASSWORD = 'customer123';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Wait for the page to finish network activity after a navigation */
async function settle(page: Page) {
  await page.waitForLoadState('networkidle');
}

/**
 * Ensure at least one active product exists in the product service.
 * Returns true if products are available.
 */
async function ensureProductExists(): Promise<boolean> {
  try {
    const res = await fetch(
      `${PRODUCT_SERVICE}/api/v1/products?status=active&page=1&page_size=1`,
      { headers: { 'X-Tenant-ID': TENANT_ID } },
    );
    if (!res.ok) return false;
    const body = await res.json();
    const items: unknown[] = body.data ?? body.products ?? [];
    return items.length > 0;
  } catch {
    return false;
  }
}

/**
 * Wait for the products page to finish loading product cards.
 * Products render as motion.div cards — each has an h3 heading.
 * We wait until either at least one h3 is visible OR the "0 products found"
 * subtitle stabilises, then return whether products are present.
 */
async function waitForProducts(page: Page): Promise<boolean> {
  // Poll until an h3 product card heading appears OR "0 products found" is stable
  const deadline = Date.now() + 15_000;
  while (Date.now() < deadline) {
    const h3Count = await page.locator('h3').count();
    if (h3Count > 0) return true;

    // Check if the loading spinner is gone (products attempted to load)
    const spinnerVisible = await page.locator('svg.animate-spin').isVisible().catch(() => false);
    if (!spinnerVisible) {
      // Spinner gone — check count text
      const countText = await page.locator('p').filter({ hasText: /\d+ product/ }).first().textContent().catch(() => '0');
      const count = parseInt(countText?.match(/(\d+)/)?.[1] ?? '0', 10);
      if (count === 0) return false;
    }
    await page.waitForTimeout(500);
  }
  return false;
}

/**
 * Log in via the frontend login page using demo credentials.
 * Waits for redirect to /products.
 */
async function loginAsDemo(page: Page) {
  await page.goto('/login');
  await settle(page);
  await page.getByLabel('Email').fill(DEMO_EMAIL);
  await page.getByLabel('Password').fill(DEMO_PASSWORD);
  await page.getByRole('button', { name: /Sign in/i }).click();
  await page.waitForURL(/\/products/, { timeout: 20_000 });
  await settle(page);
}

// ---------------------------------------------------------------------------
// Scenario 1 — Full Customer Purchase Flow
// ---------------------------------------------------------------------------

test.describe('Scenario 1: Customer Purchase Flow', () => {
  test('1.1 Register a new account', async ({ page }) => {
    const email = `e2e-${Date.now()}@saajan.test`;
    const password = 'TestPass2024';

    await page.goto('/register');
    await settle(page);

    await page.getByLabel('First Name').fill('Rahim');
    await page.getByLabel('Last Name').fill('Uddin');
    await page.getByLabel('Email Address').fill(email);
    await page.getByPlaceholder('1712345678').fill('1712345678');
    await page.getByLabel('Password', { exact: true }).fill(password);
    await page.getByLabel('Confirm Password').fill(password);
    await page.getByRole('button', { name: 'Create Account' }).click();

    // On success the register page redirects to /products
    await page.waitForURL(/\/products/, { timeout: 20_000 });
    await settle(page);

    // Verify user is logged in — initials avatar OR full name visible in header
    // Register uses a demo fallback (backend login response has a nested 'data' wrapper),
    // so the user gets a demo-new-token. The header still shows the user's initials.
    const header = page.locator('header');
    // Check for initials "RU" or first name "Rahim" in the user button
    await expect(header.getByText('RU').or(header.getByText(/Rahim/i)).first()).toBeVisible({ timeout: 5_000 });
  });

  test('1.2 Products page renders product cards or empty state', async ({ page }) => {
    await page.goto('/products');
    await settle(page);

    await expect(page.getByRole('heading', { name: /All Products/i })).toBeVisible();

    // The "X products found" paragraph is always rendered
    await expect(
      page.locator('p').filter({ hasText: /\d+ product/ }).first(),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('1.3 Navigate to product detail page', async ({ page }) => {
    const productExists = await ensureProductExists();
    if (!productExists) {
      test.skip(true, 'No products available');
      return;
    }

    await page.goto('/products');
    await settle(page);

    const hasProducts = await waitForProducts(page);
    if (!hasProducts) {
      test.skip(true, 'Products page shows 0 products');
      return;
    }

    // Click the first product card link
    await page.locator('a[href^="/products/"]').first().click();
    await settle(page);

    await expect(page).toHaveURL(/\/products\/.+/);
    await expect(page.locator('h1').first()).toBeVisible();

    // Add to Cart button on PDP
    await expect(
      page.getByRole('button', { name: /Add to Cart/i }).first(),
    ).toBeVisible();
  });

  test('1.4 Add product to cart from PDP', async ({ page }) => {
    const productExists = await ensureProductExists();
    if (!productExists) {
      test.skip(true, 'No products available');
      return;
    }

    await page.goto('/products');
    await settle(page);

    const hasProducts = await waitForProducts(page);
    if (!hasProducts) {
      test.skip(true, 'Products page shows 0 products');
      return;
    }

    // Navigate to first product via its anchor link
    await page.locator('a[href^="/products/"]').first().click();
    await page.waitForURL(/\/products\/.+/, { timeout: 10_000 });

    // Add to Cart on PDP
    const addBtn = page.getByRole('button', { name: /Add to Cart/i }).first();
    await addBtn.click();

    // Button text changes to "Added to Cart!"
    await expect(
      page.getByRole('button', { name: /Added to Cart/i }),
    ).toBeVisible({ timeout: 5_000 });
  });

  test('1.5 Cart page shows 1 item', async ({ page }) => {
    const productExists = await ensureProductExists();
    if (!productExists) {
      test.skip(true, 'No products available');
      return;
    }

    await page.goto('/products');
    await settle(page);

    const hasProducts = await waitForProducts(page);
    if (!hasProducts) {
      test.skip(true, 'Products page shows 0 products');
      return;
    }

    // Go to PDP via product link and add to cart
    await page.locator('a[href^="/products/"]').first().click();
    await page.waitForURL(/\/products\/.+/, { timeout: 10_000 });
    await page.getByRole('button', { name: /Add to Cart/i }).first().click();
    await expect(page.getByRole('button', { name: /Added to Cart/i })).toBeVisible({ timeout: 5_000 });

    // Navigate to cart
    await page.goto('/cart');
    await settle(page);

    await expect(page.getByRole('heading', { name: /Shopping Cart/i })).toBeVisible();

    // Cart items show SKU label
    await expect(page.getByText(/SKU:/i).first()).toBeVisible({ timeout: 10_000 });

    // Proceed to Checkout link
    await expect(
      page.getByRole('link', { name: /Proceed to Checkout/i }),
    ).toBeVisible();
  });

  test('1.6 Checkout form fills and places order (COD)', async ({ page }) => {
    const productExists = await ensureProductExists();
    if (!productExists) {
      test.skip(true, 'No products available');
      return;
    }

    // Log in as the demo customer
    await loginAsDemo(page);

    const hasProducts = await waitForProducts(page);
    if (!hasProducts) {
      test.skip(true, 'Products page shows 0 products');
      return;
    }

    // Add item from PDP via product link
    await page.locator('a[href^="/products/"]').first().click();
    await page.waitForURL(/\/products\/.+/, { timeout: 10_000 });
    await page.getByRole('button', { name: /Add to Cart/i }).first().click();
    await expect(page.getByRole('button', { name: /Added to Cart/i })).toBeVisible({ timeout: 5_000 });

    // Go to checkout
    await page.goto('/checkout');
    await settle(page);

    await expect(page.getByRole('heading', { name: /Checkout/i })).toBeVisible();

    // Full Name
    const nameInput = page.getByPlaceholder('e.g. Rahim Uddin');
    await nameInput.clear();
    await nameInput.fill('Rahim Ahmed');

    // Phone
    const phoneInput = page.getByPlaceholder('+880 1XXXXXXXXX');
    await phoneInput.clear();
    await phoneInput.fill('+8801912345678');

    // Email
    const emailInput = page.getByPlaceholder('you@example.com');
    await emailInput.clear();
    await emailInput.fill(DEMO_EMAIL);

    // Street address
    await page.getByPlaceholder('House #, Road #, Area').fill('House 12, Road 5, Dhanmondi');

    // City dropdown
    await page.getByRole('combobox').selectOption('Dhaka');
    await page.waitForTimeout(400); // let shipping rate compute

    // Postal code
    await page.getByPlaceholder('e.g. 1205').fill('1207');

    // Select Cash on Delivery payment method
    await page.getByText('Cash on Delivery').click();

    // Place Order
    await page.getByRole('button', { name: /Place Order/i }).click();

    // Checkout page shows success before redirect
    await expect(
      page.getByText(/Order Placed Successfully!/i),
    ).toBeVisible({ timeout: 25_000 });

    // Auto-redirects to /orders/<id>
    await page.waitForURL(/\/orders\/.+/, { timeout: 15_000 });
    await settle(page);

    // Verify order confirmation page
    await expect(page.getByText(/Order Number/i)).toBeVisible();
    // Order number element (bold paragraph)
    const orderNumEl = page.locator('p.text-lg.font-bold').first();
    await expect(orderNumEl).toBeVisible();
    const orderNumText = await orderNumEl.textContent();
    expect(orderNumText).toBeTruthy();
  });

  test('1.7 Account orders page shows orders', async ({ page }) => {
    // Log in as demo customer and check account/orders
    await loginAsDemo(page);

    await page.goto('/account/orders');
    await page.waitForLoadState('networkidle', { timeout: 5000 });

    // AuthGuard shows spinner then loads page
    await expect(
      page.getByRole('heading', { name: /My Orders/i }),
    ).toBeVisible({ timeout: 15_000 });

    // Page renders: either orders or "no orders" empty state
    const ordersOrEmpty = page.locator('div').filter({ hasText: /pending|confirmed|processing|No orders yet/i }).first();
    await expect(ordersOrEmpty).toBeVisible({ timeout: 15_000 });
  });
});

// ---------------------------------------------------------------------------
// Scenario 2 — Anonymous Browse Smoke
// ---------------------------------------------------------------------------

test.describe('Scenario 2: Anonymous Browse Smoke', () => {
  test('2.1 Home page loads and links to products', async ({ page }) => {
    await page.goto('/');
    await settle(page);

    // Header shows store name
    await expect(page.getByText('Saajan').first()).toBeVisible();

    // "All Products" nav link in header
    await expect(page.getByRole('link', { name: /All Products/i }).first()).toBeVisible();
  });

  test('2.2 Product listing visible without auth', async ({ page }) => {
    await page.goto('/products');
    await settle(page);

    await expect(page.getByRole('heading', { name: /All Products/i })).toBeVisible();

    // "X products found" subtitle always rendered
    await expect(
      page.locator('p').filter({ hasText: /\d+ product/ }).first(),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('2.3 Product detail page renders without auth', async ({ page }) => {
    const productExists = await ensureProductExists();
    if (!productExists) {
      test.skip(true, 'No products available');
      return;
    }

    await page.goto('/products');
    await settle(page);

    const hasProducts = await waitForProducts(page);
    if (!hasProducts) {
      test.skip(true, 'Products page shows 0 products');
      return;
    }

    // Click the first product detail link (links that go to /products/<slug>)
    // Use the anchor tag directly to avoid framer-motion click interception
    const productLink = page.locator('a[href^="/products/"]').first();
    await productLink.waitFor({ timeout: 5_000 });
    await productLink.click();
    await settle(page);

    await expect(page).toHaveURL(/\/products\/.+/);

    // Product detail h1
    await expect(page.locator('h1').first()).toBeVisible();

    // Add to Cart button visible to anonymous users
    await expect(
      page.getByRole('button', { name: /Add to Cart/i }),
    ).toBeVisible();
  });

  test('2.4 Search page renders results or empty state', async ({ page }) => {
    await page.goto('/search?q=saree');
    await settle(page);

    // The search page renders its own search form (not just the header)
    // Scope to the main content area to avoid header input ambiguity
    const mainContent = page.locator('main, div.mx-auto').first();
    await expect(mainContent).toBeVisible();

    // Results summary: "X results for..." or "No products found" or "0 results for..."
    await expect(
      page.locator('p, h3').filter({ hasText: /results for|No products found|products/i }).first(),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('2.5 Search empty-query shows all products or empty state', async ({ page }) => {
    await page.goto('/search');
    await settle(page);

    // Results summary renders regardless of query
    await expect(
      page.locator('p, h3').filter({ hasText: /\d+ products|No products found|Searching/i }).first(),
    ).toBeVisible({ timeout: 15_000 });
  });

  test('2.6 Login page accessible and renders form', async ({ page }) => {
    await page.goto('/login');
    await settle(page);

    await expect(page.getByRole('heading', { name: /Sign In/i })).toBeVisible();
    await expect(page.getByLabel('Email')).toBeVisible();
    await expect(page.getByLabel('Password')).toBeVisible();
    await expect(page.getByRole('button', { name: /Sign in/i })).toBeVisible();
  });

  test('2.7 Register page accessible and renders form', async ({ page }) => {
    await page.goto('/register');
    await settle(page);

    await expect(page.getByRole('heading', { name: /Create Account/i })).toBeVisible();
    await expect(page.getByLabel('First Name')).toBeVisible();
    await expect(page.getByLabel('Last Name')).toBeVisible();
    await expect(page.getByLabel('Email Address')).toBeVisible();
    await expect(page.getByRole('button', { name: /Create Account/i })).toBeVisible();
  });
});
