'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type ThemeMode = 'light' | 'dark' | 'system';
export type AccentPreset = 'default' | 'ocean' | 'sunset' | 'forest' | 'lavender' | 'rose';

export const ACCENT_PRESETS: Record<AccentPreset, { label: string; primary: string; accent: string; primaryDark: string; primaryLight: string; accentLight: string }> = {
  default: { label: 'Default', primary: '#006A4E', accent: '#F42A41', primaryDark: '#004D38', primaryLight: '#E6F2EE', accentLight: '#FEE8EB' },
  ocean: { label: 'Ocean', primary: '#0369a1', accent: '#06b6d4', primaryDark: '#075985', primaryLight: '#e0f2fe', accentLight: '#cffafe' },
  sunset: { label: 'Sunset', primary: '#ea580c', accent: '#e11d48', primaryDark: '#c2410c', primaryLight: '#fff7ed', accentLight: '#ffe4e6' },
  forest: { label: 'Forest', primary: '#15803d', accent: '#ca8a04', primaryDark: '#166534', primaryLight: '#f0fdf4', accentLight: '#fefce8' },
  lavender: { label: 'Lavender', primary: '#7c3aed', accent: '#ec4899', primaryDark: '#6d28d9', primaryLight: '#f5f3ff', accentLight: '#fce7f3' },
  rose: { label: 'Rose', primary: '#e11d48', accent: '#f59e0b', primaryDark: '#be123c', primaryLight: '#fff1f2', accentLight: '#fef3c7' },
};

interface ThemeState {
  mode: ThemeMode;
  accent: AccentPreset;
  setMode: (mode: ThemeMode) => void;
  setAccent: (accent: AccentPreset) => void;
  resolvedMode: () => 'light' | 'dark';
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      mode: 'light',
      accent: 'default',
      setMode: (mode) => set({ mode }),
      setAccent: (accent) => set({ accent }),
      resolvedMode: () => {
        const { mode } = get();
        if (mode !== 'system') return mode;
        if (typeof window === 'undefined') return 'light';
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
      },
    }),
    { name: 'theme-storage' },
  ),
);
