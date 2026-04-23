'use client';

import { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Plus, X, Upload, Save, Loader2 } from 'lucide-react';
import { useProductStore } from '@/stores/products';
import { useAuthStore } from '@/stores/auth';
import { productApi } from '@/lib/api';

interface Variant {
  id: string;
  name: string;
  value: string;
  sku: string;
  price: string;
  stock: string;
}

interface ImageEntry {
  id: string;
  url: string;
  altText: string;
}

export default function EditProductPage() {
  const router = useRouter();
  const params = useParams();
  const productId = params.id as string;
  const updateProduct = useProductStore((s) => s.updateProduct);
  const categories = useProductStore((s) => s.categories);
  const fetchCategories = useProductStore((s) => s.fetchCategories);
  const tenantId = useAuthStore((s) => s.tenantId);
  const user = useAuthStore((s) => s.user);

  const [pageLoading, setPageLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [formError, setFormError] = useState('');

  // Basic info
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [sku, setSku] = useState('');
  const [category, setCategory] = useState('');
  const [status, setStatus] = useState<'active' | 'draft' | 'archived'>('draft');
  const [tags, setTags] = useState('');

  // Pricing
  const [price, setPrice] = useState('');
  const [compareAtPrice, setCompareAtPrice] = useState('');

  // Variants
  const [variants, setVariants] = useState<Variant[]>([]);

  // Images
  const [images, setImages] = useState<ImageEntry[]>([]);

  // Load product data
  useEffect(() => {
    if (!tenantId || !productId) return;

    async function load() {
      try {
        const [product] = await Promise.all([
          productApi.get(productId, tenantId!),
          fetchCategories(tenantId!),
        ]);

        setName(product.name);
        setSlug(product.slug || '');
        setDescription(product.description || '');
        setSku(product.sku);
        setCategory(product.category_id);
        setStatus(product.status as 'active' | 'draft' | 'archived');
        setTags(product.tags?.join(', ') || '');
        setPrice(String(product.price));
        setCompareAtPrice(product.compare_at_price ? String(product.compare_at_price) : '');

        if (product.images && product.images.length > 0) {
          setImages(
            product.images.map((url, i) => ({
              id: `img-${i}`,
              url,
              altText: '',
            })),
          );
        }

        if (product.variants && product.variants.length > 0) {
          setVariants(
            product.variants.map((v, i) => ({
              id: v.id || `v-${i}`,
              name: v.name,
              value: v.value || '',
              sku: v.sku,
              price: String(v.price),
              stock: String(v.stock || 0),
            })),
          );
        }
      } catch (err) {
        setFormError((err as Error).message || 'Failed to load product');
      } finally {
        setPageLoading(false);
      }
    }

    load();
  }, [tenantId, productId, fetchCategories]);

  function generateSlug(value: string) {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, '')
      .replace(/\s+/g, '-')
      .replace(/-+/g, '-')
      .trim();
  }

  function handleNameChange(value: string) {
    setName(value);
    if (!slug || slug === generateSlug(name)) {
      setSlug(generateSlug(value));
    }
  }

  function addVariant() {
    setVariants([
      ...variants,
      { id: `v-${Date.now()}`, name: '', value: '', sku: '', price: '', stock: '' },
    ]);
  }

  function updateVariant(id: string, field: keyof Variant, value: string) {
    setVariants(variants.map((v) => (v.id === id ? { ...v, [field]: value } : v)));
  }

  function removeVariant(id: string) {
    setVariants(variants.filter((v) => v.id !== id));
  }

  function addImage() {
    setImages([...images, { id: `img-${Date.now()}`, url: '', altText: '' }]);
  }

  function updateImage(id: string, field: 'url' | 'altText', value: string) {
    setImages(images.map((img) => (img.id === id ? { ...img, [field]: value } : img)));
  }

  function removeImage(id: string) {
    setImages(images.filter((img) => img.id !== id));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!tenantId) return;
    setSaving(true);
    setFormError('');

    const parsedTags = tags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean);

    try {
      await updateProduct(
        productId,
        {
          name,
          description,
          price: Number(price) || 0,
          compare_at_price: compareAtPrice ? Number(compareAtPrice) : undefined,
          category_id: category,
          status,
          tags: parsedTags,
          images: images.filter((img) => img.url).map((img) => img.url),
          variants: variants.map((v) => ({
            name: v.name,
            value: v.value,
            sku: v.sku,
            price: Number(v.price) || 0,
            stock: Number(v.stock) || 0,
          })),
          updated_by: user?.username || 'admin',
        },
        tenantId,
      );

      setSaving(false);
      setSaved(true);
      setTimeout(() => {
        router.push('/admin/products');
      }, 1500);
    } catch (err) {
      setSaving(false);
      setFormError((err as Error).message || 'Failed to update product');
    }
  }

  if (pageLoading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (saved) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center">
        <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
          <Save className="h-8 w-8 text-green-600" />
        </div>
        <h2 className="text-xl font-semibold text-gray-900">Product Updated!</h2>
        <p className="mt-1 text-sm text-gray-500">Redirecting to products list...</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link
          href="/admin/products"
          className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Edit Product</h1>
          <p className="text-sm text-gray-500">Update product details</p>
        </div>
      </div>

      {formError && (
        <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{formError}</div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Information */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Basic Information</h2>
          <div className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label htmlFor="name" className="mb-1.5 block text-sm font-medium text-gray-700">
                  Product Name <span className="text-red-500">*</span>
                </label>
                <input
                  id="name"
                  type="text"
                  required
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="e.g. Premium Jamdani Saree"
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div>
                <label htmlFor="slug" className="mb-1.5 block text-sm font-medium text-gray-700">
                  Slug
                </label>
                <input
                  id="slug"
                  type="text"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  placeholder="premium-jamdani-saree"
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm font-mono text-gray-600 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>

            <div>
              <label htmlFor="description" className="mb-1.5 block text-sm font-medium text-gray-700">
                Description
              </label>
              <textarea
                id="description"
                rows={4}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe your product..."
                className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div>
                <label htmlFor="sku" className="mb-1.5 block text-sm font-medium text-gray-700">
                  SKU
                </label>
                <input
                  id="sku"
                  type="text"
                  disabled
                  value={sku}
                  className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3.5 py-2.5 text-sm font-mono text-gray-500"
                />
              </div>
              <div>
                <label htmlFor="category" className="mb-1.5 block text-sm font-medium text-gray-700">
                  Category <span className="text-red-500">*</span>
                </label>
                <select
                  id="category"
                  required
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  <option value="">Select category</option>
                  {categories.map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                  {categories.length === 0 && (
                    <option disabled>Loading categories...</option>
                  )}
                </select>
              </div>
              <div>
                <label htmlFor="status" className="mb-1.5 block text-sm font-medium text-gray-700">
                  Status
                </label>
                <select
                  id="status"
                  value={status}
                  onChange={(e) => setStatus(e.target.value as 'active' | 'draft' | 'archived')}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  <option value="draft">Draft</option>
                  <option value="active">Active</option>
                  <option value="archived">Archived</option>
                </select>
              </div>
            </div>

            <div>
              <label htmlFor="tags" className="mb-1.5 block text-sm font-medium text-gray-700">
                Tags
              </label>
              <input
                id="tags"
                type="text"
                value={tags}
                onChange={(e) => setTags(e.target.value)}
                placeholder="jamdani, saree, handwoven (comma separated)"
                className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              {tags && (
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {tags.split(',').map((tag, i) => {
                    const trimmed = tag.trim();
                    if (!trimmed) return null;
                    return (
                      <span
                        key={i}
                        className="inline-flex items-center rounded-full bg-primary-light px-2.5 py-0.5 text-xs font-medium text-primary"
                      >
                        {trimmed}
                      </span>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Pricing */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Pricing</h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label htmlFor="price" className="mb-1.5 block text-sm font-medium text-gray-700">
                Price (BDT) <span className="text-red-500">*</span>
              </label>
              <div className="relative">
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm text-gray-400">{'\u09F3'}</span>
                <input
                  id="price"
                  type="number"
                  required
                  min="0"
                  step="0.01"
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                  placeholder="0.00"
                  className="w-full rounded-lg border border-gray-300 py-2.5 pl-8 pr-3.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>
            <div>
              <label htmlFor="comparePrice" className="mb-1.5 block text-sm font-medium text-gray-700">
                Compare at Price (BDT)
              </label>
              <div className="relative">
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm text-gray-400">{'\u09F3'}</span>
                <input
                  id="comparePrice"
                  type="number"
                  min="0"
                  step="0.01"
                  value={compareAtPrice}
                  onChange={(e) => setCompareAtPrice(e.target.value)}
                  placeholder="0.00"
                  className="w-full rounded-lg border border-gray-300 py-2.5 pl-8 pr-3.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              {price && compareAtPrice && Number(compareAtPrice) > Number(price) && (
                <p className="mt-1 text-xs text-green-600">
                  {Math.round(((Number(compareAtPrice) - Number(price)) / Number(compareAtPrice)) * 100)}% discount
                </p>
              )}
            </div>
          </div>
        </div>

        {/* Images */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Images</h2>
            <button
              type="button"
              onClick={addImage}
              className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
            >
              <Upload className="h-4 w-4" />
              Add Image URL
            </button>
          </div>
          {images.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-12 text-center">
              <Upload className="mb-2 h-8 w-8 text-gray-300" />
              <p className="text-sm text-gray-500">No images added yet</p>
              <button
                type="button"
                onClick={addImage}
                className="mt-2 text-sm font-medium text-primary hover:text-primary-dark"
              >
                Add image URL
              </button>
            </div>
          ) : (
            <div className="space-y-3">
              {images.map((img, index) => (
                <div key={img.id} className="flex items-start gap-3 rounded-lg border border-gray-200 p-3">
                  <div className="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 text-xs text-gray-400 overflow-hidden">
                    {img.url ? (
                      <img src={img.url} alt={img.altText || `Image ${index + 1}`} className="h-full w-full object-cover"
                        onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; (e.target as HTMLImageElement).parentElement!.innerHTML = `<span class="text-center text-[10px] break-all px-1 text-gray-400">${img.url.slice(-15)}</span>`; }} />
                    ) : (
                      `#${index + 1}`
                    )}
                  </div>
                  <div className="flex-1 space-y-2">
                    <input
                      type="url"
                      value={img.url}
                      onChange={(e) => updateImage(img.id, 'url', e.target.value)}
                      placeholder="https://cdn.example.com/image.jpg"
                      className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                    <input
                      type="text"
                      value={img.altText}
                      onChange={(e) => updateImage(img.id, 'altText', e.target.value)}
                      placeholder="Alt text"
                      className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <button
                    type="button"
                    onClick={() => removeImage(img.id)}
                    className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Variants */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Variants</h2>
              <p className="text-sm text-gray-500">Add size, color, or other product options</p>
            </div>
            <button
              type="button"
              onClick={addVariant}
              className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
            >
              <Plus className="h-4 w-4" />
              Add Variant
            </button>
          </div>
          {variants.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-8 text-center">
              <p className="text-sm text-gray-500">No variants -- this product has a single option</p>
              <button
                type="button"
                onClick={addVariant}
                className="mt-2 text-sm font-medium text-primary hover:text-primary-dark"
              >
                Add variant
              </button>
            </div>
          ) : (
            <div className="space-y-3">
              {variants.map((variant) => (
                <div key={variant.id} className="rounded-lg border border-gray-200 p-4">
                  <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Option</label>
                      <input
                        type="text"
                        value={variant.name}
                        onChange={(e) => updateVariant(variant.id, 'name', e.target.value)}
                        placeholder="Color"
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Value</label>
                      <input
                        type="text"
                        value={variant.value}
                        onChange={(e) => updateVariant(variant.id, 'value', e.target.value)}
                        placeholder="Red"
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">SKU</label>
                      <input
                        type="text"
                        value={variant.sku}
                        onChange={(e) => updateVariant(variant.id, 'sku', e.target.value.toUpperCase())}
                        placeholder="SAR-JAM-RED"
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-mono focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Price (BDT)</label>
                      <input
                        type="number"
                        min="0"
                        step="0.01"
                        value={variant.price}
                        onChange={(e) => updateVariant(variant.id, 'price', e.target.value)}
                        placeholder="0.00"
                        className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                    </div>
                    <div className="flex items-end gap-2">
                      <div className="flex-1">
                        <label className="mb-1 block text-xs font-medium text-gray-500">Stock</label>
                        <input
                          type="number"
                          min="0"
                          value={variant.stock}
                          onChange={(e) => updateVariant(variant.id, 'stock', e.target.value)}
                          placeholder="0"
                          className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => removeVariant(variant.id)}
                        className="mb-0.5 rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <Link
            href="/admin/products"
            className="rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
          >
            Cancel
          </Link>
          <div className="flex items-center gap-3">
            <button
              type="submit"
              disabled={saving}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Save className="h-4 w-4" />
              {saving ? 'Saving...' : 'Update Product'}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
