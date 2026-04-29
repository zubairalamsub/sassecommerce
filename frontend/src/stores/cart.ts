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
    sku: item.product_id,
    price: item.price,
    quantity: item.quantity,
    image: item.image_url,
    backendId: item.id,
  };
}

// Tracks local mutations so an in-flight fetchFromBackend does not overwrite
// optimistic updates (e.g. user removes an item while the initial fetch is
// still pending).
let _mutationVersion = 0;

export const useCartStore = create<CartState>()(
  persist(
    (set, get) => ({
      items: [],
      syncing: false,

      fetchFromBackend: async ({ userId, tenantId, token }) => {
        set({ syncing: true });
        const versionBefore = _mutationVersion;
        try {
          const cart = await cartApi.get(userId, tenantId, token);
          // Only apply backend state if no local mutations happened during the fetch
          if (_mutationVersion === versionBefore) {
            set({ items: cart.items.map(fromBackendItem), syncing: false });
          } else {
            set({ syncing: false });
          }
        } catch {
          // Backend unreachable — keep local items as fallback
          set({ syncing: false });
        }
      },

      addItem: async (item, auth) => {
        _mutationVersion++;
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
            // Backend responded — use its state as truth
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

        _mutationVersion++;
        // Optimistic local removal
        set((state) => ({
          items: state.items.filter(
            (i) => !(i.productId === productId && i.variantId === variantId),
          ),
        }));

        if (auth && item?.backendId) {
          try {
            const cart = await cartApi.removeItem(item.backendId, auth.userId, auth.tenantId, auth.token);
            // Backend responded — use its state as truth
            set({ items: cart.items.map(fromBackendItem) });
          } catch {
            // Keep local removal
          }
        }
      },

      updateQuantity: async (productId, quantity, variantId, auth) => {
        const item = get().items.find(
          (i) => i.productId === productId && i.variantId === variantId,
        );

        _mutationVersion++;
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
              const cart = await cartApi.removeItem(item.backendId, auth.userId, auth.tenantId, auth.token);
              set({ items: cart.items.map(fromBackendItem) });
            } else {
              const cart = await cartApi.updateItem(
                item.backendId,
                quantity,
                auth.userId,
                auth.tenantId,
                auth.token,
              );
              // Backend responded — use its state as truth
              set({ items: cart.items.map(fromBackendItem) });
            }
          } catch {
            // Keep optimistic state
          }
        }
      },

      clearCart: async (auth) => {
        _mutationVersion++;
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
