# Saajan E2E Tests (Playwright)

End-to-end smoke tests for the Saajan storefront. They run against a live dev
server and require all backend services to be up.

---

## Prerequisites

1. **Backend services** — start the full stack:
   ```bash
   make infra-up   # postgres, mongo, redis, kafka
   # then start each Go/.NET service, or via docker-compose
   ```
   Services required: tenant (8081), user (8082), product (8083), order (8080),
   cart (8089), review (8088), notification (8087).

2. **Dev server** — in a separate terminal:
   ```bash
   cd frontend
   npm run dev      # starts at http://localhost:3000
   ```
   Wait until you see `Ready in Xs` before running tests.

---

## Running Tests

```bash
# Headless (CI-friendly, terminal output)
npm run e2e

# Headed — watch the browser
npm run e2e:headed

# Interactive UI mode (step through, time-travel)
npm run e2e:ui

# Single spec
npx playwright test e2e/customer-journey.spec.ts --reporter=list

# One specific test by title
npx playwright test -g "1.6 Checkout"
```

---

## Debugging a Failing Test

1. **Run in headed mode** to see what the browser is doing:
   ```bash
   npm run e2e:headed
   ```

2. **Check traces** — on failure, Playwright saves a trace archive to
   `test-results/`. Open it with:
   ```bash
   npx playwright show-trace test-results/<failing-test>/trace.zip
   ```

3. **Pause on failure** — add `await page.pause()` inside the test to drop
   into Playwright Inspector mid-run.

4. **Screenshots** — `screenshot: 'only-on-failure'` is on by default. Check
   `test-results/` after a failed run.

5. **Verbose output**:
   ```bash
   npx playwright test --reporter=line --debug
   ```

---

## Common Pitfalls

### Network races on first page load
Next.js hydrates client components asynchronously. Always use
`await page.waitForLoadState('networkidle')` (aliased as `settle(page)` in
the helper) after `page.goto()` and after form submissions.

### Hydration timing for Zustand stores
The cart and auth stores are persisted to `localStorage` and hydrated on the
client. Components show a spinner until `hydrated = true`. If you see a
loading spinner that never resolves, the store hydration may be hanging — check
that `localStorage` is accessible in headless Chromium (it is by default) and
that no `beforeEach` hook is clearing storage unexpectedly.

### Products page empty on a fresh stack
The tests call `ensureProductExists()` before attempting any cart flow. If the
product service has no seed data and the seed create request fails (e.g.
because the product service is not running), those tests call `test.skip()`.
Fix: start the product service and ensure the admin user `admin@saajan.com` /
`admin123` exists in the user service.

### Auth cookie vs. Zustand JWT
The frontend stores the JWT in `localStorage` via Zustand persist. Playwright
runs each test in a fresh browser context, so there is no shared auth state
between tests in different `test()` blocks. Tests that need an authenticated
user perform a fresh login at the start of the test.

### Phone number validation
The register form strips non-digits and prepends `+880` internally. The phone
input expects raw digits (e.g. `1712345678`). The checkout phone field expects
the full `+8801712345678` string.

### Checkout redirect timing
After a successful order, the checkout page shows a success message for 2 s
then calls `router.push('/orders/<id>')`. Use `page.waitForURL(/\/orders\/.+/,
{ timeout: 15_000 })` to wait for the redirect — do not rely on a fixed sleep.

---

## Test Data

Each run generates a unique email (`e2e-{timestamp}@saajan.test`) so re-runs
never collide on the user service. No cleanup is needed — test accounts and
orders accumulate in the DB and do not affect production data because this runs
against the local stack.
