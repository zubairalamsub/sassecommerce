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
import { analyticsApi, orderApi } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

// ---------------------------------------------------------------------------
// Demo data
// ---------------------------------------------------------------------------

const sparklineRevenue = [
  { v: 160000 },
  { v: 185000 },
  { v: 170000 },
  { v: 195000 },
  { v: 210000 },
  { v: 178000 },
  { v: 186500 },
];
const sparklineOrders = [
  { v: 42 },
  { v: 48 },
  { v: 51 },
  { v: 45 },
  { v: 55 },
  { v: 58 },
  { v: 57 },
];
const sparklineCustomers = [
  { v: 165 },
  { v: 172 },
  { v: 168 },
  { v: 180 },
  { v: 185 },
  { v: 178 },
  { v: 197 },
];
const sparklineAOV = [
  { v: 3800 },
  { v: 3700 },
  { v: 3650 },
  { v: 3720 },
  { v: 3580 },
  { v: 3620 },
  { v: 3609 },
];

const stats = [
  {
    title: 'Total Revenue',
    value: formatCurrency(1284500),
    change: '+12.5%',
    trend: 'up' as const,
    icon: DollarSign,
    color: '#006A4E',
    bgColor: 'bg-emerald-100 dark:bg-emerald-900/40',
    sparkline: sparklineRevenue,
  },
  {
    title: 'Total Orders',
    value: '356',
    change: '+8.2%',
    trend: 'up' as const,
    icon: ShoppingCart,
    color: '#3b82f6',
    bgColor: 'bg-blue-100 dark:bg-blue-900/40',
    sparkline: sparklineOrders,
  },
  {
    title: 'Total Customers',
    value: '1,245',
    change: '+3.1%',
    trend: 'up' as const,
    icon: Users,
    color: '#8b5cf6',
    bgColor: 'bg-violet-100 dark:bg-violet-900/40',
    sparkline: sparklineCustomers,
  },
  {
    title: 'Avg Order Value',
    value: formatCurrency(3609),
    change: '-2.4%',
    trend: 'down' as const,
    icon: TrendingUp,
    color: '#f59e0b',
    bgColor: 'bg-amber-100 dark:bg-amber-900/40',
    sparkline: sparklineAOV,
  },
];

const revenueData = [
  { day: 'Mon', revenue: 160000 },
  { day: 'Tue', revenue: 185000 },
  { day: 'Wed', revenue: 170000 },
  { day: 'Thu', revenue: 195000 },
  { day: 'Fri', revenue: 210000 },
  { day: 'Sat', revenue: 178000 },
  { day: 'Sun', revenue: 186500 },
];

const orderStatusData = [
  { name: 'Pending', value: 12, color: '#f59e0b' },
  { name: 'Confirmed', value: 8, color: '#3b82f6' },
  { name: 'Shipped', value: 15, color: '#8b5cf6' },
  { name: 'Delivered', value: 45, color: '#10b981' },
  { name: 'Cancelled', value: 3, color: '#F42A41' },
];

const recentOrders = [
  {
    id: 'ORD-2026-001',
    customer: 'Rahim Uddin',
    total: 4500,
    status: 'delivered',
    date: '2026-04-17',
  },
  {
    id: 'ORD-2026-002',
    customer: 'Fatima Akter',
    total: 12800,
    status: 'shipped',
    date: '2026-04-17',
  },
  {
    id: 'ORD-2026-003',
    customer: 'Kamal Hossain',
    total: 3200,
    status: 'confirmed',
    date: '2026-04-16',
  },
  {
    id: 'ORD-2026-004',
    customer: 'Nusrat Jahan',
    total: 8750,
    status: 'pending',
    date: '2026-04-16',
  },
  {
    id: 'ORD-2026-005',
    customer: 'Shakib Ahmed',
    total: 2100,
    status: 'cancelled',
    date: '2026-04-15',
  },
];

const topProducts = [
  { name: 'Premium Basmati Rice (5kg)', sold: 124, revenue: 186000 },
  { name: 'Organic Mustard Oil (1L)', sold: 98, revenue: 73500 },
  { name: 'Hilsha Fish (Fresh, 1kg)', sold: 87, revenue: 130500 },
  { name: 'Date Molasses (Khejur Gur)', sold: 76, revenue: 45600 },
  { name: 'Handloom Jamdani Saree', sold: 45, revenue: 337500 },
];

const maxProductSold = Math.max(...topProducts.map((p) => p.sold));

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
// Page
// ---------------------------------------------------------------------------

export default function DashboardPage() {
  const { tenantId, token } = useAuthStore();
  const [liveStats, setLiveStats] = useState(stats);
  const [liveRevenue, setLiveRevenue] = useState(revenueData);
  const [liveOrders, setLiveOrders] = useState(recentOrders);
  const [liveProducts, setLiveProducts] = useState(topProducts);
  const [liveOrderStatus, setLiveOrderStatus] = useState(orderStatusData);

  useEffect(() => {
    if (!tenantId) return;
    const endDate = new Date().toISOString().split('T')[0];
    const startDate = new Date(Date.now() - 7 * 86400000).toISOString().split('T')[0];

    analyticsApi.sales(tenantId, startDate, endDate, 'daily').then((res) => {
      if (res.data_points && res.data_points.length > 0) {
        setLiveRevenue(res.data_points.map((dp) => ({
          day: new Date(dp.date).toLocaleDateString('en-BD', { weekday: 'short' }),
          revenue: dp.revenue,
        })));
        setLiveStats((prev) => prev.map((s) => {
          if (s.title === 'Total Revenue') return { ...s, value: formatCurrency(res.total_revenue) };
          if (s.title === 'Total Orders') return { ...s, value: res.total_orders.toLocaleString() };
          if (s.title === 'Avg Order Value') return { ...s, value: formatCurrency(res.avg_order_value) };
          return s;
        }));
      }
    }).catch(() => {});

    orderApi.listByTenant(tenantId, token || undefined, 1, 5).then((res) => {
      if (res.data && res.data.length > 0) {
        setLiveOrders(res.data.map((o) => ({
          id: o.order_number,
          customer: o.customer_id,
          total: o.total,
          status: o.status,
          date: o.created_at?.split('T')[0] || '',
        })));
        const statusCounts: Record<string, number> = {};
        res.data.forEach((o) => { statusCounts[o.status] = (statusCounts[o.status] || 0) + 1; });
        const colors: Record<string, string> = { pending: '#f59e0b', confirmed: '#3b82f6', shipped: '#8b5cf6', delivered: '#10b981', cancelled: '#F42A41' };
        const statusArr = Object.entries(statusCounts).map(([name, value]) => ({ name, value, color: colors[name] || '#9ca3af' }));
        if (statusArr.length > 0) setLiveOrderStatus(statusArr);
      }
    }).catch(() => {});

    analyticsApi.products(tenantId, startDate, endDate).then((res) => {
      if (res.top_products && res.top_products.length > 0) {
        setLiveProducts(res.top_products.map((p) => ({
          name: p.name,
          sold: p.units_sold,
          revenue: p.revenue,
        })));
      }
    }).catch(() => {});
  }, [tenantId]);

  const maxSold = Math.max(...liveProducts.map((p) => p.sold), 1);

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
      <motion.div
        className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4"
        variants={containerVariants}
        initial="hidden"
        animate="show"
      >
        {liveStats.map((stat) => {
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

              <div className="mt-3 flex items-end justify-between gap-4">
                <div>
                  <span className="text-2xl font-bold text-text">{stat.value}</span>
                  <div className="mt-1.5 flex items-center text-sm">
                    {stat.trend === 'up' ? (
                      <ArrowUpRight className="mr-0.5 h-4 w-4 text-green-500" />
                    ) : (
                      <ArrowDownRight className="mr-0.5 h-4 w-4 text-red-500" />
                    )}
                    <span
                      className={
                        stat.trend === 'up' ? 'text-green-500' : 'text-red-500'
                      }
                    >
                      {stat.change}
                    </span>
                    <span className="ml-1 text-text-muted">vs last month</span>
                  </div>
                </div>

                {/* Mini sparkline */}
                <div className="h-[30px] w-[80px] shrink-0">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={stat.sparkline}>
                      <Line
                        type="monotone"
                        dataKey="v"
                        stroke={stat.color}
                        strokeWidth={2}
                        dot={false}
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </div>
            </motion.div>
          );
        })}
      </motion.div>

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
          <div className="h-[260px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={liveRevenue}>
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
                <Tooltip
                  content={<ChartTooltip isCurrency />}
                />
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
          <div className="flex h-[200px] items-center justify-center">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={liveOrderStatus}
                  cx="50%"
                  cy="50%"
                  innerRadius={55}
                  outerRadius={80}
                  paddingAngle={3}
                  dataKey="value"
                  stroke="none"
                >
                  {liveOrderStatus.map((entry) => (
                    <Cell key={entry.name} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip
                  content={<ChartTooltip prefix="Orders: " />}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
          <div className="mt-2 flex flex-wrap justify-center gap-x-4 gap-y-1.5">
            {liveOrderStatus.map((s) => (
              <div key={s.name} className="flex items-center gap-1.5 text-xs text-text-secondary">
                <span
                  className="inline-block h-2.5 w-2.5 rounded-full"
                  style={{ backgroundColor: s.color }}
                />
                {s.name} ({s.value})
              </div>
            ))}
          </div>
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
                {liveOrders.map((order) => (
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
          <ul className="space-y-4">
            {liveProducts.map((product, i) => (
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
        </motion.div>
      </div>
    </div>
  );
}
