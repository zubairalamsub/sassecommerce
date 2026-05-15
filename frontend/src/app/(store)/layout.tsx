'use client';

import { useEffect, useState } from 'react';
import StoreHeader from '@/components/store/header';
import StoreFooter from '@/components/store/footer';
import AnnouncementPopup from '@/components/store/announcement-popup';
import { tenantApi, type TenantConfig } from '@/lib/api';
import { useStoreConfigStore } from '@/stores/store-config';
import { useThemeStore } from '@/stores/theme';
import { DEFAULT_TENANT_ID as TENANT_ID } from '@/lib/tenant';

// Cache tenant info on `window` so the result survives Fast Refresh, Turbopack
// module re-evaluation, and any layout remount in dev. Without this guard, each
// time the (store) layout's effect re-fires (StrictMode mount-unmount-mount,
// navigation across the route group, etc.) it would issue a fresh tenant fetch.
type TenantValue = Awaited<ReturnType<typeof tenantApi.get>>;
type TenantCache = { promise: Promise<TenantValue> | null; resolved: TenantValue | null };
function tenantCache(): TenantCache {
  if (typeof window === 'undefined') return { promise: null, resolved: null };
  const w = window as typeof window & { __saajanTenantCache?: TenantCache };
  if (!w.__saajanTenantCache) w.__saajanTenantCache = { promise: null, resolved: null };
  return w.__saajanTenantCache;
}
function loadTenantOnce(): Promise<TenantValue> {
  const cache = tenantCache();
  if (cache.resolved) { console.log('[saajan] tenant RESOLVED-cache'); return Promise.resolve(cache.resolved); }
  if (cache.promise) { console.log('[saajan] tenant PROMISE-cache'); return cache.promise; }
  console.log('[saajan] tenant FRESH FETCH');
  cache.promise = tenantApi.get(TENANT_ID).then(
    (tenant) => {
      cache.resolved = tenant;
      return tenant;
    },
    (err) => {
      // Drop the failed promise so the *next* mount that needs the tenant can
      // retry — this is one retry per mount, not a polling loop.
      cache.promise = null;
      throw err;
    },
  );
  return cache.promise;
}

export default function StoreLayout({ children }: { children: React.ReactNode }) {
  const [storeName, setStoreName] = useState('Demo Store');
  const [branding, setBranding] = useState<TenantConfig['branding'] | null>(null);
  const accent = useThemeStore((s) => s.accent);

  useEffect(() => {
    let cancelled = false;
    loadTenantOnce()
      .then((tenant) => {
        if (cancelled) return;
        setStoreName(tenant.name || 'Demo Store');
        if (tenant.config?.branding) {
          setBranding(tenant.config.branding);
        }
      })
      .catch(() => {
        // Tenant fetch failed — keep fallback storeName/no-branding. The next
        // mount will retry via the cleared cache promise.
      });
    // store-config has its own loaded-once + in-flight de-dupe guards. Pulling
    // it via getState() avoids re-firing this effect on any state change and
    // keeps `fetchConfig` out of the dependency array.
    useStoreConfigStore.getState().fetchConfig(TENANT_ID);
    return () => {
      cancelled = true;
    };
  }, []);

  // Apply tenant branding on <html> only when user hasn't picked a custom accent
  useEffect(() => {
    if (accent !== 'default' || !branding) return;

    const root = document.documentElement;
    const primary = branding.primary_color || '#006A4E';
    const secondary = branding.secondary_color || '#F42A41';

    root.style.setProperty('--color-primary', primary);
    root.style.setProperty('--color-primary-dark', adjustColor(primary, -30));
    root.style.setProperty('--color-primary-light', adjustColor(primary, 200));
    root.style.setProperty('--color-accent', secondary);
  }, [branding, accent]);

  return (
    <div className="flex min-h-screen flex-col bg-surface-secondary">
      <StoreHeader storeName={storeName} logoUrl={branding?.logo_url} />
      <main className="flex-1">{children}</main>
      <StoreFooter storeName={storeName} />
      <AnnouncementPopup />
    </div>
  );
}

function adjustColor(hex: string, amount: number): string {
  hex = hex.replace('#', '');
  if (hex.length === 3) hex = hex.split('').map((c) => c + c).join('');

  const r = Math.max(0, Math.min(255, parseInt(hex.substring(0, 2), 16) + amount));
  const g = Math.max(0, Math.min(255, parseInt(hex.substring(2, 4), 16) + amount));
  const b = Math.max(0, Math.min(255, parseInt(hex.substring(4, 6), 16) + amount));

  return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
}
