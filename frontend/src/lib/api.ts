// API calls are proxied through Next.js rewrites (see next.config.ts)
// to avoid CORS issues. Browser calls /proxy/{service}/... which Next.js
// forwards to the actual backend service.
function serviceUrl(service: string): string {
  return `/proxy/${service}`;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public details?: unknown,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
  tenantId?: string;
  token?: string;
}

async function request<T>(
  service: string,
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { body, tenantId, token, headers: extraHeaders, ...fetchOptions } = options;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(tenantId ? { 'X-Tenant-ID': tenantId } : {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(extraHeaders as Record<string, string>),
  };

  const url = `${serviceUrl(service)}${path}`;

  const res = await fetch(url, {
    ...fetchOptions,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: res.statusText }));
    throw new ApiError(res.status, error.message || res.statusText, error);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

// Image Upload (Next.js API route)
// Returns relative paths like "products/abc123.jpg" — use mediaUrl() to build display URLs
export async function uploadImages(files: File[], folder = 'products'): Promise<string[]> {
  const formData = new FormData();
  files.forEach((file) => formData.append('files', file));
  formData.append('folder', folder);
  const res = await fetch('/api/upload', { method: 'POST', body: formData });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(err.error || 'Upload failed');
  }
  const data = await res.json();
  return data.paths as string[];
}

// Tenant Service
export const tenantApi = {
  list: (page = 1, pageSize = 20, tenantId?: string) =>
    request<PaginatedResponse<Tenant>>('tenant', `/api/v1/tenants?page=${page}&page_size=${pageSize}`, { tenantId }),
  get: (id: string, tenantId?: string) =>
    request<Tenant>('tenant', `/api/v1/tenants/${id}`, { tenantId }),
  create: (data: CreateTenantRequest, tenantId?: string) =>
    request<Tenant>('tenant', '/api/v1/tenants', { method: 'POST', body: data, tenantId }),
  update: (id: string, data: Partial<Tenant>, tenantId?: string) =>
    request<Tenant>('tenant', `/api/v1/tenants/${id}`, { method: 'PUT', body: data, tenantId }),
  delete: (id: string, tenantId?: string) =>
    request<void>('tenant', `/api/v1/tenants/${id}`, { method: 'DELETE', tenantId }),
  updateConfig: (id: string, config: TenantConfig) =>
    request<{ message: string }>('tenant', `/api/v1/tenants/${id}/config`, { method: 'PATCH', body: config }),
};

// Config Service
export const configApi = {
  get: (namespace: string, key: string, environment = 'all', tenantId?: string) => {
    const params = new URLSearchParams({ namespace, key, environment });
    if (tenantId) params.set('tenant_id', tenantId);
    return request<ConfigEntry>('config', `/api/v1/config/get?${params}`);
  },
  set: (data: SetConfigRequest) =>
    request<ConfigEntry>('config', '/api/v1/config/set', { method: 'POST', body: data }),
  listByNamespace: async (namespace: string, environment = 'all', tenantId?: string): Promise<ConfigEntry[]> => {
    const params = new URLSearchParams({ environment });
    if (tenantId) params.set('tenant_id', tenantId);
    const res = await request<{ data: ConfigEntry[]; count: number }>('config', `/api/v1/config/namespace/${namespace}?${params}`);
    return res.data || [];
  },
  listNamespaces: async (): Promise<{ namespace: string; count: number }[]> => {
    const res = await request<{ namespaces: { namespace: string; count: number }[] }>('config', '/api/v1/config/namespaces');
    return res.namespaces || [];
  },
  bulkSet: (entries: SetConfigRequest[]) =>
    request<{ data: ConfigEntry[]; count: number }>('config', '/api/v1/config/bulk/set', { method: 'POST', body: { entries } }),
};

// User / Auth Service
export const authApi = {
  register: (data: RegisterRequest, tenantId: string) =>
    request<User>('user', '/api/v1/auth/register', { method: 'POST', body: data, tenantId }),
  login: (data: LoginRequest, tenantId: string) =>
    request<LoginResponse>('user', '/api/v1/auth/login', { method: 'POST', body: data, tenantId }),
  profile: (tenantId: string, token: string) =>
    request<User>('user', '/api/v1/auth/profile', { tenantId, token }),
  forgotPassword: (tenantId: string, email: string) =>
    request<{ message: string }>('user', '/api/v1/auth/forgot-password', { method: 'POST', body: { tenant_id: tenantId, email } }),
  resetPassword: (token: string, newPassword: string) =>
    request<{ message: string }>('user', '/api/v1/auth/reset-password', { method: 'POST', body: { token, new_password: newPassword } }),
  changePassword: (oldPassword: string, newPassword: string, tenantId: string, authToken: string) =>
    request<{ message: string }>('user', '/api/v1/auth/change-password', { method: 'POST', body: { old_password: oldPassword, new_password: newPassword }, tenantId, token: authToken }),
  updateProfile: (userId: string, data: { first_name?: string; last_name?: string; phone?: string }, tenantId: string, authToken: string) =>
    request<User>('user', `/api/v1/users/${userId}`, { method: 'PUT', body: data, tenantId, token: authToken }),
};

export const userApi = {
  list: (tenantId: string, token: string, page = 1, pageSize = 20) =>
    request<PaginatedResponse<User>>('user', `/api/v1/users?page=${page}&page_size=${pageSize}`, { tenantId, token }),
  get: (id: string, tenantId: string, token: string) =>
    request<User>('user', `/api/v1/users/${id}`, { tenantId, token }),
};

// Product Service
export const productApi = {
  list: (tenantId: string, page = 1, pageSize = 20, status?: string) => {
    const params = new URLSearchParams({ tenant_id: tenantId, limit: String(pageSize), offset: String((page - 1) * pageSize) });
    if (status) params.set('status', status);
    return request<ProductListResponse>('product', `/api/v1/products?${params}`, { tenantId });
  },
  get: (id: string, tenantId: string) =>
    request<ProductResponse>('product', `/api/v1/products/${id}`, { tenantId }),
  search: (tenantId: string, q: string, minPrice?: number, maxPrice?: number) => {
    const params = new URLSearchParams({ tenant_id: tenantId, q });
    if (minPrice !== undefined) params.set('min_price', String(minPrice));
    if (maxPrice !== undefined) params.set('max_price', String(maxPrice));
    return request<ProductListResponse>('product', `/api/v1/products/search?${params}`, { tenantId });
  },
  create: (data: CreateProductRequest, tenantId: string, token?: string) =>
    request<ProductResponse>('product', '/api/v1/products', { method: 'POST', body: data, tenantId, token }),
  update: (id: string, data: Partial<CreateProductRequest> & { updated_by: string }, tenantId: string, token?: string) =>
    request<ProductResponse>('product', `/api/v1/products/${id}`, { method: 'PUT', body: data, tenantId, token }),
  delete: (id: string, tenantId: string, token?: string) =>
    request<void>('product', `/api/v1/products/${id}`, { method: 'DELETE', tenantId, token }),
};

export const categoryApi = {
  list: (tenantId: string) =>
    request<CategoryListResponse>('product', `/api/v1/categories?tenant_id=${tenantId}`, { tenantId }),
  get: (id: string, tenantId: string) =>
    request<CategoryResponse>('product', `/api/v1/categories/${id}`, { tenantId }),
  create: (data: CreateCategoryRequest, tenantId: string) =>
    request<CategoryResponse>('product', '/api/v1/categories', { method: 'POST', body: data, tenantId }),
  update: (id: string, data: UpdateCategoryRequest, tenantId: string) =>
    request<CategoryResponse>('product', `/api/v1/categories/${id}`, { method: 'PUT', body: data, tenantId }),
  delete: (id: string, tenantId: string) =>
    request<void>('product', `/api/v1/categories/${id}`, { method: 'DELETE', tenantId }),
  updateStatus: (id: string, status: 'active' | 'inactive', tenantId: string) =>
    request<CategoryResponse>('product', `/api/v1/categories/${id}/status`, { method: 'PATCH', body: { status }, tenantId }),
};

// Order Service
// Backend response shapes differ from frontend types — map them here.

/* eslint-disable @typescript-eslint/no-explicit-any */
function mapOrder(raw: any): Order {
  // Single-order endpoint wraps as { order: {...}, items: [...] }
  const o = raw.order ?? raw;
  const items: OrderItem[] = (raw.items ?? o.items ?? []).map((i: any) => ({
    id: i.id,
    product_id: i.product_id,
    variant_id: i.variant_id ?? '',
    sku: i.sku ?? '',
    name: i.name ?? '',
    quantity: i.quantity,
    unit_price: i.unit_price,
    total_price: i.total_price ?? i.unit_price * i.quantity,
  }));
  return {
    id: o.id,
    tenant_id: o.tenant_id ?? '',
    customer_id: o.customer_id ?? '',
    order_number: o.order_number ?? o.id?.slice(0, 8).toUpperCase() ?? '',
    status: o.status ?? 'pending',
    currency: o.currency ?? 'BDT',
    items,
    subtotal: o.subtotal ?? o.total_amount ?? 0,
    shipping_cost: o.shipping_cost ?? 0,
    tax: o.tax ?? 0,
    total: o.total ?? o.total_amount ?? 0,
    shipping_address: o.shipping_address ?? { street: '', city: '', state: '', postal_code: '', country: '' },
    billing_address: o.billing_address ?? { street: '', city: '', state: '', postal_code: '', country: '' },
    tracking_number: o.tracking_number ?? null,
    carrier: o.carrier ?? null,
    created_at: o.created_at ?? '',
    updated_at: o.updated_at ?? '',
  };
}

function mapOrderList(raw: any): PaginatedResponse<Order> {
  const orders = (raw.orders ?? raw.data ?? []).map(mapOrder);
  const p = raw.pagination ?? {};
  return {
    data: orders,
    total: p.count ?? orders.length,
    page: p.offset != null && p.limit ? Math.floor(p.offset / p.limit) + 1 : 1,
    page_size: p.limit ?? 20,
  };
}
/* eslint-enable @typescript-eslint/no-explicit-any */

export const orderApi = {
  create: (data: CreateOrderRequest, tenantId: string, token?: string) =>
    request<CreateOrderResponse>('order', '/api/v1/orders', { method: 'POST', body: data, tenantId, token }),
  addItem: (orderId: string, data: AddOrderItemRequest, tenantId: string, token?: string) =>
    request<void>('order', `/api/v1/orders/${orderId}/items`, { method: 'POST', body: data, tenantId, token }),
  get: (id: string, tenantId: string, token?: string) =>
    request<any>('order', `/api/v1/orders/${id}`, { tenantId, token }).then(mapOrder),
  listByTenant: (tenantId: string, token?: string, page = 1, pageSize = 20) =>
    request<any>('order', `/api/v1/tenants/${tenantId}/orders?page=${page}&page_size=${pageSize}`, { tenantId, token }).then(mapOrderList),
  listByCustomer: (customerId: string, tenantId: string, token?: string) =>
    request<any>('order', `/api/v1/customers/${customerId}/orders`, { tenantId, token }).then(mapOrderList),
  confirm: (id: string, confirmedBy: string, tenantId: string, token?: string) =>
    request<any>('order', `/api/v1/orders/${id}/confirm`, { method: 'POST', body: { confirmed_by: confirmedBy }, tenantId, token }).then(mapOrder),
  cancel: (id: string, reason: string, cancelledBy: string, tenantId: string, token?: string) =>
    request<any>('order', `/api/v1/orders/${id}/cancel`, { method: 'POST', body: { reason, cancelled_by: cancelledBy }, tenantId, token }).then(mapOrder),
  ship: (id: string, data: { tracking_number: string; carrier: string; shipped_by: string }, tenantId: string, token?: string) =>
    request<any>('order', `/api/v1/orders/${id}/ship`, { method: 'POST', body: data, tenantId, token }).then(mapOrder),
};

// Promotion Service
export const promotionApi = {
  validate: (code: string, data: { tenant_id: string; user_id: string; order_total: number }) =>
    request<CouponValidateResponse>('promotion', `/api/v1/coupons/validate/${encodeURIComponent(code)}`, { method: 'POST', body: data }),
  apply: (data: { tenant_id: string; user_id: string; order_id: string; order_total: number; code: string }) =>
    request<CouponValidateResponse>('promotion', '/api/v1/coupons/apply', { method: 'POST', body: data }),
};

// Cart Service
export const cartApi = {
  get: (userId: string, tenantId: string, token: string) =>
    request<CartResponse>('cart', `/api/v1/cart?tenant_id=${tenantId}&user_id=${userId}`, { tenantId, token }),
  addItem: (data: CartAddItemRequest, tenantId: string, token: string) =>
    request<CartResponse>('cart', '/api/v1/cart/items', { method: 'POST', body: data, tenantId, token }),
  updateItem: (itemId: string, quantity: number, userId: string, tenantId: string, token: string) =>
    request<CartResponse>('cart', `/api/v1/cart/items/${itemId}?tenant_id=${tenantId}&user_id=${userId}`, { method: 'PUT', body: { quantity }, tenantId, token }),
  removeItem: (itemId: string, userId: string, tenantId: string, token: string) =>
    request<CartResponse>('cart', `/api/v1/cart/items/${itemId}?tenant_id=${tenantId}&user_id=${userId}`, { method: 'DELETE', tenantId, token }),
  clear: (userId: string, tenantId: string, token: string) =>
    request<void>('cart', `/api/v1/cart?tenant_id=${tenantId}&user_id=${userId}`, { method: 'DELETE', tenantId, token }),
};

// Vendor Service
export const vendorApi = {
  list: (tenantId: string, token: string, status?: string, page = 1, pageSize = 20) => {
    const p = new URLSearchParams({ tenant_id: tenantId, page: String(page), page_size: String(pageSize) });
    if (status) p.set('status', status);
    return request<{ vendors: Vendor[]; total: number; page: number; page_size: number }>('vendor', `/api/v1/vendors?${p}`, { tenantId, token });
  },
  get: (id: string, tenantId: string, token: string) =>
    request<Vendor>('vendor', `/api/v1/vendors/${id}`, { tenantId, token }),
  register: (data: RegisterVendorRequest, tenantId: string, token: string) =>
    request<Vendor>('vendor', '/api/v1/vendors/register', { method: 'POST', body: data, tenantId, token }),
  updateStatus: (id: string, status: string, reason: string, tenantId: string, token: string) =>
    request<Vendor>('vendor', `/api/v1/vendors/${id}/status`, { method: 'PUT', body: { status, reason }, tenantId, token }),
};

// Wishlist Service (via user-service)
export const wishlistApi = {
  get: (tenantId: string, token: string) =>
    request<WishlistResponse>('user', `/api/v1/wishlist?tenant_id=${tenantId}`, { tenantId, token }),
  addItem: (data: AddWishlistItemRequest, tenantId: string, token: string) =>
    request<WishlistResponse>('user', '/api/v1/wishlist/items', { method: 'POST', body: data, tenantId, token }),
  removeItem: (productId: string, tenantId: string, token: string) =>
    request<void>('user', `/api/v1/wishlist/items/${encodeURIComponent(productId)}`, { method: 'DELETE', tenantId, token }),
  clear: (tenantId: string, token: string) =>
    request<void>('user', '/api/v1/wishlist', { method: 'DELETE', tenantId, token }),
};

// Search Service
export const searchApi = {
  search: (params: {
    q?: string;
    tenant_id: string;
    category_id?: string;
    brand?: string;
    min_price?: number;
    max_price?: number;
    tags?: string;
    in_stock?: boolean;
    sort_by?: string;
    sort_order?: string;
    page?: number;
    page_size?: number;
  }) => {
    const p = new URLSearchParams({ tenant_id: params.tenant_id });
    if (params.q) p.set('q', params.q);
    if (params.category_id) p.set('category_id', params.category_id);
    if (params.brand) p.set('brand', params.brand);
    if (params.min_price !== undefined) p.set('min_price', String(params.min_price));
    if (params.max_price !== undefined) p.set('max_price', String(params.max_price));
    if (params.tags) p.set('tags', params.tags);
    if (params.in_stock !== undefined) p.set('in_stock', String(params.in_stock));
    if (params.sort_by) p.set('sort_by', params.sort_by);
    if (params.sort_order) p.set('sort_order', params.sort_order);
    if (params.page) p.set('page', String(params.page));
    if (params.page_size) p.set('page_size', String(params.page_size));
    return request<SearchResponse>('search', `/api/v1/search/products?${p}`);
  },
  autocomplete: (q: string, tenantId: string, limit = 8) =>
    request<AutocompleteResponse>('search', `/api/v1/search/autocomplete?q=${encodeURIComponent(q)}&tenant_id=${tenantId}&limit=${limit}`),
};

// Shipping Service
export const shippingApi = {
  getRates: (data: ShippingRateRequest, token?: string) =>
    request<ShippingRatesResponse>('shipping', '/api/v1/rates', { method: 'POST', body: data, token }),
};

// Inventory Service
export const inventoryApi = {
  listItems: (tenantId: string, token: string, offset = 0, limit = 20) =>
    request<PaginatedResponse<InventoryItem>>('inventory', `/api/v1/inventory/items?tenantId=${tenantId}&offset=${offset}&limit=${limit}`, { tenantId, token }),
  lowStock: (tenantId: string, token: string) =>
    request<InventoryItem[]>('inventory', `/api/v1/inventory/items/low-stock?tenantId=${tenantId}`, { tenantId, token }),
  listWarehouses: (tenantId: string, token: string) =>
    request<PaginatedResponse<Warehouse>>('inventory', `/api/v1/inventory/warehouses?tenantId=${tenantId}`, { tenantId, token }),
  createWarehouse: (data: CreateWarehouseRequest, token: string) =>
    request<Warehouse>('inventory', '/api/v1/inventory/warehouses', { method: 'POST', body: data, tenantId: data.tenantId, token }),
  createItem: (data: CreateInventoryItemRequest, token: string) =>
    request<InventoryItem>('inventory', '/api/v1/inventory/items', { method: 'POST', body: data, tenantId: data.tenantId, token }),
  adjustStock: (itemId: string, data: { quantity: number; reason: string; type: string }, tenantId: string, token: string) =>
    request<InventoryItem>('inventory', `/api/v1/inventory/items/${itemId}/adjust`, { method: 'POST', body: data, tenantId, token }),
};

// Audit Logs (via Tenant Service)
export interface AuditLog {
  id: string;
  tenant_id: string;
  user_id: string;
  action: string;
  resource: string;
  resource_id: string;
  method: string;
  path: string;
  ip_address: string;
  user_agent: string;
  request_body?: string;
  response_code: number;
  old_value?: string;
  new_value?: string;
  metadata?: string;
  error_message?: string;
  duration_ms: number;
  created_at: string;
}

export interface AuditLogResponse {
  data: AuditLog[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export const auditApi = {
  list: (tenantId: string, token: string, filters?: {
    user_id?: string;
    action?: string;
    resource?: string;
    resource_id?: string;
    start_date?: string;
    end_date?: string;
    page?: number;
    page_size?: number;
  }) => {
    const params = new URLSearchParams({ tenant_id: tenantId });
    if (filters?.user_id) params.set('user_id', filters.user_id);
    if (filters?.action) params.set('action', filters.action);
    if (filters?.resource) params.set('resource', filters.resource);
    if (filters?.resource_id) params.set('resource_id', filters.resource_id);
    if (filters?.start_date) params.set('start_date', filters.start_date);
    if (filters?.end_date) params.set('end_date', filters.end_date);
    params.set('page', String(filters?.page ?? 1));
    params.set('page_size', String(filters?.page_size ?? 25));
    return request<AuditLogResponse>('tenant', `/api/v1/audit-logs?${params}`, { tenantId, token });
  },
  get: (id: string, tenantId: string, token: string) =>
    request<AuditLog>('tenant', `/api/v1/audit-logs/${id}`, { tenantId, token }),
};

// Payment Service
export const paymentApi = {
  list: (tenantId: string, token?: string, page = 1, pageSize = 20) =>
    request<PaginatedResponse<Payment>>('payment', `/api/v1/payments?tenant_id=${tenantId}&page=${page}&page_size=${pageSize}`, { tenantId, token }),
  get: (id: string, tenantId: string, token?: string) =>
    request<Payment>('payment', `/api/v1/payments/${id}`, { tenantId, token }),
  process: (data: ProcessPaymentRequest, tenantId: string, token?: string) =>
    request<Payment>('payment', '/api/v1/payments', { method: 'POST', body: data, tenantId, token }),
};

// Analytics Service
export const analyticsApi = {
  sales: (tenantId: string, startDate: string, endDate: string, granularity = 'daily') =>
    request<SalesReport>('analytics', `/api/v1/analytics/sales?tenant_id=${tenantId}&start_date=${startDate}&end_date=${endDate}&granularity=${granularity}`, { tenantId }),
  customers: (tenantId: string, startDate: string, endDate: string) =>
    request<CustomerInsights>('analytics', `/api/v1/analytics/customers?tenant_id=${tenantId}&start_date=${startDate}&end_date=${endDate}`, { tenantId }),
  products: (tenantId: string, startDate: string, endDate: string) =>
    request<ProductPerformance>('analytics', `/api/v1/analytics/products?tenant_id=${tenantId}&start_date=${startDate}&end_date=${endDate}`, { tenantId }),
};

// Review Service
export const reviewApi = {
  listByProduct: (productId: string, tenantId: string, page = 1, pageSize = 20) =>
    request<ReviewListResponse>('review', `/api/v1/reviews/product/${productId}?tenant_id=${tenantId}&page=${page}&page_size=${pageSize}`, { tenantId }),
  summary: (productId: string, tenantId: string) =>
    request<ReviewSummary>('review', `/api/v1/reviews/product/${productId}/summary?tenant_id=${tenantId}`, { tenantId }),
  create: (data: CreateReviewRequest, tenantId: string, token: string) =>
    request<ReviewResponse>('review', '/api/v1/reviews', { method: 'POST', body: data, tenantId, token }),
  helpful: (reviewId: string, userId: string, helpful: boolean, tenantId: string, token: string) =>
    request<void>('review', `/api/v1/reviews/${reviewId}/helpful`, { method: 'POST', body: { user_id: userId, helpful }, tenantId, token }),
};

// Recommendation Service
export const recommendationApi = {
  forProduct: (productId: string, tenantId: string, limit = 10) =>
    request<RecommendationResponse>('recommendation', `/api/v1/recommendations/product/${productId}?tenant_id=${tenantId}&limit=${limit}`, { tenantId }),
  forUser: (userId: string, tenantId: string, limit = 10) =>
    request<RecommendationResponse>('recommendation', `/api/v1/recommendations/user/${userId}?tenant_id=${tenantId}&limit=${limit}`, { tenantId }),
};

// Types

export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  page_size: number;
  total: number;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  domain: string | null;
  email: string;
  status: 'active' | 'suspended' | 'cancelled' | 'pending';
  tier: 'free' | 'starter' | 'professional' | 'enterprise';
  config: TenantConfig;
  created_at: string;
  updated_at: string;
}

export interface TenantConfig {
  general: {
    timezone: string;
    currency: string;
    language: string;
    date_format: string;
    time_format: string;
    contact_email: string;
    contact_phone: string;
    support_url: string;
  };
  branding: {
    logo_url: string;
    favicon_url: string;
    primary_color: string;
    secondary_color: string;
    custom_css: string;
    custom_fonts: Record<string, string>;
  };
  features: {
    multi_currency: boolean;
    wishlist: boolean;
    product_reviews: boolean;
    guest_checkout: boolean;
    social_login: boolean;
    ai_recommendations: boolean;
    loyalty_program: boolean;
    subscriptions: boolean;
    gift_cards: boolean;
    [key: string]: boolean;
  };
}

export interface CreateTenantRequest {
  name: string;
  email: string;
  tier: string;
}

export interface User {
  id: string;
  tenant_id: string;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  phone: string | null;
  avatar: string | null;
  status: 'active' | 'inactive' | 'suspended';
  role: 'admin' | 'moderator' | 'customer' | 'guest';
  email_verified: boolean;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface RegisterRequest {
  tenant_id: string;
  email: string;
  username: string;
  password: string;
  first_name: string;
  last_name: string;
  phone?: string;
}

export interface LoginRequest {
  tenant_id: string;
  email: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  token: string;
  expires_at: string;
}

export interface DeliveryProfile {
  id: string;
  name: string;
  inside_dhaka_rate: number;
  outside_dhaka_rate: number;
  inside_dhaka_express_rate: number;
  outside_dhaka_express_rate: number;
  estimated_delivery_dhaka: string;
  estimated_delivery_outside: string;
  is_default: boolean;
}

export interface Product {
  id: string;
  tenant_id: string;
  sku: string;
  name: string;
  slug: string;
  description: string;
  category_id: string;
  brand?: string;
  price: number;
  compare_at_price?: number | null;
  cost_per_item?: number;
  delivery_profile_id?: string;
  images: string[] | null;
  tags: string[] | null;
  status: 'active' | 'draft' | 'archived' | 'inactive';
  variants?: ProductVariant[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ProductVariant {
  id?: string;
  sku: string;
  name: string;
  price: number;
  value?: string;
  stock?: number;
  options?: Record<string, string>;
}

export interface CreateProductRequest {
  tenant_id: string;
  name: string;
  slug?: string;
  description?: string;
  sku: string;
  price: number;
  compare_at_price?: number;
  category_id: string;
  delivery_profile_id?: string;
  status?: string;
  images?: string[];
  variants?: ProductVariant[];
  tags?: string[];
  created_by: string;
}

export type ProductResponse = Product;

export interface ProductListResponse {
  data: Product[];
  pagination?: { page: number; page_size: number; total_items: number; total_pages: number };
  limit?: number;
  offset?: number;
  total?: number;
}

export interface Category {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  description: string;
  parent_id: string | null;
  image_url: string | null;
  status: 'active' | 'inactive';
  created_at: string;
  updated_at: string;
}

export type CategoryResponse = Category;

export interface CategoryListResponse {
  data: Category[];
  limit?: number;
  offset?: number;
  total?: number;
}

export interface CreateCategoryRequest {
  tenant_id: string;
  name: string;
  slug: string;
  description?: string;
  parent_id?: string | null;
  image?: string;
  icon?: string;
  sort_order?: number;
  created_by: string;
}

export interface UpdateCategoryRequest {
  name?: string;
  slug?: string;
  description?: string;
  parent_id?: string | null;
  image?: string;
  icon?: string;
  sort_order?: number;
  updated_by: string;
}

export interface Order {
  id: string;
  tenant_id: string;
  customer_id: string;
  order_number: string;
  status: 'pending' | 'confirmed' | 'shipped' | 'delivered' | 'cancelled';
  currency: string;
  items: OrderItem[];
  subtotal: number;
  shipping_cost: number;
  tax: number;
  total: number;
  shipping_address: Address;
  billing_address: Address;
  tracking_number: string | null;
  carrier: string | null;
  created_at: string;
  updated_at: string;
}

export interface OrderItem {
  id: string;
  product_id: string;
  variant_id: string;
  sku: string;
  name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

export interface Address {
  street: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
}

export interface CreateOrderRequest {
  tenant_id: string;
  customer_id?: string;
  guest_email?: string;
  guest_name?: string;
  guest_phone?: string;
  shipping_address: Address;
  billing_address: Address;
}

export interface CreateOrderResponse {
  order_id: string;
  id?: string;
  message?: string;
}

export interface AddOrderItemRequest {
  product_id: string;
  variant_id?: string;
  sku: string;
  name: string;
  quantity: number;
  unit_price: number;
}

export interface InventoryItem {
  id: string;
  tenantId: string;
  warehouseId: string;
  warehouseName: string;
  productId: string;
  variantId: string | null;
  sku: string;
  quantityOnHand: number;
  quantityReserved: number;
  quantityAvailable: number;
  reorderPoint: number;
  reorderQuantity: number;
  maxStock: number | null;
  binLocation: string | null;
  needsReorder: boolean;
  lastStockCheckAt: string | null;
  lastReceivedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface Warehouse {
  id: string;
  tenantId: string;
  code: string;
  name: string;
  description: string | null;
  address: string;
  city: string;
  state: string;
  country: string;
  postalCode: string;
  phone: string | null;
  email: string | null;
  isActive: boolean;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateWarehouseRequest {
  tenantId: string;
  code: string;
  name: string;
  description?: string;
  address: string;
  city: string;
  state: string;
  country: string;
  postalCode: string;
  phone?: string;
  email?: string;
  isActive?: boolean;
  isDefault?: boolean;
  createdBy: string;
}

export interface CreateInventoryItemRequest {
  tenantId: string;
  warehouseId: string;
  productId: string;
  variantId?: string;
  sku: string;
  initialQuantity?: number;
  reorderPoint?: number;
  reorderQuantity?: number;
  maxStock?: number;
  binLocation?: string;
  createdBy: string;
}

export interface Payment {
  id: string;
  tenant_id: string;
  customer_id: string;
  order_id: string;
  amount: number;
  currency: string;
  method: string;
  status: 'Pending' | 'Completed' | 'Failed' | 'Cancelled' | 'Refunded';
  transaction_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface ProcessPaymentRequest {
  tenant_id: string;
  customer_id: string;
  order_id: string;
  amount: number;
  currency: string;
  method: string;
}

export interface SalesReport {
  total_revenue: number;
  total_orders: number;
  avg_order_value: number;
  data_points: { date: string; revenue: number; orders: number }[];
}

export interface CustomerInsights {
  total_customers: number;
  new_customers: number;
  returning_customers: number;
  top_customers: { id: string; name: string; total_spent: number }[];
}

export interface ProductPerformance {
  top_products: { id: string; name: string; revenue: number; units_sold: number }[];
  categories_breakdown: { category: string; revenue: number; percentage: number }[];
}

export interface ConfigEntry {
  id: string;
  namespace: string;
  key: string;
  value: string;
  value_type: 'string' | 'number' | 'boolean' | 'json';
  description: string;
  environment: string;
  tenant_id: string;
  is_secret: boolean;
  version: number;
  created_at: string;
  updated_at: string;
  updated_by: string;
}

export interface SetConfigRequest {
  namespace: string;
  key: string;
  value: string;
  value_type: 'string' | 'number' | 'boolean' | 'json';
  description?: string;
  environment?: string;
  tenant_id?: string;
  is_secret?: boolean;
  updated_by?: string;
}

export interface ReviewResponse {
  id: string;
  tenant_id: string;
  product_id: string;
  user_id: string;
  user_name: string;
  order_id?: string;
  rating: number;
  title: string;
  comment: string;
  images?: string[];
  status: string;
  helpful_count: number;
  seller_response?: string;
  created_at: string;
  updated_at: string;
}

export interface ReviewListResponse {
  data: ReviewResponse[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
}

export interface ReviewSummary {
  product_id: string;
  average_rating: number;
  total_reviews: number;
  distribution: Record<string, number>;
}

export interface CreateReviewRequest {
  tenant_id: string;
  product_id: string;
  user_id: string;
  order_id?: string;
  rating: number;
  title: string;
  comment: string;
  images?: string[];
}

export interface ProductRecommendation {
  product_id: string;
  score: number;
  reason: string;
}

export interface RecommendationResponse {
  user_id?: string;
  product_id?: string;
  recommendations: ProductRecommendation[];
  strategy: string;
  generated_at: string;
}

export interface CouponValidateResponse {
  valid: boolean;
  code: string;
  discount_type?: 'percentage' | 'fixed';
  discount_value?: number;
  discount_amount?: number;
  message?: string;
}

export interface CartAddItemRequest {
  tenant_id: string;
  user_id: string;
  product_id: string;
  name: string;
  price: number;
  quantity: number;
  image_url?: string;
}

export interface CartItemResponse {
  id: string;
  product_id: string;
  name: string;
  price: number;
  quantity: number;
  subtotal: number;
  image_url?: string;
  added_at: string;
}

export interface CartResponse {
  tenant_id: string;
  user_id: string;
  items: CartItemResponse[];
  total_items: number;
  total_amount: number;
  updated_at: string;
}

export interface ShippingAddress {
  name: string;
  street: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
}

export interface ShippingRateRequest {
  tenant_id: string;
  from_address: ShippingAddress;
  to_address: ShippingAddress;
  weight_oz: number;
  length_in?: number;
  width_in?: number;
  height_in?: number;
}

export interface ShippingRate {
  carrier: string;
  service_type: string;
  rate: number;
  currency: string;
  estimated_days: number;
}

export interface ShippingRatesResponse {
  rates: ShippingRate[];
}

export interface SearchProduct {
  id: string;
  tenant_id: string;
  sku: string;
  name: string;
  description: string;
  brand: string;
  category_id: string;
  price: number;
  images: string[] | null;
  tags: string[] | null;
  status: string;
  in_stock: boolean;
  stock_quantity: number;
  _score?: number;
}

export interface SearchFacets {
  categories: { key: string; count: number }[];
  brands: { key: string; count: number }[];
  tags: { key: string; count: number }[];
  price_range: { min: number; max: number; avg: number };
}

export interface SearchResponse {
  products: SearchProduct[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
  facets: SearchFacets;
}

export interface AutocompleteResult {
  text: string;
  type: string;
  id: string;
  _score?: number;
}

export interface AutocompleteResponse {
  suggestions: AutocompleteResult[];
}

export interface Vendor {
  id: string;
  tenant_id: string;
  name: string;
  email: string;
  phone: string;
  description: string;
  logo_url: string;
  address: string;
  city: string;
  country: string;
  status: 'pending' | 'active' | 'suspended' | 'rejected';
  commission_rate: number;
  total_revenue: number;
  total_orders: number;
  total_products: number;
  rating: number;
  suspend_reason?: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface RegisterVendorRequest {
  tenant_id: string;
  name: string;
  email: string;
  phone?: string;
  description?: string;
  address?: string;
  city?: string;
  country?: string;
}

export interface WishlistItemResponse {
  id: string;
  user_id: string;
  tenant_id: string;
  product_id: string;
  name: string;
  slug: string;
  price: number;
  image?: string;
  added_at: string;
}

export interface WishlistResponse {
  items: WishlistItemResponse[];
  count: number;
}

export interface AddWishlistItemRequest {
  product_id: string;
  name: string;
  slug?: string;
  price?: number;
  image?: string;
}
