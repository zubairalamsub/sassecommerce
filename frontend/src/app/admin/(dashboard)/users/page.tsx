'use client';

import { useState } from 'react';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import {
  Users,
  UserPlus,
  Shield,
  ShoppingBag,
  Search,
  MoreVertical,
  X,
} from 'lucide-react';

type RoleFilter = 'all' | 'staff' | 'customer';

interface StaffUser {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  role: 'admin' | 'moderator' | 'customer';
  status: 'active' | 'inactive';
  last_login: string;
}

const DEMO_USERS: StaffUser[] = [
  {
    id: 'u-001',
    first_name: 'Karim',
    last_name: 'Rahman',
    email: 'admin@fashion.com.bd',
    phone: '+8801712345678',
    role: 'admin',
    status: 'active',
    last_login: '2026-04-18T10:30:00Z',
  },
  {
    id: 'u-002',
    first_name: 'Nusrat',
    last_name: 'Jahan',
    email: 'staff@fashion.com.bd',
    phone: '+8801812345678',
    role: 'moderator',
    status: 'active',
    last_login: '2026-04-17T14:20:00Z',
  },
  {
    id: 'u-003',
    first_name: 'Aminul',
    last_name: 'Haque',
    email: 'aminul@fashion.com.bd',
    phone: '+8801612345678',
    role: 'moderator',
    status: 'active',
    last_login: '2026-04-16T09:15:00Z',
  },
  {
    id: 'u-004',
    first_name: 'Rahim',
    last_name: 'Ahmed',
    email: 'rahim@example.com',
    phone: '+8801912345678',
    role: 'customer',
    status: 'active',
    last_login: '2026-04-18T08:45:00Z',
  },
  {
    id: 'u-005',
    first_name: 'Fatima',
    last_name: 'Begum',
    email: 'fatima@example.com',
    phone: '+8801512345678',
    role: 'customer',
    status: 'active',
    last_login: '2026-04-15T16:30:00Z',
  },
  {
    id: 'u-006',
    first_name: 'Arif',
    last_name: 'Islam',
    email: 'arif@example.com',
    phone: '+8801412345678',
    role: 'customer',
    status: 'active',
    last_login: '2026-04-14T11:20:00Z',
  },
  {
    id: 'u-007',
    first_name: 'Taslima',
    last_name: 'Khatun',
    email: 'taslima@example.com',
    phone: '+8801312345678',
    role: 'customer',
    status: 'inactive',
    last_login: '2026-03-20T13:00:00Z',
  },
  {
    id: 'u-008',
    first_name: 'Jahangir',
    last_name: 'Alam',
    email: 'jahangir@example.com',
    phone: '+8801212345678',
    role: 'customer',
    status: 'active',
    last_login: '2026-04-17T19:10:00Z',
  },
];

const ROLE_BADGE: Record<string, string> = {
  admin: 'bg-green-100 text-green-800',
  moderator: 'bg-blue-100 text-blue-800',
  customer: 'bg-gray-100 text-gray-800',
};

export default function UsersPage() {
  const user = useAuthStore((s) => s.user);
  const isAdmin = user?.role === 'admin';
  const [tab, setTab] = useState<RoleFilter>('all');
  const [search, setSearch] = useState('');
  const [showInviteModal, setShowInviteModal] = useState(false);

  const totalUsers = DEMO_USERS.length + 16; // 24 total
  const staffCount = DEMO_USERS.filter((u) => u.role !== 'customer').length;
  const customerCount = totalUsers - staffCount;

  const filtered = DEMO_USERS.filter((u) => {
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
        {isAdmin && (
          <button
            onClick={() => setShowInviteModal(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
          >
            <UserPlus className="h-4 w-4" />
            Invite User
          </button>
        )}
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
              <p className="text-2xl font-bold text-gray-900">{totalUsers}</p>
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
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50">
                <th className="px-5 py-3 font-medium text-gray-600">User</th>
                <th className="px-5 py-3 font-medium text-gray-600">Phone</th>
                <th className="px-5 py-3 font-medium text-gray-600">Role</th>
                <th className="px-5 py-3 font-medium text-gray-600">Status</th>
                <th className="px-5 py-3 font-medium text-gray-600">Last Login</th>
                <th className="px-5 py-3 font-medium text-gray-600">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {filtered.map((u) => (
                <tr key={u.id} className="hover:bg-gray-50 transition-colors">
                  <td className="px-5 py-3.5">
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary-light text-xs font-semibold text-primary">
                        {u.first_name[0]}
                        {u.last_name[0]}
                      </div>
                      <div>
                        <p className="font-medium text-gray-900">
                          {u.first_name} {u.last_name}
                        </p>
                        <p className="text-xs text-gray-500">{u.email}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-5 py-3.5 text-gray-600">{u.phone}</td>
                  <td className="px-5 py-3.5">
                    <span
                      className={cn(
                        'inline-block rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                        ROLE_BADGE[u.role],
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
                    {new Date(u.last_login).toLocaleDateString('en-GB', {
                      day: 'numeric',
                      month: 'short',
                      year: 'numeric',
                    })}
                  </td>
                  <td className="px-5 py-3.5">
                    <button className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
                      <MoreVertical className="h-4 w-4" />
                    </button>
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
      </div>

      {/* Invite Modal (placeholder) */}
      {showInviteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">Invite User</h2>
              <button
                onClick={() => setShowInviteModal(false)}
                className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <p className="text-sm text-gray-500 mb-6">
              Send an invitation to a new staff member or moderator to join your store.
            </p>
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Email address
                </label>
                <input
                  type="email"
                  placeholder="user@example.com"
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm text-gray-900 placeholder:text-gray-400 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Role
                </label>
                <select className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm text-gray-900 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary">
                  <option value="moderator">Moderator</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
            </div>
            <div className="mt-6 flex gap-3 justify-end">
              <button
                onClick={() => setShowInviteModal(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => setShowInviteModal(false)}
                className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-dark transition-colors"
              >
                Send Invite
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
