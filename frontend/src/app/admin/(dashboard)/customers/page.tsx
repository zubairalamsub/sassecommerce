'use client';

import { useState, useEffect } from 'react';
import {
  Users,
  UserCheck,
  UserPlus,
  Search,
  Loader2,
  Mail,
  Phone,
} from 'lucide-react';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { userApi, type User } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

export default function CustomersPage() {
  const { tenantId, token } = useAuthStore();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  useEffect(() => {
    loadCustomers();
  }, [tenantId, token]);

  async function loadCustomers() {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const res = await userApi.list(tenantId, token, 1, 200);
      const customers = (res.data ?? []).filter((u) => u.role === 'customer');
      setUsers(customers);
    } catch {
      setUsers([]);
    } finally {
      setLoading(false);
    }
  }

  const filtered = users.filter((u) => {
    if (statusFilter && u.status !== statusFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return (
        u.first_name.toLowerCase().includes(q) ||
        u.last_name.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q) ||
        (u.phone ?? '').toLowerCase().includes(q)
      );
    }
    return true;
  });

  const stats = [
    {
      title: 'Total Customers',
      value: users.length,
      icon: Users,
      color: 'text-gray-900',
    },
    {
      title: 'Active',
      value: users.filter((u) => u.status === 'active').length,
      icon: UserCheck,
      color: 'text-green-600',
    },
    {
      title: 'New This Month',
      value: users.filter((u) => {
        const d = new Date(u.created_at);
        const now = new Date();
        return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
      }).length,
      icon: UserPlus,
      color: 'text-primary',
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Customers</h1>
          <p className="mt-1 text-sm text-gray-500">View and manage your customer base.</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.title} className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
              <div className="flex items-center gap-4">
                <span className="rounded-lg bg-primary-light p-3">
                  <Icon className="h-6 w-6 text-primary" />
                </span>
                <div>
                  <p className="text-sm text-gray-500">{stat.title}</p>
                  <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name, email or phone..."
            className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none"
        >
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="suspended">Suspended</option>
        </select>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        {loading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <Users className="h-10 w-10 text-gray-300" />
            <p className="mt-3 text-sm font-medium text-gray-700">No customers found</p>
            <p className="mt-1 text-xs text-gray-400">
              {search || statusFilter ? 'Try adjusting your search or filter' : 'Customers who register will appear here'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3 font-medium">Name</th>
                  <th className="px-6 py-3 font-medium">Email</th>
                  <th className="px-6 py-3 font-medium">Phone</th>
                  <th className="px-6 py-3 font-medium">Email Verified</th>
                  <th className="px-6 py-3 font-medium">Status</th>
                  <th className="px-6 py-3 font-medium">Joined</th>
                  <th className="px-6 py-3 font-medium">Last Login</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((customer) => (
                  <tr
                    key={customer.id}
                    className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                  >
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-semibold text-primary">
                          {customer.first_name[0]}{customer.last_name[0]}
                        </div>
                        <span className="text-sm font-medium text-gray-900">
                          {customer.first_name} {customer.last_name}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-1.5 text-sm text-gray-500">
                        <Mail className="h-3.5 w-3.5 text-gray-400" />
                        {customer.email}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {customer.phone ? (
                        <div className="flex items-center gap-1.5">
                          <Phone className="h-3.5 w-3.5 text-gray-400" />
                          {customer.phone}
                        </div>
                      ) : (
                        <span className="text-gray-300">—</span>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      <span className={cn(
                        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                        customer.email_verified
                          ? 'bg-green-100 text-green-800'
                          : 'bg-yellow-100 text-yellow-800',
                      )}>
                        {customer.email_verified ? 'Verified' : 'Unverified'}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span className={cn(
                        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                        statusColor(customer.status),
                      )}>
                        {customer.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {formatDate(customer.created_at)}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {customer.last_login_at ? formatDate(customer.last_login_at) : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
