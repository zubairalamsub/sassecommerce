'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface WishlistItem {
  productId: string;
  name: string;
  slug: string;
  price: number;
  image?: string;
  addedAt: string;
}

interface WishlistState {
  items: WishlistItem[];
  addItem: (item: Omit<WishlistItem, 'addedAt'>) => void;
  removeItem: (productId: string) => void;
  toggleItem: (item: Omit<WishlistItem, 'addedAt'>) => void;
  isInWishlist: (productId: string) => boolean;
  clear: () => void;
}

export const useWishlistStore = create<WishlistState>()(
  persist(
    (set, get) => ({
      items: [],

      addItem: (item) => {
        if (get().items.some((i) => i.productId === item.productId)) return;
        set((state) => ({
          items: [...state.items, { ...item, addedAt: new Date().toISOString() }],
        }));
      },

      removeItem: (productId) => {
        set((state) => ({
          items: state.items.filter((i) => i.productId !== productId),
        }));
      },

      toggleItem: (item) => {
        const exists = get().items.some((i) => i.productId === item.productId);
        if (exists) {
          get().removeItem(item.productId);
        } else {
          get().addItem(item);
        }
      },

      isInWishlist: (productId) => get().items.some((i) => i.productId === productId),

      clear: () => set({ items: [] }),
    }),
    { name: 'wishlist-storage' },
  ),
);
