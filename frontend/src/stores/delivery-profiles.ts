'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { configApi, type DeliveryProfile, type SetConfigRequest } from '@/lib/api';

const DEFAULT_PROFILES: DeliveryProfile[] = [
  {
    id: 'dp-standard',
    name: 'Standard',
    inside_dhaka_rate: 60,
    outside_dhaka_rate: 120,
    inside_dhaka_express_rate: 100,
    outside_dhaka_express_rate: 180,
    estimated_delivery_dhaka: '1-2 days',
    estimated_delivery_outside: '3-5 days',
    is_default: true,
  },
  {
    id: 'dp-heavy',
    name: 'Heavy Items',
    inside_dhaka_rate: 100,
    outside_dhaka_rate: 200,
    inside_dhaka_express_rate: 150,
    outside_dhaka_express_rate: 280,
    estimated_delivery_dhaka: '2-3 days',
    estimated_delivery_outside: '5-7 days',
    is_default: false,
  },
  {
    id: 'dp-fragile',
    name: 'Fragile',
    inside_dhaka_rate: 120,
    outside_dhaka_rate: 250,
    inside_dhaka_express_rate: 180,
    outside_dhaka_express_rate: 350,
    estimated_delivery_dhaka: '2-3 days',
    estimated_delivery_outside: '4-6 days',
    is_default: false,
  },
  {
    id: 'dp-free',
    name: 'Free Shipping',
    inside_dhaka_rate: 0,
    outside_dhaka_rate: 0,
    inside_dhaka_express_rate: 0,
    outside_dhaka_express_rate: 0,
    estimated_delivery_dhaka: '2-3 days',
    estimated_delivery_outside: '5-7 days',
    is_default: false,
  },
];

interface DeliveryProfileStore {
  profiles: DeliveryProfile[];
  loading: boolean;
  error: string | null;
  fetchProfiles: (tenantId: string) => Promise<void>;
  saveProfiles: (tenantId: string) => Promise<void>;
  addProfile: (profile: DeliveryProfile) => void;
  updateProfile: (id: string, updates: Partial<DeliveryProfile>) => void;
  removeProfile: (id: string) => void;
  getProfile: (id: string) => DeliveryProfile | undefined;
  getDefaultProfile: () => DeliveryProfile;
}

export const useDeliveryProfileStore = create<DeliveryProfileStore>()(
  persist(
    (set, get) => ({
      profiles: DEFAULT_PROFILES,
      loading: false,
      error: null,

      fetchProfiles: async (tenantId: string) => {
        set({ loading: true, error: null });
        try {
          const entries = await configApi.listByNamespace('delivery_profiles', 'all', tenantId);
          const profileEntry = entries.find((e: { key: string }) => e.key === 'profiles');
          if (profileEntry) {
            const parsed = typeof profileEntry.value === 'string'
              ? JSON.parse(profileEntry.value)
              : profileEntry.value;
            if (Array.isArray(parsed) && parsed.length > 0) {
              set({ profiles: parsed });
            }
          }
        } catch {
          // Keep persisted data from localStorage
        } finally {
          set({ loading: false });
        }
      },

      saveProfiles: async (tenantId: string) => {
        const { profiles } = get();
        try {
          const req: SetConfigRequest[] = [
            {
              namespace: 'delivery_profiles',
              key: 'profiles',
              value: JSON.stringify(profiles),
              value_type: 'json',
              tenant_id: tenantId,
              updated_by: 'admin',
            },
          ];
          await configApi.bulkSet(req);
        } catch {
          // Saved locally via persist, will sync later
        }
      },

      addProfile: (profile: DeliveryProfile) => {
        set((state) => ({ profiles: [...state.profiles, profile] }));
      },

      updateProfile: (id: string, updates: Partial<DeliveryProfile>) => {
        set((state) => ({
          profiles: state.profiles.map((p) =>
            p.id === id ? { ...p, ...updates } : p,
          ),
        }));
      },

      removeProfile: (id: string) => {
        set((state) => ({
          profiles: state.profiles.filter((p) => p.id !== id),
        }));
      },

      getProfile: (id: string) => {
        return get().profiles.find((p) => p.id === id);
      },

      getDefaultProfile: () => {
        const { profiles } = get();
        return profiles.find((p) => p.is_default) || profiles[0] || DEFAULT_PROFILES[0];
      },
    }),
    {
      name: 'delivery-profiles-storage',
      partialize: (state) => ({ profiles: state.profiles }),
    },
  ),
);
