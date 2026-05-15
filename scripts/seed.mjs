#!/usr/bin/env node
// Seed demo data across tenant / user / product / inventory / order services.
// Usage: node scripts/seed.mjs
// Env overrides: API_BASE=http://localhost, TENANT_NAME="Demo Store"

const API_BASE = process.env.API_BASE || 'http://localhost';
const PORTS = {
  tenant: Number(process.env.TENANT_PORT) || 8081,
  user: Number(process.env.USER_PORT) || 8082,
  product: Number(process.env.PRODUCT_PORT) || 8083,
  inventory: Number(process.env.INVENTORY_PORT) || 8084,
  order: Number(process.env.ORDER_PORT) || 8096,
};

const TENANT_NAME = process.env.TENANT_NAME || 'Demo Store';
const TENANT_EMAIL = process.env.TENANT_EMAIL || 'owner@demostore.test';
const TENANT_TIER = process.env.TENANT_TIER || 'starter';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || 'Password123!';

const log = (...a) => console.log('[seed]', ...a);
const warn = (...a) => console.warn('[seed:warn]', ...a);
const die = (msg, err) => { console.error('[seed:fatal]', msg, err?.message || err || ''); process.exit(1); };

async function http(service, path, { method = 'GET', body, tenantId, token } = {}) {
  const url = `${API_BASE}:${PORTS[service]}${path}`;
  const headers = { 'Content-Type': 'application/json' };
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = await fetch(url, { method, headers, body: body ? JSON.stringify(body) : undefined });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = text; }
  if (!res.ok) {
    const err = new Error(`${method} ${url} -> ${res.status}: ${typeof data === 'string' ? data : JSON.stringify(data)}`);
    err.status = res.status; err.data = data;
    throw err;
  }
  return data;
}

async function waitFor(service, path = '/health', attempts = 30, delayMs = 2000) {
  for (let i = 1; i <= attempts; i++) {
    try {
      await http(service, path);
      log(`${service} is up`);
      return true;
    } catch (e) {
      if (i === attempts) { warn(`${service} not responding after ${attempts} attempts:`, e.message); return false; }
      await new Promise(r => setTimeout(r, delayMs));
    }
  }
}

// ----- seed steps -----

async function createTenant() {
  // Reuse explicit tenant id if provided
  if (process.env.TENANT_ID) {
    log(`reusing tenant id=${process.env.TENANT_ID}`);
    return await http('tenant', `/api/v1/tenants/${process.env.TENANT_ID}`);
  }
  // Otherwise look up any existing tenant with the same email (the service auto-suffixes slugs,
  // so creating on every run yields duplicates — match by email instead).
  try {
    const list = await http('tenant', '/api/v1/tenants?page=1&page_size=100');
    const match = (list.data || []).find(t => t.email === TENANT_EMAIL);
    if (match) {
      log(`reusing existing tenant "${match.slug}" (matched by email)`);
      if (match.status !== 'active') {
        await http('tenant', `/api/v1/tenants/${match.id}`, {
          method: 'PUT', body: { status: 'active' },
        }).catch(() => {});
      }
      return match;
    }
  } catch {}

  log(`creating tenant "${TENANT_NAME}"...`);
  const tenant = await http('tenant', '/api/v1/tenants', {
    method: 'POST',
    body: { name: TENANT_NAME, email: TENANT_EMAIL, tier: TENANT_TIER },
  });
  if (tenant.status !== 'active') {
    log(`activating tenant ${tenant.id}`);
    await http('tenant', `/api/v1/tenants/${tenant.id}`, {
      method: 'PUT',
      body: { status: 'active' },
    }).catch(e => warn('failed to activate tenant:', e.message));
  }
  return tenant;
}

async function registerUser(tenantId, userData) {
  try {
    return await http('user', '/api/v1/auth/register', {
      method: 'POST',
      body: { tenant_id: tenantId, ...userData },
      tenantId,
    });
  } catch (e) {
    if (e.status === 409 || /already|duplicate/i.test(e.message)) {
      log(`user ${userData.email} already exists`);
      return null;
    }
    warn(`register ${userData.email} failed:`, e.message);
    return null;
  }
}

async function login(tenantId, email, password) {
  const res = await http('user', '/api/v1/auth/login', {
    method: 'POST',
    body: { tenant_id: tenantId, email, password },
    tenantId,
  });
  const payload = res.data || res;
  return payload.token;
}

async function createCategory(tenantId, data, createdBy, token) {
  try {
    const res = await http('product', '/api/v1/categories', {
      method: 'POST',
      body: { tenant_id: tenantId, created_by: createdBy, ...data },
      tenantId, token,
    });
    return res.data || res;
  } catch (e) {
    if (/already|duplicate|slug/i.test(e.message)) {
      log(`category ${data.slug} already exists, looking up`);
      try {
        const list = await http('product', `/api/v1/categories?tenant_id=${tenantId}`, { tenantId, token });
        const existing = (list.data || []).find(c => c.slug === data.slug);
        if (existing) return existing;
      } catch {}
    }
    throw e;
  }
}

async function createProduct(tenantId, data, createdBy, token) {
  try {
    const res = await http('product', '/api/v1/products', {
      method: 'POST',
      body: { tenant_id: tenantId, created_by: createdBy, status: 'active', ...data },
      tenantId, token,
    });
    return res.data || res;
  } catch (e) {
    if (/already|duplicate|sku/i.test(e.message)) {
      log(`product sku ${data.sku} exists, skipping`);
      return null;
    }
    throw e;
  }
}

async function createWarehouse(tenantId, data, createdBy, token) {
  try {
    return await http('inventory', '/api/v1/Inventory/warehouses', {
      method: 'POST',
      body: { tenantId, createdBy, ...data },
      tenantId, token,
    });
  } catch (e) {
    if (/already|duplicate|code/i.test(e.message)) return null;
    throw e;
  }
}

async function createInventoryItem(tenantId, warehouseId, data, createdBy, token) {
  try {
    return await http('inventory', '/api/v1/Inventory/items', {
      method: 'POST',
      body: { tenantId, warehouseId, createdBy, ...data },
      tenantId, token,
    });
  } catch (e) {
    warn(`inventory item for ${data.sku} failed:`, e.message);
    return null;
  }
}

async function createOrder(tenantId, customerId, token, { items }) {
  const addr = {
    street: '123 Demo St', city: 'Dhaka', state: 'Dhaka',
    postal_code: '1209', country: 'BD',
  };
  const res = await http('order', '/api/v1/orders', {
    method: 'POST',
    body: { tenant_id: tenantId, customer_id: customerId, shipping_address: addr, billing_address: addr },
    tenantId, token,
  });
  const order = res?.data || res;
  const orderId = order?.id || order?.order_id || order?.orderId;
  if (!orderId) { warn('order create returned no id:', JSON.stringify(res).slice(0, 200)); return order; }
  for (const it of items) {
    await http('order', `/api/v1/orders/${orderId}/items`, {
      method: 'POST', body: it, tenantId, token,
    }).catch(e => warn(`add item ${it.sku}:`, e.message));
  }
  return { ...order, id: orderId };
}

// ----- catalog data -----

const CATEGORIES = [
  { name: 'Electronics',   slug: 'electronics',   description: 'Phones, laptops, and accessories' },
  { name: 'Apparel',       slug: 'apparel',       description: 'Clothing and accessories' },
  { name: 'Home & Garden', slug: 'home-garden',   description: 'Household items' },
  { name: 'Books',         slug: 'books',         description: 'Fiction, non-fiction, textbooks' },
  { name: 'Sports & Outdoors', slug: 'sports-outdoors', description: 'Gear for active lifestyles' },
  { name: 'Beauty & Personal Care', slug: 'beauty', description: 'Skincare, makeup, grooming' },
  { name: 'Toys & Games',  slug: 'toys-games',    description: 'Kids toys and board games' },
  { name: 'Kitchen',       slug: 'kitchen',       description: 'Cookware and appliances' },
];

const PRODUCTS = (catIdx) => [
  // Electronics
  { sku: 'PHONE-001', name: 'Acme Phone X1', slug: 'acme-phone-x1', price: 69999, compare_at_price: 79999, category_id: catIdx.electronics, description: 'Flagship smartphone', brand: 'Acme', tags: ['phone','flagship'] },
  { sku: 'PHONE-002', name: 'Acme Phone Lite', slug: 'acme-phone-lite', price: 24999, category_id: catIdx.electronics, description: 'Affordable smartphone', brand: 'Acme', tags: ['phone','budget'] },
  { sku: 'LAPTOP-001', name: 'Acme Laptop Pro 14', slug: 'acme-laptop-pro-14', price: 129999, category_id: catIdx.electronics, description: '14-inch productivity laptop', brand: 'Acme', tags: ['laptop'] },
  { sku: 'LAPTOP-002', name: 'Acme Gaming Laptop 16', slug: 'acme-gaming-laptop-16', price: 189999, category_id: catIdx.electronics, description: '16-inch gaming laptop with RTX GPU', brand: 'Acme', tags: ['laptop','gaming'] },
  { sku: 'HEADPH-001', name: 'Acme Noise-Cancelling Headphones', slug: 'acme-nc-headphones', price: 12999, category_id: catIdx.electronics, description: 'Wireless over-ear headphones', brand: 'Acme', tags: ['audio'] },
  { sku: 'EARBUD-001', name: 'Acme Wireless Earbuds', slug: 'acme-earbuds', price: 4999, category_id: catIdx.electronics, description: 'True wireless earbuds with case', brand: 'Acme', tags: ['audio'] },
  { sku: 'WATCH-001', name: 'Acme SmartWatch 3', slug: 'acme-smartwatch-3', price: 19999, category_id: catIdx.electronics, description: 'Fitness and notifications on your wrist', brand: 'Acme', tags: ['wearable'] },
  { sku: 'TABLET-001', name: 'Acme Tablet 11', slug: 'acme-tablet-11', price: 34999, category_id: catIdx.electronics, description: '11-inch tablet with stylus support', brand: 'Acme', tags: ['tablet'] },

  // Apparel
  { sku: 'TSHIRT-BLK-M', name: 'Classic Cotton T-Shirt (Black, M)', slug: 'classic-cotton-tshirt-black-m', price: 899, category_id: catIdx.apparel, description: '100% cotton tee', brand: 'BasicWear', tags: ['tshirt','cotton'] },
  { sku: 'TSHIRT-WHT-L', name: 'Classic Cotton T-Shirt (White, L)', slug: 'classic-cotton-tshirt-white-l', price: 899, category_id: catIdx.apparel, description: '100% cotton tee', brand: 'BasicWear', tags: ['tshirt','cotton'] },
  { sku: 'JEANS-001', name: 'Slim Fit Jeans', slug: 'slim-fit-jeans', price: 2499, category_id: catIdx.apparel, description: 'Stretch denim', brand: 'BasicWear', tags: ['jeans'] },
  { sku: 'JEANS-002', name: 'Relaxed Fit Jeans', slug: 'relaxed-fit-jeans', price: 2299, category_id: catIdx.apparel, description: 'Comfortable everyday jeans', brand: 'BasicWear', tags: ['jeans'] },
  { sku: 'HOODY-001', name: 'Zip-Up Hoodie', slug: 'zip-up-hoodie', price: 1999, category_id: catIdx.apparel, description: 'Soft fleece hoodie', brand: 'BasicWear', tags: ['hoodie'] },
  { sku: 'JACKET-001', name: 'Waterproof Shell Jacket', slug: 'waterproof-shell-jacket', price: 5999, category_id: catIdx.apparel, description: 'Lightweight rain jacket', brand: 'BasicWear', tags: ['outerwear'] },
  { sku: 'SHOE-001', name: 'Urban Sneakers', slug: 'urban-sneakers', price: 3499, category_id: catIdx.apparel, description: 'Everyday street sneakers', brand: 'StepUp', tags: ['shoes'] },
  { sku: 'CAP-001', name: 'Six-Panel Cap', slug: 'six-panel-cap', price: 799, category_id: catIdx.apparel, description: 'Adjustable cotton cap', brand: 'BasicWear', tags: ['accessory'] },

  // Home & Garden
  { sku: 'LAMP-001', name: 'LED Desk Lamp', slug: 'led-desk-lamp', price: 1499, category_id: catIdx['home-garden'], description: 'Dimmable desk lamp', brand: 'HomeGlow', tags: ['lamp'] },
  { sku: 'PLANT-001', name: 'Snake Plant (Medium)', slug: 'snake-plant-medium', price: 699, category_id: catIdx['home-garden'], description: 'Low-maintenance indoor plant', brand: 'GreenThumb', tags: ['plant'] },
  { sku: 'PLANT-002', name: 'Monstera Deliciosa', slug: 'monstera-deliciosa', price: 1299, category_id: catIdx['home-garden'], description: 'Statement indoor plant', brand: 'GreenThumb', tags: ['plant'] },
  { sku: 'CUSHION-001', name: 'Linen Throw Cushion', slug: 'linen-throw-cushion', price: 899, category_id: catIdx['home-garden'], description: 'Soft natural linen cover', brand: 'HomeGlow', tags: ['decor'] },
  { sku: 'RUG-001', name: 'Handwoven Area Rug 5x7', slug: 'area-rug-5x7', price: 4499, category_id: catIdx['home-garden'], description: 'Textured jute rug', brand: 'HomeGlow', tags: ['decor'] },

  // Kitchen
  { sku: 'MUG-001', name: 'Ceramic Coffee Mug Set of 2', slug: 'ceramic-mug-set', price: 499, category_id: catIdx.kitchen, description: 'Microwave safe mugs', brand: 'HomeGlow', tags: ['kitchen'] },
  { sku: 'PAN-001', name: 'Cast Iron Skillet 10"', slug: 'cast-iron-skillet-10', price: 1999, category_id: catIdx.kitchen, description: 'Pre-seasoned cast iron', brand: 'CookPro', tags: ['cookware'] },
  { sku: 'KNIFE-001', name: 'Chef Knife 8"', slug: 'chef-knife-8', price: 2499, category_id: catIdx.kitchen, description: 'German stainless steel', brand: 'CookPro', tags: ['cookware'] },
  { sku: 'BOTTLE-001', name: 'Insulated Water Bottle 750ml', slug: 'insulated-bottle-750', price: 1199, category_id: catIdx.kitchen, description: '24hr cold, 12hr hot', brand: 'HydraFlow', tags: ['drinkware'] },

  // Books
  { sku: 'BOOK-001', name: 'The Pragmatic Programmer', slug: 'pragmatic-programmer', price: 1899, category_id: catIdx.books, description: 'Classic software craft book', brand: 'Addison-Wesley', tags: ['programming'] },
  { sku: 'BOOK-002', name: 'Designing Data-Intensive Applications', slug: 'ddia', price: 2299, category_id: catIdx.books, description: 'Martin Kleppmann', brand: 'O\'Reilly', tags: ['programming'] },
  { sku: 'BOOK-003', name: 'Atomic Habits', slug: 'atomic-habits', price: 1299, category_id: catIdx.books, description: 'James Clear', brand: 'Penguin', tags: ['self-help'] },

  // Sports
  { sku: 'YOGA-001', name: 'Non-Slip Yoga Mat', slug: 'yoga-mat', price: 1499, category_id: catIdx['sports-outdoors'], description: '6mm TPE yoga mat', brand: 'FlexFit', tags: ['yoga'] },
  { sku: 'DUMBBELL-001', name: 'Adjustable Dumbbell 20kg', slug: 'adjustable-dumbbell-20', price: 5499, category_id: catIdx['sports-outdoors'], description: 'Quick-change weight plates', brand: 'FlexFit', tags: ['strength'] },
  { sku: 'BIKE-001', name: 'City Commuter Bike', slug: 'city-commuter-bike', price: 18999, category_id: catIdx['sports-outdoors'], description: '7-speed commuter bicycle', brand: 'Velocity', tags: ['cycling'] },

  // Beauty
  { sku: 'CREAM-001', name: 'Hyaluronic Acid Moisturizer', slug: 'ha-moisturizer', price: 899, category_id: catIdx.beauty, description: 'Daily hydrating face cream', brand: 'PureGlow', tags: ['skincare'] },
  { sku: 'SERUM-001', name: 'Vitamin C Brightening Serum', slug: 'vitc-serum', price: 1299, category_id: catIdx.beauty, description: '15% Vitamin C, antioxidant', brand: 'PureGlow', tags: ['skincare'] },
  { sku: 'SHAMPOO-001', name: 'Argan Oil Shampoo 500ml', slug: 'argan-shampoo', price: 699, category_id: catIdx.beauty, description: 'Sulfate-free shampoo', brand: 'PureGlow', tags: ['haircare'] },

  // Toys
  { sku: 'LEGO-001', name: 'Building Blocks Starter Set', slug: 'building-blocks-starter', price: 1799, category_id: catIdx['toys-games'], description: '500-piece creative set', brand: 'BrickLab', tags: ['toys'] },
  { sku: 'PUZZLE-001', name: '1000-Piece World Map Puzzle', slug: 'world-map-puzzle', price: 999, category_id: catIdx['toys-games'], description: 'Educational jigsaw puzzle', brand: 'PuzzleCo', tags: ['puzzles'] },
  { sku: 'BOARD-001', name: 'Strategy Board Game: Settlers', slug: 'settlers-board-game', price: 2499, category_id: catIdx['toys-games'], description: 'Classic strategy game', brand: 'BoardCraft', tags: ['board-games'] },
];

// ----- main -----

(async () => {
  log(`API_BASE=${API_BASE}`);

  const tenantUp = await waitFor('tenant');
  const userUp = await waitFor('user');
  const productUp = await waitFor('product');
  const inventoryUp = await waitFor('inventory');
  const orderUp = await waitFor('order');

  if (!tenantUp || !userUp || !productUp) die('core services (tenant/user/product) not reachable');

  const tenant = await createTenant();
  log(`tenant: id=${tenant.id} slug=${tenant.slug}`);

  // Users — emails/usernames must be globally unique (service-level constraint),
  // so derive per-tenant identifiers from the tenant slug.
  const suffix = (tenant.slug || tenant.id || Date.now().toString()).replace(/[^a-z0-9]/gi, '').slice(0, 12);
  const adminData = { email: `admin+${suffix}@demostore.test`, username: `admin_${suffix}`, password: ADMIN_PASSWORD, first_name: 'Demo', last_name: 'Admin' };
  const c1Data    = { email: `alice+${suffix}@demostore.test`, username: `alice_${suffix}`, password: ADMIN_PASSWORD, first_name: 'Alice', last_name: 'Anderson' };
  const c2Data    = { email: `bob+${suffix}@demostore.test`,   username: `bob_${suffix}`,   password: ADMIN_PASSWORD, first_name: 'Bob',   last_name: 'Brown' };
  await registerUser(tenant.id, adminData);
  await registerUser(tenant.id, c1Data);
  await registerUser(tenant.id, c2Data);

  // Promote the seed admin user to role='admin' via direct DB update
  // (UpdateUserRole API is admin-only, so we bootstrap via SQL).
  try {
    const { execSync } = await import('node:child_process');
    execSync(
      `docker exec ecommerce-postgres psql -U postgres -d user_db -c "UPDATE users SET role='admin' WHERE email='${adminData.email}';"`,
      { stdio: 'pipe' }
    );
    log(`promoted ${adminData.email} to admin role`);
  } catch (e) {
    warn('failed to promote admin via DB:', e.message);
  }

  // Login as admin to get a token (for order service which requires JWT)
  let adminToken = null, adminUserId = null;
  try {
    adminToken = await login(tenant.id, adminData.email, adminData.password);
    // JWT payload decode to get user id (optional, best-effort)
    const payload = JSON.parse(Buffer.from(adminToken.split('.')[1], 'base64').toString('utf8'));
    adminUserId = payload.user_id || payload.sub || payload.uid || null;
    log(`admin logged in, user_id=${adminUserId}`);
  } catch (e) {
    warn('admin login failed; order seeding will be skipped:', e.message);
  }

  let aliceToken = null, aliceUserId = null;
  try {
    aliceToken = await login(tenant.id, c1Data.email, c1Data.password);
    const payload = JSON.parse(Buffer.from(aliceToken.split('.')[1], 'base64').toString('utf8'));
    aliceUserId = payload.user_id || payload.sub || payload.uid || null;
  } catch (e) { warn('alice login failed:', e.message); }

  // Categories
  const catIdx = {};
  for (const c of CATEGORIES) {
    const created = await createCategory(tenant.id, c, adminUserId || tenant.id, adminToken);
    if (created?.id) catIdx[c.slug] = created.id;
  }
  log(`categories: ${Object.keys(catIdx).length} ready`);

  // Products
  const products = [];
  for (const p of PRODUCTS(catIdx)) {
    if (!p.category_id) { warn(`skipping ${p.sku}: missing category`); continue; }
    const created = await createProduct(tenant.id, p, adminUserId || tenant.id, adminToken);
    if (created?.id) products.push(created);
  }
  log(`products: ${products.length} created`);

  // Inventory (optional — .NET service; skip if not reachable)
  let warehouse = null;
  if (inventoryUp) {
    warehouse = await createWarehouse(tenant.id, {
      code: 'DHK-01', name: 'Dhaka Main Warehouse',
      address: '1 Warehouse Rd', city: 'Dhaka', state: 'Dhaka',
      country: 'BD', postalCode: '1209', isActive: true, isDefault: true,
    }, adminUserId || tenant.id, adminToken);
    if (warehouse?.id) {
      log(`warehouse: ${warehouse.id}`);
      for (const p of products) {
        await createInventoryItem(tenant.id, warehouse.id, {
          productId: p.id, sku: p.sku, initialQuantity: 100, reorderPoint: 10, reorderQuantity: 50,
        }, adminUserId || tenant.id, adminToken);
      }
      log(`inventory items seeded for ${products.length} products`);
    }
  } else {
    warn('inventory service not up; skipping warehouse/stock');
  }

  // Orders (optional — needs JWT)
  if (orderUp && adminToken && aliceToken && aliceUserId && products.length >= 2) {
    try {
      const o1 = await createOrder(tenant.id, aliceUserId, aliceToken, {
        items: [
          { product_id: products[0].id, sku: products[0].sku, name: products[0].name, quantity: 1, unit_price: products[0].price },
          { product_id: products[2].id, sku: products[2].sku, name: products[2].name, quantity: 2, unit_price: products[2].price },
        ],
      });
      log(`order: ${o1.id}`);
    } catch (e) { warn('seed order failed:', e.message); }
  } else {
    warn('skipping order seed (service / token / product count insufficient)');
  }

  log('--- summary ---');
  console.log(JSON.stringify({
    tenant: { id: tenant.id, slug: tenant.slug, email: TENANT_EMAIL },
    admin: { email: adminData.email, password: ADMIN_PASSWORD, user_id: adminUserId },
    customers: [c1Data.email, c2Data.email],
    categories: catIdx,
    product_count: products.length,
    warehouse_id: warehouse?.id || null,
  }, null, 2));
})().catch(e => die('unhandled error', e));
