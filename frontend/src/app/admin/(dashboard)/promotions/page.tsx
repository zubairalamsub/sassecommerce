'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import Link from 'next/link';
import {
  BadgePercent,
  Ticket,
  Plus,
  Search,
  Loader2,
  Copy,
  Calendar,
  TrendingUp,
  AlertCircle,
  ChevronDown,
  Percent,
  DollarSign,
  Truck,
  Tag,
} from 'lucide-react';
import { cn, formatCurrency, formatDate } from '@/lib/utils';
import {
  promotionApi,
  type Promotion,
  type Coupon,
  type DiscountType,
  type PromotionStatus,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';

type Tab = 'promotions' | 'coupons';

function discountTypeIcon(type: DiscountType) {
  switch (type) {
    case 'percentage':
      return Percent;
    case 'fixed_amount':
      return DollarSign;
    case 'free_shipping':
      return Truck;
    default:
      return Tag;
  }
}

function discountTypeLabel(type: DiscountType): string {
  switch (type) {
    case 'percentage':
      return 'Percentage';
    case 'fixed_amount':
      return 'Fixed amount';
    case 'free_shipping':
      return 'Free shipping';
    default:
      return type;
  }
}

function formatDiscount(p: { discount_type: DiscountType; discount_value: number }): string {
  switch (p.discount_type) {
    case 'percentage':
      return `${p.discount_value}% off`;
    case 'fixed_amount':
      return `${formatCurrency(p.discount_value)} off`;
    case 'free_shipping':
      return 'Free shipping';
    default:
      return String(p.discount_value);
  }
}

function statusBadgeClass(status: PromotionStatus | string): string {
  switch (status) {
    case 'active':
      return 'bg-green-100 text-green-800';
    case 'draft':
      return 'bg-yellow-100 text-yellow-800';
    case 'expired':
      return 'bg-gray-100 text-gray-800';
    case 'disabled':
      return 'bg-red-100 text-red-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
}

/**
 * Derive a UI-facing status from a promotion's dates and persisted status.
 * Backend only flips to "expired" when activity happens; UI also needs
 * "scheduled" for future-dated promotions.
 */
function effectiveStatus(p: Promotion): PromotionStatus | 'scheduled' {
  const now = Date.now();
  const start = new Date(p.start_date).getTime();
  const end = new Date(p.end_date).getTime();
  if (p.status === 'disabled') return 'disabled';
  if (end < now) return 'expired';
  if (start > now) return 'scheduled';
  return p.status;
}

export default function PromotionsPage() {
  const { tenantId, token } = useAuthStore();
  const [activeTab, setActiveTab] = useState<Tab>('promotions');
  const [promotions, setPromotions] = useState<Promotion[]>([]);
  const [coupons, setCoupons] = useState<Coupon[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<'' | 'active' | 'scheduled' | 'expired'>('');
  const [menuOpen, setMenuOpen] = useState(false);

  const load = useCallback(async () => {
    if (!tenantId) return;
    setLoading(true);
    try {
      // Backend exposes only /promotions/active — there's no list-all endpoint.
      // We hydrate coupons from the per-promotion shape returned (some deployments
      // include them) or leave the coupons list empty until users create them here.
      const promos = await promotionApi.listActivePromotions(tenantId, token || undefined);
      setPromotions(Array.isArray(promos) ? promos : []);
    } catch {
      setPromotions([]);
    } finally {
      setLoading(false);
    }
  }, [tenantId, token]);

  useEffect(() => {
    load();
  }, [load]);

  // Pull any locally-created coupons from session so freshly created ones show up
  // (backend lacks a list endpoint).
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      const raw = sessionStorage.getItem('admin:recent_coupons');
      if (raw) setCoupons(JSON.parse(raw) as Coupon[]);
    } catch {
      /* ignore */
    }
  }, []);

  const stats = useMemo(() => {
    const now = Date.now();
    const weekFromNow = now + 7 * 24 * 60 * 60 * 1000;
    const active = promotions.filter((p) => effectiveStatus(p) === 'active').length;
    const expiringSoon = promotions.filter((p) => {
      const end = new Date(p.end_date).getTime();
      return end > now && end <= weekFromNow;
    }).length;
    const totalRedemptions = coupons.reduce((sum, c) => sum + (c.used_count || 0), 0);
    const totalDiscount = coupons.reduce((sum, c) => {
      const v = c.discount_value || 0;
      return sum + v * (c.used_count || 0);
    }, 0);
    return [
      { title: 'Active promotions', value: String(active), icon: BadgePercent },
      { title: 'Total redemptions', value: String(totalRedemptions), icon: TrendingUp },
      { title: 'Total discount given', value: formatCurrency(totalDiscount), icon: DollarSign },
      { title: 'Expiring this week', value: String(expiringSoon), icon: AlertCircle },
    ];
  }, [promotions, coupons]);

  const filteredPromotions = useMemo(() => {
    return promotions.filter((p) => {
      const status = effectiveStatus(p);
      if (statusFilter && status !== statusFilter) return false;
      if (search) {
        const q = search.toLowerCase();
        return (
          p.name.toLowerCase().includes(q) ||
          (p.description || '').toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [promotions, search, statusFilter]);

  const filteredCoupons = useMemo(() => {
    return coupons.filter((c) => {
      if (statusFilter === 'active' && !c.is_active) return false;
      if (statusFilter === 'expired' && c.is_active) return false;
      if (search) {
        return c.code.toLowerCase().includes(search.toLowerCase());
      }
      return true;
    });
  }, [coupons, search, statusFilter]);

  function handleCopyCode(code: string) {
    if (typeof navigator === 'undefined' || !navigator.clipboard) return;
    navigator.clipboard.writeText(code).then(
      () => toast.success('Code copied'),
      () => toast.error('Could not copy code'),
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-text">Promotions & Coupons</h1>
          <p className="mt-1 text-sm text-text-secondary">
            Create discounts, gift codes, and special offers for your store.
          </p>
        </div>
        <div className="relative">
          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-dark"
          >
            <Plus className="h-4 w-4" />
            New
            <ChevronDown className="h-4 w-4" />
          </button>
          {menuOpen && (
            <>
              <div
                className="fixed inset-0 z-10"
                onClick={() => setMenuOpen(false)}
                aria-hidden
              />
              <div className="absolute right-0 z-20 mt-2 w-52 overflow-hidden rounded-lg border border-border bg-surface shadow-lg">
                <Link
                  href="/admin/promotions/new"
                  className="flex items-center gap-2 px-4 py-2.5 text-sm text-text hover:bg-surface-hover"
                  onClick={() => setMenuOpen(false)}
                >
                  <BadgePercent className="h-4 w-4 text-primary" />
                  New Promotion
                </Link>
                <Link
                  href="/admin/promotions/new?type=coupon"
                  className="flex items-center gap-2 border-t border-border px-4 py-2.5 text-sm text-text hover:bg-surface-hover"
                  onClick={() => setMenuOpen(false)}
                >
                  <Ticket className="h-4 w-4 text-primary" />
                  New Coupon
                </Link>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((s) => {
          const Icon = s.icon;
          return (
            <div
              key={s.title}
              className="rounded-xl border border-border bg-surface p-5 shadow-sm"
            >
              <div className="flex items-center gap-3">
                <span className="rounded-lg bg-primary-light p-2.5">
                  <Icon className="h-5 w-5 text-primary" />
                </span>
                <div className="min-w-0">
                  <p className="text-xs text-text-secondary">{s.title}</p>
                  <p className="truncate text-xl font-semibold text-text">{s.value}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Tabs */}
      <div className="border-b border-border">
        <nav className="-mb-px flex gap-6">
          {(['promotions', 'coupons'] as Tab[]).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={cn(
                'border-b-2 pb-3 text-sm font-medium capitalize transition-colors',
                activeTab === tab
                  ? 'border-primary text-primary'
                  : 'border-transparent text-text-secondary hover:border-border hover:text-text',
              )}
            >
              {tab}
              <span className="ml-2 rounded-full bg-surface-hover px-2 py-0.5 text-xs text-text-muted">
                {tab === 'promotions' ? promotions.length : coupons.length}
              </span>
            </button>
          ))}
        </nav>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative min-w-[200px] flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={
              activeTab === 'promotions'
                ? 'Search promotions...'
                : 'Search coupon codes...'
            }
            className="w-full rounded-lg border border-border bg-surface py-2 pl-9 pr-3 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}
          className="rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none"
        >
          <option value="">All statuses</option>
          <option value="active">Active</option>
          <option value="scheduled">Scheduled</option>
          <option value="expired">Expired</option>
        </select>
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : activeTab === 'promotions' ? (
        <PromotionsList items={filteredPromotions} />
      ) : (
        <CouponsList items={filteredCoupons} onCopy={handleCopyCode} />
      )}
    </div>
  );
}

function PromotionsList({ items }: { items: Promotion[] }) {
  if (items.length === 0) {
    return <EmptyState message="No promotions yet" hint="Create one to start offering discounts" />;
  }

  return (
    <>
      {/* Desktop table */}
      <div className="hidden overflow-hidden rounded-xl border border-border bg-surface shadow-sm md:block">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border text-left text-sm text-text-secondary">
                <th className="px-6 py-3 font-medium">Name</th>
                <th className="px-6 py-3 font-medium">Type</th>
                <th className="px-6 py-3 font-medium">Discount</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Starts</th>
                <th className="px-6 py-3 font-medium">Ends</th>
              </tr>
            </thead>
            <tbody>
              {items.map((p) => {
                const Icon = discountTypeIcon(p.discount_type);
                const status = effectiveStatus(p);
                return (
                  <tr
                    key={p.id}
                    className="border-b border-border-light transition-colors hover:bg-surface-hover"
                  >
                    <td className="px-6 py-4">
                      <div className="text-sm font-medium text-text">{p.name}</div>
                      {p.description && (
                        <div className="text-xs text-text-muted line-clamp-1">{p.description}</div>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center gap-1.5 text-sm text-text-secondary">
                        <Icon className="h-4 w-4 text-primary" />
                        {discountTypeLabel(p.discount_type)}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm font-semibold text-text">
                      {formatDiscount(p)}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          statusBadgeClass(status),
                        )}
                      >
                        {status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-text-secondary">{formatDate(p.start_date)}</td>
                    <td className="px-6 py-4 text-sm text-text-secondary">{formatDate(p.end_date)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Mobile cards */}
      <div className="space-y-3 md:hidden">
        {items.map((p) => {
          const Icon = discountTypeIcon(p.discount_type);
          const status = effectiveStatus(p);
          return (
            <div
              key={p.id}
              className="rounded-xl border border-border bg-surface p-4 shadow-sm"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <Icon className="h-4 w-4 flex-shrink-0 text-primary" />
                    <p className="truncate text-sm font-semibold text-text">{p.name}</p>
                  </div>
                  {p.description && (
                    <p className="mt-1 text-xs text-text-muted line-clamp-2">{p.description}</p>
                  )}
                </div>
                <span
                  className={cn(
                    'flex-shrink-0 rounded-full px-2.5 py-0.5 text-[10px] font-medium capitalize',
                    statusBadgeClass(status),
                  )}
                >
                  {status}
                </span>
              </div>
              <div className="mt-3 text-lg font-bold text-primary">{formatDiscount(p)}</div>
              <div className="mt-2 flex items-center gap-1.5 text-xs text-text-muted">
                <Calendar className="h-3.5 w-3.5" />
                {formatDate(p.start_date)} – {formatDate(p.end_date)}
              </div>
            </div>
          );
        })}
      </div>
    </>
  );
}

function CouponsList({
  items,
  onCopy,
}: {
  items: Coupon[];
  onCopy: (code: string) => void;
}) {
  if (items.length === 0) {
    return (
      <EmptyState
        message="No coupons yet"
        hint="Coupons you create will appear here. The backend does not yet expose a list endpoint, so existing codes from other sessions are not shown."
      />
    );
  }

  return (
    <>
      {/* Desktop table */}
      <div className="hidden overflow-hidden rounded-xl border border-border bg-surface shadow-sm md:block">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border text-left text-sm text-text-secondary">
                <th className="px-6 py-3 font-medium">Code</th>
                <th className="px-6 py-3 font-medium">Type</th>
                <th className="px-6 py-3 font-medium">Value</th>
                <th className="px-6 py-3 font-medium">Usage</th>
                <th className="px-6 py-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {items.map((c) => {
                const Icon = c.discount_type ? discountTypeIcon(c.discount_type) : Tag;
                return (
                  <tr
                    key={c.id}
                    className="border-b border-border-light transition-colors hover:bg-surface-hover"
                  >
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <code className="rounded bg-surface-hover px-2 py-1 font-mono text-sm text-text">
                          {c.code}
                        </code>
                        <button
                          type="button"
                          onClick={() => onCopy(c.code)}
                          className="rounded-lg p-1 text-text-muted transition-colors hover:bg-surface-hover hover:text-primary"
                          aria-label="Copy code"
                        >
                          <Copy className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center gap-1.5 text-sm text-text-secondary">
                        <Icon className="h-4 w-4 text-primary" />
                        {c.discount_type ? discountTypeLabel(c.discount_type) : '—'}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm font-semibold text-text">
                      {c.discount_type
                        ? formatDiscount({
                            discount_type: c.discount_type,
                            discount_value: c.discount_value || 0,
                          })
                        : '—'}
                    </td>
                    <td className="px-6 py-4 text-sm text-text-secondary">
                      {c.used_count} / {c.max_uses === 0 ? '∞' : c.max_uses}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                          c.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800',
                        )}
                      >
                        {c.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Mobile cards */}
      <div className="space-y-3 md:hidden">
        {items.map((c) => (
          <div key={c.id} className="rounded-xl border border-border bg-surface p-4 shadow-sm">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-2 min-w-0">
                <code className="truncate rounded bg-surface-hover px-2 py-1 font-mono text-sm text-text">
                  {c.code}
                </code>
                <button
                  type="button"
                  onClick={() => onCopy(c.code)}
                  className="rounded-lg p-1 text-text-muted transition-colors hover:bg-surface-hover hover:text-primary"
                  aria-label="Copy code"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
              <span
                className={cn(
                  'flex-shrink-0 rounded-full px-2.5 py-0.5 text-[10px] font-medium',
                  c.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800',
                )}
              >
                {c.is_active ? 'Active' : 'Inactive'}
              </span>
            </div>
            <div className="mt-2 text-sm text-text-secondary">
              {c.discount_type
                ? formatDiscount({
                    discount_type: c.discount_type,
                    discount_value: c.discount_value || 0,
                  })
                : 'Linked to promotion'}
            </div>
            <div className="mt-1 text-xs text-text-muted">
              Used {c.used_count} of {c.max_uses === 0 ? 'unlimited' : c.max_uses}
            </div>
          </div>
        ))}
      </div>
    </>
  );
}

function EmptyState({ message, hint }: { message: string; hint?: string }) {
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-surface py-16 text-center shadow-sm">
      <BadgePercent className="h-10 w-10 text-text-muted" />
      <p className="mt-3 text-sm font-medium text-text">{message}</p>
      {hint && <p className="mt-1 max-w-md text-xs text-text-muted">{hint}</p>}
    </div>
  );
}
