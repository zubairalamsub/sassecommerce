'use client';

import { useState, useEffect } from 'react';
import { Eye, Loader2 } from 'lucide-react';
import Link from 'next/link';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';
import { orderApi, type Order } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

type OrderStatus = 'pending' | 'confirmed' | 'shipped' | 'delivered' | 'cancelled';

const tabs: { label: string; value: OrderStatus | 'all' }[] = [
  { label: 'All', value: 'all' },
  { label: 'Pending', value: 'pending' },
  { label: 'Confirmed', value: 'confirmed' },
  { label: 'Shipped', value: 'shipped' },
  { label: 'Delivered', value: 'delivered' },
  { label: 'Cancelled', value: 'cancelled' },
];

const demoOrders = [
  { id: 'ORD-2026-001', customer_id: 'cu-001', order_number: 'ORD-2026-001', customer: 'Rahim Uddin', items: 3, total: 4500, status: 'delivered' as OrderStatus, date: '2026-04-17' },
  { id: 'ORD-2026-002', customer_id: 'cu-002', order_number: 'ORD-2026-002', customer: 'Fatima Akter', items: 1, total: 15000, status: 'shipped' as OrderStatus, date: '2026-04-17' },
  { id: 'ORD-2026-003', customer_id: 'cu-003', order_number: 'ORD-2026-003', customer: 'Kamal Hossain', items: 2, total: 3200, status: 'confirmed' as OrderStatus, date: '2026-04-16' },
  { id: 'ORD-2026-004', customer_id: 'cu-004', order_number: 'ORD-2026-004', customer: 'Nusrat Jahan', items: 4, total: 8750, status: 'pending' as OrderStatus, date: '2026-04-16' },
  { id: 'ORD-2026-005', customer_id: 'cu-005', order_number: 'ORD-2026-005', customer: 'Shakib Ahmed', items: 1, total: 2100, status: 'cancelled' as OrderStatus, date: '2026-04-15' },
  { id: 'ORD-2026-006', customer_id: 'cu-006', order_number: 'ORD-2026-006', customer: 'Tahmina Begum', items: 2, total: 6300, status: 'pending' as OrderStatus, date: '2026-04-15' },
];

interface DisplayOrder {
  id: string;
  order_number: string;
  customer: string;
  items: number;
  total: number;
  status: OrderStatus;
  date: string;
}

export default function OrdersPage() {
  const [activeTab, setActiveTab] = useState<OrderStatus | 'all'>('all');
  const [orders, setOrders] = useState<DisplayOrder[]>(demoOrders);
  const [loading, setLoading] = useState(true);
  const { tenantId } = useAuthStore();

  useEffect(() => {
    async function loadOrders() {
      if (!tenantId) {
        setOrders(demoOrders);
        setLoading(false);
        return;
      }
      try {
        const res = await orderApi.listByTenant(tenantId);
        const mapped: DisplayOrder[] = res.data.map((o: Order) => ({
          id: o.id,
          order_number: o.order_number,
          customer: o.customer_id,
          items: o.items?.length || 0,
          total: o.total,
          status: o.status,
          date: o.created_at?.split('T')[0] || '',
        }));
        setOrders(mapped.length > 0 ? mapped : demoOrders);
      } catch {
        setOrders(demoOrders);
      } finally {
        setLoading(false);
      }
    }
    loadOrders();
  }, [tenantId]);

  const filtered =
    activeTab === 'all' ? orders : orders.filter((o) => o.status === activeTab);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-text">Orders</h1>
        <p className="mt-1 text-sm text-text-secondary">
          Manage and track all customer orders.
        </p>
      </div>

      <div className="border-b border-border">
        <nav className="-mb-px flex gap-6">
          {tabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => setActiveTab(tab.value)}
              className={cn(
                'border-b-2 pb-3 text-sm font-medium transition-colors',
                activeTab === tab.value
                  ? 'border-primary text-primary'
                  : 'border-transparent text-text-secondary hover:border-border hover:text-text',
              )}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      <div className="rounded-xl border border-border bg-surface shadow-sm">
        {loading ? (
          <div className="py-16 text-center">
            <Loader2 className="mx-auto h-6 w-6 animate-spin text-primary" />
            <p className="mt-2 text-sm text-text-muted">Loading orders...</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border text-left text-sm text-text-secondary">
                  <th className="px-6 py-3 font-medium">Order #</th>
                  <th className="px-6 py-3 font-medium">Customer</th>
                  <th className="px-6 py-3 font-medium">Items</th>
                  <th className="px-6 py-3 font-medium">Total</th>
                  <th className="px-6 py-3 font-medium">Status</th>
                  <th className="px-6 py-3 font-medium">Date</th>
                  <th className="px-6 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((order) => (
                  <tr
                    key={order.id}
                    className="border-b border-border-light transition-colors hover:bg-surface-hover"
                  >
                    <td className="px-6 py-4 text-sm font-medium text-primary">
                      <Link href={`/admin/orders/${order.id}`}>{order.order_number}</Link>
                    </td>
                    <td className="px-6 py-4 text-sm text-text">{order.customer}</td>
                    <td className="px-6 py-4 text-sm text-text-secondary">
                      {order.items} {order.items === 1 ? 'item' : 'items'}
                    </td>
                    <td className="px-6 py-4 text-sm text-text">
                      {formatCurrency(order.total)}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          statusColor(order.status),
                        )}
                      >
                        {order.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-text-secondary">
                      {formatDate(order.date)}
                    </td>
                    <td className="px-6 py-4">
                      <Link
                        href={`/admin/orders/${order.id}`}
                        className="inline-flex items-center gap-1 rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-text"
                      >
                        <Eye className="h-4 w-4" />
                      </Link>
                    </td>
                  </tr>
                ))}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-6 py-12 text-center text-sm text-text-muted">
                      No orders found.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
