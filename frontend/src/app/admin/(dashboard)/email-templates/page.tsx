'use client';

/**
 * Admin list page for notification templates.
 *
 * Templates persisted here override the hardcoded RenderEmailHTML calls in
 * notification-service/consumer.go. The list is unpaginated because we
 * expect at most a few dozen templates per tenant; once they become active
 * the consumer will pick them up on its next event.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import {
  Mail,
  Plus,
  Search,
  Loader2,
  Edit2,
  Trash2,
  ToggleLeft,
  ToggleRight,
  Sparkles,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  notificationTemplateApi,
  type NotificationChannel,
  type NotificationTemplate,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';
import { useConfirm } from '@/stores/confirm';
import { TEMPLATE_TYPES } from '@/components/admin/email-template-editor';

function typeLabel(type: string): string {
  return TEMPLATE_TYPES.find((t) => t.value === type)?.label || type;
}

function channelBadgeClass(channel: NotificationChannel): string {
  switch (channel) {
    case 'email':
      return 'bg-blue-100 text-blue-800';
    case 'sms':
      return 'bg-purple-100 text-purple-800';
    case 'push':
      return 'bg-amber-100 text-amber-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
}

export default function EmailTemplatesPage() {
  const { tenantId, token } = useAuthStore();
  const confirm = useConfirm();

  const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('');
  const [channelFilter, setChannelFilter] = useState<string>('');
  const [activeFilter, setActiveFilter] = useState<'' | 'active' | 'inactive'>('');
  const [toggling, setToggling] = useState<string | null>(null);
  // Installer dialog state. We use a bespoke inline dialog (rather than the
  // generic useConfirm) because we need two confirmation paths: a plain
  // "Install" (force=false) and "Install and overwrite" (force=true).
  const [installerOpen, setInstallerOpen] = useState(false);
  const [installing, setInstalling] = useState(false);

  const load = useCallback(async () => {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const data = await notificationTemplateApi.list(tenantId, token);
      setTemplates(Array.isArray(data) ? data : []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load templates');
      setTemplates([]);
    } finally {
      setLoading(false);
    }
  }, [tenantId, token]);

  useEffect(() => {
    load();
  }, [load]);

  const filtered = useMemo(() => {
    return templates.filter((t) => {
      if (typeFilter && t.type !== typeFilter) return false;
      if (channelFilter && t.channel !== channelFilter) return false;
      if (activeFilter === 'active' && !t.is_active) return false;
      if (activeFilter === 'inactive' && t.is_active) return false;
      if (search) {
        const q = search.toLowerCase();
        return (
          t.name.toLowerCase().includes(q) ||
          (t.subject_template || '').toLowerCase().includes(q) ||
          String(t.type).toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [templates, search, typeFilter, channelFilter, activeFilter]);

  async function handleToggleActive(t: NotificationTemplate) {
    if (!tenantId || !token) return;
    setToggling(t.id);
    try {
      const updated = await notificationTemplateApi.update(
        t.id,
        { is_active: !t.is_active },
        tenantId,
        token,
      );
      setTemplates((prev) => prev.map((x) => (x.id === t.id ? updated : x)));
      toast.success(updated.is_active ? 'Template activated' : 'Template deactivated');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Toggle failed');
    } finally {
      setToggling(null);
    }
  }

  // handleInstallDefaults dispatches the install request and refreshes the
  // list on success. Force=true overwrites existing templates that share a
  // (type, channel) pair; force=false skips them. Either path is non-
  // destructive in the sense that admins can re-edit or delete after install.
  async function handleInstallDefaults(force: boolean) {
    if (!tenantId || !token) {
      toast.error('Sign in required');
      return;
    }
    setInstalling(true);
    try {
      const res = await notificationTemplateApi.installDefaults(force, tenantId, token);
      const parts: string[] = [];
      if (res.created) parts.push(`${res.created} created`);
      if (res.updated) parts.push(`${res.updated} updated`);
      if (res.skipped) parts.push(`${res.skipped} skipped`);
      toast.success(`Installed starter pack: ${parts.join(', ') || 'no changes'}`);
      setInstallerOpen(false);
      // Re-fetch so the new rows appear immediately. We could splice them in
      // locally from res.templates but a full reload keeps sort/timestamps
      // consistent with the rest of the page.
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Install failed');
    } finally {
      setInstalling(false);
    }
  }

  async function handleDelete(t: NotificationTemplate) {
    if (!tenantId || !token) return;
    const ok = await confirm({
      title: 'Delete template?',
      description: `"${t.name}" will be removed. The hardcoded fallback will be used for ${typeLabel(t.type)} notifications.`,
      variant: 'danger',
      confirmLabel: 'Delete',
    });
    if (!ok) return;
    try {
      await notificationTemplateApi.delete(t.id, tenantId, token);
      setTemplates((prev) => prev.filter((x) => x.id !== t.id));
      toast.success('Template deleted');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Delete failed');
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-text">Email templates</h1>
          <p className="mt-1 text-sm text-text-secondary">
            Customize transactional emails. Active templates override the
            built-in HTML used by the notification service.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/*
            The header-level "Install defaults" button is the secondary path —
            tenants that already have some templates use it to top up with
            anything missing. The empty-state CTA below is the primary entry
            point for brand-new tenants.
          */}
          {templates.length > 0 && (
            <button
              type="button"
              onClick={() => setInstallerOpen(true)}
              className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface px-3 py-2 text-sm font-medium text-text-secondary shadow-sm transition-colors hover:bg-surface-hover hover:text-text"
            >
              <Sparkles className="h-4 w-4" />
              Install defaults
            </button>
          )}
          <Link
            href="/admin/email-templates/new"
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-dark"
          >
            <Plus className="h-4 w-4" />
            New template
          </Link>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative min-w-[200px] flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search templates…"
            className="w-full rounded-lg border border-border bg-surface py-2 pl-9 pr-3 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none"
        >
          <option value="">All types</option>
          {TEMPLATE_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
        <select
          value={channelFilter}
          onChange={(e) => setChannelFilter(e.target.value)}
          className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none"
        >
          <option value="">All channels</option>
          <option value="email">Email</option>
          <option value="sms">SMS</option>
          <option value="push">Push</option>
        </select>
        <select
          value={activeFilter}
          onChange={(e) => setActiveFilter(e.target.value as typeof activeFilter)}
          className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none"
        >
          <option value="">All</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </div>

      {/* List */}
      {loading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          hasAnyTemplates={templates.length > 0}
          onInstallDefaults={() => setInstallerOpen(true)}
        />
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden overflow-hidden rounded-xl border border-border bg-surface shadow-sm md:block">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border text-left text-sm text-text-secondary">
                    <th className="px-6 py-3 font-medium">Name</th>
                    <th className="px-6 py-3 font-medium">Type</th>
                    <th className="px-6 py-3 font-medium">Channel</th>
                    <th className="px-6 py-3 font-medium">Active</th>
                    <th className="px-6 py-3 font-medium text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((t) => (
                    <tr
                      key={t.id}
                      className="border-b border-border-light transition-colors hover:bg-surface-hover"
                    >
                      <td className="px-6 py-4">
                        <Link
                          href={`/admin/email-templates/${t.id}`}
                          className="text-sm font-medium text-text hover:text-primary"
                        >
                          {t.name}
                        </Link>
                        {t.subject_template && (
                          <div className="text-xs text-text-muted line-clamp-1">
                            {t.subject_template}
                          </div>
                        )}
                      </td>
                      <td className="px-6 py-4 text-sm text-text-secondary">
                        {typeLabel(t.type)}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={cn(
                            'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                            channelBadgeClass(t.channel),
                          )}
                        >
                          {t.channel}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <button
                          type="button"
                          onClick={() => handleToggleActive(t)}
                          disabled={toggling === t.id}
                          className="inline-flex items-center gap-1.5 text-sm text-text-secondary hover:text-primary disabled:opacity-50"
                          aria-label={t.is_active ? 'Deactivate' : 'Activate'}
                        >
                          {t.is_active ? (
                            <ToggleRight className="h-5 w-5 text-primary" />
                          ) : (
                            <ToggleLeft className="h-5 w-5 text-text-muted" />
                          )}
                          {t.is_active ? 'Active' : 'Inactive'}
                        </button>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center justify-end gap-2">
                          <Link
                            href={`/admin/email-templates/${t.id}`}
                            className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-primary"
                            aria-label="Edit"
                          >
                            <Edit2 className="h-4 w-4" />
                          </Link>
                          <button
                            type="button"
                            onClick={() => handleDelete(t)}
                            className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-red-600"
                            aria-label="Delete"
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
          </div>

          {/* Mobile cards */}
          <div className="space-y-3 md:hidden">
            {filtered.map((t) => (
              <div
                key={t.id}
                className="rounded-xl border border-border bg-surface p-4 shadow-sm"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <Link
                      href={`/admin/email-templates/${t.id}`}
                      className="block truncate text-sm font-semibold text-text hover:text-primary"
                    >
                      {t.name}
                    </Link>
                    <p className="mt-0.5 text-xs text-text-muted">{typeLabel(t.type)}</p>
                  </div>
                  <span
                    className={cn(
                      'flex-shrink-0 rounded-full px-2.5 py-0.5 text-[10px] font-medium capitalize',
                      channelBadgeClass(t.channel),
                    )}
                  >
                    {t.channel}
                  </span>
                </div>
                {t.subject_template && (
                  <p className="mt-2 truncate text-xs text-text-secondary">
                    {t.subject_template}
                  </p>
                )}
                <div className="mt-3 flex items-center justify-between">
                  <button
                    type="button"
                    onClick={() => handleToggleActive(t)}
                    disabled={toggling === t.id}
                    className="inline-flex items-center gap-1.5 text-xs text-text-secondary hover:text-primary disabled:opacity-50"
                  >
                    {t.is_active ? (
                      <ToggleRight className="h-4 w-4 text-primary" />
                    ) : (
                      <ToggleLeft className="h-4 w-4 text-text-muted" />
                    )}
                    {t.is_active ? 'Active' : 'Inactive'}
                  </button>
                  <div className="flex items-center gap-1">
                    <Link
                      href={`/admin/email-templates/${t.id}`}
                      className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-primary"
                      aria-label="Edit"
                    >
                      <Edit2 className="h-4 w-4" />
                    </Link>
                    <button
                      type="button"
                      onClick={() => handleDelete(t)}
                      className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-red-600"
                      aria-label="Delete"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {installerOpen && (
        <InstallerDialog
          installing={installing}
          onCancel={() => (installing ? undefined : setInstallerOpen(false))}
          onInstall={() => handleInstallDefaults(false)}
          onForceInstall={() => handleInstallDefaults(true)}
        />
      )}
    </div>
  );
}

interface EmptyStateProps {
  hasAnyTemplates: boolean;
  onInstallDefaults: () => void;
}

// EmptyState appears both when the tenant truly has no templates and when
// every template is filtered out by the search/type/channel selectors. We
// only surface the install CTA in the genuine zero-state — once the tenant
// has any templates the header-level button is the right entry point.
function EmptyState({ hasAnyTemplates, onInstallDefaults }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-surface py-16 text-center shadow-sm">
      <Mail className="h-10 w-10 text-text-muted" />
      <p className="mt-3 text-sm font-medium text-text">
        {hasAnyTemplates ? 'No templates match your filters' : 'No templates yet'}
      </p>
      <p className="mt-1 max-w-md text-xs text-text-muted">
        {hasAnyTemplates
          ? 'Try clearing the filters above, or create a new template.'
          : 'Get started in one click with our pre-designed starter pack, or build your own from scratch.'}
      </p>
      <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
        {!hasAnyTemplates && (
          <button
            type="button"
            onClick={onInstallDefaults}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-dark"
          >
            <Sparkles className="h-4 w-4" />
            Install starter pack
          </button>
        )}
        <Link
          href="/admin/email-templates/new"
          className={cn(
            'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium shadow-sm transition-colors',
            hasAnyTemplates
              ? 'bg-primary text-white hover:bg-primary-dark'
              : 'border border-border bg-surface text-text-secondary hover:bg-surface-hover hover:text-text',
          )}
        >
          <Plus className="h-4 w-4" />
          New template
        </Link>
      </div>
    </div>
  );
}

interface InstallerDialogProps {
  installing: boolean;
  onCancel: () => void;
  onInstall: () => void;
  onForceInstall: () => void;
}

// InstallerDialog renders the two-action confirmation modal for the starter
// pack installer. We roll a bespoke component here instead of useConfirm
// because the latter only supports a single confirm path, and we want
// "Install" and "Install and overwrite" to be visually adjacent so the admin
// can compare them at a glance.
function InstallerDialog({ installing, onCancel, onInstall, onForceInstall }: InstallerDialogProps) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="installer-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={onCancel}
    >
      <div
        className="w-full max-w-md rounded-xl border border-border bg-surface p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start gap-3">
          <span className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            <Sparkles className="h-5 w-5" />
          </span>
          <div className="min-w-0">
            <h2 id="installer-title" className="text-lg font-semibold text-text">
              Install starter email templates?
            </h2>
            <p className="mt-1 text-sm text-text-secondary">
              This will create 10 ready-to-use templates (welcome, order
              confirmation, shipping, password reset, and more). Existing
              templates with the same type won&apos;t be overwritten.
            </p>
          </div>
        </div>
        <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={onCancel}
            disabled={installing}
            className="rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover hover:text-text disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onForceInstall}
            disabled={installing}
            className="rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover hover:text-text disabled:opacity-50"
            title="Replace existing templates of the same type with the starter versions"
          >
            Install and overwrite existing
          </button>
          <button
            type="button"
            onClick={onInstall}
            disabled={installing}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-dark disabled:opacity-50"
          >
            {installing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            {installing ? 'Installing…' : 'Install'}
          </button>
        </div>
      </div>
    </div>
  );
}
