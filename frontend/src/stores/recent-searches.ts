'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

const MAX_RECENT = 5;

interface RecentSearchesState {
  /** Most-recent-first list of normalised search queries. */
  queries: string[];
  /** Push a query to the front (deduped, trimmed, lower-cased compare). */
  push: (query: string) => void;
  /** Remove a single query from the list. */
  remove: (query: string) => void;
  /** Clear the entire list. */
  clear: () => void;
}

export const useRecentSearchesStore = create<RecentSearchesState>()(
  persist(
    (set) => ({
      queries: [],

      push: (query) => {
        const trimmed = query.trim();
        if (trimmed.length < 2) return;
        set((state) => {
          const lower = trimmed.toLowerCase();
          const without = state.queries.filter((q) => q.toLowerCase() !== lower);
          return { queries: [trimmed, ...without].slice(0, MAX_RECENT) };
        });
      },

      remove: (query) =>
        set((state) => ({
          queries: state.queries.filter(
            (q) => q.toLowerCase() !== query.trim().toLowerCase(),
          ),
        })),

      clear: () => set({ queries: [] }),
    }),
    {
      name: 'recent-searches-storage',
      partialize: (state) => ({ queries: state.queries }),
    },
  ),
);
