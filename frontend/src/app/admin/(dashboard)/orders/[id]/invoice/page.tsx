'use client';

import { use, useEffect, useState } from 'react';
import Link from 'next/link';
import { ArrowLeft, Loader2, Printer, Download, PackageX } from 'lucide-react';
import Invoice from '@/components/invoice';
import { orderApi, tenantApi, type Order, type Tenant } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

export default function AdminInvoicePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { tenantId, token } = useAuthStore();
  const [order, setOrder] = useState<Order | null>(null);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!tenantId) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setNotFound(false);

    Promise.all([
      orderApi.get(id, tenantId, token || undefined),
      tenantApi.get(tenantId).catch(() => null),
    ])
      .then(([o, t]) => {
        if (cancelled) return;
        setOrder(o);
        setTenant(t);
      })
      .catch(() => {
        if (!cancelled) setNotFound(true);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [id, tenantId, token]);

  function handlePrint() {
    window.print();
  }

  if (loading) {
    return (
      <div className="py-16 text-center">
        <Loader2 className="mx-auto h-6 w-6 animate-spin text-primary" />
        <p className="mt-2 text-sm text-text-muted">Loading invoice...</p>
      </div>
    );
  }

  if (notFound || !order) {
    return (
      <div className="space-y-4">
        <Link
          href={`/admin/orders/${id}`}
          className="inline-flex items-center gap-1 text-sm text-text-secondary hover:text-text"
        >
          <ArrowLeft className="h-4 w-4" /> Back to order
        </Link>
        <div className="flex flex-col items-center rounded-xl border border-border bg-surface p-16 text-center">
          <PackageX className="h-10 w-10 text-text-muted" />
          <p className="mt-3 text-text-secondary">Order not found.</p>
        </div>
      </div>
    );
  }

  // Gate by tenant feature flag.
  const invoicesEnabled = tenant?.config?.features?.invoices_enabled === true;
  if (tenant && !invoicesEnabled) {
    return (
      <div className="space-y-4">
        <Link
          href={`/admin/orders/${order.id}`}
          className="inline-flex items-center gap-1 text-sm text-text-secondary hover:text-text"
        >
          <ArrowLeft className="h-4 w-4" /> Back to order
        </Link>
        <div className="rounded-xl border border-border bg-surface p-10 text-center">
          <p className="text-text-secondary">Invoices are not enabled for this store.</p>
          <p className="mt-1 text-sm text-text-muted">
            Enable invoices from Settings &rarr; Features to use this page.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Action bar — hidden in print */}
      <div className="no-print flex flex-wrap items-center justify-between gap-3">
        <Link
          href={`/admin/orders/${order.id}`}
          className="inline-flex items-center gap-1 text-sm text-text-secondary transition-colors hover:text-text"
        >
          <ArrowLeft className="h-4 w-4" /> Back to order
        </Link>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={handlePrint}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
          >
            <Printer className="h-4 w-4" />
            Print
          </button>
          <button
            type="button"
            onClick={handlePrint}
            title="Opens the print dialog — choose Save as PDF as the destination."
            className="inline-flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover"
          >
            <Download className="h-4 w-4" />
            Download PDF
          </button>
        </div>
      </div>

      <Invoice order={order} tenant={tenant} />
    </div>
  );
}
