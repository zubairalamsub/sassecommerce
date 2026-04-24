'use client';

import { useState } from 'react';
import { Save, Loader2 } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';

type Tab = 'general' | 'email' | 'security' | 'billing';

const tabs: { label: string; value: Tab }[] = [
  { label: 'General', value: 'general' },
  { label: 'Email', value: 'email' },
  { label: 'Security', value: 'security' },
  { label: 'Billing', value: 'billing' },
];

export default function PlatformSettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('general');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  // General
  const [platformName, setPlatformName] = useState('Saajan');
  const [platformUrl, setPlatformUrl] = useState('https://saajan.com.bd');
  const [supportEmail, setSupportEmail] = useState('support@saajan.com.bd');
  const [defaultTier, setDefaultTier] = useState('starter');
  const [trialDays, setTrialDays] = useState('14');
  const [maxTenants, setMaxTenants] = useState('1000');

  // Email
  const [smtpHost, setSmtpHost] = useState('smtp.gmail.com');
  const [smtpPort, setSmtpPort] = useState('587');
  const [smtpUser, setSmtpUser] = useState('');
  const [smtpPass, setSmtpPass] = useState('');
  const [fromName, setFromName] = useState('Saajan Platform');
  const [fromEmail, setFromEmail] = useState('noreply@saajan.com.bd');

  // Security
  const [jwtExpiry, setJwtExpiry] = useState('24');
  const [maxLoginAttempts, setMaxLoginAttempts] = useState('5');
  const [rateLimitPerMinute, setRateLimitPerMinute] = useState('60');
  const [enforceHttps, setEnforceHttps] = useState(true);
  const [corsOrigins, setCorsOrigins] = useState('*.saajan.com.bd');

  // Billing
  const [currency, setCurrency] = useState('BDT');
  const [taxRate, setTaxRate] = useState('0');
  const [invoicePrefix, setInvoicePrefix] = useState('INV');
  const [paymentGateway, setPaymentGateway] = useState('sslcommerz');

  async function handleSave() {
    setSaving(true);
    // Settings will be persisted to config service when backend is available
    await new Promise((r) => setTimeout(r, 500));
    setSaving(false);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  }

  return (
    <div className="space-y-6">
      <motion.div
        className="flex items-center justify-between"
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <div>
          <h1 className="text-2xl font-bold text-text">Platform Settings</h1>
          <p className="mt-1 text-sm text-text-secondary">Configure platform-wide settings</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-violet-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
          {saved ? 'Saved!' : 'Save Changes'}
        </button>
      </motion.div>

      {/* Tabs */}
      <div className="border-b border-border">
        <nav className="-mb-px flex gap-6">
          {tabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => setActiveTab(tab.value)}
              className={cn(
                'border-b-2 pb-3 text-sm font-medium transition-colors',
                activeTab === tab.value
                  ? 'border-violet-600 text-violet-600 dark:text-violet-400'
                  : 'border-transparent text-text-secondary hover:border-border hover:text-text',
              )}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      <motion.div
        className="rounded-2xl border border-border bg-surface p-6 shadow-sm"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        {activeTab === 'general' && (
          <div className="space-y-5">
            <h2 className="text-lg font-semibold text-text">General Settings</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="Platform Name" value={platformName} onChange={setPlatformName} />
              <Field label="Platform URL" value={platformUrl} onChange={setPlatformUrl} />
              <Field label="Support Email" value={supportEmail} onChange={setSupportEmail} type="email" />
              <div>
                <label className="mb-1.5 block text-sm font-medium text-text">Default Plan for New Tenants</label>
                <select
                  value={defaultTier}
                  onChange={(e) => setDefaultTier(e.target.value)}
                  className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                >
                  <option value="free">Free</option>
                  <option value="starter">Starter</option>
                  <option value="professional">Professional</option>
                </select>
              </div>
              <Field label="Trial Period (days)" value={trialDays} onChange={setTrialDays} type="number" />
              <Field label="Max Tenants" value={maxTenants} onChange={setMaxTenants} type="number" />
            </div>
          </div>
        )}

        {activeTab === 'email' && (
          <div className="space-y-5">
            <h2 className="text-lg font-semibold text-text">Email Configuration</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="SMTP Host" value={smtpHost} onChange={setSmtpHost} />
              <Field label="SMTP Port" value={smtpPort} onChange={setSmtpPort} />
              <Field label="SMTP Username" value={smtpUser} onChange={setSmtpUser} />
              <Field label="SMTP Password" value={smtpPass} onChange={setSmtpPass} type="password" />
              <Field label="From Name" value={fromName} onChange={setFromName} />
              <Field label="From Email" value={fromEmail} onChange={setFromEmail} type="email" />
            </div>
          </div>
        )}

        {activeTab === 'security' && (
          <div className="space-y-5">
            <h2 className="text-lg font-semibold text-text">Security Settings</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="JWT Token Expiry (hours)" value={jwtExpiry} onChange={setJwtExpiry} type="number" />
              <Field label="Max Login Attempts" value={maxLoginAttempts} onChange={setMaxLoginAttempts} type="number" />
              <Field label="API Rate Limit (req/min)" value={rateLimitPerMinute} onChange={setRateLimitPerMinute} type="number" />
              <Field label="CORS Allowed Origins" value={corsOrigins} onChange={setCorsOrigins} />
            </div>
            <div className="flex items-center justify-between rounded-lg border border-border p-4">
              <div>
                <p className="text-sm font-medium text-text">Enforce HTTPS</p>
                <p className="text-xs text-text-muted">Redirect all HTTP traffic to HTTPS</p>
              </div>
              <Toggle checked={enforceHttps} onChange={setEnforceHttps} />
            </div>
          </div>
        )}

        {activeTab === 'billing' && (
          <div className="space-y-5">
            <h2 className="text-lg font-semibold text-text">Billing Settings</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-text">Default Currency</label>
                <select
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value)}
                  className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                >
                  <option value="BDT">BDT - Bangladeshi Taka</option>
                  <option value="USD">USD - US Dollar</option>
                  <option value="EUR">EUR - Euro</option>
                </select>
              </div>
              <Field label="Tax Rate (%)" value={taxRate} onChange={setTaxRate} type="number" />
              <Field label="Invoice Number Prefix" value={invoicePrefix} onChange={setInvoicePrefix} />
              <div>
                <label className="mb-1.5 block text-sm font-medium text-text">Payment Gateway</label>
                <select
                  value={paymentGateway}
                  onChange={(e) => setPaymentGateway(e.target.value)}
                  className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                >
                  <option value="sslcommerz">SSLCommerz</option>
                  <option value="stripe">Stripe</option>
                  <option value="paddle">Paddle</option>
                </select>
              </div>
            </div>
          </div>
        )}
      </motion.div>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  type = 'text',
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-text">{label}</label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
      />
    </div>
  );
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      onClick={() => onChange(!checked)}
      className={cn(
        'relative h-6 w-11 rounded-full transition-colors',
        checked ? 'bg-violet-600' : 'bg-gray-300 dark:bg-gray-600',
      )}
    >
      <div
        className={cn(
          'absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform',
          checked ? 'translate-x-[22px]' : 'translate-x-0.5',
        )}
      />
    </button>
  );
}
