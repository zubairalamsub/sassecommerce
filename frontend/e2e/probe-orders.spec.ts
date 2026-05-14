import { test } from '@playwright/test';

test.setTimeout(120_000);

test('probe orders network for 30s', async ({ page }) => {
  let navStartRef = Date.now();
  const apiRequests: { url: string; t: number; initiator: string }[] = [];
  page.on('console', (msg) => {
    if (msg.text().startsWith('[probe]') || msg.text().startsWith('[saajan]')) console.log(msg.text());
  });
  page.on('request', (req) => {
    const u = req.url();
    // Filter to API calls only (proxy)
    if (u.includes('/proxy/') || u.includes('/api/')) {
      apiRequests.push({ url: u, t: Date.now() - navStartRef, initiator: '' });
    }
  });

  // Login
  await page.goto('/login');
  await page.waitForLoadState('domcontentloaded');
  await page.getByLabel('Email').fill('rahim@example.com');
  await page.getByLabel('Password').fill('customer123');
  await page.getByRole('button', { name: /Sign in/i }).click();
  await page.waitForURL(/\/products/, { timeout: 20_000 });

  apiRequests.length = 0;
  navStartRef = Date.now();
  await page.goto('/account/orders');

  await page.waitForTimeout(15_000);
  const elapsed = Date.now() - navStartRef;
  console.log('\n===== API REQUESTS within ' + elapsed + 'ms after /account/orders nav =====');
  for (const r of apiRequests) {
    console.log('  +' + r.t + 'ms  ' + r.url);
  }
  console.log('===== TOTAL: ' + apiRequests.length + ' =====');
});
