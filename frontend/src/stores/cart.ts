'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { cartApi, type CartItemResponse } from '@/lib/api';

export interface CartItem {
  productId: string;
  variantId?: string;
  name: string;
  sku: string;
  price: number;
  quantity: number;
  image?: string;
  // Backend item ID (set when synced with cart-service)
  backendId?: string;
}

interface CartAuth {
  userId: string;
  tenantId: string;
  token: string;
}

interface CartState {
  items: CartItem[];
  syncing: boolean;

  addItem: (item: CartItem, auth?: CartAuth) => Promise<void>;
  removeItem: (productId: string, variantId?: string, auth?: CartAuth) => Promise<void>;
  updateQuantity: (productId: string, quantity: number, variantId?: string, auth?: CartAuth) => Promise<void>;
  clearCart: (auth?: CartAuth) => Promise<void>;
  fetchFromBackend: (auth: CartAuth) => Promise<void>;
  total: () => number;
  itemCount: () => number;
}

function fromBackendItem(item: CartItemResponse): CartItem {
  return {
    productId: item.product_id,
    name: item.name,
    sku: item.product_id, // cart-service doesn't store SKU separately
    price: item.price,
    quantity: item.quantity,
    image: item.image_url,
    backendId: item.id,
  };
}

export const useCartStore = create<CartState>()(
  persist(
    (set, get) => ({
      items: [],
      syncing: false,

      fetchFromBackend: async ({ userId, tenantId, token }) => {
        set({ syncing: true });
        try {
          const cart = await cartApi.get(userId, tenantId, token);
          set({
            items: cart.items.map(fromBackendItem),
            syncing: false,
          });
        } catch {
          set({ syncing: false });
        }
      },

      addItem: async (item, auth) => {
        // Optimistic local update
        set((state) => {
          const existing = state.items.find(
            (i) => i.productId === item.productId && i.variantId === item.variantId,
          );
          if (existing) {
            return {
              items: state.items.map((i) =>
                i.productId === item.productId && i.variantId === item.variantId
                  ? { ...i, quantity: i.quantity + item.quantity }
                  : i,
              ),
            };
          }
          return { items: [...state.items, item] };
        });

        if (auth) {
          try {
            const cart = await cartApi.addItem(
              {
                tenant_id: auth.tenantId,
                user_id: auth.userId,
                product_id: item.productId,
                name: item.name,
                price: item.price,
                quantity: item.quantity,
                image_url: item.image,
              },
              auth.tenantId,
              auth.token,
            );
            // Reconcile with server response
            set({ items: cart.items.map(fromBackendItem) });
          } catch {
            // Keep optimistic state on failure
          }
        }
      },

      removeItem: async (productId, variantId, auth) => {
        const item = get().items.find(
          (i) => i.productId === productId && i.variantId === variantId,
        );

        // Optimistic local removal
        set((state) => ({
          items: state.items.filter(
            (i) => !(i.productId === productId && i.variantId === variantId),
          ),
        }));

        if (auth && item?.backendId) {
          try {
            await cartApi.removeItem(item.backendId, auth.tenantId, auth.token);
          } catch {
            // Keep local removal
          }
        }
      },

      updateQuantity: async (productId, quantity, variantId, auth) => {
        const item = get().items.find(
          (i) => i.productId === productId && i.variantId === variantId,
        );

        // Optimistic local update
        set((state) => ({
          items:
            quantity <= 0
              ? state.items.filter(
                  (i) => !(i.productId === productId && i.variantId === variantId),
                )
              : state.items.map((i) =>
                  i.productId === productId && i.variantId === variantId
                    ? { ...i, quantity }
                    : i,
                ),
        }));

        if (auth && item?.backendId) {
          try {
            if (quantity <= 0) {
              await cartApi.removeItem(item.backendId, auth.tenantId, auth.token);
            } else {
              const cart = await cartApi.updateItem(
                item.backendId,
                quantity,
                auth.tenantId,
                auth.token,
              );
              set({ items: cart.items.map(fromBackendItem) });
            }
          } catch {
            // Keep optimistic state
          }
        }
      },

      clearCart: async (auth) => {
        set({ items: [] });
        if (auth) {
          try {
            await cartApi.clear(auth.userId, auth.tenantId, auth.token);
          } catch {
            // Already cleared locally
          }
        }
      },

      total: () => get().items.reduce((sum, i) => sum + i.price * i.quantity, 0),
      itemCount: () => get().items.reduce((sum, i) => sum + i.quantity, 0),
    }),
    { name: 'cart-storage' },
  ),
);
