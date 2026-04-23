'use client';

import { useState, useEffect } from 'react';
import { Plus, Pencil, Trash2, Loader2 } from 'lucide-react';
import Link from 'next/link';
import { cn, formatCurrency, statusColor } from '@/lib/utils';
import { useProductStore } from '@/stores/products';
import { useAuthStore } from '@/stores/auth';

type ProductStatus = 'active' | 'draft' | 'archived';

const tabs: { label: string; value: ProductStatus | 'all' }[] = [
  { label: 'All', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Draft', value: 'draft' },
  { label: 'Archived', value: 'archived' },
];

export default function ProductsPage() {
  const { products, loading, error, fetchProducts, deleteProduct } = useProductStore();
  const tenantId = useAuthStore((s) => s.tenantId);
  const [activeTab, setActiveTab] = useState<ProductStatus | 'all'>('all');
  const [deleting, setDeleting] = useState<string | null>(null);

  useEffect(() => {
    if (tenantId) {
      fetchProducts(tenantId);
    }
  }, [tenantId, fetchProducts]);

  const filtered =
    activeTab === 'all' ? products : products.filter((p) => p.status === activeTab);

  async function handleDelete(id: string, name: string) {
    if (!tenantId) return;
    if (confirm(`Delete "${name}"? This cannot be undone.`)) {
      setDeleting(id);
      try {
        await deleteProduct(id, tenantId);
      } catch {
        alert('Failed to delete product');
      }
      setDeleting(null);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Products</h1>
          <p className="text-sm text-gray-500">{products.length} total products</p>
        </div>
        <Link
          href="/admin/products/new"
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
        >
          <Plus className="h-4 w-4" />
          Add Product
        </Link>
      </div>

      {error && (
        <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6">
          {tabs.map((tab) => {
            const count =
              tab.value === 'all'
                ? products.length
                : products.filter((p) => p.status === tab.value).length;
            return (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                className={cn(
                  'border-b-2 pb-3 text-sm font-medium transition-colors',
                  activeTab === tab.value
                    ? 'border-primary text-primary'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700',
                )}
              >
                {tab.label} ({count})
              </button>
            );
          })}
        </nav>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                <th className="px-6 py-3 font-medium">Product</th>
                <th className="px-6 py-3 font-medium">SKU</th>
                <th className="px-6 py-3 font-medium">Price</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-primary" />
                    <p className="mt-2 text-sm text-gray-500">Loading products...</p>
                  </td>
                </tr>
              ) : filtered.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12 text-center text-sm text-gray-500">
                    No products found
                  </td>
                </tr>
              ) : (
                filtered.map((product) => (
                  <tr
                    key={product.id}
                    className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                  >
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-primary-light text-sm font-bold text-primary">
                          {product.name.charAt(0)}
                        </div>
                        <div>
                          <span className="text-sm font-medium text-gray-900">
                            {product.name}
                          </span>
                          {product.tags && product.tags.length > 0 && (
                            <div className="flex gap-1 mt-0.5">
                              {product.tags.slice(0, 2).map((tag) => (
                                <span key={tag} className="text-[10px] text-gray-400">#{tag}</span>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500 font-mono">
                      {product.sku}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      {formatCurrency(product.price)}
                      {product.compare_at_price ? (
                        <span className="ml-1.5 text-xs text-gray-400 line-through">
                          {formatCurrency(product.compare_at_price)}
                        </span>
                      ) : null}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                          statusColor(product.status),
                        )}
                      >
                        {product.status}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <Link
                          href={`/admin/products/${product.id}/edit`}
                          className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
                        >
                          <Pencil className="h-4 w-4" />
                        </Link>
                        <button
                          onClick={() => handleDelete(product.id, product.name)}
                          disabled={deleting === product.id}
                          className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 disabled:opacity-50"
                        >
                          {deleting === product.id ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <Trash2 className="h-4 w-4" />
                          )}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
