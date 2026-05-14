'use client';

import { use, useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import {
  ArrowLeft,
  Loader2,
  Mail,
  Phone,
  ShieldCheck,
  ShieldAlert,
  ShoppingBag,
  CalendarDays,
  Clock,
  UserX,
  UserCheck,
} from 'lucide-react';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { userApi, orderApi, type User, type Order } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';

export default function CustomerDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { tenantId, token } = useAuthStore();

  const [customer, setCustomer] = useState<User | null>(null);
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    if (!tenantId || !token) {
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [user, orderList] = await Promise.all([
        userApi.get(id, tenantId, token),
        orderApi.listByCustomer(id, tenantId, token).catch(() => ({ data: [] as Order[] })),
      ]);
      setCustomer(user);
      setOrders(orderList.data ?? []);
    } catch {
      setCustomer(null);
    } finally {
      setLoading(false);
    }
  }, [id, tenantId, token]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleStatusToggle() {
    if (!customer || !tenantId || !token) return;
    const next = customer.status === 'active' ? 'suspended' : 'active';
    setActionLoading(true);
    setError('');
    try {
      await userApi.updateStatus(customer.id, next, tenantId, token);
      setCustomer({ ...customer, status: next });
      toast.success(next === 'suspended' ? 'Account suspended' : 'Account reactivated');
    } catch (err) {
      const message = (err as Error).message || 'Failed to update status';
      setError(message);
      toast.error("Couldn't update account");
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!customer) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-12 text-center">
        <p className="text-gray-500">Customer not found.</p>
        <Link
          href="/admin/customers"
          className="mt-4 inline-flex items-center gap-2 text-sm font-medium text-blue-600 hover:text-blue-700"
        >
          <ArrowLeft className="h-4 w-4" /> Back to customers
        </Link>
      </div>
    );
  }

  const totalSpent = orders.reduce((sum, o) => sum + (o.total ?? 0), 0);
  const fullName = `${customer.first_name} ${customer.last_name}`.trim() || customer.username || customer.email;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Link
          href="/admin/customers"
          className="inline-flex items-center gap-2 text-sm font-medium text-gray-500 hover:text-gray-900"
        >
          <ArrowLeft className="h-4 w-4" /> Back to customers
        </Link>
        <button
          onClick={handleStatusToggle}
          disabled={actionLoading}
          className={cn(
            'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium shadow-sm transition',
            customer.status === 'active'
              ? 'border border-red-200 bg-white text-red-700 hover:bg-red-50'
              : 'border border-emerald-200 bg-white text-emerald-700 hover:bg-emerald-50',
            actionLoading && 'opacity-50',
          )}
        >
          {actionLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : customer.status === 'active' ? (
            <UserX className="h-4 w-4" />
          ) : (
            <UserCheck className="h-4 w-4" />
          )}
          {customer.status === 'active' ? 'Suspend account' : 'Reactivate account'}
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="flex items-start gap-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 text-xl font-semibold text-white">
                {(customer.first_name[0] ?? customer.email[0] ?? '?').toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <h1 className="text-xl font-semibold text-gray-900">{fullName}</h1>
                  <span className={cn('rounded-full px-2.5 py-0.5 text-xs font-medium', statusColor(customer.status))}>
                    {customer.status}
                  </span>
                  {customer.email_verified ? (
                    <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700">
                      <ShieldCheck className="h-3.5 w-3.5" /> Verified
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-0.5 text-xs font-medium text-amber-700">
                      <ShieldAlert className="h-3.5 w-3.5" /> Email unverified
                    </span>
                  )}
                </div>
                <p className="mt-1 text-sm text-gray-500">@{customer.username || customer.email.split('@')[0]}</p>
              </div>
            </div>

            <dl className="mt-6 grid grid-cols-1 gap-4 text-sm sm:grid-cols-2">
              <Detail icon={Mail} label="Email" value={customer.email} />
              <Detail icon={Phone} label="Phone" value={customer.phone || '—'} />
              <Detail
                icon={CalendarDays}
                label="Joined"
                value={customer.created_at ? formatDate(customer.created_at) : '—'}
              />
              <Detail
                icon={Clock}
                label="Last seen"
                value={customer.last_login_at ? formatDate(customer.last_login_at) : 'Never'}
              />
            </dl>
          </div>

          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-base font-semibold text-gray-900">Order history</h2>
              <p className="mt-0.5 text-sm text-gray-500">
                {orders.length} order{orders.length === 1 ? '' : 's'}
              </p>
            </div>
            {orders.length === 0 ? (
              <div className="px-6 py-12 text-center text-sm text-gray-500">No orders yet.</div>
            ) : (
              <div className="divide-y divide-gray-100">
                {orders.map((o) => (
                  <Link
                    key={o.id}
                    href={`/admin/orders/${o.id}`}
                    className="flex items-center justify-between gap-4 px-6 py-4 hover:bg-gray-50"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-100">
                        <ShoppingBag className="h-5 w-5 text-gray-500" />
                      </div>
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-gray-900">#{o.order_number}</p>
                        <p className="text-xs text-gray-500">{formatDate(o.created_at)}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-medium text-gray-900">{formatCurrency(o.total)}</p>
                      <span className={cn('mt-0.5 inline-block rounded-full px-2 py-0.5 text-xs font-medium', statusColor(o.status))}>
                        {o.status}
                      </span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>

        <aside className="space-y-6">
          <Stat title="Orders" value={String(orders.length)} icon={ShoppingBag} />
          <Stat title="Lifetime value" value={formatCurrency(totalSpent)} icon={CalendarDays} />
          <Stat title="Average order" value={orders.length ? formatCurrency(totalSpent / orders.length) : '—'} icon={Clock} />
        </aside>
      </div>
    </div>
  );
}

function Detail({ icon: Icon, label, value }: { icon: React.ElementType; label: string; value: string }) {
  return (
    <div>
      <dt className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-gray-500">
        <Icon className="h-3.5 w-3.5" /> {label}
      </dt>
      <dd className="mt-1 break-words text-gray-900">{value}</dd>
    </div>
  );
}

function Stat({ title, value, icon: Icon }: { title: string; value: string; icon: React.ElementType }) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">{title}</p>
        <Icon className="h-4 w-4 text-gray-400" />
      </div>
      <p className="mt-2 text-2xl font-semibold text-gray-900">{value}</p>
    </div>
  );
}
