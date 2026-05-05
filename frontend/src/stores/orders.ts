'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { orderApi, type Order } from '@/lib/api';

export type OrderStatus = 'pending' | 'confirmed' | 'shipped' | 'delivered' | 'cancelled';

export interface DisplayOrder {
  id: string;
  order_number: string;
  customer: string;
  items: number;
  total: number;
  status: OrderStatus;
  date: string;
}

interface OrderStore {
  orders: DisplayOrder[];
  loading: boolean;
  error: string | null;
  fetchOrders: (tenantId: string, token?: string) => Promise<void>;
}

export const useOrderStore = create<OrderStore>()(
  persist(
    (set) => ({
      orders: [],
      loading: false,
      error: null,

      fetchOrders: async (tenantId: string, token?: string) => {
        set({ loading: true, error: null });
        try {
          const res = await orderApi.listByTenant(tenantId, token);
          const orders = res.data || [];
          // List endpoint returns total_amount=0; fetch each order for real totals
          const detailed = await Promise.all(
            orders.map((o) =>
              orderApi.get(o.id, tenantId, token).catch(() => o)
            )
          );
          const mapped: DisplayOrder[] = detailed.map((o: Order) => ({
            id: o.id,
            order_number: o.order_number,
            customer: o.customer_id,
            items: o.items?.length || 0,
            total: o.total,
            status: o.status as OrderStatus,
            date: o.created_at?.split('T')[0] || '',
          }));
          set({ orders: mapped, loading: false });
        } catch {
          // Backend unavailable — keep previously persisted data
          set({ loading: false });
        }
      },
    }),
    {
      name: 'order-storage',
      partialize: (state) => ({ orders: state.orders }),
    },
  ),
);
