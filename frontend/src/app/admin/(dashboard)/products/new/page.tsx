'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Plus, X, Save, Link2 } from 'lucide-react';
import { useProductStore } from '@/stores/products';
import { useAuthStore } from '@/stores/auth';
import { useDeliveryProfileStore } from '@/stores/delivery-profiles';
import { uploadImages } from '@/lib/api';
import { mediaUrl } from '@/lib/utils';
import FileUpload, { type UploadedFile } from '@/components/ui/file-upload';

interface Variant {
  id: string;
  name: string;
  value: string;
  sku: string;
  price: string;
  stock: string;
}


export default function NewProductPage() {
  const router = useRouter();
  const addProduct = useProductStore((s) => s.addProduct);
  const categories = useProductStore((s) => s.categories);
  const fetchCategories = useProductStore((s) => s.fetchCategories);
  const tenantId = useAuthStore((s) => s.tenantId);
  const token = useAuthStore((s) => s.token);
  const user = useAuthStore((s) => s.user);
  const deliveryProfiles = useDeliveryProfileStore((s) => s.profiles);
  const getDefaultProfile = useDeliveryProfileStore((s) => s.getDefaultProfile);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [formError, setFormError] = useState('');

  useEffect(() => {
    if (tenantId) fetchCategories(tenantId);
  }, [tenantId, fetchCategories]);

  // Basic info
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [sku, setSku] = useState('');
  const [category, setCategory] = useState('');
  const [status, setStatus] = useState<'active' | 'draft'>('draft');
  const [tags, setTags] = useState('');

  // Pricing & Stock
  const [price, setPrice] = useState('');
  const [compareAtPrice, setCompareAtPrice] = useState('');
  const [stock, setStock] = useState('');

  // Variants
  const [variants, setVariants] = useState<Variant[]>([]);

  // Delivery
  const [deliveryProfileId, setDeliveryProfileId] = useState('');

  // Images
  const [imageFiles, setImageFiles] = useState<UploadedFile[]>([]);
  const [imageUrlInput, setImageUrlInput] = useState('');

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

  async function handleFilesAdded(files: File[]) {
    // Add placeholders with progress
    const placeholders = files.map((file) => ({
      id: `img-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      name: file.name,
      size: file.size,
      type: file.type,
      url: '',
      path: '',
      progress: 30,
    }));
    setImageFiles((prev) => [...prev, ...placeholders]);

    try {
      const paths = await uploadImages(files);
      setImageFiles((prev) =>
        prev.map((f) => {
          const idx = placeholders.findIndex((p) => p.id === f.id);
          if (idx >= 0 && paths[idx]) {
            return { ...f, url: mediaUrl(paths[idx]), path: paths[idx], progress: 100 };
          }
          return f;
        }),
      );
    } catch {
      const ids = new Set(placeholders.map((p) => p.id));
      setImageFiles((prev) => prev.filter((f) => !ids.has(f.id) || f.url));
      setFormError('Image upload failed. Please try again.');
    }
  }

  function handleFileRemoved(id: string) {
    setImageFiles((prev) => prev.filter((f) => f.id !== id));
  }

  function addImageByUrl() {
    const url = imageUrlInput.trim();
    if (!url) return;
    setImageFiles((prev) => [
      ...prev,
      { id: `url-${Date.now()}`, name: url.split('/').pop() || 'image', size: 0, type: 'image/*', url },
    ]);
    setImageUrlInput('');
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
      await addProduct(
        {
          tenant_id: tenantId,
          name,
          slug: slug || generateSlug(name),
          description,
          sku,
          price: Number(price) || 0,
          compare_at_price: compareAtPrice ? Number(compareAtPrice) : undefined,
          category_id: category,
          delivery_profile_id: deliveryProfileId || undefined,
          status,
          tags: parsedTags,
          images: imageFiles.filter((f) => f.path || f.url).map((f) => f.path || f.url!),
          variants: variants.map((v) => ({
            name: v.name,
            value: v.value,
            sku: v.sku,
            price: Number(v.price) || 0,
            stock: Number(v.stock) || 0,
          })),
          created_by: user?.username || 'admin',
        },
        tenantId,
        token || undefined,
      );

      setSaving(false);
      setSaved(true);
      setTimeout(() => {
        router.push('/admin/products');
      }, 1500);
    } catch (err) {
      setSaving(false);
      setFormError((err as Error).message || 'Failed to save product. Make sure the backend is running.');
    }
  }

  if (saved) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center">
        <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
          <Save className="h-8 w-8 text-green-600" />
        </div>
        <h2 className="text-xl font-semibold text-gray-900">Product Created!</h2>
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
          <h1 className="text-2xl font-bold text-gray-900">New Product</h1>
          <p className="text-sm text-gray-500">Add a new product to your store</p>
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
                  SKU <span className="text-red-500">*</span>
                </label>
                <input
                  id="sku"
                  type="text"
                  required
                  value={sku}
                  onChange={(e) => setSku(e.target.value.toUpperCase())}
                  placeholder="SAR-JAM-001"
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm font-mono focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
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
                  onChange={(e) => setStatus(e.target.value as 'active' | 'draft')}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  <option value="draft">Draft</option>
                  <option value="active">Active</option>
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
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div>
              <label htmlFor="price" className="mb-1.5 block text-sm font-medium text-gray-700">
                Price (BDT) <span className="text-red-500">*</span>
              </label>
              <div className="relative">
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm text-gray-400">৳</span>
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
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm text-gray-400">৳</span>
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
            <div>
              <label htmlFor="stock" className="mb-1.5 block text-sm font-medium text-gray-700">
                Stock Quantity
              </label>
              <input
                id="stock"
                type="number"
                min="0"
                value={stock}
                onChange={(e) => setStock(e.target.value)}
                placeholder="0"
                className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>
        </div>

        {/* Delivery Profile */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Delivery</h2>
          <div>
            <label htmlFor="deliveryProfile" className="mb-1.5 block text-sm font-medium text-gray-700">
              Delivery Charge Profile
            </label>
            <select
              id="deliveryProfile"
              value={deliveryProfileId}
              onChange={(e) => setDeliveryProfileId(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="">Default ({getDefaultProfile().name})</option>
              {deliveryProfiles.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} — Dhaka: ৳{p.inside_dhaka_rate} / Outside: ৳{p.outside_dhaka_rate}
                </option>
              ))}
            </select>
            <p className="mt-1 text-xs text-gray-500">Select a delivery charge profile for this product</p>
          </div>
        </div>

        {/* Images */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Images</h2>
          <FileUpload
            files={imageFiles}
            onFilesAdded={handleFilesAdded}
            onFileRemoved={handleFileRemoved}
            accept="image/*"
            multiple
            maxSize={5 * 1024 * 1024}
            maxFiles={10}
          />
          <div className="mt-4">
            <label className="mb-1.5 block text-xs font-medium text-gray-500">Or add by URL</label>
            <div className="flex gap-2">
              <input
                type="url"
                value={imageUrlInput}
                onChange={(e) => setImageUrlInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addImageByUrl(); } }}
                placeholder="https://cdn.example.com/image.jpg"
                className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <button
                type="button"
                onClick={addImageByUrl}
                className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
              >
                <Link2 className="h-4 w-4" />
                Add
              </button>
            </div>
          </div>
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
              <p className="text-sm text-gray-500">No variants — this product has a single option</p>
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
              {saving ? 'Saving...' : 'Save Product'}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
