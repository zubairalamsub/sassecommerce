'use client';

import { useState } from 'react';
import Link from 'next/link';
import {
  Search,
  Plus,
  MoreHorizontal,
  Eye,
  Pause,
  Trash2,
} from 'lucide-react';
import { cn, formatDate, statusColor } from '@/lib/utils';

interface Tenant {
  id: string;
  name: string;
  slug: string;
  email: string;
  tier: string;
  status: string;
  users: number;
  created: string;
}

const tenants: Tenant[] = [
  {
    id: 'tenant-001',
    name: 'Saajan Fashion House',
    slug: 'saajan-fashion',
    email: 'admin@fashion.com.bd',
    tier: 'Professional',
    status: 'active',
    users: 8,
    created: '2025-06-01',
  },
  {
    id: 'tenant-002',
    name: 'Dhaka Electronics',
    slug: 'dhaka-electronics',
    email: 'admin@dhaka-electronics.com.bd',
    tier: 'Enterprise',
    status: 'active',
    users: 15,
    created: '2025-03-15',
  },
  {
    id: 'tenant-003',
    name: 'Chittagong Crafts',
    slug: 'ctg-crafts',
    email: 'admin@ctg-crafts.com.bd',
    tier: 'Starter',
    status: 'active',
    users: 3,
    created: '2025-09-20',
  },
  {
    id: 'tenant-004',
    name: 'Sylhet Tea Store',
    slug: 'sylhet-tea',
    email: 'admin@sylhet-tea.com.bd',
    tier: 'Free',
    status: 'pending',
    users: 1,
    created: '2026-03-01',
  },
  {
    id: 'tenant-005',
    name: 'Rajshahi Silk',
    slug: 'rajshahi-silk',
    email: 'admin@rajshahi-silk.com.bd',
    tier: 'Professional',
    status: 'suspended',
    users: 6,
    created: '2025-12-10',
  },
  {
    id: 'tenant-006',
    name: 'Comilla Sweets',
    slug: 'comilla-sweets',
    email: 'admin@comilla-sweets.com.bd',
    tier: 'Starter',
    status: 'active',
    users: 4,
    created: '2025-11-05',
  },
  {
    id: 'tenant-007',
    name: 'Rangpur Pottery',
    slug: 'rangpur-pottery',
    email: 'admin@rangpur-pottery.com.bd',
    tier: 'Free',
    status: 'active',
    users: 2,
    created: '2026-01-18',
  },
  {
    id: 'tenant-008',
    name: 'Khulna Jute Gallery',
    slug: 'khulna-jute',
    email: 'admin@khulna-jute.com.bd',
    tier: 'Enterprise',
    status: 'active',
    users: 12,
    created: '2025-07-22',
  },
  {
    id: 'tenant-009',
    name: 'Barisal Fish Market',
    slug: 'barisal-fish',
    email: 'admin@barisal-fish.com.bd',
    tier: 'Free',
    status: 'pending',
    users: 1,
    created: '2026-04-02',
  },
  {
    id: 'tenant-010',
    name: 'Mymensingh Agro',
    slug: 'mymensingh-agro',
    email: 'admin@mymensingh-agro.com.bd',
    tier: 'Starter',
    status: 'active',
    users: 5,
    created: '2025-10-14',
  },
];

const tierColor: Record<string, string> = {
  Free: 'bg-gray-100 text-gray-800',
  Starter: 'bg-blue-100 text-blue-800',
  Professional: 'bg-indigo-100 text-indigo-800',
  Enterprise: 'bg-purple-100 text-purple-800',
};

type FilterTab = 'all' | 'active' | 'pending' | 'suspended';

const filterTabs: { key: FilterTab; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'active', label: 'Active' },
  { key: 'pending', label: 'Pending' },
  { key: 'suspended', label: 'Suspended' },
];

export default function TenantsPage() {
  const [search, setSearch] = useState('');
  const [activeTab, setActiveTab] = useState<FilterTab>('all');
  const [actionMenuId, setActionMenuId] = useState<string | null>(null);

  const filteredTenants = tenants.filter((tenant) => {
    const matchesSearch =
      tenant.name.toLowerCase().includes(search.toLowerCase()) ||
      tenant.email.toLowerCase().includes(search.toLowerCase()) ||
      tenant.slug.toLowerCase().includes(search.toLowerCase());
    const matchesTab = activeTab === 'all' || tenant.status === activeTab;
    return matchesSearch && matchesTab;
  });

  const tabCounts: Record<FilterTab, number> = {
    all: tenants.length,
    active: tenants.filter((t) => t.status === 'active').length,
    pending: tenants.filter((t) => t.status === 'pending').length,
    suspended: tenants.filter((t) => t.status === 'suspended').length,
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Tenants</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage all tenants on the platform
          </p>
        </div>
        <button className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-indigo-700">
          <Plus className="h-4 w-4" />
          Add Tenant
        </button>
      </div>

      {/* Search & Filters */}
      <div className="space-y-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search tenants by name, email, or slug..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-gray-300 py-2.5 pl-10 pr-4 text-sm text-gray-900 placeholder:text-gray-400 focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600"
          />
        </div>

        <div className="flex gap-1 border-b border-gray-200">
          {filterTabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={cn(
                'px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px',
                activeTab === tab.key
                  ? 'border-indigo-600 text-indigo-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
              )}
            >
              {tab.label}
              <span
                className={cn(
                  'ml-2 inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                  activeTab === tab.key
                    ? 'bg-indigo-100 text-indigo-700'
                    : 'bg-gray-100 text-gray-600',
                )}
              >
                {tabCounts[tab.key]}
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                <th className="px-6 py-3 font-medium">Name</th>
                <th className="px-6 py-3 font-medium">Slug</th>
                <th className="px-6 py-3 font-medium">Email</th>
                <th className="px-6 py-3 font-medium">Tier</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Users</th>
                <th className="px-6 py-3 font-medium">Created</th>
                <th className="px-6 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredTenants.map((tenant) => (
                <tr
                  key={tenant.id}
                  className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                >
                  <td className="px-6 py-4 text-sm font-medium text-gray-900">
                    {tenant.name}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 font-mono">
                    {tenant.slug}
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
                  <td className="px-6 py-4 text-sm text-gray-900">{tenant.users}</td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {formatDate(tenant.created)}
                  </td>
                  <td className="px-6 py-4">
                    <div className="relative">
                      <button
                        onClick={() =>
                          setActionMenuId(
                            actionMenuId === tenant.id ? null : tenant.id,
                          )
                        }
                        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                      >
                        <MoreHorizontal className="h-4 w-4" />
                      </button>
                      {actionMenuId === tenant.id && (
                        <div className="absolute right-0 z-10 mt-1 w-40 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                          <Link
                            href={`/super-admin/tenants/${tenant.id}`}
                            className="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
                            onClick={() => setActionMenuId(null)}
                          >
                            <Eye className="h-4 w-4" />
                            Manage
                          </Link>
                          <button
                            className="flex w-full items-center gap-2 px-4 py-2 text-sm text-yellow-700 hover:bg-yellow-50"
                            onClick={() => setActionMenuId(null)}
                          >
                            <Pause className="h-4 w-4" />
                            Suspend
                          </button>
                          <button
                            className="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-700 hover:bg-red-50"
                            onClick={() => setActionMenuId(null)}
                          >
                            <Trash2 className="h-4 w-4" />
                            Delete
                          </button>
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
              {filteredTenants.length === 0 && (
                <tr>
                  <td
                    colSpan={8}
                    className="px-6 py-12 text-center text-sm text-gray-500"
                  >
                    No tenants found matching your criteria.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
