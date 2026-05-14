'use client';

import { use, useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Loader2, Printer, Download, Package } from 'lucide-react';
import Invoice from '@/components/invoice';
import { orderApi, tenantApi, type Order, type Tenant } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

const TENANT_ID = 'tenant_saajan';

export default function CustomerInvoicePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const [order, setOrder] = useState<Order | null>(null);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  // Auth gate — bounce unauthenticated users to login with deep link preserved.
  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace(`/login?next=${encodeURIComponent(`/orders/${id}/invoice`)}`);
    }
  }, [isAuthenticated, router, id]);

  useEffect(() => {
    if (!user || !token) return;
    let cancelled = false;
    setLoading(true);
    setNotFound(false);

    Promise.all([
      orderApi.get(id, TENANT_ID, token),
      tenantApi.get(TENANT_ID).catch(() => null),
    ])
      .then(([o, t]) => {
        if (cancelled) return;
        // Ownership check — don't leak invoice contents to other customers.
        if (o.customer_id && user.id && o.customer_id !== user.id) {
          setNotFound(true);
          setOrder(null);
        } else {
          setOrder(o);
          setTenant(t);
        }
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
  }, [id, user, token]);

  function handlePrint() {
    window.print();
  }

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (notFound || !order) {
    return (
      <div className="mx-auto max-w-xl px-4 py-16 text-center sm:px-6 lg:px-8">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-surface-hover text-text-muted">
          <Package className="h-8 w-8" />
        </div>
        <h1 className="mt-4 text-2xl font-bold text-text">We couldn&apos;t find this invoice</h1>
        <p className="mt-2 text-sm text-text-secondary">
          It may have been removed, or this order doesn&apos;t belong to your account.
        </p>
        <Link
          href="/account/orders"
          className="mt-6 inline-flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-dark"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to my orders
        </Link>
      </div>
    );
  }

  // Gate by tenant feature flag.
  const invoicesEnabled = tenant?.config?.features?.invoices_enabled === true;
  if (tenant && !invoicesEnabled) {
    return (
      <div className="mx-auto max-w-xl px-4 py-16 text-center sm:px-6 lg:px-8">
        <p className="text-text-secondary">Invoices are not available for this store.</p>
        <Link
          href={`/orders/${order.id}`}
          className="mt-6 inline-flex items-center gap-2 rounded-lg border border-gray-200 px-4 py-2.5 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to order
        </Link>
      </div>
    );
  }

  const customer = user
    ? {
        name: [user.first_name, user.last_name].filter(Boolean).join(' '),
        email: user.email,
        phone: user.phone || '',
      }
    : null;

  return (
    <div className="mx-auto max-w-4xl px-4 py-6 sm:px-6 sm:py-10 lg:px-8">
      <div className="no-print mb-4 flex flex-wrap items-center justify-between gap-3">
        <Link
          href={`/orders/${order.id}`}
          className="inline-flex items-center gap-1.5 text-sm text-text-secondary hover:text-text transition-colors"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to order
        </Link>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={handlePrint}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-dark"
          >
            <Printer className="h-4 w-4" />
            Print
          </button>
          <button
            type="button"
            onClick={handlePrint}
            title="Opens the print dialog — choose Save as PDF as the destination."
            className="inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-surface px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover"
          >
            <Download className="h-4 w-4" />
            Download PDF
          </button>
        </div>
      </div>

      <Invoice order={order} tenant={tenant} customer={customer} />
    </div>
  );
}
