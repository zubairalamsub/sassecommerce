'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  Search, Plus, Loader2, Pencil, Trash2, CheckCircle, XCircle,
  Clock, Eye, Building2, ShieldCheck, AlertTriangle, Filter, MoreHorizontal,
  Globe, Database, Zap,
} from 'lucide-react';
import { cn, formatDate, formatDateTime, statusColor, formatCurrency } from '@/lib/utils';
import { useTenantStore } from '@/stores/tenants';
import type { Tenant } from '@/lib/api';
import Link from 'next/link';

const tierColor: Record<string, string> = {
  free: 'bg-gray-100 text-gray-800',
  starter: 'bg-blue-100 text-blue-800',
  professional: 'bg-indigo-100 text-indigo-800',
  enterprise: 'bg-purple-100 text-purple-800',
};

const tierPrices: Record<string, number> = {
  free: 0, starter: 2999, professional: 9999, enterprise: 29999,
};

type TabId = 'all' | 'pending' | 'active' | 'suspended' | 'provisioning';

const TABS: { id: TabId; label: string; icon: typeof Building2 }[] = [
  { id: 'all', label: 'All Tenants', icon: Building2 },
  { id: 'pending', label: 'Pending Approval', icon: Clock },
  { id: 'active', label: 'Active', icon: CheckCircle },
  { id: 'suspended', label: 'Suspended', icon: AlertTriangle },
  { id: 'provisioning', label: 'Provisioning', icon: Database },
];

// Demo provisioning tasks
interface ProvisioningTask {
  tenantId: string;
  tenantName: string;
  steps: { name: string; status: 'completed' | 'in_progress' | 'pending' }[];
  startedAt: string;
}

const demoProvisioning: ProvisioningTask[] = [
  {
    tenantId: 'prov-1',
    tenantName: 'FreshMart BD',
    startedAt: '2026-04-24T18:30:00Z',
    steps: [
      { name: 'Create database schema', status: 'completed' },
      { name: 'Configure storage bucket', status: 'completed' },
      { name: 'Set up default products', status: 'in_progress' },
      { name: 'Configure payment gateway', status: 'pending' },
      { name: 'Generate SSL certificate', status: 'pending' },
      { name: 'DNS configuration', status: 'pending' },
    ],
  },
];

export default function SuperAdminTenantsPage() {
  const { tenants, loading, fetchTenants, addTenant, updateTenant, deleteTenant } = useTenantStore();
  const [search, setSearch] = useState('');
  const [tierFilter, setTierFilter] = useState('');
  const [activeTab, setActiveTab] = useState<TabId>('all');
  const [deleting, setDeleting] = useState<string | null>(null);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const [editStatus, setEditStatus] = useState('');
  const [editTier, setEditTier] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showApprovalModal, setShowApprovalModal] = useState<Tenant | null>(null);
  const [approvalNote, setApprovalNote] = useState('');
  const [toast, setToast] = useState('');

  // Create form
  const [newName, setNewName] = useState('');
  const [newEmail, setNewEmail] = useState('');
  const [newDomain, setNewDomain] = useState('');
  const [newTier, setNewTier] = useState('starter');
  const [newAutoProvision, setNewAutoProvision] = useState(true);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  useEffect(() => {
    if (toast) {
      const t = setTimeout(() => setToast(''), 3000);
      return () => clearTimeout(t);
    }
  }, [toast]);

  const tabCounts = useMemo(() => ({
    all: tenants.length,
    pending: tenants.filter((t) => t.status === 'pending').length,
    active: tenants.filter((t) => t.status === 'active').length,
    suspended: tenants.filter((t) => t.status === 'suspended').length,
    provisioning: demoProvisioning.length,
  }), [tenants]);

  const filtered = useMemo(() => {
    return tenants.filter((t) => {
      if (activeTab === 'pending' && t.status !== 'pending') return false;
      if (activeTab === 'active' && t.status !== 'active') return false;
      if (activeTab === 'suspended' && t.status !== 'suspended') return false;
      if (tierFilter && t.tier !== tierFilter) return false;
      if (search) {
        const q = search.toLowerCase();
        return (
          t.name.toLowerCase().includes(q) ||
          t.email.toLowerCase().includes(q) ||
          (t.domain ?? '').toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [tenants, activeTab, tierFilter, search]);

  async function handleApprove(tenant: Tenant) {
    await updateTenant(tenant.id, { status: 'active' });
    setShowApprovalModal(null);
    setApprovalNote('');
    setToast(`"${tenant.name}" has been approved and activated.`);
  }

  async function handleReject(tenant: Tenant) {
    await updateTenant(tenant.id, { status: 'cancelled' });
    setShowApprovalModal(null);
    setApprovalNote('');
    setToast(`"${tenant.name}" registration has been rejected.`);
  }

  async function handleDelete(tenant: Tenant) {
    if (!confirm(`Delete "${tenant.name}"? This cannot be undone.`)) return;
    setDeleting(tenant.id);
    await deleteTenant(tenant.id);
    setDeleting(null);
    setToast(`"${tenant.name}" has been deleted.`);
  }

  async function handleUpdate(e: React.FormEvent) {
    e.preventDefault();
    if (!editingTenant) return;
    await updateTenant(editingTenant.id, {
      status: editStatus as Tenant['status'],
      tier: editTier as Tenant['tier'],
    });
    setEditingTenant(null);
    setToast('Tenant updated successfully.');
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      await addTenant({
        name: newName,
        email: newEmail,
        tier: newTier,
      });
      setShowCreateModal(false);
      setNewName('');
      setNewEmail('');
      setNewDomain('');
      setNewTier('starter');
      setToast('Tenant created successfully.');
    } catch {
      setToast('Failed to create tenant.');
    }
    setCreating(false);
  }

  return (
    <div className="space-y-6">
      {/* Toast */}
      {toast && (
        <div className="fixed right-6 top-6 z-[60] rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm font-medium text-green-800 shadow-lg">
          {toast}
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Tenant Management</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage store registrations, approvals, and provisioning.
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
        >
          <Plus className="h-4 w-4" />
          Add Tenant
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {[
          { label: 'Total Tenants', value: tenants.length, icon: Building2, color: 'bg-indigo-50 text-indigo-600' },
          { label: 'Pending Approval', value: tabCounts.pending, icon: Clock, color: 'bg-yellow-50 text-yellow-600' },
          { label: 'Active Stores', value: tabCounts.active, icon: CheckCircle, color: 'bg-green-50 text-green-600' },
          { label: 'Total MRR', value: formatCurrency(tenants.reduce((s, t) => s + (tierPrices[t.tier] || 0), 0)), icon: Zap, color: 'bg-purple-50 text-purple-600' },
        ].map((s) => {
          const Icon = s.icon;
          return (
            <div key={s.label} className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-gray-500">{s.label}</span>
                <span className={cn('rounded-lg p-1.5', s.color)}><Icon className="h-4 w-4" /></span>
              </div>
              <p className="mt-2 text-xl font-bold text-gray-900">{s.value}</p>
            </div>
          );
        })}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 overflow-x-auto rounded-lg border border-gray-200 bg-gray-50 p-1">
        {TABS.map((tab) => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex items-center gap-2 whitespace-nowrap rounded-md px-3 py-2 text-sm font-medium transition-colors',
                activeTab === tab.id
                  ? 'bg-white text-indigo-700 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700',
              )}
            >
              <Icon className="h-4 w-4" />
              {tab.label}
              {tabCounts[tab.id] > 0 && (
                <span className={cn(
                  'rounded-full px-2 py-0.5 text-xs font-semibold',
                  activeTab === tab.id ? 'bg-indigo-100 text-indigo-700' : 'bg-gray-200 text-gray-600',
                )}>
                  {tabCounts[tab.id]}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Provisioning Tab Content */}
      {activeTab === 'provisioning' ? (
        <div className="space-y-4">
          {demoProvisioning.length === 0 ? (
            <div className="rounded-xl border border-gray-200 bg-white py-16 text-center">
              <Database className="mx-auto h-12 w-12 text-gray-300" />
              <p className="mt-2 text-sm text-gray-400">No active provisioning tasks.</p>
            </div>
          ) : (
            demoProvisioning.map((task) => (
              <div key={task.tenantId} className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
                <div className="mb-4 flex items-center justify-between">
                  <div>
                    <h3 className="text-base font-semibold text-gray-900">{task.tenantName}</h3>
                    <p className="text-xs text-gray-500">Started {formatDateTime(task.startedAt)}</p>
                  </div>
                  <span className="inline-flex items-center gap-1.5 rounded-full bg-blue-100 px-2.5 py-1 text-xs font-medium text-blue-700">
                    <Loader2 className="h-3 w-3 animate-spin" /> Provisioning
                  </span>
                </div>
                <div className="space-y-3">
                  {task.steps.map((step, i) => (
                    <div key={i} className="flex items-center gap-3">
                      <div className={cn(
                        'flex h-6 w-6 items-center justify-center rounded-full',
                        step.status === 'completed' ? 'bg-green-100' : step.status === 'in_progress' ? 'bg-blue-100' : 'bg-gray-100',
                      )}>
                        {step.status === 'completed' ? (
                          <CheckCircle className="h-4 w-4 text-green-600" />
                        ) : step.status === 'in_progress' ? (
                          <Loader2 className="h-4 w-4 animate-spin text-blue-600" />
                        ) : (
                          <Clock className="h-4 w-4 text-gray-400" />
                        )}
                      </div>
                      <span className={cn(
                        'text-sm',
                        step.status === 'completed' ? 'text-gray-500 line-through' : step.status === 'in_progress' ? 'font-medium text-gray-900' : 'text-gray-400',
                      )}>
                        {step.name}
                      </span>
                    </div>
                  ))}
                </div>
                <div className="mt-4">
                  <div className="h-2 overflow-hidden rounded-full bg-gray-100">
                    <div
                      className="h-full rounded-full bg-indigo-500 transition-all"
                      style={{ width: `${(task.steps.filter((s) => s.status === 'completed').length / task.steps.length) * 100}%` }}
                    />
                  </div>
                  <p className="mt-1 text-xs text-gray-500">
                    {task.steps.filter((s) => s.status === 'completed').length} of {task.steps.length} steps completed
                  </p>
                </div>
              </div>
            ))
          )}
        </div>
      ) : (
        <>
          {/* Filters */}
          <div className="flex flex-wrap gap-3">
            <div className="relative min-w-[200px] flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by name, email, or domain..."
                className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <select
              value={tierFilter}
              onChange={(e) => setTierFilter(e.target.value)}
              className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
            >
              <option value="">All Tiers</option>
              <option value="free">Free</option>
              <option value="starter">Starter</option>
              <option value="professional">Professional</option>
              <option value="enterprise">Enterprise</option>
            </select>
          </div>

          {/* Pending Approval Banner */}
          {activeTab === 'pending' && filtered.length > 0 && (
            <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4">
              <div className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-yellow-600" />
                <p className="text-sm font-medium text-yellow-800">
                  {filtered.length} tenant(s) awaiting your approval. Review and approve or reject each registration below.
                </p>
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
              <div className="py-16 text-center">
                <Building2 className="mx-auto h-12 w-12 text-gray-300" />
                <p className="mt-2 text-sm text-gray-400">
                  {activeTab === 'pending' ? 'No pending approvals.' : 'No tenants found.'}
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                      <th className="px-6 py-3 font-medium">Store</th>
                      <th className="px-6 py-3 font-medium">Contact</th>
                      <th className="px-6 py-3 font-medium">Domain</th>
                      <th className="px-6 py-3 font-medium">Tier</th>
                      <th className="px-6 py-3 font-medium">MRR</th>
                      <th className="px-6 py-3 font-medium">Status</th>
                      <th className="px-6 py-3 font-medium">Created</th>
                      <th className="px-6 py-3 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((tenant) => (
                      <tr key={tenant.id} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-3">
                            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-100 text-sm font-bold text-indigo-600">
                              {tenant.name.charAt(0)}
                            </div>
                            <div>
                              <Link href={`/super-admin/tenants/${tenant.id}`} className="text-sm font-medium text-gray-900 hover:text-indigo-600">
                                {tenant.name}
                              </Link>
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-500">{tenant.email}</td>
                        <td className="px-6 py-4">
                          {tenant.domain ? (
                            <span className="inline-flex items-center gap-1 text-sm text-gray-500">
                              <Globe className="h-3.5 w-3.5" />
                              {tenant.domain}
                            </span>
                          ) : (
                            <span className="text-sm text-gray-300">—</span>
                          )}
                        </td>
                        <td className="px-6 py-4">
                          <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', tierColor[tenant.tier])}>
                            {tenant.tier}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-sm font-medium text-gray-900">
                          {formatCurrency(tierPrices[tenant.tier] || 0)}
                        </td>
                        <td className="px-6 py-4">
                          <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(tenant.status))}>
                            {tenant.status}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-500">{formatDate(tenant.created_at)}</td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-1">
                            {tenant.status === 'pending' ? (
                              <>
                                <button
                                  onClick={() => handleApprove(tenant)}
                                  className="rounded-lg p-1.5 text-green-500 transition-colors hover:bg-green-50 hover:text-green-700"
                                  title="Approve"
                                >
                                  <CheckCircle className="h-4 w-4" />
                                </button>
                                <button
                                  onClick={() => { setShowApprovalModal(tenant); setApprovalNote(''); }}
                                  className="rounded-lg p-1.5 text-red-400 transition-colors hover:bg-red-50 hover:text-red-600"
                                  title="Review & Reject"
                                >
                                  <XCircle className="h-4 w-4" />
                                </button>
                              </>
                            ) : (
                              <>
                                <Link
                                  href={`/super-admin/tenants/${tenant.id}`}
                                  className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                                  title="View Details"
                                >
                                  <Eye className="h-4 w-4" />
                                </Link>
                                <button
                                  onClick={() => { setEditingTenant(tenant); setEditStatus(tenant.status); setEditTier(tenant.tier); }}
                                  className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                                  title="Edit"
                                >
                                  <Pencil className="h-4 w-4" />
                                </button>
                                <button
                                  onClick={() => handleDelete(tenant)}
                                  disabled={deleting === tenant.id}
                                  className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 disabled:opacity-50"
                                  title="Delete"
                                >
                                  {deleting === tenant.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                                </button>
                              </>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}

      {/* ── Create Tenant Modal ─────────────────────────────── */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-lg rounded-2xl border border-gray-200 bg-white shadow-xl">
            <div className="border-b border-gray-100 px-6 py-4">
              <h3 className="text-lg font-semibold text-gray-900">Create New Tenant</h3>
              <p className="text-sm text-gray-500">Provision a new store on the platform.</p>
            </div>
            <form onSubmit={handleCreate} className="space-y-4 p-6">
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2">
                  <label className="mb-1 block text-sm font-medium text-gray-700">Store Name *</label>
                  <input value={newName} onChange={(e) => setNewName(e.target.value)} required placeholder="e.g. Aarong Digital"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500" />
                </div>
                <div className="col-span-2">
                  <label className="mb-1 block text-sm font-medium text-gray-700">Contact Email *</label>
                  <input value={newEmail} onChange={(e) => setNewEmail(e.target.value)} required type="email" placeholder="admin@store.com"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500" />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Custom Domain</label>
                  <input value={newDomain} onChange={(e) => setNewDomain(e.target.value)} placeholder="store.example.com"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500" />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Subscription Tier</label>
                  <select value={newTier} onChange={(e) => setNewTier(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none">
                    <option value="free">Free — {formatCurrency(0)}/mo</option>
                    <option value="starter">Starter — {formatCurrency(2999)}/mo</option>
                    <option value="professional">Professional — {formatCurrency(9999)}/mo</option>
                    <option value="enterprise">Enterprise — {formatCurrency(29999)}/mo</option>
                  </select>
                </div>
              </div>

              <div className="rounded-lg border border-gray-100 bg-gray-50 p-4">
                <h4 className="text-sm font-medium text-gray-700 mb-3">Provisioning Options</h4>
                <label className="flex items-center gap-2 text-sm text-gray-600">
                  <input type="checkbox" checked={newAutoProvision} onChange={(e) => setNewAutoProvision(e.target.checked)}
                    className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500" />
                  Auto-provision store (create database, configure defaults, generate SSL)
                </label>
              </div>

              <div className="flex gap-3 border-t border-gray-100 pt-4">
                <button type="submit" disabled={creating}
                  className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:opacity-50">
                  {creating && <Loader2 className="h-4 w-4 animate-spin" />}
                  Create & Provision
                </button>
                <button type="button" onClick={() => setShowCreateModal(false)}
                  className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50">
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── Edit Tenant Modal ─────────────────────────────── */}
      {editingTenant && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-sm rounded-2xl border border-gray-200 bg-white p-6 shadow-xl">
            <h3 className="text-base font-semibold text-gray-900 mb-1">Edit: {editingTenant.name}</h3>
            <p className="text-xs text-gray-500 mb-4">Update status, tier, and governance settings.</p>
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
              {editStatus === 'suspended' && (
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Suspension Reason</label>
                  <textarea rows={2} placeholder="Reason for suspension..."
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none" />
                </div>
              )}
              <div className="flex gap-3 pt-2">
                <button type="submit" className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition-colors">
                  Save Changes
                </button>
                <button type="button" onClick={() => setEditingTenant(null)}
                  className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── Approval Review Modal ─────────────────────────── */}
      {showApprovalModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-md rounded-2xl border border-gray-200 bg-white shadow-xl">
            <div className="border-b border-gray-100 px-6 py-4">
              <h3 className="text-lg font-semibold text-gray-900">Review Registration</h3>
            </div>
            <div className="p-6 space-y-4">
              <div className="rounded-lg border border-gray-100 bg-gray-50 p-4 space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Store Name</span>
                  <span className="font-medium text-gray-900">{showApprovalModal.name}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Email</span>
                  <span className="font-medium text-gray-900">{showApprovalModal.email}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Requested Tier</span>
                  <span className={cn('rounded-full px-2 py-0.5 text-xs font-medium capitalize', tierColor[showApprovalModal.tier])}>
                    {showApprovalModal.tier}
                  </span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Applied On</span>
                  <span className="font-medium text-gray-900">{formatDate(showApprovalModal.created_at)}</span>
                </div>
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Review Note</label>
                <textarea value={approvalNote} onChange={(e) => setApprovalNote(e.target.value)}
                  rows={3} placeholder="Optional note for the tenant..."
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none" />
              </div>

              <div className="flex gap-3">
                <button onClick={() => handleApprove(showApprovalModal)}
                  className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 transition-colors">
                  <CheckCircle className="h-4 w-4" /> Approve
                </button>
                <button onClick={() => handleReject(showApprovalModal)}
                  className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 transition-colors">
                  <XCircle className="h-4 w-4" /> Reject
                </button>
                <button onClick={() => setShowApprovalModal(null)}
                  className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
