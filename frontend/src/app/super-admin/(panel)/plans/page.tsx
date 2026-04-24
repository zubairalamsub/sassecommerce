'use client';

import { useEffect } from 'react';
import { Check, Star } from 'lucide-react';
import { cn, formatCurrency } from '@/lib/utils';
import { useTenantStore } from '@/stores/tenants';

interface PlanDef {
  name: string;
  tier: string;
  price: number;
  description: string;
  features: string[];
  popular: boolean;
  tierColor: string;
}

const planDefs: PlanDef[] = [
  {
    name: 'Free', tier: 'free', price: 0,
    description: 'For trying out the platform',
    features: ['100 products', '2 staff members', 'Basic support'],
    popular: false, tierColor: 'border-gray-200',
  },
  {
    name: 'Starter', tier: 'starter', price: 2999,
    description: 'For small businesses getting started',
    features: ['1,000 products', '5 staff members', 'Email support'],
    popular: false, tierColor: 'border-blue-200',
  },
  {
    name: 'Professional', tier: 'professional', price: 7999,
    description: 'For growing businesses',
    features: ['Unlimited products', '15 staff members', 'Priority support', 'Analytics'],
    popular: true, tierColor: 'border-indigo-400',
  },
  {
    name: 'Enterprise', tier: 'enterprise', price: 19999,
    description: 'For large-scale operations',
    features: ['Unlimited everything', 'Dedicated support', 'Custom integrations', 'SLA'],
    popular: false, tierColor: 'border-purple-200',
  },
];

export default function PlansPage() {
  const { tenants, fetchTenants } = useTenantStore();

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const subscriberCounts = tenants.reduce<Record<string, number>>((acc, t) => {
    acc[t.tier] = (acc[t.tier] || 0) + 1;
    return acc;
  }, {});

  const plans = planDefs.map((p) => ({
    ...p,
    subscribers: subscriberCounts[p.tier] || 0,
  }));

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Plans & Billing</h1>
        <p className="mt-1 text-sm text-gray-500">
          Manage subscription plans and view billing overview
        </p>
      </div>

      {/* Plan Cards */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
        {plans.map((plan) => (
          <div
            key={plan.name}
            className={cn(
              'relative flex flex-col rounded-xl border-2 bg-white p-6 shadow-sm transition-shadow hover:shadow-md',
              plan.popular ? 'border-indigo-400 ring-1 ring-indigo-400' : plan.tierColor,
            )}
          >
            {plan.popular && (
              <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                <span className="inline-flex items-center gap-1 rounded-full bg-indigo-600 px-3 py-1 text-xs font-medium text-white">
                  <Star className="h-3 w-3 fill-current" />
                  Most Popular
                </span>
              </div>
            )}

            <div className="mb-4">
              <h3 className="text-lg font-semibold text-gray-900">{plan.name}</h3>
              <p className="mt-1 text-sm text-gray-500">{plan.description}</p>
            </div>

            <div className="mb-6">
              <span className="text-3xl font-bold text-gray-900">
                {plan.price === 0 ? 'Free' : formatCurrency(plan.price)}
              </span>
              {plan.price > 0 && (
                <span className="text-sm text-gray-500">/mo</span>
              )}
            </div>

            <ul className="mb-6 flex-1 space-y-3">
              {plan.features.map((feature) => (
                <li key={feature} className="flex items-start gap-2">
                  <Check className={cn('mt-0.5 h-4 w-4 shrink-0', plan.popular ? 'text-indigo-600' : 'text-gray-400')} />
                  <span className="text-sm text-gray-600">{feature}</span>
                </li>
              ))}
            </ul>

            <div className="border-t border-gray-100 pt-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-500">Active stores</span>
                <span className="text-sm font-semibold text-gray-900">{plan.subscribers}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Revenue Summary */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">Revenue Summary</h2>
        <div className="mt-4 overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                <th className="pb-3 font-medium">Plan</th>
                <th className="pb-3 font-medium">Price</th>
                <th className="pb-3 font-medium">Stores</th>
                <th className="pb-3 text-right font-medium">Monthly Revenue</th>
              </tr>
            </thead>
            <tbody>
              {plans.map((plan) => (
                <tr key={plan.name} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                  <td className="py-3 text-sm font-medium text-gray-900">{plan.name}</td>
                  <td className="py-3 text-sm text-gray-500">
                    {plan.price === 0 ? 'Free' : formatCurrency(plan.price)}
                  </td>
                  <td className="py-3 text-sm text-gray-900">{plan.subscribers}</td>
                  <td className="py-3 text-right text-sm font-medium text-gray-900">
                    {formatCurrency(plan.price * plan.subscribers)}
                  </td>
                </tr>
              ))}
              <tr className="bg-gray-50">
                <td className="py-3 text-sm font-semibold text-gray-900" colSpan={3}>Total MRR</td>
                <td className="py-3 text-right text-sm font-bold text-indigo-600">
                  {formatCurrency(plans.reduce((sum, p) => sum + p.price * p.subscribers, 0))}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
