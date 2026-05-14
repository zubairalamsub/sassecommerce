'use client';

import { mediaUrl, formatCurrency, formatDate, cn, statusColor } from '@/lib/utils';
import type { Order, Tenant } from '@/lib/api';

interface InvoiceProps {
  order: Order;
  tenant: Tenant | null;
  /** Optional customer info used to populate the "Bill to" section.
   * Falls back to the order's shipping address fields when missing. */
  customer?: {
    name?: string;
    email?: string;
    phone?: string;
  } | null;
  /** Payment metadata — optional; we render placeholders if absent. */
  payment?: {
    method?: string;
    status?: string;
  } | null;
}

/** Rough heuristic — POS / paid orders shouldn't show "Due". */
function isPaid(order: Order): boolean {
  return order.status === 'shipped' || order.status === 'delivered';
}

/** Discount: backend doesn't expose a discount column on Order; fall back to
 *  subtotal+shipping+tax minus total, which surfaces any coupon-applied savings. */
function discountAmount(order: Order): number {
  const expected = (order.subtotal ?? 0) + (order.shipping_cost ?? 0) + (order.tax ?? 0);
  const diff = expected - (order.total ?? 0);
  return diff > 0.01 ? diff : 0;
}

export default function Invoice({ order, tenant, customer, payment }: InvoiceProps) {
  const branding = tenant?.config?.branding;
  const general = tenant?.config?.general;
  const tenantName = tenant?.name || 'Saajan';
  const logoUrl = branding?.logo_url ? mediaUrl(branding.logo_url) : '';

  const paid = isPaid(order);
  const discount = discountAmount(order);

  const customerName = customer?.name?.trim() || order.customer_id || 'Customer';
  const customerEmail = customer?.email || '';
  const customerPhone = customer?.phone || '';

  return (
    <div className="invoice-print mx-auto max-w-[210mm] bg-white p-8 text-gray-900 shadow-sm sm:p-10 print:p-0 print:shadow-none">
      {/* 1. Header */}
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-gray-200 pb-6">
        <div className="flex items-center gap-3">
          {logoUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={logoUrl}
              alt={`${tenantName} logo`}
              className="h-20 w-auto object-contain"
            />
          ) : (
            <div className="text-2xl font-bold text-primary">{tenantName}</div>
          )}
        </div>
        <div className="text-right">
          <h1 className="text-3xl font-bold tracking-tight text-primary">INVOICE</h1>
          <p className="mt-1 text-sm text-gray-600">
            <span className="font-medium text-gray-800">{order.order_number}</span>
          </p>
          <p className="mt-0.5 text-xs text-gray-500">
            Issued {formatDate(order.created_at)}
          </p>
        </div>
      </header>

      {/* 2. From / To */}
      <section className="mt-6 grid grid-cols-1 gap-6 sm:grid-cols-2">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-gray-500">From</p>
          <div className="mt-2 space-y-0.5 text-sm text-gray-700">
            <p className="font-semibold text-gray-900">{tenantName}</p>
            <p>Dhaka, Bangladesh</p>
            {general?.contact_email && <p>{general.contact_email}</p>}
            {general?.contact_phone && <p>{general.contact_phone}</p>}
          </div>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-gray-500">Bill to</p>
          <div className="mt-2 space-y-0.5 text-sm text-gray-700">
            <p className="font-semibold text-gray-900">{customerName}</p>
            {order.shipping_address?.street && <p>{order.shipping_address.street}</p>}
            {(order.shipping_address?.city || order.shipping_address?.postal_code) && (
              <p>
                {[order.shipping_address?.city, order.shipping_address?.postal_code]
                  .filter(Boolean)
                  .join(', ')}
              </p>
            )}
            {order.shipping_address?.state && <p>{order.shipping_address.state}</p>}
            {order.shipping_address?.country && <p>{order.shipping_address.country}</p>}
            {customerEmail && <p className="mt-1">{customerEmail}</p>}
            {customerPhone && <p>{customerPhone}</p>}
          </div>
        </div>
      </section>

      {/* 3. Order metadata */}
      <section className="mt-6 grid grid-cols-2 gap-4 rounded-md border border-gray-200 bg-gray-50 p-4 text-sm sm:grid-cols-4">
        <div>
          <p className="text-xs uppercase tracking-wider text-gray-500">Invoice #</p>
          <p className="mt-1 font-medium text-gray-900">{order.order_number}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wider text-gray-500">Issue date</p>
          <p className="mt-1 font-medium text-gray-900">{formatDate(order.created_at)}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wider text-gray-500">Due date</p>
          <p className="mt-1 font-medium text-gray-900">
            {paid ? 'PAID' : formatDate(order.created_at)}
          </p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wider text-gray-500">Status</p>
          <p className="mt-1">
            <span
              className={cn(
                'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                statusColor(order.status),
              )}
            >
              {order.status}
            </span>
          </p>
        </div>
      </section>

      {/* 4. Line items */}
      <section className="mt-6">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b-2 border-gray-300 text-left">
              <th className="py-2 pr-2 text-xs font-semibold uppercase tracking-wider text-gray-600">SKU</th>
              <th className="py-2 pr-2 text-xs font-semibold uppercase tracking-wider text-gray-600">
                Description
              </th>
              <th className="py-2 pr-2 text-right text-xs font-semibold uppercase tracking-wider text-gray-600">
                Qty
              </th>
              <th className="py-2 pr-2 text-right text-xs font-semibold uppercase tracking-wider text-gray-600">
                Unit price
              </th>
              <th className="py-2 text-right text-xs font-semibold uppercase tracking-wider text-gray-600">
                Total
              </th>
            </tr>
          </thead>
          <tbody>
            {order.items.map((item) => (
              <tr key={item.id || `${item.product_id}-${item.sku}`} className="border-b border-gray-200">
                <td className="py-3 pr-2 align-top font-mono text-xs text-gray-700">{item.sku || '—'}</td>
                <td className="py-3 pr-2 align-top text-gray-900">
                  <div className="font-medium">{item.name}</div>
                  {item.variant_id && (
                    <div className="mt-0.5 text-xs text-gray-500">Variant: {item.variant_id}</div>
                  )}
                </td>
                <td className="py-3 pr-2 text-right align-top text-gray-700">{item.quantity}</td>
                <td className="py-3 pr-2 text-right align-top text-gray-700">
                  {formatCurrency(item.unit_price, order.currency)}
                </td>
                <td className="py-3 text-right align-top font-medium text-gray-900">
                  {formatCurrency(
                    item.total_price || item.unit_price * item.quantity,
                    order.currency,
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {/* 5. Totals */}
      <section className="mt-6 flex justify-end">
        <dl className="w-full max-w-xs space-y-1.5 text-sm">
          <div className="flex justify-between">
            <dt className="text-gray-600">Subtotal</dt>
            <dd className="text-gray-900">{formatCurrency(order.subtotal, order.currency)}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-gray-600">Shipping</dt>
            <dd className="text-gray-900">
              {order.shipping_cost > 0 ? formatCurrency(order.shipping_cost, order.currency) : 'Free'}
            </dd>
          </div>
          {discount > 0 && (
            <div className="flex justify-between">
              <dt className="text-gray-600">Discount</dt>
              <dd className="text-gray-900">-{formatCurrency(discount, order.currency)}</dd>
            </div>
          )}
          {order.tax > 0 && (
            <div className="flex justify-between">
              <dt className="text-gray-600">Tax</dt>
              <dd className="text-gray-900">{formatCurrency(order.tax, order.currency)}</dd>
            </div>
          )}
          <div className="mt-2 flex justify-between border-t border-gray-300 pt-2">
            <dt className="text-base font-semibold text-gray-900">Grand total</dt>
            <dd className="text-xl font-bold text-primary">
              {formatCurrency(order.total, order.currency)}
            </dd>
          </div>
        </dl>
      </section>

      {/* 6. Payment details */}
      <section className="mt-6 border-t border-gray-200 pt-4">
        <p className="text-xs font-semibold uppercase tracking-wider text-gray-500">Payment</p>
        <div className="mt-2 flex flex-wrap items-center gap-x-6 gap-y-1 text-sm text-gray-700">
          <div>
            <span className="text-gray-500">Method:</span>{' '}
            <span className="font-medium text-gray-900">{payment?.method || 'Cash on delivery'}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-gray-500">Status:</span>
            <span
              className={cn(
                'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                statusColor(payment?.status || (paid ? 'Completed' : 'Pending')),
              )}
            >
              {payment?.status || (paid ? 'Completed' : 'Pending')}
            </span>
          </div>
        </div>
      </section>

      {/* 7. Footer */}
      <footer className="mt-10 border-t border-gray-200 pt-6 text-center">
        <p className="text-sm font-medium text-gray-900">Thank you for shopping with {tenantName}.</p>
        {(general?.contact_email || general?.contact_phone || general?.support_url) && (
          <p className="mt-1 text-xs text-gray-600">
            Need help? Reach us at{' '}
            {general?.contact_email && <span>{general.contact_email}</span>}
            {general?.contact_email && general?.contact_phone && <span> · </span>}
            {general?.contact_phone && <span>{general.contact_phone}</span>}
          </p>
        )}
        <p className="mt-4 text-[11px] text-gray-400">
          This invoice was generated electronically and does not require a signature.
        </p>
      </footer>
    </div>
  );
}
