'use client';

import { create } from 'zustand';
import { configApi, type SetConfigRequest } from '@/lib/api';

export interface BannerSlide {
  id: string;
  image_url: string;
  title: string;
  subtitle: string;
  cta_text: string;
  cta_link: string;
  bg_color: string;
}

export interface StoreSection {
  id: string;
  type: 'hot_products' | 'discount' | 'new_arrivals' | 'category_showcase' | 'campaign' | 'custom';
  title: string;
  subtitle: string;
  enabled: boolean;
  position: number;
  config: Record<string, string>;
}

export interface FooterConfig {
  about_text: string;
  contact_email: string;
  contact_phone: string;
  contact_address: string;
  social_facebook: string;
  social_instagram: string;
  social_youtube: string;
  shop_links: { label: string; href: string }[];
  support_links: { label: string; href: string }[];
  copyright_text: string;
}

export interface AboutConfig {
  title: string;
  content: string;
  mission: string;
  vision: string;
  image_url: string;
}

export interface StorefrontConfig {
  banners: BannerSlide[];
  sections: StoreSection[];
  footer: FooterConfig;
  about: AboutConfig;
  announcement_bar: { enabled: boolean; text: string; bg_color: string; text_color: string };
}

const NAMESPACE = 'storefront';

const defaultConfig: StorefrontConfig = {
  banners: [
    {
      id: 'b1',
      image_url: '',
      title: 'Welcome to Saajan',
      subtitle: 'Discover the finest collection of traditional and modern fashion',
      cta_text: 'Shop Now',
      cta_link: '/products',
      bg_color: '#3b82f6',
    },
  ],
  sections: [
    { id: 's1', type: 'hot_products', title: 'Hot Products', subtitle: 'Trending items our customers love', enabled: true, position: 1, config: {} },
    { id: 's2', type: 'discount', title: 'On Sale', subtitle: 'Great deals you don\'t want to miss', enabled: true, position: 2, config: {} },
    { id: 's3', type: 'new_arrivals', title: 'New Arrivals', subtitle: 'Fresh additions to our collection', enabled: true, position: 3, config: {} },
    { id: 's4', type: 'category_showcase', title: 'Shop by Category', subtitle: 'Browse our collections', enabled: true, position: 4, config: {} },
  ],
  footer: {
    about_text: 'Your trusted e-commerce platform in Bangladesh. Quality products, fast delivery.',
    contact_email: 'support@saajan.com.bd',
    contact_phone: '+880 1712-345678',
    contact_address: 'Dhaka, Bangladesh',
    social_facebook: '',
    social_instagram: '',
    social_youtube: '',
    shop_links: [
      { label: 'All Products', href: '/products' },
      { label: 'Sarees', href: '/products?category=sarees' },
      { label: 'Electronics', href: '/products?category=electronics' },
    ],
    support_links: [
      { label: 'Contact Us', href: '/contact' },
      { label: 'Shipping Info', href: '/shipping' },
      { label: 'Returns', href: '/returns' },
    ],
    copyright_text: 'Saajan E-Commerce. All rights reserved.',
  },
  about: {
    title: 'About Saajan',
    content: 'We are dedicated to bringing you the best products from Bangladesh and beyond.',
    mission: 'To make quality products accessible to everyone.',
    vision: 'Becoming Bangladesh\'s most trusted e-commerce platform.',
    image_url: '',
  },
  announcement_bar: {
    enabled: false,
    text: 'Free shipping on orders over BDT 2,000!',
    bg_color: '#3b82f6',
    text_color: '#ffffff',
  },
};

interface StoreConfigState {
  config: StorefrontConfig;
  loading: boolean;
  error: string | null;
  fetchConfig: (tenantId: string) => Promise<void>;
  saveConfig: (tenantId: string, config: StorefrontConfig) => Promise<void>;
  updateConfig: (config: Partial<StorefrontConfig>) => void;
}

export const useStoreConfigStore = create<StoreConfigState>()((set, get) => ({
  config: defaultConfig,
  loading: false,
  error: null,

  fetchConfig: async (tenantId: string) => {
    set({ loading: true, error: null });
    try {
      const entries = await configApi.listByNamespace(NAMESPACE, 'all', tenantId);
      if (Array.isArray(entries) && entries.length > 0) {
        const configMap: Record<string, string> = {};
        entries.forEach((e) => { configMap[e.key] = e.value; });

        const parsed: StorefrontConfig = {
          banners: configMap.banners ? JSON.parse(configMap.banners) : defaultConfig.banners,
          sections: configMap.sections ? JSON.parse(configMap.sections) : defaultConfig.sections,
          footer: configMap.footer ? JSON.parse(configMap.footer) : defaultConfig.footer,
          about: configMap.about ? JSON.parse(configMap.about) : defaultConfig.about,
          announcement_bar: configMap.announcement_bar ? JSON.parse(configMap.announcement_bar) : defaultConfig.announcement_bar,
        };
        set({ config: parsed, loading: false });
      } else {
        set({ config: defaultConfig, loading: false });
      }
    } catch {
      set({ config: defaultConfig, loading: false });
    }
  },

  saveConfig: async (tenantId: string, config: StorefrontConfig) => {
    const entries: SetConfigRequest[] = [
      { namespace: NAMESPACE, key: 'banners', value: JSON.stringify(config.banners), value_type: 'json', tenant_id: tenantId, updated_by: 'admin' },
      { namespace: NAMESPACE, key: 'sections', value: JSON.stringify(config.sections), value_type: 'json', tenant_id: tenantId, updated_by: 'admin' },
      { namespace: NAMESPACE, key: 'footer', value: JSON.stringify(config.footer), value_type: 'json', tenant_id: tenantId, updated_by: 'admin' },
      { namespace: NAMESPACE, key: 'about', value: JSON.stringify(config.about), value_type: 'json', tenant_id: tenantId, updated_by: 'admin' },
      { namespace: NAMESPACE, key: 'announcement_bar', value: JSON.stringify(config.announcement_bar), value_type: 'json', tenant_id: tenantId, updated_by: 'admin' },
    ];
    await configApi.bulkSet(entries);
    set({ config });
  },

  updateConfig: (partial: Partial<StorefrontConfig>) => {
    set((state) => ({ config: { ...state.config, ...partial } }));
  },
}));

export { defaultConfig };
