'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Package, ArrowLeft, ChevronRight, ShoppingBag, Loader2 } from 'lucide-react';
import AuthGuard from '@/components/auth/auth-guard';
import { useAuthStore } from '@/stores/auth';
import { orderApi, type Order } from '@/lib/api';
import { formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { DEFAULT_TENANT_ID as TENANT_ID } from '@/lib/tenant';

function OrdersContent() {
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!user) return;
    orderApi.listByCustomer(user.id, TENANT_ID, token || undefined)
      .then((res) => setOrders(res.data ?? []))
      .catch((err) => {
        setOrders([]);
        setError(err instanceof Error ? err.message : 'Failed to load orders');
      })
      .finally(() => setLoading(false));
  }, [user?.id, token]);

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
      {/* Header */}
      <div className="mb-8 flex items-center gap-3">
        <Link
          href="/account"
          className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-text-secondary"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-text">My Orders</h1>
          {!loading && (
            <p className="text-sm text-text-secondary">
              {orders.length} {orders.length === 1 ? 'order' : 'orders'} placed
            </p>
          )}
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="mb-6 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 px-4 py-3 text-sm text-red-700 dark:text-red-300">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading ? (
        <div className="flex justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : orders.length === 0 && !error ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-border bg-surface py-16 text-center shadow-sm">
          <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-surface-hover">
            <ShoppingBag className="h-8 w-8 text-text-muted" />
          </div>
          <h2 className="text-lg font-semibold text-text">No orders yet</h2>
          <p className="mt-1 text-sm text-text-secondary">
            When you place orders, they will appear here.
          </p>
          <Link
            href="/products"
            className="mt-6 rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-dark"
          >
            Start Shopping
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          {orders.map((order) => (
            <Link
              key={order.id}
              href={`/orders/${order.id}`}
              className="group flex rounded-xl border border-border bg-surface p-5 shadow-sm transition-all hover:border-primary/30 hover:shadow-md"
            >
              <div className="flex flex-1 flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                {/* Left: order info */}
                <div className="flex items-start gap-4">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Package className="h-5 w-5" />
                  </div>
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="text-sm font-semibold text-text">
                        {order.order_number || order.id.slice(0, 12).toUpperCase()}
                      </p>
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${statusColor(order.status)}`}>
                        {order.status}
                      </span>
                    </div>
                    <p className="mt-1 text-xs text-text-secondary">
                      Placed on {formatDate(order.created_at)}
                    </p>
                    {order.items && order.items.length > 0 && (
                      <p className="mt-1.5 line-clamp-1 text-sm text-text-secondary">
                        {order.items.map((i) => i.name).join(', ')}
                      </p>
                    )}
                  </div>
                </div>

                {/* Right: total + caret */}
                <div className="flex items-center justify-between sm:flex-col sm:items-end sm:gap-1">
                  <div className="text-right">
                    <p className="text-sm font-bold text-text">
                      {formatCurrency(order.total)}
                    </p>
                    {order.items && (
                      <p className="text-xs text-text-secondary">
                        {order.items.length} {order.items.length === 1 ? 'item' : 'items'}
                      </p>
                    )}
                  </div>
                  <ChevronRight className="h-4 w-4 text-text-muted transition-transform group-hover:translate-x-0.5 sm:mt-1" />
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

export default function AccountOrdersPage() {
  return (
    <AuthGuard requiredRole="customer">
      <OrdersContent />
    </AuthGuard>
  );
}
