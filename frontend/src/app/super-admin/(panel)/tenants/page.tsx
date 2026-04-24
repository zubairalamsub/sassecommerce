'use client';

import { useState, useEffect } from 'react';
import { Search, Plus, Loader2, Pencil, Trash2 } from 'lucide-react';
import { cn, formatDate, statusColor } from '@/lib/utils';
import { useTenantStore } from '@/stores/tenants';
import type { Tenant } from '@/lib/api';

const tierColor: Record<string, string> = {
  free: 'bg-gray-100 text-gray-800',
  starter: 'bg-blue-100 text-blue-800',
  professional: 'bg-indigo-100 text-indigo-800',
  enterprise: 'bg-purple-100 text-purple-800',
};

export default function SuperAdminTenantsPage() {
  const { tenants, loading, fetchTenants, updateTenant, deleteTenant } = useTenantStore();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [deleting, setDeleting] = useState<string | null>(null);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const [editStatus, setEditStatus] = useState('');
  const [editTier, setEditTier] = useState('');

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const filtered = tenants.filter((t) => {
    if (statusFilter && t.status !== statusFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return t.name.toLowerCase().includes(q) || t.email.toLowerCase().includes(q) || (t.domain ?? '').toLowerCase().includes(q);
    }
    return true;
  });

  async function handleDelete(tenant: Tenant) {
    if (!confirm(`Delete "${tenant.name}"? This cannot be undone.`)) return;
    setDeleting(tenant.id);
    await deleteTenant(tenant.id);
    setDeleting(null);
  }

  async function handleUpdate(e: React.FormEvent) {
    e.preventDefault();
    if (!editingTenant) return;
    await updateTenant(editingTenant.id, { status: editStatus as Tenant['status'], tier: editTier as Tenant['tier'] });
    setEditingTenant(null);
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Tenants</h1>
          <p className="mt-1 text-sm text-gray-500">{tenants.length} total stores</p>
        </div>
        <button
          onClick={() => {}}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
        >
          <Plus className="h-4 w-4" />
          Add Tenant
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search tenants..."
            className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
        >
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="pending">Pending</option>
          <option value="suspended">Suspended</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      {/* Edit dialog */}
      {editingTenant && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-sm rounded-2xl border border-gray-200 bg-white p-6 shadow-xl">
            <h3 className="text-base font-semibold text-gray-900 mb-4">Edit Tenant: {editingTenant.name}</h3>
            <form onSubmit={handleUpdate} className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Status</label>
                <select value={editStatus} onChange={(e) => setEditStatus(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none">
                  <option value="active">Active</option>
                  <option value="pending">Pending</option>
                  <option value="suspended">Suspended</option>
                  <option value="cancelled">Cancelled</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Tier</label>
                <select value={editTier} onChange={(e) => setEditTier(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none">
                  <option value="free">Free</option>
                  <option value="starter">Starter</option>
                  <option value="professional">Professional</option>
                  <option value="enterprise">Enterprise</option>
                </select>
              </div>
              <div className="flex gap-3 pt-2">
                <button type="submit" className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors">
                  Save
                </button>
                <button type="button" onClick={() => setEditingTenant(null)} className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
        {loading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-16 text-center text-sm text-gray-400">No tenants found.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3 font-medium">Name</th>
                  <th className="px-6 py-3 font-medium">Email</th>
                  <th className="px-6 py-3 font-medium">Domain</th>
                  <th className="px-6 py-3 font-medium">Tier</th>
                  <th className="px-6 py-3 font-medium">Status</th>
                  <th className="px-6 py-3 font-medium">Created</th>
                  <th className="px-6 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((tenant) => (
                  <tr key={tenant.id} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">{tenant.name}</td>
                    <td className="px-6 py-4 text-sm text-gray-500">{tenant.email}</td>
                    <td className="px-6 py-4 text-sm font-mono text-gray-500">
                      {tenant.domain || <span className="text-gray-300">—</span>}
                    </td>
                    <td className="px-6 py-4">
                      <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', tierColor[tenant.tier] ?? 'bg-gray-100 text-gray-800')}>
                        {tenant.tier}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(tenant.status))}>
                        {tenant.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">{formatDate(tenant.created_at)}</td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => { setEditingTenant(tenant); setEditStatus(tenant.status); setEditTier(tenant.tier); }}
                          className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
                        >
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(tenant)}
                          disabled={deleting === tenant.id}
                          className="rounded-lg p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors disabled:opacity-50"
                        >
                          {deleting === tenant.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
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
    </div>
  );
}
