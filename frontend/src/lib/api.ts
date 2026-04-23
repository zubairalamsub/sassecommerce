const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost';

const SERVICE_PORTS: Record<string, number> = {
  tenant: 8081,
  user: 8082,
  order: 8080,
  product: 8083,
  inventory: 8084,
  payment: 8085,
  shipping: 8086,
  notification: 8087,
  review: 8088,
  cart: 8089,
  search: 8090,
  promotion: 8091,
  vendor: 8092,
  analytics: 8093,
  recommendation: 8094,
  config: 8095,
};

function serviceUrl(service: string): string {
  const port = SERVICE_PORTS[service] || 8080;
  return `${API_BASE}:${port}`;
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
  create: (data: CreateProductRequest, tenantId: string) =>
    request<ProductResponse>('product', '/api/v1/products', { method: 'POST', body: data, tenantId }),
  update: (id: string, data: Partial<CreateProductRequest> & { updated_by: string }, tenantId: string) =>
    request<ProductResponse>('product', `/api/v1/products/${id}`, { method: 'PUT', body: data, tenantId }),
  delete: (id: string, tenantId: string) =>
    request<void>('product', `/api/v1/products/${id}`, { method: 'DELETE', tenantId }),
};

export const categoryApi = {
  list: (tenantId: string) =>
    request<CategoryListResponse>('product', `/api/v1/categories?tenant_id=${tenantId}`, { tenantId }),
  get: (id: string, tenantId: string) =>
    request<CategoryResponse>('product', `/api/v1/categories/${id}`, { tenantId }),
};

// Order Service
export const orderApi = {
  create: (data: CreateOrderRequest, tenantId: string) =>
    request<Order>('order', '/api/v1/orders', { method: 'POST', body: data, tenantId }),
  get: (id: string, tenantId: string) =>
    request<Order>('order', `/api/v1/orders/${id}`, { tenantId }),
  listByTenant: (tenantId: string, page = 1, pageSize = 20) =>
    request<PaginatedResponse<Order>>('order', `/api/v1/tenants/${tenantId}/orders?page=${page}&page_size=${pageSize}`, { tenantId }),
  listByCustomer: (customerId: string, tenantId: string) =>
    request<PaginatedResponse<Order>>('order', `/api/v1/customers/${customerId}/orders`, { tenantId }),
  confirm: (id: string, confirmedBy: string, tenantId: string) =>
    request<Order>('order', `/api/v1/orders/${id}/confirm`, { method: 'POST', body: { confirmed_by: confirmedBy }, tenantId }),
  cancel: (id: string, reason: string, cancelledBy: string, tenantId: string) =>
    request<Order>('order', `/api/v1/orders/${id}/cancel`, { method: 'POST', body: { reason, cancelled_by: cancelledBy }, tenantId }),
  ship: (id: string, data: { tracking_number: string; carrier: string; shipped_by: string }, tenantId: string) =>
    request<Order>('order', `/api/v1/orders/${id}/ship`, { method: 'POST', body: data, tenantId }),
};

// Inventory Service
export const inventoryApi = {
  listItems: (tenantId: string, page = 1, pageSize = 20) =>
    request<PaginatedResponse<InventoryItem>>('inventory', `/api/v1/inventory/items?tenant_id=${tenantId}&page=${page}&page_size=${pageSize}`, { tenantId }),
  lowStock: (tenantId: string) =>
    request<InventoryItem[]>('inventory', `/api/v1/inventory/items/low-stock?tenant_id=${tenantId}`, { tenantId }),
  listWarehouses: (tenantId: string) =>
    request<Warehouse[]>('inventory', `/api/v1/inventory/warehouses?tenant_id=${tenantId}`, { tenantId }),
  adjustStock: (itemId: string, data: { quantity: number; reason: string; type: string }, tenantId: string) =>
    request<InventoryItem>('inventory', `/api/v1/inventory/items/${itemId}/adjust`, { method: 'POST', body: data, tenantId }),
};

// Payment Service
export const paymentApi = {
  list: (tenantId: string, page = 1, pageSize = 20) =>
    request<PaginatedResponse<Payment>>('payment', `/api/v1/payments?tenant_id=${tenantId}&page=${page}&page_size=${pageSize}`, { tenantId }),
  get: (id: string, tenantId: string) =>
    request<Payment>('payment', `/api/v1/payments/${id}`, { tenantId }),
  process: (data: ProcessPaymentRequest, tenantId: string) =>
    request<Payment>('payment', '/api/v1/payments', { method: 'POST', body: data, tenantId }),
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
  customer_id: string;
  shipping_address: Address;
  billing_address: Address;
}

export interface InventoryItem {
  id: string;
  tenant_id: string;
  warehouse_id: string;
  product_id: string;
  sku: string;
  quantity_on_hand: number;
  reorder_point: number;
  reorder_quantity: number;
  created_at: string;
  updated_at: string;
}

export interface Warehouse {
  id: string;
  tenant_id: string;
  code: string;
  name: string;
  city: string;
  is_active: boolean;
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
