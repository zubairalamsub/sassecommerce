'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { wishlistApi } from '@/lib/api';

export interface WishlistItem {
  productId: string;
  name: string;
  slug: string;
  price: number;
  image?: string;
  addedAt: string;
}

interface WishlistAuth {
  userId: string;
  tenantId: string;
  token: string;
}

interface WishlistState {
  items: WishlistItem[];
  syncing: boolean;

  fetchFromBackend: (auth: WishlistAuth) => Promise<void>;
  addItem: (item: Omit<WishlistItem, 'addedAt'>, auth?: WishlistAuth) => Promise<void>;
  removeItem: (productId: string, auth?: WishlistAuth) => Promise<void>;
  toggleItem: (item: Omit<WishlistItem, 'addedAt'>, auth?: WishlistAuth) => Promise<void>;
  isInWishlist: (productId: string) => boolean;
  clear: (auth?: WishlistAuth) => Promise<void>;
}

export const useWishlistStore = create<WishlistState>()(
  persist(
    (set, get) => ({
      items: [],
      syncing: false,

      fetchFromBackend: async ({ tenantId, token }) => {
        set({ syncing: true });
        try {
          const res = await wishlistApi.get(tenantId, token);
          set({
            items: (res.items ?? []).map((i) => ({
              productId: i.product_id,
              name: i.name,
              slug: i.slug,
              price: i.price,
              image: i.image,
              addedAt: i.added_at,
            })),
            syncing: false,
          });
        } catch {
          set({ syncing: false });
        }
      },

      addItem: async (item, auth) => {
        // Skip if already in wishlist
        if (get().items.some((i) => i.productId === item.productId)) return;

        // Optimistic update
        set((state) => ({
          items: [...state.items, { ...item, addedAt: new Date().toISOString() }],
        }));

        if (auth) {
          try {
            await wishlistApi.addItem(
              {
                product_id: item.productId,
                name: item.name,
                slug: item.slug,
                price: item.price,
                image: item.image,
              },
              auth.tenantId,
              auth.token,
            );
          } catch {
            // Keep optimistic state
          }
        }
      },

      removeItem: async (productId, auth) => {
        // Optimistic removal
        set((state) => ({
          items: state.items.filter((i) => i.productId !== productId),
        }));

        if (auth) {
          try {
            await wishlistApi.removeItem(productId, auth.tenantId, auth.token);
          } catch {
            // Keep local removal
          }
        }
      },

      toggleItem: async (item, auth) => {
        const exists = get().items.some((i) => i.productId === item.productId);
        if (exists) {
          await get().removeItem(item.productId, auth);
        } else {
          await get().addItem(item, auth);
        }
      },

      isInWishlist: (productId) => get().items.some((i) => i.productId === productId),

      clear: async (auth) => {
        set({ items: [] });
        if (auth) {
          try {
            await wishlistApi.clear(auth.tenantId, auth.token);
          } catch {
            // Already cleared locally
          }
        }
      },
    }),
    { name: 'wishlist-storage' },
  ),
);
