'use client';

import { useEffect, useState } from 'react';
import StoreHeader from '@/components/store/header';
import StoreFooter from '@/components/store/footer';
import { tenantApi, type TenantConfig } from '@/lib/api';
import { useStoreConfigStore } from '@/stores/store-config';
import { useThemeStore } from '@/stores/theme';

const TENANT_ID = 'tenant_saajan';

export default function StoreLayout({ children }: { children: React.ReactNode }) {
  const [storeName, setStoreName] = useState('Saajan');
  const [branding, setBranding] = useState<TenantConfig['branding'] | null>(null);
  const fetchConfig = useStoreConfigStore((s) => s.fetchConfig);
  const accent = useThemeStore((s) => s.accent);

  useEffect(() => {
    async function loadTenant() {
      try {
        const tenant = await tenantApi.get(TENANT_ID);
        setStoreName(tenant.name || 'Saajan');
        if (tenant.config?.branding) {
          setBranding(tenant.config.branding);
        }
      } catch {
        // use defaults
      }
      fetchConfig(TENANT_ID);
    }
    loadTenant();
  }, [fetchConfig]);

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
