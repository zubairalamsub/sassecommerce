'use client';

import { useState } from 'react';
import {
  Search,
  Plus,
  X,
  Copy,
  Eye,
  EyeOff,
  Trash2,
  Pencil,
  Check,
  RefreshCw,
  Key,
  Webhook,
  Blocks,
  ExternalLink,
  Shield,
  Zap,
  ChevronDown,
  AlertCircle,
} from 'lucide-react';
import { cn } from '@/lib/utils';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

type TabKey = 'library' | 'apikeys' | 'webhooks';

type IntegrationStatus = 'connected' | 'available' | 'coming_soon';
type IntegrationCategory = 'Payment' | 'Analytics' | 'Marketing' | 'Shipping' | 'Communication' | 'CRM';

interface Integration {
  id: string;
  name: string;
  category: IntegrationCategory;
  description: string;
  status: IntegrationStatus;
  color: string;
  apiKey?: string;
  secretKey?: string;
  webhookUrl?: string;
  enabledForAll?: boolean;
}

interface ApiKey {
  id: string;
  name: string;
  key: string;
  created: string;
  lastUsed: string;
  status: 'active' | 'revoked';
  scopes: string[];
}

interface WebhookEntry {
  id: string;
  url: string;
  events: string[];
  status: 'active' | 'inactive';
  successRate: number;
  lastTriggered: string;
  secret: string;
}

/* ------------------------------------------------------------------ */
/*  Demo Data                                                          */
/* ------------------------------------------------------------------ */

const initialIntegrations: Integration[] = [
  // Payment
  { id: 'sslcommerz', name: 'SSLCommerz', category: 'Payment', description: 'Accept payments via SSLCommerz payment gateway with support for cards, mobile banking, and internet banking in Bangladesh.', status: 'connected', color: 'bg-green-500', apiKey: 'ssl_live_abc123xxxxxxxx', secretKey: 'ssl_sec_xxxxxxxxxxxxxxxx', webhookUrl: 'https://api.platform.com/webhooks/sslcommerz', enabledForAll: true },
  { id: 'bkash', name: 'bKash', category: 'Payment', description: 'Enable bKash mobile wallet payments for seamless checkout experience across Bangladesh.', status: 'connected', color: 'bg-pink-500', apiKey: 'bkash_live_def456xxxxxxxx', secretKey: 'bkash_sec_xxxxxxxxxxxxxxxx', webhookUrl: 'https://api.platform.com/webhooks/bkash', enabledForAll: true },
  { id: 'nagad', name: 'Nagad', category: 'Payment', description: 'Integrate Nagad digital financial service for mobile-based payments in Bangladesh.', status: 'available', color: 'bg-orange-500' },
  { id: 'stripe', name: 'Stripe', category: 'Payment', description: 'Full-featured payment processing with Stripe for international cards, subscriptions, and payouts.', status: 'connected', color: 'bg-indigo-500', apiKey: 'sk_live_xxxxxxxxxxxxxxxx', secretKey: 'whsec_xxxxxxxxxxxxxxxx', webhookUrl: 'https://api.platform.com/webhooks/stripe', enabledForAll: false },
  { id: 'paypal', name: 'PayPal', category: 'Payment', description: 'Accept PayPal payments including credit cards, debit cards, and PayPal balance globally.', status: 'available', color: 'bg-blue-600' },

  // Analytics
  { id: 'google-analytics', name: 'Google Analytics', category: 'Analytics', description: 'Track website traffic, user behavior, and conversions with Google Analytics 4 integration.', status: 'connected', color: 'bg-yellow-500', apiKey: 'G-XXXXXXXXXX', secretKey: '', webhookUrl: '', enabledForAll: true },
  { id: 'facebook-pixel', name: 'Facebook Pixel', category: 'Analytics', description: 'Measure ad effectiveness and track conversions from Facebook and Instagram ads.', status: 'available', color: 'bg-blue-500' },
  { id: 'mixpanel', name: 'Mixpanel', category: 'Analytics', description: 'Advanced product analytics to understand user engagement, retention, and conversion funnels.', status: 'coming_soon', color: 'bg-purple-500' },

  // Marketing
  { id: 'mailchimp', name: 'Mailchimp', category: 'Marketing', description: 'Automated email marketing campaigns, audience segmentation, and newsletter management.', status: 'available', color: 'bg-yellow-600' },
  { id: 'sendgrid', name: 'SendGrid', category: 'Marketing', description: 'Transactional and marketing email delivery with high deliverability and analytics.', status: 'connected', color: 'bg-blue-400', apiKey: 'SG.xxxxxxxxxxxxxxxx', secretKey: '', webhookUrl: 'https://api.platform.com/webhooks/sendgrid', enabledForAll: true },
  { id: 'facebook-ads', name: 'Facebook Ads', category: 'Marketing', description: 'Sync product catalog and audiences with Facebook Ads for dynamic retargeting campaigns.', status: 'coming_soon', color: 'bg-blue-700' },

  // Shipping
  { id: 'pathao', name: 'Pathao Courier', category: 'Shipping', description: 'Integrate with Pathao Courier for last-mile delivery, COD collection, and real-time tracking in Bangladesh.', status: 'connected', color: 'bg-red-500', apiKey: 'pathao_live_xxxxxxxx', secretKey: 'pathao_sec_xxxxxxxx', webhookUrl: 'https://api.platform.com/webhooks/pathao', enabledForAll: true },
  { id: 'steadfast', name: 'Steadfast', category: 'Shipping', description: 'Reliable parcel delivery service with COD support across Bangladesh via Steadfast Courier.', status: 'available', color: 'bg-teal-500' },
  { id: 'redx', name: 'RedX', category: 'Shipping', description: 'Fast e-commerce delivery and fulfillment with RedX logistics network in Bangladesh.', status: 'available', color: 'bg-red-600' },
  { id: 'paperfly', name: 'Paperfly', category: 'Shipping', description: 'Nationwide delivery coverage with Paperfly for e-commerce order fulfillment in Bangladesh.', status: 'coming_soon', color: 'bg-amber-600' },

  // Communication
  { id: 'twilio', name: 'Twilio SMS', category: 'Communication', description: 'Send transactional SMS, OTP verification, and promotional messages via Twilio.', status: 'available', color: 'bg-red-400' },
  { id: 'firebase-push', name: 'Firebase Push', category: 'Communication', description: 'Send push notifications to web and mobile apps using Firebase Cloud Messaging.', status: 'connected', color: 'bg-amber-500', apiKey: 'firebase_key_xxxxxxxx', secretKey: '', webhookUrl: '', enabledForAll: true },

  // CRM
  { id: 'hubspot', name: 'HubSpot', category: 'CRM', description: 'Sync customer data, track interactions, and automate marketing workflows with HubSpot CRM.', status: 'coming_soon', color: 'bg-orange-600' },
];

const initialApiKeys: ApiKey[] = [
  { id: 'ak-1', name: 'Production API', key: 'pk_live_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6', created: '2026-01-15', lastUsed: '2026-04-24', status: 'active', scopes: ['tenants:read', 'tenants:write', 'orders:read', 'analytics:read'] },
  { id: 'ak-2', name: 'Analytics Dashboard', key: 'pk_live_q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2', created: '2026-02-03', lastUsed: '2026-04-25', status: 'active', scopes: ['analytics:read', 'orders:read'] },
  { id: 'ak-3', name: 'Mobile App Backend', key: 'pk_live_g3h4i5j6k7l8m9n0o1p2q3r4s5t6u7v8', created: '2026-02-20', lastUsed: '2026-04-23', status: 'active', scopes: ['tenants:read', 'users:read', 'orders:read'] },
  { id: 'ak-4', name: 'Legacy Integration', key: 'pk_live_w9x0y1z2a3b4c5d6e7f8g9h0i1j2k3l4', created: '2025-11-10', lastUsed: '2026-03-01', status: 'revoked', scopes: ['tenants:read', 'users:read', 'users:write'] },
  { id: 'ak-5', name: 'CI/CD Pipeline', key: 'pk_live_m5n6o7p8q9r0s1t2u3v4w5x6y7z8a9b0', created: '2026-03-12', lastUsed: '2026-04-25', status: 'active', scopes: ['config:write', 'tenants:read'] },
];

const initialWebhooks: WebhookEntry[] = [
  { id: 'wh-1', url: 'https://analytics.internal.com/events', events: ['order.created', 'order.completed', 'payment.received'], status: 'active', successRate: 99.2, lastTriggered: '2026-04-25T10:32:00Z', secret: 'whsec_a1b2c3d4e5f6g7h8i9j0' },
  { id: 'wh-2', url: 'https://crm.example.com/api/hooks', events: ['tenant.created', 'user.registered'], status: 'active', successRate: 97.8, lastTriggered: '2026-04-24T18:15:00Z', secret: 'whsec_k1l2m3n4o5p6q7r8s9t0' },
  { id: 'wh-3', url: 'https://slack-bot.internal.com/notify', events: ['tenant.created', 'tenant.deleted', 'order.completed'], status: 'active', successRate: 100, lastTriggered: '2026-04-25T09:45:00Z', secret: 'whsec_u1v2w3x4y5z6a7b8c9d0' },
  { id: 'wh-4', url: 'https://legacy-system.example.com/webhook', events: ['order.created', 'payment.received'], status: 'inactive', successRate: 85.4, lastTriggered: '2026-03-18T14:22:00Z', secret: 'whsec_e1f2g3h4i5j6k7l8m9n0' },
];

const allCategories: ('All' | IntegrationCategory)[] = ['All', 'Payment', 'Analytics', 'Marketing', 'Shipping', 'Communication', 'CRM'];

const categoryColors: Record<IntegrationCategory, string> = {
  Payment: 'bg-green-100 text-green-800',
  Analytics: 'bg-blue-100 text-blue-800',
  Marketing: 'bg-purple-100 text-purple-800',
  Shipping: 'bg-amber-100 text-amber-800',
  Communication: 'bg-pink-100 text-pink-800',
  CRM: 'bg-orange-100 text-orange-800',
};

const allScopes = [
  'tenants:read',
  'tenants:write',
  'users:read',
  'users:write',
  'orders:read',
  'analytics:read',
  'config:write',
];

const allEvents = [
  'tenant.created',
  'tenant.updated',
  'tenant.deleted',
  'order.created',
  'order.completed',
  'payment.received',
  'user.registered',
];

/* ------------------------------------------------------------------ */
/*  Helpers                                                            */
/* ------------------------------------------------------------------ */

function maskKey(key: string): string {
  if (key.length <= 12) return key;
  return key.slice(0, 7) + '...' + key.slice(-4);
}

function generateId(): string {
  return Math.random().toString(36).slice(2, 10);
}

function generateKey(): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = 'pk_live_';
  for (let i = 0; i < 32; i++) result += chars[Math.floor(Math.random() * chars.length)];
  return result;
}

function generateSecret(): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = 'whsec_';
  for (let i = 0; i < 20; i++) result += chars[Math.floor(Math.random() * chars.length)];
  return result;
}

function formatDateTime(date: string): string {
  return new Date(date).toLocaleString('en-BD', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export default function IntegrationsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>('library');

  /* --- Integrations Library State --- */
  const [integrations, setIntegrations] = useState<Integration[]>(initialIntegrations);
  const [categoryFilter, setCategoryFilter] = useState<'All' | IntegrationCategory>('All');
  const [integrationSearch, setIntegrationSearch] = useState('');
  const [configuring, setConfiguring] = useState<Integration | null>(null);
  const [configForm, setConfigForm] = useState({ apiKey: '', secretKey: '', enabledForAll: true });
  const [testResult, setTestResult] = useState<'idle' | 'testing' | 'success' | 'error'>('idle');

  /* --- API Keys State --- */
  const [apiKeys, setApiKeys] = useState<ApiKey[]>(initialApiKeys);
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set());
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [newKeyForm, setNewKeyForm] = useState({ name: '', description: '', scopes: [] as string[], expiry: '90d' });
  const [generatedKey, setGeneratedKey] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  /* --- Webhooks State --- */
  const [webhooks, setWebhooks] = useState<WebhookEntry[]>(initialWebhooks);
  const [showWebhookModal, setShowWebhookModal] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState<WebhookEntry | null>(null);
  const [webhookForm, setWebhookForm] = useState({ url: '', events: [] as string[], active: true, secret: generateSecret() });
  const [webhookTestId, setWebhookTestId] = useState<string | null>(null);

  /* --- Integrations Library Logic --- */
  const filteredIntegrations = integrations.filter((i) => {
    if (categoryFilter !== 'All' && i.category !== categoryFilter) return false;
    if (integrationSearch) {
      const q = integrationSearch.toLowerCase();
      return i.name.toLowerCase().includes(q) || i.description.toLowerCase().includes(q) || i.category.toLowerCase().includes(q);
    }
    return true;
  });

  function openConfigModal(integration: Integration) {
    setConfiguring(integration);
    setConfigForm({
      apiKey: integration.apiKey || '',
      secretKey: integration.secretKey || '',
      enabledForAll: integration.enabledForAll ?? true,
    });
    setTestResult('idle');
  }

  function handleEnableIntegration(integration: Integration) {
    setConfiguring({ ...integration, status: 'connected' });
    setConfigForm({ apiKey: '', secretKey: '', enabledForAll: true });
    setTestResult('idle');
  }

  function saveConfig() {
    if (!configuring) return;
    setIntegrations((prev) =>
      prev.map((i) =>
        i.id === configuring.id
          ? { ...i, status: 'connected' as IntegrationStatus, apiKey: configForm.apiKey, secretKey: configForm.secretKey, enabledForAll: configForm.enabledForAll, webhookUrl: i.webhookUrl || `https://api.platform.com/webhooks/${i.id}` }
          : i,
      ),
    );
    setConfiguring(null);
  }

  function handleTestConnection() {
    setTestResult('testing');
    setTimeout(() => {
      setTestResult(configForm.apiKey.length > 3 ? 'success' : 'error');
    }, 1500);
  }

  /* --- API Keys Logic --- */
  function toggleKeyVisibility(id: string) {
    setVisibleKeys((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function copyToClipboard(text: string, id: string) {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  }

  function revokeKey(id: string) {
    setApiKeys((prev) => prev.map((k) => (k.id === id ? { ...k, status: 'revoked' as const } : k)));
  }

  function deleteKey(id: string) {
    setApiKeys((prev) => prev.filter((k) => k.id !== id));
  }

  function openGenerateModal() {
    setNewKeyForm({ name: '', description: '', scopes: [], expiry: '90d' });
    setGeneratedKey(null);
    setShowGenerateModal(true);
  }

  function toggleScope(scope: string) {
    setNewKeyForm((prev) => ({
      ...prev,
      scopes: prev.scopes.includes(scope) ? prev.scopes.filter((s) => s !== scope) : [...prev.scopes, scope],
    }));
  }

  function handleGenerateKey() {
    const key = generateKey();
    setGeneratedKey(key);
    const newKey: ApiKey = {
      id: 'ak-' + generateId(),
      name: newKeyForm.name,
      key,
      created: new Date().toISOString().split('T')[0],
      lastUsed: 'Never',
      status: 'active',
      scopes: newKeyForm.scopes,
    };
    setApiKeys((prev) => [newKey, ...prev]);
  }

  /* --- Webhooks Logic --- */
  function openAddWebhookModal() {
    setEditingWebhook(null);
    setWebhookForm({ url: '', events: [], active: true, secret: generateSecret() });
    setShowWebhookModal(true);
  }

  function openEditWebhookModal(wh: WebhookEntry) {
    setEditingWebhook(wh);
    setWebhookForm({ url: wh.url, events: [...wh.events], active: wh.status === 'active', secret: wh.secret });
    setShowWebhookModal(true);
  }

  function toggleEvent(event: string) {
    setWebhookForm((prev) => ({
      ...prev,
      events: prev.events.includes(event) ? prev.events.filter((e) => e !== event) : [...prev.events, event],
    }));
  }

  function saveWebhook() {
    if (editingWebhook) {
      setWebhooks((prev) =>
        prev.map((w) =>
          w.id === editingWebhook.id
            ? { ...w, url: webhookForm.url, events: webhookForm.events, status: webhookForm.active ? 'active' : 'inactive', secret: webhookForm.secret }
            : w,
        ),
      );
    } else {
      const newWh: WebhookEntry = {
        id: 'wh-' + generateId(),
        url: webhookForm.url,
        events: webhookForm.events,
        status: webhookForm.active ? 'active' : 'inactive',
        successRate: 100,
        lastTriggered: 'Never',
        secret: webhookForm.secret,
      };
      setWebhooks((prev) => [newWh, ...prev]);
    }
    setShowWebhookModal(false);
    setEditingWebhook(null);
  }

  function deleteWebhook(id: string) {
    setWebhooks((prev) => prev.filter((w) => w.id !== id));
  }

  function testWebhook(id: string) {
    setWebhookTestId(id);
    setTimeout(() => setWebhookTestId(null), 2000);
  }

  /* --- Tab definitions --- */
  const tabs: { key: TabKey; label: string; icon: typeof Blocks }[] = [
    { key: 'library', label: 'Integrations Library', icon: Blocks },
    { key: 'apikeys', label: 'API Keys', icon: Key },
    { key: 'webhooks', label: 'Webhooks', icon: Webhook },
  ];

  return (
    <div className="space-y-6">
      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  'flex items-center gap-2 border-b-2 px-1 pb-3 text-sm font-medium transition-colors',
                  activeTab === tab.key
                    ? 'border-indigo-600 text-indigo-600'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
                )}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      {/* ============================================================ */}
      {/*  TAB 1 — Integrations Library                                 */}
      {/* ============================================================ */}
      {activeTab === 'library' && (
        <div className="space-y-6">
          {/* Header */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Platform Integrations</h1>
              <p className="mt-1 text-sm text-gray-500">
                Connect third-party services to extend platform capabilities for all tenants.
              </p>
            </div>
            <div className="relative min-w-[260px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                value={integrationSearch}
                onChange={(e) => setIntegrationSearch(e.target.value)}
                placeholder="Search integrations..."
                className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
          </div>

          {/* Category Filters */}
          <div className="flex flex-wrap gap-2">
            {allCategories.map((cat) => (
              <button
                key={cat}
                onClick={() => setCategoryFilter(cat)}
                className={cn(
                  'rounded-full px-4 py-1.5 text-sm font-medium transition-colors',
                  categoryFilter === cat
                    ? 'bg-indigo-600 text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200',
                )}
              >
                {cat}
              </button>
            ))}
          </div>

          {/* Integration Cards Grid */}
          {filteredIntegrations.length === 0 ? (
            <div className="py-16 text-center text-sm text-gray-400">No integrations match your search.</div>
          ) : (
            <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
              {filteredIntegrations.map((integration) => (
                <div
                  key={integration.id}
                  className="flex flex-col rounded-xl border border-gray-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md"
                >
                  <div className="flex items-start gap-4">
                    {/* Icon */}
                    <div className={cn('flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-white font-bold text-lg', integration.color)}>
                      {integration.name[0]}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="text-sm font-semibold text-gray-900 truncate">{integration.name}</h3>
                        <span className={cn('inline-flex shrink-0 rounded-full px-2 py-0.5 text-xs font-medium', categoryColors[integration.category])}>
                          {integration.category}
                        </span>
                      </div>
                      <p className="mt-1 text-xs text-gray-500 line-clamp-2">{integration.description}</p>
                    </div>
                  </div>

                  <div className="mt-4 flex items-center justify-between border-t border-gray-100 pt-4">
                    {/* Status */}
                    <span
                      className={cn(
                        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium',
                        integration.status === 'connected' && 'bg-green-100 text-green-800',
                        integration.status === 'available' && 'bg-blue-100 text-blue-800',
                        integration.status === 'coming_soon' && 'bg-gray-100 text-gray-500',
                      )}
                    >
                      {integration.status === 'connected' && <span className="h-1.5 w-1.5 rounded-full bg-green-500" />}
                      {integration.status === 'connected' ? 'Connected' : integration.status === 'available' ? 'Available' : 'Coming Soon'}
                    </span>

                    {/* Action */}
                    {integration.status === 'connected' && (
                      <button
                        onClick={() => openConfigModal(integration)}
                        className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-50 px-3 py-1.5 text-xs font-medium text-indigo-700 transition-colors hover:bg-indigo-100"
                      >
                        <Pencil className="h-3 w-3" />
                        Configure
                      </button>
                    )}
                    {integration.status === 'available' && (
                      <button
                        onClick={() => handleEnableIntegration(integration)}
                        className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-indigo-700"
                      >
                        <Zap className="h-3 w-3" />
                        Enable
                      </button>
                    )}
                    {integration.status === 'coming_soon' && (
                      <span className="text-xs text-gray-400">Coming Soon</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Configure Modal */}
          {configuring && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
              <div className="w-full max-w-lg rounded-2xl border border-gray-200 bg-white shadow-xl">
                <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
                  <div className="flex items-center gap-3">
                    <div className={cn('flex h-9 w-9 items-center justify-center rounded-full text-white font-bold', configuring.color)}>
                      {configuring.name[0]}
                    </div>
                    <div>
                      <h3 className="text-base font-semibold text-gray-900">{configuring.name}</h3>
                      <p className="text-xs text-gray-500">{configuring.category} Integration</p>
                    </div>
                  </div>
                  <button onClick={() => setConfiguring(null)} className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors">
                    <X className="h-5 w-5" />
                  </button>
                </div>

                <div className="space-y-4 px-6 py-5">
                  <p className="text-sm text-gray-500">{configuring.description}</p>

                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">API Key</label>
                    <input
                      type="text"
                      value={configForm.apiKey}
                      onChange={(e) => setConfigForm((f) => ({ ...f, apiKey: e.target.value }))}
                      placeholder="Enter API key"
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Secret Key</label>
                    <input
                      type="password"
                      value={configForm.secretKey}
                      onChange={(e) => setConfigForm((f) => ({ ...f, secretKey: e.target.value }))}
                      placeholder="Enter secret key"
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Webhook URL</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        readOnly
                        value={configuring.webhookUrl || `https://api.platform.com/webhooks/${configuring.id}`}
                        className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm font-mono text-gray-500 focus:outline-none"
                      />
                      <button
                        onClick={() => copyToClipboard(configuring.webhookUrl || `https://api.platform.com/webhooks/${configuring.id}`, 'webhook-url')}
                        className="shrink-0 rounded-lg border border-gray-200 p-2 text-gray-400 hover:bg-gray-50 hover:text-gray-600 transition-colors"
                      >
                        {copiedId === 'webhook-url' ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>

                  {/* Tenant access toggle */}
                  <div className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3">
                    <div>
                      <p className="text-sm font-medium text-gray-700">Enable for all tenants</p>
                      <p className="text-xs text-gray-500">Toggle off to restrict to specific tenants only</p>
                    </div>
                    <button
                      onClick={() => setConfigForm((f) => ({ ...f, enabledForAll: !f.enabledForAll }))}
                      className={cn(
                        'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full transition-colors',
                        configForm.enabledForAll ? 'bg-indigo-600' : 'bg-gray-200',
                      )}
                    >
                      <span
                        className={cn(
                          'inline-block h-5 w-5 rounded-full bg-white shadow-sm transition-transform mt-0.5',
                          configForm.enabledForAll ? 'translate-x-5 ml-0.5' : 'translate-x-0.5',
                        )}
                      />
                    </button>
                  </div>

                  {/* Test connection */}
                  <button
                    onClick={handleTestConnection}
                    disabled={testResult === 'testing'}
                    className={cn(
                      'inline-flex w-full items-center justify-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors',
                      testResult === 'success'
                        ? 'border-green-200 bg-green-50 text-green-700'
                        : testResult === 'error'
                          ? 'border-red-200 bg-red-50 text-red-700'
                          : 'border-gray-200 text-gray-700 hover:bg-gray-50',
                    )}
                  >
                    {testResult === 'testing' && <RefreshCw className="h-4 w-4 animate-spin" />}
                    {testResult === 'success' && <Check className="h-4 w-4" />}
                    {testResult === 'error' && <AlertCircle className="h-4 w-4" />}
                    {testResult === 'idle' && <Zap className="h-4 w-4" />}
                    {testResult === 'testing' ? 'Testing...' : testResult === 'success' ? 'Connection Successful' : testResult === 'error' ? 'Connection Failed' : 'Test Connection'}
                  </button>
                </div>

                <div className="flex gap-3 border-t border-gray-200 px-6 py-4">
                  <button
                    onClick={saveConfig}
                    className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                  >
                    Save Configuration
                  </button>
                  <button
                    onClick={() => setConfiguring(null)}
                    className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ============================================================ */}
      {/*  TAB 2 — API Keys                                            */}
      {/* ============================================================ */}
      {activeTab === 'apikeys' && (
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
              <p className="mt-1 text-sm text-gray-500">Manage API keys for programmatic access to the platform.</p>
            </div>
            <button
              onClick={openGenerateModal}
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              <Plus className="h-4 w-4" />
              Generate API Key
            </button>
          </div>

          {/* API Keys Table */}
          <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
            {apiKeys.length === 0 ? (
              <div className="py-16 text-center text-sm text-gray-400">No API keys created yet.</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                      <th className="px-6 py-3 font-medium">Name</th>
                      <th className="px-6 py-3 font-medium">Key</th>
                      <th className="px-6 py-3 font-medium">Created</th>
                      <th className="px-6 py-3 font-medium">Last Used</th>
                      <th className="px-6 py-3 font-medium">Status</th>
                      <th className="px-6 py-3 font-medium">Scopes</th>
                      <th className="px-6 py-3 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {apiKeys.map((ak) => (
                      <tr key={ak.id} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                        <td className="px-6 py-4 text-sm font-medium text-gray-900">{ak.name}</td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <code className="text-xs font-mono text-gray-600">
                              {visibleKeys.has(ak.id) ? ak.key : maskKey(ak.key)}
                            </code>
                            <button
                              onClick={() => toggleKeyVisibility(ak.id)}
                              className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
                              title={visibleKeys.has(ak.id) ? 'Hide' : 'Show'}
                            >
                              {visibleKeys.has(ak.id) ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                            </button>
                          </div>
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-500">{ak.created}</td>
                        <td className="px-6 py-4 text-sm text-gray-500">{ak.lastUsed}</td>
                        <td className="px-6 py-4">
                          <span
                            className={cn(
                              'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                              ak.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800',
                            )}
                          >
                            {ak.status}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex flex-wrap gap-1">
                            {ak.scopes.slice(0, 2).map((scope) => (
                              <span key={scope} className="inline-flex rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600">
                                {scope}
                              </span>
                            ))}
                            {ak.scopes.length > 2 && (
                              <span className="inline-flex rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600">
                                +{ak.scopes.length - 2}
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-1.5">
                            <button
                              onClick={() => copyToClipboard(ak.key, ak.id)}
                              className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
                              title="Copy key"
                            >
                              {copiedId === ak.id ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                            </button>
                            {ak.status === 'active' && (
                              <button
                                onClick={() => revokeKey(ak.id)}
                                className="rounded-lg p-1.5 text-gray-400 hover:bg-amber-50 hover:text-amber-600 transition-colors"
                                title="Revoke"
                              >
                                <Shield className="h-4 w-4" />
                              </button>
                            )}
                            <button
                              onClick={() => deleteKey(ak.id)}
                              className="rounded-lg p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors"
                              title="Delete"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Generate API Key Modal */}
          {showGenerateModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
              <div className="w-full max-w-lg rounded-2xl border border-gray-200 bg-white shadow-xl">
                <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
                  <h3 className="text-base font-semibold text-gray-900">Generate API Key</h3>
                  <button onClick={() => setShowGenerateModal(false)} className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors">
                    <X className="h-5 w-5" />
                  </button>
                </div>

                {generatedKey ? (
                  <div className="space-y-4 px-6 py-5">
                    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4">
                      <div className="flex items-start gap-3">
                        <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
                        <div>
                          <p className="text-sm font-medium text-amber-800">Copy your API key now</p>
                          <p className="mt-1 text-xs text-amber-700">
                            This is the only time you will see this key. Store it securely.
                          </p>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3">
                      <code className="flex-1 break-all text-sm font-mono text-gray-800">{generatedKey}</code>
                      <button
                        onClick={() => copyToClipboard(generatedKey, 'generated')}
                        className="shrink-0 rounded-lg border border-gray-200 bg-white p-2 text-gray-400 hover:text-gray-600 transition-colors"
                      >
                        {copiedId === 'generated' ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                      </button>
                    </div>
                    <button
                      onClick={() => setShowGenerateModal(false)}
                      className="w-full rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                    >
                      Done
                    </button>
                  </div>
                ) : (
                  <div className="space-y-4 px-6 py-5">
                    <div>
                      <label className="mb-1 block text-sm font-medium text-gray-700">Key Name</label>
                      <input
                        type="text"
                        value={newKeyForm.name}
                        onChange={(e) => setNewKeyForm((f) => ({ ...f, name: e.target.value }))}
                        placeholder="e.g., Production API Key"
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                    </div>

                    <div>
                      <label className="mb-1 block text-sm font-medium text-gray-700">Description</label>
                      <input
                        type="text"
                        value={newKeyForm.description}
                        onChange={(e) => setNewKeyForm((f) => ({ ...f, description: e.target.value }))}
                        placeholder="What will this key be used for?"
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                    </div>

                    <div>
                      <label className="mb-2 block text-sm font-medium text-gray-700">Scopes</label>
                      <div className="grid grid-cols-2 gap-2">
                        {allScopes.map((scope) => (
                          <label
                            key={scope}
                            className={cn(
                              'flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors',
                              newKeyForm.scopes.includes(scope)
                                ? 'border-indigo-300 bg-indigo-50 text-indigo-700'
                                : 'border-gray-200 text-gray-600 hover:bg-gray-50',
                            )}
                          >
                            <input
                              type="checkbox"
                              checked={newKeyForm.scopes.includes(scope)}
                              onChange={() => toggleScope(scope)}
                              className="sr-only"
                            />
                            <div
                              className={cn(
                                'flex h-4 w-4 shrink-0 items-center justify-center rounded border',
                                newKeyForm.scopes.includes(scope)
                                  ? 'border-indigo-600 bg-indigo-600'
                                  : 'border-gray-300',
                              )}
                            >
                              {newKeyForm.scopes.includes(scope) && <Check className="h-3 w-3 text-white" />}
                            </div>
                            <span className="font-mono text-xs">{scope}</span>
                          </label>
                        ))}
                      </div>
                    </div>

                    <div>
                      <label className="mb-1 block text-sm font-medium text-gray-700">Expiry</label>
                      <div className="relative">
                        <select
                          value={newKeyForm.expiry}
                          onChange={(e) => setNewKeyForm((f) => ({ ...f, expiry: e.target.value }))}
                          className="w-full appearance-none rounded-lg border border-gray-200 px-3 py-2 pr-8 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        >
                          <option value="30d">30 days</option>
                          <option value="90d">90 days</option>
                          <option value="1y">1 year</option>
                          <option value="never">Never expires</option>
                        </select>
                        <ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                      </div>
                    </div>

                    <div className="flex gap-3 pt-2">
                      <button
                        onClick={handleGenerateKey}
                        disabled={!newKeyForm.name || newKeyForm.scopes.length === 0}
                        className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        Generate Key
                      </button>
                      <button
                        onClick={() => setShowGenerateModal(false)}
                        className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* ============================================================ */}
      {/*  TAB 3 — Webhooks                                            */}
      {/* ============================================================ */}
      {activeTab === 'webhooks' && (
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Webhooks</h1>
              <p className="mt-1 text-sm text-gray-500">
                Configure endpoints to receive real-time event notifications from the platform.
              </p>
            </div>
            <button
              onClick={openAddWebhookModal}
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              <Plus className="h-4 w-4" />
              Add Webhook
            </button>
          </div>

          {/* Webhooks Table */}
          <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
            {webhooks.length === 0 ? (
              <div className="py-16 text-center text-sm text-gray-400">No webhooks configured yet.</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                      <th className="px-6 py-3 font-medium">Endpoint URL</th>
                      <th className="px-6 py-3 font-medium">Events</th>
                      <th className="px-6 py-3 font-medium">Status</th>
                      <th className="px-6 py-3 font-medium">Success Rate</th>
                      <th className="px-6 py-3 font-medium">Last Triggered</th>
                      <th className="px-6 py-3 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {webhooks.map((wh) => (
                      <tr key={wh.id} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <ExternalLink className="h-3.5 w-3.5 shrink-0 text-gray-400" />
                            <span className="text-sm font-mono text-gray-700 truncate max-w-[260px]">{wh.url}</span>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex flex-wrap gap-1">
                            {wh.events.slice(0, 2).map((ev) => (
                              <span key={ev} className="inline-flex rounded bg-indigo-50 px-1.5 py-0.5 text-[10px] font-medium text-indigo-700">
                                {ev}
                              </span>
                            ))}
                            {wh.events.length > 2 && (
                              <span className="inline-flex rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600">
                                +{wh.events.length - 2}
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span
                            className={cn(
                              'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                              wh.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-500',
                            )}
                          >
                            <span className={cn('h-1.5 w-1.5 rounded-full', wh.status === 'active' ? 'bg-green-500' : 'bg-gray-400')} />
                            {wh.status}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <div className="h-1.5 w-16 overflow-hidden rounded-full bg-gray-100">
                              <div
                                className={cn(
                                  'h-full rounded-full',
                                  wh.successRate >= 95 ? 'bg-green-500' : wh.successRate >= 80 ? 'bg-yellow-500' : 'bg-red-500',
                                )}
                                style={{ width: `${wh.successRate}%` }}
                              />
                            </div>
                            <span className="text-sm font-medium text-gray-700">{wh.successRate}%</span>
                          </div>
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-500">
                          {wh.lastTriggered === 'Never' ? 'Never' : formatDateTime(wh.lastTriggered)}
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-1.5">
                            <button
                              onClick={() => openEditWebhookModal(wh)}
                              className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
                              title="Edit"
                            >
                              <Pencil className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => testWebhook(wh.id)}
                              className="rounded-lg p-1.5 text-gray-400 hover:bg-blue-50 hover:text-blue-600 transition-colors"
                              title="Send test event"
                            >
                              {webhookTestId === wh.id ? <Check className="h-4 w-4 text-green-600" /> : <Zap className="h-4 w-4" />}
                            </button>
                            <button
                              onClick={() => deleteWebhook(wh.id)}
                              className="rounded-lg p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors"
                              title="Delete"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Add / Edit Webhook Modal */}
          {showWebhookModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
              <div className="w-full max-w-lg rounded-2xl border border-gray-200 bg-white shadow-xl">
                <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
                  <h3 className="text-base font-semibold text-gray-900">
                    {editingWebhook ? 'Edit Webhook' : 'Add Webhook'}
                  </h3>
                  <button
                    onClick={() => { setShowWebhookModal(false); setEditingWebhook(null); }}
                    className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors"
                  >
                    <X className="h-5 w-5" />
                  </button>
                </div>

                <div className="space-y-4 px-6 py-5">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Endpoint URL</label>
                    <input
                      type="url"
                      value={webhookForm.url}
                      onChange={(e) => setWebhookForm((f) => ({ ...f, url: e.target.value }))}
                      placeholder="https://your-server.com/webhook"
                      className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>

                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Signing Secret</label>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        readOnly
                        value={webhookForm.secret}
                        className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm font-mono text-gray-500 focus:outline-none"
                      />
                      <button
                        onClick={() => setWebhookForm((f) => ({ ...f, secret: generateSecret() }))}
                        className="shrink-0 rounded-lg border border-gray-200 p-2 text-gray-400 hover:bg-gray-50 hover:text-gray-600 transition-colors"
                        title="Regenerate"
                      >
                        <RefreshCw className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => copyToClipboard(webhookForm.secret, 'wh-secret')}
                        className="shrink-0 rounded-lg border border-gray-200 p-2 text-gray-400 hover:bg-gray-50 hover:text-gray-600 transition-colors"
                        title="Copy"
                      >
                        {copiedId === 'wh-secret' ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>

                  <div>
                    <label className="mb-2 block text-sm font-medium text-gray-700">Events</label>
                    <div className="grid grid-cols-2 gap-2">
                      {allEvents.map((event) => (
                        <label
                          key={event}
                          className={cn(
                            'flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors',
                            webhookForm.events.includes(event)
                              ? 'border-indigo-300 bg-indigo-50 text-indigo-700'
                              : 'border-gray-200 text-gray-600 hover:bg-gray-50',
                          )}
                        >
                          <input
                            type="checkbox"
                            checked={webhookForm.events.includes(event)}
                            onChange={() => toggleEvent(event)}
                            className="sr-only"
                          />
                          <div
                            className={cn(
                              'flex h-4 w-4 shrink-0 items-center justify-center rounded border',
                              webhookForm.events.includes(event)
                                ? 'border-indigo-600 bg-indigo-600'
                                : 'border-gray-300',
                            )}
                          >
                            {webhookForm.events.includes(event) && <Check className="h-3 w-3 text-white" />}
                          </div>
                          <span className="font-mono text-xs">{event}</span>
                        </label>
                      ))}
                    </div>
                  </div>

                  {/* Active toggle */}
                  <div className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3">
                    <div>
                      <p className="text-sm font-medium text-gray-700">Active</p>
                      <p className="text-xs text-gray-500">Enable or disable this webhook endpoint</p>
                    </div>
                    <button
                      onClick={() => setWebhookForm((f) => ({ ...f, active: !f.active }))}
                      className={cn(
                        'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full transition-colors',
                        webhookForm.active ? 'bg-indigo-600' : 'bg-gray-200',
                      )}
                    >
                      <span
                        className={cn(
                          'inline-block h-5 w-5 rounded-full bg-white shadow-sm transition-transform mt-0.5',
                          webhookForm.active ? 'translate-x-5 ml-0.5' : 'translate-x-0.5',
                        )}
                      />
                    </button>
                  </div>
                </div>

                <div className="flex gap-3 border-t border-gray-200 px-6 py-4">
                  <button
                    onClick={saveWebhook}
                    disabled={!webhookForm.url || webhookForm.events.length === 0}
                    className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {editingWebhook ? 'Save Changes' : 'Create Webhook'}
                  </button>
                  <button
                    onClick={() => { setShowWebhookModal(false); setEditingWebhook(null); }}
                    className="flex-1 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
