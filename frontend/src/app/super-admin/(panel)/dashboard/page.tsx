'use client';

import { useEffect } from 'react';
import { Building2, CheckCircle, DollarSign, TrendingUp } from 'lucide-react';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { useTenantStore } from '@/stores/tenants';
import Link from 'next/link';

const tierColor: Record<string, string> = {
  free: 'bg-gray-100 text-gray-800',
  starter: 'bg-blue-100 text-blue-800',
  professional: 'bg-indigo-100 text-indigo-800',
  enterprise: 'bg-purple-100 text-purple-800',
};

const tierPrices: Record<string, number> = { free: 0, starter: 2999, professional: 9999, enterprise: 29999 };

export default function SuperAdminDashboardPage() {
  const user = useAuthStore((s) => s.user);
  const { tenants, fetchTenants } = useTenantStore();

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const activeTenants = tenants.filter((t) => t.status === 'active').length;
  const totalMRR = tenants.reduce((sum, t) => sum + (tierPrices[t.tier] || 0), 0);

  const tierCounts = tenants.reduce<Record<string, number>>((acc, t) => {
    acc[t.tier] = (acc[t.tier] || 0) + 1;
    return acc;
  }, {});

  const recentTenants = [...tenants].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()).slice(0, 5);

  const stats = [
    { title: 'Total Tenants', value: tenants.length, icon: Building2, iconBg: 'bg-indigo-50', iconColor: 'text-indigo-600' },
    { title: 'Active Tenants', value: activeTenants, icon: CheckCircle, iconBg: 'bg-green-50', iconColor: 'text-green-600' },
    { title: 'Total MRR', value: formatCurrency(totalMRR), icon: DollarSign, iconBg: 'bg-purple-50', iconColor: 'text-purple-600' },
    { title: 'ARR', value: formatCurrency(totalMRR * 12), icon: TrendingUp, iconBg: 'bg-blue-50', iconColor: 'text-blue-600' },
  ];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Platform Dashboard</h1>
        <p className="mt-1 text-sm text-gray-500">
          Welcome back, {user?.first_name || 'Admin'}! Here&apos;s your platform overview.
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.title} className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-gray-500">{stat.title}</span>
                <span className={cn('rounded-lg p-2', stat.iconBg)}>
                  <Icon className={cn('h-5 w-5', stat.iconColor)} />
                </span>
              </div>
              <div className="mt-3">
                <span className="text-2xl font-bold text-gray-900">{stat.value}</span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Two-column section */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Tenants by Tier */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
          <div className="border-b border-gray-200 px-6 py-4">
            <h2 className="text-lg font-semibold text-gray-900">Tenants by Tier</h2>
          </div>
          <div className="p-6">
            <div className="space-y-4">
              {['enterprise', 'professional', 'starter', 'free'].map((tier) => {
                const count = tierCounts[tier] || 0;
                const pct = tenants.length > 0 ? (count / tenants.length) * 100 : 0;
                return (
                  <div key={tier} className="flex items-center justify-between">
                    <span className={cn('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', tierColor[tier])}>
                      {tier}
                    </span>
                    <div className="flex items-center gap-3">
                      <div className="h-2 w-24 overflow-hidden rounded-full bg-gray-100">
                        <div className="h-full rounded-full bg-indigo-500" style={{ width: `${pct}%` }} />
                      </div>
                      <span className="text-sm font-medium text-gray-900">{count}</span>
                    </div>
                  </div>
                );
              })}
            </div>
            <div className="mt-4 border-t border-gray-100 pt-4">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium text-gray-500">Total</span>
                <span className="font-semibold text-gray-900">{tenants.length}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Recent Tenants */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm lg:col-span-2">
          <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
            <h2 className="text-lg font-semibold text-gray-900">Recent Tenants</h2>
            <Link href="/super-admin/tenants" className="text-sm font-medium text-indigo-600 hover:underline">
              View all
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3 font-medium">Name</th>
                  <th className="px-6 py-3 font-medium">Email</th>
                  <th className="px-6 py-3 font-medium">Tier</th>
                  <th className="px-6 py-3 font-medium">Status</th>
                  <th className="px-6 py-3 font-medium">Created</th>
                </tr>
              </thead>
              <tbody>
                {recentTenants.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-8 text-center text-sm text-gray-400">
                      No tenants yet.
                    </td>
                  </tr>
                ) : (
                  recentTenants.map((tenant) => (
                    <tr key={tenant.id} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                      <td className="px-6 py-4 text-sm font-medium text-gray-900">{tenant.name}</td>
                      <td className="px-6 py-4 text-sm text-gray-500">{tenant.email}</td>
                      <td className="px-6 py-4">
                        <span className={cn('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', tierColor[tenant.tier])}>
                          {tenant.tier}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <span className={cn('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(tenant.status))}>
                          {tenant.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">{formatDate(tenant.created_at)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
