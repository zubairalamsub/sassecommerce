'use client';

/**
 * Receipt — the leaner cousin of <Invoice />.
 *
 * Where the invoice is an A4 billing document, the receipt is what the
 * cashier hands the customer at the counter. Designed for 80 mm thermal
 * paper, but it scales gracefully on screen too (uses a 320–384 px wrapper
 * so it preview-renders comfortably inside a success card).
 *
 * The print stylesheet that backs `.receipt-print` lives in
 * `src/app/globals.css` and intentionally co-exists with the invoice's
 * `.invoice-print` rules — both selectors hide page chrome on print but
 * use different `@page` sizes (A4 vs. 80 mm thermal).
 */

import { mediaUrl, formatCurrency, cn } from '@/lib/utils';
import type { Order, TenantConfig } from '@/lib/api';

interface ReceiptProps {
  order: Order;
  /** Branding/general config blocks; address & contact pulled from `general`. */
  tenantBranding?: TenantConfig['branding'];
  /** Tenant general info — address, phone, support URL. Optional. */
  tenantGeneral?: TenantConfig['general'];
  /** Tenant display name — falls back to "Saajan". */
  tenantName?: string;
  /** Cashier name to print on the receipt; defaults to "—". */
  cashierName?: string;
  /** Human-readable payment method label (e.g. "Cash", "bKash"). */
  paymentMethod: string;
  /** Optional — only meaningful for cash payments. */
  amountReceived?: number;
  /** Customer label override (e.g. "Walk-in") when order has no customer info. */
  customerLabel?: string;
  className?: string;
}

/** Format a short receipt number from the order's id/order_number. */
function shortRef(order: Order): string {
  const raw = order.order_number || order.id || '';
  // Strip non-alnum, take the last 8 chars, uppercased.
  const cleaned = raw.replace(/[^A-Za-z0-9]/g, '').toUpperCase();
  return cleaned.slice(-8) || 'RECEIPT';
}

/** Compact date+time, e.g. "May 13, 2026 · 3:42 PM". */
function formatReceiptTimestamp(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const date = d.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
  const time = d.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
  return `${date} · ${time}`;
}

/**
 * Heuristic discount: order-service doesn't yet store discount as a column,
 * so we infer it the same way Invoice does — subtotal+shipping+tax − total.
 */
function discountAmount(order: Order): number {
  const expected = (order.subtotal ?? 0) + (order.shipping_cost ?? 0) + (order.tax ?? 0);
  const diff = expected - (order.total ?? 0);
  return diff > 0.01 ? diff : 0;
}

export default function Receipt({
  order,
  tenantBranding,
  tenantGeneral,
  tenantName,
  cashierName,
  paymentMethod,
  amountReceived,
  customerLabel,
  className,
}: ReceiptProps) {
  const storeName = tenantName?.trim() || 'Saajan';
  const logoUrl = tenantBranding?.logo_url ? mediaUrl(tenantBranding.logo_url) : '';
  const phone = tenantGeneral?.contact_phone?.trim() || '';
  const supportUrl = tenantGeneral?.support_url?.trim() || '';

  // Address line — prefer tenant general (when populated later), otherwise
  // fall back to the order's shipping address city/country so we don't print
  // an empty header.
  const addressLine = order.shipping_address?.city
    ? [order.shipping_address.city, order.shipping_address.country]
        .filter(Boolean)
        .join(', ')
    : '';

  const discount = discountAmount(order);
  const change =
    typeof amountReceived === 'number' && amountReceived > 0
      ? Math.max(0, amountReceived - order.total)
      : 0;

  const customer =
    customerLabel?.trim() ||
    (order.customer_id ? order.customer_id : 'Walk-in customer');

  return (
    <div
      className={cn(
        'receipt-print mx-auto w-full max-w-sm bg-white px-4 py-5 text-[12px] leading-tight text-gray-900 shadow-sm print:max-w-none print:shadow-none',
        className,
      )}
      style={{ fontFamily: '"Courier New", ui-monospace, monospace' }}
    >
      {/* ── Header ─────────────────────────────────────────────── */}
      <div className="text-center">
        {logoUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={logoUrl}
            alt={storeName}
            className="mx-auto mb-1 h-10 w-auto object-contain"
          />
        ) : null}
        <p className="text-base font-bold uppercase tracking-wide text-primary">
          {storeName}
        </p>
        {addressLine && <p className="mt-0.5 text-[11px] text-gray-700">{addressLine}</p>}
        {phone && <p className="text-[11px] text-gray-700">{phone}</p>}
        {supportUrl && <p className="text-[11px] text-gray-700">{supportUrl}</p>}
      </div>

      <Divider />

      {/* ── Meta ──────────────────────────────────────────────── */}
      <div className="space-y-0.5">
        <p>
          <span className="font-semibold">RECEIPT #</span>
          {shortRef(order)}
        </p>
        <p>{formatReceiptTimestamp(order.created_at || new Date().toISOString())}</p>
        <p>Cashier: {cashierName?.trim() || '—'}</p>
        <p>Customer: {customer}</p>
      </div>

      <Divider />

      {/* ── Line items ────────────────────────────────────────── */}
      <div className="space-y-1">
        {order.items.length === 0 ? (
          <p className="text-center text-gray-500">No items.</p>
        ) : (
          order.items.map((item) => {
            const lineTotal = item.total_price || item.unit_price * item.quantity;
            return (
              <div
                key={item.id || `${item.product_id}-${item.sku}`}
                className="flex items-start justify-between gap-2"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate">
                    {item.quantity} × {item.name}
                  </p>
                  {item.sku && (
                    <p className="text-[10px] text-gray-500">SKU {item.sku}</p>
                  )}
                </div>
                <span className="shrink-0 tabular-nums">
                  {formatCurrency(lineTotal, order.currency)}
                </span>
              </div>
            );
          })
        )}
      </div>

      <Divider />

      {/* ── Totals ────────────────────────────────────────────── */}
      <div className="space-y-0.5">
        <Row label="Subtotal" value={formatCurrency(order.subtotal, order.currency)} />
        {discount > 0 && (
          <Row
            label="Discount"
            value={`-${formatCurrency(discount, order.currency)}`}
          />
        )}
        {order.shipping_cost > 0 && (
          <Row
            label="Shipping"
            value={formatCurrency(order.shipping_cost, order.currency)}
          />
        )}
        {order.tax > 0 && (
          <Row label="Tax" value={formatCurrency(order.tax, order.currency)} />
        )}
        <div className="mt-1 flex items-baseline justify-between gap-2 border-t border-dashed border-gray-400 pt-1">
          <span className="text-sm font-bold uppercase">Total</span>
          <span className="text-base font-bold tabular-nums text-primary">
            {formatCurrency(order.total, order.currency)}
          </span>
        </div>
      </div>

      <Divider />

      {/* ── Payment ───────────────────────────────────────────── */}
      <div className="space-y-0.5">
        <Row label="Payment" value={paymentMethod} />
        {typeof amountReceived === 'number' && amountReceived > 0 && (
          <>
            <Row
              label="Received"
              value={formatCurrency(amountReceived, order.currency)}
            />
            <Row label="Change" value={formatCurrency(change, order.currency)} />
          </>
        )}
      </div>

      <Divider />

      {/* ── Footer ────────────────────────────────────────────── */}
      <div className="text-center">
        <p className="font-medium">Thank you for shopping!</p>
        {supportUrl ? (
          <p className="mt-0.5 text-[11px] text-gray-700">Visit {supportUrl}</p>
        ) : (
          <p className="mt-0.5 text-[11px] text-gray-700">Visit saajan.com</p>
        )}
      </div>
    </div>
  );
}

// ── tiny presentational helpers ──────────────────────────────

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <span className="text-gray-700">{label}</span>
      <span className="tabular-nums">{value}</span>
    </div>
  );
}

function Divider() {
  // Dashed border matches thermal-paper aesthetic. Tailwind doesn't ship a
  // 1px-tall dashed divider helper, so we inline the styles.
  return (
    <div
      className="my-2 border-t border-dashed border-gray-400"
      role="separator"
      aria-hidden="true"
    />
  );
}
