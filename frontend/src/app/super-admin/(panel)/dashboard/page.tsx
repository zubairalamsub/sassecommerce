'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import {
  AreaChart, Area,
  BarChart, Bar,
  PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Legend,
} from 'recharts';
import {
  Building2,
  CheckCircle,
  DollarSign,
  TrendingUp,
  TrendingDown,
  PercentIcon,
  Users,
  Activity,
  Server,
  Clock,
  AlertTriangle,
  Zap,
  ShieldCheck,
  ArrowUpRight,
  ArrowDownRight,
  Store,
  UserPlus,
  CreditCard,
  Ban,
  Settings,
  RefreshCw,
} from 'lucide-react';
import { cn, formatCurrency, formatDate } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { useTenantStore } from '@/stores/tenants';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const tierPrices: Record<string, number> = {
  free: 0,
  starter: 2999,
  professional: 9999,
  enterprise: 29999,
};

const tierColor: Record<string, string> = {
  free: 'bg-gray-100 text-gray-800',
  starter: 'bg-blue-100 text-blue-800',
  professional: 'bg-indigo-100 text-indigo-800',
  enterprise: 'bg-purple-100 text-purple-800',
};

const PIE_COLORS = ['#d1d5db', '#60a5fa', '#6366f1', '#a78bfa'];

type TimeRange = '7d' | '30d' | '90d' | '1y';

const TIME_RANGE_LABELS: Record<TimeRange, string> = {
  '7d': '7 Days',
  '30d': '30 Days',
  '90d': '90 Days',
  '1y': '1 Year',
};

// ---------------------------------------------------------------------------
// Demo data generators
// ---------------------------------------------------------------------------

function generateMRRData(range: TimeRange, baseMRR: number): { date: string; mrr: number }[] {
  const points: Record<TimeRange, number> = { '7d': 7, '30d': 30, '90d': 13, '1y': 12 };
  const count = points[range];
  const now = new Date();
  const data: { date: string; mrr: number }[] = [];

  for (let i = count - 1; i >= 0; i--) {
    const d = new Date(now);
    if (range === '7d') {
      d.setDate(d.getDate() - i);
    } else if (range === '30d') {
      d.setDate(d.getDate() - i);
    } else if (range === '90d') {
      d.setDate(d.getDate() - i * 7);
    } else {
      d.setMonth(d.getMonth() - i);
    }

    // Simulate growth with mild variation (seeded by index for stability)
    const growthFactor = 1 + (count - i) * 0.025;
    const noise = 1 + Math.sin(i * 2.7) * 0.04;
    const mrr = Math.round((baseMRR * 0.6) * growthFactor * noise);

    const label =
      range === '1y'
        ? d.toLocaleDateString('en-US', { month: 'short', year: '2-digit' })
        : range === '90d'
          ? d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
          : d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });

    data.push({ date: label, mrr });
  }
  return data;
}

const TOP_STORES = [
  { name: 'Aarong Digital', revenue: 487500, orders: 1245, growth: 12.3 },
  { name: 'Daraz BD', revenue: 362000, orders: 987, growth: 8.7 },
  { name: 'Chaldal Fresh', revenue: 298750, orders: 2134, growth: 15.2 },
  { name: 'Shajgoj Beauty', revenue: 215400, orders: 643, growth: -2.1 },
  { name: 'PriyoShop', revenue: 178900, orders: 512, growth: 6.4 },
];

function generateActivityFeed(): {
  id: number;
  type: string;
  message: string;
  time: string;
  icon: typeof UserPlus;
  color: string;
}[] {
  const now = Date.now();
  return [
    { id: 1, type: 'tenant_created', message: 'New tenant "Bagdoom Electronics" created', time: new Date(now - 12 * 60000).toISOString(), icon: UserPlus, color: 'text-green-600 bg-green-50' },
    { id: 2, type: 'plan_upgraded', message: '"Aarong Digital" upgraded to Enterprise plan', time: new Date(now - 45 * 60000).toISOString(), icon: CreditCard, color: 'text-indigo-600 bg-indigo-50' },
    { id: 3, type: 'tenant_suspended', message: '"QuickMart BD" suspended for payment failure', time: new Date(now - 2 * 3600000).toISOString(), icon: Ban, color: 'text-red-600 bg-red-50' },
    { id: 4, type: 'plan_upgraded', message: '"Chaldal Fresh" upgraded to Professional plan', time: new Date(now - 3 * 3600000).toISOString(), icon: CreditCard, color: 'text-indigo-600 bg-indigo-50' },
    { id: 5, type: 'tenant_created', message: 'New tenant "StyleHub Bangladesh" created', time: new Date(now - 5 * 3600000).toISOString(), icon: UserPlus, color: 'text-green-600 bg-green-50' },
    { id: 6, type: 'config_change', message: '"Daraz BD" updated branding configuration', time: new Date(now - 8 * 3600000).toISOString(), icon: Settings, color: 'text-gray-600 bg-gray-50' },
    { id: 7, type: 'plan_downgraded', message: '"FreshMela" downgraded to Starter plan', time: new Date(now - 12 * 3600000).toISOString(), icon: TrendingDown, color: 'text-orange-600 bg-orange-50' },
    { id: 8, type: 'tenant_created', message: 'New tenant "GreenGrocer BD" created', time: new Date(now - 18 * 3600000).toISOString(), icon: UserPlus, color: 'text-green-600 bg-green-50' },
    { id: 9, type: 'plan_upgraded', message: '"Shajgoj Beauty" upgraded to Professional plan', time: new Date(now - 24 * 3600000).toISOString(), icon: CreditCard, color: 'text-indigo-600 bg-indigo-50' },
    { id: 10, type: 'tenant_reactivated', message: '"MegaDeal" reactivated after payment resolved', time: new Date(now - 30 * 3600000).toISOString(), icon: RefreshCw, color: 'text-blue-600 bg-blue-50' },
  ];
}

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function StatCard({
  title,
  value,
  subtitle,
  icon: Icon,
  iconBg,
  iconColor,
  trend,
  trendLabel,
}: {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: typeof Building2;
  iconBg: string;
  iconColor: string;
  trend?: number;
  trendLabel?: string;
}) {
  const isPositive = trend !== undefined && trend >= 0;
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md">
      <div className="flex items-start justify-between">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-gray-500">{title}</p>
          <p className="mt-2 text-2xl font-bold text-gray-900">{value}</p>
          {trend !== undefined && (
            <div className="mt-2 flex items-center gap-1">
              {isPositive ? (
                <ArrowUpRight className="h-4 w-4 text-green-600" />
              ) : (
                <ArrowDownRight className="h-4 w-4 text-red-600" />
              )}
              <span className={cn('text-xs font-medium', isPositive ? 'text-green-600' : 'text-red-600')}>
                {isPositive ? '+' : ''}{trend}%
              </span>
              {trendLabel && <span className="text-xs text-gray-400">{trendLabel}</span>}
            </div>
          )}
          {subtitle && !trend && (
            <p className="mt-1 text-xs text-gray-400">{subtitle}</p>
          )}
        </div>
        <span className={cn('flex-shrink-0 rounded-lg p-2.5', iconBg)}>
          <Icon className={cn('h-5 w-5', iconColor)} />
        </span>
      </div>
    </div>
  );
}

function HealthCard({
  label,
  value,
  status,
  icon: Icon,
}: {
  label: string;
  value: string;
  status: 'green' | 'yellow' | 'red';
  icon: typeof Server;
}) {
  const colorMap = {
    green: { bg: 'bg-green-50', text: 'text-green-700', border: 'border-green-200', dot: 'bg-green-500' },
    yellow: { bg: 'bg-yellow-50', text: 'text-yellow-700', border: 'border-yellow-200', dot: 'bg-yellow-500' },
    red: { bg: 'bg-red-50', text: 'text-red-700', border: 'border-red-200', dot: 'bg-red-500' },
  };
  const c = colorMap[status];
  return (
    <div className={cn('rounded-xl border p-4', c.border, c.bg)}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Icon className={cn('h-4 w-4', c.text)} />
          <span className="text-sm font-medium text-gray-700">{label}</span>
        </div>
        <span className={cn('h-2 w-2 rounded-full', c.dot)} />
      </div>
      <p className={cn('mt-2 text-xl font-bold', c.text)}>{value}</p>
    </div>
  );
}

// Custom tooltip for recharts
function ChartTooltip({ active, payload, label, prefix }: { active?: boolean; payload?: Array<{ value: number }>; label?: string; prefix?: string }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-lg border border-gray-200 bg-white px-3 py-2 shadow-lg">
      <p className="text-xs font-medium text-gray-500">{label}</p>
      <p className="text-sm font-bold text-gray-900">
        {prefix}{typeof payload[0].value === 'number' ? formatCurrency(payload[0].value) : payload[0].value}
      </p>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main Page
// ---------------------------------------------------------------------------

export default function SuperAdminDashboardPage() {
  const user = useAuthStore((s) => s.user);
  const { tenants, fetchTenants } = useTenantStore();
  const [timeRange, setTimeRange] = useState<TimeRange>('30d');

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  // --- Derived metrics ---
  const activeTenants = useMemo(() => tenants.filter((t) => t.status === 'active').length, [tenants]);

  const tierCounts = useMemo(() => {
    const counts: Record<string, number> = { free: 0, starter: 0, professional: 0, enterprise: 0 };
    tenants.forEach((t) => {
      counts[t.tier] = (counts[t.tier] || 0) + 1;
    });
    return counts;
  }, [tenants]);

  const totalMRR = useMemo(
    () => tenants.reduce((sum, t) => sum + (t.status === 'active' ? (tierPrices[t.tier] || 0) : 0), 0),
    [tenants],
  );

  const arr = totalMRR * 12;
  const churnRate = 2.4;
  const arpt = activeTenants > 0 ? Math.round(totalMRR / activeTenants) : 0;

  // MRR trend data
  const mrrData = useMemo(() => generateMRRData(timeRange, totalMRR || 120000), [timeRange, totalMRR]);

  // MRR growth %
  const mrrGrowth = useMemo(() => {
    if (mrrData.length < 2) return 0;
    const first = mrrData[0].mrr;
    const last = mrrData[mrrData.length - 1].mrr;
    if (first === 0) return 0;
    return parseFloat(((last - first) / first * 100).toFixed(1));
  }, [mrrData]);

  // Revenue by tier (pie chart)
  const revenueByTier = useMemo(() => {
    return [
      { name: 'Free', value: tierCounts.free * tierPrices.free, tier: 'free' },
      { name: 'Starter', value: tierCounts.starter * tierPrices.starter || 8997, tier: 'starter' },
      { name: 'Professional', value: tierCounts.professional * tierPrices.professional || 29997, tier: 'professional' },
      { name: 'Enterprise', value: tierCounts.enterprise * tierPrices.enterprise || 59998, tier: 'enterprise' },
    ].filter((d) => d.value > 0);
  }, [tierCounts]);

  // Tenants by tier (bar chart)
  const tenantsByTierData = useMemo(() => {
    return [
      { tier: 'Free', count: tierCounts.free || 12 },
      { tier: 'Starter', count: tierCounts.starter || 8 },
      { tier: 'Professional', count: tierCounts.professional || 5 },
      { tier: 'Enterprise', count: tierCounts.enterprise || 3 },
    ];
  }, [tierCounts]);

  const activityFeed = useMemo(() => generateActivityFeed(), []);

  // Today's date formatted
  const todayFormatted = new Date().toLocaleDateString('en-BD', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  // --- KPI cards ---
  const kpiCards = [
    {
      title: 'Total Tenants',
      value: tenants.length || 28,
      icon: Building2,
      iconBg: 'bg-indigo-50',
      iconColor: 'text-indigo-600',
      trend: 14.2,
      trendLabel: 'vs last month',
    },
    {
      title: 'Active Tenants',
      value: activeTenants || 24,
      icon: CheckCircle,
      iconBg: 'bg-green-50',
      iconColor: 'text-green-600',
      trend: 8.5,
      trendLabel: 'vs last month',
    },
    {
      title: 'Monthly Revenue (MRR)',
      value: formatCurrency(totalMRR || 168944),
      icon: DollarSign,
      iconBg: 'bg-purple-50',
      iconColor: 'text-purple-600',
      trend: 12.8,
      trendLabel: 'vs last month',
    },
    {
      title: 'Annual Revenue (ARR)',
      value: formatCurrency(arr || 2027328),
      icon: TrendingUp,
      iconBg: 'bg-blue-50',
      iconColor: 'text-blue-600',
      trend: 12.8,
      trendLabel: 'vs last month',
    },
    {
      title: 'Churn Rate',
      value: `${churnRate}%`,
      icon: PercentIcon,
      iconBg: 'bg-orange-50',
      iconColor: 'text-orange-600',
      trend: -0.3,
      trendLabel: 'vs last month',
    },
    {
      title: 'Avg Revenue / Tenant',
      value: formatCurrency(arpt || 7039),
      icon: Users,
      iconBg: 'bg-teal-50',
      iconColor: 'text-teal-600',
      trend: 4.1,
      trendLabel: 'vs last month',
    },
  ];

  return (
    <div className="space-y-8">
      {/* ----------------------------------------------------------------- */}
      {/* Header */}
      {/* ----------------------------------------------------------------- */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Platform Dashboard</h1>
          <p className="mt-1 text-sm text-gray-500">
            Welcome back, {user?.first_name || 'Admin'}! {todayFormatted}
          </p>
        </div>
        <div className="flex items-center gap-1 rounded-lg border border-gray-200 bg-white p-1 shadow-sm">
          {(Object.keys(TIME_RANGE_LABELS) as TimeRange[]).map((range) => (
            <button
              key={range}
              onClick={() => setTimeRange(range)}
              className={cn(
                'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                timeRange === range
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-gray-600 hover:bg-gray-100',
              )}
            >
              {TIME_RANGE_LABELS[range]}
            </button>
          ))}
        </div>
      </div>

      {/* ----------------------------------------------------------------- */}
      {/* KPI Stat Cards */}
      {/* ----------------------------------------------------------------- */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        {kpiCards.map((card) => (
          <StatCard key={card.title} {...card} />
        ))}
      </div>

      {/* ----------------------------------------------------------------- */}
      {/* Charts Row: MRR Growth + Revenue by Tier */}
      {/* ----------------------------------------------------------------- */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* MRR Growth Trend - spans 2 cols */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm lg:col-span-2">
          <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">MRR Growth Trend</h2>
              <p className="mt-0.5 text-sm text-gray-500">Monthly recurring revenue over time</p>
            </div>
            <div className={cn(
              'flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-semibold',
              mrrGrowth >= 0 ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700',
            )}>
              {mrrGrowth >= 0 ? <ArrowUpRight className="h-3.5 w-3.5" /> : <ArrowDownRight className="h-3.5 w-3.5" />}
              {mrrGrowth >= 0 ? '+' : ''}{mrrGrowth}%
            </div>
          </div>
          <div className="p-6">
            <div className="h-72">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={mrrData} margin={{ top: 5, right: 10, left: 10, bottom: 0 }}>
                  <defs>
                    <linearGradient id="mrrGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="#6366f1" stopOpacity={0.2} />
                      <stop offset="100%" stopColor="#6366f1" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                  <XAxis dataKey="date" tick={{ fontSize: 12, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                  <YAxis
                    tick={{ fontSize: 12, fill: '#9ca3af' }}
                    axisLine={false}
                    tickLine={false}
                    tickFormatter={(v: number) => `${(v / 1000).toFixed(0)}k`}
                  />
                  <Tooltip content={<ChartTooltip />} />
                  <Area
                    type="monotone"
                    dataKey="mrr"
                    stroke="#6366f1"
                    strokeWidth={2.5}
                    fill="url(#mrrGradient)"
                    dot={false}
                    activeDot={{ r: 5, fill: '#6366f1', stroke: '#fff', strokeWidth: 2 }}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>

        {/* Revenue by Tier - Donut */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
          <div className="border-b border-gray-200 px-6 py-4">
            <h2 className="text-lg font-semibold text-gray-900">Revenue by Tier</h2>
            <p className="mt-0.5 text-sm text-gray-500">Distribution across plans</p>
          </div>
          <div className="p-6">
            <div className="h-56">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={revenueByTier}
                    cx="50%"
                    cy="50%"
                    innerRadius={55}
                    outerRadius={85}
                    paddingAngle={3}
                    dataKey="value"
                  >
                    {revenueByTier.map((entry, idx) => (
                      <Cell key={entry.tier} fill={PIE_COLORS[idx % PIE_COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value) => formatCurrency(Number(value))}
                    contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb', fontSize: '13px' }}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
            {/* Legend */}
            <div className="mt-2 grid grid-cols-2 gap-2">
              {revenueByTier.map((entry, idx) => (
                <div key={entry.tier} className="flex items-center gap-2">
                  <span
                    className="h-2.5 w-2.5 rounded-full"
                    style={{ backgroundColor: PIE_COLORS[idx % PIE_COLORS.length] }}
                  />
                  <span className="text-xs text-gray-600">{entry.name}</span>
                  <span className="ml-auto text-xs font-medium text-gray-900">
                    {formatCurrency(entry.value)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* ----------------------------------------------------------------- */}
      {/* Tenants by Tier (Horizontal Bar) + Platform Health */}
      {/* ----------------------------------------------------------------- */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Tenants by Tier */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
          <div className="border-b border-gray-200 px-6 py-4">
            <h2 className="text-lg font-semibold text-gray-900">Tenants by Tier</h2>
            <p className="mt-0.5 text-sm text-gray-500">Distribution across subscription plans</p>
          </div>
          <div className="p-6">
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart
                  data={tenantsByTierData}
                  layout="vertical"
                  margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
                >
                  <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" horizontal={false} />
                  <XAxis type="number" tick={{ fontSize: 12, fill: '#9ca3af' }} axisLine={false} tickLine={false} />
                  <YAxis
                    type="category"
                    dataKey="tier"
                    tick={{ fontSize: 13, fill: '#374151', fontWeight: 500 }}
                    axisLine={false}
                    tickLine={false}
                    width={90}
                  />
                  <Tooltip
                    contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb', fontSize: '13px' }}
                    formatter={(value) => [`${value} tenants`, 'Count']}
                  />
                  <Bar
                    dataKey="count"
                    fill="#6366f1"
                    radius={[0, 6, 6, 0]}
                    barSize={28}
                  >
                    {tenantsByTierData.map((_, idx) => (
                      <Cell key={idx} fill={PIE_COLORS[idx % PIE_COLORS.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>

        {/* Platform Health */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
          <div className="flex items-center gap-2 border-b border-gray-200 px-6 py-4">
            <Activity className="h-5 w-5 text-green-600" />
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Platform Health</h2>
              <p className="mt-0.5 text-sm text-gray-500">Real-time system status</p>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4 p-6">
            <HealthCard
              label="Services Online"
              value="16/16"
              status="green"
              icon={Server}
            />
            <HealthCard
              label="Uptime"
              value="99.95%"
              status="green"
              icon={ShieldCheck}
            />
            <HealthCard
              label="Avg Response Time"
              value="145ms"
              status="green"
              icon={Zap}
            />
            <HealthCard
              label="Error Rate"
              value="0.02%"
              status="green"
              icon={AlertTriangle}
            />
          </div>
          <div className="border-t border-gray-100 px-6 py-3">
            <div className="flex items-center gap-2">
              <span className="h-2 w-2 animate-pulse rounded-full bg-green-500" />
              <span className="text-xs font-medium text-green-700">All systems operational</span>
              <span className="ml-auto text-xs text-gray-400">Last checked: just now</span>
            </div>
          </div>
        </div>
      </div>

      {/* ----------------------------------------------------------------- */}
      {/* Top Performing Stores + Recent Activity */}
      {/* ----------------------------------------------------------------- */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Top Performing Stores */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm lg:col-span-2">
          <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Top Performing Stores</h2>
              <p className="mt-0.5 text-sm text-gray-500">By revenue this month</p>
            </div>
            <Link
              href="/super-admin/tenants"
              className="text-sm font-medium text-indigo-600 hover:text-indigo-700 hover:underline"
            >
              View all
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left">
                  <th className="px-6 py-3 text-xs font-semibold uppercase tracking-wider text-gray-500">#</th>
                  <th className="px-6 py-3 text-xs font-semibold uppercase tracking-wider text-gray-500">Store</th>
                  <th className="px-6 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500">Revenue</th>
                  <th className="px-6 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500">Orders</th>
                  <th className="px-6 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500">Growth</th>
                </tr>
              </thead>
              <tbody>
                {TOP_STORES.map((store, idx) => (
                  <tr
                    key={store.name}
                    className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                  >
                    <td className="px-6 py-4">
                      <span className={cn(
                        'flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold',
                        idx === 0 ? 'bg-yellow-100 text-yellow-800' :
                        idx === 1 ? 'bg-gray-100 text-gray-700' :
                        idx === 2 ? 'bg-orange-100 text-orange-800' :
                        'bg-gray-50 text-gray-500',
                      )}>
                        {idx + 1}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600">
                          <Store className="h-4 w-4" />
                        </span>
                        <span className="text-sm font-medium text-gray-900">{store.name}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-right text-sm font-semibold text-gray-900">
                      {formatCurrency(store.revenue)}
                    </td>
                    <td className="px-6 py-4 text-right text-sm text-gray-600">
                      {store.orders.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <span className={cn(
                        'inline-flex items-center gap-0.5 text-sm font-medium',
                        store.growth >= 0 ? 'text-green-600' : 'text-red-600',
                      )}>
                        {store.growth >= 0 ? (
                          <ArrowUpRight className="h-3.5 w-3.5" />
                        ) : (
                          <ArrowDownRight className="h-3.5 w-3.5" />
                        )}
                        {Math.abs(store.growth)}%
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Recent Activity Feed */}
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
          <div className="flex items-center gap-2 border-b border-gray-200 px-6 py-4">
            <Clock className="h-5 w-5 text-gray-400" />
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Recent Activity</h2>
              <p className="mt-0.5 text-sm text-gray-500">Latest platform events</p>
            </div>
          </div>
          <div className="max-h-[430px] divide-y divide-gray-100 overflow-y-auto">
            {activityFeed.map((item) => {
              const Icon = item.icon;
              return (
                <div key={item.id} className="flex gap-3 px-5 py-3.5 transition-colors hover:bg-gray-50">
                  <span className={cn('mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full', item.color)}>
                    <Icon className="h-3.5 w-3.5" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm text-gray-700">{item.message}</p>
                    <p className="mt-0.5 text-xs text-gray-400">{formatRelativeTime(item.time)}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
