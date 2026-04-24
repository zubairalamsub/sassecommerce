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
  addProduct: (data: CreateProductRequest, tenantId: string) => Promise<StoreProduct>;
  deleteProduct: (id: string, tenantId: string) => Promise<void>;
  updateProduct: (id: string, data: Partial<CreateProductRequest> & { updated_by: string }, tenantId: string) => Promise<void>;
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
          set({ products: res.data || [], loading: false });
        } catch {
          set({ loading: false });
        }
      },

      fetchCategories: async (tenantId: string) => {
        try {
          const res = await categoryApi.list(tenantId);
          set({ categories: res.data || [] });
        } catch {
          // Backend unavailable — keep previously persisted data
        }
      },

      addProduct: async (data: CreateProductRequest, tenantId: string) => {
        try {
          const product = await productApi.create(data, tenantId);
          set((state) => ({ products: [product, ...state.products] }));
          return product;
        } catch {
          // Backend unavailable — save locally
          const now = new Date().toISOString();
          const product: StoreProduct = {
            id: `prod-${Date.now()}`,
            tenant_id: tenantId,
            name: data.name,
            slug: data.slug || data.name.toLowerCase().replace(/\s+/g, '-'),
            description: data.description || '',
            sku: data.sku,
            price: data.price,
            compare_at_price: data.compare_at_price,
            category_id: data.category_id || '',
            images: data.images || [],
            tags: data.tags || [],
            status: (data.status as StoreProduct['status']) || 'draft',
            created_by: data.created_by,
            created_at: now,
            updated_at: now,
          };
          set((state) => ({ products: [product, ...state.products] }));
          return product;
        }
      },

      deleteProduct: async (id: string, tenantId: string) => {
        try {
          await productApi.delete(id, tenantId);
        } catch {
          // Backend unavailable — delete locally
        }
        set((state) => ({
          products: state.products.filter((p) => p.id !== id),
        }));
      },

      updateProduct: async (id: string, data: Partial<CreateProductRequest> & { updated_by: string }, tenantId: string) => {
        try {
          const updated = await productApi.update(id, data, tenantId);
          set((state) => ({
            products: state.products.map((p) => (p.id === id ? updated : p)),
          }));
        } catch {
          // Backend unavailable — update locally
          set((state) => ({
            products: state.products.map((p) =>
              p.id === id
                ? {
                    ...p,
                    ...data,
                    status: (data.status as StoreProduct['status']) || p.status,
                    updated_at: new Date().toISOString(),
                  }
                : p,
            ),
          }));
        }
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
