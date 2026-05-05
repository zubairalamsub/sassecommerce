'use client';

import { useState, useEffect } from 'react';
import {
  DollarSign,
  ShoppingCart,
  Users,
  TrendingUp,
  ArrowUpRight,
  ArrowDownRight,
  Package,
  Loader2,
} from 'lucide-react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import {
  AreaChart,
  Area,
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { analyticsApi, orderApi, productApi, userApi } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

// ---------------------------------------------------------------------------
// Animations
// ---------------------------------------------------------------------------

const containerVariants = {
  hidden: { opacity: 0 },
  show: {
    opacity: 1,
    transition: { staggerChildren: 0.1 },
  },
};

const cardVariants = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: [0, 0, 0.2, 1] as const } },
};

const sectionVariants = {
  hidden: { opacity: 0, y: 24 },
  show: { opacity: 1, y: 0, transition: { duration: 0.5, ease: [0, 0, 0.2, 1] as const } },
};

// ---------------------------------------------------------------------------
// Custom Tooltip
// ---------------------------------------------------------------------------

function ChartTooltip({
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

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface StatCard {
  title: string;
  value: string;
  change: string;
  trend: 'up' | 'down';
  icon: typeof DollarSign;
  color: string;
  bgColor: string;
  sparkline: { v: number }[];
}

interface RecentOrder {
  id: string;
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
// Page
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  const { tenantId, token } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [totalRevenue, setTotalRevenue] = useState(0);
  const [totalOrders, setTotalOrders] = useState(0);
  const [totalCustomers, setTotalCustomers] = useState(0);
  const [avgOrderValue, setAvgOrderValue] = useState(0);
  const [revenueData, setRevenueData] = useState<{ day: string; revenue: number }[]>([]);
  const [recentOrders, setRecentOrders] = useState<RecentOrder[]>([]);
  const [topProducts, setTopProducts] = useState<TopProduct[]>([]);
  const [orderStatusData, setOrderStatusData] = useState<OrderStatusEntry[]>([]);

  useEffect(() => {
    if (!tenantId) {
      setLoading(false);
      return;
    }

    const endDate = new Date().toISOString().split('T')[0];
    const startDate = new Date(Date.now() - 30 * 86400000).toISOString().split('T')[0];

    let completed = 0;
    const total = 4;
    const done = () => { completed++; if (completed >= total) setLoading(false); };

    // Fetch sales analytics
    analyticsApi.sales(tenantId, startDate, endDate, 'daily', token || undefined).then((res) => {
      setTotalRevenue(res.total_revenue ?? 0);
      setTotalOrders(res.total_orders ?? 0);
      setAvgOrderValue(res.avg_order_value ?? 0);
      if (res.data_points && res.data_points.length > 0) {
        setRevenueData(res.data_points.map((dp) => ({
          day: new Date(dp.date).toLocaleDateString('en-BD', { weekday: 'short' }),
          revenue: dp.revenue,
        })));
      }
    }).catch(() => {}).finally(done);

    // Fetch recent orders, then fetch details for each to get real totals
    orderApi.listByTenant(tenantId, token || undefined, 1, 10).then(async (res) => {
      if (res.total > 0) {
        setTotalOrders((prev) => prev || res.total);
      }
      if (res.data && res.data.length > 0) {
        // Fetch individual order details to get real total_amount (list endpoint returns 0)
        const detailed = await Promise.all(
          res.data.map((o) =>
            orderApi.get(o.id, tenantId, token || undefined).catch(() => o)
          )
        );

        setRecentOrders(detailed.slice(0, 5).map((o) => ({
          id: o.order_number,
          customer: o.customer_id,
          total: o.total,
          status: o.status,
          date: o.created_at?.split('T')[0] || '',
        })));

        // Compute revenue & avg from real order totals as fallback
        const orderTotals = detailed.map((o) => o.total).filter((t) => t > 0);
        if (orderTotals.length > 0) {
          const revenue = orderTotals.reduce((sum, t) => sum + t, 0);
          setTotalRevenue((prev) => prev || revenue);
          setAvgOrderValue((prev) => prev || Math.round(revenue / orderTotals.length));
        }

        const statusCounts: Record<string, number> = {};
        detailed.forEach((o) => { statusCounts[o.status] = (statusCounts[o.status] || 0) + 1; });
        const colors: Record<string, string> = { pending: '#f59e0b', confirmed: '#3b82f6', shipped: '#8b5cf6', delivered: '#10b981', cancelled: '#F42A41' };
        setOrderStatusData(Object.entries(statusCounts).map(([name, value]) => ({ name, value, color: colors[name] || '#9ca3af' })));
      }
    }).catch(() => {}).finally(done);

    // Fetch top products
    analyticsApi.products(tenantId, startDate, endDate, token || undefined).then((res) => {
      if (res.top_products && res.top_products.length > 0) {
        setTopProducts(res.top_products.map((p) => ({
          name: p.name,
          sold: p.units_sold,
          revenue: p.revenue,
        })));
      }
    }).catch(() => {}).finally(done);

    // Fetch total customers — response has pagination.total_items
    userApi.list(tenantId, token || '', 1, 1).then((res) => {
      const raw = res as any;
      const count = res.total ?? raw.pagination?.total_items ?? 0;
      setTotalCustomers(count);
    }).catch(() => {}).finally(done);
  }, [tenantId, token]);

  const statCards: StatCard[] = [
    { title: 'Total Revenue', value: formatCurrency(totalRevenue), change: '', trend: 'up', icon: DollarSign, color: '#006A4E', bgColor: 'bg-emerald-100 dark:bg-emerald-900/40', sparkline: [] },
    { title: 'Total Orders', value: totalOrders.toLocaleString(), change: '', trend: 'up', icon: ShoppingCart, color: '#3b82f6', bgColor: 'bg-blue-100 dark:bg-blue-900/40', sparkline: [] },
    { title: 'Total Customers', value: totalCustomers.toLocaleString(), change: '', trend: 'up', icon: Users, color: '#8b5cf6', bgColor: 'bg-violet-100 dark:bg-violet-900/40', sparkline: [] },
    { title: 'Avg Order Value', value: formatCurrency(avgOrderValue), change: '', trend: 'up', icon: TrendingUp, color: '#f59e0b', bgColor: 'bg-amber-100 dark:bg-amber-900/40', sparkline: [] },
  ];

  const maxSold = Math.max(...topProducts.map((p) => p.sold), 1);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-32">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <h1 className="text-2xl font-bold text-text">Dashboard</h1>
        <p className="mt-1 text-sm text-text-secondary">
          Welcome back! Here&apos;s what&apos;s happening with your store today.
        </p>
      </motion.div>

      {/* Stat Cards */}
      {statCards.length > 0 && (
        <motion.div
          className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4"
          variants={containerVariants}
          initial="hidden"
          animate="show"
        >
          {statCards.map((stat) => {
            const Icon = stat.icon;
            return (
              <motion.div
                key={stat.title}
                variants={cardVariants}
                className="rounded-2xl border border-border bg-surface p-6"
              >
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-text-secondary">
                    {stat.title}
                  </span>
                  <span className={cn('rounded-xl p-2.5', stat.bgColor)}>
                    <Icon className="h-5 w-5" style={{ color: stat.color }} />
                  </span>
                </div>
                <div className="mt-3">
                  <span className="text-2xl font-bold text-text">{stat.value}</span>
                </div>
              </motion.div>
            );
          })}
        </motion.div>
      )}

      {/* Charts Row */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Revenue Overview */}
        <motion.div
          className="rounded-2xl border border-border bg-surface p-6 lg:col-span-2"
          variants={sectionVariants}
          initial="hidden"
          animate="show"
          transition={{ delay: 0.3 }}
        >
          <h2 className="mb-4 text-lg font-semibold text-text">Revenue Overview</h2>
          {revenueData.length > 0 ? (
            <div className="h-[260px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={revenueData}>
                  <defs>
                    <linearGradient id="revenueGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#006A4E" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#006A4E" stopOpacity={0} />
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
                    stroke="#006A4E"
                    strokeWidth={2.5}
                    fill="url(#revenueGrad)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="flex h-[260px] items-center justify-center text-sm text-text-muted">
              No revenue data yet. Revenue will appear here once you have orders.
            </div>
          )}
        </motion.div>

        {/* Order Status Breakdown */}
        <motion.div
          className="rounded-2xl border border-border bg-surface p-6"
          variants={sectionVariants}
          initial="hidden"
          animate="show"
          transition={{ delay: 0.4 }}
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
            <div className="flex h-[200px] items-center justify-center text-sm text-text-muted">
              No orders yet.
            </div>
          )}
        </motion.div>
      </div>

      {/* Bottom Row */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Recent Orders */}
        <motion.div
          className="rounded-2xl border border-border bg-surface lg:col-span-2"
          variants={sectionVariants}
          initial="hidden"
          animate="show"
          transition={{ delay: 0.5 }}
        >
          <div className="flex items-center justify-between border-b border-border px-6 py-4">
            <h2 className="text-lg font-semibold text-text">Recent Orders</h2>
            <Link
              href="/admin/orders"
              className="text-sm font-medium text-primary hover:text-primary-dark"
            >
              View all
            </Link>
          </div>
          {recentOrders.length > 0 ? (
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
                        <Link href={`/admin/orders/${order.id}`}>{order.id}</Link>
                      </td>
                      <td className="px-6 py-4 text-sm text-text">{order.customer}</td>
                      <td className="px-6 py-4 text-sm text-text">
                        {formatCurrency(order.total)}
                      </td>
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
                        {formatDate(order.date)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <ShoppingCart className="h-8 w-8 text-text-muted" />
              <p className="mt-2 text-sm text-text-muted">No orders yet.</p>
            </div>
          )}
        </motion.div>

        {/* Top Products */}
        <motion.div
          className="rounded-2xl border border-border bg-surface p-6"
          variants={sectionVariants}
          initial="hidden"
          animate="show"
          transition={{ delay: 0.6 }}
        >
          <h2 className="mb-5 text-lg font-semibold text-text">Top Products</h2>
          {topProducts.length > 0 ? (
            <ul className="space-y-4">
              {topProducts.map((product, i) => (
                <li key={product.name}>
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-3">
                      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-xs font-bold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400">
                        {i + 1}
                      </span>
                      <span className="text-sm font-medium text-text leading-tight">
                        {product.name}
                      </span>
                    </div>
                    <span className="shrink-0 text-xs text-text-muted">
                      {product.sold} sold
                    </span>
                  </div>
                  <div className="mt-2 ml-11">
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                      <motion.div
                        className="h-full rounded-full"
                        style={{ backgroundColor: '#006A4E' }}
                        initial={{ width: 0 }}
                        animate={{
                          width: `${(product.sold / maxSold) * 100}%`,
                        }}
                        transition={{ duration: 0.8, delay: 0.7 + i * 0.1 }}
                      />
                    </div>
                    <p className="mt-1 text-xs text-text-secondary">
                      {formatCurrency(product.revenue)}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          ) : (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <Package className="h-8 w-8 text-text-muted" />
              <p className="mt-2 text-sm text-text-muted">No product data yet.</p>
            </div>
          )}
        </motion.div>
      </div>
    </div>
  );
}
