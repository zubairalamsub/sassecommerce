'use client';

import Link from 'next/link';
import { Package, ArrowLeft, ChevronRight, ShoppingBag } from 'lucide-react';
import AuthGuard from '@/components/auth/auth-guard';
import { formatCurrency, formatDate, statusColor } from '@/lib/utils';

interface DemoOrder {
  id: string;
  date: string;
  status: string;
  items: number;
  total: number;
  products: string[];
}

const DEMO_ORDERS: DemoOrder[] = [
  {
    id: 'ORD-2026-001',
    date: '2026-04-15T10:30:00Z',
    status: 'delivered',
    items: 2,
    total: 18500,
    products: ['Jamdani Saree - Royal Blue', 'Cotton Panjabi - White'],
  },
  {
    id: 'ORD-2026-005',
    date: '2026-04-12T14:20:00Z',
    status: 'shipped',
    items: 1,
    total: 3500,
    products: ['Leather Wallet - Brown'],
  },
  {
    id: 'ORD-2026-009',
    date: '2026-04-08T09:15:00Z',
    status: 'confirmed',
    items: 3,
    total: 27000,
    products: ['Silk Kameez Set', 'Handloom Shawl', 'Embroidered Clutch'],
  },
  {
    id: 'ORD-2026-014',
    date: '2026-03-25T16:45:00Z',
    status: 'delivered',
    items: 1,
    total: 45000,
    products: ['Dhakai Muslin Saree - Heritage Collection'],
  },
];

function OrdersContent() {
  const orders = DEMO_ORDERS;

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
      {/* Header */}
      <div className="mb-8 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href="/account"
            className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">My Orders</h1>
            <p className="text-sm text-gray-500">
              {orders.length} {orders.length === 1 ? 'order' : 'orders'} placed
            </p>
          </div>
        </div>
      </div>

      {/* Orders List */}
      {orders.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-gray-200 bg-white py-16 text-center shadow-sm">
          <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100">
            <ShoppingBag className="h-8 w-8 text-gray-400" />
          </div>
          <h2 className="text-lg font-semibold text-gray-900">No orders yet</h2>
          <p className="mt-1 text-sm text-gray-500">
            When you place orders, they will appear here.
          </p>
          <Link
            href="/products"
            className="mt-6 rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-dark"
          >
            Start Shopping
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          {orders.map((order) => (
            <div
              key={order.id}
              className="group rounded-xl border border-gray-200 bg-white p-5 shadow-sm transition-all hover:border-primary/30 hover:shadow-md"
            >
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                {/* Left: order info */}
                <div className="flex items-start gap-4">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
                    <Package className="h-5 w-5" />
                  </div>
                  <div>
                    <div className="flex items-center gap-3">
                      <p className="text-sm font-semibold text-gray-900">{order.id}</p>
                      <span
                        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${statusColor(order.status)}`}
                      >
                        {order.status}
                      </span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500">
                      Placed on {formatDate(order.date)}
                    </p>
                    <p className="mt-1.5 text-sm text-gray-600 line-clamp-1">
                      {order.products.join(', ')}
                    </p>
                  </div>
                </div>

                {/* Right: total + items */}
                <div className="flex items-center justify-between sm:flex-col sm:items-end sm:gap-1">
                  <div className="text-right">
                    <p className="text-sm font-bold text-gray-900">
                      {formatCurrency(order.total)}
                    </p>
                    <p className="text-xs text-gray-500">
                      {order.items} {order.items === 1 ? 'item' : 'items'}
                    </p>
                  </div>
                  <ChevronRight className="h-4 w-4 text-gray-400 transition-transform group-hover:translate-x-0.5 sm:mt-1" />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function AccountOrdersPage() {
  return (
    <AuthGuard requiredRole="customer">
      <OrdersContent />
    </AuthGuard>
  );
}
