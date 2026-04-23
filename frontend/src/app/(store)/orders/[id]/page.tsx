'use client';

import { use } from 'react';
import Link from 'next/link';
import {
  CheckCircle,
  Package,
  MapPin,
  CreditCard,
  ArrowRight,
  Clock,
} from 'lucide-react';
import { formatCurrency, formatDate, statusColor } from '@/lib/utils';

interface DemoOrderItem {
  name: string;
  sku: string;
  quantity: number;
  unitPrice: number;
}

interface DemoOrder {
  id: string;
  orderNumber: string;
  status: string;
  date: string;
  items: DemoOrderItem[];
  subtotal: number;
  shipping: number;
  total: number;
  shippingAddress: {
    name: string;
    phone: string;
    street: string;
    city: string;
    postalCode: string;
  };
  paymentMethod: string;
}

function generateDemoOrder(id: string): DemoOrder {
  const items: DemoOrderItem[] = [
    { name: 'Jamdani Saree - White & Gold', sku: 'JAM-SAR-WG', quantity: 1, unitPrice: 15000 },
    { name: 'Cotton Panjabi - L', sku: 'PAN-COT-L', quantity: 2, unitPrice: 3500 },
  ];
  const subtotal = items.reduce((sum, i) => sum + i.unitPrice * i.quantity, 0);
  const shipping = 100;

  return {
    id,
    orderNumber: id.startsWith('ORD-') ? id : `ORD-${id.slice(0, 8).toUpperCase()}`,
    status: 'confirmed',
    date: new Date().toISOString(),
    items,
    subtotal,
    shipping,
    total: subtotal + shipping,
    shippingAddress: {
      name: 'Rahim Uddin',
      phone: '+880 1712345678',
      street: 'House 12, Road 5, Dhanmondi',
      city: 'Dhaka',
      postalCode: '1205',
    },
    paymentMethod: 'bKash',
  };
}

export default function OrderConfirmationPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const order = generateDemoOrder(id);

  return (
    <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Success banner */}
      <div className="rounded-xl bg-green-50 border border-green-200 p-6 text-center">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
          <CheckCircle className="h-8 w-8 text-green-600" />
        </div>
        <h1 className="mt-4 text-2xl font-bold text-green-900">
          Order Placed Successfully!
        </h1>
        <p className="mt-1 text-green-700">
          Thank you for your purchase. We will process your order shortly.
        </p>
      </div>

      {/* Order details */}
      <div className="mt-8 rounded-xl border border-gray-200 bg-white overflow-hidden">
        {/* Order header */}
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-gray-200 px-6 py-4">
          <div>
            <p className="text-sm text-gray-500">Order Number</p>
            <p className="text-lg font-bold text-gray-900">{order.orderNumber}</p>
          </div>
          <div className="text-right">
            <p className="text-sm text-gray-500">Date</p>
            <div className="flex items-center gap-1 text-sm text-gray-700">
              <Clock className="h-3.5 w-3.5" />
              {formatDate(order.date)}
            </div>
          </div>
          <span
            className={`rounded-full px-3 py-1 text-xs font-semibold capitalize ${statusColor(order.status)}`}
          >
            {order.status}
          </span>
        </div>

        {/* Order items */}
        <div className="px-6 py-4">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-gray-900">
            <Package className="h-4 w-4" />
            Items
          </h2>
          <div className="mt-3 divide-y divide-gray-100">
            {order.items.map((item, idx) => (
              <div key={idx} className="flex items-center gap-4 py-3">
                <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-primary/20 to-primary/5">
                  <span className="text-lg font-bold text-primary/40">
                    {item.name.charAt(0)}
                  </span>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-gray-900">{item.name}</p>
                  <p className="text-xs text-gray-500">
                    SKU: {item.sku} &middot; Qty: {item.quantity}
                  </p>
                </div>
                <span className="font-medium text-gray-900">
                  {formatCurrency(item.unitPrice * item.quantity)}
                </span>
              </div>
            ))}
          </div>

          {/* Totals */}
          <div className="mt-4 space-y-2 border-t border-gray-200 pt-4">
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">Subtotal</span>
              <span className="text-gray-900">{formatCurrency(order.subtotal)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">Shipping</span>
              <span className="text-gray-900">{formatCurrency(order.shipping)}</span>
            </div>
            <div className="flex justify-between border-t border-gray-200 pt-2">
              <span className="font-semibold text-gray-900">Total</span>
              <span className="text-lg font-bold text-gray-900">
                {formatCurrency(order.total)}
              </span>
            </div>
          </div>
        </div>

        {/* Shipping & Payment info */}
        <div className="grid grid-cols-1 gap-px bg-gray-200 sm:grid-cols-2">
          <div className="bg-white px-6 py-4">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-900">
              <MapPin className="h-4 w-4" />
              Shipping Address
            </h3>
            <div className="mt-2 space-y-1 text-sm text-gray-600">
              <p className="font-medium text-gray-900">{order.shippingAddress.name}</p>
              <p>{order.shippingAddress.street}</p>
              <p>
                {order.shippingAddress.city}, {order.shippingAddress.postalCode}
              </p>
              <p>{order.shippingAddress.phone}</p>
            </div>
          </div>
          <div className="bg-white px-6 py-4">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-900">
              <CreditCard className="h-4 w-4" />
              Payment Method
            </h3>
            <p className="mt-2 text-sm text-gray-600">{order.paymentMethod}</p>
          </div>
        </div>
      </div>

      {/* Continue Shopping */}
      <div className="mt-8 text-center">
        <Link
          href="/products"
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark"
        >
          Continue Shopping
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>
    </div>
  );
}
