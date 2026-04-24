'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { tenantApi, type Tenant, type CreateTenantRequest } from '@/lib/api';

interface TenantStore {
  tenants: Tenant[];
  loading: boolean;
  fetchTenants: () => Promise<void>;
  addTenant: (data: CreateTenantRequest) => Promise<void>;
  updateTenant: (id: string, data: Partial<Tenant>) => Promise<void>;
  deleteTenant: (id: string) => Promise<void>;
  updateTenantStatus: (id: string, status: Tenant['status']) => void;
}

export const useTenantStore = create<TenantStore>()(
  persist(
    (set, get) => ({
      tenants: [],
      loading: false,

      fetchTenants: async () => {
        set({ loading: true });
        try {
          const res = await tenantApi.list(1, 100);
          set({ tenants: res.data || [], loading: false });
        } catch {
          set({ loading: false });
        }
      },

      addTenant: async (data: CreateTenantRequest) => {
        try {
          const tenant = await tenantApi.create(data);
          set((s) => ({ tenants: [tenant, ...s.tenants] }));
        } catch {
          const now = new Date().toISOString();
          const tenant: Tenant = {
            id: `tenant_${data.name.toLowerCase().replace(/\s+/g, '_')}_${Date.now()}`,
            name: data.name,
            slug: data.name.toLowerCase().replace(/\s+/g, '-'),
            domain: null,
            email: data.email,
            status: 'active',
            tier: data.tier as Tenant['tier'],
            config: {
              general: { timezone: 'Asia/Dhaka', currency: 'BDT', language: 'en', date_format: 'DD/MM/YYYY', time_format: '12h', contact_email: data.email, contact_phone: '', support_url: '' },
              branding: { logo_url: '', favicon_url: '', primary_color: '#006A4E', secondary_color: '#F42A41', custom_css: '', custom_fonts: {} },
              features: { multi_currency: false, wishlist: true, product_reviews: true, guest_checkout: true, social_login: false, ai_recommendations: false, loyalty_program: false, subscriptions: false, gift_cards: false },
            },
            created_at: now,
            updated_at: now,
          };
          set((s) => ({ tenants: [tenant, ...s.tenants] }));
        }
      },

      updateTenant: async (id: string, data: Partial<Tenant>) => {
        try {
          await tenantApi.update(id, data);
          await get().fetchTenants();
        } catch {
          set((s) => ({
            tenants: s.tenants.map((t) =>
              t.id === id ? { ...t, ...data, updated_at: new Date().toISOString() } : t,
            ),
          }));
        }
      },

      deleteTenant: async (id: string) => {
        try {
          await tenantApi.delete(id);
        } catch {
          // delete locally
        }
        set((s) => ({ tenants: s.tenants.filter((t) => t.id !== id) }));
      },

      updateTenantStatus: (id: string, status: Tenant['status']) => {
        set((s) => ({
          tenants: s.tenants.map((t) =>
            t.id === id ? { ...t, status, updated_at: new Date().toISOString() } : t,
          ),
        }));
      },
    }),
    {
      name: 'tenant-storage',
      partialize: (state) => ({ tenants: state.tenants }),
    },
  ),
);
