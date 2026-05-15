'use client';

import { use, useEffect, useState } from 'react';
import Link from 'next/link';
import {
  CheckCircle,
  Package,
  MapPin,
  CreditCard,
  ArrowRight,
  Clock,
  Loader2,
  Truck,
  FileText,
} from 'lucide-react';
import { orderApi, tenantApi, type Order } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { DEFAULT_TENANT_ID as TENANT_ID } from '@/lib/tenant';

const STATUS_STEPS = ['pending', 'confirmed', 'shipped', 'delivered'];

function StatusTimeline({ status }: { status: string }) {
  const currentIdx = STATUS_STEPS.indexOf(status);
  if (status === 'cancelled') {
    return (
      <div className="rounded-lg bg-red-50 px-4 py-2 text-sm font-medium text-red-700">
        Order Cancelled
      </div>
    );
  }
  return (
    <div className="flex items-center gap-1">
      {STATUS_STEPS.map((step, idx) => {
        const done = idx <= currentIdx;
        const active = idx === currentIdx;
        return (
          <div key={step} className="flex flex-1 items-center">
            <div className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold transition-colors ${
              done ? 'bg-primary text-white' : 'bg-surface-hover text-text-muted'
            } ${active ? 'ring-2 ring-primary ring-offset-1' : ''}`}>
              {idx + 1}
            </div>
            {idx < STATUS_STEPS.length - 1 && (
              <div className={`h-0.5 flex-1 transition-colors ${done && idx < currentIdx ? 'bg-primary' : 'bg-border'}`} />
            )}
          </div>
        );
      })}
    </div>
  );
}

export default function OrderConfirmationPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const token = useAuthStore((s) => s.token);
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  // Tenant-level feature flag. We default to false so we never flash an
  // invoice link before knowing whether the feature is enabled.
  const [invoicesEnabled, setInvoicesEnabled] = useState(false);

  useEffect(() => {
    orderApi.get(id, TENANT_ID, token || undefined)
      .then(setOrder)
      .catch(() => setOrder(null))
      .finally(() => setLoading(false));
  }, [id, token]);

  useEffect(() => {
    let cancelled = false;
    tenantApi
      .get(TENANT_ID)
      .then((t) => {
        if (!cancelled) setInvoicesEnabled(t.config?.features?.invoices_enabled === true);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  // If API fetch failed (order not in system yet / guest order), show confirmation
  // with whatever ID we have
  const orderId = order?.order_number ?? (id.startsWith('ORD-') ? id : `ORD-${id.slice(0, 8).toUpperCase()}`);

  return (
    <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Success banner */}
      <div className="rounded-xl bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-6 text-center">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/40">
          <CheckCircle className="h-8 w-8 text-green-600 dark:text-green-400" />
        </div>
        <h1 className="mt-4 text-2xl font-bold text-green-900 dark:text-green-200">
          Order Placed Successfully!
        </h1>
        <p className="mt-1 text-green-700 dark:text-green-300">
          Thank you for your purchase. We will process your order shortly.
        </p>
      </div>

      {/* Order details */}
      <div className="mt-8 rounded-xl border border-border bg-surface overflow-hidden">
        {/* Order header */}
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-border px-6 py-4">
          <div>
            <p className="text-xs text-text-secondary uppercase tracking-wide">Order Number</p>
            <p className="text-lg font-bold text-text">{orderId}</p>
          </div>
          {order && (
            <div>
              <p className="text-xs text-text-secondary uppercase tracking-wide">Date</p>
              <div className="flex items-center gap-1 text-sm text-text-secondary">
                <Clock className="h-3.5 w-3.5" />
                {formatDate(order.created_at)}
              </div>
            </div>
          )}
          {order && (
            <span className={`rounded-full px-3 py-1 text-xs font-semibold capitalize ${statusColor(order.status)}`}>
              {order.status}
            </span>
          )}
        </div>

        {/* Status timeline */}
        {order && (
          <div className="px-6 py-4 border-b border-border">
            <div className="mb-3 flex items-center gap-2 text-sm font-medium text-text-secondary">
              <Truck className="h-4 w-4" />
              Order Progress
            </div>
            <StatusTimeline status={order.status} />
            <div className="mt-2 flex justify-between text-xs text-text-muted">
              {STATUS_STEPS.map((s) => (
                <span key={s} className="capitalize">{s}</span>
              ))}
            </div>
            {order.tracking_number && (
              <p className="mt-3 text-sm text-text-secondary">
                Tracking: <span className="font-medium text-text">{order.tracking_number}</span>
                {order.carrier && <span className="ml-1 text-text-muted">via {order.carrier}</span>}
              </p>
            )}
          </div>
        )}

        {/* Order items */}
        <div className="px-6 py-4">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-text">
            <Package className="h-4 w-4" />
            Items
          </h2>
          {order && order.items && order.items.length > 0 ? (
            <div className="mt-3 divide-y divide-border-light">
              {order.items.map((item) => (
                <div key={item.id} className="flex items-center gap-4 py-3">
                  <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-primary/20 to-primary/5">
                    <span className="text-lg font-bold text-primary/40">
                      {item.name.charAt(0)}
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-text">{item.name}</p>
                    <p className="text-xs text-text-secondary">
                      SKU: {item.sku} &middot; Qty: {item.quantity}
                    </p>
                  </div>
                  <span className="font-medium text-text">
                    {formatCurrency(item.unit_price * item.quantity)}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-3 text-sm text-text-secondary">
              Items will appear once the order is confirmed.
            </p>
          )}

          {/* Totals */}
          {order && (
            <div className="mt-4 space-y-2 border-t border-border pt-4">
              {order.subtotal > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-text-secondary">Subtotal</span>
                  <span className="text-text">{formatCurrency(order.subtotal)}</span>
                </div>
              )}
              {order.shipping_cost > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-text-secondary">Shipping</span>
                  <span className="text-text">{formatCurrency(order.shipping_cost)}</span>
                </div>
              )}
              {order.tax > 0 && (
                <div className="flex justify-between text-sm">
                  <span className="text-text-secondary">Tax</span>
                  <span className="text-text">{formatCurrency(order.tax)}</span>
                </div>
              )}
              <div className="flex justify-between border-t border-border pt-2">
                <span className="font-semibold text-text">Total</span>
                <span className="text-lg font-bold text-text">
                  {formatCurrency(order.total)}
                </span>
              </div>
            </div>
          )}
        </div>

        {/* Shipping & Payment info */}
        {order && (
          <div className="grid grid-cols-1 gap-px bg-border sm:grid-cols-2">
            {order.shipping_address && (
            <div className="bg-surface px-6 py-4">
              <h3 className="flex items-center gap-2 text-sm font-semibold text-text">
                <MapPin className="h-4 w-4" />
                Shipping Address
              </h3>
              <div className="mt-2 space-y-1 text-sm text-text-secondary">
                <p>{order.shipping_address.street}</p>
                <p>
                  {order.shipping_address.city}, {order.shipping_address.postal_code}
                </p>
                <p>{order.shipping_address.country}</p>
              </div>
            </div>
            )}
            <div className="bg-surface px-6 py-4">
              <h3 className="flex items-center gap-2 text-sm font-semibold text-text">
                <CreditCard className="h-4 w-4" />
                Order ID
              </h3>
              <p className="mt-2 font-mono text-xs text-text-secondary break-all">{id}</p>
            </div>
          </div>
        )}
      </div>

      {/* CTA */}
      <div className="mt-8 flex flex-wrap items-center justify-center gap-4">
        {invoicesEnabled && order && (
          <Link
            href={`/orders/${order.id}/invoice`}
            className="inline-flex items-center gap-2 rounded-lg border border-border px-6 py-3 font-medium text-text-secondary transition-colors hover:border-primary/40 hover:bg-surface-hover"
          >
            <FileText className="h-4 w-4" />
            Download invoice
          </Link>
        )}
        <Link
          href="/account/orders"
          className="inline-flex items-center gap-2 rounded-lg border border-border px-6 py-3 font-medium text-text-secondary transition-colors hover:border-primary/40 hover:bg-surface-hover"
        >
          View All Orders
        </Link>
        <Link
          href="/products"
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark"
        >
          Continue Shopping
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    </div>
  );
}
