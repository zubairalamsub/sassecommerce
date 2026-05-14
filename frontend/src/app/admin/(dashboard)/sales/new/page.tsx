'use client';

/**
 * Instant Sell (POS) — admins record in-store, phone, or walk-in sales here
 * and produce a confirmed/paid order without going through the customer-facing
 * checkout. The order is created via `orderApi.create` then each line is added
 * via `orderApi.addItem` (event-sourced order service requires sequential adds).
 *
 * Layout: two columns on lg (left = product/customer picker, right = sticky
 * summary + payment). Single column on mobile; the summary moves to the bottom.
 */

import { useState, useEffect, useMemo, useRef } from 'react';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import Link from 'next/link';
import {
  Zap,
  Search,
  X,
  Plus,
  Minus,
  User as UserIcon,
  UserPlus,
  Trash2,
  Receipt as ReceiptIcon,
  Loader2,
  Printer,
  CheckCircle2,
  ArrowLeft,
  Mail,
} from 'lucide-react';
import { cn, formatCurrency, mediaUrl } from '@/lib/utils';
import {
  orderApi,
  tenantApi,
  userApi,
  type Order,
  type Tenant,
  type User,
  type Product,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { useProductStore } from '@/stores/products';
import { toast } from '@/stores/toast';
import Receipt from '@/components/receipt';

// ---- Types ----

interface LineItem {
  /** Stable client-side id so React keys survive quantity changes. */
  key: string;
  product_id: string;
  variant_id: string;
  sku: string;
  name: string;
  /** Editable unit price — admin can negotiate at the counter. */
  unit_price: number;
  quantity: number;
  /** Stock cap from product/variant — used to disable the + button at limit. */
  stock: number | null;
  image: string | null;
}

type DiscountMode = 'percent' | 'fixed';
type PaymentMethod = 'cash' | 'card' | 'bkash' | 'nagad' | 'bank_transfer' | 'cod';

const PAYMENT_METHODS: { value: PaymentMethod; label: string }[] = [
  { value: 'cash', label: 'Cash' },
  { value: 'card', label: 'Card' },
  { value: 'bkash', label: 'bKash' },
  { value: 'nagad', label: 'Nagad' },
  { value: 'bank_transfer', label: 'Bank Transfer' },
  { value: 'cod', label: 'COD (Cash on Delivery)' },
];

// Treat the first variant as the "default" if there are variants and the
// product has no top-level stock. Returns the matching variant or null.
function getVariantStock(product: Product, variantId: string): number | null {
  const v = product.variants?.find((x) => x.id === variantId);
  if (!v) return null;
  return typeof v.stock === 'number' ? v.stock : null;
}

export default function InstantSellPage() {
  const router = useRouter();
  const { user, tenantId, token } = useAuthStore();
  const { products, fetchProducts } = useProductStore();

  // Customer state
  const [customers, setCustomers] = useState<User[]>([]);
  const [customerQuery, setCustomerQuery] = useState('');
  const [customerOpen, setCustomerOpen] = useState(false);
  const [selectedCustomer, setSelectedCustomer] = useState<User | null>(null);
  const [walkIn, setWalkIn] = useState(false);

  // Product picker state
  const [productQuery, setProductQuery] = useState('');
  const [productOpen, setProductOpen] = useState(false);
  const [items, setItems] = useState<LineItem[]>([]);

  // Pricing / meta state
  const [discountMode, setDiscountMode] = useState<DiscountMode>('fixed');
  const [discountValue, setDiscountValue] = useState('');
  const [notes, setNotes] = useState('');

  // Payment state
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('cash');
  const [amountReceived, setAmountReceived] = useState('');

  // Submission state
  const [submitting, setSubmitting] = useState(false);
  const [createdOrder, setCreatedOrder] = useState<Order | null>(null);
  /** Snapshot of the items used in the just-completed sale — we keep this
   *  alongside the order so the printable receipt still shows local line
   *  items even if the backend Order projection truncates or reorders them. */
  const [completedSnapshot, setCompletedSnapshot] = useState<{
    items: LineItem[];
    subtotal: number;
    discount: number;
    total: number;
    amountReceived: number | undefined;
    paymentMethod: PaymentMethod;
    customerLabel: string;
    customerEmail: string;
  } | null>(null);
  /** Loaded once when entering the success view so the receipt header can
   *  show the store's logo, address, and contact info. */
  const [tenant, setTenant] = useState<Tenant | null>(null);
  /** Email send button states. */
  const [sendingEmail, setSendingEmail] = useState(false);
  const [recipientEmail, setRecipientEmail] = useState('');

  const productSearchRef = useRef<HTMLDivElement>(null);
  const customerSearchRef = useRef<HTMLDivElement>(null);
  /** Used to auto-focus "New sale" 3s after the success screen renders. */
  const newSaleRef = useRef<HTMLButtonElement>(null);

  // Load data on mount
  useEffect(() => {
    if (!tenantId) return;
    fetchProducts(tenantId);
    if (token) {
      userApi
        .list(tenantId, token, 1, 200)
        .then((r) => setCustomers((r.data ?? []).filter((u) => u.role === 'customer')))
        .catch(() => setCustomers([]));
    }
  }, [tenantId, token, fetchProducts]);

  // Close dropdowns on outside click
  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (productSearchRef.current && !productSearchRef.current.contains(e.target as Node)) {
        setProductOpen(false);
      }
      if (customerSearchRef.current && !customerSearchRef.current.contains(e.target as Node)) {
        setCustomerOpen(false);
      }
    }
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, []);

  // ---- Derived ----

  const filteredCustomers = useMemo(() => {
    const q = customerQuery.trim().toLowerCase();
    if (!q) return customers.slice(0, 8);
    return customers
      .filter((c) => {
        const blob = `${c.first_name} ${c.last_name} ${c.email} ${c.phone ?? ''}`.toLowerCase();
        return blob.includes(q);
      })
      .slice(0, 8);
  }, [customers, customerQuery]);

  const filteredProducts = useMemo(() => {
    const q = productQuery.trim().toLowerCase();
    const active = products.filter((p) => p.status === 'active');
    if (!q) return active.slice(0, 10);
    return active
      .filter((p) => {
        const blob = `${p.name} ${p.sku} ${p.brand ?? ''}`.toLowerCase();
        return blob.includes(q);
      })
      .slice(0, 10);
  }, [products, productQuery]);

  const subtotal = useMemo(
    () => items.reduce((acc, i) => acc + i.unit_price * i.quantity, 0),
    [items],
  );

  const discountAmount = useMemo(() => {
    const v = Number(discountValue) || 0;
    if (v <= 0) return 0;
    if (discountMode === 'percent') {
      return Math.min(subtotal, (subtotal * v) / 100);
    }
    return Math.min(subtotal, v);
  }, [discountValue, discountMode, subtotal]);

  const total = Math.max(0, subtotal - discountAmount);

  const cashReceived = Number(amountReceived) || 0;
  const change = paymentMethod === 'cash' ? cashReceived - total : 0;

  const canSubmit =
    items.length > 0 &&
    !submitting &&
    (paymentMethod !== 'cash' || cashReceived >= total);

  // ---- Item management ----

  function addProduct(product: Product, variantId = '') {
    const variant = variantId ? product.variants?.find((v) => v.id === variantId) : undefined;
    const stock = variant ? getVariantStock(product, variantId) : null;
    const key = `${product.id}-${variantId || 'default'}`;

    setItems((prev) => {
      const existing = prev.find((i) => i.key === key);
      if (existing) {
        // Bump quantity (respecting stock if known)
        if (stock !== null && existing.quantity >= stock) {
          toast.warning(`Only ${stock} in stock`);
          return prev;
        }
        return prev.map((i) =>
          i.key === key ? { ...i, quantity: i.quantity + 1 } : i,
        );
      }
      return [
        ...prev,
        {
          key,
          product_id: product.id,
          variant_id: variantId,
          sku: variant?.sku || product.sku,
          name: variant ? `${product.name} — ${variant.name}` : product.name,
          unit_price: variant?.price ?? product.price,
          quantity: 1,
          stock,
          image: product.images?.[0] ?? null,
        },
      ];
    });
    setProductQuery('');
    setProductOpen(false);
  }

  function updateQty(key: string, qty: number) {
    setItems((prev) =>
      prev
        .map((i) => {
          if (i.key !== key) return i;
          if (qty <= 0) return null;
          if (i.stock !== null && qty > i.stock) {
            toast.warning(`Only ${i.stock} in stock`);
            return { ...i, quantity: i.stock };
          }
          return { ...i, quantity: qty };
        })
        .filter((i): i is LineItem => i !== null),
    );
  }

  function updatePrice(key: string, price: number) {
    setItems((prev) =>
      prev.map((i) => (i.key === key ? { ...i, unit_price: Math.max(0, price) } : i)),
    );
  }

  function removeItem(key: string) {
    setItems((prev) => prev.filter((i) => i.key !== key));
  }

  // ---- Customer selection ----

  function selectCustomer(c: User) {
    setSelectedCustomer(c);
    setWalkIn(false);
    setCustomerQuery('');
    setCustomerOpen(false);
  }

  function selectWalkIn() {
    setSelectedCustomer(null);
    setWalkIn(true);
    setCustomerQuery('');
    setCustomerOpen(false);
  }

  function clearCustomer() {
    setSelectedCustomer(null);
    setWalkIn(false);
  }

  // ---- Submission ----

  async function handleCreate() {
    if (!tenantId || items.length === 0) return;
    setSubmitting(true);
    try {
      // Build a "POS sale" annotation that lands in `notes` since the order
      // API doesn't yet carry a top-level `source` field.
      const sourceTag = '[POS]';
      const customerLabel = walkIn
        ? 'Walk-in customer'
        : selectedCustomer
        ? `${selectedCustomer.first_name} ${selectedCustomer.last_name}`
        : 'Walk-in customer';
      const paymentLabel = PAYMENT_METHODS.find((p) => p.value === paymentMethod)?.label ?? paymentMethod;
      const discountLabel =
        discountAmount > 0
          ? ` Discount: ${formatCurrency(discountAmount)}${
              discountMode === 'percent' ? ` (${discountValue}%)` : ''
            }.`
          : '';
      const adminNote = notes.trim() ? ` Note: ${notes.trim()}` : '';
      const orderNotes = `${sourceTag} ${customerLabel}. Payment: ${paymentLabel}.${discountLabel}${adminNote}`;

      // Address: walk-ins are in-store pickup so we ship a placeholder address
      // (the order API requires non-empty shipping/billing addresses).
      const placeholderAddress = {
        street: walkIn ? 'In-store pickup' : 'On file',
        city: 'Dhaka',
        state: 'Dhaka',
        postal_code: '1000',
        country: 'Bangladesh',
      };

      const createReq: Parameters<typeof orderApi.create>[0] = {
        tenant_id: tenantId,
        shipping_address: placeholderAddress,
        billing_address: placeholderAddress,
      };

      if (walkIn || !selectedCustomer) {
        createReq.guest_name = walkIn ? 'Walk-in customer' : customerLabel;
        createReq.guest_email = '';
        createReq.guest_phone = '';
      } else {
        createReq.customer_id = selectedCustomer.id;
      }

      const res = await orderApi.create(createReq, tenantId, token || undefined);
      const orderId = res.order_id || res.id || '';
      if (!orderId) throw new Error('Order created but no ID returned');

      // Add items sequentially (event-sourced order uses optimistic concurrency,
      // so parallel adds cause version conflicts — mirror checkout behavior).
      for (const item of items) {
        await orderApi.addItem(
          orderId,
          {
            product_id: item.product_id,
            variant_id: item.variant_id || '',
            sku: item.sku,
            name: item.name,
            quantity: item.quantity,
            unit_price: item.unit_price,
          },
          tenantId,
          token || undefined,
        );
      }

      // Mark confirmed (POS sale is already paid — no pending step).
      if (user) {
        try {
          await orderApi.confirm(orderId, user.id, tenantId, token || undefined);
        } catch {
          // Confirm failure shouldn't block the sale — the order still exists.
        }
      }

      void orderNotes; // notes are derived for the receipt; backend doesn't store them yet

      // Build a synthetic Order to drive the Receipt component. We mostly
      // know everything client-side already; we don't refetch the order
      // here to keep the post-submit UX snappy (one extra round-trip would
      // delay the receipt screen). If the backend projection ever lags,
      // this also sidesteps that race.
      const customerName = walkIn
        ? 'Walk-in customer'
        : selectedCustomer
        ? `${selectedCustomer.first_name} ${selectedCustomer.last_name}`
        : 'Walk-in customer';
      const customerEmail = walkIn ? '' : selectedCustomer?.email || '';
      const nowIso = new Date().toISOString();
      const orderItemsSnapshot = items.map((it, idx) => ({
        id: `${orderId}-${idx}`,
        product_id: it.product_id,
        variant_id: it.variant_id || '',
        sku: it.sku,
        name: it.name,
        quantity: it.quantity,
        unit_price: it.unit_price,
        total_price: it.unit_price * it.quantity,
      }));
      const syntheticOrder: Order = {
        id: orderId,
        tenant_id: tenantId,
        customer_id: selectedCustomer?.id || '',
        order_number: orderId.slice(0, 8).toUpperCase(),
        status: 'confirmed',
        currency: 'BDT',
        items: orderItemsSnapshot,
        subtotal,
        shipping_cost: 0,
        tax: 0,
        total,
        shipping_address: placeholderAddress,
        billing_address: placeholderAddress,
        tracking_number: null,
        carrier: null,
        created_at: nowIso,
        updated_at: nowIso,
      };
      setCreatedOrder(syntheticOrder);
      setCompletedSnapshot({
        items,
        subtotal,
        discount: discountAmount,
        total,
        amountReceived: paymentMethod === 'cash' ? cashReceived : undefined,
        paymentMethod,
        customerLabel: customerName,
        customerEmail,
      });
      setRecipientEmail(customerEmail);
      // Load tenant for receipt branding (best-effort — receipt renders fine
      // even if this fails, falling back to "Saajan").
      tenantApi
        .get(tenantId, tenantId)
        .then((t) => setTenant(t))
        .catch(() => setTenant(null));
      toast.success('Sale recorded', { title: 'Done!' });
    } catch (err) {
      const message = err instanceof Error ? err.message : "Couldn't create sale";
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  }

  // ---- Receipt success overlay ----

  function resetForNextSale() {
    setItems([]);
    setSelectedCustomer(null);
    setWalkIn(false);
    setDiscountValue('');
    setNotes('');
    setPaymentMethod('cash');
    setAmountReceived('');
    setCreatedOrder(null);
    setCompletedSnapshot(null);
    setTenant(null);
    setRecipientEmail('');
  }

  async function handleSendEmail() {
    if (!createdOrder || !tenantId) return;
    const email = recipientEmail.trim();
    if (!email) {
      toast.error('Enter an email address first');
      return;
    }
    setSendingEmail(true);
    try {
      await orderApi.sendReceipt(createdOrder.id, email, tenantId, token || undefined);
      toast.success('Receipt sent');
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Couldn't send receipt";
      toast.error(msg);
    } finally {
      setSendingEmail(false);
    }
  }

  // Auto-focus "New sale" 3s after the success view appears so the cashier
  // can press Enter to continue without touching the mouse.
  useEffect(() => {
    if (!createdOrder) return;
    const t = window.setTimeout(() => {
      newSaleRef.current?.focus();
    }, 3000);
    return () => window.clearTimeout(t);
  }, [createdOrder]);

  if (createdOrder && completedSnapshot) {
    const paymentLabel =
      PAYMENT_METHODS.find((p) => p.value === completedSnapshot.paymentMethod)?.label ??
      completedSnapshot.paymentMethod;
    const cashierName = user
      ? [user.first_name, user.last_name].filter(Boolean).join(' ').trim() || user.email
      : '';

    return (
      <>
        {/* Action overlay — hidden when printing */}
        <div className="no-print space-y-6">
          <Link
            href="/admin/orders"
            className="inline-flex items-center gap-1 text-sm text-text-secondary transition-colors hover:text-text"
          >
            <ArrowLeft className="h-4 w-4" /> Back to Orders
          </Link>

          <div className="mx-auto max-w-xl rounded-2xl border border-border bg-surface p-8 shadow-sm">
            <div className="text-center">
              <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-primary-light">
                <CheckCircle2 className="h-9 w-9 text-primary" />
              </div>
              <h2 className="mt-4 text-2xl font-bold text-text">Sale recorded</h2>
              <p className="mt-1 text-sm text-text-secondary">
                Order #{createdOrder.order_number} · {formatCurrency(completedSnapshot.total)} via {paymentLabel}
              </p>
            </div>

            {/* Live preview of the printable receipt */}
            <div className="mt-6 rounded-lg border border-border-light bg-surface-secondary p-4">
              <Receipt
                order={createdOrder}
                tenantBranding={tenant?.config?.branding}
                tenantGeneral={tenant?.config?.general}
                tenantName={tenant?.name}
                cashierName={cashierName || undefined}
                paymentMethod={paymentLabel}
                amountReceived={completedSnapshot.amountReceived}
                customerLabel={completedSnapshot.customerLabel}
              />
            </div>

            {/* Email row */}
            <div className="mt-6 space-y-2">
              <label
                htmlFor="receipt-email"
                className="block text-xs font-semibold uppercase tracking-wider text-text-secondary"
              >
                Email receipt to
              </label>
              <div className="flex flex-wrap gap-2">
                <input
                  id="receipt-email"
                  type="email"
                  value={recipientEmail}
                  onChange={(e) => setRecipientEmail(e.target.value)}
                  placeholder="customer@example.com"
                  className="flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
                <button
                  type="button"
                  onClick={handleSendEmail}
                  disabled={sendingEmail || !recipientEmail.trim()}
                  className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text transition-colors hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {sendingEmail ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Mail className="h-4 w-4" />
                  )}
                  Email to customer
                </button>
              </div>
              {!completedSnapshot.customerEmail && (
                <p className="text-xs text-text-muted">
                  Walk-in sale — enter an email address to send a receipt.
                </p>
              )}
            </div>

            {/* Action buttons */}
            <div className="mt-6 flex flex-wrap justify-center gap-3">
              <button
                type="button"
                onClick={() => window.print()}
                className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text transition-colors hover:bg-surface-hover"
              >
                <Printer className="h-4 w-4" />
                Print
              </button>
              <button
                type="button"
                onClick={() => router.push(`/admin/orders/${createdOrder.id}`)}
                className="inline-flex items-center gap-2 rounded-lg bg-surface px-4 py-2 text-sm font-medium text-text border border-border transition-colors hover:bg-surface-hover"
              >
                View order
              </button>
              <button
                ref={newSaleRef}
                type="button"
                onClick={resetForNextSale}
                className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-primary-dark focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
              >
                <Zap className="h-4 w-4" /> New sale
              </button>
            </div>
          </div>
        </div>

        {/* Print-target — wrapper is hidden on screen, expanded by the
            print stylesheet in globals.css. */}
        <div className="hidden print:block">
          <Receipt
            order={createdOrder}
            tenantBranding={tenant?.config?.branding}
            tenantGeneral={tenant?.config?.general}
            tenantName={tenant?.name}
            cashierName={cashierName || undefined}
            paymentMethod={paymentLabel}
            amountReceived={completedSnapshot.amountReceived}
            customerLabel={completedSnapshot.customerLabel}
          />
        </div>
      </>
    );
  }

  // ---- Main composer view ----

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
            <Zap className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-text">Instant Sell</h1>
            <p className="text-sm text-text-secondary">
              Record in-store, phone, or walk-in sales.
            </p>
          </div>
        </div>
        <Link
          href="/admin/orders"
          className="text-sm text-text-secondary transition-colors hover:text-text"
        >
          Back to Orders
        </Link>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_400px]">
        {/* ===== Left column ===== */}
        <div className="space-y-6">
          {/* Customer */}
          <section className="rounded-xl border border-border bg-surface p-5 shadow-sm">
            <div className="mb-3 flex items-center gap-2">
              <UserIcon className="h-4 w-4 text-text-secondary" />
              <h2 className="text-sm font-semibold uppercase tracking-wider text-text-secondary">
                Customer
              </h2>
            </div>

            {selectedCustomer ? (
              <div className="flex items-start justify-between gap-3 rounded-lg border border-border-light bg-surface-secondary p-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-light text-sm font-semibold text-primary">
                    {selectedCustomer.first_name[0]}
                    {selectedCustomer.last_name[0]}
                  </div>
                  <div>
                    <p className="text-sm font-medium text-text">
                      {selectedCustomer.first_name} {selectedCustomer.last_name}
                    </p>
                    <p className="text-xs text-text-muted">{selectedCustomer.email}</p>
                    {selectedCustomer.phone && (
                      <p className="text-xs text-text-muted">{selectedCustomer.phone}</p>
                    )}
                  </div>
                </div>
                <button
                  onClick={clearCustomer}
                  className="text-xs font-medium text-primary hover:underline"
                >
                  Change
                </button>
              </div>
            ) : walkIn ? (
              <div className="flex items-center justify-between gap-3 rounded-lg border border-border-light bg-surface-secondary p-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-light text-primary">
                    <UserPlus className="h-5 w-5" />
                  </div>
                  <p className="text-sm font-medium text-text">Walk-in customer</p>
                </div>
                <button
                  onClick={clearCustomer}
                  className="text-xs font-medium text-primary hover:underline"
                >
                  Change
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                <div ref={customerSearchRef} className="relative">
                  <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
                  <input
                    value={customerQuery}
                    onChange={(e) => {
                      setCustomerQuery(e.target.value);
                      setCustomerOpen(true);
                    }}
                    onFocus={() => setCustomerOpen(true)}
                    placeholder="Search customer by name, email, phone..."
                    className="w-full rounded-lg border border-border bg-surface py-2.5 pl-9 pr-3 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                  {customerOpen && filteredCustomers.length > 0 && (
                    <div className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-lg border border-border bg-surface shadow-lg">
                      {filteredCustomers.map((c) => (
                        <button
                          key={c.id}
                          onClick={() => selectCustomer(c)}
                          className="flex w-full items-center gap-3 px-3 py-2 text-left transition-colors hover:bg-surface-hover"
                        >
                          <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-semibold text-primary">
                            {c.first_name[0]}
                            {c.last_name[0]}
                          </div>
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-sm font-medium text-text">
                              {c.first_name} {c.last_name}
                            </p>
                            <p className="truncate text-xs text-text-muted">{c.email}</p>
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
                <button
                  onClick={selectWalkIn}
                  className="inline-flex items-center gap-2 rounded-lg border border-dashed border-border px-3 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover hover:text-text"
                >
                  <UserPlus className="h-4 w-4" />
                  Walk-in customer
                </button>
              </div>
            )}
          </section>

          {/* Products */}
          <section className="rounded-xl border border-border bg-surface p-5 shadow-sm">
            <div className="mb-3 flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold uppercase tracking-wider text-text-secondary">
                Products
              </h2>
              <span className="text-xs text-text-muted">
                {items.length} {items.length === 1 ? 'item' : 'items'}
              </span>
            </div>

            <div ref={productSearchRef} className="relative mb-4">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
              <input
                value={productQuery}
                onChange={(e) => {
                  setProductQuery(e.target.value);
                  setProductOpen(true);
                }}
                onFocus={() => setProductOpen(true)}
                placeholder="Search by name or SKU..."
                className="w-full rounded-lg border border-border bg-surface py-2.5 pl-9 pr-3 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              {productOpen && filteredProducts.length > 0 && (
                <div className="absolute z-20 mt-1 max-h-80 w-full overflow-auto rounded-lg border border-border bg-surface shadow-lg">
                  {filteredProducts.map((p) => (
                    <ProductRow key={p.id} product={p} onPick={addProduct} />
                  ))}
                </div>
              )}
              {productOpen && filteredProducts.length === 0 && productQuery && (
                <div className="absolute z-20 mt-1 w-full rounded-lg border border-border bg-surface px-3 py-3 text-sm text-text-muted shadow-lg">
                  No products match &quot;{productQuery}&quot;.
                </div>
              )}
            </div>

            {items.length === 0 ? (
              <div className="rounded-lg border border-dashed border-border px-4 py-10 text-center">
                <ReceiptIcon className="mx-auto h-8 w-8 text-text-muted" />
                <p className="mt-2 text-sm text-text-secondary">No products yet</p>
                <p className="mt-1 text-xs text-text-muted">
                  Search above to add items to the sale.
                </p>
              </div>
            ) : (
              <ul className="space-y-2">
                {items.map((it) => (
                  <li
                    key={it.key}
                    className="flex flex-wrap items-center gap-3 rounded-lg border border-border-light p-3 sm:flex-nowrap"
                  >
                    <div className="h-12 w-12 flex-shrink-0 overflow-hidden rounded-lg bg-surface-secondary">
                      {it.image ? (
                        <Image
                          src={mediaUrl(it.image)}
                          alt={it.name}
                          width={48}
                          height={48}
                          className="h-full w-full object-cover"
                          unoptimized
                        />
                      ) : (
                        <div className="flex h-full w-full items-center justify-center text-text-muted">
                          <ReceiptIcon className="h-4 w-4" />
                        </div>
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-text">{it.name}</p>
                      <p className="text-xs text-text-muted">SKU {it.sku}</p>
                    </div>
                    <div className="flex items-center gap-1.5 rounded-lg border border-border">
                      <button
                        type="button"
                        onClick={() => updateQty(it.key, it.quantity - 1)}
                        className="p-1.5 text-text-secondary transition-colors hover:bg-surface-hover hover:text-text"
                        aria-label="Decrease"
                      >
                        <Minus className="h-3.5 w-3.5" />
                      </button>
                      <input
                        type="number"
                        min={1}
                        value={it.quantity}
                        onChange={(e) => updateQty(it.key, Number(e.target.value))}
                        className="w-10 border-x border-border bg-transparent py-1 text-center text-sm text-text focus:outline-none"
                      />
                      <button
                        type="button"
                        onClick={() => updateQty(it.key, it.quantity + 1)}
                        className="p-1.5 text-text-secondary transition-colors hover:bg-surface-hover hover:text-text"
                        aria-label="Increase"
                      >
                        <Plus className="h-3.5 w-3.5" />
                      </button>
                    </div>
                    <div className="flex flex-col">
                      <label className="text-[10px] uppercase tracking-wider text-text-muted">
                        Unit price
                      </label>
                      <input
                        type="number"
                        min={0}
                        step="0.01"
                        value={it.unit_price}
                        onChange={(e) => updatePrice(it.key, Number(e.target.value))}
                        className="w-24 rounded border border-border bg-surface px-2 py-1 text-sm text-text focus:border-primary focus:outline-none"
                      />
                    </div>
                    <div className="text-right">
                      <p className="text-[10px] uppercase tracking-wider text-text-muted">
                        Total
                      </p>
                      <p className="text-sm font-semibold text-text">
                        {formatCurrency(it.unit_price * it.quantity)}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => removeItem(it.key)}
                      className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                      aria-label="Remove"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>

          {/* Discount + notes */}
          <section className="rounded-xl border border-border bg-surface p-5 shadow-sm">
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-text-secondary">
              Discount &amp; notes
            </h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-text-secondary">
                  Discount
                </label>
                <div className="flex gap-2">
                  <select
                    value={discountMode}
                    onChange={(e) => setDiscountMode(e.target.value as DiscountMode)}
                    className="rounded-lg border border-border bg-surface px-2 py-2 text-sm text-text focus:border-primary focus:outline-none"
                  >
                    <option value="fixed">BDT</option>
                    <option value="percent">%</option>
                  </select>
                  <input
                    type="number"
                    min={0}
                    step="0.01"
                    value={discountValue}
                    onChange={(e) => setDiscountValue(e.target.value)}
                    placeholder="0"
                    className="flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-text-secondary">
                  Admin notes (optional)
                </label>
                <textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  rows={2}
                  placeholder="Internal reference..."
                  className="w-full resize-none rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>
          </section>
        </div>

        {/* ===== Right column (summary + payment) ===== */}
        <aside className="space-y-6 lg:sticky lg:top-6 lg:self-start">
          <section className="rounded-xl border border-border bg-surface p-5 shadow-sm">
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-text-secondary">
              Summary
            </h2>
            {items.length === 0 ? (
              <p className="text-sm text-text-muted">
                Add products to see totals.
              </p>
            ) : (
              <ul className="mb-4 space-y-1.5 border-b border-border-light pb-3">
                {items.map((it) => (
                  <li key={it.key} className="flex justify-between gap-3 text-xs">
                    <span className="min-w-0 flex-1 truncate text-text-secondary">
                      {it.quantity} × {it.name}
                    </span>
                    <span className="text-text">
                      {formatCurrency(it.unit_price * it.quantity)}
                    </span>
                  </li>
                ))}
              </ul>
            )}

            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-text-secondary">Subtotal</span>
                <span className="text-text">{formatCurrency(subtotal)}</span>
              </div>
              {discountAmount > 0 && (
                <div className="flex justify-between">
                  <span className="text-text-secondary">Discount</span>
                  <span className="text-accent">
                    −{formatCurrency(discountAmount)}
                  </span>
                </div>
              )}
            </div>

            <div className="mt-3 border-t border-border pt-3">
              <div className="flex items-baseline justify-between">
                <span className="text-xs font-medium uppercase tracking-wider text-text-secondary">
                  Total
                </span>
                <span className="text-3xl font-bold text-primary">
                  {formatCurrency(total)}
                </span>
              </div>
            </div>
          </section>

          <section className="rounded-xl border border-border bg-surface p-5 shadow-sm">
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-text-secondary">
              Payment
            </h2>
            <div className="grid grid-cols-2 gap-2">
              {PAYMENT_METHODS.map((p) => (
                <button
                  key={p.value}
                  type="button"
                  onClick={() => setPaymentMethod(p.value)}
                  className={cn(
                    'rounded-lg border px-3 py-2 text-sm font-medium transition-colors',
                    paymentMethod === p.value
                      ? 'border-primary bg-primary-light text-primary'
                      : 'border-border bg-surface text-text-secondary hover:bg-surface-hover',
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>

            {paymentMethod === 'cash' && (
              <div className="mt-4 space-y-3 rounded-lg border border-border-light bg-surface-secondary p-3">
                <div>
                  <label className="mb-1 block text-xs font-medium text-text-secondary">
                    Amount received
                  </label>
                  <input
                    type="number"
                    min={0}
                    step="0.01"
                    value={amountReceived}
                    onChange={(e) => setAmountReceived(e.target.value)}
                    placeholder="0"
                    className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-text-secondary">Change</span>
                  <span
                    className={cn(
                      'font-semibold',
                      change > 0
                        ? 'text-green-600 dark:text-green-400'
                        : change < 0
                        ? 'text-accent'
                        : 'text-text',
                    )}
                  >
                    {formatCurrency(Math.max(0, change))}
                  </span>
                </div>
              </div>
            )}

            <button
              type="button"
              onClick={handleCreate}
              disabled={!canSubmit}
              className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-primary-dark disabled:cursor-not-allowed disabled:opacity-50"
            >
              {submitting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Creating sale...
                </>
              ) : (
                <>
                  <Zap className="h-4 w-4" />
                  Create sale · {formatCurrency(total)}
                </>
              )}
            </button>
            {items.length === 0 && (
              <p className="mt-2 text-center text-xs text-text-muted">
                Add at least one product to continue.
              </p>
            )}
            {paymentMethod === 'cash' && cashReceived < total && items.length > 0 && (
              <p className="mt-2 text-center text-xs text-accent">
                Cash received is less than total.
              </p>
            )}
          </section>
        </aside>
      </div>
    </div>
  );
}

// ---- Inline subcomponents ----

function ProductRow({ product, onPick }: { product: Product; onPick: (p: Product, variantId?: string) => void }) {
  const hasVariants = (product.variants?.length ?? 0) > 0;
  const [showVariants, setShowVariants] = useState(false);

  if (hasVariants && showVariants) {
    return (
      <div className="border-b border-border-light last:border-b-0">
        <div className="flex items-center gap-3 px-3 py-2">
          <button
            onClick={() => setShowVariants(false)}
            className="text-xs text-text-secondary hover:text-text"
          >
            <X className="h-3.5 w-3.5" />
          </button>
          <p className="text-xs font-medium text-text-secondary">
            Pick a variant of {product.name}
          </p>
        </div>
        {product.variants?.map((v) => {
          const stock = typeof v.stock === 'number' ? v.stock : null;
          const outOfStock = stock !== null && stock <= 0;
          return (
            <button
              key={v.id || v.sku}
              onClick={() => !outOfStock && onPick(product, v.id || '')}
              disabled={outOfStock}
              className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left transition-colors hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-50"
            >
              <div className="min-w-0">
                <p className="truncate text-sm text-text">{v.name}</p>
                <p className="text-xs text-text-muted">SKU {v.sku}</p>
              </div>
              <div className="text-right">
                <p className="text-sm font-medium text-text">{formatCurrency(v.price)}</p>
                <p className="text-[10px] text-text-muted">
                  {stock !== null ? `${stock} in stock` : 'Stock unknown'}
                </p>
              </div>
            </button>
          );
        })}
      </div>
    );
  }

  return (
    <button
      onClick={() => (hasVariants ? setShowVariants(true) : onPick(product))}
      className="flex w-full items-center gap-3 border-b border-border-light px-3 py-2 text-left transition-colors last:border-b-0 hover:bg-surface-hover"
    >
      <div className="h-10 w-10 flex-shrink-0 overflow-hidden rounded-lg bg-surface-secondary">
        {product.images?.[0] ? (
          <Image
            src={mediaUrl(product.images[0])}
            alt={product.name}
            width={40}
            height={40}
            className="h-full w-full object-cover"
            unoptimized
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-text-muted">
            <ReceiptIcon className="h-4 w-4" />
          </div>
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-text">{product.name}</p>
        <p className="text-xs text-text-muted">SKU {product.sku}</p>
      </div>
      <div className="text-right">
        <p className="text-sm font-medium text-text">{formatCurrency(product.price)}</p>
        {hasVariants && (
          <p className="text-[10px] text-text-muted">
            {product.variants?.length} variants
          </p>
        )}
      </div>
    </button>
  );
}

