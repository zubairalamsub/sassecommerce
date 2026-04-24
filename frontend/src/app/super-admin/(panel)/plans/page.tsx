'use client';

import { useEffect, useState } from 'react';
import {
  Check,
  X,
  Star,
  Plus,
  Pencil,
  Trash2,
  Download,
  Bell,
  CreditCard,
  Receipt,
  Clock,
  Filter,
  Smartphone,
  Globe,
  BarChart3,
  Code,
  Headphones,
  Palette,
  DollarSign,
  Truck,
  ShoppingCart,
  Settings,
  ChevronDown,
  AlertTriangle,
} from 'lucide-react';
import { cn, formatCurrency, formatDate } from '@/lib/utils';
import { useTenantStore } from '@/stores/tenants';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface PlanFeatures {
  customDomain: boolean;
  analytics: boolean;
  apiAccess: boolean;
  prioritySupport: boolean;
  whiteLabeling: boolean;
  multiCurrency: boolean;
  advancedShipping: boolean;
  customCheckout: boolean;
}

interface PlanLimits {
  maxProducts: number; // -1 = unlimited
  maxStorageGB: number;
  maxUsers: number;
  maxOrdersPerMonth: number; // -1 = unlimited
}

interface Plan {
  id: string;
  name: string;
  tier: string;
  description: string;
  monthlyPrice: number;
  yearlyPrice: number;
  features: PlanFeatures;
  limits: PlanLimits;
  trialDays: number;
  popular: boolean;
}

type InvoiceStatus = 'paid' | 'pending' | 'overdue' | 'cancelled';

interface Invoice {
  id: string;
  invoiceNumber: string;
  tenantName: string;
  plan: string;
  amount: number;
  status: InvoiceStatus;
  dueDate: string;
  paidDate: string | null;
}

interface BillingSettings {
  taxRate: number;
  invoicePrefix: string;
  billingCycleDay: number;
  gracePeriodDays: number;
  autoSuspendEnabled: boolean;
  autoSuspendAfterDays: number;
}

interface PaymentGateway {
  id: string;
  name: string;
  logo: string;
  connected: boolean;
  enabled: boolean;
  description: string;
}

// ---------------------------------------------------------------------------
// Feature metadata for display
// ---------------------------------------------------------------------------

const featureMeta: { key: keyof PlanFeatures; label: string; icon: React.ElementType }[] = [
  { key: 'customDomain', label: 'Custom Domain', icon: Globe },
  { key: 'analytics', label: 'Analytics Dashboard', icon: BarChart3 },
  { key: 'apiAccess', label: 'API Access', icon: Code },
  { key: 'prioritySupport', label: 'Priority Support', icon: Headphones },
  { key: 'whiteLabeling', label: 'White Labeling', icon: Palette },
  { key: 'multiCurrency', label: 'Multi-Currency', icon: DollarSign },
  { key: 'advancedShipping', label: 'Advanced Shipping', icon: Truck },
  { key: 'customCheckout', label: 'Custom Checkout', icon: ShoppingCart },
];

// ---------------------------------------------------------------------------
// Default plans
// ---------------------------------------------------------------------------

const defaultPlans: Plan[] = [
  {
    id: 'plan_free',
    name: 'Free',
    tier: 'free',
    description: 'For trying out the platform with basic features',
    monthlyPrice: 0,
    yearlyPrice: 0,
    features: {
      customDomain: false,
      analytics: false,
      apiAccess: false,
      prioritySupport: false,
      whiteLabeling: false,
      multiCurrency: false,
      advancedShipping: false,
      customCheckout: false,
    },
    limits: { maxProducts: 10, maxStorageGB: 0.1, maxUsers: 1, maxOrdersPerMonth: 50 },
    trialDays: 0,
    popular: false,
  },
  {
    id: 'plan_starter',
    name: 'Starter',
    tier: 'starter',
    description: 'For small businesses getting started online',
    monthlyPrice: 2999,
    yearlyPrice: 29990,
    features: {
      customDomain: true,
      analytics: true,
      apiAccess: false,
      prioritySupport: false,
      whiteLabeling: false,
      multiCurrency: false,
      advancedShipping: false,
      customCheckout: true,
    },
    limits: { maxProducts: 500, maxStorageGB: 5, maxUsers: 5, maxOrdersPerMonth: 1000 },
    trialDays: 14,
    popular: false,
  },
  {
    id: 'plan_professional',
    name: 'Professional',
    tier: 'professional',
    description: 'For growing businesses that need more power',
    monthlyPrice: 9999,
    yearlyPrice: 99990,
    features: {
      customDomain: true,
      analytics: true,
      apiAccess: true,
      prioritySupport: true,
      whiteLabeling: false,
      multiCurrency: true,
      advancedShipping: true,
      customCheckout: true,
    },
    limits: { maxProducts: -1, maxStorageGB: 50, maxUsers: 25, maxOrdersPerMonth: -1 },
    trialDays: 14,
    popular: true,
  },
  {
    id: 'plan_enterprise',
    name: 'Enterprise',
    tier: 'enterprise',
    description: 'For large-scale operations with full capabilities',
    monthlyPrice: 29999,
    yearlyPrice: 299990,
    features: {
      customDomain: true,
      analytics: true,
      apiAccess: true,
      prioritySupport: true,
      whiteLabeling: true,
      multiCurrency: true,
      advancedShipping: true,
      customCheckout: true,
    },
    limits: { maxProducts: -1, maxStorageGB: -1, maxUsers: -1, maxOrdersPerMonth: -1 },
    trialDays: 30,
    popular: false,
  },
];

// ---------------------------------------------------------------------------
// Demo invoices
// ---------------------------------------------------------------------------

const demoInvoices: Invoice[] = [
  { id: 'inv_1', invoiceNumber: 'INV-2026-001', tenantName: 'Aarong Online', plan: 'Enterprise', amount: 29999, status: 'paid', dueDate: '2026-03-01', paidDate: '2026-02-28' },
  { id: 'inv_2', invoiceNumber: 'INV-2026-002', tenantName: 'Daraz BD', plan: 'Professional', amount: 9999, status: 'paid', dueDate: '2026-03-01', paidDate: '2026-03-01' },
  { id: 'inv_3', invoiceNumber: 'INV-2026-003', tenantName: 'Chaldal Groceries', plan: 'Professional', amount: 9999, status: 'paid', dueDate: '2026-03-05', paidDate: '2026-03-04' },
  { id: 'inv_4', invoiceNumber: 'INV-2026-004', tenantName: 'Shajgoj Beauty', plan: 'Starter', amount: 2999, status: 'pending', dueDate: '2026-04-28', paidDate: null },
  { id: 'inv_5', invoiceNumber: 'INV-2026-005', tenantName: 'Bagdoom Electronics', plan: 'Enterprise', amount: 29999, status: 'paid', dueDate: '2026-03-10', paidDate: '2026-03-09' },
  { id: 'inv_6', invoiceNumber: 'INV-2026-006', tenantName: 'Othoba Mart', plan: 'Starter', amount: 2999, status: 'overdue', dueDate: '2026-04-10', paidDate: null },
  { id: 'inv_7', invoiceNumber: 'INV-2026-007', tenantName: 'Priyoshop Fashion', plan: 'Professional', amount: 9999, status: 'paid', dueDate: '2026-03-15', paidDate: '2026-03-14' },
  { id: 'inv_8', invoiceNumber: 'INV-2026-008', tenantName: 'Ekshop Digital', plan: 'Starter', amount: 2999, status: 'paid', dueDate: '2026-03-15', paidDate: '2026-03-15' },
  { id: 'inv_9', invoiceNumber: 'INV-2026-009', tenantName: 'Rokomari Books', plan: 'Professional', amount: 9999, status: 'pending', dueDate: '2026-04-30', paidDate: null },
  { id: 'inv_10', invoiceNumber: 'INV-2026-010', tenantName: 'Pickaboo Tech', plan: 'Enterprise', amount: 29999, status: 'cancelled', dueDate: '2026-03-20', paidDate: null },
  { id: 'inv_11', invoiceNumber: 'INV-2026-011', tenantName: 'Sindabad Wholesale', plan: 'Starter', amount: 2999, status: 'paid', dueDate: '2026-03-20', paidDate: '2026-03-19' },
  { id: 'inv_12', invoiceNumber: 'INV-2026-012', tenantName: 'Bikroy Marketplace', plan: 'Professional', amount: 9999, status: 'overdue', dueDate: '2026-04-05', paidDate: null },
  { id: 'inv_13', invoiceNumber: 'INV-2026-013', tenantName: 'Deshi Crafts', plan: 'Free', amount: 0, status: 'paid', dueDate: '2026-03-25', paidDate: '2026-03-25' },
  { id: 'inv_14', invoiceNumber: 'INV-2026-014', tenantName: 'PriyoBazar Retail', plan: 'Starter', amount: 2999, status: 'pending', dueDate: '2026-05-01', paidDate: null },
  { id: 'inv_15', invoiceNumber: 'INV-2026-015', tenantName: 'Meena Bazaar Online', plan: 'Enterprise', amount: 29999, status: 'paid', dueDate: '2026-04-01', paidDate: '2026-03-30' },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatLimit(value: number, suffix = ''): string {
  if (value === -1) return 'Unlimited';
  if (suffix === 'GB' && value < 1) return `${value * 1000}MB`;
  return `${value.toLocaleString()}${suffix ? ` ${suffix}` : ''}`;
}

const statusBadgeColor: Record<InvoiceStatus, string> = {
  paid: 'bg-green-100 text-green-800',
  pending: 'bg-yellow-100 text-yellow-800',
  overdue: 'bg-red-100 text-red-800',
  cancelled: 'bg-gray-100 text-gray-600',
};

const emptyPlanForm: Plan = {
  id: '',
  name: '',
  tier: '',
  description: '',
  monthlyPrice: 0,
  yearlyPrice: 0,
  features: {
    customDomain: false,
    analytics: false,
    apiAccess: false,
    prioritySupport: false,
    whiteLabeling: false,
    multiCurrency: false,
    advancedShipping: false,
    customCheckout: false,
  },
  limits: { maxProducts: 100, maxStorageGB: 1, maxUsers: 1, maxOrdersPerMonth: 100 },
  trialDays: 14,
  popular: false,
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function PlansPage() {
  const { tenants, fetchTenants } = useTenantStore();

  // Tab state
  const [activeTab, setActiveTab] = useState<'plans' | 'billing'>('plans');

  // Plans state
  const [plans, setPlans] = useState<Plan[]>(defaultPlans);
  const [showYearly, setShowYearly] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<Plan | null>(null);
  const [planForm, setPlanForm] = useState<Plan>(emptyPlanForm);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);

  // Billing state
  const [invoices, setInvoices] = useState<Invoice[]>(demoInvoices);
  const [invoiceFilter, setInvoiceFilter] = useState<InvoiceStatus | 'all'>('all');
  const [billingSettings, setBillingSettings] = useState<BillingSettings>({
    taxRate: 15,
    invoicePrefix: 'INV',
    billingCycleDay: 1,
    gracePeriodDays: 7,
    autoSuspendEnabled: true,
    autoSuspendAfterDays: 14,
  });
  const [gateways, setGateways] = useState<PaymentGateway[]>([
    { id: 'sslcommerz', name: 'SSLCommerz', logo: '🔒', connected: true, enabled: true, description: 'Accept Visa, Mastercard, bKash, Nagad, and 30+ payment methods' },
    { id: 'bkash', name: 'bKash', logo: '📱', connected: true, enabled: true, description: 'Mobile financial service with 65M+ users in Bangladesh' },
    { id: 'nagad', name: 'Nagad', logo: '💳', connected: false, enabled: false, description: 'Digital financial service by Bangladesh Post Office' },
  ]);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const subscriberCounts = tenants.reduce<Record<string, number>>((acc, t) => {
    acc[t.tier] = (acc[t.tier] || 0) + 1;
    return acc;
  }, {});

  // ---------------------------------------------------------------------------
  // Plan CRUD handlers
  // ---------------------------------------------------------------------------

  function openCreateModal() {
    setEditingPlan(null);
    setPlanForm({ ...emptyPlanForm, id: `plan_${Date.now()}`, tier: '' });
    setModalOpen(true);
  }

  function openEditModal(plan: Plan) {
    setEditingPlan(plan);
    setPlanForm({ ...plan, features: { ...plan.features }, limits: { ...plan.limits } });
    setModalOpen(true);
  }

  function savePlan() {
    if (!planForm.name.trim()) return;
    const finalPlan = { ...planForm, tier: planForm.tier || planForm.name.toLowerCase().replace(/\s+/g, '_') };
    if (editingPlan) {
      setPlans((prev) => prev.map((p) => (p.id === editingPlan.id ? finalPlan : p)));
    } else {
      setPlans((prev) => [...prev, finalPlan]);
    }
    setModalOpen(false);
  }

  function deletePlan(id: string) {
    setPlans((prev) => prev.filter((p) => p.id !== id));
    setDeleteConfirmId(null);
  }

  // ---------------------------------------------------------------------------
  // Billing computations
  // ---------------------------------------------------------------------------

  const totalRevenue = invoices
    .filter((inv) => inv.status === 'paid')
    .reduce((sum, inv) => sum + inv.amount, 0);

  const outstandingInvoices = invoices.filter(
    (inv) => inv.status === 'pending' || inv.status === 'overdue',
  );

  const outstandingAmount = outstandingInvoices.reduce((sum, inv) => sum + inv.amount, 0);

  const paidInvoices = invoices.filter((inv) => inv.status === 'paid' && inv.paidDate);
  const avgDaysToPay =
    paidInvoices.length > 0
      ? Math.round(
          paidInvoices.reduce((sum, inv) => {
            const due = new Date(inv.dueDate).getTime();
            const paid = new Date(inv.paidDate!).getTime();
            return sum + Math.max(0, Math.ceil((paid - due) / (1000 * 60 * 60 * 24)));
          }, 0) / paidInvoices.length,
        )
      : 0;

  const filteredInvoices =
    invoiceFilter === 'all' ? invoices : invoices.filter((inv) => inv.status === invoiceFilter);

  // ---------------------------------------------------------------------------
  // Revenue summary for plans tab
  // ---------------------------------------------------------------------------

  const plansWithSubs = plans.map((p) => ({
    ...p,
    subscribers: subscriberCounts[p.tier] || 0,
  }));

  const totalMRR = plansWithSubs.reduce((sum, p) => sum + p.monthlyPrice * p.subscribers, 0);

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Plans & Billing</h1>
        <p className="mt-1 text-sm text-gray-500">
          Manage subscription plans, payment gateways, and billing settings
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6">
          <button
            onClick={() => setActiveTab('plans')}
            className={cn(
              'whitespace-nowrap border-b-2 pb-3 text-sm font-medium transition-colors',
              activeTab === 'plans'
                ? 'border-indigo-600 text-indigo-600'
                : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
            )}
          >
            Plans
          </button>
          <button
            onClick={() => setActiveTab('billing')}
            className={cn(
              'whitespace-nowrap border-b-2 pb-3 text-sm font-medium transition-colors',
              activeTab === 'billing'
                ? 'border-indigo-600 text-indigo-600'
                : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
            )}
          >
            Billing & Invoices
          </button>
        </nav>
      </div>

      {/* ====================================================================
          PLANS TAB
      ==================================================================== */}
      {activeTab === 'plans' && (
        <div className="space-y-8">
          {/* Plans header */}
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Subscription Plans</h2>
              <p className="mt-0.5 text-sm text-gray-500">
                {plans.length} plan{plans.length !== 1 ? 's' : ''} configured
              </p>
            </div>
            <div className="flex items-center gap-4">
              {/* Yearly toggle */}
              <div className="flex items-center gap-2">
                <span className={cn('text-sm', !showYearly ? 'font-medium text-gray-900' : 'text-gray-500')}>
                  Monthly
                </span>
                <button
                  onClick={() => setShowYearly(!showYearly)}
                  className={cn(
                    'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors',
                    showYearly ? 'bg-indigo-600' : 'bg-gray-200',
                  )}
                >
                  <span
                    className={cn(
                      'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform',
                      showYearly ? 'translate-x-5' : 'translate-x-0',
                    )}
                  />
                </button>
                <span className={cn('text-sm', showYearly ? 'font-medium text-gray-900' : 'text-gray-500')}>
                  Yearly
                </span>
              </div>
              <button
                onClick={openCreateModal}
                className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-indigo-700"
              >
                <Plus className="h-4 w-4" />
                Create Plan
              </button>
            </div>
          </div>

          {/* Plan cards grid */}
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
            {plansWithSubs.map((plan) => {
              const displayPrice = showYearly ? plan.yearlyPrice : plan.monthlyPrice;
              const period = showYearly ? '/yr' : '/mo';
              return (
                <div
                  key={plan.id}
                  className={cn(
                    'relative flex flex-col rounded-xl border-2 bg-white p-6 shadow-sm transition-shadow hover:shadow-md',
                    plan.popular
                      ? 'border-indigo-400 ring-1 ring-indigo-400'
                      : 'border-gray-200',
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

                  {/* Plan name & description */}
                  <div className="mb-4">
                    <h3 className="text-lg font-semibold text-gray-900">{plan.name}</h3>
                    <p className="mt-1 text-sm text-gray-500">{plan.description}</p>
                  </div>

                  {/* Price */}
                  <div className="mb-5">
                    <span className="text-3xl font-bold text-gray-900">
                      {displayPrice === 0 ? 'Free' : formatCurrency(displayPrice)}
                    </span>
                    {displayPrice > 0 && (
                      <span className="text-sm text-gray-500">{period}</span>
                    )}
                    {showYearly && plan.yearlyPrice > 0 && plan.monthlyPrice > 0 && (
                      <p className="mt-1 text-xs text-green-600">
                        Save {formatCurrency(plan.monthlyPrice * 12 - plan.yearlyPrice)}/yr
                      </p>
                    )}
                  </div>

                  {/* Features */}
                  <ul className="mb-4 flex-1 space-y-2">
                    {featureMeta.map(({ key, label }) => {
                      const enabled = plan.features[key];
                      return (
                        <li key={key} className="flex items-center gap-2">
                          {enabled ? (
                            <Check className="h-4 w-4 shrink-0 text-indigo-600" />
                          ) : (
                            <X className="h-4 w-4 shrink-0 text-gray-300" />
                          )}
                          <span className={cn('text-sm', enabled ? 'text-gray-700' : 'text-gray-400')}>
                            {label}
                          </span>
                        </li>
                      );
                    })}
                  </ul>

                  {/* Limits */}
                  <div className="mb-4 space-y-1.5 rounded-lg bg-gray-50 p-3">
                    <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">Limits</p>
                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                      <span className="text-gray-500">Products</span>
                      <span className="text-right font-medium text-gray-900">
                        {formatLimit(plan.limits.maxProducts)}
                      </span>
                      <span className="text-gray-500">Storage</span>
                      <span className="text-right font-medium text-gray-900">
                        {formatLimit(plan.limits.maxStorageGB, 'GB')}
                      </span>
                      <span className="text-gray-500">Users</span>
                      <span className="text-right font-medium text-gray-900">
                        {formatLimit(plan.limits.maxUsers)}
                      </span>
                      <span className="text-gray-500">Orders/mo</span>
                      <span className="text-right font-medium text-gray-900">
                        {formatLimit(plan.limits.maxOrdersPerMonth)}
                      </span>
                    </div>
                  </div>

                  {/* Subscribers */}
                  <div className="border-t border-gray-100 pt-3">
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-gray-500">Active subscribers</span>
                      <span className="text-sm font-semibold text-gray-900">{plan.subscribers}</span>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="mt-4 flex gap-2">
                    <button
                      onClick={() => openEditModal(plan)}
                      className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                    >
                      <Pencil className="h-3.5 w-3.5" />
                      Edit
                    </button>
                    {deleteConfirmId === plan.id ? (
                      <div className="flex flex-1 gap-1">
                        <button
                          onClick={() => deletePlan(plan.id)}
                          className="flex-1 rounded-lg bg-red-600 px-2 py-2 text-xs font-medium text-white hover:bg-red-700"
                        >
                          Confirm
                        </button>
                        <button
                          onClick={() => setDeleteConfirmId(null)}
                          className="flex-1 rounded-lg border border-gray-200 px-2 py-2 text-xs font-medium text-gray-600 hover:bg-gray-50"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <button
                        onClick={() => setDeleteConfirmId(plan.id)}
                        className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-red-200 bg-white px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                        Delete
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {/* Revenue Summary Table */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-gray-900">Revenue Summary</h2>
            <div className="mt-4 overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                    <th className="pb-3 font-medium">Plan</th>
                    <th className="pb-3 font-medium">Price</th>
                    <th className="pb-3 font-medium">Subscribers</th>
                    <th className="pb-3 text-right font-medium">Monthly Revenue</th>
                  </tr>
                </thead>
                <tbody>
                  {plansWithSubs.map((plan) => (
                    <tr
                      key={plan.id}
                      className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                    >
                      <td className="py-3 text-sm font-medium text-gray-900">{plan.name}</td>
                      <td className="py-3 text-sm text-gray-500">
                        {plan.monthlyPrice === 0 ? 'Free' : formatCurrency(plan.monthlyPrice)}
                      </td>
                      <td className="py-3 text-sm text-gray-900">{plan.subscribers}</td>
                      <td className="py-3 text-right text-sm font-medium text-gray-900">
                        {formatCurrency(plan.monthlyPrice * plan.subscribers)}
                      </td>
                    </tr>
                  ))}
                  <tr className="bg-gray-50">
                    <td className="py-3 text-sm font-semibold text-gray-900" colSpan={3}>
                      Total MRR
                    </td>
                    <td className="py-3 text-right text-sm font-bold text-indigo-600">
                      {formatCurrency(totalMRR)}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* ====================================================================
          BILLING & INVOICES TAB
      ==================================================================== */}
      {activeTab === 'billing' && (
        <div className="space-y-8">
          {/* Billing stat cards */}
          <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
            <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-green-100 p-2.5">
                  <CreditCard className="h-5 w-5 text-green-600" />
                </div>
                <div>
                  <p className="text-sm text-gray-500">Total Revenue (YTD)</p>
                  <p className="text-2xl font-bold text-gray-900">{formatCurrency(totalRevenue)}</p>
                </div>
              </div>
            </div>
            <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-yellow-100 p-2.5">
                  <Receipt className="h-5 w-5 text-yellow-600" />
                </div>
                <div>
                  <p className="text-sm text-gray-500">Outstanding Invoices</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {outstandingInvoices.length}
                    <span className="ml-2 text-base font-normal text-gray-400">
                      ({formatCurrency(outstandingAmount)})
                    </span>
                  </p>
                </div>
              </div>
            </div>
            <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-blue-100 p-2.5">
                  <Clock className="h-5 w-5 text-blue-600" />
                </div>
                <div>
                  <p className="text-sm text-gray-500">Avg Days to Pay</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {avgDaysToPay}
                    <span className="ml-1 text-base font-normal text-gray-400">days</span>
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Payment Gateway Settings */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-5 flex items-center gap-2">
              <CreditCard className="h-5 w-5 text-gray-700" />
              <h2 className="text-lg font-semibold text-gray-900">Payment Gateway Settings</h2>
            </div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              {gateways.map((gw) => (
                <div
                  key={gw.id}
                  className={cn(
                    'rounded-lg border-2 p-5 transition-colors',
                    gw.connected ? 'border-gray-200' : 'border-dashed border-gray-300',
                  )}
                >
                  <div className="mb-3 flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-2xl">{gw.logo}</span>
                      <div>
                        <h3 className="font-semibold text-gray-900">{gw.name}</h3>
                        <span
                          className={cn(
                            'inline-block rounded-full px-2 py-0.5 text-xs font-medium',
                            gw.connected
                              ? 'bg-green-100 text-green-800'
                              : 'bg-gray-100 text-gray-500',
                          )}
                        >
                          {gw.connected ? 'Connected' : 'Disconnected'}
                        </span>
                      </div>
                    </div>
                    {gw.connected && (
                      <button
                        onClick={() =>
                          setGateways((prev) =>
                            prev.map((g) =>
                              g.id === gw.id ? { ...g, enabled: !g.enabled } : g,
                            ),
                          )
                        }
                        className={cn(
                          'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors',
                          gw.enabled ? 'bg-indigo-600' : 'bg-gray-200',
                        )}
                      >
                        <span
                          className={cn(
                            'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform',
                            gw.enabled ? 'translate-x-5' : 'translate-x-0',
                          )}
                        />
                      </button>
                    )}
                  </div>
                  <p className="mb-4 text-sm text-gray-500">{gw.description}</p>
                  <button
                    className={cn(
                      'w-full rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                      gw.connected
                        ? 'border border-gray-200 bg-white text-gray-700 hover:bg-gray-50'
                        : 'bg-indigo-600 text-white hover:bg-indigo-700',
                    )}
                  >
                    {gw.connected ? 'Configure' : 'Connect'}
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Invoices Table */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-2">
                <Receipt className="h-5 w-5 text-gray-700" />
                <h2 className="text-lg font-semibold text-gray-900">Invoices</h2>
                <span className="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">
                  {filteredInvoices.length}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Filter className="h-4 w-4 text-gray-400" />
                <div className="relative">
                  <select
                    value={invoiceFilter}
                    onChange={(e) => setInvoiceFilter(e.target.value as InvoiceStatus | 'all')}
                    className="appearance-none rounded-lg border border-gray-200 bg-white py-2 pl-3 pr-8 text-sm text-gray-700 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  >
                    <option value="all">All Statuses</option>
                    <option value="paid">Paid</option>
                    <option value="pending">Pending</option>
                    <option value="overdue">Overdue</option>
                    <option value="cancelled">Cancelled</option>
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                </div>
              </div>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                    <th className="pb-3 font-medium">Invoice #</th>
                    <th className="pb-3 font-medium">Tenant</th>
                    <th className="pb-3 font-medium">Plan</th>
                    <th className="pb-3 font-medium">Amount</th>
                    <th className="pb-3 font-medium">Status</th>
                    <th className="pb-3 font-medium">Due Date</th>
                    <th className="pb-3 font-medium">Paid Date</th>
                    <th className="pb-3 text-right font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredInvoices.map((inv) => (
                    <tr
                      key={inv.id}
                      className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                    >
                      <td className="py-3 text-sm font-medium text-gray-900">
                        {inv.invoiceNumber}
                      </td>
                      <td className="py-3 text-sm text-gray-700">{inv.tenantName}</td>
                      <td className="py-3 text-sm text-gray-500">{inv.plan}</td>
                      <td className="py-3 text-sm font-medium text-gray-900">
                        {formatCurrency(inv.amount)}
                      </td>
                      <td className="py-3">
                        <span
                          className={cn(
                            'inline-block rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                            statusBadgeColor[inv.status],
                          )}
                        >
                          {inv.status}
                        </span>
                      </td>
                      <td className="py-3 text-sm text-gray-500">{formatDate(inv.dueDate)}</td>
                      <td className="py-3 text-sm text-gray-500">
                        {inv.paidDate ? formatDate(inv.paidDate) : '--'}
                      </td>
                      <td className="py-3 text-right">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            className="rounded-md p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                            title="Download PDF"
                          >
                            <Download className="h-4 w-4" />
                          </button>
                          {(inv.status === 'pending' || inv.status === 'overdue') && (
                            <button
                              className="rounded-md p-1.5 text-gray-400 transition-colors hover:bg-yellow-50 hover:text-yellow-600"
                              title="Send Reminder"
                            >
                              <Bell className="h-4 w-4" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                  {filteredInvoices.length === 0 && (
                    <tr>
                      <td colSpan={8} className="py-12 text-center text-sm text-gray-400">
                        No invoices match the selected filter
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Billing Settings */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-5 flex items-center gap-2">
              <Settings className="h-5 w-5 text-gray-700" />
              <h2 className="text-lg font-semibold text-gray-900">Billing Settings</h2>
            </div>
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
              {/* Tax Rate */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Tax Rate (%)
                </label>
                <input
                  type="number"
                  min={0}
                  max={100}
                  step={0.5}
                  value={billingSettings.taxRate}
                  onChange={(e) =>
                    setBillingSettings((s) => ({ ...s, taxRate: parseFloat(e.target.value) || 0 }))
                  }
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
                <p className="mt-1 text-xs text-gray-400">VAT applied to all invoices</p>
              </div>

              {/* Invoice Prefix */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Invoice Prefix
                </label>
                <input
                  type="text"
                  value={billingSettings.invoicePrefix}
                  onChange={(e) =>
                    setBillingSettings((s) => ({ ...s, invoicePrefix: e.target.value }))
                  }
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
                <p className="mt-1 text-xs text-gray-400">e.g. INV-2026-001</p>
              </div>

              {/* Billing Cycle Day */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Billing Cycle Day
                </label>
                <input
                  type="number"
                  min={1}
                  max={28}
                  value={billingSettings.billingCycleDay}
                  onChange={(e) =>
                    setBillingSettings((s) => ({
                      ...s,
                      billingCycleDay: parseInt(e.target.value) || 1,
                    }))
                  }
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
                <p className="mt-1 text-xs text-gray-400">Day of month invoices are generated</p>
              </div>

              {/* Grace Period */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Grace Period (days)
                </label>
                <input
                  type="number"
                  min={0}
                  max={30}
                  value={billingSettings.gracePeriodDays}
                  onChange={(e) =>
                    setBillingSettings((s) => ({
                      ...s,
                      gracePeriodDays: parseInt(e.target.value) || 0,
                    }))
                  }
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
                <p className="mt-1 text-xs text-gray-400">Days after due date before marking overdue</p>
              </div>

              {/* Auto-suspend */}
              <div className="md:col-span-2 lg:col-span-2">
                <label className="mb-1.5 block text-sm font-medium text-gray-700">
                  Auto-Suspend Overdue Accounts
                </label>
                <div className="flex items-center gap-4">
                  <button
                    onClick={() =>
                      setBillingSettings((s) => ({
                        ...s,
                        autoSuspendEnabled: !s.autoSuspendEnabled,
                      }))
                    }
                    className={cn(
                      'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors',
                      billingSettings.autoSuspendEnabled ? 'bg-indigo-600' : 'bg-gray-200',
                    )}
                  >
                    <span
                      className={cn(
                        'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform',
                        billingSettings.autoSuspendEnabled ? 'translate-x-5' : 'translate-x-0',
                      )}
                    />
                  </button>
                  {billingSettings.autoSuspendEnabled && (
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-gray-500">Suspend after</span>
                      <input
                        type="number"
                        min={1}
                        max={90}
                        value={billingSettings.autoSuspendAfterDays}
                        onChange={(e) =>
                          setBillingSettings((s) => ({
                            ...s,
                            autoSuspendAfterDays: parseInt(e.target.value) || 14,
                          }))
                        }
                        className="w-20 rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                      <span className="text-sm text-gray-500">days overdue</span>
                    </div>
                  )}
                </div>
                <p className="mt-1.5 flex items-center gap-1.5 text-xs text-gray-400">
                  <AlertTriangle className="h-3.5 w-3.5 text-yellow-500" />
                  Suspended tenants lose access to their storefront until payment is made
                </p>
              </div>
            </div>

            <div className="mt-6 flex justify-end border-t border-gray-100 pt-4">
              <button className="rounded-lg bg-indigo-600 px-5 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-indigo-700">
                Save Settings
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ====================================================================
          CREATE / EDIT PLAN MODAL
      ==================================================================== */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/50 p-4 pt-[5vh]">
          <div
            className="relative w-full max-w-2xl rounded-xl bg-white shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal header */}
            <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">
                {editingPlan ? 'Edit Plan' : 'Create Plan'}
              </h2>
              <button
                onClick={() => setModalOpen(false)}
                className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {/* Modal body */}
            <div className="max-h-[70vh] overflow-y-auto px-6 py-5">
              <div className="space-y-6">
                {/* Basic info */}
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div className="sm:col-span-2">
                    <label className="mb-1.5 block text-sm font-medium text-gray-700">
                      Plan Name
                    </label>
                    <input
                      type="text"
                      value={planForm.name}
                      onChange={(e) => setPlanForm((f) => ({ ...f, name: e.target.value }))}
                      placeholder="e.g. Professional"
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div className="sm:col-span-2">
                    <label className="mb-1.5 block text-sm font-medium text-gray-700">
                      Description
                    </label>
                    <textarea
                      value={planForm.description}
                      onChange={(e) => setPlanForm((f) => ({ ...f, description: e.target.value }))}
                      rows={2}
                      placeholder="Brief description of this plan"
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1.5 block text-sm font-medium text-gray-700">
                      Monthly Price (BDT)
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={planForm.monthlyPrice}
                      onChange={(e) =>
                        setPlanForm((f) => ({ ...f, monthlyPrice: parseInt(e.target.value) || 0 }))
                      }
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1.5 block text-sm font-medium text-gray-700">
                      Yearly Price (BDT)
                    </label>
                    <input
                      type="number"
                      min={0}
                      value={planForm.yearlyPrice}
                      onChange={(e) =>
                        setPlanForm((f) => ({ ...f, yearlyPrice: parseInt(e.target.value) || 0 }))
                      }
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                </div>

                {/* Feature toggles */}
                <div>
                  <p className="mb-3 text-sm font-semibold text-gray-900">Features</p>
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    {featureMeta.map(({ key, label, icon: Icon }) => (
                      <label
                        key={key}
                        className={cn(
                          'flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors',
                          planForm.features[key]
                            ? 'border-indigo-200 bg-indigo-50'
                            : 'border-gray-200 bg-white hover:bg-gray-50',
                        )}
                      >
                        <input
                          type="checkbox"
                          checked={planForm.features[key]}
                          onChange={() =>
                            setPlanForm((f) => ({
                              ...f,
                              features: { ...f.features, [key]: !f.features[key] },
                            }))
                          }
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                        />
                        <Icon className="h-4 w-4 text-gray-500" />
                        <span className="text-sm text-gray-700">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* Limits */}
                <div>
                  <p className="mb-3 text-sm font-semibold text-gray-900">
                    Limits{' '}
                    <span className="font-normal text-gray-400">(-1 for unlimited)</span>
                  </p>
                  <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">
                        Max Products
                      </label>
                      <input
                        type="number"
                        min={-1}
                        value={planForm.limits.maxProducts}
                        onChange={(e) =>
                          setPlanForm((f) => ({
                            ...f,
                            limits: { ...f.limits, maxProducts: parseInt(e.target.value) || 0 },
                          }))
                        }
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">
                        Storage (GB)
                      </label>
                      <input
                        type="number"
                        min={-1}
                        step={0.1}
                        value={planForm.limits.maxStorageGB}
                        onChange={(e) =>
                          setPlanForm((f) => ({
                            ...f,
                            limits: {
                              ...f.limits,
                              maxStorageGB: parseFloat(e.target.value) || 0,
                            },
                          }))
                        }
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">
                        Team Members
                      </label>
                      <input
                        type="number"
                        min={-1}
                        value={planForm.limits.maxUsers}
                        onChange={(e) =>
                          setPlanForm((f) => ({
                            ...f,
                            limits: { ...f.limits, maxUsers: parseInt(e.target.value) || 0 },
                          }))
                        }
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">
                        Orders/Month
                      </label>
                      <input
                        type="number"
                        min={-1}
                        value={planForm.limits.maxOrdersPerMonth}
                        onChange={(e) =>
                          setPlanForm((f) => ({
                            ...f,
                            limits: {
                              ...f.limits,
                              maxOrdersPerMonth: parseInt(e.target.value) || 0,
                            },
                          }))
                        }
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                    </div>
                  </div>
                </div>

                {/* Trial days */}
                <div className="max-w-[200px]">
                  <label className="mb-1.5 block text-sm font-medium text-gray-700">
                    Trial Days
                  </label>
                  <input
                    type="number"
                    min={0}
                    max={90}
                    value={planForm.trialDays}
                    onChange={(e) =>
                      setPlanForm((f) => ({ ...f, trialDays: parseInt(e.target.value) || 0 }))
                    }
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
              </div>
            </div>

            {/* Modal footer */}
            <div className="flex items-center justify-end gap-3 border-t border-gray-200 px-6 py-4">
              <button
                onClick={() => setModalOpen(false)}
                className="rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={savePlan}
                disabled={!planForm.name.trim()}
                className="rounded-lg bg-indigo-600 px-5 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {editingPlan ? 'Save Changes' : 'Create Plan'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
