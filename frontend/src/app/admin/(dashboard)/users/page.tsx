'use client';

import { useState, useEffect } from 'react';
import { cn, formatDate } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { userApi, type User } from '@/lib/api';
import {
  Users,
  Shield,
  ShoppingBag,
  Search,
  Loader2,
} from 'lucide-react';

type RoleFilter = 'all' | 'staff' | 'customer';

const ROLE_BADGE: Record<string, string> = {
  admin: 'bg-green-100 text-green-800',
  moderator: 'bg-blue-100 text-blue-800',
  customer: 'bg-gray-100 text-gray-800',
};

export default function UsersPage() {
  const { tenantId, token, user: currentUser } = useAuthStore();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<RoleFilter>('all');
  const [search, setSearch] = useState('');

  useEffect(() => {
    loadUsers();
  }, [tenantId, token]);

  async function loadUsers() {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const res = await userApi.list(tenantId, token, 1, 200);
      setUsers(res.data ?? []);
    } catch {
      setUsers([]);
    } finally {
      setLoading(false);
    }
  }

  const staffCount = users.filter((u) => u.role === 'admin' || u.role === 'moderator').length;
  const customerCount = users.filter((u) => u.role === 'customer').length;

  const filtered = users.filter((u) => {
    if (tab === 'staff') return u.role === 'admin' || u.role === 'moderator';
    if (tab === 'customer') return u.role === 'customer';
    return true;
  }).filter((u) => {
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      u.first_name.toLowerCase().includes(q) ||
      u.last_name.toLowerCase().includes(q) ||
      u.email.toLowerCase().includes(q)
    );
  });

  const tabs: { key: RoleFilter; label: string }[] = [
    { key: 'all', label: 'All' },
    { key: 'staff', label: 'Staff' },
    { key: 'customer', label: 'Customers' },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Users &amp; Staff</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage your store team and customer accounts
          </p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-gray-200 bg-white p-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-light">
              <Users className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Total Users</p>
              <p className="text-2xl font-bold text-gray-900">{users.length}</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-50">
              <Shield className="h-5 w-5 text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Staff Members</p>
              <p className="text-2xl font-bold text-gray-900">{staffCount}</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100">
              <ShoppingBag className="h-5 w-5 text-gray-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Customers</p>
              <p className="text-2xl font-bold text-gray-900">{customerCount}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs + Search */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex gap-1 rounded-lg border border-gray-200 bg-white p-1">
          {tabs.map((t) => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={cn(
                'rounded-md px-4 py-1.5 text-sm font-medium transition-colors',
                tab === t.key
                  ? 'bg-primary text-white'
                  : 'text-gray-600 hover:bg-gray-100',
              )}
            >
              {t.label}
            </button>
          ))}
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search users..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm text-gray-900 placeholder:text-gray-400 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white">
        {loading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50">
                  <th className="px-5 py-3 font-medium text-gray-600">User</th>
                  <th className="px-5 py-3 font-medium text-gray-600">Phone</th>
                  <th className="px-5 py-3 font-medium text-gray-600">Role</th>
                  <th className="px-5 py-3 font-medium text-gray-600">Status</th>
                  <th className="px-5 py-3 font-medium text-gray-600">Last Login</th>
                  <th className="px-5 py-3 font-medium text-gray-600">Joined</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {filtered.map((u) => (
                  <tr key={u.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary-light text-xs font-semibold text-primary">
                          {u.first_name[0]}{u.last_name[0]}
                        </div>
                        <div>
                          <p className="font-medium text-gray-900">
                            {u.first_name} {u.last_name}
                          </p>
                          <p className="text-xs text-gray-500">{u.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-5 py-3.5 text-gray-600">
                      {u.phone ?? <span className="text-gray-300">—</span>}
                    </td>
                    <td className="px-5 py-3.5">
                      <span
                        className={cn(
                          'inline-block rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          ROLE_BADGE[u.role] ?? 'bg-gray-100 text-gray-800',
                        )}
                      >
                        {u.role}
                      </span>
                    </td>
                    <td className="px-5 py-3.5">
                      <span
                        className={cn(
                          'inline-flex items-center gap-1.5 text-xs font-medium capitalize',
                          u.status === 'active' ? 'text-green-700' : 'text-gray-400',
                        )}
                      >
                        <span
                          className={cn(
                            'h-1.5 w-1.5 rounded-full',
                            u.status === 'active' ? 'bg-green-500' : 'bg-gray-300',
                          )}
                        />
                        {u.status}
                      </span>
                    </td>
                    <td className="px-5 py-3.5 text-gray-500">
                      {u.last_login_at ? formatDate(u.last_login_at) : '—'}
                    </td>
                    <td className="px-5 py-3.5 text-gray-500">
                      {formatDate(u.created_at)}
                    </td>
                  </tr>
                ))}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={6} className="px-5 py-10 text-center text-sm text-gray-500">
                      No users found.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
