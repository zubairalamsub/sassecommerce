'use client';

import { create } from 'zustand';
import { productApi, categoryApi, type Product, type CreateProductRequest, type Category } from '@/lib/api';

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
}

export const useProductStore = create<ProductStore>()((set, get) => ({
  products: [],
  categories: [],
  loading: false,
  error: null,

  fetchProducts: async (tenantId: string) => {
    set({ loading: true, error: null });
    try {
      const res = await productApi.list(tenantId, 1, 100);
      set({ products: res.data || [], loading: false });
    } catch (err) {
      set({ error: (err as Error).message, loading: false });
    }
  },

  fetchCategories: async (tenantId: string) => {
    try {
      const res = await categoryApi.list(tenantId);
      set({ categories: res.data || [] });
    } catch {
      // categories are optional, don't block
    }
  },

  addProduct: async (data: CreateProductRequest, tenantId: string) => {
    const product = await productApi.create(data, tenantId);
    set((state) => ({ products: [product, ...state.products] }));
    return product;
  },

  deleteProduct: async (id: string, tenantId: string) => {
    await productApi.delete(id, tenantId);
    set((state) => ({
      products: state.products.filter((p) => p.id !== id),
    }));
  },

  updateProduct: async (id: string, data: Partial<CreateProductRequest> & { updated_by: string }, tenantId: string) => {
    const updated = await productApi.update(id, data, tenantId);
    set((state) => ({
      products: state.products.map((p) => (p.id === id ? updated : p)),
    }));
  },
}));
