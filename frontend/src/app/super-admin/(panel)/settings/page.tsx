'use client';

import { useState, useEffect } from 'react';
import { Save, Loader2, Globe, CreditCard, Truck, Bell, Search, ChevronDown, ChevronRight, Shield } from 'lucide-react';
import { configApi, type ConfigEntry, type SetConfigRequest } from '@/lib/api';
import { cn } from '@/lib/utils';

type SettingsTab = 'platform' | 'business' | 'services' | 'notifications';

const tabs: { label: string; value: SettingsTab; icon: React.ElementType }[] = [
  { label: 'Platform', value: 'platform', icon: Globe },
  { label: 'Business', value: 'business', icon: CreditCard },
  { label: 'Services', value: 'services', icon: Shield },
  { label: 'Notifications', value: 'notifications', icon: Bell },
];

const NAMESPACE_GROUPS: Record<SettingsTab, string[]> = {
  platform: ['global', 'tenant.plans', 'search', 'cart', 'analytics'],
  business: ['business.shipping', 'business.vendor', 'business.loyalty', 'business.promotion', 'payment'],
  services: ['services', 'kafka'],
  notifications: ['notification'],
};

const NAMESPACE_LABELS: Record<string, string> = {
  global: 'Global Settings',
  'tenant.plans': 'Tenant Plans',
  search: 'Search',
  cart: 'Cart',
  analytics: 'Analytics',
  'business.shipping': 'Shipping',
  'business.vendor': 'Vendor',
  'business.loyalty': 'Loyalty',
  'business.promotion': 'Promotions',
  payment: 'Payment',
  services: 'Service Ports',
  kafka: 'Kafka',
  notification: 'Notifications',
};

interface EditableConfig extends ConfigEntry {
  dirty?: boolean;
  editValue?: string;
}

export default function PlatformSettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('platform');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  const [configs, setConfigs] = useState<Record<string, EditableConfig[]>>({});
  const [expandedNs, setExpandedNs] = useState<Record<string, boolean>>({});
  const [searchQuery, setSearchQuery] = useState('');

  // Load configs for current tab
  useEffect(() => {
    loadTabConfigs(activeTab);
  }, [activeTab]);

  async function loadTabConfigs(tab: SettingsTab) {
    setLoading(true);
    setError('');
    const namespaces = NAMESPACE_GROUPS[tab];
    try {
      const results = await Promise.all(
        namespaces.map(async (ns) => {
          try {
            const entries = await configApi.listByNamespace(ns);
            return { ns, entries: Array.isArray(entries) ? entries : [] };
          } catch {
            return { ns, entries: [] };
          }
        }),
      );

      const newConfigs: Record<string, EditableConfig[]> = {};
      const newExpanded: Record<string, boolean> = {};
      results.forEach(({ ns, entries }) => {
        newConfigs[ns] = entries.map((e) => ({ ...e, dirty: false, editValue: e.value }));
        newExpanded[ns] = entries.length > 0;
      });

      setConfigs(newConfigs);
      setExpandedNs(newExpanded);
    } catch (err) {
      setError((err as Error).message || 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  }

  function updateConfigValue(namespace: string, key: string, newValue: string) {
    setSaved(false);
    setConfigs((prev) => ({
      ...prev,
      [namespace]: (prev[namespace] || []).map((c) =>
        c.key === key ? { ...c, editValue: newValue, dirty: c.value !== newValue } : c,
      ),
    }));
  }

  function getDirtyConfigs(): { namespace: string; config: EditableConfig }[] {
    const dirty: { namespace: string; config: EditableConfig }[] = [];
    Object.entries(configs).forEach(([ns, entries]) => {
      entries.forEach((c) => {
        if (c.dirty) dirty.push({ namespace: ns, config: c });
      });
    });
    return dirty;
  }

  async function handleSave() {
    const dirty = getDirtyConfigs();
    if (dirty.length === 0) return;

    setSaving(true);
    setError('');
    try {
      const requests: SetConfigRequest[] = dirty.map(({ config: c }) => ({
        namespace: c.namespace,
        key: c.key,
        value: c.editValue || c.value,
        value_type: c.value_type,
        description: c.description,
        environment: c.environment,
        tenant_id: c.tenant_id || undefined,
        updated_by: 'super_admin',
      }));

      await configApi.bulkSet(requests);

      // Mark all as clean
      setConfigs((prev) => {
        const updated = { ...prev };
        Object.keys(updated).forEach((ns) => {
          updated[ns] = updated[ns].map((c) =>
            c.dirty ? { ...c, value: c.editValue || c.value, dirty: false } : c,
          );
        });
        return updated;
      });

      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err) {
      setError((err as Error).message || 'Failed to save settings');
    } finally {
      setSaving(false);
    }
  }

  function toggleNamespace(ns: string) {
    setExpandedNs((prev) => ({ ...prev, [ns]: !prev[ns] }));
  }

  const dirtyCount = getDirtyConfigs().length;

  // Filter configs by search
  function getFilteredConfigs(ns: string): EditableConfig[] {
    const entries = configs[ns] || [];
    if (!searchQuery) return entries;
    const q = searchQuery.toLowerCase();
    return entries.filter(
      (c) =>
        c.key.toLowerCase().includes(q) ||
        c.description.toLowerCase().includes(q) ||
        c.value.toLowerCase().includes(q),
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Platform Settings</h1>
          <p className="text-sm text-gray-500">Configure global platform settings across all services</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving || dirtyCount === 0}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          {saving ? 'Saving...' : saved ? 'Saved!' : dirtyCount > 0 ? `Save ${dirtyCount} Change${dirtyCount > 1 ? 's' : ''}` : 'Save Changes'}
        </button>
      </div>

      {error && (
        <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}
      {saved && (
        <div className="rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">Settings saved successfully.</div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                className={cn(
                  'flex items-center gap-2 border-b-2 pb-3 text-sm font-medium transition-colors',
                  activeTab === tab.value
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

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search settings by key, description, or value..."
          className="w-full rounded-lg border border-gray-300 py-2.5 pl-10 pr-3.5 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600"
        />
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex min-h-[40vh] items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : (
        <div className="space-y-4">
          {NAMESPACE_GROUPS[activeTab].map((ns) => {
            const filtered = getFilteredConfigs(ns);
            if (searchQuery && filtered.length === 0) return null;
            const isExpanded = expandedNs[ns];

            return (
              <div key={ns} className="rounded-xl border border-gray-200 bg-white shadow-sm">
                {/* Namespace Header */}
                <button
                  onClick={() => toggleNamespace(ns)}
                  className="flex w-full items-center justify-between px-6 py-4 text-left"
                >
                  <div className="flex items-center gap-2">
                    {isExpanded ? (
                      <ChevronDown className="h-4 w-4 text-gray-400" />
                    ) : (
                      <ChevronRight className="h-4 w-4 text-gray-400" />
                    )}
                    <h3 className="text-sm font-semibold text-gray-900">
                      {NAMESPACE_LABELS[ns] || ns}
                    </h3>
                    <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
                      {filtered.length}
                    </span>
                  </div>
                  <span className="text-xs font-mono text-gray-400">{ns}</span>
                </button>

                {/* Config Entries */}
                {isExpanded && filtered.length > 0 && (
                  <div className="border-t border-gray-100">
                    <div className="divide-y divide-gray-50">
                      {filtered.map((config) => (
                        <div key={`${config.namespace}-${config.key}`} className="px-6 py-3">
                          <div className="flex items-start justify-between gap-4">
                            <div className="min-w-0 flex-1">
                              <div className="flex items-center gap-2">
                                <p className="text-sm font-medium text-gray-900 font-mono">{config.key}</p>
                                <span className={cn(
                                  'rounded px-1.5 py-0.5 text-[10px] font-medium',
                                  config.value_type === 'boolean' ? 'bg-purple-50 text-purple-700' :
                                  config.value_type === 'number' ? 'bg-blue-50 text-blue-700' :
                                  config.value_type === 'json' ? 'bg-amber-50 text-amber-700' :
                                  'bg-gray-50 text-gray-600',
                                )}>
                                  {config.value_type}
                                </span>
                                {config.is_secret && (
                                  <span className="rounded bg-red-50 px-1.5 py-0.5 text-[10px] font-medium text-red-700">
                                    secret
                                  </span>
                                )}
                                {config.dirty && (
                                  <span className="rounded bg-yellow-50 px-1.5 py-0.5 text-[10px] font-medium text-yellow-700">
                                    modified
                                  </span>
                                )}
                              </div>
                              {config.description && (
                                <p className="mt-0.5 text-xs text-gray-500">{config.description}</p>
                              )}
                            </div>
                            <div className="w-72 flex-shrink-0">
                              {config.value_type === 'boolean' ? (
                                <button
                                  type="button"
                                  onClick={() =>
                                    updateConfigValue(
                                      config.namespace,
                                      config.key,
                                      config.editValue === 'true' ? 'false' : 'true',
                                    )
                                  }
                                  className={cn(
                                    'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                                    config.editValue === 'true' ? 'bg-indigo-600' : 'bg-gray-200',
                                  )}
                                >
                                  <span
                                    className={cn(
                                      'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                                      config.editValue === 'true' ? 'translate-x-6' : 'translate-x-1',
                                    )}
                                  />
                                </button>
                              ) : config.value_type === 'json' || (config.editValue || '').length > 80 ? (
                                <textarea
                                  rows={3}
                                  value={config.editValue || ''}
                                  onChange={(e) => updateConfigValue(config.namespace, config.key, e.target.value)}
                                  className={cn(
                                    'w-full rounded-lg border px-3 py-1.5 font-mono text-xs focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600',
                                    config.dirty ? 'border-yellow-400 bg-yellow-50' : 'border-gray-300',
                                  )}
                                />
                              ) : (
                                <input
                                  type={config.value_type === 'number' ? 'number' : 'text'}
                                  value={config.is_secret ? '********' : (config.editValue || '')}
                                  disabled={config.is_secret}
                                  onChange={(e) => updateConfigValue(config.namespace, config.key, e.target.value)}
                                  className={cn(
                                    'w-full rounded-lg border px-3 py-1.5 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600',
                                    config.dirty ? 'border-yellow-400 bg-yellow-50' : 'border-gray-300',
                                    config.is_secret && 'bg-gray-50 text-gray-400',
                                  )}
                                />
                              )}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {isExpanded && filtered.length === 0 && (
                  <div className="border-t border-gray-100 px-6 py-8 text-center text-sm text-gray-500">
                    No config entries found in this namespace.
                    {!searchQuery && ' The config service may need to be started.'}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
