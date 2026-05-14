'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

const MAX_RECENT = 12;

interface RecentlyViewedState {
  /** Product IDs, most-recent-first. */
  ids: string[];
  /** Push a product ID to the front and dedupe. */
  push: (productId: string) => void;
  /** Remove a specific product ID (e.g. archived/deleted). */
  remove: (productId: string) => void;
  /** Clear the entire list. */
  clear: () => void;
  /** Return up to `limit` IDs, excluding `excludeId`. */
  getOthers: (excludeId?: string, limit?: number) => string[];
}

export const useRecentlyViewedStore = create<RecentlyViewedState>()(
  persist(
    (set, get) => ({
      ids: [],

      push: (productId) => {
        if (!productId) return;
        set((state) => {
          const without = state.ids.filter((id) => id !== productId);
          return { ids: [productId, ...without].slice(0, MAX_RECENT) };
        });
      },

      remove: (productId) =>
        set((state) => ({ ids: state.ids.filter((id) => id !== productId) })),

      clear: () => set({ ids: [] }),

      getOthers: (excludeId, limit = MAX_RECENT) => {
        const ids = get().ids;
        return (excludeId ? ids.filter((id) => id !== excludeId) : ids).slice(0, limit);
      },
    }),
    {
      name: 'recently-viewed-storage',
      partialize: (state) => ({ ids: state.ids }),
    },
  ),
);
