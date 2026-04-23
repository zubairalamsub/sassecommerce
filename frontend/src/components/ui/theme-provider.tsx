'use client';

import { useEffect, useLayoutEffect } from 'react';
import { useThemeStore, ACCENT_PRESETS } from '@/stores/theme';

const useIsomorphicLayout = typeof window !== 'undefined' ? useLayoutEffect : useEffect;

export default function ThemeProvider({ children }: { children: React.ReactNode }) {
  const { mode, accent, resolvedMode } = useThemeStore();

  useIsomorphicLayout(() => {
    const resolved = resolvedMode();
    const root = document.documentElement;

    root.classList.toggle('dark', resolved === 'dark');

    // Only override colors when user has explicitly picked a non-default accent
    if (accent !== 'default') {
      const preset = ACCENT_PRESETS[accent];
      root.style.setProperty('--color-primary', preset.primary);
      root.style.setProperty('--color-primary-dark', preset.primaryDark);
      root.style.setProperty('--color-primary-light', preset.primaryLight);
      root.style.setProperty('--color-accent', preset.accent);
      root.style.setProperty('--color-accent-light', preset.accentLight);
    }
  }, [mode, accent, resolvedMode]);

  useEffect(() => {
    if (mode !== 'system') return;
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => {
      document.documentElement.classList.toggle('dark', mq.matches);
    };
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [mode]);

  return <>{children}</>;
}
