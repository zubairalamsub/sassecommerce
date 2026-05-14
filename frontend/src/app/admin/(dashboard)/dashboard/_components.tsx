'use client';

import {
  ArrowDownRight,
  ArrowUpRight,
  Bell,
  CheckCircle2,
  Clock,
  DollarSign,
  Package,
  ShoppingCart,
  Star,
  UserPlus,
  Users,
  XCircle,
} from 'lucide-react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import { Line, LineChart, ResponsiveContainer } from 'recharts';
import { cn, formatCurrency } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Shared types & helpers
// ---------------------------------------------------------------------------

export type RangeKey = 'today' | '7d' | '30d' | '90d' | 'year' | 'custom';

export interface TopCustomer {
  id: string;
  name: string;
  orders: number;
  spent: number;
}

export interface LowStockEntry {
  id: string;
  productId: string;
  name: string;
  current: number;
  reorder: number;
  image?: string;
}

export interface ActivityItem {
  id: string;
  kind: 'order' | 'customer' | 'stock' | 'review';
  message: string;
  time: number;
  href?: string;
}

export const containerVariants = {
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { staggerChildren: 0.06 } },
};

export const cardVariants = {
  hidden: { opacity: 0, y: 16 },
  show: { opacity: 1, y: 0, transition: { duration: 0.35, ease: [0, 0, 0.2, 1] as const } },
};

export const sectionVariants = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { duration: 0.45, ease: [0, 0, 0.2, 1] as const } },
};

const feedItemVariants = {
  hidden: { opacity: 0, x: -8 },
  show: { opacity: 1, x: 0, transition: { duration: 0.3, ease: [0, 0, 0.2, 1] as const } },
};

export function relativeTime(epoch: number): string {
  const diff = Math.max(0, Date.now() - epoch);
  if (diff < 60_000) return 'just now';
  const mins = Math.floor(diff / 60_000);
  if (mins < 60) return `${mins} min ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs} hr ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

function initials(first?: string, last?: string, fallback = '?'): string {
  const a = (first || '').trim()[0];
  const b = (last || '').trim()[0];
  return ((a || '') + (b || '') || fallback).toUpperCase();
}

// ---------------------------------------------------------------------------
// Sparkline + change badge + skeletons
// ---------------------------------------------------------------------------

export function Sparkline({ values, color }: { values: number[]; color: string }) {
  if (values.length < 2) return <div className="h-6 w-[100px]" aria-hidden />;
  const data = values.map((v, i) => ({ i, v }));
  return (
    <div className="h-6 w-[100px]">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 2, right: 0, left: 0, bottom: 2 }}>
          <Line type="monotone" dataKey="v" stroke={color} strokeWidth={1.8} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

export function ChartTooltip({
  active,
  payload,
  label,
  prefix = '',
  isCurrency = false,
}: {
  active?: boolean;
  payload?: { value: number }[];
  label?: string;
  prefix?: string;
  isCurrency?: boolean;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-lg border border-border bg-surface px-3 py-2 text-xs shadow-lg">
      {label && <p className="mb-1 text-text-secondary">{label}</p>}
      <p className="font-semibold text-text">
        {prefix}
        {isCurrency ? formatCurrency(payload[0].value) : payload[0].value.toLocaleString()}
      </p>
    </div>
  );
}

export function StatCardSkeleton() {
  return (
    <div className="rounded-2xl border border-border bg-surface p-6">
      <div className="flex items-center justify-between">
        <div className="skeleton h-4 w-24" />
        <div className="skeleton h-10 w-10 rounded-xl" />
      </div>
      <div className="mt-3 skeleton h-7 w-32" />
      <div className="mt-3 flex items-center justify-between">
        <div className="skeleton h-3 w-16" />
        <div className="skeleton h-6 w-[100px]" />
      </div>
    </div>
  );
}

export function ChartSkeleton({ height = 260 }: { height?: number }) {
  return (
    <div className="rounded-2xl border border-border bg-surface p-6">
      <div className="skeleton mb-4 h-5 w-40" />
      <div className="skeleton w-full" style={{ height }} />
    </div>
  );
}

export function ChangeBadge({ value }: { value: number | null }) {
  if (value === null) return <span className="text-xs text-text-muted">—</span>;
  const positive = value >= 0;
  const Icon = positive ? ArrowUpRight : ArrowDownRight;
  return (
    <span
      className={cn(
        'inline-flex items-center gap-0.5 text-xs',
        positive ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400',
      )}
    >
      <Icon className="h-3 w-3" />
      {Math.abs(value).toFixed(1)}%
    </span>
  );
}

export function EmptyState({
  icon: Icon,
  text,
  className,
}: {
  icon: typeof DollarSign;
  text: string;
  className?: string;
}) {
  return (
    <div className={cn('flex flex-col items-center justify-center gap-2 py-8 text-center', className)}>
      <Icon className="h-8 w-8 text-text-muted" />
      <p className="text-sm text-text-muted">{text}</p>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Date range tabs + custom popover
// ---------------------------------------------------------------------------

export function DateRangeTabs({
  activeKey,
  onChange,
}: {
  activeKey: RangeKey;
  onChange: (k: RangeKey) => void;
}) {
  const tabs: { key: RangeKey; label: string }[] = [
    { key: 'today', label: 'Today' },
    { key: '7d', label: '7d' },
    { key: '30d', label: '30d' },
    { key: '90d', label: '90d' },
    { key: 'year', label: 'Year' },
    { key: 'custom', label: 'Custom' },
  ];
  return (
    <div className="inline-flex overflow-hidden rounded-lg border border-border bg-surface">
      {tabs.map((t) => (
        <button
          key={t.key}
          type="button"
          onClick={() => onChange(t.key)}
          className={cn(
            'px-2.5 py-1.5 text-xs font-medium transition-colors sm:px-3 sm:text-sm',
            activeKey === t.key ? 'bg-primary text-white' : 'text-text-secondary hover:bg-surface-hover',
          )}
        >
          {t.label}
        </button>
      ))}
    </div>
  );
}

export function CustomRangePopover({
  start,
  end,
  onStartChange,
  onEndChange,
  onCancel,
  onApply,
}: {
  start: string;
  end: string;
  onStartChange: (v: string) => void;
  onEndChange: (v: string) => void;
  onCancel: () => void;
  onApply: () => void;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: -8 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-wrap items-end gap-3 rounded-xl border border-border bg-surface p-4"
    >
      <label className="flex flex-col text-xs text-text-secondary">
        Start
        <input
          type="date"
          value={start}
          onChange={(e) => onStartChange(e.target.value)}
          className="mt-1 rounded-md border border-border bg-surface px-2 py-1.5 text-sm text-text"
        />
      </label>
      <label className="flex flex-col text-xs text-text-secondary">
        End
        <input
          type="date"
          value={end}
          onChange={(e) => onEndChange(e.target.value)}
          className="mt-1 rounded-md border border-border bg-surface px-2 py-1.5 text-sm text-text"
        />
      </label>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={onApply}
          className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-dark"
        >
          Apply
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-secondary hover:bg-surface-hover"
        >
          Cancel
        </button>
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------------------------
// Today snapshot
// ---------------------------------------------------------------------------

export function TodaySnapshot({
  revenue,
  orders,
  customers,
  lastOrderAt,
}: {
  revenue: number;
  orders: number;
  customers: number;
  lastOrderAt: number | null;
}) {
  const items = [
    { label: "Today's revenue", value: formatCurrency(revenue), icon: DollarSign },
    { label: "Today's orders", value: orders.toLocaleString(), icon: ShoppingCart },
    { label: 'New customers', value: customers.toLocaleString(), icon: UserPlus },
    { label: 'Last order', value: lastOrderAt ? relativeTime(lastOrderAt) : '—', icon: Clock },
  ];
  return (
    <div className="rounded-2xl border border-border bg-gradient-to-br from-primary-light to-surface p-4">
      <div className="-mx-1 flex gap-3 overflow-x-auto px-1 sm:grid sm:grid-cols-4 sm:gap-4 sm:overflow-visible">
        {items.map((it) => {
          const Icon = it.icon;
          return (
            <div
              key={it.label}
              className="flex min-w-[140px] items-center gap-3 rounded-xl bg-surface px-3 py-2.5 shadow-sm sm:min-w-0"
            >
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
                <Icon className="h-4 w-4" />
              </span>
              <div className="min-w-0">
                <p className="truncate text-xs text-text-secondary">{it.label}</p>
                <p className="truncate text-base font-semibold text-text">{it.value}</p>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Low stock panel
// ---------------------------------------------------------------------------

export function LowStockPanel({
  items,
  total,
  loading,
}: {
  items: LowStockEntry[];
  total: number;
  loading: boolean;
}) {
  return (
    <motion.div
      variants={sectionVariants}
      initial="hidden"
      animate="show"
      className="rounded-2xl border border-border bg-surface"
    >
      <div className="flex items-center justify-between border-b border-border px-6 py-4">
        <h2 className="text-lg font-semibold text-text">Low Stock</h2>
        {items.length > 0 && (
          <Link
            href="/admin/inventory?filter=low-stock"
            className="text-sm font-medium text-primary hover:text-primary-dark"
          >
            View all{total > items.length ? ` (${total})` : ''}
          </Link>
        )}
      </div>
      <div className="p-4">
        {loading ? (
          <div className="space-y-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="skeleton h-12 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 py-8 text-center">
            <CheckCircle2 className="h-8 w-8 text-emerald-500" />
            <p className="text-sm text-text-secondary">All products in stock</p>
          </div>
        ) : (
          <ul className="divide-y divide-border">
            {items.map((it) => (
              <li key={it.id} className="flex items-center gap-3 py-2.5">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-rose-100 text-rose-600 dark:bg-rose-900/40 dark:text-rose-400">
                  <Package className="h-4 w-4" />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-text">{it.name}</p>
                  <p className="text-xs text-text-secondary">Reorder at {it.reorder}</p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-rose-600 dark:text-rose-400">{it.current}</p>
                  <Link
                    href="/admin/inventory?filter=low-stock"
                    className="text-xs text-primary hover:text-primary-dark"
                  >
                    Restock
                  </Link>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------------------------
// Top customers panel
// ---------------------------------------------------------------------------

export function TopCustomersPanel({
  items,
  loading,
}: {
  items: TopCustomer[];
  loading: boolean;
}) {
  return (
    <motion.div
      variants={sectionVariants}
      initial="hidden"
      animate="show"
      transition={{ delay: 0.05 }}
      className="rounded-2xl border border-border bg-surface"
    >
      <div className="flex items-center justify-between border-b border-border px-6 py-4">
        <h2 className="text-lg font-semibold text-text">Top Customers</h2>
        <Link href="/admin/customers" className="text-sm font-medium text-primary hover:text-primary-dark">
          View all
        </Link>
      </div>
      <div className="p-4">
        {loading ? (
          <div className="space-y-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="skeleton h-12 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState icon={Users} text="No customer activity in this period." />
        ) : (
          <ul className="divide-y divide-border">
            {items.map((c, i) => {
              const isGuest = c.id === 'guest';
              const inner = (
                <div className="flex items-center gap-3 py-2.5">
                  <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-semibold text-primary">
                    {isGuest
                      ? '?'
                      : initials(
                          c.name.split(' ')[0],
                          c.name.split(' ').slice(-1)[0],
                          String(i + 1),
                        )}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-text">{c.name}</p>
                    <p className="text-xs text-text-secondary">
                      {c.orders} order{c.orders === 1 ? '' : 's'}
                    </p>
                  </div>
                  <span className="shrink-0 text-sm font-semibold text-text">{formatCurrency(c.spent)}</span>
                </div>
              );
              return (
                <li key={c.id}>
                  {isGuest ? (
                    inner
                  ) : (
                    <Link
                      href={`/admin/customers/${c.id}`}
                      className="block rounded-md px-1 hover:bg-surface-hover"
                    >
                      {inner}
                    </Link>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </motion.div>
  );
}

// ---------------------------------------------------------------------------
// Activity feed
// ---------------------------------------------------------------------------

function activityMeta(kind: ActivityItem['kind']): {
  icon: typeof DollarSign;
  bg: string;
  text: string;
} {
  switch (kind) {
    case 'order':
      return { icon: ShoppingCart, bg: 'bg-blue-100 dark:bg-blue-900/40', text: 'text-blue-600 dark:text-blue-400' };
    case 'customer':
      return { icon: UserPlus, bg: 'bg-violet-100 dark:bg-violet-900/40', text: 'text-violet-600 dark:text-violet-400' };
    case 'stock':
      return { icon: XCircle, bg: 'bg-rose-100 dark:bg-rose-900/40', text: 'text-rose-600 dark:text-rose-400' };
    case 'review':
      return { icon: Star, bg: 'bg-amber-100 dark:bg-amber-900/40', text: 'text-amber-600 dark:text-amber-400' };
  }
}

export function ActivityFeed({ items, loading }: { items: ActivityItem[]; loading: boolean }) {
  return (
    <motion.div
      variants={sectionVariants}
      initial="hidden"
      animate="show"
      className="rounded-2xl border border-border bg-surface"
    >
      <div className="flex items-center justify-between border-b border-border px-6 py-4">
        <h2 className="text-lg font-semibold text-text">Recent Activity</h2>
        <Bell className="h-4 w-4 text-text-muted" />
      </div>
      <div className="p-4">
        {loading ? (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="skeleton h-12 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState icon={Bell} text="No recent activity." />
        ) : (
          <motion.ul variants={containerVariants} initial="hidden" animate="show" className="space-y-1">
            {items.map((a) => {
              const meta = activityMeta(a.kind);
              const Icon = meta.icon;
              const row = (
                <div className="flex items-start gap-3 rounded-lg px-3 py-2 transition-colors hover:bg-surface-hover">
                  <span
                    className={cn('mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full', meta.bg)}
                  >
                    <Icon className={cn('h-3.5 w-3.5', meta.text)} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm text-text">{a.message}</p>
                    <p className="text-xs text-text-muted">{relativeTime(a.time)}</p>
                  </div>
                </div>
              );
              return (
                <motion.li key={a.id} variants={feedItemVariants}>
                  {a.href ? (
                    <Link href={a.href} className="block">
                      {row}
                    </Link>
                  ) : (
                    row
                  )}
                </motion.li>
              );
            })}
          </motion.ul>
        )}
      </div>
    </motion.div>
  );
}
