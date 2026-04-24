'use client';

import { use, useEffect, useState } from 'react';
import Link from 'next/link';
import { ArrowLeft, Globe, Mail, Calendar, Loader2 } from 'lucide-react';
import { cn, formatDate, statusColor } from '@/lib/utils';
import { tenantApi, type Tenant } from '@/lib/api';

const tierColors: Record<string, string> = {
  free: 'bg-gray-100 text-gray-700',
  starter: 'bg-blue-100 text-blue-700',
  professional: 'bg-indigo-100 text-indigo-700',
  enterprise: 'bg-purple-100 text-purple-700',
};

export default function TenantDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    tenantApi.get(id).then((t) => {
      setTenant(t);
    }).catch(() => {
      setTenant(null);
    }).finally(() => setLoading(false));
  }, [id]);

  if (loading) {
    return (
      <div className="py-16 text-center">
        <Loader2 className="mx-auto h-6 w-6 animate-spin text-indigo-600" />
      </div>
    );
  }

  if (!tenant) {
    return (
      <div className="space-y-4">
        <Link href="/super-admin/tenants" className="inline-flex items-center gap-2 text-sm text-gray-500 hover:text-gray-900">
          <ArrowLeft className="h-4 w-4" /> Back to Tenants
        </Link>
        <div className="rounded-xl border border-gray-200 bg-white p-16 text-center">
          <p className="text-gray-500">Tenant not found.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link href="/super-admin/tenants" className="inline-flex items-center gap-2 text-sm text-gray-500 hover:text-gray-900 transition-colors">
        <ArrowLeft className="h-4 w-4" /> Back to Tenants
      </Link>

      {/* Header */}
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-indigo-100 text-xl font-bold text-indigo-700">
          {tenant.name.charAt(0).toUpperCase()}
        </div>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{tenant.name}</h1>
          <div className="mt-1 flex items-center gap-2">
            <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', tierColors[tenant.tier] ?? 'bg-gray-100 text-gray-700')}>
              {tenant.tier}
            </span>
            <span className={cn('inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize', statusColor(tenant.status))}>
              {tenant.status}
            </span>
          </div>
        </div>
      </div>

      {/* Details Grid */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <div className="flex items-center gap-2 text-sm text-gray-500 mb-3">
            <Mail className="h-4 w-4" /> Contact
          </div>
          <p className="text-sm font-medium text-gray-900">{tenant.email}</p>
          {tenant.config?.general?.contact_phone && (
            <p className="mt-1 text-sm text-gray-500">{tenant.config.general.contact_phone}</p>
          )}
        </div>

        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <div className="flex items-center gap-2 text-sm text-gray-500 mb-3">
            <Globe className="h-4 w-4" /> Domain
          </div>
          <p className="text-sm font-medium text-gray-900 font-mono">
            {tenant.domain || <span className="text-gray-400 not-italic font-normal">Not configured</span>}
          </p>
        </div>

        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <div className="flex items-center gap-2 text-sm text-gray-500 mb-3">
            <Calendar className="h-4 w-4" /> Created
          </div>
          <p className="text-sm font-medium text-gray-900">{formatDate(tenant.created_at)}</p>
        </div>
      </div>

      {/* Config */}
      {tenant.config && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Configuration</h2>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
            {[
              { label: 'Currency', value: tenant.config.general?.currency },
              { label: 'Language', value: tenant.config.general?.language },
              { label: 'Timezone', value: tenant.config.general?.timezone },
              { label: 'Date Format', value: tenant.config.general?.date_format },
            ].filter((item) => item.value).map((item) => (
              <div key={item.label}>
                <p className="text-xs text-gray-500">{item.label}</p>
                <p className="mt-0.5 text-sm font-medium text-gray-900">{item.value}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Features */}
      {tenant.config?.features && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Features</h2>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {Object.entries(tenant.config.features).map(([key, enabled]) => (
              <div key={key} className="flex items-center gap-2">
                <span className={cn('h-2 w-2 rounded-full flex-shrink-0', enabled ? 'bg-green-500' : 'bg-gray-300')} />
                <span className={cn('text-sm capitalize', enabled ? 'text-gray-900' : 'text-gray-400')}>
                  {key.replace(/_/g, ' ')}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
