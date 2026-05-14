'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  Clock,
  DollarSign,
  Package,
  Plus,
  Receipt,
  RefreshCw,
  ShoppingCart,
  TrendingUp,
  Users,
  Zap,
} from 'lucide-react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import {
  Area,
  AreaChart,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import {
  analyticsApi,
  inventoryApi,
  orderApi,
  productApi,
  userApi,
  type InventoryItem,
  type Order,
  type Product,
  type User,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/stores/toast';
import {
  ActivityFeed,
  ChangeBadge,
  ChartSkeleton,
  ChartTooltip,
  CustomRangePopover,
  DateRangeTabs,
  EmptyState,
  LowStockPanel,
  Sparkline,
  StatCardSkeleton,
  TodaySnapshot,
  TopCustomersPanel,
  cardVariants,
  containerVariants,
  sectionVariants,
  type ActivityItem,
  type LowStockEntry,
  type RangeKey,
  type TopCustomer,
} from './_components';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface RangeState {
  key: RangeKey;
  start: Date;
  end: Date;
}

interface StatCard {
  title: string;
  value: string;
  change: number | null;
  icon: typeof DollarSign;
  accent: string;
  bgAccent: string;
  sparkline: number[];
  href?: string;
}

interface RecentOrder {
  id: string;
  orderId: string;
  customer: string;
  total: number;
  status: string;
  date: string;
}

interface TopProduct {
  name: string;
  sold: number;
  revenue: number;
}

interface OrderStatusEntry {
  name: string;
  value: number;
  color: string;
}

// ---------------------------------------------------------------------------
// Range / numeric helpers
// ---------------------------------------------------------------------------

const DAY_MS = 86400000;

function startOfDay(d: Date): Date {
  const x = new Date(d);
  x.setHours(0, 0, 0, 0);
  return x;
}

function endOfDay(d: Date): Date {
  const x = new Date(d);
  x.setHours(23, 59, 59, 999);
  return x;
}

function toIsoDate(d: Date): string {
  return d.toISOString().split('T')[0];
}

function rangeFor(key: Exclude<RangeKey, 'custom'>): { start: Date; end: Date } {
  const end = endOfDay(new Date());
  const start = startOfDay(new Date());
  switch (key) {
    case 'today':
      return { start, end };
    case '7d':
      return { start: startOfDay(new Date(Date.now() - 6 * DAY_MS)), end };
    case '30d':
      return { start: startOfDay(new Date(Date.now() - 29 * DAY_MS)), end };
    case '90d':
      return { start: startOfDay(new Date(Date.now() - 89 * DAY_MS)), end };
    case 'year':
      return { start: startOfDay(new Date(Date.now() - 364 * DAY_MS)), end };
  }
}

function rangeLabel(r: RangeState): string {
  const fmt = (d: Date) => d.toLocaleDateString('en-BD', { month: 'short', day: 'numeric' });
  if (r.key === 'today') return 'Today';
  return `${fmt(r.start)} – ${fmt(r.end)}`;
}

function rangeTitle(r: RangeState): string {
  switch (r.key) {
    case 'today': return "Today's overview";
    case '7d': return '7 day overview';
    case '30d': return '30 day overview';
    case '90d': return '90 day overview';
    case 'year': return '1 year overview';
    case 'custom': return 'Custom range';
  }
}

function pctChange(current: number, prev: number): number | null {
  if (!Number.isFinite(prev) || prev === 0) return current > 0 ? 100 : null;
  return ((current - prev) / prev) * 100;
}

function bucketSparkline(values: number[], buckets = 14): number[] {
  if (values.length === 0) return [];
  if (values.length <= buckets) return values;
  const size = Math.ceil(values.length / buckets);
  const out: number[] = [];
  for (let i = 0; i < values.length; i += size) {
    out.push(values.slice(i, i + size).reduce((a, b) => a + b, 0));
  }
  return out;
}

function mergeActivity(existing: ActivityItem[], next: ActivityItem[]): ActivityItem[] {
  const map = new Map<string, ActivityItem>();
  [...existing, ...next].forEach((a) => map.set(a.id, a));
  return Array.from(map.values()).sort((a, b) => b.time - a.time).slice(0, 10);
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  const { tenantId, token } = useAuthStore();

  const [range, setRange] = useState<RangeState>(() => {
    const r = rangeFor('30d');
    return { key: '30d', ...r };
  });
  const [customStart, setCustomStart] = useState<string>(() => toIsoDate(range.start));
  const [customEnd, setCustomEnd] = useState<string>(() => toIsoDate(range.end));
  const [customOpen, setCustomOpen] = useState(false);

  // Loading is `true` until at least one fetch cycle completes for a tenant.
  // No tenant → nothing to fetch, so treat that as "loaded".
  const [loading, setLoading] = useState<boolean>(() => Boolean(tenantId));
  const [refreshing, setRefreshing] = useState(false);
  const [refreshTick, setRefreshTick] = useState(0);

  // Period metrics
  const [totalRevenue, setTotalRevenue] = useState(0);
  const [totalOrders, setTotalOrders] = useState(0);
  const [totalCustomers, setTotalCustomers] = useState(0);
  const [totalProducts, setTotalProducts] = useState(0);
  const [avgOrderValue, setAvgOrderValue] = useState(0);
  const [pendingOrders, setPendingOrders] = useState(0);

  // Previous-period comparison
  const [prevRevenue, setPrevRevenue] = useState(0);
  const [prevOrders, setPrevOrders] = useState(0);
  const [prevAov, setPrevAov] = useState(0);

  // Series
  const [revenueData, setRevenueData] = useState<{ day: string; revenue: number }[]>([]);
  const [revenueSeries, setRevenueSeries] = useState<number[]>([]);
  const [ordersSeries, setOrdersSeries] = useState<number[]>([]);

  // Lists
  const [recentOrders, setRecentOrders] = useState<RecentOrder[]>([]);
  const [topProducts, setTopProducts] = useState<TopProduct[]>([]);
  const [orderStatusData, setOrderStatusData] = useState<OrderStatusEntry[]>([]);
  const [topCustomers, setTopCustomers] = useState<TopCustomer[]>([]);
  const [lowStock, setLowStock] = useState<LowStockEntry[]>([]);
  const [lowStockTotal, setLowStockTotal] = useState(0);
  const [activity, setActivity] = useState<ActivityItem[]>([]);

  // Today snapshot
  const [todayRevenue, setTodayRevenue] = useState(0);
  const [todayOrders, setTodayOrders] = useState(0);
  const [todayNewCustomers, setTodayNewCustomers] = useState(0);
  const [lastOrderAt, setLastOrderAt] = useState<number | null>(null);

  const handleRangeChange = (key: RangeKey) => {
    if (key === 'custom') {
      setCustomOpen(true);
      return;
    }
    const r = rangeFor(key);
    setRange({ key, ...r });
  };

  const applyCustomRange = () => {
    if (!customStart || !customEnd) return;
    const s = startOfDay(new Date(customStart));
    const e = endOfDay(new Date(customEnd));
    if (e < s) {
      toast.error('End date must be after start date');
      return;
    }
    setRange({ key: 'custom', start: s, end: e });
    setCustomOpen(false);
  };

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    setRefreshTick((t) => t + 1);
  }, []);

  // -------------------------------------------------------------------------
  // Data fetch
  // -------------------------------------------------------------------------

  useEffect(() => {
    if (!tenantId) return;

    let cancelled = false;
    const startStr = toIsoDate(range.start);
    const endStr = toIsoDate(range.end);

    // Previous comparison period of equal length, ending the day before `start`.
    const spanMs = Math.max(DAY_MS, range.end.getTime() - range.start.getTime());
    const prevEnd = new Date(range.start.getTime() - DAY_MS);
    const prevStart = new Date(prevEnd.getTime() - spanMs);
    const prevStartStr = toIsoDate(startOfDay(prevStart));
    const prevEndStr = toIsoDate(endOfDay(prevEnd));

    const todayStart = startOfDay(new Date());
    const todayEnd = endOfDay(new Date());

    const tasks: Promise<unknown>[] = [];

    // Current sales report
    tasks.push(
      analyticsApi
        .sales(tenantId, startStr, endStr, 'daily', token || undefined)
        .then((res) => {
          if (cancelled) return;
          setTotalRevenue(res.total_revenue ?? 0);
          setTotalOrders(res.total_orders ?? 0);
          setAvgOrderValue(res.avg_order_value ?? 0);
          const pts = res.data_points ?? [];
          if (pts.length > 0) {
            setRevenueData(
              pts.map((dp) => ({
                day: new Date(dp.date).toLocaleDateString('en-BD', { weekday: 'short' }),
                revenue: dp.revenue,
              })),
            );
            setRevenueSeries(pts.map((dp) => dp.revenue));
            setOrdersSeries(pts.map((dp) => dp.orders));
          } else {
            setRevenueData([]);
            setRevenueSeries([]);
            setOrdersSeries([]);
          }
        })
        .catch(() => {
          if (cancelled) return;
          setRevenueData([]);
          setRevenueSeries([]);
          setOrdersSeries([]);
        }),
    );

    // Previous-period sales for comparison
    tasks.push(
      analyticsApi
        .sales(tenantId, prevStartStr, prevEndStr, 'daily', token || undefined)
        .then((res) => {
          if (cancelled) return;
          setPrevRevenue(res.total_revenue ?? 0);
          setPrevOrders(res.total_orders ?? 0);
          setPrevAov(res.avg_order_value ?? 0);
        })
        .catch(() => {
          if (cancelled) return;
          setPrevRevenue(0);
          setPrevOrders(0);
          setPrevAov(0);
        }),
    );

    // Top products in range
    tasks.push(
      analyticsApi
        .products(tenantId, startStr, endStr, token || undefined)
        .then((res) => {
          if (cancelled) return;
          setTopProducts(
            (res.top_products || []).slice(0, 5).map((p) => ({
              name: p.name,
              sold: p.units_sold,
              revenue: p.revenue,
            })),
          );
        })
        .catch(() => {
          if (cancelled) return;
          setTopProducts([]);
        }),
    );

    // Orders — wide page powers recent, status pie, top customers, activity.
    tasks.push(
      orderApi
        .listByTenant(tenantId, token || undefined, 1, 100)
        .then(async (res) => {
          if (cancelled) return;
          const orders: Order[] = res.data || [];

          // Hydrate detail for the most recent few to get accurate totals.
          const detailLimit = Math.min(orders.length, 20);
          const detailed = await Promise.all(
            orders.slice(0, detailLimit).map((o) =>
              orderApi.get(o.id, tenantId, token || undefined).catch(() => o),
            ),
          );
          if (cancelled) return;
          const all: Order[] = [...detailed, ...orders.slice(detailLimit)];

          setRecentOrders(
            all.slice(0, 5).map((o) => ({
              id: o.id,
              orderId: o.order_number || o.id.slice(0, 8).toUpperCase(),
              customer: o.customer_id || 'Guest',
              total: o.total,
              status: o.status,
              date: o.created_at?.split('T')[0] || '',
            })),
          );

          setPendingOrders(
            all.filter((o) => o.status === 'pending' || o.status === 'confirmed').length,
          );

          const statusCounts: Record<string, number> = {};
          all.forEach((o) => {
            statusCounts[o.status] = (statusCounts[o.status] || 0) + 1;
          });
          const colors: Record<string, string> = {
            pending: '#f59e0b',
            confirmed: '#3b82f6',
            shipped: '#8b5cf6',
            delivered: '#10b981',
            cancelled: '#F42A41',
          };
          setOrderStatusData(
            Object.entries(statusCounts).map(([name, value]) => ({
              name,
              value,
              color: colors[name] || '#9ca3af',
            })),
          );

          // Fallback revenue/AOV if analytics returned zero.
          const totals = all.map((o) => o.total).filter((t) => t > 0);
          if (totals.length > 0) {
            const rev = totals.reduce((s, t) => s + t, 0);
            setTotalRevenue((prev) => prev || rev);
            setTotalOrders((prev) => prev || all.length);
            setAvgOrderValue((prev) => prev || Math.round(rev / totals.length));
          }

          // Top customers by spend in range
          const inRange = all.filter((o) => {
            if (!o.created_at) return false;
            const t = new Date(o.created_at).getTime();
            return t >= range.start.getTime() && t <= range.end.getTime();
          });
          const byCustomer: Record<string, { orders: number; spent: number }> = {};
          inRange.forEach((o) => {
            const cid = o.customer_id || 'guest';
            if (!byCustomer[cid]) byCustomer[cid] = { orders: 0, spent: 0 };
            byCustomer[cid].orders += 1;
            byCustomer[cid].spent += o.total || 0;
          });
          const topCust: TopCustomer[] = Object.entries(byCustomer)
            .map(([id, v]) => ({ id, name: id === 'guest' ? 'Guest' : id, ...v }))
            .sort((a, b) => b.spent - a.spent)
            .slice(0, 5);
          setTopCustomers(topCust);

          // Today snapshot
          const todays = all.filter((o) => {
            if (!o.created_at) return false;
            const t = new Date(o.created_at).getTime();
            return t >= todayStart.getTime() && t <= todayEnd.getTime();
          });
          setTodayRevenue(todays.reduce((s, o) => s + (o.total || 0), 0));
          setTodayOrders(todays.length);
          const newestTs = all
            .map((o) => (o.created_at ? new Date(o.created_at).getTime() : 0))
            .reduce((a, b) => Math.max(a, b), 0);
          setLastOrderAt(newestTs > 0 ? newestTs : null);

          // Activity feed seed
          const orderActivity: ActivityItem[] = all.slice(0, 10).map((o) => ({
            id: `order-${o.id}`,
            kind: 'order',
            message: `New order ${o.order_number || o.id.slice(0, 8).toUpperCase()} · ${formatCurrency(o.total)}`,
            time: o.created_at ? new Date(o.created_at).getTime() : Date.now(),
            href: `/admin/orders/${o.id}`,
          }));
          setActivity((prev) => mergeActivity(prev, orderActivity));
        })
        .catch(() => {
          if (cancelled) return;
          setRecentOrders([]);
          setOrderStatusData([]);
          setTopCustomers([]);
        }),
    );

    // Customers
    tasks.push(
      userApi
        .list(tenantId, token || '', 1, 50)
        .then((res) => {
          if (cancelled) return;
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const raw = res as any;
          const count = res.total ?? raw.pagination?.total_items ?? res.data?.length ?? 0;
          setTotalCustomers(count);
          const users: User[] = res.data || [];

          const newToday = users.filter((u) => {
            if (!u.created_at) return false;
            const t = new Date(u.created_at).getTime();
            return t >= todayStart.getTime() && t <= todayEnd.getTime();
          });
          setTodayNewCustomers(newToday.length);

          // Enrich top customers with names
          setTopCustomers((cur) => {
            if (cur.length === 0) return cur;
            return cur.map((c) => {
              const u = users.find((x) => x.id === c.id);
              if (!u) return c;
              const full = `${u.first_name || ''} ${u.last_name || ''}`.trim() || u.username || u.email;
              return { ...c, name: full };
            });
          });

          const userActivity: ActivityItem[] = users.slice(0, 5).map((u) => ({
            id: `user-${u.id}`,
            kind: 'customer',
            message: `${u.first_name || u.username || u.email} registered`,
            time: u.created_at ? new Date(u.created_at).getTime() : Date.now(),
            href: `/admin/customers/${u.id}`,
          }));
          setActivity((prev) => mergeActivity(prev, userActivity));
        })
        .catch(() => {}),
    );

    // Products — count + OOS activity signals
    tasks.push(
      productApi
        .list(tenantId, 1, 50)
        .then((res) => {
          if (cancelled) return;
          const total = res.pagination?.total_items ?? res.total ?? res.data?.length ?? 0;
          setTotalProducts(total);

          const products: Product[] = res.data || [];
          const oos = products
            .map<ActivityItem | null>((p) => {
              const noStock =
                Array.isArray(p.variants) &&
                p.variants.length > 0 &&
                p.variants.every((v) => (v.stock ?? 0) <= 0);
              if (!noStock) return null;
              return {
                id: `oos-${p.id}`,
                kind: 'stock',
                message: `${p.name} is out of stock`,
                time: p.updated_at ? new Date(p.updated_at).getTime() : Date.now(),
                href: `/admin/products/${p.id}`,
              };
            })
            .filter((x): x is ActivityItem => x !== null)
            .slice(0, 5);
          if (oos.length > 0) {
            setActivity((prev) => mergeActivity(prev, oos));
          }
        })
        .catch(() => {}),
    );

    // Low-stock inventory
    tasks.push(
      inventoryApi
        .lowStock(tenantId, token || '')
        .then((items: InventoryItem[]) => {
          if (cancelled) return;
          const list: LowStockEntry[] = (items || [])
            .slice()
            .sort((a, b) => a.quantityAvailable - b.quantityAvailable)
            .slice(0, 5)
            .map((i) => ({
              id: i.id,
              productId: i.productId,
              name: i.sku || i.productId,
              current: i.quantityAvailable,
              reorder: i.reorderPoint,
            }));
          setLowStock(list);
          setLowStockTotal(items?.length || 0);
        })
        .catch(() => {
          if (cancelled) return;
          setLowStock([]);
          setLowStockTotal(0);
        }),
    );

    Promise.allSettled(tasks).then(() => {
      if (cancelled) return;
      setLoading(false);
      if (refreshing) {
        setRefreshing(false);
        toast.success('Dashboard refreshed');
      }
    });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, token, range.key, range.start.getTime(), range.end.getTime(), refreshTick]);

  // -------------------------------------------------------------------------
  // Derived
  // -------------------------------------------------------------------------

  const revenueChange = useMemo(() => pctChange(totalRevenue, prevRevenue), [totalRevenue, prevRevenue]);
  const ordersChange = useMemo(() => pctChange(totalOrders, prevOrders), [totalOrders, prevOrders]);
  const aovChange = useMemo(() => pctChange(avgOrderValue, prevAov), [avgOrderValue, prevAov]);

  const revenueSpark = useMemo(() => bucketSparkline(revenueSeries), [revenueSeries]);
  const ordersSpark = useMemo(() => bucketSparkline(ordersSeries), [ordersSeries]);

  const statCards: StatCard[] = [
    { title: 'Total Revenue', value: formatCurrency(totalRevenue), change: revenueChange, icon: DollarSign, accent: 'var(--color-primary)', bgAccent: 'bg-emerald-100 dark:bg-emerald-900/40', sparkline: revenueSpark },
    { title: 'Total Orders', value: totalOrders.toLocaleString(), change: ordersChange, icon: ShoppingCart, accent: '#3b82f6', bgAccent: 'bg-blue-100 dark:bg-blue-900/40', sparkline: ordersSpark, href: '/admin/orders' },
    { title: 'Total Customers', value: totalCustomers.toLocaleString(), change: null, icon: Users, accent: '#8b5cf6', bgAccent: 'bg-violet-100 dark:bg-violet-900/40', sparkline: [], href: '/admin/customers' },
    { title: 'Total Products', value: totalProducts.toLocaleString(), change: null, icon: Package, accent: '#0ea5e9', bgAccent: 'bg-sky-100 dark:bg-sky-900/40', sparkline: [], href: '/admin/products' },
    { title: 'Avg Order Value', value: formatCurrency(avgOrderValue), change: aovChange, icon: TrendingUp, accent: '#f59e0b', bgAccent: 'bg-amber-100 dark:bg-amber-900/40', sparkline: [] },
    { title: 'Conversion Rate', value: '—', change: null, icon: TrendingUp, accent: '#14b8a6', bgAccent: 'bg-teal-100 dark:bg-teal-900/40', sparkline: [] },
    { title: 'Pending Orders', value: pendingOrders.toLocaleString(), change: null, icon: Clock, accent: '#f97316', bgAccent: 'bg-orange-100 dark:bg-orange-900/40', sparkline: [], href: '/admin/orders?status=pending' },
  ];

  const maxSold = Math.max(...topProducts.map((p) => p.sold), 1);

  const quickActions = [
    { label: 'New product', href: '/admin/products/new', icon: Plus, accent: 'var(--color-primary)' },
    { label: 'Instant sell', href: '/admin/sales/new', icon: Zap, accent: '#f59e0b' },
    { label: 'View orders', href: '/admin/orders', icon: Receipt, accent: '#3b82f6' },
    { label: 'Low stock', href: '/admin/inventory?filter=low-stock', icon: AlertTriangle, accent: '#F42A41' },
  ];

  // -------------------------------------------------------------------------
  // Render
  // -------------------------------------------------------------------------

  return (
    <div className="space-y-6">
      <motion.div
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
        className="flex flex-wrap items-start justify-between gap-4"
      >
        <div>
          <h1 className="text-2xl font-bold text-text">Dashboard</h1>
          <p className="mt-1 text-sm text-text-secondary">
            {rangeTitle(range)} · {rangeLabel(range)}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <DateRangeTabs activeKey={range.key} onChange={handleRangeChange} />
          <button
            type="button"
            onClick={handleRefresh}
            disabled={refreshing || loading}
            className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-1.5 text-sm text-text-secondary transition-colors hover:bg-surface-hover disabled:opacity-50"
            aria-label="Refresh dashboard"
          >
            <RefreshCw className={cn('h-4 w-4', refreshing && 'animate-spin')} />
            <span className="hidden sm:inline">Refresh</span>
          </button>
        </div>
      </motion.div>

      {customOpen && (
        <CustomRangePopover
          start={customStart}
          end={customEnd}
          onStartChange={setCustomStart}
          onEndChange={setCustomEnd}
          onCancel={() => setCustomOpen(false)}
          onApply={applyCustomRange}
        />
      )}

      {/* Quick Actions */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {quickActions.map((qa) => {
          const Icon = qa.icon;
          return (
            <Link
              key={qa.label}
              href={qa.href}
              className="group flex items-center gap-3 rounded-xl border border-border bg-surface p-3 transition-all hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
            >
              <span
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-light transition-transform group-hover:scale-110"
                style={{ color: qa.accent }}
              >
                <Icon className="h-4 w-4" />
              </span>
              <span className="text-sm font-medium text-text">{qa.label}</span>
            </Link>
          );
        })}
      </div>

      {/* Today snapshot strip */}
      <TodaySnapshot
        revenue={todayRevenue}
        orders={todayOrders}
        customers={todayNewCustomers}
        lastOrderAt={lastOrderAt}
      />

      {/* Stat Cards */}
      {loading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <StatCardSkeleton key={i} />
          ))}
        </div>
      ) : (
        <motion.div
          className="grid grid-cols-2 gap-4 sm:grid-cols-2 lg:grid-cols-4"
          variants={containerVariants}
          initial="hidden"
          animate="show"
        >
          {statCards.map((stat) => {
            const Icon = stat.icon;
            const body = (
              <div className="flex h-full flex-col rounded-2xl border border-border bg-surface p-5 transition-colors hover:bg-surface-hover">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium uppercase tracking-wide text-text-secondary">
                    {stat.title}
                  </span>
                  <span className={cn('rounded-xl p-2', stat.bgAccent)}>
                    <Icon className="h-4 w-4" style={{ color: stat.accent }} />
                  </span>
                </div>
                <div className="mt-3">
                  <span className="text-2xl font-bold text-text">{stat.value}</span>
                </div>
                <div className="mt-3 flex items-center justify-between">
                  <ChangeBadge value={stat.change} />
                  <Sparkline values={stat.sparkline} color={stat.accent} />
                </div>
              </div>
            );
            return (
              <motion.div key={stat.title} variants={cardVariants}>
                {stat.href ? (
                  <Link href={stat.href} className="block h-full">{body}</Link>
                ) : (
                  body
                )}
              </motion.div>
            );
          })}
        </motion.div>
      )}

      {/* Charts Row */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {loading ? (
          <>
            <div className="lg:col-span-2"><ChartSkeleton /></div>
            <ChartSkeleton height={200} />
          </>
        ) : (
          <>
            <motion.div
              className="rounded-2xl border border-border bg-surface p-6 lg:col-span-2"
              variants={sectionVariants}
              initial="hidden"
              animate="show"
            >
              <h2 className="mb-4 text-lg font-semibold text-text">Revenue Overview</h2>
              {revenueData.length > 0 ? (
                <div className="h-[260px] w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={revenueData}>
                      <defs>
                        <linearGradient id="revenueGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.3} />
                          <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <XAxis
                        dataKey="day"
                        axisLine={false}
                        tickLine={false}
                        tick={{ fontSize: 12, fill: 'var(--color-text-muted, #9ca3af)' }}
                      />
                      <YAxis
                        axisLine={false}
                        tickLine={false}
                        tickFormatter={(v: number) => `${(v / 1000).toFixed(0)}k`}
                        tick={{ fontSize: 12, fill: 'var(--color-text-muted, #9ca3af)' }}
                        width={48}
                      />
                      <Tooltip content={<ChartTooltip isCurrency />} />
                      <Area
                        type="monotone"
                        dataKey="revenue"
                        stroke="var(--color-primary)"
                        strokeWidth={2.5}
                        fill="url(#revenueGrad)"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              ) : (
                <EmptyState icon={DollarSign} text="No revenue data for this period yet." />
              )}
            </motion.div>

            <motion.div
              className="rounded-2xl border border-border bg-surface p-6"
              variants={sectionVariants}
              initial="hidden"
              animate="show"
              transition={{ delay: 0.1 }}
            >
              <h2 className="mb-4 text-lg font-semibold text-text">Order Status</h2>
              {orderStatusData.length > 0 ? (
                <>
                  <div className="flex h-[200px] items-center justify-center">
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={orderStatusData}
                          cx="50%"
                          cy="50%"
                          innerRadius={55}
                          outerRadius={80}
                          paddingAngle={3}
                          dataKey="value"
                          stroke="none"
                        >
                          {orderStatusData.map((entry) => (
                            <Cell key={entry.name} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip content={<ChartTooltip prefix="Orders: " />} />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>
                  <div className="mt-2 flex flex-wrap justify-center gap-x-4 gap-y-1.5">
                    {orderStatusData.map((s) => (
                      <div key={s.name} className="flex items-center gap-1.5 text-xs text-text-secondary">
                        <span
                          className="inline-block h-2.5 w-2.5 rounded-full"
                          style={{ backgroundColor: s.color }}
                        />
                        {s.name} ({s.value})
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <EmptyState icon={ShoppingCart} text="No orders yet." />
              )}
            </motion.div>
          </>
        )}
      </div>

      {/* Low stock + Top customers */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <LowStockPanel items={lowStock} total={lowStockTotal} loading={loading} />
        <TopCustomersPanel items={topCustomers} loading={loading} />
      </div>

      {/* Recent orders + Top products */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <motion.div
          className="rounded-2xl border border-border bg-surface lg:col-span-2"
          variants={sectionVariants}
          initial="hidden"
          animate="show"
        >
          <div className="flex items-center justify-between border-b border-border px-6 py-4">
            <h2 className="text-lg font-semibold text-text">Recent Orders</h2>
            <Link href="/admin/orders" className="text-sm font-medium text-primary hover:text-primary-dark">
              View all
            </Link>
          </div>
          {loading ? (
            <div className="space-y-3 p-6">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="skeleton h-10 w-full" />
              ))}
            </div>
          ) : recentOrders.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border text-left text-sm text-text-secondary">
                    <th className="px-6 py-3 font-medium">Order #</th>
                    <th className="px-6 py-3 font-medium">Customer</th>
                    <th className="px-6 py-3 font-medium">Total</th>
                    <th className="px-6 py-3 font-medium">Status</th>
                    <th className="px-6 py-3 font-medium">Date</th>
                  </tr>
                </thead>
                <tbody>
                  {recentOrders.map((order) => (
                    <tr
                      key={order.id}
                      className="border-b border-border transition-colors last:border-b-0 hover:bg-surface-hover"
                    >
                      <td className="px-6 py-4 text-sm font-medium text-primary">
                        <Link href={`/admin/orders/${order.id}`}>{order.orderId}</Link>
                      </td>
                      <td className="px-6 py-4 text-sm text-text">{order.customer}</td>
                      <td className="px-6 py-4 text-sm text-text">{formatCurrency(order.total)}</td>
                      <td className="px-6 py-4">
                        <span
                          className={cn(
                            'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                            statusColor(order.status),
                          )}
                        >
                          {order.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-text-muted">
                        {order.date ? formatDate(order.date) : '—'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState icon={ShoppingCart} text="No orders yet." className="py-12" />
          )}
        </motion.div>

        <motion.div
          className="rounded-2xl border border-border bg-surface p-6"
          variants={sectionVariants}
          initial="hidden"
          animate="show"
          transition={{ delay: 0.1 }}
        >
          <h2 className="mb-5 text-lg font-semibold text-text">Top Products</h2>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="skeleton h-10 w-full" />
              ))}
            </div>
          ) : topProducts.length > 0 ? (
            <ul className="space-y-4">
              {topProducts.map((product, i) => (
                <li key={product.name}>
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-3">
                      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary-light text-xs font-bold text-primary">
                        {i + 1}
                      </span>
                      <span className="text-sm font-medium leading-tight text-text">{product.name}</span>
                    </div>
                    <span className="shrink-0 text-xs text-text-muted">{product.sold} sold</span>
                  </div>
                  <div className="mt-2 ml-11">
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-hover">
                      <motion.div
                        className="h-full rounded-full bg-primary"
                        initial={{ width: 0 }}
                        animate={{ width: `${(product.sold / maxSold) * 100}%` }}
                        transition={{ duration: 0.8, delay: 0.2 + i * 0.08 }}
                      />
                    </div>
                    <p className="mt-1 text-xs text-text-secondary">{formatCurrency(product.revenue)}</p>
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState icon={Package} text="No product data yet." />
          )}
        </motion.div>
      </div>

      <ActivityFeed items={activity} loading={loading} />
    </div>
  );
}
