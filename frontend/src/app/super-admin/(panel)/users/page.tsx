'use client';

import { useState, useMemo } from 'react';
import {
  Search,
  Plus,
  Users,
  ShieldCheck,
  Store,
  UserCheck,
  Pencil,
  Ban,
  Trash2,
  CheckCircle,
  XCircle,
  X,
  ChevronDown,
  Mail,
  Clock,
} from 'lucide-react';
import { cn, formatDate, formatDateTime } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type UserRole =
  | 'super_admin'
  | 'platform_admin'
  | 'tenant_admin'
  | 'store_manager'
  | 'customer';

type UserStatus = 'active' | 'pending' | 'suspended';

interface DemoUser {
  id: string;
  firstName: string;
  lastName: string;
  email: string;
  role: UserRole;
  tenant: string | null; // null = Platform-level
  status: UserStatus;
  lastLogin: string | null;
  createdAt: string;
  needsApproval: boolean;
}

// ---------------------------------------------------------------------------
// Permission catalogue
// ---------------------------------------------------------------------------

interface PermissionGroup {
  category: string;
  permissions: { key: string; label: string }[];
}

const permissionGroups: PermissionGroup[] = [
  {
    category: 'Tenant Management',
    permissions: [
      { key: 'tenant:create', label: 'Create tenants' },
      { key: 'tenant:read', label: 'View tenants' },
      { key: 'tenant:update', label: 'Update tenants' },
      { key: 'tenant:delete', label: 'Delete tenants' },
      { key: 'tenant:approve', label: 'Approve tenants' },
    ],
  },
  {
    category: 'User Management',
    permissions: [
      { key: 'user:create', label: 'Create users' },
      { key: 'user:read', label: 'View users' },
      { key: 'user:update', label: 'Update users' },
      { key: 'user:suspend', label: 'Suspend users' },
    ],
  },
  {
    category: 'Billing',
    permissions: [
      { key: 'billing:invoices', label: 'View invoices' },
      { key: 'billing:plans', label: 'Manage plans' },
      { key: 'billing:refunds', label: 'Issue refunds' },
    ],
  },
  {
    category: 'Content',
    permissions: [
      { key: 'content:pages', label: 'Manage pages' },
      { key: 'content:themes', label: 'Manage themes' },
      { key: 'content:integrations', label: 'Manage integrations' },
    ],
  },
  {
    category: 'System',
    permissions: [
      { key: 'system:logs', label: 'View logs' },
      { key: 'system:backups', label: 'Manage backups' },
      { key: 'system:security', label: 'Configure security' },
    ],
  },
];

const defaultPermissions: Record<UserRole, string[]> = {
  super_admin: permissionGroups.flatMap((g) => g.permissions.map((p) => p.key)),
  platform_admin: [
    'tenant:read',
    'tenant:update',
    'tenant:approve',
    'user:create',
    'user:read',
    'user:update',
    'user:suspend',
    'billing:invoices',
    'billing:plans',
    'content:pages',
    'content:themes',
    'system:logs',
  ],
  tenant_admin: [
    'user:create',
    'user:read',
    'user:update',
    'billing:invoices',
    'content:pages',
    'content:themes',
    'content:integrations',
  ],
  store_manager: [
    'user:read',
    'content:pages',
    'content:themes',
  ],
  customer: [],
};

// ---------------------------------------------------------------------------
// Demo data - 20 users with realistic Bangladeshi names
// ---------------------------------------------------------------------------

const tenantNames = [
  'Aarong Fashion',
  'Daraz BD',
  'Chaldal Groceries',
  'Shajgoj Beauty',
  'Rokomari Books',
];

const demoUsers: DemoUser[] = [
  { id: 'u1', firstName: 'Rashida', lastName: 'Begum', email: 'rashida.begum@platform.io', role: 'super_admin', tenant: null, status: 'active', lastLogin: '2026-04-25T09:12:00Z', createdAt: '2024-01-10T08:00:00Z', needsApproval: false },
  { id: 'u2', firstName: 'Arif', lastName: 'Hossain', email: 'arif.hossain@platform.io', role: 'platform_admin', tenant: null, status: 'active', lastLogin: '2026-04-24T17:45:00Z', createdAt: '2024-03-15T10:30:00Z', needsApproval: false },
  { id: 'u3', firstName: 'Nazia', lastName: 'Sultana', email: 'nazia.sultana@platform.io', role: 'platform_admin', tenant: null, status: 'active', lastLogin: '2026-04-23T14:20:00Z', createdAt: '2024-05-01T09:00:00Z', needsApproval: false },
  { id: 'u4', firstName: 'Kamal', lastName: 'Uddin', email: 'kamal@aarong.com', role: 'tenant_admin', tenant: 'Aarong Fashion', status: 'active', lastLogin: '2026-04-25T08:30:00Z', createdAt: '2024-06-10T11:00:00Z', needsApproval: false },
  { id: 'u5', firstName: 'Fatema', lastName: 'Akter', email: 'fatema@daraz.bd', role: 'tenant_admin', tenant: 'Daraz BD', status: 'active', lastLogin: '2026-04-24T11:15:00Z', createdAt: '2024-07-20T14:00:00Z', needsApproval: false },
  { id: 'u6', firstName: 'Mizanur', lastName: 'Rahman', email: 'mizan@chaldal.com', role: 'tenant_admin', tenant: 'Chaldal Groceries', status: 'suspended', lastLogin: '2026-03-10T09:00:00Z', createdAt: '2024-08-05T10:00:00Z', needsApproval: false },
  { id: 'u7', firstName: 'Tahmina', lastName: 'Islam', email: 'tahmina@shajgoj.com', role: 'store_manager', tenant: 'Shajgoj Beauty', status: 'active', lastLogin: '2026-04-25T07:50:00Z', createdAt: '2024-09-12T08:30:00Z', needsApproval: false },
  { id: 'u8', firstName: 'Shafikul', lastName: 'Alam', email: 'shafik@rokomari.com', role: 'store_manager', tenant: 'Rokomari Books', status: 'active', lastLogin: '2026-04-22T16:00:00Z', createdAt: '2024-10-01T12:00:00Z', needsApproval: false },
  { id: 'u9', firstName: 'Sumaiya', lastName: 'Khan', email: 'sumaiya.khan@gmail.com', role: 'customer', tenant: 'Aarong Fashion', status: 'active', lastLogin: '2026-04-25T10:05:00Z', createdAt: '2025-01-15T09:00:00Z', needsApproval: false },
  { id: 'u10', firstName: 'Jubayer', lastName: 'Ahmed', email: 'jubayer.ahmed@yahoo.com', role: 'customer', tenant: 'Daraz BD', status: 'active', lastLogin: '2026-04-20T18:30:00Z', createdAt: '2025-02-10T11:00:00Z', needsApproval: false },
  { id: 'u11', firstName: 'Hasina', lastName: 'Parvin', email: 'hasina.parvin@outlook.com', role: 'customer', tenant: 'Chaldal Groceries', status: 'active', lastLogin: '2026-04-18T14:00:00Z', createdAt: '2025-03-05T07:30:00Z', needsApproval: false },
  { id: 'u12', firstName: 'Rafiqul', lastName: 'Haque', email: 'rafiqul.haque@gmail.com', role: 'customer', tenant: 'Shajgoj Beauty', status: 'suspended', lastLogin: '2026-02-14T10:00:00Z', createdAt: '2025-04-20T16:00:00Z', needsApproval: false },
  { id: 'u13', firstName: 'Anika', lastName: 'Tasnim', email: 'anika.tasnim@gmail.com', role: 'customer', tenant: 'Rokomari Books', status: 'active', lastLogin: '2026-04-24T20:15:00Z', createdAt: '2025-05-08T13:00:00Z', needsApproval: false },
  { id: 'u14', firstName: 'Imran', lastName: 'Chowdhury', email: 'imran.chowdhury@platform.io', role: 'platform_admin', tenant: null, status: 'pending', lastLogin: null, createdAt: '2026-04-20T09:00:00Z', needsApproval: true },
  { id: 'u15', firstName: 'Sabrina', lastName: 'Nahar', email: 'sabrina@aarong.com', role: 'store_manager', tenant: 'Aarong Fashion', status: 'pending', lastLogin: null, createdAt: '2026-04-22T10:30:00Z', needsApproval: true },
  { id: 'u16', firstName: 'Mahfuz', lastName: 'Reza', email: 'mahfuz.reza@daraz.bd', role: 'tenant_admin', tenant: 'Daraz BD', status: 'pending', lastLogin: null, createdAt: '2026-04-23T15:00:00Z', needsApproval: true },
  { id: 'u17', firstName: 'Farzana', lastName: 'Yeasmin', email: 'farzana.yeasmin@gmail.com', role: 'customer', tenant: 'Aarong Fashion', status: 'active', lastLogin: '2026-04-21T12:00:00Z', createdAt: '2025-06-15T08:00:00Z', needsApproval: false },
  { id: 'u18', firstName: 'Tanvir', lastName: 'Morshed', email: 'tanvir.morshed@chaldal.com', role: 'store_manager', tenant: 'Chaldal Groceries', status: 'active', lastLogin: '2026-04-24T09:45:00Z', createdAt: '2025-07-01T10:00:00Z', needsApproval: false },
  { id: 'u19', firstName: 'Nazma', lastName: 'Khatun', email: 'nazma.khatun@outlook.com', role: 'customer', tenant: 'Daraz BD', status: 'pending', lastLogin: null, createdAt: '2026-04-24T11:00:00Z', needsApproval: true },
  { id: 'u20', firstName: 'Abdur', lastName: 'Rahim', email: 'abdur.rahim@rokomari.com', role: 'tenant_admin', tenant: 'Rokomari Books', status: 'active', lastLogin: '2026-04-25T06:30:00Z', createdAt: '2025-08-10T14:00:00Z', needsApproval: false },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const roleLabelMap: Record<UserRole, string> = {
  super_admin: 'Super Admin',
  platform_admin: 'Platform Admin',
  tenant_admin: 'Store Admin',
  store_manager: 'Store Manager',
  customer: 'Customer',
};

const roleColorMap: Record<UserRole, string> = {
  super_admin: 'bg-red-100 text-red-800',
  platform_admin: 'bg-indigo-100 text-indigo-800',
  tenant_admin: 'bg-purple-100 text-purple-800',
  store_manager: 'bg-blue-100 text-blue-800',
  customer: 'bg-gray-100 text-gray-800',
};

const statusColorMap: Record<UserStatus, string> = {
  active: 'bg-green-100 text-green-800',
  pending: 'bg-yellow-100 text-yellow-800',
  suspended: 'bg-red-100 text-red-800',
};

function initials(firstName: string, lastName: string) {
  return `${firstName.charAt(0)}${lastName.charAt(0)}`.toUpperCase();
}

const avatarColors = [
  'bg-indigo-500',
  'bg-emerald-500',
  'bg-amber-500',
  'bg-rose-500',
  'bg-cyan-500',
  'bg-purple-500',
  'bg-fuchsia-500',
  'bg-teal-500',
];

function avatarColor(id: string) {
  let hash = 0;
  for (let i = 0; i < id.length; i++) {
    hash = id.charCodeAt(i) + ((hash << 5) - hash);
  }
  return avatarColors[Math.abs(hash) % avatarColors.length];
}

// ---------------------------------------------------------------------------
// Tabs
// ---------------------------------------------------------------------------

type TabKey = 'all' | 'platform_admins' | 'store_admins' | 'customers' | 'pending';

const tabs: { key: TabKey; label: string }[] = [
  { key: 'all', label: 'All Users' },
  { key: 'platform_admins', label: 'Platform Admins' },
  { key: 'store_admins', label: 'Store Admins' },
  { key: 'customers', label: 'Customers' },
  { key: 'pending', label: 'Pending Approvals' },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function SuperAdminUsersPage() {
  const [users, setUsers] = useState<DemoUser[]>(demoUsers);
  const [activeTab, setActiveTab] = useState<TabKey>('all');
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<'' | UserRole>('');
  const [statusFilter, setStatusFilter] = useState<'' | UserStatus>('');

  // Modals
  const [editingUser, setEditingUser] = useState<DemoUser | null>(null);
  const [editRole, setEditRole] = useState<UserRole>('customer');
  const [editPermissions, setEditPermissions] = useState<string[]>([]);

  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteFirstName, setInviteFirstName] = useState('');
  const [inviteLastName, setInviteLastName] = useState('');
  const [inviteRole, setInviteRole] = useState<UserRole>('customer');
  const [inviteTenant, setInviteTenant] = useState('');

  // Delete confirmation
  const [deletingUserId, setDeletingUserId] = useState<string | null>(null);

  // ---------- Derived ----------

  const stats = useMemo(() => {
    const total = users.length;
    const platformAdmins = users.filter(
      (u) => u.role === 'super_admin' || u.role === 'platform_admin',
    ).length;
    const storeAdmins = users.filter(
      (u) => u.role === 'tenant_admin' || u.role === 'store_manager',
    ).length;
    const customers = users.filter((u) => u.role === 'customer').length;
    return { total, platformAdmins, storeAdmins, customers };
  }, [users]);

  const filtered = useMemo(() => {
    let list = users;

    // Tab filter
    if (activeTab === 'platform_admins') {
      list = list.filter((u) => u.role === 'super_admin' || u.role === 'platform_admin');
    } else if (activeTab === 'store_admins') {
      list = list.filter((u) => u.role === 'tenant_admin' || u.role === 'store_manager');
    } else if (activeTab === 'customers') {
      list = list.filter((u) => u.role === 'customer');
    } else if (activeTab === 'pending') {
      list = list.filter((u) => u.needsApproval);
    }

    // Role filter
    if (roleFilter) {
      list = list.filter((u) => u.role === roleFilter);
    }

    // Status filter
    if (statusFilter) {
      list = list.filter((u) => u.status === statusFilter);
    }

    // Search
    if (search) {
      const q = search.toLowerCase();
      list = list.filter(
        (u) =>
          u.firstName.toLowerCase().includes(q) ||
          u.lastName.toLowerCase().includes(q) ||
          u.email.toLowerCase().includes(q) ||
          (u.tenant ?? '').toLowerCase().includes(q),
      );
    }

    return list;
  }, [users, activeTab, roleFilter, statusFilter, search]);

  // ---------- Handlers ----------

  function openEditRole(user: DemoUser) {
    setEditingUser(user);
    setEditRole(user.role);
    setEditPermissions([...defaultPermissions[user.role]]);
  }

  function handleEditRoleChange(role: UserRole) {
    setEditRole(role);
    setEditPermissions([...defaultPermissions[role]]);
  }

  function togglePermission(key: string) {
    setEditPermissions((prev) =>
      prev.includes(key) ? prev.filter((p) => p !== key) : [...prev, key],
    );
  }

  function saveRole() {
    if (!editingUser) return;
    setUsers((prev) =>
      prev.map((u) => (u.id === editingUser.id ? { ...u, role: editRole } : u)),
    );
    setEditingUser(null);
  }

  function toggleSuspend(user: DemoUser) {
    setUsers((prev) =>
      prev.map((u) =>
        u.id === user.id
          ? { ...u, status: u.status === 'suspended' ? 'active' : 'suspended' }
          : u,
      ),
    );
  }

  function deleteUser(id: string) {
    setUsers((prev) => prev.filter((u) => u.id !== id));
    setDeletingUserId(null);
  }

  function approveUser(id: string) {
    setUsers((prev) =>
      prev.map((u) =>
        u.id === id ? { ...u, status: 'active', needsApproval: false } : u,
      ),
    );
  }

  function rejectUser(id: string) {
    setUsers((prev) => prev.filter((u) => u.id !== id));
  }

  function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    const newUser: DemoUser = {
      id: `u${Date.now()}`,
      firstName: inviteFirstName,
      lastName: inviteLastName,
      email: inviteEmail,
      role: inviteRole,
      tenant:
        inviteRole === 'tenant_admin' || inviteRole === 'store_manager'
          ? inviteTenant || null
          : null,
      status: 'pending',
      lastLogin: null,
      createdAt: new Date().toISOString(),
      needsApproval: true,
    };
    setUsers((prev) => [newUser, ...prev]);
    setInviteOpen(false);
    setInviteEmail('');
    setInviteFirstName('');
    setInviteLastName('');
    setInviteRole('customer');
    setInviteTenant('');
  }

  // ---------- Stats cards definition ----------

  const statCards = [
    {
      title: 'Total Users',
      value: stats.total,
      icon: Users,
      iconBg: 'bg-indigo-50',
      iconColor: 'text-indigo-600',
    },
    {
      title: 'Platform Admins',
      value: stats.platformAdmins,
      icon: ShieldCheck,
      iconBg: 'bg-red-50',
      iconColor: 'text-red-600',
    },
    {
      title: 'Store Admins',
      value: stats.storeAdmins,
      icon: Store,
      iconBg: 'bg-purple-50',
      iconColor: 'text-purple-600',
    },
    {
      title: 'Customers',
      value: stats.customers,
      icon: UserCheck,
      iconBg: 'bg-green-50',
      iconColor: 'text-green-600',
    },
  ];

  // ---------- Render ----------

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Users &amp; Roles</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage platform users, roles, and permissions across all tenants.
          </p>
        </div>
        <button
          onClick={() => setInviteOpen(true)}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
        >
          <Plus className="h-4 w-4" />
          Invite User
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {statCards.map((stat) => {
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

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6 overflow-x-auto" aria-label="Tabs">
          {tabs.map((tab) => {
            const isActive = activeTab === tab.key;
            const pendingCount =
              tab.key === 'pending'
                ? users.filter((u) => u.needsApproval).length
                : undefined;
            return (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  'whitespace-nowrap border-b-2 px-1 pb-3 text-sm font-medium transition-colors',
                  isActive
                    ? 'border-indigo-600 text-indigo-600'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
                )}
              >
                {tab.label}
                {pendingCount !== undefined && pendingCount > 0 && (
                  <span className="ml-2 inline-flex items-center rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-800">
                    {pendingCount}
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Search + Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative min-w-[200px] flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name, email, or tenant..."
            className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
          />
        </div>
        <select
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value as '' | UserRole)}
          className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
        >
          <option value="">All Roles</option>
          <option value="super_admin">Super Admin</option>
          <option value="platform_admin">Platform Admin</option>
          <option value="tenant_admin">Store Admin</option>
          <option value="store_manager">Store Manager</option>
          <option value="customer">Customer</option>
        </select>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as '' | UserStatus)}
          className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
        >
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="pending">Pending</option>
          <option value="suspended">Suspended</option>
        </select>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
        {filtered.length === 0 ? (
          <div className="py-16 text-center text-sm text-gray-400">No users found.</div>
        ) : activeTab === 'pending' ? (
          /* ---------- Pending Approvals Table ---------- */
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3 font-medium">User</th>
                  <th className="px-6 py-3 font-medium">Role</th>
                  <th className="px-6 py-3 font-medium">Tenant</th>
                  <th className="px-6 py-3 font-medium">Requested</th>
                  <th className="px-6 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((user) => (
                  <tr
                    key={user.id}
                    className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                  >
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div
                          className={cn(
                            'flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-semibold text-white',
                            avatarColor(user.id),
                          )}
                        >
                          {initials(user.firstName, user.lastName)}
                        </div>
                        <div>
                          <div className="text-sm font-medium text-gray-900">
                            {user.firstName} {user.lastName}
                          </div>
                          <div className="text-sm text-gray-500">{user.email}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                          roleColorMap[user.role],
                        )}
                      >
                        {roleLabelMap[user.role]}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {user.tenant ?? 'Platform'}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {formatDate(user.createdAt)}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => approveUser(user.id)}
                          className="inline-flex items-center gap-1.5 rounded-lg bg-green-50 px-3 py-1.5 text-xs font-medium text-green-700 transition-colors hover:bg-green-100"
                        >
                          <CheckCircle className="h-3.5 w-3.5" />
                          Approve
                        </button>
                        <button
                          onClick={() => rejectUser(user.id)}
                          className="inline-flex items-center gap-1.5 rounded-lg bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 transition-colors hover:bg-red-100"
                        >
                          <XCircle className="h-3.5 w-3.5" />
                          Reject
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          /* ---------- Main Users Table ---------- */
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3 font-medium">User</th>
                  <th className="px-6 py-3 font-medium">Role</th>
                  <th className="px-6 py-3 font-medium">Tenant</th>
                  <th className="px-6 py-3 font-medium">Status</th>
                  <th className="px-6 py-3 font-medium">Last Login</th>
                  <th className="px-6 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((user) => (
                  <tr
                    key={user.id}
                    className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                  >
                    {/* Avatar + Name + Email */}
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div
                          className={cn(
                            'flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-semibold text-white',
                            avatarColor(user.id),
                          )}
                        >
                          {initials(user.firstName, user.lastName)}
                        </div>
                        <div>
                          <div className="text-sm font-medium text-gray-900">
                            {user.firstName} {user.lastName}
                          </div>
                          <div className="text-sm text-gray-500">{user.email}</div>
                        </div>
                      </div>
                    </td>
                    {/* Role */}
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                          roleColorMap[user.role],
                        )}
                      >
                        {roleLabelMap[user.role]}
                      </span>
                    </td>
                    {/* Tenant */}
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {user.tenant ?? (
                        <span className="font-medium text-indigo-600">Platform</span>
                      )}
                    </td>
                    {/* Status */}
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          statusColorMap[user.status],
                        )}
                      >
                        {user.status}
                      </span>
                    </td>
                    {/* Last Login */}
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {user.lastLogin ? (
                        <span className="inline-flex items-center gap-1.5">
                          <Clock className="h-3.5 w-3.5 text-gray-400" />
                          {formatDateTime(user.lastLogin)}
                        </span>
                      ) : (
                        <span className="text-gray-300">Never</span>
                      )}
                    </td>
                    {/* Actions */}
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => openEditRole(user)}
                          title="Edit role"
                          className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                        >
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => toggleSuspend(user)}
                          title={user.status === 'suspended' ? 'Activate' : 'Suspend'}
                          className={cn(
                            'rounded-lg p-1.5 transition-colors',
                            user.status === 'suspended'
                              ? 'text-green-500 hover:bg-green-50 hover:text-green-700'
                              : 'text-gray-400 hover:bg-yellow-50 hover:text-yellow-600',
                          )}
                        >
                          {user.status === 'suspended' ? (
                            <CheckCircle className="h-4 w-4" />
                          ) : (
                            <Ban className="h-4 w-4" />
                          )}
                        </button>
                        <button
                          onClick={() => setDeletingUserId(user.id)}
                          title="Delete"
                          className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* ---------- Role Management Modal ---------- */}
      {editingUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-gray-200 bg-white shadow-xl">
            {/* Modal header */}
            <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
              <h3 className="text-base font-semibold text-gray-900">
                Edit User Role &amp; Permissions
              </h3>
              <button
                onClick={() => setEditingUser(null)}
                className="rounded-lg p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="px-6 py-5 space-y-6">
              {/* User info */}
              <div className="flex items-center gap-3">
                <div
                  className={cn(
                    'flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-semibold text-white',
                    avatarColor(editingUser.id),
                  )}
                >
                  {initials(editingUser.firstName, editingUser.lastName)}
                </div>
                <div>
                  <div className="text-sm font-medium text-gray-900">
                    {editingUser.firstName} {editingUser.lastName}
                  </div>
                  <div className="text-sm text-gray-500">{editingUser.email}</div>
                </div>
              </div>

              {/* Role selector */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Role
                </label>
                <div className="relative">
                  <select
                    value={editRole}
                    onChange={(e) => handleEditRoleChange(e.target.value as UserRole)}
                    className="w-full appearance-none rounded-lg border border-gray-200 px-3 py-2 pr-9 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  >
                    <option value="super_admin">Super Admin</option>
                    <option value="platform_admin">Platform Admin</option>
                    <option value="tenant_admin">Store Admin</option>
                    <option value="store_manager">Store Manager</option>
                    <option value="customer">Customer</option>
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                </div>
              </div>

              {/* Permissions checklist */}
              <div>
                <label className="mb-3 block text-sm font-medium text-gray-700">
                  Permissions
                </label>
                <div className="space-y-5">
                  {permissionGroups.map((group) => (
                    <div key={group.category}>
                      <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-400">
                        {group.category}
                      </h4>
                      <div className="space-y-1.5">
                        {group.permissions.map((perm) => {
                          const checked = editPermissions.includes(perm.key);
                          return (
                            <label
                              key={perm.key}
                              className="flex cursor-pointer items-center gap-2.5 rounded-lg px-2 py-1.5 transition-colors hover:bg-gray-50"
                            >
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => togglePermission(perm.key)}
                                className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                              />
                              <span className="text-sm text-gray-700">{perm.label}</span>
                            </label>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Modal footer */}
            <div className="flex gap-3 border-t border-gray-200 px-6 py-4">
              <button
                onClick={saveRole}
                className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
              >
                Save Changes
              </button>
              <button
                onClick={() => setEditingUser(null)}
                className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ---------- Invite User Modal ---------- */}
      {inviteOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-md rounded-2xl border border-gray-200 bg-white shadow-xl">
            {/* Modal header */}
            <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
              <h3 className="text-base font-semibold text-gray-900">Invite User</h3>
              <button
                onClick={() => setInviteOpen(false)}
                className="rounded-lg p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <form onSubmit={handleInvite} className="px-6 py-5 space-y-4">
              {/* Email */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Email Address
                </label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  <input
                    type="email"
                    required
                    value={inviteEmail}
                    onChange={(e) => setInviteEmail(e.target.value)}
                    placeholder="user@example.com"
                    className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
              </div>

              {/* First / Last Name */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-gray-700">
                    First Name
                  </label>
                  <input
                    type="text"
                    required
                    value={inviteFirstName}
                    onChange={(e) => setInviteFirstName(e.target.value)}
                    placeholder="First name"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-gray-700">
                    Last Name
                  </label>
                  <input
                    type="text"
                    required
                    value={inviteLastName}
                    onChange={(e) => setInviteLastName(e.target.value)}
                    placeholder="Last name"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
              </div>

              {/* Role */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Role
                </label>
                <div className="relative">
                  <select
                    value={inviteRole}
                    onChange={(e) => setInviteRole(e.target.value as UserRole)}
                    className="w-full appearance-none rounded-lg border border-gray-200 px-3 py-2 pr-9 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  >
                    <option value="platform_admin">Platform Admin</option>
                    <option value="tenant_admin">Store Admin</option>
                    <option value="store_manager">Store Manager</option>
                    <option value="customer">Customer</option>
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                </div>
              </div>

              {/* Tenant selector - only for tenant-level roles */}
              {(inviteRole === 'tenant_admin' || inviteRole === 'store_manager') && (
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-gray-700">
                    Tenant
                  </label>
                  <div className="relative">
                    <select
                      value={inviteTenant}
                      onChange={(e) => setInviteTenant(e.target.value)}
                      required
                      className="w-full appearance-none rounded-lg border border-gray-200 px-3 py-2 pr-9 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    >
                      <option value="">Select a tenant...</option>
                      {tenantNames.map((name) => (
                        <option key={name} value={name}>
                          {name}
                        </option>
                      ))}
                    </select>
                    <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  </div>
                </div>
              )}

              {/* Footer */}
              <div className="flex gap-3 pt-2">
                <button
                  type="submit"
                  className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                >
                  Send Invite
                </button>
                <button
                  type="button"
                  onClick={() => setInviteOpen(false)}
                  className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ---------- Delete Confirmation Modal ---------- */}
      {deletingUserId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-sm rounded-2xl border border-gray-200 bg-white p-6 shadow-xl">
            <h3 className="text-base font-semibold text-gray-900">Confirm Deletion</h3>
            <p className="mt-2 text-sm text-gray-500">
              Are you sure you want to delete this user? This action cannot be undone.
            </p>
            <div className="mt-5 flex gap-3">
              <button
                onClick={() => deleteUser(deletingUserId)}
                className="flex-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700"
              >
                Delete
              </button>
              <button
                onClick={() => setDeletingUserId(null)}
                className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
