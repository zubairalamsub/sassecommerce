'use client';

import { useState, useEffect } from 'react';
import {
  CreditCard,
  Package,
  ShoppingCart,
  Users,
  HardDrive,
  ArrowUpRight,
  Check,
  Star,
  CalendarDays,
} from 'lucide-react';
import { motion } from 'framer-motion';
import { cn, formatCurrency, formatDate } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { tenantApi, productApi, orderApi, userApi } from '@/lib/api';
import type { Tenant } from '@/lib/api';

// ---------------------------------------------------------------------------
// Plan definitions (matching super-admin plans)
// ---------------------------------------------------------------------------

interface PlanDef {
  name: string;
  tier: Tenant['tier'];
  price: number;
  features: string[];
  limits: { products: number; staff: number; storage: number };
  popular: boolean;
}

const plans: PlanDef[] = [
  {
    name: 'Free',
    tier: 'free',
    price: 0,
    features: ['100 products', '2 staff members', 'Basic support'],
    limits: { products: 100, staff: 2, storage: 500 },
    popular: false,
  },
  {
    name: 'Starter',
    tier: 'starter',
    price: 2999,
    features: ['1,000 products', '5 staff members', 'Email support'],
    limits: { products: 1000, staff: 5, storage: 2048 },
    popular: false,
  },
  {
    name: 'Professional',
    tier: 'professional',
    price: 7999,
    features: ['Unlimited products', '15 staff members', 'Priority support', 'Analytics'],
    limits: { products: -1, staff: 15, storage: 10240 },
    popular: true,
  },
  {
    name: 'Enterprise',
    tier: 'enterprise',
    price: 19999,
    features: ['Unlimited everything', 'Dedicated support', 'Custom integrations', 'SLA'],
    limits: { products: -1, staff: -1, storage: -1 },
    popular: false,
  },
];

// No demo data — all usage is fetched from real APIs

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
// Usage Progress Bar
// ---------------------------------------------------------------------------

function UsageBar({ used, limit, color }: { used: number; limit: number; color: string }) {
  const percentage = limit === -1 ? 5 : Math.min((used / limit) * 100, 100);
  const isUnlimited = limit === -1;

  return (
    <div>
      <div className="flex items-center justify-between text-xs mb-1.5">
        <span className="text-text-secondary">
          {used.toLocaleString()} {isUnlimited ? 'used' : `/ ${limit.toLocaleString()}`}
        </span>
        {!isUnlimited && (
          <span className={cn(
            'font-medium',
            percentage > 90 ? 'text-red-500' : percentage > 70 ? 'text-amber-500' : 'text-text-muted',
          )}>
            {percentage.toFixed(0)}%
          </span>
        )}
        {isUnlimited && <span className="text-text-muted">Unlimited</span>}
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
        <motion.div
          className="h-full rounded-full"
          style={{ backgroundColor: color }}
          initial={{ width: 0 }}
          animate={{ width: `${percentage}%` }}
          transition={{ duration: 0.8, delay: 0.3 }}
        />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function BillingPage() {
  const { tenantId, token } = useAuthStore();
  const [currentTier, setCurrentTier] = useState<Tenant['tier']>('free');
  const [tenantName, setTenantName] = useState('My Store');
  const [usage, setUsage] = useState({ products: 0, orders: 0, staff: 0, storage: 0 });
  const [tenantCreated, setTenantCreated] = useState('');

  const currentPlan = plans.find((p) => p.tier === currentTier) || plans[0];

  useEffect(() => {
    if (!tenantId) return;

    tenantApi.get(tenantId).then((tenant) => {
      setCurrentTier(tenant.tier);
      setTenantName(tenant.name);
      setTenantCreated(tenant.created_at);
    }).catch(() => {});

    // Fetch real product count
    productApi.list(tenantId, 1, 1).then((res) => {
      setUsage((prev) => ({ ...prev, products: res.total ?? 0 }));
    }).catch(() => {});

    // Fetch real order count
    orderApi.listByTenant(tenantId, token || undefined, 1, 1).then((res) => {
      setUsage((prev) => ({ ...prev, orders: res.total ?? 0 }));
    }).catch(() => {});

    // Fetch real staff count — response has pagination.total_items
    if (token) {
      userApi.list(tenantId, token, 1, 1).then((res) => {
        const raw = res as { pagination?: { total_items?: number } };
        const count = res.total ?? raw.pagination?.total_items ?? 0;
        setUsage((prev) => ({ ...prev, staff: count }));
      }).catch(() => {});
    }
  }, [tenantId, token]);

  const usageCards = [
    {
      title: 'Products',
      value: usage.products,
      limit: currentPlan.limits.products,
      icon: Package,
      color: '#006A4E',
      bgColor: 'bg-emerald-100 dark:bg-emerald-900/40',
    },
    {
      title: 'Orders This Month',
      value: usage.orders,
      limit: -1,
      icon: ShoppingCart,
      color: '#3b82f6',
      bgColor: 'bg-blue-100 dark:bg-blue-900/40',
    },
    {
      title: 'Staff Accounts',
      value: usage.staff,
      limit: currentPlan.limits.staff,
      icon: Users,
      color: '#8b5cf6',
      bgColor: 'bg-violet-100 dark:bg-violet-900/40',
    },
    {
      title: 'Storage (MB)',
      value: usage.storage,
      limit: currentPlan.limits.storage,
      icon: HardDrive,
      color: '#f59e0b',
      bgColor: 'bg-amber-100 dark:bg-amber-900/40',
    },
  ];

  return (
    <div className="space-y-8">
      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <h1 className="text-2xl font-bold text-text">Billing</h1>
        <p className="mt-1 text-sm text-text-secondary">
          Manage your subscription plan and view usage for {tenantName}.
        </p>
      </motion.div>

      {/* Current Plan Card */}
      <motion.div
        className="rounded-2xl border border-border bg-surface p-6"
        variants={sectionVariants}
        initial="hidden"
        animate="show"
      >
        <div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex items-start gap-4">
            <span className="rounded-xl bg-primary/10 p-3">
              <CreditCard className="h-6 w-6 text-primary" />
            </span>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold text-text">
                  {currentPlan.name} Plan
                </h2>
                <span className={cn(
                  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                  currentTier === 'free'
                    ? 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300'
                    : 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
                )}>
                  {currentTier === 'free' ? 'Free Tier' : 'Active'}
                </span>
              </div>
              <p className="mt-1 text-2xl font-bold text-text">
                {currentPlan.price === 0 ? 'Free' : (
                  <>
                    {formatCurrency(currentPlan.price)}
                    <span className="text-sm font-normal text-text-muted">/mo</span>
                  </>
                )}
              </p>
              {tenantCreated && (
                <div className="mt-2 flex items-center gap-1.5 text-sm text-text-secondary">
                  <CalendarDays className="h-4 w-4" />
                  <span>Member since: {formatDate(tenantCreated.split('T')[0])}</span>
                </div>
              )}
            </div>
          </div>
          {currentTier !== 'enterprise' && (
            <button className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark">
              <ArrowUpRight className="h-4 w-4" />
              Upgrade Plan
            </button>
          )}
        </div>
      </motion.div>

      {/* Usage Cards */}
      <motion.div
        className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4"
        variants={containerVariants}
        initial="hidden"
        animate="show"
      >
        {usageCards.map((card) => {
          const Icon = card.icon;
          return (
            <motion.div
              key={card.title}
              variants={cardVariants}
              className="rounded-2xl border border-border bg-surface p-6"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-text-secondary">{card.title}</span>
                <span className={cn('rounded-xl p-2.5', card.bgColor)}>
                  <Icon className="h-5 w-5" style={{ color: card.color }} />
                </span>
              </div>
              <div className="mt-3">
                <span className="text-2xl font-bold text-text">{card.value.toLocaleString()}</span>
              </div>
              <div className="mt-3">
                <UsageBar used={card.value} limit={card.limit} color={card.color} />
              </div>
            </motion.div>
          );
        })}
      </motion.div>

      {/* Plans */}
      <motion.div
        className="rounded-2xl border border-border bg-surface p-6"
        variants={sectionVariants}
        initial="hidden"
        animate="show"
        transition={{ delay: 0.3 }}
      >
          <h2 className="mb-4 text-lg font-semibold text-text">Available Plans</h2>
          <ul className="space-y-3">
            {plans.map((plan) => {
              const isCurrentPlan = plan.tier === currentTier;
              const isUpgrade = plans.indexOf(plan) > plans.findIndex((p) => p.tier === currentTier);
              return (
                <li
                  key={plan.tier}
                  className={cn(
                    'rounded-xl border p-4 transition-colors',
                    isCurrentPlan
                      ? 'border-primary bg-primary/5'
                      : 'border-border hover:border-primary/30',
                  )}
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-1.5">
                        <span className="text-sm font-semibold text-text">{plan.name}</span>
                        {plan.popular && (
                          <Star className="h-3.5 w-3.5 fill-amber-400 text-amber-400" />
                        )}
                      </div>
                      <p className="mt-0.5 text-sm font-medium text-text-secondary">
                        {plan.price === 0 ? 'Free' : `${formatCurrency(plan.price)}/mo`}
                      </p>
                    </div>
                    {isCurrentPlan && (
                      <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                        <Check className="h-3 w-3" />
                        Current
                      </span>
                    )}
                    {isUpgrade && (
                      <button className="rounded-lg bg-primary/10 px-3 py-1 text-xs font-medium text-primary transition-colors hover:bg-primary/20">
                        Upgrade
                      </button>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
      </motion.div>
    </div>
  );
}
