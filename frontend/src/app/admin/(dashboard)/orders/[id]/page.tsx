'use client';

import { use, useState, useEffect } from 'react';
import Link from 'next/link';
import { ArrowLeft, Loader2, X } from 'lucide-react';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { orderApi, type Order } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

const demoOrder = {
  id: 'ORD-2026-002',
  tenant_id: 'tenant_saajan',
  customer_id: 'cu-002',
  order_number: 'ORD-2026-002',
  status: 'shipped' as const,
  currency: 'BDT',
  items: [
    { id: '1', product_id: 'p1', variant_id: 'v1', sku: 'SAR-JAM-001', name: 'Jamdani Saree', quantity: 1, unit_price: 15000, total_price: 15000 },
  ],
  subtotal: 15000,
  shipping_cost: 120,
  tax: 0,
  total: 15120,
  shipping_address: { street: '45 Gulshan Avenue, Road 12', city: 'Dhaka', state: 'Dhaka Division', postal_code: '1212', country: 'Bangladesh' },
  billing_address: { street: '45 Gulshan Avenue, Road 12', city: 'Dhaka', state: 'Dhaka Division', postal_code: '1212', country: 'Bangladesh' },
  tracking_number: 'SA-BD-78542136',
  carrier: 'Sundarban Courier',
  created_at: '2026-04-17T10:30:00Z',
  updated_at: '2026-04-17T10:30:00Z',
};

export default function OrderDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { user, tenantId } = useAuthStore();
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [showShipDialog, setShowShipDialog] = useState(false);
  const [showCancelDialog, setShowCancelDialog] = useState(false);
  const [trackingNumber, setTrackingNumber] = useState('');
  const [carrier, setCarrier] = useState('');
  const [cancelReason, setCancelReason] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    async function loadOrder() {
      if (!tenantId) {
        setOrder({ ...demoOrder, id, order_number: id });
        setLoading(false);
        return;
      }
      try {
        const data = await orderApi.get(id, tenantId);
        setOrder(data);
      } catch {
        setOrder({ ...demoOrder, id, order_number: id });
      } finally {
        setLoading(false);
      }
    }
    loadOrder();
  }, [id, tenantId]);

  async function handleConfirm() {
    if (!order || !tenantId || !user) return;
    setActionLoading(true);
    setError('');
    try {
      const updated = await orderApi.confirm(order.id, user.id, tenantId);
      setOrder(updated);
    } catch (err) {
      setError((err as Error).message || 'Failed to confirm order');
    } finally {
      setActionLoading(false);
    }
  }

  async function handleShip() {
    if (!order || !tenantId || !user || !trackingNumber.trim() || !carrier.trim()) return;
    setActionLoading(true);
    setError('');
    try {
      const updated = await orderApi.ship(order.id, {
        tracking_number: trackingNumber.trim(),
        carrier: carrier.trim(),
        shipped_by: user.id,
      }, tenantId);
      setOrder(updated);
      setShowShipDialog(false);
      setTrackingNumber('');
      setCarrier('');
    } catch (err) {
      setError((err as Error).message || 'Failed to ship order');
    } finally {
      setActionLoading(false);
    }
  }

  async function handleCancel() {
    if (!order || !tenantId || !user || !cancelReason.trim()) return;
    setActionLoading(true);
    setError('');
    try {
      const updated = await orderApi.cancel(order.id, cancelReason.trim(), user.id, tenantId);
      setOrder(updated);
      setShowCancelDialog(false);
      setCancelReason('');
    } catch (err) {
      setError((err as Error).message || 'Failed to cancel order');
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="py-16 text-center">
        <Loader2 className="mx-auto h-6 w-6 animate-spin text-primary" />
        <p className="mt-2 text-sm text-text-muted">Loading order...</p>
      </div>
    );
  }

  if (!order) {
    return (
      <div className="space-y-4">
        <Link href="/admin/orders" className="inline-flex items-center gap-1 text-sm text-text-secondary hover:text-text">
          <ArrowLeft className="h-4 w-4" /> Back to Orders
        </Link>
        <div className="rounded-xl border border-border bg-surface p-16 text-center">
          <p className="text-text-secondary">Order not found.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link href="/admin/orders" className="inline-flex items-center gap-1 text-sm text-text-secondary transition-colors hover:text-text">
        <ArrowLeft className="h-4 w-4" /> Back to Orders
      </Link>

      {error && (
        <div className="rounded-lg bg-red-50 dark:bg-red-900/20 px-4 py-3 text-sm text-red-700 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Order Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-bold text-text">{order.order_number}</h1>
          <span className={cn('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(order.status))}>
            {order.status}
          </span>
        </div>
        <div className="flex gap-3">
          {order.status === 'pending' && (
            <>
              <button onClick={handleConfirm} disabled={actionLoading}
                className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-50">
                {actionLoading ? 'Processing...' : 'Confirm Order'}
              </button>
              <button onClick={() => setShowCancelDialog(true)} disabled={actionLoading}
                className="rounded-lg border border-red-300 dark:border-red-700 px-4 py-2 text-sm font-medium text-red-600 dark:text-red-400 transition-colors hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50">
                Cancel Order
              </button>
            </>
          )}
          {order.status === 'confirmed' && (
            <button onClick={() => setShowShipDialog(true)} disabled={actionLoading}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-50">
              Mark as Shipped
            </button>
          )}
          {order.status === 'shipped' && (
            <button onClick={async () => {
              // For "delivered" there's no dedicated API, but we can extend later
              setError('Mark as delivered is handled by the delivery confirmation system.');
            }} disabled={actionLoading}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-50">
              Mark as Delivered
            </button>
          )}
        </div>
      </div>

      {/* Two columns */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          {/* Order Items */}
          <div className="rounded-xl border border-border bg-surface shadow-sm">
            <div className="border-b border-border px-6 py-4">
              <h2 className="text-lg font-semibold text-text">Order Items</h2>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border text-left text-sm text-text-secondary">
                    <th className="px-6 py-3 font-medium">Product</th>
                    <th className="px-6 py-3 font-medium">SKU</th>
                    <th className="px-6 py-3 font-medium">Qty</th>
                    <th className="px-6 py-3 font-medium">Unit Price</th>
                    <th className="px-6 py-3 font-medium text-right">Total</th>
                  </tr>
                </thead>
                <tbody>
                  {order.items.map((item) => (
                    <tr key={item.id} className="border-b border-border-light transition-colors hover:bg-surface-hover">
                      <td className="px-6 py-4 text-sm font-medium text-text">{item.name}</td>
                      <td className="px-6 py-4 text-sm text-text-secondary font-mono">{item.sku}</td>
                      <td className="px-6 py-4 text-sm text-text">{item.quantity}</td>
                      <td className="px-6 py-4 text-sm text-text">{formatCurrency(item.unit_price)}</td>
                      <td className="px-6 py-4 text-sm text-text text-right">{formatCurrency(item.total_price)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Addresses */}
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
            <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
              <h3 className="mb-3 text-sm font-semibold text-text">Shipping Address</h3>
              <div className="space-y-1 text-sm text-text-secondary">
                <p>{order.shipping_address.street}</p>
                <p>{order.shipping_address.city}, {order.shipping_address.state}</p>
                <p>{order.shipping_address.postal_code}, {order.shipping_address.country}</p>
              </div>
            </div>
            <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
              <h3 className="mb-3 text-sm font-semibold text-text">Billing Address</h3>
              <div className="space-y-1 text-sm text-text-secondary">
                <p>{order.billing_address.street}</p>
                <p>{order.billing_address.city}, {order.billing_address.state}</p>
                <p>{order.billing_address.postal_code}, {order.billing_address.country}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Right sidebar */}
        <div className="space-y-6">
          <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-text">Order Summary</h2>
            <div className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">Subtotal</span>
                <span className="text-text">{formatCurrency(order.subtotal)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">Shipping</span>
                <span className="text-text">{formatCurrency(order.shipping_cost)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-text-secondary">Tax</span>
                <span className="text-text">{formatCurrency(order.tax)}</span>
              </div>
              <div className="border-t border-border pt-3">
                <div className="flex justify-between">
                  <span className="font-semibold text-text">Total</span>
                  <span className="font-semibold text-text">{formatCurrency(order.total)}</span>
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-text">Customer</h2>
            <div className="space-y-2 text-sm">
              <p className="font-medium text-text">{order.customer_id}</p>
            </div>
          </div>

          {order.tracking_number && (
            <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
              <h2 className="mb-4 text-lg font-semibold text-text">Shipping</h2>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-text-secondary">Carrier</span>
                  <span className="text-text">{order.carrier}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-text-secondary">Tracking #</span>
                  <span className="font-mono text-text">{order.tracking_number}</span>
                </div>
              </div>
            </div>
          )}

          <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-text">Timeline</h2>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-text-secondary">Order placed</span>
                <span className="text-text">{formatDate(order.created_at.split('T')[0])}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Ship Dialog */}
      {showShipDialog && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => setShowShipDialog(false)} />
          <div className="relative w-full max-w-md rounded-2xl border border-border bg-surface p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-text">Ship Order</h3>
              <button onClick={() => setShowShipDialog(false)} className="rounded-lg p-1.5 text-text-muted hover:bg-surface-hover">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-text-secondary">Carrier</label>
                <input value={carrier} onChange={(e) => setCarrier(e.target.value)} placeholder="e.g. Sundarban Courier"
                  className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-text-secondary">Tracking Number</label>
                <input value={trackingNumber} onChange={(e) => setTrackingNumber(e.target.value)} placeholder="e.g. SA-BD-78542136"
                  className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button onClick={() => setShowShipDialog(false)}
                  className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-text-secondary hover:bg-surface-hover transition-colors">
                  Cancel
                </button>
                <button onClick={handleShip} disabled={actionLoading || !carrier.trim() || !trackingNumber.trim()}
                  className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-dark disabled:opacity-50 transition-colors">
                  {actionLoading ? 'Shipping...' : 'Confirm Shipment'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Cancel Dialog */}
      {showCancelDialog && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={() => setShowCancelDialog(false)} />
          <div className="relative w-full max-w-md rounded-2xl border border-border bg-surface p-6 shadow-2xl">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-text">Cancel Order</h3>
              <button onClick={() => setShowCancelDialog(false)} className="rounded-lg p-1.5 text-text-muted hover:bg-surface-hover">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-text-secondary">Reason for cancellation</label>
                <textarea value={cancelReason} onChange={(e) => setCancelReason(e.target.value)} rows={3} placeholder="Enter reason..."
                  className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button onClick={() => setShowCancelDialog(false)}
                  className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-text-secondary hover:bg-surface-hover transition-colors">
                  Go Back
                </button>
                <button onClick={handleCancel} disabled={actionLoading || !cancelReason.trim()}
                  className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50 transition-colors">
                  {actionLoading ? 'Cancelling...' : 'Cancel Order'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
