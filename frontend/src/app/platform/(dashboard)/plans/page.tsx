'use client';

import { useState } from 'react';
import { Check, X } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn, formatCurrency } from '@/lib/utils';
import { useTenantStore } from '@/stores/tenants';

interface Plan {
  id: string;
  name: string;
  tier: string;
  price: number;
  description: string;
  features: { name: string; included: boolean }[];
  limits: { products: string; storage: string; users: string; orders: string };
  popular?: boolean;
}

const plans: Plan[] = [
  {
    id: 'free',
    name: 'Free',
    tier: 'free',
    price: 0,
    description: 'Get started with basic e-commerce features',
    features: [
      { name: 'Up to 25 products', included: true },
      { name: '1 admin user', included: true },
      { name: 'Basic analytics', included: true },
      { name: 'Email support', included: true },
      { name: 'Custom domain', included: false },
      { name: 'Priority support', included: false },
      { name: 'Advanced reports', included: false },
      { name: 'API access', included: false },
    ],
    limits: { products: '25', storage: '500 MB', users: '1', orders: '50/mo' },
  },
  {
    id: 'starter',
    name: 'Starter',
    tier: 'starter',
    price: 2999,
    description: 'Perfect for small businesses getting started',
    features: [
      { name: 'Up to 500 products', included: true },
      { name: '3 admin users', included: true },
      { name: 'Standard analytics', included: true },
      { name: 'Email + chat support', included: true },
      { name: 'Custom domain', included: true },
      { name: 'Priority support', included: false },
      { name: 'Advanced reports', included: false },
      { name: 'API access', included: false },
    ],
    limits: { products: '500', storage: '5 GB', users: '3', orders: '500/mo' },
  },
  {
    id: 'professional',
    name: 'Professional',
    tier: 'professional',
    price: 9999,
    description: 'For growing businesses that need more power',
    popular: true,
    features: [
      { name: 'Unlimited products', included: true },
      { name: '10 admin users', included: true },
      { name: 'Advanced analytics', included: true },
      { name: 'Priority support', included: true },
      { name: 'Custom domain', included: true },
      { name: 'Advanced reports', included: true },
      { name: 'API access', included: true },
      { name: 'Multi-language', included: false },
    ],
    limits: { products: 'Unlimited', storage: '50 GB', users: '10', orders: 'Unlimited' },
  },
  {
    id: 'enterprise',
    name: 'Enterprise',
    tier: 'enterprise',
    price: 29999,
    description: 'For large businesses with custom needs',
    features: [
      { name: 'Unlimited products', included: true },
      { name: 'Unlimited admin users', included: true },
      { name: 'Advanced analytics', included: true },
      { name: 'Dedicated support', included: true },
      { name: 'Custom domain', included: true },
      { name: 'Advanced reports', included: true },
      { name: 'Full API access', included: true },
      { name: 'Multi-language + currency', included: true },
    ],
    limits: { products: 'Unlimited', storage: '500 GB', users: 'Unlimited', orders: 'Unlimited' },
  },
];

export default function PlansPage() {
  const tenants = useTenantStore((s) => s.tenants);
  const [billing, setBilling] = useState<'monthly' | 'yearly'>('monthly');

  return (
    <div className="space-y-6">
      <motion.div initial={{ opacity: 0, y: -12 }} animate={{ opacity: 1, y: 0 }}>
        <h1 className="text-2xl font-bold text-text">Subscription Plans</h1>
        <p className="mt-1 text-sm text-text-secondary">Manage pricing tiers and features</p>
      </motion.div>

      {/* Billing toggle */}
      <div className="flex items-center justify-center gap-3">
        <span className={cn('text-sm font-medium', billing === 'monthly' ? 'text-text' : 'text-text-muted')}>Monthly</span>
        <button
          onClick={() => setBilling(billing === 'monthly' ? 'yearly' : 'monthly')}
          className={cn(
            'relative h-6 w-11 rounded-full transition-colors',
            billing === 'yearly' ? 'bg-violet-600' : 'bg-gray-300 dark:bg-gray-600',
          )}
        >
          <div
            className={cn(
              'absolute top-0.5 h-5 w-5 rounded-full bg-white transition-transform shadow',
              billing === 'yearly' ? 'translate-x-[22px]' : 'translate-x-0.5',
            )}
          />
        </button>
        <span className={cn('text-sm font-medium', billing === 'yearly' ? 'text-text' : 'text-text-muted')}>
          Yearly <span className="text-green-600 text-xs">(Save 20%)</span>
        </span>
      </div>

      {/* Plan cards */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
        {plans.map((plan, i) => {
          const subscriberCount = tenants.filter((t) => t.tier === plan.tier).length;
          const displayPrice = billing === 'yearly' ? Math.round(plan.price * 0.8) : plan.price;
          return (
            <motion.div
              key={plan.id}
              className={cn(
                'relative rounded-2xl border p-6 shadow-sm',
                plan.popular
                  ? 'border-violet-500 bg-surface ring-1 ring-violet-500'
                  : 'border-border bg-surface',
              )}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.1 }}
            >
              {plan.popular && (
                <div className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-violet-600 px-3 py-0.5 text-xs font-semibold text-white">
                  Most Popular
                </div>
              )}

              <h3 className="text-lg font-semibold text-text">{plan.name}</h3>
              <p className="mt-1 text-sm text-text-secondary">{plan.description}</p>

              <div className="mt-4">
                {displayPrice > 0 ? (
                  <div className="flex items-baseline gap-1">
                    <span className="text-3xl font-bold text-text">{formatCurrency(displayPrice)}</span>
                    <span className="text-sm text-text-muted">/{billing === 'yearly' ? 'mo' : 'mo'}</span>
                  </div>
                ) : (
                  <span className="text-3xl font-bold text-text">Free</span>
                )}
              </div>

              <div className="mt-2 text-xs text-text-muted">
                {subscriberCount} {subscriberCount === 1 ? 'subscriber' : 'subscribers'}
              </div>

              <hr className="my-4 border-border" />

              <ul className="space-y-2.5">
                {plan.features.map((f) => (
                  <li key={f.name} className="flex items-center gap-2 text-sm">
                    {f.included ? (
                      <Check className="h-4 w-4 flex-shrink-0 text-green-500" />
                    ) : (
                      <X className="h-4 w-4 flex-shrink-0 text-gray-300 dark:text-gray-600" />
                    )}
                    <span className={f.included ? 'text-text' : 'text-text-muted'}>{f.name}</span>
                  </li>
                ))}
              </ul>

              <hr className="my-4 border-border" />

              <div className="space-y-1.5 text-xs text-text-secondary">
                <div className="flex justify-between"><span>Products</span><span className="font-medium">{plan.limits.products}</span></div>
                <div className="flex justify-between"><span>Storage</span><span className="font-medium">{plan.limits.storage}</span></div>
                <div className="flex justify-between"><span>Users</span><span className="font-medium">{plan.limits.users}</span></div>
                <div className="flex justify-between"><span>Orders</span><span className="font-medium">{plan.limits.orders}</span></div>
              </div>
            </motion.div>
          );
        })}
      </div>
    </div>
  );
}
