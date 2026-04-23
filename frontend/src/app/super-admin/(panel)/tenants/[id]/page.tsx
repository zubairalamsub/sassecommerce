'use client';

import { use } from 'react';
import Link from 'next/link';
import { ArrowLeft, Globe, Mail, Calendar, Users, Package, ShoppingCart, DollarSign, Shield, Clock } from 'lucide-react';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';

const tenantData: Record<string, {
  name: string; slug: string; email: string; tier: string; status: string;
  users: number; created: string; phone: string; address: string;
  products: number; orders: number; revenue: number; monthlyRevenue: number;
  config: { currency: string; language: string; timezone: string };
  recentOrders: { id: string; customer: string; total: number; status: string; date: string }[];
}> = {
  'tenant-001': {
    name: 'Saajan Fashion House', slug: 'saajan-fashion', email: 'admin@saajan.com.bd',
    tier: 'professional', status: 'active', users: 12, created: '2024-01-15',
    phone: '+880 1712-345678', address: 'House 12, Road 5, Dhanmondi, Dhaka',
    products: 156, orders: 2845, revenue: 12450000, monthlyRevenue: 850000,
    config: { currency: 'BDT', language: 'en', timezone: 'Asia/Dhaka' },
    recentOrders: [
      { id: 'ORD-001', customer: 'Rahim Uddin', total: 4500, status: 'delivered', date: '2026-04-19' },
      { id: 'ORD-002', customer: 'Fatima Akter', total: 12800, status: 'shipped', date: '2026-04-19' },
      { id: 'ORD-003', customer: 'Kamal Hossain', total: 3200, status: 'pending', date: '2026-04-18' },
    ],
  },
  'tenant-002': {
    name: 'Dhaka Electronics Hub', slug: 'dhaka-electronics', email: 'info@dhakaelec.com',
    tier: 'enterprise', status: 'active', users: 25, created: '2023-11-01',
    phone: '+880 1911-222333', address: 'Elephant Road, Dhaka 1205',
    products: 320, orders: 5120, revenue: 45800000, monthlyRevenue: 3200000,
    config: { currency: 'BDT', language: 'en', timezone: 'Asia/Dhaka' },
    recentOrders: [
      { id: 'ORD-101', customer: 'Nusrat Jahan', total: 85000, status: 'confirmed', date: '2026-04-20' },
      { id: 'ORD-102', customer: 'Shakib Ahmed', total: 22000, status: 'delivered', date: '2026-04-19' },
    ],
  },
};

const tierColors: Record<string, string> = {
  free: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  starter: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  professional: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400',
  enterprise: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
};

export default function TenantDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const tenant = tenantData[id];

  if (!tenant) {
    return (
      <div className="space-y-4">
        <Link href="/super-admin/tenants" className="inline-flex items-center gap-2 text-sm text-text-secondary hover:text-text">
          <ArrowLeft className="h-4 w-4" /> Back to Tenants
        </Link>
        <div className="rounded-2xl border border-border bg-surface p-16 text-center">
          <p className="text-text-secondary">Tenant not found.</p>
        </div>
      </div>
    );
  }

  const stats = [
    { label: 'Total Revenue', value: formatCurrency(tenant.revenue), icon: DollarSign, color: 'text-green-500 bg-green-100 dark:bg-green-900/30' },
    { label: 'Monthly Revenue', value: formatCurrency(tenant.monthlyRevenue), icon: DollarSign, color: 'text-blue-500 bg-blue-100 dark:bg-blue-900/30' },
    { label: 'Total Orders', value: tenant.orders.toLocaleString(), icon: ShoppingCart, color: 'text-indigo-500 bg-indigo-100 dark:bg-indigo-900/30' },
    { label: 'Products', value: tenant.products.toString(), icon: Package, color: 'text-amber-500 bg-amber-100 dark:bg-amber-900/30' },
  ];

  return (
    <div className="space-y-6">
      {/* Back */}
      <Link href="/super-admin/tenants" className="inline-flex items-center gap-2 text-sm text-text-secondary hover:text-text transition-colors">
        <ArrowLeft className="h-4 w-4" /> Back to Tenants
      </Link>

      {/* Tenant header */}
      <div className="rounded-2xl border border-border bg-surface p-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-100 dark:bg-indigo-900/30 text-xl font-bold text-indigo-600 dark:text-indigo-400">
              {tenant.name[0]}
            </div>
            <div>
              <h1 className="text-xl font-bold text-text">{tenant.name}</h1>
              <div className="flex items-center gap-2 mt-1">
                <span className="text-sm text-text-muted">{tenant.slug}</span>
                <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(tenant.status))}>
                  {tenant.status}
                </span>
                <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', tierColors[tenant.tier] || tierColors.free)}>
                  {tenant.tier}
                </span>
              </div>
            </div>
          </div>
          <div className="flex gap-2">
            <button className="rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-secondary hover:bg-surface-hover transition-colors">
              Suspend
            </button>
            <button className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors">
              Edit Tenant
            </button>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.label} className="rounded-2xl border border-border bg-surface p-5">
              <div className="flex items-center justify-between">
                <span className="text-sm text-text-secondary">{stat.label}</span>
                <div className={cn('rounded-lg p-2', stat.color)}>
                  <Icon className="h-4 w-4" />
                </div>
              </div>
              <p className="mt-2 text-2xl font-bold text-text">{stat.value}</p>
            </div>
          );
        })}
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Details */}
        <div className="rounded-2xl border border-border bg-surface p-6 lg:col-span-1">
          <h2 className="text-base font-semibold text-text mb-4">Details</h2>
          <div className="space-y-4">
            <div className="flex items-start gap-3">
              <Mail className="h-4 w-4 text-text-muted mt-0.5" />
              <div><p className="text-xs text-text-muted">Email</p><p className="text-sm text-text">{tenant.email}</p></div>
            </div>
            <div className="flex items-start gap-3">
              <Globe className="h-4 w-4 text-text-muted mt-0.5" />
              <div><p className="text-xs text-text-muted">Phone</p><p className="text-sm text-text">{tenant.phone}</p></div>
            </div>
            <div className="flex items-start gap-3">
              <Shield className="h-4 w-4 text-text-muted mt-0.5" />
              <div><p className="text-xs text-text-muted">Address</p><p className="text-sm text-text">{tenant.address}</p></div>
            </div>
            <div className="flex items-start gap-3">
              <Calendar className="h-4 w-4 text-text-muted mt-0.5" />
              <div><p className="text-xs text-text-muted">Created</p><p className="text-sm text-text">{formatDate(tenant.created)}</p></div>
            </div>
            <div className="flex items-start gap-3">
              <Users className="h-4 w-4 text-text-muted mt-0.5" />
              <div><p className="text-xs text-text-muted">Users</p><p className="text-sm text-text">{tenant.users}</p></div>
            </div>
            <div className="flex items-start gap-3">
              <Clock className="h-4 w-4 text-text-muted mt-0.5" />
              <div>
                <p className="text-xs text-text-muted">Configuration</p>
                <p className="text-sm text-text">{tenant.config.currency} / {tenant.config.language} / {tenant.config.timezone}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Recent orders */}
        <div className="rounded-2xl border border-border bg-surface p-6 lg:col-span-2">
          <h2 className="text-base font-semibold text-text mb-4">Recent Orders</h2>
          {tenant.recentOrders.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border text-left text-xs text-text-muted">
                    <th className="pb-3 font-medium">Order</th>
                    <th className="pb-3 font-medium">Customer</th>
                    <th className="pb-3 font-medium">Total</th>
                    <th className="pb-3 font-medium">Status</th>
                    <th className="pb-3 font-medium">Date</th>
                  </tr>
                </thead>
                <tbody>
                  {tenant.recentOrders.map((order) => (
                    <tr key={order.id} className="border-b border-border-light">
                      <td className="py-3 text-sm font-medium text-indigo-600 dark:text-indigo-400">{order.id}</td>
                      <td className="py-3 text-sm text-text">{order.customer}</td>
                      <td className="py-3 text-sm text-text">{formatCurrency(order.total)}</td>
                      <td className="py-3">
                        <span className={cn('inline-flex rounded-full px-2 py-0.5 text-xs font-medium capitalize', statusColor(order.status))}>
                          {order.status}
                        </span>
                      </td>
                      <td className="py-3 text-sm text-text-muted">{formatDate(order.date)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-text-muted py-4">No recent orders</p>
          )}
        </div>
      </div>
    </div>
  );
}
