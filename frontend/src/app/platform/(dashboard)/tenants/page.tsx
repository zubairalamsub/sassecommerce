'use client';

import { useState, useEffect } from 'react';
import { Plus, Pencil, Trash2, Loader2, X, Building2, Search } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { useTenantStore } from '@/stores/tenants';
import type { Tenant } from '@/lib/api';

type TenantStatus = Tenant['status'];
type TenantTier = Tenant['tier'];

const statusTabs: { label: string; value: TenantStatus | 'all' }[] = [
  { label: 'All', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Pending', value: 'pending' },
  { label: 'Suspended', value: 'suspended' },
  { label: 'Cancelled', value: 'cancelled' },
];

// ---------------------------------------------------------------------------
// Modal
// ---------------------------------------------------------------------------

interface TenantFormData {
  name: string;
  email: string;
  tier: TenantTier;
  status: TenantStatus;
  domain: string;
}

function TenantModal({
  open,
  tenant,
  saving,
  onClose,
  onSave,
}: {
  open: boolean;
  tenant: Tenant | null;
  saving: boolean;
  onClose: () => void;
  onSave: (data: TenantFormData) => void;
}) {
  const [form, setForm] = useState<TenantFormData>({
    name: '',
    email: '',
    tier: 'starter',
    status: 'active',
    domain: '',
  });

  useEffect(() => {
    if (tenant) {
      setForm({
        name: tenant.name,
        email: tenant.email,
        tier: tenant.tier,
        status: tenant.status,
        domain: tenant.domain || '',
      });
    } else {
      setForm({ name: '', email: '', tier: 'starter', status: 'active', domain: '' });
    }
  }, [tenant, open]);

  if (!open) return null;

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
      >
        <motion.div
          className="relative w-full max-w-lg rounded-2xl border border-border bg-surface p-6 shadow-xl"
          initial={{ opacity: 0, scale: 0.95, y: 20 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 20 }}
          transition={{ duration: 0.2 }}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-semibold text-text">
              {tenant ? 'Edit Tenant' : 'New Tenant'}
            </h2>
            <button onClick={onClose} className="rounded-lg p-1.5 text-text-secondary hover:bg-surface-hover transition-colors">
              <X className="h-5 w-5" />
            </button>
          </div>

          <form
            onSubmit={(e) => { e.preventDefault(); onSave(form); }}
            className="space-y-4"
          >
            <div>
              <label className="block text-sm font-medium text-text mb-1.5">
                Store Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                required
                value={form.name}
                onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                placeholder="e.g. Dhaka Fashion House"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-text mb-1.5">
                Email <span className="text-red-500">*</span>
              </label>
              <input
                type="email"
                required
                value={form.email}
                onChange={(e) => setForm((p) => ({ ...p, email: e.target.value }))}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                placeholder="admin@store.com.bd"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-text mb-1.5">Domain</label>
              <input
                type="text"
                value={form.domain}
                onChange={(e) => setForm((p) => ({ ...p, domain: e.target.value }))}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text font-mono placeholder:text-text-muted focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                placeholder="store.saajan.com.bd"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-text mb-1.5">Plan</label>
                <select
                  value={form.tier}
                  onChange={(e) => setForm((p) => ({ ...p, tier: e.target.value as TenantTier }))}
                  className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                >
                  <option value="free">Free</option>
                  <option value="starter">Starter</option>
                  <option value="professional">Professional</option>
                  <option value="enterprise">Enterprise</option>
                </select>
              </div>

              {tenant && (
                <div>
                  <label className="block text-sm font-medium text-text mb-1.5">Status</label>
                  <select
                    value={form.status}
                    onChange={(e) => setForm((p) => ({ ...p, status: e.target.value as TenantStatus }))}
                    className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                  >
                    <option value="active">Active</option>
                    <option value="pending">Pending</option>
                    <option value="suspended">Suspended</option>
                    <option value="cancelled">Cancelled</option>
                  </select>
                </div>
              )}
            </div>

            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={saving || !form.name.trim() || !form.email.trim()}
                className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-violet-700 disabled:opacity-50"
              >
                {saving && <Loader2 className="h-4 w-4 animate-spin" />}
                {tenant ? 'Update' : 'Create'}
              </button>
            </div>
          </form>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function TenantsPage() {
  const { tenants, loading, fetchTenants, addTenant, updateTenant, deleteTenant, updateTenantStatus } = useTenantStore();
  const [activeTab, setActiveTab] = useState<TenantStatus | 'all'>('all');
  const [search, setSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const filtered = tenants
    .filter((t) => activeTab === 'all' || t.status === activeTab)
    .filter((t) =>
      search ? t.name.toLowerCase().includes(search.toLowerCase()) || t.email.toLowerCase().includes(search.toLowerCase()) : true,
    );

  const tierPrices: Record<string, number> = { free: 0, starter: 2999, professional: 9999, enterprise: 29999 };

  async function handleSave(data: TenantFormData) {
    setSaving(true);
    if (editingTenant) {
      await updateTenant(editingTenant.id, {
        name: data.name,
        email: data.email,
        tier: data.tier,
        status: data.status,
        domain: data.domain || null,
      });
    } else {
      await addTenant({ name: data.name, email: data.email, tier: data.tier });
    }
    setModalOpen(false);
    setSaving(false);
  }

  async function handleDelete(tenant: Tenant) {
    if (!confirm(`Delete tenant "${tenant.name}"? This cannot be undone.`)) return;
    setDeleting(tenant.id);
    await deleteTenant(tenant.id);
    setDeleting(null);
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <motion.div
        className="flex items-center justify-between"
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <div>
          <h1 className="text-2xl font-bold text-text">Tenants</h1>
          <p className="mt-1 text-sm text-text-secondary">{tenants.length} total stores</p>
        </div>
        <button
          onClick={() => { setEditingTenant(null); setModalOpen(true); }}
          className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-violet-700"
        >
          <Plus className="h-4 w-4" />
          Add Tenant
        </button>
      </motion.div>

      {/* Search + Tabs */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="border-b border-border">
          <nav className="-mb-px flex gap-6">
            {statusTabs.map((tab) => {
              const count = tab.value === 'all' ? tenants.length : tenants.filter((t) => t.status === tab.value).length;
              return (
                <button
                  key={tab.value}
                  onClick={() => setActiveTab(tab.value)}
                  className={cn(
                    'border-b-2 pb-3 text-sm font-medium transition-colors',
                    activeTab === tab.value
                      ? 'border-violet-600 text-violet-600 dark:text-violet-400'
                      : 'border-transparent text-text-secondary hover:border-border hover:text-text',
                  )}
                >
                  {tab.label} ({count})
                </button>
              );
            })}
          </nav>
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
          <input
            type="text"
            placeholder="Search tenants..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-surface py-2 pl-9 pr-3 text-sm text-text placeholder:text-text-muted focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500 sm:w-64"
          />
        </div>
      </div>

      {/* Table */}
      <motion.div
        className="rounded-2xl border border-border bg-surface shadow-sm"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border text-left text-sm text-text-secondary">
                <th className="px-6 py-3 font-medium">Store</th>
                <th className="px-6 py-3 font-medium">Domain</th>
                <th className="px-6 py-3 font-medium">Plan</th>
                <th className="px-6 py-3 font-medium">MRR</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Created</th>
                <th className="px-6 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={7} className="px-6 py-16 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-violet-600" />
                    <p className="mt-2 text-sm text-text-secondary">Loading tenants...</p>
                  </td>
                </tr>
              ) : filtered.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-16 text-center">
                    <Building2 className="mx-auto h-10 w-10 text-text-muted" />
                    <p className="mt-3 text-sm font-medium text-text">No tenants found</p>
                    <p className="mt-1 text-sm text-text-muted">Create your first tenant to get started.</p>
                  </td>
                </tr>
              ) : (
                filtered.map((tenant) => (
                  <tr key={tenant.id} className="border-b border-border-light last:border-0 transition-colors hover:bg-surface-hover">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/30 text-sm font-bold text-violet-600 dark:text-violet-400">
                          {tenant.name.charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <p className="text-sm font-medium text-text">{tenant.name}</p>
                          <p className="text-xs text-text-muted">{tenant.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-text-secondary font-mono">
                      {tenant.domain || <span className="text-text-muted">--</span>}
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex rounded-full bg-violet-100 dark:bg-violet-900/30 px-2.5 py-0.5 text-xs font-medium capitalize text-violet-700 dark:text-violet-400">
                        {tenant.tier}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-text">
                      {tierPrices[tenant.tier] > 0 ? formatCurrency(tierPrices[tenant.tier]) : 'Free'}
                      {tierPrices[tenant.tier] > 0 && <span className="text-text-muted font-normal">/mo</span>}
                    </td>
                    <td className="px-6 py-4">
                      <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(tenant.status))}>
                        {tenant.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-text-muted">{formatDate(tenant.created_at)}</td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => { setEditingTenant(tenant); setModalOpen(true); }}
                          className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-text"
                        >
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(tenant)}
                          disabled={deleting === tenant.id}
                          className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 disabled:opacity-50"
                        >
                          {deleting === tenant.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </motion.div>

      <TenantModal
        open={modalOpen}
        tenant={editingTenant}
        saving={saving}
        onClose={() => setModalOpen(false)}
        onSave={handleSave}
      />
    </div>
  );
}
