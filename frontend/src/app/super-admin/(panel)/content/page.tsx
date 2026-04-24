'use client';

import { useState } from 'react';
import {
  Plus,
  Pencil,
  Eye,
  Trash2,
  FileText,
  Palette,
  Globe,
  Upload,
  Check,
  X,
  Bold,
  Italic,
  Underline,
  List,
  ListOrdered,
  Link,
  Image,
  AlignLeft,
  AlignCenter,
  AlignRight,
  Heading1,
  Heading2,
  Search,
  RotateCcw,
  Minus,
} from 'lucide-react';
import { cn, formatDate } from '@/lib/utils';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Tab = 'cms' | 'themes' | 'marketing';
type PageStatus = 'published' | 'draft' | 'archived';

interface CmsPage {
  id: string;
  title: string;
  slug: string;
  status: PageStatus;
  lastUpdated: string;
  author: string;
  content: string;
  applyTo: 'all' | 'specific';
  metaTitle: string;
  metaDescription: string;
}

interface ThemeTemplate {
  id: string;
  name: string;
  description: string;
  active: boolean;
  colors: string[];
}

interface TenantOverride {
  id: string;
  name: string;
  activeTheme: string;
  customColors: boolean;
  customLogo: boolean;
}

interface FeatureHighlight {
  icon: string;
  title: string;
  description: string;
}

interface FaqItem {
  id: string;
  title: string;
  answer: string;
}

// ---------------------------------------------------------------------------
// Demo data
// ---------------------------------------------------------------------------

const initialPages: CmsPage[] = [
  { id: '1', title: 'Terms of Service', slug: 'terms-of-service', status: 'published', lastUpdated: '2026-04-20T10:30:00Z', author: 'Admin', content: '', applyTo: 'all', metaTitle: 'Terms of Service', metaDescription: 'Read our terms of service.' },
  { id: '2', title: 'Privacy Policy', slug: 'privacy-policy', status: 'published', lastUpdated: '2026-04-18T14:00:00Z', author: 'Admin', content: '', applyTo: 'all', metaTitle: 'Privacy Policy', metaDescription: 'Learn about our privacy practices.' },
  { id: '3', title: 'Refund Policy', slug: 'refund-policy', status: 'published', lastUpdated: '2026-04-15T09:00:00Z', author: 'Admin', content: '', applyTo: 'all', metaTitle: 'Refund Policy', metaDescription: 'Our refund and return policies.' },
  { id: '4', title: 'About Us', slug: 'about-us', status: 'published', lastUpdated: '2026-04-10T11:00:00Z', author: 'Admin', content: '', applyTo: 'all', metaTitle: 'About Us', metaDescription: 'Learn more about our platform.' },
  { id: '5', title: 'FAQ', slug: 'faq', status: 'draft', lastUpdated: '2026-04-22T16:30:00Z', author: 'Editor', content: '', applyTo: 'all', metaTitle: 'FAQ', metaDescription: 'Frequently asked questions.' },
  { id: '6', title: 'Shipping Information', slug: 'shipping-information', status: 'published', lastUpdated: '2026-04-12T08:00:00Z', author: 'Admin', content: '', applyTo: 'all', metaTitle: 'Shipping Information', metaDescription: 'Shipping details and delivery times.' },
  { id: '7', title: 'Contact Us', slug: 'contact-us', status: 'published', lastUpdated: '2026-04-08T13:00:00Z', author: 'Admin', content: '', applyTo: 'all', metaTitle: 'Contact Us', metaDescription: 'Get in touch with us.' },
];

const themeTemplates: ThemeTemplate[] = [
  { id: 'modern', name: 'Modern', description: 'Clean minimalist design with sharp lines and ample whitespace', active: true, colors: ['#4F46E5', '#10B981', '#F9FAFB', '#111827'] },
  { id: 'classic', name: 'Classic', description: 'Traditional e-commerce layout with a familiar, trustworthy feel', active: false, colors: ['#1E40AF', '#D97706', '#FFFFFF', '#1F2937'] },
  { id: 'bold', name: 'Bold', description: 'Vibrant colors and large imagery for high-impact storefronts', active: false, colors: ['#DC2626', '#7C3AED', '#FEF3C7', '#0F172A'] },
  { id: 'minimal', name: 'Minimal', description: 'Ultra-clean aesthetic focused on whitespace and typography', active: false, colors: ['#18181B', '#A1A1AA', '#FFFFFF', '#3F3F46'] },
];

const tenantOverrides: TenantOverride[] = [
  { id: '1', name: 'FreshMart BD', activeTheme: 'Modern', customColors: true, customLogo: true },
  { id: '2', name: 'StyleHub', activeTheme: 'Bold', customColors: true, customLogo: false },
  { id: '3', name: 'TechZone', activeTheme: 'Modern', customColors: false, customLogo: true },
  { id: '4', name: 'BookNook', activeTheme: 'Classic', customColors: false, customLogo: false },
  { id: '5', name: 'GreenGrocer', activeTheme: 'Minimal', customColors: true, customLogo: true },
];

const statusBadge: Record<PageStatus, string> = {
  published: 'bg-green-100 text-green-800',
  draft: 'bg-gray-100 text-gray-800',
  archived: 'bg-orange-100 text-orange-800',
};

const fontOptions = ['Inter', 'Poppins', 'Roboto', 'Open Sans'];

const iconOptions = ['Sparkles', 'Globe', 'LayoutGrid', 'Palette', 'FileText', 'Search'];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '');
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function ContentManagementPage() {
  const [activeTab, setActiveTab] = useState<Tab>('cms');

  // ---- CMS state ----
  const [pages, setPages] = useState<CmsPage[]>(initialPages);
  const [pageSearch, setPageSearch] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingPage, setEditingPage] = useState<CmsPage | null>(null);
  const [formTitle, setFormTitle] = useState('');
  const [formSlug, setFormSlug] = useState('');
  const [formContent, setFormContent] = useState('');
  const [formStatus, setFormStatus] = useState<PageStatus>('draft');
  const [formApplyTo, setFormApplyTo] = useState<'all' | 'specific'>('all');
  const [formMetaTitle, setFormMetaTitle] = useState('');
  const [formMetaDesc, setFormMetaDesc] = useState('');

  // ---- Themes state ----
  const [platformName, setPlatformName] = useState('BracSaajan Commerce');
  const [primaryColor, setPrimaryColor] = useState('#4F46E5');
  const [secondaryColor, setSecondaryColor] = useState('#10B981');
  const [fontFamily, setFontFamily] = useState('Inter');
  const [themes, setThemes] = useState(themeTemplates);
  const [overrides, setOverrides] = useState(tenantOverrides);

  // ---- Marketing state ----
  const [heroTitle, setHeroTitle] = useState('Launch Your Online Store in Minutes');
  const [heroSubtitle, setHeroSubtitle] = useState('The all-in-one multi-tenant e-commerce platform built for Bangladesh and beyond.');
  const [ctaText, setCtaText] = useState('Get Started Free');
  const [ctaLink, setCtaLink] = useState('/signup');
  const [features, setFeatures] = useState<FeatureHighlight[]>([
    { icon: 'Sparkles', title: 'Easy Setup', description: 'Go live in under 5 minutes with guided onboarding.' },
    { icon: 'Globe', title: 'Multi-Tenant', description: 'Each store gets its own domain, theme, and data.' },
    { icon: 'LayoutGrid', title: 'Feature-Rich', description: 'Inventory, orders, payments, and analytics built-in.' },
  ]);
  const [showPricing, setShowPricing] = useState(true);
  const [showTestimonials, setShowTestimonials] = useState(true);
  const [showIntegrations, setShowIntegrations] = useState(false);
  const [pricingHeading, setPricingHeading] = useState('Simple, Transparent Pricing');
  const [showMonthlyYearly, setShowMonthlyYearly] = useState(true);
  const [showFeatureComparison, setShowFeatureComparison] = useState(true);
  const [pricingFaqs, setPricingFaqs] = useState<FaqItem[]>([
    { id: '1', title: 'Can I change my plan later?', answer: 'Yes, you can upgrade or downgrade your plan at any time from your dashboard.' },
    { id: '2', title: 'Is there a free trial?', answer: 'Our Free plan is always free. You can upgrade whenever you are ready.' },
  ]);
  const [defaultMetaTitle, setDefaultMetaTitle] = useState('BracSaajan Commerce - Multi-Tenant E-commerce Platform');
  const [defaultMetaDesc, setDefaultMetaDesc] = useState('Launch and manage multiple online stores from one powerful platform.');
  const [gaId, setGaId] = useState('');
  const [fbPixelId, setFbPixelId] = useState('');
  const [headScripts, setHeadScripts] = useState('');

  // ---- CMS helpers ----

  function openCreateModal() {
    setEditingPage(null);
    setFormTitle('');
    setFormSlug('');
    setFormContent('');
    setFormStatus('draft');
    setFormApplyTo('all');
    setFormMetaTitle('');
    setFormMetaDesc('');
    setModalOpen(true);
  }

  function openEditModal(page: CmsPage) {
    setEditingPage(page);
    setFormTitle(page.title);
    setFormSlug(page.slug);
    setFormContent(page.content);
    setFormStatus(page.status);
    setFormApplyTo(page.applyTo);
    setFormMetaTitle(page.metaTitle);
    setFormMetaDesc(page.metaDescription);
    setModalOpen(true);
  }

  function handleSave(publishOverride?: boolean) {
    const now = new Date().toISOString();
    const status = publishOverride ? 'published' : formStatus;
    if (editingPage) {
      setPages((prev) =>
        prev.map((p) =>
          p.id === editingPage.id
            ? { ...p, title: formTitle, slug: formSlug, content: formContent, status, applyTo: formApplyTo, metaTitle: formMetaTitle, metaDescription: formMetaDesc, lastUpdated: now }
            : p,
        ),
      );
    } else {
      const newPage: CmsPage = {
        id: String(Date.now()),
        title: formTitle,
        slug: formSlug || slugify(formTitle),
        content: formContent,
        status,
        lastUpdated: now,
        author: 'Admin',
        applyTo: formApplyTo,
        metaTitle: formMetaTitle,
        metaDescription: formMetaDesc,
      };
      setPages((prev) => [newPage, ...prev]);
    }
    setModalOpen(false);
  }

  function handleDelete(id: string) {
    if (!confirm('Delete this page? This action cannot be undone.')) return;
    setPages((prev) => prev.filter((p) => p.id !== id));
  }

  const filteredPages = pages.filter((p) => {
    if (!pageSearch) return true;
    const q = pageSearch.toLowerCase();
    return p.title.toLowerCase().includes(q) || p.slug.toLowerCase().includes(q);
  });

  // ---- Theme helpers ----

  function activateTheme(id: string) {
    setThemes((prev) => prev.map((t) => ({ ...t, active: t.id === id })));
  }

  function resetOverride(id: string) {
    setOverrides((prev) =>
      prev.map((o) => (o.id === id ? { ...o, activeTheme: 'Modern', customColors: false, customLogo: false } : o)),
    );
  }

  // ---- Marketing helpers ----

  function updateFeature(index: number, field: keyof FeatureHighlight, value: string) {
    setFeatures((prev) => prev.map((f, i) => (i === index ? { ...f, [field]: value } : f)));
  }

  function addFaq() {
    setPricingFaqs((prev) => [...prev, { id: String(Date.now()), title: '', answer: '' }]);
  }

  function updateFaq(id: string, field: 'title' | 'answer', value: string) {
    setPricingFaqs((prev) => prev.map((f) => (f.id === id ? { ...f, [field]: value } : f)));
  }

  function removeFaq(id: string) {
    setPricingFaqs((prev) => prev.filter((f) => f.id !== id));
  }

  // ---- Tab config ----

  const tabs: { key: Tab; label: string; icon: typeof FileText }[] = [
    { key: 'cms', label: 'CMS Pages', icon: FileText },
    { key: 'themes', label: 'Themes & Branding', icon: Palette },
    { key: 'marketing', label: 'Marketing Site', icon: Globe },
  ];

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Content & Branding</h1>
        <p className="mt-1 text-sm text-gray-500">Manage CMS pages, storefront themes, and your marketing website.</p>
      </div>

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
                  'inline-flex items-center gap-2 border-b-2 px-1 py-3 text-sm font-medium transition-colors',
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

      {/* ================================================================= */}
      {/* CMS Pages Tab                                                      */}
      {/* ================================================================= */}
      {activeTab === 'cms' && (
        <div className="space-y-6">
          {/* Header row */}
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Content Management</h2>
              <p className="text-sm text-gray-500">{pages.length} pages total</p>
            </div>
            <button
              onClick={openCreateModal}
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
            >
              <Plus className="h-4 w-4" />
              Create Page
            </button>
          </div>

          {/* Search */}
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              value={pageSearch}
              onChange={(e) => setPageSearch(e.target.value)}
              placeholder="Search pages..."
              className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>

          {/* Pages table */}
          <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                    <th className="px-6 py-3 font-medium">Page Title</th>
                    <th className="px-6 py-3 font-medium">Slug</th>
                    <th className="px-6 py-3 font-medium">Status</th>
                    <th className="px-6 py-3 font-medium">Last Updated</th>
                    <th className="px-6 py-3 font-medium">Author</th>
                    <th className="px-6 py-3 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredPages.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-16 text-center text-sm text-gray-400">
                        No pages found.
                      </td>
                    </tr>
                  ) : (
                    filteredPages.map((page) => (
                      <tr key={page.id} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                        <td className="px-6 py-4 text-sm font-medium text-gray-900">{page.title}</td>
                        <td className="px-6 py-4 text-sm font-mono text-gray-500">/{page.slug}</td>
                        <td className="px-6 py-4">
                          <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusBadge[page.status])}>
                            {page.status}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-sm text-gray-500">{formatDate(page.lastUpdated)}</td>
                        <td className="px-6 py-4 text-sm text-gray-500">{page.author}</td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => openEditModal(page)}
                              className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                              title="Edit"
                            >
                              <Pencil className="h-4 w-4" />
                            </button>
                            <button
                              className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                              title="Preview"
                            >
                              <Eye className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => handleDelete(page.id)}
                              className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                              title="Delete"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Create / Edit Modal */}
          {modalOpen && (
            <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/40 p-4 pt-16">
              <div className="w-full max-w-2xl rounded-2xl border border-gray-200 bg-white shadow-xl">
                {/* Modal header */}
                <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
                  <h3 className="text-lg font-semibold text-gray-900">{editingPage ? 'Edit Page' : 'Create Page'}</h3>
                  <button onClick={() => setModalOpen(false)} className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600">
                    <X className="h-5 w-5" />
                  </button>
                </div>

                {/* Modal body */}
                <div className="space-y-5 px-6 py-5">
                  {/* Title & Slug */}
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                      <label className="mb-1 block text-sm font-medium text-gray-700">Page Title</label>
                      <input
                        value={formTitle}
                        onChange={(e) => {
                          setFormTitle(e.target.value);
                          if (!editingPage) setFormSlug(slugify(e.target.value));
                        }}
                        placeholder="e.g. Return Policy"
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-sm font-medium text-gray-700">Slug</label>
                      <div className="flex items-center rounded-lg border border-gray-200 text-sm focus-within:border-indigo-500 focus-within:ring-1 focus-within:ring-indigo-500">
                        <span className="pl-3 text-gray-400">/</span>
                        <input
                          value={formSlug}
                          onChange={(e) => setFormSlug(e.target.value)}
                          className="w-full bg-transparent py-2 pl-1 pr-3 focus:outline-none"
                        />
                      </div>
                    </div>
                  </div>

                  {/* Rich text area with toolbar */}
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Content</label>
                    <div className="overflow-hidden rounded-lg border border-gray-200 focus-within:border-indigo-500 focus-within:ring-1 focus-within:ring-indigo-500">
                      {/* Toolbar */}
                      <div className="flex flex-wrap items-center gap-0.5 border-b border-gray-200 bg-gray-50 px-2 py-1.5">
                        {[Heading1, Heading2, Bold, Italic, Underline].map((Icon, i) => (
                          <button key={i} type="button" className="rounded p-1.5 text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700">
                            <Icon className="h-4 w-4" />
                          </button>
                        ))}
                        <span className="mx-1 h-5 w-px bg-gray-300" />
                        {[List, ListOrdered].map((Icon, i) => (
                          <button key={i} type="button" className="rounded p-1.5 text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700">
                            <Icon className="h-4 w-4" />
                          </button>
                        ))}
                        <span className="mx-1 h-5 w-px bg-gray-300" />
                        {[AlignLeft, AlignCenter, AlignRight].map((Icon, i) => (
                          <button key={i} type="button" className="rounded p-1.5 text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700">
                            <Icon className="h-4 w-4" />
                          </button>
                        ))}
                        <span className="mx-1 h-5 w-px bg-gray-300" />
                        {[Link, Image].map((Icon, i) => (
                          <button key={i} type="button" className="rounded p-1.5 text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700">
                            <Icon className="h-4 w-4" />
                          </button>
                        ))}
                      </div>
                      <textarea
                        value={formContent}
                        onChange={(e) => setFormContent(e.target.value)}
                        rows={8}
                        placeholder="Write your page content here..."
                        className="w-full resize-y px-4 py-3 text-sm focus:outline-none"
                      />
                    </div>
                  </div>

                  {/* Status & Apply To */}
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                      <label className="mb-1 block text-sm font-medium text-gray-700">Status</label>
                      <select
                        value={formStatus}
                        onChange={(e) => setFormStatus(e.target.value as PageStatus)}
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
                      >
                        <option value="draft">Draft</option>
                        <option value="published">Published</option>
                        <option value="archived">Archived</option>
                      </select>
                    </div>
                    <div>
                      <label className="mb-1 block text-sm font-medium text-gray-700">Apply To</label>
                      <select
                        value={formApplyTo}
                        onChange={(e) => setFormApplyTo(e.target.value as 'all' | 'specific')}
                        className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
                      >
                        <option value="all">All Tenants</option>
                        <option value="specific">Specific Tenants</option>
                      </select>
                    </div>
                  </div>

                  {/* SEO */}
                  <div className="rounded-lg border border-gray-200 p-4">
                    <h4 className="mb-3 text-sm font-semibold text-gray-700">SEO Settings</h4>
                    <div className="space-y-3">
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">Meta Title</label>
                        <input
                          value={formMetaTitle}
                          onChange={(e) => setFormMetaTitle(e.target.value)}
                          className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">Meta Description</label>
                        <textarea
                          value={formMetaDesc}
                          onChange={(e) => setFormMetaDesc(e.target.value)}
                          rows={2}
                          className="w-full resize-none rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                {/* Modal footer */}
                <div className="flex items-center justify-end gap-3 border-t border-gray-200 px-6 py-4">
                  <button
                    onClick={() => setModalOpen(false)}
                    className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => handleSave(false)}
                    className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                  >
                    Save as {formStatus === 'draft' ? 'Draft' : formStatus === 'archived' ? 'Archived' : 'Published'}
                  </button>
                  {formStatus !== 'published' && (
                    <button
                      onClick={() => handleSave(true)}
                      className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                    >
                      Publish
                    </button>
                  )}
                  {formStatus === 'published' && (
                    <button
                      onClick={() => handleSave(false)}
                      className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
                    >
                      Update
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ================================================================= */}
      {/* Themes & Branding Tab                                              */}
      {/* ================================================================= */}
      {activeTab === 'themes' && (
        <div className="space-y-8">
          {/* Global Branding */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Global Branding</h2>
              <p className="mt-1 text-sm text-gray-500">Configure your platform-wide brand identity.</p>
            </div>
            <div className="grid grid-cols-1 gap-6 p-6 lg:grid-cols-2">
              {/* Platform name */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Platform Name</label>
                <input
                  value={platformName}
                  onChange={(e) => setPlatformName(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>

              {/* Font family */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Font Family</label>
                <select
                  value={fontFamily}
                  onChange={(e) => setFontFamily(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
                >
                  {fontOptions.map((f) => (
                    <option key={f} value={f}>{f}</option>
                  ))}
                </select>
              </div>

              {/* Logo upload */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Logo</label>
                <div className="flex items-center gap-4">
                  <div className="flex h-16 w-16 items-center justify-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 text-gray-400">
                    <Image className="h-6 w-6" />
                  </div>
                  <button className="inline-flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50">
                    <Upload className="h-4 w-4" />
                    Upload Logo
                  </button>
                </div>
              </div>

              {/* Favicon upload */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Favicon</label>
                <div className="flex items-center gap-4">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 text-gray-400">
                    <Image className="h-4 w-4" />
                  </div>
                  <button className="inline-flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50">
                    <Upload className="h-4 w-4" />
                    Upload Favicon
                  </button>
                </div>
              </div>

              {/* Primary color */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Primary Color</label>
                <div className="flex items-center gap-3">
                  <input
                    type="color"
                    value={primaryColor}
                    onChange={(e) => setPrimaryColor(e.target.value)}
                    className="h-10 w-10 cursor-pointer rounded-lg border border-gray-200 p-0.5"
                  />
                  <input
                    value={primaryColor}
                    onChange={(e) => setPrimaryColor(e.target.value)}
                    className="w-28 rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                  <div className="h-8 w-8 rounded-full border border-gray-200" style={{ backgroundColor: primaryColor }} />
                </div>
              </div>

              {/* Secondary color */}
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Secondary Color</label>
                <div className="flex items-center gap-3">
                  <input
                    type="color"
                    value={secondaryColor}
                    onChange={(e) => setSecondaryColor(e.target.value)}
                    className="h-10 w-10 cursor-pointer rounded-lg border border-gray-200 p-0.5"
                  />
                  <input
                    value={secondaryColor}
                    onChange={(e) => setSecondaryColor(e.target.value)}
                    className="w-28 rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                  <div className="h-8 w-8 rounded-full border border-gray-200" style={{ backgroundColor: secondaryColor }} />
                </div>
              </div>
            </div>

            {/* Save branding */}
            <div className="border-t border-gray-200 px-6 py-4">
              <button className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700">
                Save Branding
              </button>
            </div>
          </div>

          {/* Theme Templates */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Theme Templates</h2>
              <p className="mt-1 text-sm text-gray-500">Choose a base theme for tenant storefronts.</p>
            </div>
            <div className="grid grid-cols-1 gap-6 p-6 sm:grid-cols-2 xl:grid-cols-4">
              {themes.map((theme) => (
                <div
                  key={theme.id}
                  className={cn(
                    'relative flex flex-col rounded-xl border-2 p-5 transition-shadow hover:shadow-md',
                    theme.active ? 'border-indigo-500 ring-1 ring-indigo-500' : 'border-gray-200',
                  )}
                >
                  {theme.active && (
                    <div className="absolute -top-2.5 left-1/2 -translate-x-1/2">
                      <span className="inline-flex items-center gap-1 rounded-full bg-indigo-600 px-2.5 py-0.5 text-xs font-medium text-white">
                        <Check className="h-3 w-3" />
                        Active
                      </span>
                    </div>
                  )}

                  {/* Color swatches */}
                  <div className="mb-4 flex gap-1.5">
                    {theme.colors.map((color, i) => (
                      <div key={i} className="h-6 w-6 rounded-full border border-gray-200" style={{ backgroundColor: color }} />
                    ))}
                  </div>

                  <h3 className="text-sm font-semibold text-gray-900">{theme.name}</h3>
                  <p className="mt-1 flex-1 text-xs text-gray-500">{theme.description}</p>

                  <div className="mt-4 flex items-center justify-between">
                    {theme.active ? (
                      <span className="text-xs font-medium text-indigo-600">Default Theme</span>
                    ) : (
                      <button
                        onClick={() => activateTheme(theme.id)}
                        className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-indigo-700"
                      >
                        Activate
                      </button>
                    )}
                    <button className="text-xs font-medium text-indigo-600 hover:underline">Preview</button>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Tenant Theme Overrides */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Tenant Theme Overrides</h2>
              <p className="mt-1 text-sm text-gray-500">View and manage per-tenant customizations.</p>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                    <th className="px-6 py-3 font-medium">Tenant Name</th>
                    <th className="px-6 py-3 font-medium">Active Theme</th>
                    <th className="px-6 py-3 font-medium">Custom Colors</th>
                    <th className="px-6 py-3 font-medium">Custom Logo</th>
                    <th className="px-6 py-3 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {overrides.map((o) => (
                    <tr key={o.id} className="border-b border-gray-50 transition-colors hover:bg-gray-50">
                      <td className="px-6 py-4 text-sm font-medium text-gray-900">{o.name}</td>
                      <td className="px-6 py-4 text-sm text-gray-500">{o.activeTheme}</td>
                      <td className="px-6 py-4">
                        {o.customColors ? (
                          <span className="inline-flex rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">Yes</span>
                        ) : (
                          <span className="inline-flex rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">No</span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        {o.customLogo ? (
                          <span className="inline-flex rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">Yes</span>
                        ) : (
                          <span className="inline-flex rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">No</span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        <button
                          onClick={() => resetOverride(o.id)}
                          className="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-100"
                        >
                          <RotateCcw className="h-3.5 w-3.5" />
                          Reset to Default
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* ================================================================= */}
      {/* Marketing Site Tab                                                 */}
      {/* ================================================================= */}
      {activeTab === 'marketing' && (
        <div className="space-y-8">
          {/* Landing Page Config */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Landing Page Configuration</h2>
              <p className="mt-1 text-sm text-gray-500">Customize your marketing landing page hero and features.</p>
            </div>
            <div className="space-y-6 p-6">
              {/* Hero */}
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Hero Title</label>
                  <input
                    value={heroTitle}
                    onChange={(e) => setHeroTitle(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Hero Subtitle</label>
                  <input
                    value={heroSubtitle}
                    onChange={(e) => setHeroSubtitle(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">CTA Button Text</label>
                  <input
                    value={ctaText}
                    onChange={(e) => setCtaText(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">CTA Button Link</label>
                  <input
                    value={ctaLink}
                    onChange={(e) => setCtaLink(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
              </div>

              {/* Feature Highlights */}
              <div>
                <h3 className="mb-3 text-sm font-semibold text-gray-700">Feature Highlights</h3>
                <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
                  {features.map((feature, idx) => (
                    <div key={idx} className="rounded-lg border border-gray-200 p-4 space-y-3">
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">Icon</label>
                        <select
                          value={feature.icon}
                          onChange={(e) => updateFeature(idx, 'icon', e.target.value)}
                          className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none"
                        >
                          {iconOptions.map((ic) => (
                            <option key={ic} value={ic}>{ic}</option>
                          ))}
                        </select>
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">Title</label>
                        <input
                          value={feature.title}
                          onChange={(e) => updateFeature(idx, 'title', e.target.value)}
                          className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
                        <textarea
                          value={feature.description}
                          onChange={(e) => updateFeature(idx, 'description', e.target.value)}
                          rows={2}
                          className="w-full resize-none rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                        />
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Toggles */}
              <div className="flex flex-wrap gap-6">
                <ToggleSwitch label="Show Pricing Section" checked={showPricing} onChange={setShowPricing} />
                <ToggleSwitch label="Show Testimonials" checked={showTestimonials} onChange={setShowTestimonials} />
                <ToggleSwitch label="Show Integration Logos" checked={showIntegrations} onChange={setShowIntegrations} />
              </div>
            </div>

            <div className="border-t border-gray-200 px-6 py-4">
              <button className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700">
                Save Landing Page
              </button>
            </div>
          </div>

          {/* Pricing Page Config */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Pricing Page Configuration</h2>
              <p className="mt-1 text-sm text-gray-500">Customize how your pricing page looks and what it shows.</p>
            </div>
            <div className="space-y-6 p-6">
              <div className="max-w-md">
                <label className="mb-1 block text-sm font-medium text-gray-700">Heading Text</label>
                <input
                  value={pricingHeading}
                  onChange={(e) => setPricingHeading(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>

              <div className="flex flex-wrap gap-6">
                <ToggleSwitch label="Show Monthly / Yearly Toggle" checked={showMonthlyYearly} onChange={setShowMonthlyYearly} />
                <ToggleSwitch label="Show Feature Comparison Table" checked={showFeatureComparison} onChange={setShowFeatureComparison} />
              </div>

              {/* FAQ Items */}
              <div>
                <div className="mb-3 flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-gray-700">Pricing FAQ</h3>
                  <button
                    onClick={addFaq}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    Add Item
                  </button>
                </div>
                <div className="space-y-3">
                  {pricingFaqs.map((faq) => (
                    <div key={faq.id} className="rounded-lg border border-gray-200 p-4">
                      <div className="flex items-start gap-3">
                        <div className="flex-1 space-y-2">
                          <input
                            value={faq.title}
                            onChange={(e) => updateFaq(faq.id, 'title', e.target.value)}
                            placeholder="Question"
                            className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                          />
                          <textarea
                            value={faq.answer}
                            onChange={(e) => updateFaq(faq.id, 'answer', e.target.value)}
                            placeholder="Answer"
                            rows={2}
                            className="w-full resize-none rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                          />
                        </div>
                        <button
                          onClick={() => removeFaq(faq.id)}
                          className="mt-1 rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                        >
                          <Minus className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  ))}
                  {pricingFaqs.length === 0 && (
                    <p className="py-4 text-center text-sm text-gray-400">No FAQ items yet. Click &quot;Add Item&quot; to create one.</p>
                  )}
                </div>
              </div>
            </div>

            <div className="border-t border-gray-200 px-6 py-4">
              <button className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700">
                Save Pricing Config
              </button>
            </div>
          </div>

          {/* SEO & Analytics */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">SEO & Analytics</h2>
              <p className="mt-1 text-sm text-gray-500">Default meta tags and tracking integrations for your marketing site.</p>
            </div>
            <div className="space-y-5 p-6">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Default Meta Title</label>
                  <input
                    value={defaultMetaTitle}
                    onChange={(e) => setDefaultMetaTitle(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Default Meta Description</label>
                  <input
                    value={defaultMetaDesc}
                    onChange={(e) => setDefaultMetaDesc(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Google Analytics ID</label>
                  <input
                    value={gaId}
                    onChange={(e) => setGaId(e.target.value)}
                    placeholder="G-XXXXXXXXXX"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Facebook Pixel ID</label>
                  <input
                    value={fbPixelId}
                    onChange={(e) => setFbPixelId(e.target.value)}
                    placeholder="XXXXXXXXXXXXXXX"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm font-mono focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  />
                </div>
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Custom Head Scripts</label>
                <textarea
                  value={headScripts}
                  onChange={(e) => setHeadScripts(e.target.value)}
                  rows={5}
                  placeholder={'<!-- Add custom scripts here -->\n<script>...</script>'}
                  className="w-full resize-y rounded-lg border border-gray-200 px-3 py-2 font-mono text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
              </div>
            </div>

            <div className="border-t border-gray-200 px-6 py-4">
              <button className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700">
                Save SEO & Analytics
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Toggle switch sub-component
// ---------------------------------------------------------------------------

function ToggleSwitch({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="inline-flex cursor-pointer items-center gap-3">
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={cn(
          'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2',
          checked ? 'bg-indigo-600' : 'bg-gray-200',
        )}
      >
        <span
          className={cn(
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            checked ? 'translate-x-5' : 'translate-x-0',
          )}
        />
      </button>
      <span className="text-sm font-medium text-gray-700">{label}</span>
    </label>
  );
}
