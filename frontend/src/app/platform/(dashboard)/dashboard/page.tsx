'use client';

import { useEffect } from 'react';
import { Building2, Users, DollarSign, Activity, ArrowUpRight } from 'lucide-react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import { cn, formatCurrency, statusColor } from '@/lib/utils';
import { useTenantStore } from '@/stores/tenants';

const containerVariants = {
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { staggerChildren: 0.1 } },
};

const cardVariants = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4 } },
};

export default function PlatformDashboardPage() {
  const { tenants, fetchTenants } = useTenantStore();

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const activeTenants = tenants.filter((t) => t.status === 'active').length;
  const totalMRR = tenants.reduce((sum, t) => {
    const prices: Record<string, number> = { free: 0, starter: 2999, professional: 9999, enterprise: 29999 };
    return sum + (prices[t.tier] || 0);
  }, 0);

  const stats = [
    { title: 'Total Tenants', value: tenants.length.toString(), icon: Building2, color: 'bg-violet-100 dark:bg-violet-900/40', iconColor: '#8b5cf6' },
    { title: 'Active Tenants', value: activeTenants.toString(), icon: Activity, color: 'bg-green-100 dark:bg-green-900/40', iconColor: '#10b981' },
    { title: 'Monthly Revenue', value: formatCurrency(totalMRR), icon: DollarSign, color: 'bg-blue-100 dark:bg-blue-900/40', iconColor: '#3b82f6' },
    { title: 'Total Users', value: (tenants.length * 12).toString(), icon: Users, color: 'bg-amber-100 dark:bg-amber-900/40', iconColor: '#f59e0b' },
  ];

  const tierCounts = tenants.reduce<Record<string, number>>((acc, t) => {
    acc[t.tier] = (acc[t.tier] || 0) + 1;
    return acc;
  }, {});

  return (
    <div className="space-y-8">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <h1 className="text-2xl font-bold text-text">Platform Overview</h1>
        <p className="mt-1 text-sm text-text-secondary">
          Monitor your SaaS platform performance
        </p>
      </motion.div>

      {/* Stats */}
      <motion.div
        className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4"
        variants={containerVariants}
        initial="hidden"
        animate="show"
      >
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <motion.div
              key={stat.title}
              variants={cardVariants}
              className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
            >
              <div className="flex items-center justify-between">
                <div className={cn('rounded-xl p-3', stat.color)}>
                  <Icon className="h-5 w-5" style={{ color: stat.iconColor }} />
                </div>
              </div>
              <p className="mt-4 text-2xl font-bold text-text">{stat.value}</p>
              <p className="mt-1 text-sm text-text-secondary">{stat.title}</p>
            </motion.div>
          );
        })}
      </motion.div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Plan Distribution */}
        <motion.div
          className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <h2 className="text-lg font-semibold text-text">Plan Distribution</h2>
          <p className="mt-1 text-sm text-text-secondary">Tenants by subscription tier</p>
          <div className="mt-6 space-y-4">
            {['enterprise', 'professional', 'starter', 'free'].map((tier) => {
              const count = tierCounts[tier] || 0;
              const pct = tenants.length > 0 ? (count / tenants.length) * 100 : 0;
              const colors: Record<string, string> = {
                enterprise: 'bg-violet-500',
                professional: 'bg-blue-500',
                starter: 'bg-green-500',
                free: 'bg-gray-400',
              };
              return (
                <div key={tier}>
                  <div className="flex items-center justify-between text-sm">
                    <span className="capitalize font-medium text-text">{tier}</span>
                    <span className="text-text-secondary">{count} tenants</span>
                  </div>
                  <div className="mt-1.5 h-2 w-full rounded-full bg-surface-hover">
                    <div
                      className={cn('h-2 rounded-full transition-all', colors[tier])}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </motion.div>

        {/* Recent Tenants */}
        <motion.div
          className="lg:col-span-2 rounded-2xl border border-border bg-surface p-6 shadow-sm"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
        >
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-text">Recent Tenants</h2>
              <p className="mt-1 text-sm text-text-secondary">Latest store registrations</p>
            </div>
            <Link
              href="/platform/tenants"
              className="inline-flex items-center gap-1 text-sm font-medium text-violet-600 dark:text-violet-400 hover:underline"
            >
              View all <ArrowUpRight className="h-3.5 w-3.5" />
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border text-left text-xs text-text-secondary">
                  <th className="pb-3 font-medium">Store</th>
                  <th className="pb-3 font-medium">Email</th>
                  <th className="pb-3 font-medium">Plan</th>
                  <th className="pb-3 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {tenants.slice(0, 5).map((t) => (
                  <tr key={t.id} className="border-b border-border-light last:border-0">
                    <td className="py-3 text-sm font-medium text-text">{t.name}</td>
                    <td className="py-3 text-sm text-text-secondary">{t.email}</td>
                    <td className="py-3">
                      <span className="inline-flex rounded-full bg-violet-100 dark:bg-violet-900/30 px-2.5 py-0.5 text-xs font-medium capitalize text-violet-700 dark:text-violet-400">
                        {t.tier}
                      </span>
                    </td>
                    <td className="py-3">
                      <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(t.status))}>
                        {t.status}
                      </span>
                    </td>
                  </tr>
                ))}
                {tenants.length === 0 && (
                  <tr>
                    <td colSpan={4} className="py-8 text-center text-sm text-text-muted">
                      No tenants yet. Create your first tenant.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </motion.div>
      </div>
    </div>
  );
}
