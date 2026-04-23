'use client';

import { useState, useEffect, useMemo } from 'react';
import {
  DollarSign,
  ShoppingCart,
  Users,
  TrendingUp,
  Package,
  ArrowUpRight,
} from 'lucide-react';
import { motion } from 'framer-motion';
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { formatCurrency } from '@/lib/utils';
import { analyticsApi } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

// ---------------------------------------------------------------------------
// Demo data generators
// ---------------------------------------------------------------------------

function generateDailyData(days: number) {
  const data = [];
  const now = new Date(2026, 3, 20); // April 20, 2026
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(now);
    d.setDate(d.getDate() - i);
    const base = 18000 + Math.round(Math.random() * 22000);
    const orders = 8 + Math.round(Math.random() * 18);
    data.push({
      date: d.toLocaleDateString('en-BD', { month: 'short', day: 'numeric' }),
      revenue: base,
      orders,
    });
  }
  return data;
}

const FULL_DAILY = generateDailyData(365);

const topCustomers = [
  { name: 'Fatima Akter', orders: 42, totalSpent: 187500, type: 'returning' as const },
  { name: 'Rahim Uddin', orders: 38, totalSpent: 162300, type: 'returning' as const },
  { name: 'Nusrat Jahan', orders: 35, totalSpent: 148900, type: 'returning' as const },
  { name: 'Kamal Hossain', orders: 29, totalSpent: 124600, type: 'returning' as const },
  { name: 'Tahmina Begum', orders: 27, totalSpent: 115200, type: 'returning' as const },
  { name: 'Shakib Ahmed', orders: 24, totalSpent: 98700, type: 'returning' as const },
  { name: 'Ayesha Siddiqua', orders: 21, totalSpent: 87400, type: 'returning' as const },
  { name: 'Mizanur Rahman', orders: 18, totalSpent: 76800, type: 'new' as const },
  { name: 'Rabeya Khatun', orders: 15, totalSpent: 64500, type: 'new' as const },
  { name: 'Jahangir Alam', orders: 12, totalSpent: 52100, type: 'new' as const },
];

const products = [
  { name: 'Jamdani Saree', unitsSold: 312, revenue: 468000, category: 'Sarees' },
  { name: 'Panjabi (Premium)', unitsSold: 278, revenue: 389200, category: 'Menswear' },
  { name: 'Muslin Dupatta', unitsSold: 245, revenue: 294000, category: 'Accessories' },
  { name: 'Nakshi Kantha', unitsSold: 198, revenue: 277200, category: 'Home Decor' },
  { name: 'Cotton Kurti', unitsSold: 187, revenue: 224400, category: 'Womenswear' },
  { name: 'Leather Wallet', unitsSold: 165, revenue: 198000, category: 'Accessories' },
  { name: 'Silk Lungi', unitsSold: 142, revenue: 170400, category: 'Menswear' },
  { name: 'Terracotta Set', unitsSold: 118, revenue: 141600, category: 'Home Decor' },
];

const categories = [
  { name: 'Sarees', value: 468000 },
  { name: 'Menswear', value: 559600 },
  { name: 'Womenswear', value: 224400 },
  { name: 'Accessories', value: 492000 },
  { name: 'Home Decor', value: 418800 },
  { name: 'Footwear', value: 156200 },
];

const CHART_COLORS = [
  'var(--color-primary)',
  '#0EA5E9',
  '#F59E0B',
  '#EF4444',
  '#8B5CF6',
  '#EC4899',
];

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type DateRange = '7d' | '30d' | '90d' | '1y';
type Tab = 'sales' | 'customers' | 'products';

const DATE_RANGE_OPTIONS: { value: DateRange; label: string }[] = [
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: '90d', label: 'Last 90 days' },
  { value: '1y', label: 'This Year' },
];

const TABS: { value: Tab; label: string; icon: typeof DollarSign }[] = [
  { value: 'sales', label: 'Sales', icon: DollarSign },
  { value: 'customers', label: 'Customers', icon: Users },
  { value: 'products', label: 'Products', icon: Package },
];

const rangeDays: Record<DateRange, number> = {
  '7d': 7,
  '30d': 30,
  '90d': 90,
  '1y': 365,
};

// ---------------------------------------------------------------------------
// Animations
// ---------------------------------------------------------------------------

const fadeUp = {
  hidden: { opacity: 0, y: 20 },
  visible: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: i * 0.07, duration: 0.4, ease: [0, 0, 0.2, 1] as const },
  }),
};

const stagger = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.07 } },
};

// ---------------------------------------------------------------------------
// Custom tooltip
// ---------------------------------------------------------------------------

function ChartTooltip({
  active,
  payload,
  label,
  currency = false,
}: {
  active?: boolean;
  payload?: { value: number; name: string; color: string }[];
  label?: string;
  currency?: boolean;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-xl border border-border bg-surface px-4 py-3 shadow-lg">
      <p className="mb-1 text-xs font-medium text-text-secondary">{label}</p>
      {payload.map((p, i) => (
        <p key={i} className="text-sm font-semibold text-text" style={{ color: p.color }}>
          {p.name}: {currency ? formatCurrency(p.value) : p.value.toLocaleString()}
        </p>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function ReportsPage() {
  const { tenantId } = useAuthStore();
  const [dateRange, setDateRange] = useState<DateRange>('30d');
  const [activeTab, setActiveTab] = useState<Tab>('sales');

  // Live data states (fallback to demo)
  const [liveDailyData, setLiveDailyData] = useState<typeof FULL_DAILY | null>(null);
  const [liveSalesTotal, setLiveSalesTotal] = useState<{ revenue: number; orders: number; aov: number } | null>(null);
  const [liveCustomerData, setLiveCustomerData] = useState<{ total: number; new_c: number; returning: number; top: typeof topCustomers } | null>(null);
  const [liveProductData, setLiveProductData] = useState<{ products: typeof products; categories: typeof categories } | null>(null);

  const days = rangeDays[dateRange];

  useEffect(() => {
    if (!tenantId) return;
    const endDate = new Date().toISOString().split('T')[0];
    const startDate = new Date(Date.now() - days * 86400000).toISOString().split('T')[0];

    analyticsApi.sales(tenantId, startDate, endDate, 'daily').then((res) => {
      if (res.data_points && res.data_points.length > 0) {
        setLiveDailyData(res.data_points.map((dp) => ({
          date: new Date(dp.date).toLocaleDateString('en-BD', { month: 'short', day: 'numeric' }),
          revenue: dp.revenue,
          orders: dp.orders,
        })));
        setLiveSalesTotal({ revenue: res.total_revenue, orders: res.total_orders, aov: res.avg_order_value });
      }
    }).catch(() => {});

    analyticsApi.customers(tenantId, startDate, endDate).then((res) => {
      if (res.total_customers > 0) {
        setLiveCustomerData({
          total: res.total_customers,
          new_c: res.new_customers,
          returning: res.returning_customers,
          top: res.top_customers.map((c) => ({
            name: c.name,
            orders: 0,
            totalSpent: c.total_spent,
            type: 'returning' as const,
          })),
        });
      }
    }).catch(() => {});

    analyticsApi.products(tenantId, startDate, endDate).then((res) => {
      if (res.top_products && res.top_products.length > 0) {
        setLiveProductData({
          products: res.top_products.map((p) => ({
            name: p.name,
            unitsSold: p.units_sold,
            revenue: p.revenue,
            category: '',
          })),
          categories: res.categories_breakdown.map((c) => ({
            name: c.category,
            value: c.revenue,
          })),
        });
      }
    }).catch(() => {});
  }, [tenantId, days]);

  const dailyData = liveDailyData || FULL_DAILY.slice(-days);
  const totalRevenue = liveSalesTotal?.revenue ?? dailyData.reduce((s, d) => s + d.revenue, 0);
  const totalOrders = liveSalesTotal?.orders ?? dailyData.reduce((s, d) => s + d.orders, 0);
  const avgOrderValue = liveSalesTotal?.aov ?? (totalOrders ? Math.round(totalRevenue / totalOrders) : 0);
  const growthRate = 12.5;

  const totalCustomers = liveCustomerData?.total ?? 1245;
  const newCustomers = liveCustomerData?.new_c ?? 186;
  const returningCustomers = liveCustomerData?.returning ?? (totalCustomers - newCustomers);
  const activeTopCustomers = liveCustomerData?.top ?? topCustomers;

  const activeProducts = liveProductData?.products ?? products;
  const activeCategories = liveProductData?.categories ?? categories;

  const maxCustomerSpent = activeTopCustomers[0]?.totalSpent ?? 1;

  const categoryTotal = activeCategories.reduce((s, c) => s + c.value, 0);
  const maxProductRevenue = activeProducts[0]?.revenue ?? 1;

  // ------------------------------------------------------------------
  // Render helpers
  // ------------------------------------------------------------------

  const salesStats = [
    {
      title: 'Total Revenue',
      value: formatCurrency(totalRevenue),
      icon: DollarSign,
      accent: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400',
    },
    {
      title: 'Total Orders',
      value: totalOrders.toLocaleString(),
      icon: ShoppingCart,
      accent: 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-400',
    },
    {
      title: 'Avg Order Value',
      value: formatCurrency(avgOrderValue),
      icon: TrendingUp,
      accent: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400',
    },
    {
      title: 'Growth Rate',
      value: `+${growthRate}%`,
      icon: ArrowUpRight,
      accent: 'bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-400',
    },
  ];

  const customerStats = [
    {
      title: 'Total Customers',
      value: totalCustomers.toLocaleString(),
      icon: Users,
      accent: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400',
    },
    {
      title: 'New Customers',
      value: newCustomers.toLocaleString(),
      icon: ArrowUpRight,
      accent: 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-400',
    },
    {
      title: 'Returning Customers',
      value: returningCustomers.toLocaleString(),
      icon: Users,
      accent: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400',
    },
  ];

  const customerPieData = [
    { name: 'New', value: newCustomers },
    { name: 'Returning', value: returningCustomers },
  ];

  // ------------------------------------------------------------------
  // JSX
  // ------------------------------------------------------------------

  return (
    <div className="space-y-8">
      {/* Header row */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text">Reports</h1>
          <p className="mt-1 text-sm text-text-secondary">
            Analytics and insights for your store.
          </p>
        </div>

        {/* Date range selector */}
        <div className="flex items-center gap-1 rounded-xl border border-border bg-surface p-1">
          {DATE_RANGE_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setDateRange(opt.value)}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${
                dateRange === opt.value
                  ? 'bg-primary text-white shadow-sm'
                  : 'text-text-secondary hover:bg-surface-hover hover:text-text'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Tab bar */}
      <div className="flex gap-1 rounded-xl border border-border bg-surface p-1">
        {TABS.map((tab) => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.value}
              onClick={() => setActiveTab(tab.value)}
              className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                activeTab === tab.value
                  ? 'bg-primary text-white shadow-sm'
                  : 'text-text-secondary hover:bg-surface-hover hover:text-text'
              }`}
            >
              <Icon className="h-4 w-4" />
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* ============================================================ */}
      {/* SALES TAB                                                     */}
      {/* ============================================================ */}
      {activeTab === 'sales' && (
        <motion.div
          key="sales"
          variants={stagger}
          initial="hidden"
          animate="visible"
          className="space-y-6"
        >
          {/* Stat cards */}
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {salesStats.map((stat, i) => {
              const Icon = stat.icon;
              return (
                <motion.div
                  key={stat.title}
                  custom={i}
                  variants={fadeUp}
                  className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-text-secondary">
                      {stat.title}
                    </span>
                    <span className={`rounded-xl p-2 ${stat.accent}`}>
                      <Icon className="h-5 w-5" />
                    </span>
                  </div>
                  <p className="mt-3 text-2xl font-bold text-text">{stat.value}</p>
                </motion.div>
              );
            })}
          </div>

          {/* Revenue area chart */}
          <motion.div
            custom={4}
            variants={fadeUp}
            className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
          >
            <h3 className="mb-4 text-lg font-semibold text-text">Revenue Over Time</h3>
            <div className="h-80">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={dailyData}>
                  <defs>
                    <linearGradient id="revGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--color-primary)" stopOpacity={0.3} />
                      <stop offset="100%" stopColor="var(--color-primary)" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 12, fill: 'var(--color-text-secondary)' }}
                    interval={days <= 7 ? 0 : days <= 30 ? 3 : days <= 90 ? 8 : 30}
                  />
                  <YAxis
                    tick={{ fontSize: 12, fill: 'var(--color-text-secondary)' }}
                    tickFormatter={(v: number) => `${(v / 1000).toFixed(0)}k`}
                  />
                  <Tooltip content={<ChartTooltip currency />} />
                  <Area
                    type="monotone"
                    dataKey="revenue"
                    name="Revenue"
                    stroke="var(--color-primary)"
                    strokeWidth={2}
                    fill="url(#revGrad)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </motion.div>

          {/* Orders bar chart */}
          <motion.div
            custom={5}
            variants={fadeUp}
            className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
          >
            <h3 className="mb-4 text-lg font-semibold text-text">Orders Per Day</h3>
            <div className="h-72">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={dailyData}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 12, fill: 'var(--color-text-secondary)' }}
                    interval={days <= 7 ? 0 : days <= 30 ? 3 : days <= 90 ? 8 : 30}
                  />
                  <YAxis tick={{ fontSize: 12, fill: 'var(--color-text-secondary)' }} />
                  <Tooltip content={<ChartTooltip />} />
                  <Bar
                    dataKey="orders"
                    name="Orders"
                    fill="var(--color-primary)"
                    radius={[6, 6, 0, 0]}
                  />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </motion.div>
        </motion.div>
      )}

      {/* ============================================================ */}
      {/* CUSTOMERS TAB                                                 */}
      {/* ============================================================ */}
      {activeTab === 'customers' && (
        <motion.div
          key="customers"
          variants={stagger}
          initial="hidden"
          animate="visible"
          className="space-y-6"
        >
          {/* Stat cards */}
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
            {customerStats.map((stat, i) => {
              const Icon = stat.icon;
              return (
                <motion.div
                  key={stat.title}
                  custom={i}
                  variants={fadeUp}
                  className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-text-secondary">
                      {stat.title}
                    </span>
                    <span className={`rounded-xl p-2 ${stat.accent}`}>
                      <Icon className="h-5 w-5" />
                    </span>
                  </div>
                  <p className="mt-3 text-2xl font-bold text-text">{stat.value}</p>
                </motion.div>
              );
            })}
          </div>

          {/* Pie + table row */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Donut chart */}
            <motion.div
              custom={3}
              variants={fadeUp}
              className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
            >
              <h3 className="mb-4 text-lg font-semibold text-text">
                New vs Returning Customers
              </h3>
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={customerPieData}
                      cx="50%"
                      cy="50%"
                      innerRadius={70}
                      outerRadius={100}
                      paddingAngle={4}
                      dataKey="value"
                      nameKey="name"
                      label={({ name, percent }: { name?: string; percent?: number }) =>
                        `${name ?? ''} ${((percent ?? 0) * 100).toFixed(0)}%`
                      }
                    >
                      {customerPieData.map((_, idx) => (
                        <Cell key={idx} fill={CHART_COLORS[idx]} />
                      ))}
                    </Pie>
                    <Tooltip content={<ChartTooltip />} />
                    <Legend />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            </motion.div>

            {/* Top customers */}
            <motion.div
              custom={4}
              variants={fadeUp}
              className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
            >
              <h3 className="mb-4 text-lg font-semibold text-text">Top Customers</h3>
              <div className="space-y-3">
                {activeTopCustomers.map((c) => (
                  <div key={c.name} className="group">
                    <div className="flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2">
                        <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-semibold text-primary">
                          {c.name
                            .split(' ')
                            .map((n) => n[0])
                            .join('')}
                        </div>
                        <span className="font-medium text-text">{c.name}</span>
                      </div>
                      <span className="font-semibold text-text">
                        {formatCurrency(c.totalSpent)}
                      </span>
                    </div>
                    {/* Horizontal bar */}
                    <div className="mt-1 ml-10 h-2 overflow-hidden rounded-full bg-surface-secondary">
                      <div
                        className="h-full rounded-full transition-all"
                        style={{
                          width: `${(c.totalSpent / maxCustomerSpent) * 100}%`,
                          backgroundColor: 'var(--color-primary)',
                        }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </motion.div>
          </div>
        </motion.div>
      )}

      {/* ============================================================ */}
      {/* PRODUCTS TAB                                                  */}
      {/* ============================================================ */}
      {activeTab === 'products' && (
        <motion.div
          key="products"
          variants={stagger}
          initial="hidden"
          animate="visible"
          className="space-y-6"
        >
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Top products horizontal bar chart */}
            <motion.div
              custom={0}
              variants={fadeUp}
              className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
            >
              <h3 className="mb-4 text-lg font-semibold text-text">
                Top Products by Revenue
              </h3>
              <div className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={activeProducts} layout="vertical">
                    <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                    <XAxis
                      type="number"
                      tick={{ fontSize: 12, fill: 'var(--color-text-secondary)' }}
                      tickFormatter={(v: number) => `${(v / 1000).toFixed(0)}k`}
                    />
                    <YAxis
                      type="category"
                      dataKey="name"
                      width={120}
                      tick={{ fontSize: 12, fill: 'var(--color-text-secondary)' }}
                    />
                    <Tooltip content={<ChartTooltip currency />} />
                    <Bar
                      dataKey="revenue"
                      name="Revenue"
                      fill="var(--color-primary)"
                      radius={[0, 6, 6, 0]}
                    />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </motion.div>

            {/* Category pie chart */}
            <motion.div
              custom={1}
              variants={fadeUp}
              className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
            >
              <h3 className="mb-4 text-lg font-semibold text-text">Category Breakdown</h3>
              <div className="h-80">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={activeCategories}
                      cx="50%"
                      cy="50%"
                      outerRadius={110}
                      paddingAngle={2}
                      dataKey="value"
                      nameKey="name"
                      label={({ name, percent }: { name?: string; percent?: number }) =>
                        `${name ?? ''} ${((percent ?? 0) * 100).toFixed(1)}%`
                      }
                    >
                      {activeCategories.map((_, idx) => (
                        <Cell key={idx} fill={CHART_COLORS[idx % CHART_COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip content={<ChartTooltip currency />} />
                    <Legend />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            </motion.div>
          </div>

          {/* Product performance table */}
          <motion.div
            custom={2}
            variants={fadeUp}
            className="rounded-2xl border border-border bg-surface shadow-sm"
          >
            <div className="border-b border-border px-6 py-4">
              <h3 className="text-lg font-semibold text-text">Product Performance</h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border text-left text-sm text-text-secondary">
                    <th className="px-6 py-3 font-medium">Product</th>
                    <th className="px-6 py-3 font-medium">Category</th>
                    <th className="px-6 py-3 font-medium text-right">Units Sold</th>
                    <th className="px-6 py-3 font-medium text-right">Revenue</th>
                    <th className="px-6 py-3 font-medium" style={{ minWidth: 160 }}>
                      Share
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {activeProducts.map((p) => (
                    <tr
                      key={p.name}
                      className="border-b border-border-light transition-colors hover:bg-surface-hover"
                    >
                      <td className="px-6 py-4 text-sm font-medium text-text">{p.name}</td>
                      <td className="px-6 py-4 text-sm text-text-secondary">{p.category}</td>
                      <td className="px-6 py-4 text-right text-sm text-text">
                        {p.unitsSold.toLocaleString()}
                      </td>
                      <td className="px-6 py-4 text-right text-sm font-semibold text-text">
                        {formatCurrency(p.revenue)}
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2">
                          <div className="h-2 flex-1 overflow-hidden rounded-full bg-surface-secondary">
                            <div
                              className="h-full rounded-full"
                              style={{
                                width: `${(p.revenue / maxProductRevenue) * 100}%`,
                                backgroundColor: 'var(--color-primary)',
                              }}
                            />
                          </div>
                          <span className="w-10 text-right text-xs text-text-muted">
                            {((p.revenue / categoryTotal) * 100).toFixed(1)}%
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </motion.div>
        </motion.div>
      )}
    </div>
  );
}
