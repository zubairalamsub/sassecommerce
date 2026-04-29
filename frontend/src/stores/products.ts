'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import {
  productApi,
  categoryApi,
  type Product,
  type CreateProductRequest,
  type Category,
  type CreateCategoryRequest,
  type UpdateCategoryRequest,
} from '@/lib/api';

export type StoreProduct = Product;

interface ProductStore {
  products: StoreProduct[];
  categories: Category[];
  loading: boolean;
  error: string | null;
  fetchProducts: (tenantId: string) => Promise<void>;
  fetchCategories: (tenantId: string) => Promise<void>;
  addProduct: (data: CreateProductRequest, tenantId: string, token?: string) => Promise<StoreProduct>;
  deleteProduct: (id: string, tenantId: string, token?: string) => Promise<void>;
  updateProduct: (id: string, data: Partial<CreateProductRequest> & { updated_by: string }, tenantId: string, token?: string) => Promise<void>;
  addCategory: (data: CreateCategoryRequest, tenantId: string) => Promise<void>;
  updateCategory: (id: string, data: UpdateCategoryRequest, tenantId: string) => Promise<void>;
  updateCategoryStatus: (id: string, status: 'active' | 'inactive', tenantId: string) => Promise<void>;
  deleteCategory: (id: string, tenantId: string) => Promise<void>;
}

export const useProductStore = create<ProductStore>()(
  persist(
    (set, get) => ({
      products: [],
      categories: [],
      loading: false,
      error: null,

      fetchProducts: async (tenantId: string) => {
        set({ loading: true, error: null });
        try {
          const res = await productApi.list(tenantId, 1, 100);
          const apiProducts = res.data || [];
          if (apiProducts.length > 0) {
            // Merge: keep local-only products that don't exist on the backend
            const apiIds = new Set(apiProducts.map((p: StoreProduct) => p.id));
            const localOnly = get().products.filter((p) => !apiIds.has(p.id));
            set({ products: [...apiProducts, ...localOnly], loading: false });
          } else {
            // API returned empty — keep locally persisted data
            set({ loading: false });
          }
        } catch {
          // Backend unavailable — keep localStorage data
          set({ loading: false });
        }
      },

      fetchCategories: async (tenantId: string) => {
        try {
          const res = await categoryApi.list(tenantId);
          const apiCategories = res.data || [];
          if (apiCategories.length > 0) {
            const apiIds = new Set(apiCategories.map((c: Category) => c.id));
            const localOnly = get().categories.filter((c) => !apiIds.has(c.id));
            set({ categories: [...apiCategories, ...localOnly] });
          }
          // If empty, keep locally persisted data
        } catch {
          // Backend unavailable — keep previously persisted data
        }
      },

      addProduct: async (data: CreateProductRequest, tenantId: string, token?: string) => {
        const product = await productApi.create(data, tenantId, token);
        set((state) => ({ products: [product, ...state.products] }));
        return product;
      },

      deleteProduct: async (id: string, tenantId: string, token?: string) => {
        await productApi.delete(id, tenantId, token);
        set((state) => ({
          products: state.products.filter((p) => p.id !== id),
        }));
      },

      updateProduct: async (id: string, data: Partial<CreateProductRequest> & { updated_by: string }, tenantId: string, token?: string) => {
        const updated = await productApi.update(id, data, tenantId, token);
        set((state) => ({
          products: state.products.map((p) => (p.id === id ? updated : p)),
        }));
      },

      addCategory: async (data: CreateCategoryRequest, tenantId: string) => {
        try {
          await categoryApi.create(data, tenantId);
          await get().fetchCategories(tenantId);
        } catch {
          // Backend unavailable — save locally
          const now = new Date().toISOString();
          const newCat: Category = {
            id: `cat-${Date.now()}`,
            tenant_id: tenantId,
            name: data.name,
            slug: data.slug,
            description: data.description || '',
            parent_id: data.parent_id || null,
            image_url: null,
            status: 'active',
            created_at: now,
            updated_at: now,
          };
          set((state) => ({ categories: [...state.categories, newCat] }));
        }
      },

      updateCategory: async (id: string, data: UpdateCategoryRequest, tenantId: string) => {
        try {
          await categoryApi.update(id, data, tenantId);
          await get().fetchCategories(tenantId);
        } catch {
          // Backend unavailable — update locally
          set((state) => ({
            categories: state.categories.map((c) =>
              c.id === id
                ? { ...c, ...data, parent_id: data.parent_id !== undefined ? data.parent_id ?? null : c.parent_id, updated_at: new Date().toISOString() }
                : c,
            ),
          }));
        }
      },

      updateCategoryStatus: async (id: string, status: 'active' | 'inactive', tenantId: string) => {
        try {
          await categoryApi.updateStatus(id, status, tenantId);
          await get().fetchCategories(tenantId);
        } catch {
          // Backend unavailable — update locally
          set((state) => ({
            categories: state.categories.map((c) =>
              c.id === id ? { ...c, status, updated_at: new Date().toISOString() } : c,
            ),
          }));
        }
      },

      deleteCategory: async (id: string, tenantId: string) => {
        try {
          await categoryApi.delete(id, tenantId);
          await get().fetchCategories(tenantId);
        } catch {
          // Backend unavailable — delete locally
          set((state) => ({
            categories: state.categories.filter((c) => c.id !== id && c.parent_id !== id),
          }));
        }
      },
    }),
    {
      name: 'product-storage',
      partialize: (state) => ({
        products: state.products,
        categories: state.categories,
      }),
    },
  ),
);
