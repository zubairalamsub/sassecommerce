'use client';

import { useState, useEffect } from 'react';
import {
  Store,
  Search,
  Plus,
  CheckCircle,
  XCircle,
  Clock,
  Star,
  Loader2,
  X,
} from 'lucide-react';
import { vendorApi, type Vendor, type RegisterVendorRequest } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { formatDate, formatCurrency } from '@/lib/utils';

const STATUS_COLORS: Record<string, string> = {
  active: 'bg-green-100 text-green-800',
  pending: 'bg-yellow-100 text-yellow-800',
  suspended: 'bg-red-100 text-red-800',
  rejected: 'bg-gray-100 text-gray-700',
};

const STATUS_ICONS: Record<string, typeof CheckCircle> = {
  active: CheckCircle,
  pending: Clock,
  suspended: XCircle,
  rejected: XCircle,
};

export default function VendorsPage() {
  const { tenantId, token } = useAuthStore();
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [addError, setAddError] = useState('');

  const [form, setForm] = useState<RegisterVendorRequest>({
    tenant_id: tenantId || '',
    name: '',
    email: '',
    phone: '',
    description: '',
    city: '',
    country: 'Bangladesh',
  });

  useEffect(() => {
    loadVendors();
  }, [tenantId, statusFilter]);

  async function loadVendors() {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const res = await vendorApi.list(tenantId, token, statusFilter || undefined);
      setVendors(res.vendors ?? []);
    } catch {
      setVendors([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleStatusChange(vendor: Vendor, newStatus: string) {
    if (!tenantId || !token) return;
    setActionLoading(vendor.id);
    try {
      const updated = await vendorApi.updateStatus(vendor.id, newStatus, '', tenantId, token);
      setVendors((prev) => prev.map((v) => (v.id === vendor.id ? updated : v)));
    } catch {
      // ignore
    } finally {
      setActionLoading(null);
    }
  }

  async function handleAddVendor(e: React.FormEvent) {
    e.preventDefault();
    if (!tenantId || !token) return;
    setAddError('');
    try {
      const newVendor = await vendorApi.register({ ...form, tenant_id: tenantId }, tenantId, token);
      setVendors((prev) => [newVendor, ...prev]);
      setShowAddForm(false);
      setForm({ tenant_id: tenantId, name: '', email: '', phone: '', description: '', city: '', country: 'Bangladesh' });
    } catch (err) {
      setAddError((err as Error).message || 'Failed to register vendor.');
    }
  }

  const filtered = vendors.filter((v) => {
    const q = search.toLowerCase();
    return !q || v.name.toLowerCase().includes(q) || v.email.toLowerCase().includes(q) || v.city.toLowerCase().includes(q);
  });

  const stats = {
    total: vendors.length,
    active: vendors.filter((v) => v.status === 'active').length,
    pending: vendors.filter((v) => v.status === 'pending').length,
    suspended: vendors.filter((v) => v.status === 'suspended').length,
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Vendors</h1>
          <p className="mt-1 text-sm text-gray-500">Manage marketplace vendors and their status</p>
        </div>
        <button
          onClick={() => setShowAddForm(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
        >
          <Plus className="h-4 w-4" />
          Add Vendor
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {[
          { label: 'Total', value: stats.total, color: 'text-gray-900' },
          { label: 'Active', value: stats.active, color: 'text-green-600' },
          { label: 'Pending', value: stats.pending, color: 'text-yellow-600' },
          { label: 'Suspended', value: stats.suspended, color: 'text-red-600' },
        ].map((s) => (
          <div key={s.label} className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
            <p className="text-xs font-medium text-gray-500">{s.label}</p>
            <p className={`mt-1 text-2xl font-bold ${s.color}`}>{s.value}</p>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search vendors..."
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
          <option value="pending">Pending</option>
          <option value="suspended">Suspended</option>
          <option value="rejected">Rejected</option>
        </select>
      </div>

      {/* Add Vendor Form */}
      {showAddForm && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-base font-semibold text-gray-900">Register New Vendor</h2>
            <button onClick={() => setShowAddForm(false)} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100">
              <X className="h-4 w-4" />
            </button>
          </div>
          {addError && <p className="mb-3 text-sm text-red-600">{addError}</p>}
          <form onSubmit={handleAddVendor} className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Vendor Name *</label>
              <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Email *</label>
              <input required type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Phone</label>
              <input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">City</label>
              <input value={form.city} onChange={(e) => setForm({ ...form, city: e.target.value })}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none" />
            </div>
            <div className="sm:col-span-2">
              <label className="mb-1 block text-xs font-medium text-gray-600">Description</label>
              <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={2}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none" />
            </div>
            <div className="sm:col-span-2 flex gap-3">
              <button type="submit" className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark">
                Register Vendor
              </button>
              <button type="button" onClick={() => setShowAddForm(false)} className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Vendor Table */}
      {loading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-gray-200 bg-white py-16 text-center">
          <Store className="h-10 w-10 text-gray-300" />
          <p className="mt-3 text-sm font-medium text-gray-700">No vendors found</p>
          <p className="mt-1 text-xs text-gray-400">
            {search ? 'Try a different search term' : 'Register your first vendor to get started'}
          </p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {['Vendor', 'Contact', 'Status', 'Revenue', 'Orders', 'Rating', 'Joined', 'Actions'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {filtered.map((vendor) => {
                const StatusIcon = STATUS_ICONS[vendor.status] ?? Clock;
                return (
                  <tr key={vendor.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary text-sm font-bold">
                          {vendor.name.charAt(0)}
                        </div>
                        <div>
                          <p className="text-sm font-medium text-gray-900">{vendor.name}</p>
                          {vendor.city && <p className="text-xs text-gray-400">{vendor.city}</p>}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-4 text-sm text-gray-600">
                      <p>{vendor.email}</p>
                      {vendor.phone && <p className="text-xs text-gray-400">{vendor.phone}</p>}
                    </td>
                    <td className="px-4 py-4">
                      <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${STATUS_COLORS[vendor.status] ?? 'bg-gray-100 text-gray-600'}`}>
                        <StatusIcon className="h-3 w-3" />
                        {vendor.status}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-sm font-medium text-gray-900">
                      {formatCurrency(vendor.total_revenue)}
                    </td>
                    <td className="px-4 py-4 text-sm text-gray-600">{vendor.total_orders}</td>
                    <td className="px-4 py-4">
                      {vendor.rating > 0 ? (
                        <div className="flex items-center gap-1 text-sm">
                          <Star className="h-3.5 w-3.5 fill-amber-400 text-amber-400" />
                          <span>{vendor.rating.toFixed(1)}</span>
                        </div>
                      ) : (
                        <span className="text-xs text-gray-400">–</span>
                      )}
                    </td>
                    <td className="px-4 py-4 text-xs text-gray-500">{formatDate(vendor.created_at)}</td>
                    <td className="px-4 py-4">
                      {actionLoading === vendor.id ? (
                        <Loader2 className="h-4 w-4 animate-spin text-gray-400" />
                      ) : (
                        <select
                          value={vendor.status}
                          onChange={(e) => handleStatusChange(vendor, e.target.value)}
                          className="rounded-lg border border-gray-200 px-2 py-1 text-xs focus:border-primary focus:outline-none"
                        >
                          <option value="pending">Pending</option>
                          <option value="active">Activate</option>
                          <option value="suspended">Suspend</option>
                          <option value="rejected">Reject</option>
                        </select>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
