'use client';

import {
  Building2,
  CheckCircle,
  DollarSign,
  TrendingUp,
} from 'lucide-react';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';

const stats = [
  {
    title: 'Total Tenants',
    value: '12',
    icon: Building2,
    iconBg: 'bg-indigo-50',
    iconColor: 'text-indigo-600',
  },
  {
    title: 'Active Tenants',
    value: '10',
    icon: CheckCircle,
    iconBg: 'bg-green-50',
    iconColor: 'text-green-600',
  },
  {
    title: 'Total Revenue',
    value: formatCurrency(4500000),
    icon: DollarSign,
    iconBg: 'bg-purple-50',
    iconColor: 'text-purple-600',
  },
  {
    title: 'MRR',
    value: formatCurrency(375000),
    icon: TrendingUp,
    iconBg: 'bg-blue-50',
    iconColor: 'text-blue-600',
  },
];

const tierDistribution = [
  { tier: 'Free', count: 4, color: 'bg-gray-100 text-gray-800' },
  { tier: 'Starter', count: 3, color: 'bg-blue-100 text-blue-800' },
  { tier: 'Professional', count: 3, color: 'bg-indigo-100 text-indigo-800' },
  { tier: 'Enterprise', count: 2, color: 'bg-purple-100 text-purple-800' },
];

const recentTenants = [
  {
    name: 'Saajan Fashion House',
    email: 'admin@fashion.com.bd',
    tier: 'Professional',
    status: 'active',
    created: '2025-06-01',
  },
  {
    name: 'Dhaka Electronics',
    email: 'admin@dhaka-electronics.com.bd',
    tier: 'Enterprise',
    status: 'active',
    created: '2025-03-15',
  },
  {
    name: 'Chittagong Crafts',
    email: 'admin@ctg-crafts.com.bd',
    tier: 'Starter',
    status: 'active',
    created: '2025-09-20',
  },
  {
    name: 'Sylhet Tea Store',
    email: 'admin@sylhet-tea.com.bd',
    tier: 'Free',
    status: 'pending',
    created: '2026-03-01',
  },
  {
    name: 'Rajshahi Silk',
    email: 'admin@rajshahi-silk.com.bd',
    tier: 'Professional',
    status: 'suspended',
    created: '2025-12-10',
  },
];

const tierColor: Record<string, string> = {
  Free: 'bg-gray-100 text-gray-800',
  Starter: 'bg-blue-100 text-blue-800',
  Professional: 'bg-indigo-100 text-indigo-800',
  Enterprise: 'bg-purple-100 text-purple-800',
};

export default function SuperAdminDashboardPage() {
  const user = useAuthStore((s) => s.user);

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
            <div
              key={stat.title}
              className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm"
            >
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
              {tierDistribution.map((item) => (
                <div key={item.tier} className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span
                      className={cn(
                        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                        item.color,
                      )}
                    >
                      {item.tier}
                    </span>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="h-2 w-24 overflow-hidden rounded-full bg-gray-100">
                      <div
                        className="h-full rounded-full bg-indigo-500"
                        style={{ width: `${(item.count / 12) * 100}%` }}
                      />
                    </div>
                    <span className="text-sm font-medium text-gray-900">{item.count}</span>
                  </div>
                </div>
              ))}
            </div>
            <div className="mt-4 border-t border-gray-100 pt-4">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium text-gray-500">Total</span>
                <span className="font-semibold text-gray-900">12</span>
              </div>
            </div>
          </div>
        </div>

        {/* Recent Tenants */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm lg:col-span-2">
          <div className="border-b border-gray-200 px-6 py-4">
            <h2 className="text-lg font-semibold text-gray-900">Recent Tenants</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3 font-medium">Tenant Name</th>
                  <th className="px-6 py-3 font-medium">Email</th>
                  <th className="px-6 py-3 font-medium">Tier</th>
                  <th className="px-6 py-3 font-medium">Status</th>
                  <th className="px-6 py-3 font-medium">Created</th>
                </tr>
              </thead>
              <tbody>
                {recentTenants.map((tenant) => (
                  <tr
                    key={tenant.email}
                    className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                  >
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">
                      {tenant.name}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">{tenant.email}</td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                          tierColor[tenant.tier],
                        )}
                      >
                        {tenant.tier}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          statusColor(tenant.status),
                        )}
                      >
                        {tenant.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {formatDate(tenant.created)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
