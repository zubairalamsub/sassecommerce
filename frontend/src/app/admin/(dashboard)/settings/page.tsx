'use client';

import { useState, useEffect, useRef } from 'react';
import {
  Save, Loader2, Store, Palette, Globe, ToggleLeft, Layout,
  Plus, X, GripVertical, Image, ChevronUp, ChevronDown, Megaphone,
  CreditCard, Truck, Mail, Shield,
} from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { tenantApi, type TenantConfig } from '@/lib/api';
import { useStoreConfigStore, type BannerSlide, type StoreSection, type StorefrontConfig } from '@/stores/store-config';
import { useDeliveryProfileStore } from '@/stores/delivery-profiles';
import type { DeliveryProfile } from '@/lib/api';
import { cn } from '@/lib/utils';

type SettingsTab = 'general' | 'branding' | 'storefront' | 'features' | 'payment' | 'shipping' | 'email' | 'security';

const tabs: { label: string; value: SettingsTab; icon: React.ElementType }[] = [
  { label: 'General', value: 'general', icon: Globe },
  { label: 'Branding', value: 'branding', icon: Palette },
  { label: 'Storefront', value: 'storefront', icon: Layout },
  { label: 'Features', value: 'features', icon: ToggleLeft },
  { label: 'Payment', value: 'payment', icon: CreditCard },
  { label: 'Shipping', value: 'shipping', icon: Truck },
  { label: 'Email', value: 'email', icon: Mail },
  { label: 'Security', value: 'security', icon: Shield },
];

const FEATURE_LABELS: Record<string, string> = {
  multi_currency: 'Multi-Currency Support',
  wishlist: 'Wishlist',
  product_reviews: 'Product Reviews',
  guest_checkout: 'Guest Checkout',
  social_login: 'Social Login',
  ai_recommendations: 'AI Recommendations',
  loyalty_program: 'Loyalty Program',
  subscriptions: 'Subscriptions',
  gift_cards: 'Gift Cards',
};

const SECTION_TYPES: { value: StoreSection['type']; label: string }[] = [
  { value: 'hot_products', label: 'Hot Products' },
  { value: 'discount', label: 'Discount / Sale' },
  { value: 'new_arrivals', label: 'New Arrivals' },
  { value: 'category_showcase', label: 'Shop by Category' },
  { value: 'campaign', label: 'Campaign Banner' },
  { value: 'custom', label: 'Custom Section' },
];

const defaultTenantConfig: TenantConfig = {
  general: {
    timezone: 'Asia/Dhaka', currency: 'BDT', language: 'en',
    date_format: 'DD/MM/YYYY', time_format: 'HH:mm:ss',
    contact_email: '', contact_phone: '', support_url: '',
  },
  branding: {
    logo_url: '', favicon_url: '', primary_color: '#3b82f6',
    secondary_color: '#10b981', custom_css: '', custom_fonts: {},
  },
  features: {
    multi_currency: false, wishlist: true, product_reviews: true,
    guest_checkout: true, social_login: false, ai_recommendations: false,
    loyalty_program: false, subscriptions: false, gift_cards: false,
  },
};

export default function StoreSettingsPage() {
  const tenantId = useAuthStore((s) => s.tenantId);
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  const [storeName, setStoreName] = useState('');
  const [storeEmail, setStoreEmail] = useState('');
  const [tenantConfig, setTenantConfig] = useState<TenantConfig>(defaultTenantConfig);

  // Payment settings
  const [paymentSettings, setPaymentSettings] = useState({
    sslcommerz_store_id: '',
    sslcommerz_store_password: '',
    sslcommerz_sandbox: true,
    cod_enabled: true,
    bkash_enabled: false,
    bkash_merchant_number: '',
    nagad_enabled: false,
    nagad_merchant_id: '',
    min_order_amount: 0,
  });

  // Shipping settings
  const [shippingSettings, setShippingSettings] = useState({
    free_shipping_enabled: false,
    free_shipping_threshold: 2000,
    flat_rate_enabled: true,
    flat_rate_amount: 100,
    inside_dhaka_rate: 60,
    outside_dhaka_rate: 120,
    estimated_delivery_dhaka: '1-2 days',
    estimated_delivery_outside: '3-5 days',
    shipping_policy: '',
  });

  // Email / Notification settings
  const [emailSettings, setEmailSettings] = useState({
    smtp_host: '',
    smtp_port: '587',
    smtp_username: '',
    smtp_password: '',
    smtp_from_name: '',
    smtp_from_email: '',
    smtp_encryption: 'tls' as 'tls' | 'ssl' | 'none',
    notify_order_placed: true,
    notify_order_confirmed: true,
    notify_order_shipped: true,
    notify_order_delivered: true,
    notify_order_cancelled: true,
    notify_low_stock: true,
    notify_new_customer: false,
  });

  // Security settings
  const [securitySettings, setSecuritySettings] = useState({
    password_min_length: 8,
    password_require_uppercase: true,
    password_require_number: true,
    password_require_special: false,
    two_factor_enabled: false,
    session_timeout_minutes: 60,
    max_login_attempts: 5,
    lockout_duration_minutes: 15,
    api_rate_limit: 100,
  });

  // Storefront config
  const storeConfig = useStoreConfigStore((s) => s.config);
  const fetchStoreConfig = useStoreConfigStore((s) => s.fetchConfig);
  const saveStoreConfig = useStoreConfigStore((s) => s.saveConfig);
  const updateStoreConfig = useStoreConfigStore((s) => s.updateConfig);

  // Delivery profiles
  const deliveryProfiles = useDeliveryProfileStore((s) => s.profiles);
  const fetchDeliveryProfiles = useDeliveryProfileStore((s) => s.fetchProfiles);
  const saveDeliveryProfiles = useDeliveryProfileStore((s) => s.saveProfiles);
  const addDeliveryProfile = useDeliveryProfileStore((s) => s.addProfile);
  const updateDeliveryProfile = useDeliveryProfileStore((s) => s.updateProfile);
  const removeDeliveryProfile = useDeliveryProfileStore((s) => s.removeProfile);
  const lastProfileRef = useRef<HTMLDivElement>(null);
  const [focusProfileId, setFocusProfileId] = useState<string | null>(null);

  useEffect(() => {
    if (!tenantId) return;
    async function load() {
      try {
        const tenant = await tenantApi.get(tenantId!);
        setStoreName(tenant.name);
        setStoreEmail(tenant.email);
        setTenantConfig({
          general: { ...defaultTenantConfig.general, ...tenant.config?.general },
          branding: { ...defaultTenantConfig.branding, ...tenant.config?.branding },
          features: { ...defaultTenantConfig.features, ...tenant.config?.features },
        });
      } catch {
        // Backend unavailable — silently use defaults
      }
      try {
        await fetchStoreConfig(tenantId!);
      } catch {
        // defaults loaded
      }
      try {
        await fetchDeliveryProfiles(tenantId!);
      } catch {
        // defaults loaded
      }
      setLoading(false);
    }
    load();
  }, [tenantId, fetchStoreConfig]);

  useEffect(() => {
    if (focusProfileId && lastProfileRef.current) {
      lastProfileRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' });
      const nameInput = lastProfileRef.current.querySelector<HTMLInputElement>('input[type="text"]');
      if (nameInput) {
        nameInput.focus();
        nameInput.select();
      }
      setFocusProfileId(null);
    }
  }, [focusProfileId, deliveryProfiles]);

  function updateGeneral<K extends keyof TenantConfig['general']>(key: K, value: TenantConfig['general'][K]) {
    setTenantConfig((prev) => ({ ...prev, general: { ...prev.general, [key]: value } }));
    setSaved(false);
  }

  function updateBranding<K extends keyof TenantConfig['branding']>(key: K, value: TenantConfig['branding'][K]) {
    setTenantConfig((prev) => ({ ...prev, branding: { ...prev.branding, [key]: value } }));
    setSaved(false);
  }

  function toggleFeature(key: string) {
    setTenantConfig((prev) => ({
      ...prev,
      features: { ...prev.features, [key]: !prev.features[key] },
    }));
    setSaved(false);
  }

  async function handleSave() {
    if (!tenantId) return;
    setSaving(true);
    setError('');
    try {
      await tenantApi.update(tenantId, { name: storeName, email: storeEmail });
      await tenantApi.updateConfig(tenantId, tenantConfig);
      await saveStoreConfig(tenantId, storeConfig);
      await saveDeliveryProfiles(tenantId);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err) {
      setError((err as Error).message || 'Failed to save settings');
    } finally {
      setSaving(false);
    }
  }

  // ---- Storefront helpers ----
  function addBanner() {
    const banners = [...storeConfig.banners, {
      id: `b-${Date.now()}`, image_url: '', title: 'New Banner',
      subtitle: 'Add your promotional text here', cta_text: 'Shop Now',
      cta_link: '/products', bg_color: '#3b82f6',
    }];
    updateStoreConfig({ banners });
    setSaved(false);
  }

  function updateBanner(id: string, field: keyof BannerSlide, value: string) {
    const banners = storeConfig.banners.map((b) => b.id === id ? { ...b, [field]: value } : b);
    updateStoreConfig({ banners });
    setSaved(false);
  }

  function removeBanner(id: string) {
    updateStoreConfig({ banners: storeConfig.banners.filter((b) => b.id !== id) });
    setSaved(false);
  }

  function addSection() {
    const sections = [...storeConfig.sections, {
      id: `s-${Date.now()}`, type: 'custom' as const, title: 'New Section',
      subtitle: '', enabled: true, position: storeConfig.sections.length + 1, config: {},
    }];
    updateStoreConfig({ sections });
    setSaved(false);
  }

  function updateSection(id: string, updates: Partial<StoreSection>) {
    const sections = storeConfig.sections.map((s) => s.id === id ? { ...s, ...updates } : s);
    updateStoreConfig({ sections });
    setSaved(false);
  }

  function removeSection(id: string) {
    updateStoreConfig({ sections: storeConfig.sections.filter((s) => s.id !== id) });
    setSaved(false);
  }

  function moveSection(id: string, dir: -1 | 1) {
    const sections = [...storeConfig.sections];
    const idx = sections.findIndex((s) => s.id === id);
    if (idx < 0) return;
    const swapIdx = idx + dir;
    if (swapIdx < 0 || swapIdx >= sections.length) return;
    [sections[idx], sections[swapIdx]] = [sections[swapIdx], sections[idx]];
    sections.forEach((s, i) => { s.position = i + 1; });
    updateStoreConfig({ sections });
    setSaved(false);
  }

  function updateFooter<K extends keyof StorefrontConfig['footer']>(key: K, value: StorefrontConfig['footer'][K]) {
    updateStoreConfig({ footer: { ...storeConfig.footer, [key]: value } });
    setSaved(false);
  }

  function updateAbout<K extends keyof StorefrontConfig['about']>(key: K, value: StorefrontConfig['about'][K]) {
    updateStoreConfig({ about: { ...storeConfig.about, [key]: value } });
    setSaved(false);
  }

  function updateAnnouncement<K extends keyof StorefrontConfig['announcement_bar']>(key: K, value: StorefrontConfig['announcement_bar'][K]) {
    updateStoreConfig({ announcement_bar: { ...storeConfig.announcement_bar, [key]: value } });
    setSaved(false);
  }

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Store Settings</h1>
          <p className="text-sm text-gray-500">Manage your store configuration and storefront</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
          {saving ? 'Saving...' : saved ? 'Saved!' : 'Save Changes'}
        </button>
      </div>

      {error && <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>}
      {saved && <div className="rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">Settings saved successfully.</div>}

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6 overflow-x-auto">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                className={cn(
                  'flex items-center gap-2 border-b-2 pb-3 text-sm font-medium transition-colors whitespace-nowrap',
                  activeTab === tab.value
                    ? 'border-primary text-primary'
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

      {/* ==================== GENERAL TAB ==================== */}
      {activeTab === 'general' && (
        <div className="space-y-6">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center gap-2">
              <Store className="h-5 w-5 text-gray-400" />
              <h2 className="text-lg font-semibold text-gray-900">Store Information</h2>
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="Store Name" value={storeName} onChange={(v) => { setStoreName(v); setSaved(false); }} />
              <Field label="Store Email" type="email" value={storeEmail} onChange={(v) => { setStoreEmail(v); setSaved(false); }} />
              <Field label="Contact Phone" type="tel" value={tenantConfig.general.contact_phone} onChange={(v) => updateGeneral('contact_phone', v)} placeholder="+880 1700-000000" />
              <Field label="Support URL" type="url" value={tenantConfig.general.support_url} onChange={(v) => updateGeneral('support_url', v)} placeholder="https://support.yourstore.com" />
            </div>
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center gap-2">
              <Globe className="h-5 w-5 text-gray-400" />
              <h2 className="text-lg font-semibold text-gray-900">Locale & Regional</h2>
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <SelectField label="Timezone" value={tenantConfig.general.timezone} onChange={(v) => updateGeneral('timezone', v)}
                options={[
                  { value: 'Asia/Dhaka', label: 'Asia/Dhaka (BST +06:00)' },
                  { value: 'Asia/Kolkata', label: 'Asia/Kolkata (IST +05:30)' },
                  { value: 'UTC', label: 'UTC' },
                  { value: 'America/New_York', label: 'America/New_York (EST)' },
                  { value: 'Europe/London', label: 'Europe/London (GMT)' },
                ]}
              />
              <SelectField label="Currency" value={tenantConfig.general.currency} onChange={(v) => updateGeneral('currency', v)}
                options={[
                  { value: 'BDT', label: 'BDT - Bangladeshi Taka' },
                  { value: 'USD', label: 'USD - US Dollar' },
                  { value: 'EUR', label: 'EUR - Euro' },
                  { value: 'GBP', label: 'GBP - British Pound' },
                  { value: 'INR', label: 'INR - Indian Rupee' },
                ]}
              />
              <SelectField label="Language" value={tenantConfig.general.language} onChange={(v) => updateGeneral('language', v)}
                options={[{ value: 'en', label: 'English' }, { value: 'bn', label: 'Bangla' }]}
              />
              <SelectField label="Date Format" value={tenantConfig.general.date_format} onChange={(v) => updateGeneral('date_format', v)}
                options={[
                  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
                  { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY' },
                  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD' },
                ]}
              />
              <SelectField label="Time Format" value={tenantConfig.general.time_format} onChange={(v) => updateGeneral('time_format', v)}
                options={[
                  { value: 'HH:mm:ss', label: '24-hour (HH:mm:ss)' },
                  { value: 'hh:mm A', label: '12-hour (hh:mm AM/PM)' },
                ]}
              />
            </div>
          </div>
        </div>
      )}

      {/* ==================== BRANDING TAB ==================== */}
      {activeTab === 'branding' && (
        <div className="space-y-6">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Brand Identity</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="Logo URL" type="url" value={tenantConfig.branding.logo_url} onChange={(v) => updateBranding('logo_url', v)} placeholder="https://cdn.yourstore.com/logo.png" />
              <Field label="Favicon URL" type="url" value={tenantConfig.branding.favicon_url} onChange={(v) => updateBranding('favicon_url', v)} placeholder="https://cdn.yourstore.com/favicon.ico" />
            </div>
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Colors</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <ColorField label="Primary Color" value={tenantConfig.branding.primary_color || '#3b82f6'} onChange={(v) => updateBranding('primary_color', v)} />
              <ColorField label="Secondary Color" value={tenantConfig.branding.secondary_color || '#10b981'} onChange={(v) => updateBranding('secondary_color', v)} />
            </div>
            <div className="mt-6">
              <label className="mb-2 block text-sm font-medium text-gray-700">Preview</label>
              <div className="flex items-center gap-4 rounded-lg border border-gray-200 bg-gray-50 p-4">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg text-white text-sm font-bold" style={{ backgroundColor: tenantConfig.branding.primary_color || '#3b82f6' }}>
                  {storeName?.[0] || 'S'}
                </div>
                <div>
                  <p className="text-sm font-semibold" style={{ color: tenantConfig.branding.primary_color || '#3b82f6' }}>{storeName || 'Store Name'}</p>
                  <p className="text-xs" style={{ color: tenantConfig.branding.secondary_color || '#10b981' }}>Secondary text</p>
                </div>
                <div className="ml-auto flex gap-2">
                  <button type="button" className="rounded-lg px-4 py-2 text-xs font-medium text-white" style={{ backgroundColor: tenantConfig.branding.primary_color || '#3b82f6' }}>Primary</button>
                  <button type="button" className="rounded-lg px-4 py-2 text-xs font-medium text-white" style={{ backgroundColor: tenantConfig.branding.secondary_color || '#10b981' }}>Secondary</button>
                </div>
              </div>
            </div>
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Custom CSS</h2>
            <textarea rows={6} value={tenantConfig.branding.custom_css} onChange={(e) => updateBranding('custom_css', e.target.value)}
              placeholder="/* Add custom CSS overrides here */"
              className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 font-mono text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
        </div>
      )}

      {/* ==================== STOREFRONT TAB ==================== */}
      {activeTab === 'storefront' && (
        <div className="space-y-6">
          {/* Announcement Bar */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Megaphone className="h-5 w-5 text-gray-400" />
                <h2 className="text-lg font-semibold text-gray-900">Announcement Bar</h2>
              </div>
              <button type="button" onClick={() => updateAnnouncement('enabled', !storeConfig.announcement_bar.enabled)}
                className={cn('relative inline-flex h-6 w-11 items-center rounded-full transition-colors', storeConfig.announcement_bar.enabled ? 'bg-primary' : 'bg-gray-200')}>
                <span className={cn('inline-block h-4 w-4 transform rounded-full bg-white transition-transform', storeConfig.announcement_bar.enabled ? 'translate-x-6' : 'translate-x-1')} />
              </button>
            </div>
            {storeConfig.announcement_bar.enabled && (
              <div className="space-y-4">
                <Field label="Announcement Text" value={storeConfig.announcement_bar.text} onChange={(v) => updateAnnouncement('text', v)} placeholder="Free shipping on orders over BDT 2,000!" />
                <div className="grid grid-cols-2 gap-4">
                  <ColorField label="Background Color" value={storeConfig.announcement_bar.bg_color} onChange={(v) => updateAnnouncement('bg_color', v)} />
                  <ColorField label="Text Color" value={storeConfig.announcement_bar.text_color} onChange={(v) => updateAnnouncement('text_color', v)} />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-gray-700">Preview</label>
                  <div className="rounded-lg px-4 py-2 text-center text-sm font-medium" style={{ backgroundColor: storeConfig.announcement_bar.bg_color, color: storeConfig.announcement_bar.text_color }}>
                    {storeConfig.announcement_bar.text || 'Announcement text here'}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Hero Banners */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Image className="h-5 w-5 text-gray-400" />
                <h2 className="text-lg font-semibold text-gray-900">Hero Banners</h2>
                <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">{storeConfig.banners.length}</span>
              </div>
              <button type="button" onClick={addBanner}
                className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50">
                <Plus className="h-4 w-4" /> Add Banner
              </button>
            </div>
            <div className="space-y-4">
              {storeConfig.banners.map((banner, idx) => (
                <div key={banner.id} className="rounded-lg border border-gray-200 p-4">
                  <div className="mb-3 flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-700">Banner {idx + 1}</span>
                    <button type="button" onClick={() => removeBanner(banner.id)} className="rounded-lg p-1 text-gray-400 hover:bg-red-50 hover:text-red-600">
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <Field label="Title" value={banner.title} onChange={(v) => updateBanner(banner.id, 'title', v)} />
                    <Field label="Subtitle" value={banner.subtitle} onChange={(v) => updateBanner(banner.id, 'subtitle', v)} />
                    <Field label="Image URL" type="url" value={banner.image_url} onChange={(v) => updateBanner(banner.id, 'image_url', v)} placeholder="https://..." />
                    <ColorField label="Background Color" value={banner.bg_color} onChange={(v) => updateBanner(banner.id, 'bg_color', v)} />
                    <Field label="Button Text" value={banner.cta_text} onChange={(v) => updateBanner(banner.id, 'cta_text', v)} />
                    <Field label="Button Link" value={banner.cta_link} onChange={(v) => updateBanner(banner.id, 'cta_link', v)} placeholder="/products" />
                  </div>
                  {/* Mini preview */}
                  <div className="mt-3 rounded-lg p-4" style={{ backgroundColor: banner.bg_color }}>
                    <p className="text-lg font-bold text-white">{banner.title || 'Banner Title'}</p>
                    <p className="text-sm text-white/80">{banner.subtitle || 'Subtitle text'}</p>
                    <span className="mt-2 inline-block rounded-lg bg-white px-3 py-1 text-xs font-semibold" style={{ color: banner.bg_color }}>
                      {banner.cta_text || 'Shop Now'}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Page Sections */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Layout className="h-5 w-5 text-gray-400" />
                <h2 className="text-lg font-semibold text-gray-900">Homepage Sections</h2>
              </div>
              <button type="button" onClick={addSection}
                className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50">
                <Plus className="h-4 w-4" /> Add Section
              </button>
            </div>
            <p className="mb-4 text-xs text-gray-500">Drag sections to reorder how they appear on your homepage. Toggle to enable/disable.</p>
            <div className="space-y-3">
              {storeConfig.sections.sort((a, b) => a.position - b.position).map((section, idx) => (
                <div key={section.id} className={cn('rounded-lg border p-4 transition-colors', section.enabled ? 'border-gray-200 bg-white' : 'border-gray-100 bg-gray-50 opacity-60')}>
                  <div className="flex items-center gap-3">
                    <div className="flex flex-col gap-0.5">
                      <button type="button" onClick={() => moveSection(section.id, -1)} disabled={idx === 0} className="rounded p-0.5 text-gray-400 hover:text-gray-600 disabled:opacity-30">
                        <ChevronUp className="h-3.5 w-3.5" />
                      </button>
                      <button type="button" onClick={() => moveSection(section.id, 1)} disabled={idx === storeConfig.sections.length - 1} className="rounded p-0.5 text-gray-400 hover:text-gray-600 disabled:opacity-30">
                        <ChevronDown className="h-3.5 w-3.5" />
                      </button>
                    </div>
                    <GripVertical className="h-4 w-4 text-gray-300" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <input type="text" value={section.title} onChange={(e) => updateSection(section.id, { title: e.target.value })}
                          className="rounded border-transparent bg-transparent px-1 py-0.5 text-sm font-medium text-gray-900 hover:border-gray-300 hover:bg-white focus:border-primary focus:bg-white focus:outline-none focus:ring-1 focus:ring-primary"
                        />
                        <span className="rounded bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-500">
                          {SECTION_TYPES.find((t) => t.value === section.type)?.label || section.type}
                        </span>
                      </div>
                      <input type="text" value={section.subtitle} onChange={(e) => updateSection(section.id, { subtitle: e.target.value })}
                        placeholder="Section subtitle"
                        className="mt-0.5 w-full rounded border-transparent bg-transparent px-1 py-0.5 text-xs text-gray-500 hover:border-gray-300 hover:bg-white focus:border-primary focus:bg-white focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    <select value={section.type} onChange={(e) => updateSection(section.id, { type: e.target.value as StoreSection['type'] })}
                      className="rounded-lg border border-gray-300 px-2 py-1 text-xs focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary">
                      {SECTION_TYPES.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
                    </select>
                    <button type="button" onClick={() => updateSection(section.id, { enabled: !section.enabled })}
                      className={cn('relative inline-flex h-5 w-9 items-center rounded-full transition-colors', section.enabled ? 'bg-primary' : 'bg-gray-200')}>
                      <span className={cn('inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform', section.enabled ? 'translate-x-4.5' : 'translate-x-0.5')} />
                    </button>
                    <button type="button" onClick={() => removeSection(section.id)} className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600">
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
              {storeConfig.sections.length === 0 && (
                <div className="rounded-lg border-2 border-dashed border-gray-300 py-8 text-center text-sm text-gray-500">
                  No sections configured. Add a section to get started.
                </div>
              )}
            </div>
          </div>

          {/* Footer */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Footer</h2>
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">About Text</label>
                <textarea rows={3} value={storeConfig.footer.about_text} onChange={(e) => updateFooter('about_text', e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <Field label="Contact Email" type="email" value={storeConfig.footer.contact_email} onChange={(v) => updateFooter('contact_email', v)} />
                <Field label="Contact Phone" value={storeConfig.footer.contact_phone} onChange={(v) => updateFooter('contact_phone', v)} />
                <Field label="Address" value={storeConfig.footer.contact_address} onChange={(v) => updateFooter('contact_address', v)} />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <Field label="Facebook URL" type="url" value={storeConfig.footer.social_facebook} onChange={(v) => updateFooter('social_facebook', v)} placeholder="https://facebook.com/..." />
                <Field label="Instagram URL" type="url" value={storeConfig.footer.social_instagram} onChange={(v) => updateFooter('social_instagram', v)} placeholder="https://instagram.com/..." />
                <Field label="YouTube URL" type="url" value={storeConfig.footer.social_youtube} onChange={(v) => updateFooter('social_youtube', v)} placeholder="https://youtube.com/..." />
              </div>
              <Field label="Copyright Text" value={storeConfig.footer.copyright_text} onChange={(v) => updateFooter('copyright_text', v)} />
            </div>
          </div>

          {/* About Page */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">About Page</h2>
            <div className="space-y-4">
              <Field label="Title" value={storeConfig.about.title} onChange={(v) => updateAbout('title', v)} />
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">Content</label>
                <textarea rows={4} value={storeConfig.about.content} onChange={(e) => updateAbout('content', e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field label="Mission" value={storeConfig.about.mission} onChange={(v) => updateAbout('mission', v)} />
                <Field label="Vision" value={storeConfig.about.vision} onChange={(v) => updateAbout('vision', v)} />
              </div>
              <Field label="Image URL" type="url" value={storeConfig.about.image_url} onChange={(v) => updateAbout('image_url', v)} placeholder="https://..." />
            </div>
          </div>
        </div>
      )}

      {/* ==================== FEATURES TAB ==================== */}
      {activeTab === 'features' && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-2 text-lg font-semibold text-gray-900">Feature Toggles</h2>
          <p className="mb-6 text-sm text-gray-500">Enable or disable features for your store</p>
          <div className="divide-y divide-gray-100">
            {Object.entries(FEATURE_LABELS).map(([key, label]) => (
              <div key={key} className="flex items-center justify-between py-4">
                <div>
                  <p className="text-sm font-medium text-gray-900">{label}</p>
                  <p className="text-xs text-gray-500">{getFeatureDescription(key)}</p>
                </div>
                <button type="button" onClick={() => toggleFeature(key)}
                  className={cn('relative inline-flex h-6 w-11 items-center rounded-full transition-colors', tenantConfig.features[key] ? 'bg-primary' : 'bg-gray-200')}>
                  <span className={cn('inline-block h-4 w-4 transform rounded-full bg-white transition-transform', tenantConfig.features[key] ? 'translate-x-6' : 'translate-x-1')} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ==================== PAYMENT TAB ==================== */}
      {activeTab === 'payment' && (
        <div className="space-y-6">
          {/* SSLCommerz */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center gap-2">
              <CreditCard className="h-5 w-5 text-gray-400" />
              <h2 className="text-lg font-semibold text-gray-900">SSLCommerz Gateway</h2>
            </div>
            <p className="mb-4 text-sm text-gray-500">Configure your SSLCommerz payment gateway credentials for online payments.</p>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="Store ID" value={paymentSettings.sslcommerz_store_id}
                onChange={(v) => { setPaymentSettings((p) => ({ ...p, sslcommerz_store_id: v })); setSaved(false); }}
                placeholder="your_store_id" />
              <Field label="Store Password" type="password" value={paymentSettings.sslcommerz_store_password}
                onChange={(v) => { setPaymentSettings((p) => ({ ...p, sslcommerz_store_password: v })); setSaved(false); }}
                placeholder="your_store_password" />
            </div>
            <div className="mt-4 flex items-center justify-between rounded-lg border border-gray-100 bg-gray-50 px-4 py-3">
              <div>
                <p className="text-sm font-medium text-gray-900">Sandbox Mode</p>
                <p className="text-xs text-gray-500">Use test credentials for development</p>
              </div>
              <ToggleButton
                enabled={paymentSettings.sslcommerz_sandbox}
                onToggle={() => { setPaymentSettings((p) => ({ ...p, sslcommerz_sandbox: !p.sslcommerz_sandbox })); setSaved(false); }}
              />
            </div>
          </div>

          {/* Payment Methods */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-2 text-lg font-semibold text-gray-900">Payment Methods</h2>
            <p className="mb-4 text-sm text-gray-500">Enable payment methods available to your customers</p>
            <div className="divide-y divide-gray-100">
              <div className="flex items-center justify-between py-4">
                <div>
                  <p className="text-sm font-medium text-gray-900">Cash on Delivery (COD)</p>
                  <p className="text-xs text-gray-500">Customers pay when they receive their order</p>
                </div>
                <ToggleButton
                  enabled={paymentSettings.cod_enabled}
                  onToggle={() => { setPaymentSettings((p) => ({ ...p, cod_enabled: !p.cod_enabled })); setSaved(false); }}
                />
              </div>
              <div className="py-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-900">bKash</p>
                    <p className="text-xs text-gray-500">Mobile financial services payment</p>
                  </div>
                  <ToggleButton
                    enabled={paymentSettings.bkash_enabled}
                    onToggle={() => { setPaymentSettings((p) => ({ ...p, bkash_enabled: !p.bkash_enabled })); setSaved(false); }}
                  />
                </div>
                {paymentSettings.bkash_enabled && (
                  <div className="mt-3 pl-0 sm:pl-4">
                    <Field label="Merchant Number" value={paymentSettings.bkash_merchant_number}
                      onChange={(v) => { setPaymentSettings((p) => ({ ...p, bkash_merchant_number: v })); setSaved(false); }}
                      placeholder="01XXXXXXXXX" />
                  </div>
                )}
              </div>
              <div className="py-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-900">Nagad</p>
                    <p className="text-xs text-gray-500">Digital financial service by Bangladesh Post Office</p>
                  </div>
                  <ToggleButton
                    enabled={paymentSettings.nagad_enabled}
                    onToggle={() => { setPaymentSettings((p) => ({ ...p, nagad_enabled: !p.nagad_enabled })); setSaved(false); }}
                  />
                </div>
                {paymentSettings.nagad_enabled && (
                  <div className="mt-3 pl-0 sm:pl-4">
                    <Field label="Merchant ID" value={paymentSettings.nagad_merchant_id}
                      onChange={(v) => { setPaymentSettings((p) => ({ ...p, nagad_merchant_id: v })); setSaved(false); }}
                      placeholder="Merchant ID" />
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Order Limits */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Order Limits</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">Minimum Order Amount (BDT)</label>
                <input type="number" min={0} value={paymentSettings.min_order_amount}
                  onChange={(e) => { setPaymentSettings((p) => ({ ...p, min_order_amount: Number(e.target.value) })); setSaved(false); }}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  placeholder="0" />
                <p className="mt-1 text-xs text-gray-500">Set to 0 for no minimum</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ==================== SHIPPING TAB ==================== */}
      {activeTab === 'shipping' && (
        <div className="space-y-6">
          {/* Delivery Charge Profiles */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-gray-900">Delivery Charge Profiles</h2>
                <p className="text-sm text-gray-500">Create delivery profiles and assign them to products</p>
              </div>
              <button
                type="button"
                onClick={() => {
                  const newId = `dp-${Date.now()}`;
                  addDeliveryProfile({
                    id: newId,
                    name: 'New Profile',
                    inside_dhaka_rate: 60,
                    outside_dhaka_rate: 120,
                    inside_dhaka_express_rate: 100,
                    outside_dhaka_express_rate: 180,
                    estimated_delivery_dhaka: '1-2 days',
                    estimated_delivery_outside: '3-5 days',
                    is_default: false,
                  });
                  setFocusProfileId(newId);
                  setSaved(false);
                }}
                className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
              >
                <Plus className="h-4 w-4" />
                Add Profile
              </button>
            </div>

            <div className="space-y-4">
              {deliveryProfiles.map((profile) => (
                <div key={profile.id} ref={profile.id === focusProfileId ? lastProfileRef : undefined} className={cn('rounded-lg border p-4', profile.is_default ? 'border-primary bg-primary/5' : 'border-gray-200')}>
                  <div className="mb-3 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <input
                        type="text"
                        value={profile.name}
                        onChange={(e) => { updateDeliveryProfile(profile.id, { name: e.target.value }); setSaved(false); }}
                        className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-semibold focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                      <label className="flex items-center gap-1.5 text-xs text-gray-500 cursor-pointer">
                        <input
                          type="radio"
                          name="defaultProfile"
                          checked={profile.is_default}
                          onChange={() => {
                            deliveryProfiles.forEach((p) => updateDeliveryProfile(p.id, { is_default: p.id === profile.id }));
                            setSaved(false);
                          }}
                          className="h-3.5 w-3.5 text-primary focus:ring-primary"
                        />
                        Default
                      </label>
                    </div>
                    {!profile.is_default && (
                      <button
                        type="button"
                        onClick={() => { removeDeliveryProfile(profile.id); setSaved(false); }}
                        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>

                  <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Dhaka Standard (৳)</label>
                      <input type="number" min={0} value={profile.inside_dhaka_rate}
                        onChange={(e) => { updateDeliveryProfile(profile.id, { inside_dhaka_rate: Number(e.target.value) }); setSaved(false); }}
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Dhaka Express (৳)</label>
                      <input type="number" min={0} value={profile.inside_dhaka_express_rate}
                        onChange={(e) => { updateDeliveryProfile(profile.id, { inside_dhaka_express_rate: Number(e.target.value) }); setSaved(false); }}
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Outside Standard (৳)</label>
                      <input type="number" min={0} value={profile.outside_dhaka_rate}
                        onChange={(e) => { updateDeliveryProfile(profile.id, { outside_dhaka_rate: Number(e.target.value) }); setSaved(false); }}
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Outside Express (৳)</label>
                      <input type="number" min={0} value={profile.outside_dhaka_express_rate}
                        onChange={(e) => { updateDeliveryProfile(profile.id, { outside_dhaka_express_rate: Number(e.target.value) }); setSaved(false); }}
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                    </div>
                  </div>

                  <div className="mt-3 grid grid-cols-2 gap-3">
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Dhaka Delivery Estimate</label>
                      <input type="text" value={profile.estimated_delivery_dhaka}
                        onChange={(e) => { updateDeliveryProfile(profile.id, { estimated_delivery_dhaka: e.target.value }); setSaved(false); }}
                        placeholder="1-2 days"
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Outside Delivery Estimate</label>
                      <input type="text" value={profile.estimated_delivery_outside}
                        onChange={(e) => { updateDeliveryProfile(profile.id, { estimated_delivery_outside: e.target.value }); setSaved(false); }}
                        placeholder="3-5 days"
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Free Shipping */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Truck className="h-5 w-5 text-gray-400" />
                <h2 className="text-lg font-semibold text-gray-900">Free Shipping Threshold</h2>
              </div>
              <ToggleButton
                enabled={shippingSettings.free_shipping_enabled}
                onToggle={() => { setShippingSettings((p) => ({ ...p, free_shipping_enabled: !p.free_shipping_enabled })); setSaved(false); }}
              />
            </div>
            {shippingSettings.free_shipping_enabled && (
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">Free Shipping Threshold (BDT)</label>
                <input type="number" min={0} value={shippingSettings.free_shipping_threshold}
                  onChange={(e) => { setShippingSettings((p) => ({ ...p, free_shipping_threshold: Number(e.target.value) })); setSaved(false); }}
                  className="w-full max-w-xs rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                <p className="mt-1 text-xs text-gray-500">Orders above this amount qualify for free shipping</p>
              </div>
            )}
          </div>

          {/* Shipping Policy */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Shipping Policy</h2>
            <textarea rows={5} value={shippingSettings.shipping_policy}
              onChange={(e) => { setShippingSettings((p) => ({ ...p, shipping_policy: e.target.value })); setSaved(false); }}
              placeholder="Enter your store's shipping policy. This will be displayed on your storefront."
              className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
          </div>
        </div>
      )}

      {/* ==================== EMAIL TAB ==================== */}
      {activeTab === 'email' && (
        <div className="space-y-6">
          {/* SMTP Configuration */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center gap-2">
              <Mail className="h-5 w-5 text-gray-400" />
              <h2 className="text-lg font-semibold text-gray-900">SMTP Configuration</h2>
            </div>
            <p className="mb-4 text-sm text-gray-500">Configure your email server for sending transactional emails.</p>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Field label="SMTP Host" value={emailSettings.smtp_host}
                onChange={(v) => { setEmailSettings((p) => ({ ...p, smtp_host: v })); setSaved(false); }}
                placeholder="smtp.gmail.com" />
              <Field label="SMTP Port" value={emailSettings.smtp_port}
                onChange={(v) => { setEmailSettings((p) => ({ ...p, smtp_port: v })); setSaved(false); }}
                placeholder="587" />
              <Field label="Username" value={emailSettings.smtp_username}
                onChange={(v) => { setEmailSettings((p) => ({ ...p, smtp_username: v })); setSaved(false); }}
                placeholder="your@email.com" />
              <Field label="Password" type="password" value={emailSettings.smtp_password}
                onChange={(v) => { setEmailSettings((p) => ({ ...p, smtp_password: v })); setSaved(false); }}
                placeholder="App password" />
              <Field label="From Name" value={emailSettings.smtp_from_name}
                onChange={(v) => { setEmailSettings((p) => ({ ...p, smtp_from_name: v })); setSaved(false); }}
                placeholder="My Store" />
              <Field label="From Email" type="email" value={emailSettings.smtp_from_email}
                onChange={(v) => { setEmailSettings((p) => ({ ...p, smtp_from_email: v })); setSaved(false); }}
                placeholder="noreply@mystore.com" />
            </div>
            <div className="mt-4">
              <SelectField label="Encryption" value={emailSettings.smtp_encryption}
                onChange={(v) => { setEmailSettings((p) => ({ ...p, smtp_encryption: v as 'tls' | 'ssl' | 'none' })); setSaved(false); }}
                options={[
                  { value: 'tls', label: 'TLS (Recommended)' },
                  { value: 'ssl', label: 'SSL' },
                  { value: 'none', label: 'None' },
                ]}
              />
            </div>
          </div>

          {/* Notification Preferences */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-2 text-lg font-semibold text-gray-900">Notification Preferences</h2>
            <p className="mb-4 text-sm text-gray-500">Choose which events trigger email notifications</p>
            <div className="divide-y divide-gray-100">
              {([
                ['notify_order_placed', 'Order Placed', 'Send email when a new order is placed'],
                ['notify_order_confirmed', 'Order Confirmed', 'Send email when an order is confirmed'],
                ['notify_order_shipped', 'Order Shipped', 'Send email with tracking info when order ships'],
                ['notify_order_delivered', 'Order Delivered', 'Send email when order is delivered'],
                ['notify_order_cancelled', 'Order Cancelled', 'Send email when an order is cancelled'],
                ['notify_low_stock', 'Low Stock Alert', 'Notify admin when product stock is low'],
                ['notify_new_customer', 'New Customer', 'Notify admin when a new customer registers'],
              ] as const).map(([key, label, desc]) => (
                <div key={key} className="flex items-center justify-between py-4">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{label}</p>
                    <p className="text-xs text-gray-500">{desc}</p>
                  </div>
                  <ToggleButton
                    enabled={emailSettings[key]}
                    onToggle={() => { setEmailSettings((p) => ({ ...p, [key]: !p[key] })); setSaved(false); }}
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ==================== SECURITY TAB ==================== */}
      {activeTab === 'security' && (
        <div className="space-y-6">
          {/* Password Policy */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center gap-2">
              <Shield className="h-5 w-5 text-gray-400" />
              <h2 className="text-lg font-semibold text-gray-900">Password Policy</h2>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">Minimum Password Length</label>
                <input type="number" min={6} max={32} value={securitySettings.password_min_length}
                  onChange={(e) => { setSecuritySettings((p) => ({ ...p, password_min_length: Number(e.target.value) })); setSaved(false); }}
                  className="w-full max-w-xs rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div className="divide-y divide-gray-100">
                {([
                  ['password_require_uppercase', 'Require Uppercase Letter', 'At least one uppercase character (A-Z)'],
                  ['password_require_number', 'Require Number', 'At least one numeric digit (0-9)'],
                  ['password_require_special', 'Require Special Character', 'At least one special character (!@#$%^&*)'],
                ] as const).map(([key, label, desc]) => (
                  <div key={key} className="flex items-center justify-between py-4">
                    <div>
                      <p className="text-sm font-medium text-gray-900">{label}</p>
                      <p className="text-xs text-gray-500">{desc}</p>
                    </div>
                    <ToggleButton
                      enabled={securitySettings[key]}
                      onToggle={() => { setSecuritySettings((p) => ({ ...p, [key]: !p[key] })); setSaved(false); }}
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Two-Factor Authentication */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-gray-900">Two-Factor Authentication</h2>
                <p className="mt-1 text-sm text-gray-500">Require 2FA for admin and staff accounts</p>
              </div>
              <ToggleButton
                enabled={securitySettings.two_factor_enabled}
                onToggle={() => { setSecuritySettings((p) => ({ ...p, two_factor_enabled: !p.two_factor_enabled })); setSaved(false); }}
              />
            </div>
          </div>

          {/* Session & Rate Limiting */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-lg font-semibold text-gray-900">Session & Rate Limiting</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">Session Timeout (minutes)</label>
                <input type="number" min={5} max={1440} value={securitySettings.session_timeout_minutes}
                  onChange={(e) => { setSecuritySettings((p) => ({ ...p, session_timeout_minutes: Number(e.target.value) })); setSaved(false); }}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                <p className="mt-1 text-xs text-gray-500">Auto-logout after inactivity</p>
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">Max Login Attempts</label>
                <input type="number" min={3} max={20} value={securitySettings.max_login_attempts}
                  onChange={(e) => { setSecuritySettings((p) => ({ ...p, max_login_attempts: Number(e.target.value) })); setSaved(false); }}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                <p className="mt-1 text-xs text-gray-500">Lock account after failed attempts</p>
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">Lockout Duration (minutes)</label>
                <input type="number" min={1} max={60} value={securitySettings.lockout_duration_minutes}
                  onChange={(e) => { setSecuritySettings((p) => ({ ...p, lockout_duration_minutes: Number(e.target.value) })); setSaved(false); }}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-700">API Rate Limit (requests/min)</label>
                <input type="number" min={10} max={1000} value={securitySettings.api_rate_limit}
                  onChange={(e) => { setSecuritySettings((p) => ({ ...p, api_rate_limit: Number(e.target.value) })); setSaved(false); }}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ---- Reusable field components ----

function Field({ label, value, onChange, type = 'text', placeholder }: {
  label: string; value: string; onChange: (v: string) => void; type?: string; placeholder?: string;
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700">{label}</label>
      <input type={type} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
    </div>
  );
}

function SelectField({ label, value, onChange, options }: {
  label: string; value: string; onChange: (v: string) => void; options: { value: string; label: string }[];
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700">{label}</label>
      <select value={value} onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary">
        {options.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
      </select>
    </div>
  );
}

function ColorField({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700">{label}</label>
      <div className="flex items-center gap-3">
        <input type="color" value={value} onChange={(e) => onChange(e.target.value)} className="h-10 w-14 cursor-pointer rounded-lg border border-gray-300" />
        <input type="text" value={value} onChange={(e) => onChange(e.target.value)} placeholder="#3b82f6"
          className="flex-1 rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm font-mono focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
      </div>
    </div>
  );
}

function ToggleButton({ enabled, onToggle }: { enabled: boolean; onToggle: () => void }) {
  return (
    <button type="button" onClick={onToggle}
      className={cn('relative inline-flex h-6 w-11 items-center rounded-full transition-colors', enabled ? 'bg-primary' : 'bg-gray-200')}>
      <span className={cn('inline-block h-4 w-4 transform rounded-full bg-white transition-transform', enabled ? 'translate-x-6' : 'translate-x-1')} />
    </button>
  );
}

function getFeatureDescription(key: string): string {
  const desc: Record<string, string> = {
    multi_currency: 'Allow customers to view prices in different currencies',
    wishlist: 'Let customers save products to a wishlist',
    product_reviews: 'Enable product ratings and reviews',
    guest_checkout: 'Allow checkout without creating an account',
    social_login: 'Sign in with Google, Facebook, etc.',
    ai_recommendations: 'Show AI-powered product recommendations',
    loyalty_program: 'Reward customers with loyalty points',
    subscriptions: 'Enable subscription-based products',
    gift_cards: 'Sell and redeem gift cards',
  };
  return desc[key] || '';
}
