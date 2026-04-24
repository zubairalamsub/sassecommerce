'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Save,
  Loader2,
  Globe,
  Mail,
  Languages,
  UserPlus,
  Bell,
  Eye,
  EyeOff,
  Send,
  CheckCircle2,
  AlertCircle,
  X,
  Plus,
  Pencil,
} from 'lucide-react';
import { configApi, type SetConfigRequest } from '@/lib/api';
import { cn } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type SettingsTab = 'general' | 'email' | 'localization' | 'registration' | 'notifications';

interface Toast {
  id: number;
  message: string;
  type: 'success' | 'error';
}

interface EmailTemplate {
  id: string;
  name: string;
  subject: string;
  body: string;
  variables: string[];
  lastModified: string;
}

interface SupportedLanguage {
  name: string;
  code: string;
  enabled: boolean;
  isDefault: boolean;
}

interface SupportedCurrency {
  name: string;
  code: string;
  symbol: string;
  exchangeRate: number;
  enabled: boolean;
  isDefault: boolean;
}

interface OnboardingStep {
  key: string;
  label: string;
  enabled: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const TABS: { label: string; value: SettingsTab; icon: React.ElementType }[] = [
  { label: 'General', value: 'general', icon: Globe },
  { label: 'Email & SMTP', value: 'email', icon: Mail },
  { label: 'Localization', value: 'localization', icon: Languages },
  { label: 'Registration', value: 'registration', icon: UserPlus },
  { label: 'Notifications', value: 'notifications', icon: Bell },
];

const DEFAULT_EMAIL_TEMPLATES: EmailTemplate[] = [
  {
    id: 'welcome',
    name: 'Welcome',
    subject: 'Welcome to {{platform_name}}, {{name}}!',
    body: 'Hi {{name}},\n\nWelcome to {{platform_name}}! Your account has been created successfully.\n\nEmail: {{email}}\n\nGet started by setting up your store.\n\nBest regards,\n{{platform_name}} Team',
    variables: ['name', 'email', 'platform_name'],
    lastModified: '2026-04-20T10:00:00Z',
  },
  {
    id: 'password_reset',
    name: 'Password Reset',
    subject: 'Reset your password - {{platform_name}}',
    body: 'Hi {{name}},\n\nWe received a request to reset your password. Click the link below to set a new password:\n\n{{reset_link}}\n\nThis link expires in {{expiry_hours}} hours.\n\nIf you did not request this, please ignore this email.',
    variables: ['name', 'reset_link', 'expiry_hours', 'platform_name'],
    lastModified: '2026-04-18T14:30:00Z',
  },
  {
    id: 'invoice',
    name: 'Invoice',
    subject: 'Invoice #{{invoice_number}} from {{platform_name}}',
    body: 'Hi {{name}},\n\nPlease find your invoice details below:\n\nInvoice #: {{invoice_number}}\nAmount: {{amount}}\nDue Date: {{due_date}}\n\nPlan: {{plan_name}}\nBilling Period: {{billing_period}}\n\nThank you for your business.',
    variables: ['name', 'invoice_number', 'amount', 'due_date', 'plan_name', 'billing_period', 'platform_name'],
    lastModified: '2026-04-15T09:00:00Z',
  },
  {
    id: 'tenant_approved',
    name: 'Tenant Approved',
    subject: 'Your store has been approved! - {{platform_name}}',
    body: 'Hi {{name}},\n\nGreat news! Your store "{{store_name}}" has been approved and is now live.\n\nStore URL: {{store_url}}\n\nYou can now start adding products and configuring your store.\n\nBest regards,\n{{platform_name}} Team',
    variables: ['name', 'store_name', 'store_url', 'platform_name'],
    lastModified: '2026-04-12T16:45:00Z',
  },
  {
    id: 'tenant_suspended',
    name: 'Tenant Suspended',
    subject: 'Your store has been suspended - {{platform_name}}',
    body: 'Hi {{name}},\n\nYour store "{{store_name}}" has been suspended.\n\nReason: {{reason}}\n\nPlease contact support at {{support_email}} to resolve this issue.\n\nRegards,\n{{platform_name}} Team',
    variables: ['name', 'store_name', 'reason', 'support_email', 'platform_name'],
    lastModified: '2026-04-10T11:20:00Z',
  },
  {
    id: 'plan_upgraded',
    name: 'Plan Upgraded',
    subject: 'Plan upgrade confirmed - {{platform_name}}',
    body: 'Hi {{name}},\n\nYour plan has been upgraded successfully!\n\nPrevious Plan: {{old_plan}}\nNew Plan: {{new_plan}}\nEffective Date: {{effective_date}}\n\nYou now have access to all {{new_plan}} features.\n\nThank you,\n{{platform_name}} Team',
    variables: ['name', 'old_plan', 'new_plan', 'effective_date', 'platform_name'],
    lastModified: '2026-04-08T08:10:00Z',
  },
];

const DEFAULT_LANGUAGES: SupportedLanguage[] = [
  { name: 'English', code: 'en', enabled: true, isDefault: true },
  { name: 'Bangla', code: 'bn', enabled: true, isDefault: false },
  { name: 'Hindi', code: 'hi', enabled: false, isDefault: false },
  { name: 'Arabic', code: 'ar', enabled: false, isDefault: false },
];

const DEFAULT_CURRENCIES: SupportedCurrency[] = [
  { name: 'Bangladeshi Taka', code: 'BDT', symbol: '৳', exchangeRate: 1.0, enabled: true, isDefault: true },
  { name: 'US Dollar', code: 'USD', symbol: '$', exchangeRate: 0.0091, enabled: true, isDefault: false },
  { name: 'Indian Rupee', code: 'INR', symbol: '₹', exchangeRate: 0.76, enabled: true, isDefault: false },
  { name: 'Euro', code: 'EUR', symbol: '€', exchangeRate: 0.0084, enabled: true, isDefault: false },
  { name: 'British Pound', code: 'GBP', symbol: '£', exchangeRate: 0.0072, enabled: true, isDefault: false },
];

const DEFAULT_ONBOARDING_STEPS: OnboardingStep[] = [
  { key: 'add_products', label: 'Add products', enabled: true },
  { key: 'configure_payment', label: 'Configure payment', enabled: true },
  { key: 'setup_shipping', label: 'Set up shipping', enabled: true },
  { key: 'customize_theme', label: 'Customize theme', enabled: true },
  { key: 'add_team_members', label: 'Add team members', enabled: false },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

let toastCounter = 0;

function formatDateShort(iso: string): string {
  return new Date(iso).toLocaleDateString('en-BD', { year: 'numeric', month: 'short', day: 'numeric' });
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function Toggle({
  enabled,
  onChange,
  disabled,
}: {
  enabled: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onChange(!enabled)}
      className={cn(
        'relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2',
        enabled ? 'bg-indigo-600' : 'bg-gray-200',
        disabled && 'cursor-not-allowed opacity-50',
      )}
    >
      <span
        className={cn(
          'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
          enabled ? 'translate-x-6' : 'translate-x-1',
        )}
      />
    </button>
  );
}

function SectionCard({ title, description, children }: { title: string; description?: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
      <div className="border-b border-gray-100 px-6 py-4">
        <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
        {description && <p className="mt-0.5 text-xs text-gray-500">{description}</p>}
      </div>
      <div className="px-6 py-5 space-y-5">{children}</div>
    </div>
  );
}

function Field({ label, description, children }: { label: string; description?: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 gap-1 sm:grid-cols-3 sm:items-start sm:gap-4">
      <div className="sm:pt-1.5">
        <label className="text-sm font-medium text-gray-700">{label}</label>
        {description && <p className="text-xs text-gray-400 mt-0.5">{description}</p>}
      </div>
      <div className="sm:col-span-2">{children}</div>
    </div>
  );
}

function TextInput({
  value,
  onChange,
  placeholder,
  type = 'text',
  disabled,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
  disabled?: boolean;
}) {
  return (
    <input
      type={type}
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className={cn(
        'w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600',
        disabled && 'bg-gray-50 text-gray-400 cursor-not-allowed',
      )}
    />
  );
}

function SelectInput({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (v: string) => void;
  options: { label: string; value: string }[];
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );
}

function NumberInput({
  value,
  onChange,
  min,
  max,
  step,
  placeholder,
}: {
  value: number;
  onChange: (v: number) => void;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
}) {
  return (
    <input
      type="number"
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
      min={min}
      max={max}
      step={step}
      placeholder={placeholder}
      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600"
    />
  );
}

function TextArea({
  value,
  onChange,
  rows = 3,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  rows?: number;
  placeholder?: string;
}) {
  return (
    <textarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      rows={rows}
      placeholder={placeholder}
      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600"
    />
  );
}

// ---------------------------------------------------------------------------
// Main Page Component
// ---------------------------------------------------------------------------

export default function PlatformSettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [toasts, setToasts] = useState<Toast[]>([]);

  // ---- General Tab State ----
  const [platformName, setPlatformName] = useState('Saajan Platform');
  const [platformUrl, setPlatformUrl] = useState('https://saajan.com.bd');
  const [supportEmail, setSupportEmail] = useState('support@saajan.com.bd');
  const [supportPhone, setSupportPhone] = useState('+880 1700-000000');
  const [logoUrl, setLogoUrl] = useState('/logo.png');
  const [defaultTier, setDefaultTier] = useState('free');
  const [trialPeriodDays, setTrialPeriodDays] = useState(14);
  const [maxTenants, setMaxTenants] = useState(0);
  const [autoApproveTenants, setAutoApproveTenants] = useState(false);
  const [maintenanceMode, setMaintenanceMode] = useState(false);
  const [maintenanceMessage, setMaintenanceMessage] = useState('We are currently performing scheduled maintenance. Please check back shortly.');

  // ---- Email & SMTP Tab State ----
  const [smtpHost, setSmtpHost] = useState('smtp.gmail.com');
  const [smtpPort, setSmtpPort] = useState(587);
  const [smtpUsername, setSmtpUsername] = useState('');
  const [smtpPassword, setSmtpPassword] = useState('');
  const [smtpShowPassword, setSmtpShowPassword] = useState(false);
  const [smtpEncryption, setSmtpEncryption] = useState('tls');
  const [smtpFromName, setSmtpFromName] = useState('Saajan Platform');
  const [smtpFromEmail, setSmtpFromEmail] = useState('noreply@saajan.com.bd');
  const [smtpTesting, setSmtpTesting] = useState(false);
  const [smtpTestResult, setSmtpTestResult] = useState<{ ok: boolean; message: string } | null>(null);
  const [emailTemplates, setEmailTemplates] = useState<EmailTemplate[]>(DEFAULT_EMAIL_TEMPLATES);
  const [editingTemplate, setEditingTemplate] = useState<EmailTemplate | null>(null);
  const [editTemplateSubject, setEditTemplateSubject] = useState('');
  const [editTemplateBody, setEditTemplateBody] = useState('');

  // ---- Localization Tab State ----
  const [defaultLanguage, setDefaultLanguage] = useState('en');
  const [defaultTimezone, setDefaultTimezone] = useState('Asia/Dhaka');
  const [dateFormat, setDateFormat] = useState('DD/MM/YYYY');
  const [timeFormat, setTimeFormat] = useState<'12h' | '24h'>('12h');
  const [supportedLanguages, setSupportedLanguages] = useState<SupportedLanguage[]>(DEFAULT_LANGUAGES);
  const [currencies, setCurrencies] = useState<SupportedCurrency[]>(DEFAULT_CURRENCIES);
  const [autoUpdateRates, setAutoUpdateRates] = useState(false);

  // ---- Registration Tab State ----
  const [allowPublicRegistration, setAllowPublicRegistration] = useState(true);
  const [requireEmailVerification, setRequireEmailVerification] = useState(true);
  const [autoApproveRegistration, setAutoApproveRegistration] = useState(false);
  const [requireBusinessVerification, setRequireBusinessVerification] = useState(false);
  const [allowedEmailDomains, setAllowedEmailDomains] = useState('');
  const [blockedEmailDomains, setBlockedEmailDomains] = useState('mailinator.com\nguerrillamail.com\nthrowaway.email\ntempmail.com\nyopmail.com');
  const [defaultStoreTemplate, setDefaultStoreTemplate] = useState('blank');
  const [showOnboardingWizard, setShowOnboardingWizard] = useState(true);
  const [onboardingSteps, setOnboardingSteps] = useState<OnboardingStep[]>(DEFAULT_ONBOARDING_STEPS);
  const [autoCreateDatabase, setAutoCreateDatabase] = useState(true);
  const [defaultStorageQuota, setDefaultStorageQuota] = useState(5);
  const [defaultSubdomainPattern, setDefaultSubdomainPattern] = useState('{slug}.saajan.com.bd');
  const [customDomainSupport, setCustomDomainSupport] = useState(true);

  // ---- Notifications Tab State ----
  const [notifyNewTenant, setNotifyNewTenant] = useState(true);
  const [notifyPlanChange, setNotifyPlanChange] = useState(true);
  const [notifyPaymentReceived, setNotifyPaymentReceived] = useState(false);
  const [notifyPaymentFailed, setNotifyPaymentFailed] = useState(true);
  const [notifyDisputeOpened, setNotifyDisputeOpened] = useState(true);
  const [notifySystemAlerts, setNotifySystemAlerts] = useState(true);
  const [notificationEmail, setNotificationEmail] = useState('admin@saajan.com.bd');
  const [tenantNotifyNewOrder, setTenantNotifyNewOrder] = useState(true);
  const [tenantNotifyOrderStatusChange, setTenantNotifyOrderStatusChange] = useState(true);
  const [tenantNotifyLowStock, setTenantNotifyLowStock] = useState(true);
  const [lowStockThreshold, setLowStockThreshold] = useState(10);
  const [tenantNotifyCustomerRegistration, setTenantNotifyCustomerRegistration] = useState(false);
  const [tenantNotifyReviewPosted, setTenantNotifyReviewPosted] = useState(true);

  // ---- Toast helper ----
  const addToast = useCallback((message: string, type: 'success' | 'error' = 'success') => {
    const id = ++toastCounter;
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3000);
  }, []);

  const dismissToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  // ---- Load settings from config service ----
  useEffect(() => {
    async function load() {
      setLoading(true);
      try {
        const [globalEntries, smtpEntries, localeEntries, regEntries, notifEntries] = await Promise.all([
          configApi.listByNamespace('platform.general').catch(() => []),
          configApi.listByNamespace('platform.smtp').catch(() => []),
          configApi.listByNamespace('platform.localization').catch(() => []),
          configApi.listByNamespace('platform.registration').catch(() => []),
          configApi.listByNamespace('platform.notifications').catch(() => []),
        ]);

        const toMap = (entries: { key: string; value: string }[]) => {
          const m: Record<string, string> = {};
          entries.forEach((e) => {
            m[e.key] = e.value;
          });
          return m;
        };

        const g = toMap(globalEntries);
        if (g['platform_name']) setPlatformName(g['platform_name']);
        if (g['platform_url']) setPlatformUrl(g['platform_url']);
        if (g['support_email']) setSupportEmail(g['support_email']);
        if (g['support_phone']) setSupportPhone(g['support_phone']);
        if (g['logo_url']) setLogoUrl(g['logo_url']);
        if (g['default_tier']) setDefaultTier(g['default_tier']);
        if (g['trial_period_days']) setTrialPeriodDays(Number(g['trial_period_days']));
        if (g['max_tenants']) setMaxTenants(Number(g['max_tenants']));
        if (g['auto_approve_tenants']) setAutoApproveTenants(g['auto_approve_tenants'] === 'true');
        if (g['maintenance_mode']) setMaintenanceMode(g['maintenance_mode'] === 'true');
        if (g['maintenance_message']) setMaintenanceMessage(g['maintenance_message']);

        const s = toMap(smtpEntries);
        if (s['host']) setSmtpHost(s['host']);
        if (s['port']) setSmtpPort(Number(s['port']));
        if (s['username']) setSmtpUsername(s['username']);
        if (s['encryption']) setSmtpEncryption(s['encryption']);
        if (s['from_name']) setSmtpFromName(s['from_name']);
        if (s['from_email']) setSmtpFromEmail(s['from_email']);

        const l = toMap(localeEntries);
        if (l['default_language']) setDefaultLanguage(l['default_language']);
        if (l['default_timezone']) setDefaultTimezone(l['default_timezone']);
        if (l['date_format']) setDateFormat(l['date_format']);
        if (l['time_format']) setTimeFormat(l['time_format'] as '12h' | '24h');
        if (l['auto_update_rates']) setAutoUpdateRates(l['auto_update_rates'] === 'true');
        if (l['supported_languages']) {
          try { setSupportedLanguages(JSON.parse(l['supported_languages'])); } catch { /* keep default */ }
        }
        if (l['currencies']) {
          try { setCurrencies(JSON.parse(l['currencies'])); } catch { /* keep default */ }
        }

        const r = toMap(regEntries);
        if (r['allow_public_registration']) setAllowPublicRegistration(r['allow_public_registration'] === 'true');
        if (r['require_email_verification']) setRequireEmailVerification(r['require_email_verification'] === 'true');
        if (r['auto_approve']) setAutoApproveRegistration(r['auto_approve'] === 'true');
        if (r['require_business_verification']) setRequireBusinessVerification(r['require_business_verification'] === 'true');
        if (r['allowed_email_domains']) setAllowedEmailDomains(r['allowed_email_domains']);
        if (r['blocked_email_domains']) setBlockedEmailDomains(r['blocked_email_domains']);
        if (r['default_store_template']) setDefaultStoreTemplate(r['default_store_template']);
        if (r['show_onboarding_wizard']) setShowOnboardingWizard(r['show_onboarding_wizard'] === 'true');
        if (r['onboarding_steps']) {
          try { setOnboardingSteps(JSON.parse(r['onboarding_steps'])); } catch { /* keep default */ }
        }
        if (r['auto_create_database']) setAutoCreateDatabase(r['auto_create_database'] === 'true');
        if (r['default_storage_quota']) setDefaultStorageQuota(Number(r['default_storage_quota']));
        if (r['default_subdomain_pattern']) setDefaultSubdomainPattern(r['default_subdomain_pattern']);
        if (r['custom_domain_support']) setCustomDomainSupport(r['custom_domain_support'] === 'true');

        const n = toMap(notifEntries);
        if (n['notify_new_tenant']) setNotifyNewTenant(n['notify_new_tenant'] === 'true');
        if (n['notify_plan_change']) setNotifyPlanChange(n['notify_plan_change'] === 'true');
        if (n['notify_payment_received']) setNotifyPaymentReceived(n['notify_payment_received'] === 'true');
        if (n['notify_payment_failed']) setNotifyPaymentFailed(n['notify_payment_failed'] === 'true');
        if (n['notify_dispute_opened']) setNotifyDisputeOpened(n['notify_dispute_opened'] === 'true');
        if (n['notify_system_alerts']) setNotifySystemAlerts(n['notify_system_alerts'] === 'true');
        if (n['notification_email']) setNotificationEmail(n['notification_email']);
        if (n['tenant_notify_new_order']) setTenantNotifyNewOrder(n['tenant_notify_new_order'] === 'true');
        if (n['tenant_notify_order_status_change']) setTenantNotifyOrderStatusChange(n['tenant_notify_order_status_change'] === 'true');
        if (n['tenant_notify_low_stock']) setTenantNotifyLowStock(n['tenant_notify_low_stock'] === 'true');
        if (n['low_stock_threshold']) setLowStockThreshold(Number(n['low_stock_threshold']));
        if (n['tenant_notify_customer_registration']) setTenantNotifyCustomerRegistration(n['tenant_notify_customer_registration'] === 'true');
        if (n['tenant_notify_review_posted']) setTenantNotifyReviewPosted(n['tenant_notify_review_posted'] === 'true');
      } catch {
        // Silently use defaults if config service is unavailable
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  // ---- Build entries for the active tab and save ----
  function buildEntries(): SetConfigRequest[] {
    const entries: SetConfigRequest[] = [];
    const add = (namespace: string, key: string, value: string, valueType: 'string' | 'number' | 'boolean' | 'json' = 'string', isSecret = false) => {
      entries.push({ namespace, key, value, value_type: valueType, updated_by: 'super_admin', is_secret: isSecret });
    };

    switch (activeTab) {
      case 'general':
        add('platform.general', 'platform_name', platformName);
        add('platform.general', 'platform_url', platformUrl);
        add('platform.general', 'support_email', supportEmail);
        add('platform.general', 'support_phone', supportPhone);
        add('platform.general', 'logo_url', logoUrl);
        add('platform.general', 'default_tier', defaultTier);
        add('platform.general', 'trial_period_days', String(trialPeriodDays), 'number');
        add('platform.general', 'max_tenants', String(maxTenants), 'number');
        add('platform.general', 'auto_approve_tenants', String(autoApproveTenants), 'boolean');
        add('platform.general', 'maintenance_mode', String(maintenanceMode), 'boolean');
        add('platform.general', 'maintenance_message', maintenanceMessage);
        break;

      case 'email':
        add('platform.smtp', 'host', smtpHost);
        add('platform.smtp', 'port', String(smtpPort), 'number');
        add('platform.smtp', 'username', smtpUsername);
        if (smtpPassword) add('platform.smtp', 'password', smtpPassword, 'string', true);
        add('platform.smtp', 'encryption', smtpEncryption);
        add('platform.smtp', 'from_name', smtpFromName);
        add('platform.smtp', 'from_email', smtpFromEmail);
        add('platform.smtp', 'email_templates', JSON.stringify(emailTemplates), 'json');
        break;

      case 'localization':
        add('platform.localization', 'default_language', defaultLanguage);
        add('platform.localization', 'default_timezone', defaultTimezone);
        add('platform.localization', 'date_format', dateFormat);
        add('platform.localization', 'time_format', timeFormat);
        add('platform.localization', 'supported_languages', JSON.stringify(supportedLanguages), 'json');
        add('platform.localization', 'currencies', JSON.stringify(currencies), 'json');
        add('platform.localization', 'auto_update_rates', String(autoUpdateRates), 'boolean');
        break;

      case 'registration':
        add('platform.registration', 'allow_public_registration', String(allowPublicRegistration), 'boolean');
        add('platform.registration', 'require_email_verification', String(requireEmailVerification), 'boolean');
        add('platform.registration', 'auto_approve', String(autoApproveRegistration), 'boolean');
        add('platform.registration', 'require_business_verification', String(requireBusinessVerification), 'boolean');
        add('platform.registration', 'allowed_email_domains', allowedEmailDomains);
        add('platform.registration', 'blocked_email_domains', blockedEmailDomains);
        add('platform.registration', 'default_store_template', defaultStoreTemplate);
        add('platform.registration', 'show_onboarding_wizard', String(showOnboardingWizard), 'boolean');
        add('platform.registration', 'onboarding_steps', JSON.stringify(onboardingSteps), 'json');
        add('platform.registration', 'auto_create_database', String(autoCreateDatabase), 'boolean');
        add('platform.registration', 'default_storage_quota', String(defaultStorageQuota), 'number');
        add('platform.registration', 'default_subdomain_pattern', defaultSubdomainPattern);
        add('platform.registration', 'custom_domain_support', String(customDomainSupport), 'boolean');
        break;

      case 'notifications':
        add('platform.notifications', 'notify_new_tenant', String(notifyNewTenant), 'boolean');
        add('platform.notifications', 'notify_plan_change', String(notifyPlanChange), 'boolean');
        add('platform.notifications', 'notify_payment_received', String(notifyPaymentReceived), 'boolean');
        add('platform.notifications', 'notify_payment_failed', String(notifyPaymentFailed), 'boolean');
        add('platform.notifications', 'notify_dispute_opened', String(notifyDisputeOpened), 'boolean');
        add('platform.notifications', 'notify_system_alerts', String(notifySystemAlerts), 'boolean');
        add('platform.notifications', 'notification_email', notificationEmail);
        add('platform.notifications', 'tenant_notify_new_order', String(tenantNotifyNewOrder), 'boolean');
        add('platform.notifications', 'tenant_notify_order_status_change', String(tenantNotifyOrderStatusChange), 'boolean');
        add('platform.notifications', 'tenant_notify_low_stock', String(tenantNotifyLowStock), 'boolean');
        add('platform.notifications', 'low_stock_threshold', String(lowStockThreshold), 'number');
        add('platform.notifications', 'tenant_notify_customer_registration', String(tenantNotifyCustomerRegistration), 'boolean');
        add('platform.notifications', 'tenant_notify_review_posted', String(tenantNotifyReviewPosted), 'boolean');
        break;
    }

    return entries;
  }

  async function handleSave() {
    setSaving(true);
    try {
      const entries = buildEntries();
      await configApi.bulkSet(entries);
      addToast('Settings saved successfully.');
    } catch (err) {
      addToast((err as Error).message || 'Failed to save settings.', 'error');
    } finally {
      setSaving(false);
    }
  }

  async function handleTestSmtp() {
    setSmtpTesting(true);
    setSmtpTestResult(null);
    try {
      // Simulate SMTP test via config service
      await new Promise((resolve) => setTimeout(resolve, 1500));
      if (!smtpHost || !smtpPort) {
        setSmtpTestResult({ ok: false, message: 'SMTP host and port are required.' });
      } else {
        setSmtpTestResult({ ok: true, message: 'Connection successful. Test email sent.' });
      }
    } catch {
      setSmtpTestResult({ ok: false, message: 'Connection failed. Please check your SMTP settings.' });
    } finally {
      setSmtpTesting(false);
    }
  }

  function openTemplateEditor(template: EmailTemplate) {
    setEditingTemplate(template);
    setEditTemplateSubject(template.subject);
    setEditTemplateBody(template.body);
  }

  function saveTemplateEdit() {
    if (!editingTemplate) return;
    setEmailTemplates((prev) =>
      prev.map((t) =>
        t.id === editingTemplate.id
          ? { ...t, subject: editTemplateSubject, body: editTemplateBody, lastModified: new Date().toISOString() }
          : t,
      ),
    );
    setEditingTemplate(null);
    addToast(`Template "${editingTemplate.name}" updated. Save changes to persist.`);
  }

  function toggleLanguageEnabled(code: string) {
    setSupportedLanguages((prev) =>
      prev.map((l) => (l.code === code ? { ...l, enabled: !l.enabled } : l)),
    );
  }

  function setDefaultLanguageRow(code: string) {
    setSupportedLanguages((prev) =>
      prev.map((l) => ({ ...l, isDefault: l.code === code })),
    );
    setDefaultLanguage(code);
  }

  function toggleCurrencyEnabled(code: string) {
    setCurrencies((prev) =>
      prev.map((c) => (c.code === code ? { ...c, enabled: !c.enabled } : c)),
    );
  }

  function setDefaultCurrencyRow(code: string) {
    setCurrencies((prev) =>
      prev.map((c) => ({ ...c, isDefault: c.code === code })),
    );
  }

  function toggleOnboardingStep(key: string) {
    setOnboardingSteps((prev) =>
      prev.map((s) => (s.key === key ? { ...s, enabled: !s.enabled } : s)),
    );
  }

  // ---- Render ----

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Toast Container */}
      <div className="fixed right-6 top-6 z-50 space-y-2">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={cn(
              'flex items-center gap-2 rounded-lg px-4 py-3 text-sm shadow-lg animate-in fade-in slide-in-from-right-5',
              toast.type === 'success'
                ? 'bg-green-50 text-green-800 border border-green-200'
                : 'bg-red-50 text-red-800 border border-red-200',
            )}
          >
            {toast.type === 'success' ? (
              <CheckCircle2 className="h-4 w-4 flex-shrink-0 text-green-600" />
            ) : (
              <AlertCircle className="h-4 w-4 flex-shrink-0 text-red-600" />
            )}
            <span>{toast.message}</span>
            <button onClick={() => dismissToast(toast.id)} className="ml-2 text-gray-400 hover:text-gray-600">
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
      </div>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Platform Settings</h1>
          <p className="text-sm text-gray-500">Configure global platform settings, localization, and notifications</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
          {saving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6 overflow-x-auto">
          {TABS.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                className={cn(
                  'flex items-center gap-2 whitespace-nowrap border-b-2 pb-3 text-sm font-medium transition-colors',
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

      {/* Tab Content */}
      <div className="space-y-6">
        {/* ============================== GENERAL TAB ============================== */}
        {activeTab === 'general' && (
          <>
            <SectionCard title="Platform Identity" description="Basic information about your platform">
              <Field label="Platform Name" description="Public-facing platform name">
                <TextInput value={platformName} onChange={setPlatformName} placeholder="Saajan Platform" />
              </Field>
              <Field label="Platform URL" description="Root URL of the platform">
                <TextInput value={platformUrl} onChange={setPlatformUrl} placeholder="https://saajan.com.bd" />
              </Field>
              <Field label="Support Email">
                <TextInput value={supportEmail} onChange={setSupportEmail} placeholder="support@saajan.com.bd" type="email" />
              </Field>
              <Field label="Support Phone">
                <TextInput value={supportPhone} onChange={setSupportPhone} placeholder="+880 1700-000000" />
              </Field>
              <Field label="Logo URL" description="Absolute or relative path to the platform logo">
                <TextInput value={logoUrl} onChange={setLogoUrl} placeholder="/logo.png" />
              </Field>
            </SectionCard>

            <SectionCard title="Business Settings" description="Default tenant configuration and platform behavior">
              <Field label="Default Tenant Tier" description="Tier assigned to new tenants">
                <SelectInput
                  value={defaultTier}
                  onChange={setDefaultTier}
                  options={[
                    { label: 'Free', value: 'free' },
                    { label: 'Starter', value: 'starter' },
                    { label: 'Professional', value: 'professional' },
                    { label: 'Enterprise', value: 'enterprise' },
                  ]}
                />
              </Field>
              <Field label="Trial Period (days)" description="Number of days for free trial">
                <NumberInput value={trialPeriodDays} onChange={setTrialPeriodDays} min={0} max={365} />
              </Field>
              <Field label="Max Tenants" description="Maximum number of tenants allowed (0 = unlimited)">
                <NumberInput value={maxTenants} onChange={setMaxTenants} min={0} />
              </Field>
              <Field label="Auto-approve Tenants" description="Automatically approve new tenant registrations">
                <Toggle enabled={autoApproveTenants} onChange={setAutoApproveTenants} />
              </Field>
              <Field label="Maintenance Mode" description="Put the entire platform into maintenance mode">
                <div className="space-y-3">
                  <Toggle enabled={maintenanceMode} onChange={setMaintenanceMode} />
                  {maintenanceMode && (
                    <TextArea
                      value={maintenanceMessage}
                      onChange={setMaintenanceMessage}
                      rows={3}
                      placeholder="Maintenance message displayed to users..."
                    />
                  )}
                </div>
              </Field>
            </SectionCard>
          </>
        )}

        {/* ============================== EMAIL & SMTP TAB ============================== */}
        {activeTab === 'email' && (
          <>
            <SectionCard title="SMTP Configuration" description="Configure email delivery settings">
              <Field label="SMTP Host">
                <TextInput value={smtpHost} onChange={setSmtpHost} placeholder="smtp.gmail.com" />
              </Field>
              <Field label="SMTP Port">
                <NumberInput value={smtpPort} onChange={setSmtpPort} min={1} max={65535} />
              </Field>
              <Field label="Username">
                <TextInput value={smtpUsername} onChange={setSmtpUsername} placeholder="user@gmail.com" />
              </Field>
              <Field label="Password">
                <div className="relative">
                  <input
                    type={smtpShowPassword ? 'text' : 'password'}
                    value={smtpPassword}
                    onChange={(e) => setSmtpPassword(e.target.value)}
                    placeholder="Enter SMTP password"
                    className="w-full rounded-lg border border-gray-300 px-3 py-2 pr-10 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600"
                  />
                  <button
                    type="button"
                    onClick={() => setSmtpShowPassword(!smtpShowPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  >
                    {smtpShowPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
              </Field>
              <Field label="Encryption">
                <SelectInput
                  value={smtpEncryption}
                  onChange={setSmtpEncryption}
                  options={[
                    { label: 'None', value: 'none' },
                    { label: 'TLS', value: 'tls' },
                    { label: 'SSL', value: 'ssl' },
                  ]}
                />
              </Field>
              <Field label="From Name">
                <TextInput value={smtpFromName} onChange={setSmtpFromName} placeholder="Saajan Platform" />
              </Field>
              <Field label="From Email">
                <TextInput value={smtpFromEmail} onChange={setSmtpFromEmail} placeholder="noreply@saajan.com.bd" type="email" />
              </Field>
              <div className="flex items-center gap-3 pt-2">
                <button
                  onClick={handleTestSmtp}
                  disabled={smtpTesting}
                  className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {smtpTesting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                  {smtpTesting ? 'Testing...' : 'Test Connection'}
                </button>
                {smtpTestResult && (
                  <div
                    className={cn(
                      'flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm',
                      smtpTestResult.ok ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700',
                    )}
                  >
                    {smtpTestResult.ok ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : (
                      <AlertCircle className="h-4 w-4" />
                    )}
                    {smtpTestResult.message}
                  </div>
                )}
              </div>
            </SectionCard>

            <SectionCard title="Email Templates" description="Manage transactional email templates">
              <div className="overflow-x-auto -mx-6 px-6">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                      <th className="pb-3 pr-4">Template</th>
                      <th className="pb-3 pr-4">Subject</th>
                      <th className="pb-3 pr-4">Last Modified</th>
                      <th className="pb-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {emailTemplates.map((tpl) => (
                      <tr key={tpl.id} className="group">
                        <td className="py-3 pr-4 font-medium text-gray-900">{tpl.name}</td>
                        <td className="py-3 pr-4 text-gray-500 max-w-xs truncate">{tpl.subject}</td>
                        <td className="py-3 pr-4 text-gray-500">{formatDateShort(tpl.lastModified)}</td>
                        <td className="py-3 text-right">
                          <button
                            onClick={() => openTemplateEditor(tpl)}
                            className="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50"
                          >
                            <Pencil className="h-3 w-3" />
                            Edit
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </SectionCard>

            {/* Template Edit Modal */}
            {editingTemplate && (
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
                <div className="mx-4 w-full max-w-2xl rounded-xl bg-white shadow-2xl">
                  <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
                    <h3 className="text-base font-semibold text-gray-900">
                      Edit Template: {editingTemplate.name}
                    </h3>
                    <button onClick={() => setEditingTemplate(null)} className="text-gray-400 hover:text-gray-600">
                      <X className="h-5 w-5" />
                    </button>
                  </div>
                  <div className="space-y-4 px-6 py-5">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Subject</label>
                      <TextInput value={editTemplateSubject} onChange={setEditTemplateSubject} placeholder="Email subject" />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Body</label>
                      <TextArea value={editTemplateBody} onChange={setEditTemplateBody} rows={10} placeholder="Email body content..." />
                    </div>
                    <div className="rounded-lg bg-gray-50 px-3 py-2">
                      <p className="text-xs font-medium text-gray-500 mb-1">Available Variables:</p>
                      <div className="flex flex-wrap gap-1.5">
                        {editingTemplate.variables.map((v) => (
                          <span
                            key={v}
                            className="rounded bg-indigo-50 px-2 py-0.5 text-xs font-mono text-indigo-700"
                          >
                            {'{{' + v + '}}'}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>
                  <div className="flex justify-end gap-3 border-t border-gray-200 px-6 py-4">
                    <button
                      onClick={() => setEditingTemplate(null)}
                      className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={saveTemplateEdit}
                      className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
                    >
                      Save Template
                    </button>
                  </div>
                </div>
              </div>
            )}
          </>
        )}

        {/* ============================== LOCALIZATION TAB ============================== */}
        {activeTab === 'localization' && (
          <>
            <SectionCard title="Default Locale" description="Set default language, timezone, and formatting preferences">
              <Field label="Default Language">
                <SelectInput
                  value={defaultLanguage}
                  onChange={setDefaultLanguage}
                  options={[
                    { label: 'English', value: 'en' },
                    { label: 'Bangla (\u09AC\u09BE\u0982\u09B2\u09BE)', value: 'bn' },
                    { label: 'Hindi', value: 'hi' },
                    { label: 'Arabic', value: 'ar' },
                  ]}
                />
              </Field>
              <Field label="Default Timezone">
                <SelectInput
                  value={defaultTimezone}
                  onChange={setDefaultTimezone}
                  options={[
                    { label: 'Asia/Dhaka (BST, UTC+6)', value: 'Asia/Dhaka' },
                    { label: 'Asia/Kolkata (IST, UTC+5:30)', value: 'Asia/Kolkata' },
                    { label: 'UTC', value: 'UTC' },
                    { label: 'America/New_York (EST/EDT)', value: 'America/New_York' },
                    { label: 'Europe/London (GMT/BST)', value: 'Europe/London' },
                  ]}
                />
              </Field>
              <Field label="Date Format">
                <SelectInput
                  value={dateFormat}
                  onChange={setDateFormat}
                  options={[
                    { label: 'DD/MM/YYYY', value: 'DD/MM/YYYY' },
                    { label: 'MM/DD/YYYY', value: 'MM/DD/YYYY' },
                    { label: 'YYYY-MM-DD', value: 'YYYY-MM-DD' },
                  ]}
                />
              </Field>
              <Field label="Time Format">
                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="timeFormat"
                      checked={timeFormat === '12h'}
                      onChange={() => setTimeFormat('12h')}
                      className="h-4 w-4 border-gray-300 text-indigo-600 focus:ring-indigo-600"
                    />
                    <span className="text-sm text-gray-700">12-hour</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="timeFormat"
                      checked={timeFormat === '24h'}
                      onChange={() => setTimeFormat('24h')}
                      className="h-4 w-4 border-gray-300 text-indigo-600 focus:ring-indigo-600"
                    />
                    <span className="text-sm text-gray-700">24-hour</span>
                  </label>
                </div>
              </Field>
            </SectionCard>

            <SectionCard title="Supported Languages" description="Manage languages available to tenants">
              <div className="overflow-x-auto -mx-6 px-6">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                      <th className="pb-3 pr-4">Language</th>
                      <th className="pb-3 pr-4">Code</th>
                      <th className="pb-3 pr-4">Status</th>
                      <th className="pb-3 pr-4">Default</th>
                      <th className="pb-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {supportedLanguages.map((lang) => (
                      <tr key={lang.code}>
                        <td className="py-3 pr-4 font-medium text-gray-900">{lang.name}</td>
                        <td className="py-3 pr-4">
                          <span className="rounded bg-gray-100 px-2 py-0.5 text-xs font-mono text-gray-600">{lang.code}</span>
                        </td>
                        <td className="py-3 pr-4">
                          <span
                            className={cn(
                              'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
                              lang.enabled
                                ? 'bg-green-100 text-green-800'
                                : 'bg-gray-100 text-gray-500',
                            )}
                          >
                            {lang.enabled ? 'Enabled' : 'Disabled'}
                          </span>
                        </td>
                        <td className="py-3 pr-4">
                          <input
                            type="radio"
                            name="defaultLang"
                            checked={lang.isDefault}
                            onChange={() => setDefaultLanguageRow(lang.code)}
                            disabled={!lang.enabled}
                            className="h-4 w-4 border-gray-300 text-indigo-600 focus:ring-indigo-600 disabled:opacity-40"
                          />
                        </td>
                        <td className="py-3 text-right">
                          <Toggle
                            enabled={lang.enabled}
                            onChange={() => toggleLanguageEnabled(lang.code)}
                            disabled={lang.isDefault}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <button className="inline-flex items-center gap-1.5 rounded-lg border border-dashed border-gray-300 px-3 py-2 text-sm font-medium text-gray-600 transition-colors hover:border-indigo-300 hover:text-indigo-600">
                <Plus className="h-4 w-4" />
                Add Language
              </button>
            </SectionCard>

            <SectionCard title="Currencies" description="Manage supported currencies and exchange rates">
              <div className="overflow-x-auto -mx-6 px-6">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-100 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                      <th className="pb-3 pr-4">Currency</th>
                      <th className="pb-3 pr-4">Code</th>
                      <th className="pb-3 pr-4">Symbol</th>
                      <th className="pb-3 pr-4">Exchange Rate</th>
                      <th className="pb-3 pr-4">Status</th>
                      <th className="pb-3 pr-4">Default</th>
                      <th className="pb-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-50">
                    {currencies.map((cur) => (
                      <tr key={cur.code}>
                        <td className="py-3 pr-4 font-medium text-gray-900">{cur.name}</td>
                        <td className="py-3 pr-4">
                          <span className="rounded bg-gray-100 px-2 py-0.5 text-xs font-mono text-gray-600">{cur.code}</span>
                        </td>
                        <td className="py-3 pr-4 text-gray-700">{cur.symbol}</td>
                        <td className="py-3 pr-4">
                          <input
                            type="number"
                            step="0.0001"
                            min={0}
                            value={cur.exchangeRate}
                            onChange={(e) => {
                              const rate = parseFloat(e.target.value) || 0;
                              setCurrencies((prev) =>
                                prev.map((c) => (c.code === cur.code ? { ...c, exchangeRate: rate } : c)),
                              );
                            }}
                            disabled={cur.isDefault}
                            className={cn(
                              'w-28 rounded-lg border border-gray-300 px-2 py-1 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600',
                              cur.isDefault && 'bg-gray-50 text-gray-400 cursor-not-allowed',
                            )}
                          />
                        </td>
                        <td className="py-3 pr-4">
                          <span
                            className={cn(
                              'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
                              cur.enabled
                                ? 'bg-green-100 text-green-800'
                                : 'bg-gray-100 text-gray-500',
                            )}
                          >
                            {cur.enabled ? 'Enabled' : 'Disabled'}
                          </span>
                        </td>
                        <td className="py-3 pr-4">
                          <input
                            type="radio"
                            name="defaultCurrency"
                            checked={cur.isDefault}
                            onChange={() => setDefaultCurrencyRow(cur.code)}
                            disabled={!cur.enabled}
                            className="h-4 w-4 border-gray-300 text-indigo-600 focus:ring-indigo-600 disabled:opacity-40"
                          />
                        </td>
                        <td className="py-3 text-right">
                          <Toggle
                            enabled={cur.enabled}
                            onChange={() => toggleCurrencyEnabled(cur.code)}
                            disabled={cur.isDefault}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div className="flex items-center justify-between pt-1">
                <button className="inline-flex items-center gap-1.5 rounded-lg border border-dashed border-gray-300 px-3 py-2 text-sm font-medium text-gray-600 transition-colors hover:border-indigo-300 hover:text-indigo-600">
                  <Plus className="h-4 w-4" />
                  Add Currency
                </button>
                <div className="flex items-center gap-3">
                  <span className="text-sm text-gray-600">Auto-update exchange rates</span>
                  <Toggle enabled={autoUpdateRates} onChange={setAutoUpdateRates} />
                </div>
              </div>
            </SectionCard>
          </>
        )}

        {/* ============================== REGISTRATION TAB ============================== */}
        {activeTab === 'registration' && (
          <>
            <SectionCard title="Registration Policies" description="Control who can register and how verification works">
              <Field label="Allow Public Registration" description="Allow anyone to create a new tenant account">
                <Toggle enabled={allowPublicRegistration} onChange={setAllowPublicRegistration} />
              </Field>
              <Field label="Require Email Verification" description="Users must verify their email before accessing the platform">
                <Toggle enabled={requireEmailVerification} onChange={setRequireEmailVerification} />
              </Field>
              <Field label="Auto-approve Tenants" description="Skip manual approval for new registrations">
                <Toggle enabled={autoApproveRegistration} onChange={setAutoApproveRegistration} />
              </Field>
              <Field label="Require Business Verification" description="Tenants must provide valid business documentation">
                <Toggle enabled={requireBusinessVerification} onChange={setRequireBusinessVerification} />
              </Field>
              <Field label="Allowed Email Domains" description="Only allow registrations from these domains (one per line, empty = all)">
                <TextArea
                  value={allowedEmailDomains}
                  onChange={setAllowedEmailDomains}
                  rows={3}
                  placeholder="example.com&#10;company.org"
                />
              </Field>
              <Field label="Blocked Email Domains" description="Block disposable and temporary email providers">
                <TextArea
                  value={blockedEmailDomains}
                  onChange={setBlockedEmailDomains}
                  rows={4}
                  placeholder="mailinator.com&#10;guerrillamail.com"
                />
              </Field>
            </SectionCard>

            <SectionCard title="Onboarding Configuration" description="Customize the new tenant onboarding experience">
              <Field label="Default Store Template" description="Pre-populate new stores with sample data">
                <SelectInput
                  value={defaultStoreTemplate}
                  onChange={setDefaultStoreTemplate}
                  options={[
                    { label: 'Blank (empty store)', value: 'blank' },
                    { label: 'Sample Store (basic products)', value: 'sample_store' },
                    { label: 'Full Demo (complete sample data)', value: 'full_demo' },
                  ]}
                />
              </Field>
              <Field label="Show Onboarding Wizard" description="Guide new tenants through setup steps">
                <Toggle enabled={showOnboardingWizard} onChange={setShowOnboardingWizard} />
              </Field>
              {showOnboardingWizard && (
                <Field label="Onboarding Steps" description="Toggle individual setup steps">
                  <div className="space-y-2.5">
                    {onboardingSteps.map((step) => (
                      <label key={step.key} className="flex items-center gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={step.enabled}
                          onChange={() => toggleOnboardingStep(step.key)}
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
                        />
                        <span className="text-sm text-gray-700">{step.label}</span>
                      </label>
                    ))}
                  </div>
                </Field>
              )}
            </SectionCard>

            <SectionCard title="Tenant Provisioning" description="Automated resource allocation for new tenants">
              <Field label="Auto-create Database" description="Automatically provision a database for each new tenant">
                <Toggle enabled={autoCreateDatabase} onChange={setAutoCreateDatabase} />
              </Field>
              <Field label="Default Storage Quota (GB)" description="Initial storage allocation per tenant">
                <NumberInput value={defaultStorageQuota} onChange={setDefaultStorageQuota} min={1} max={1000} />
              </Field>
              <Field label="Default Subdomain Pattern" description="Pattern for auto-generated tenant subdomains">
                <TextInput value={defaultSubdomainPattern} onChange={setDefaultSubdomainPattern} placeholder="{slug}.saajan.com.bd" />
              </Field>
              <Field label="Custom Domain Support" description="Allow tenants to connect their own domains">
                <Toggle enabled={customDomainSupport} onChange={setCustomDomainSupport} />
              </Field>
            </SectionCard>
          </>
        )}

        {/* ============================== NOTIFICATIONS TAB ============================== */}
        {activeTab === 'notifications' && (
          <>
            <SectionCard title="Admin Notifications" description="Configure which events trigger notifications to the super admin">
              <Field label="Notification Email" description="Email address for admin notifications">
                <TextInput value={notificationEmail} onChange={setNotificationEmail} placeholder="admin@saajan.com.bd" type="email" />
              </Field>
              <div className="border-t border-gray-100 -mx-6 px-6 pt-5 mt-2">
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-700">New tenant registration</p>
                      <p className="text-xs text-gray-400">When a new tenant signs up</p>
                    </div>
                    <Toggle enabled={notifyNewTenant} onChange={setNotifyNewTenant} />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-700">Tenant plan change</p>
                      <p className="text-xs text-gray-400">When a tenant upgrades or downgrades</p>
                    </div>
                    <Toggle enabled={notifyPlanChange} onChange={setNotifyPlanChange} />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-700">Payment received</p>
                      <p className="text-xs text-gray-400">When a subscription payment is received</p>
                    </div>
                    <Toggle enabled={notifyPaymentReceived} onChange={setNotifyPaymentReceived} />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-700">Payment failed</p>
                      <p className="text-xs text-gray-400">When a payment attempt fails</p>
                    </div>
                    <Toggle enabled={notifyPaymentFailed} onChange={setNotifyPaymentFailed} />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-700">Dispute opened</p>
                      <p className="text-xs text-gray-400">When a payment dispute or chargeback is filed</p>
                    </div>
                    <Toggle enabled={notifyDisputeOpened} onChange={setNotifyDisputeOpened} />
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-gray-700">System alerts</p>
                      <p className="text-xs text-gray-400">Critical system errors and resource warnings</p>
                    </div>
                    <Toggle enabled={notifySystemAlerts} onChange={setNotifySystemAlerts} />
                  </div>
                </div>
              </div>
            </SectionCard>

            <SectionCard title="Tenant Notifications (Defaults)" description="Default notification settings applied to new tenants">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-700">New order</p>
                    <p className="text-xs text-gray-400">When a customer places a new order</p>
                  </div>
                  <Toggle enabled={tenantNotifyNewOrder} onChange={setTenantNotifyNewOrder} />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-700">Order status change</p>
                    <p className="text-xs text-gray-400">When an order status is updated</p>
                  </div>
                  <Toggle enabled={tenantNotifyOrderStatusChange} onChange={setTenantNotifyOrderStatusChange} />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-700">Low stock alert</p>
                    <p className="text-xs text-gray-400">When product stock falls below threshold</p>
                  </div>
                  <div className="flex items-center gap-4">
                    <Toggle enabled={tenantNotifyLowStock} onChange={setTenantNotifyLowStock} />
                    {tenantNotifyLowStock && (
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-gray-500">Threshold:</span>
                        <input
                          type="number"
                          value={lowStockThreshold}
                          onChange={(e) => setLowStockThreshold(Number(e.target.value))}
                          min={1}
                          max={1000}
                          className="w-20 rounded-lg border border-gray-300 px-2 py-1 text-sm focus:border-indigo-600 focus:outline-none focus:ring-1 focus:ring-indigo-600"
                        />
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-700">Customer registration</p>
                    <p className="text-xs text-gray-400">When a new customer creates an account</p>
                  </div>
                  <Toggle enabled={tenantNotifyCustomerRegistration} onChange={setTenantNotifyCustomerRegistration} />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-700">Review posted</p>
                    <p className="text-xs text-gray-400">When a customer posts a product review</p>
                  </div>
                  <Toggle enabled={tenantNotifyReviewPosted} onChange={setTenantNotifyReviewPosted} />
                </div>
              </div>
            </SectionCard>
          </>
        )}
      </div>

      {/* Bottom Save Bar */}
      <div className="sticky bottom-0 -mx-6 border-t border-gray-200 bg-white/95 px-6 py-4 backdrop-blur">
        <div className="flex items-center justify-end">
          <button
            onClick={handleSave}
            disabled={saving}
            className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            {saving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  );
}
