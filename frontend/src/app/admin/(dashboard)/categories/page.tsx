'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Plus,
  Pencil,
  Trash2,
  Loader2,
  X,
  FolderTree,
  ChevronRight,
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { cn, formatDate } from '@/lib/utils';
import type {
  Category,
  CreateCategoryRequest,
  UpdateCategoryRequest,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { useProductStore } from '@/stores/products';

type CategoryStatus = 'active' | 'inactive';

const tabs: { label: string; value: CategoryStatus | 'all' }[] = [
  { label: 'All', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Inactive', value: 'inactive' },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

function buildTree(categories: Category[]): (Category & { children: Category[] })[] {
  const map = new Map<string, Category & { children: Category[] }>();
  const roots: (Category & { children: Category[] })[] = [];

  categories.forEach((c) => map.set(c.id, { ...c, children: [] }));

  categories.forEach((c) => {
    const node = map.get(c.id)!;
    if (c.parent_id && map.has(c.parent_id)) {
      map.get(c.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
}

// ---------------------------------------------------------------------------
// Modal
// ---------------------------------------------------------------------------

interface CategoryFormData {
  name: string;
  slug: string;
  description: string;
  parent_id: string;
  status: CategoryStatus;
}

function CategoryModal({
  open,
  category,
  categories,
  saving,
  onClose,
  onSave,
}: {
  open: boolean;
  category: Category | null;
  categories: Category[];
  saving: boolean;
  onClose: () => void;
  onSave: (data: CategoryFormData) => void;
}) {
  const [form, setForm] = useState<CategoryFormData>({
    name: '',
    slug: '',
    description: '',
    parent_id: '',
    status: 'active',
  });
  const [autoSlug, setAutoSlug] = useState(true);

  useEffect(() => {
    if (category) {
      setForm({
        name: category.name,
        slug: category.slug,
        description: category.description || '',
        parent_id: category.parent_id || '',
        status: category.status,
      });
      setAutoSlug(false);
    } else {
      setForm({ name: '', slug: '', description: '', parent_id: '', status: 'active' });
      setAutoSlug(true);
    }
  }, [category, open]);

  function handleNameChange(name: string) {
    setForm((prev) => ({
      ...prev,
      name,
      slug: autoSlug ? slugify(name) : prev.slug,
    }));
  }

  function handleSlugChange(slug: string) {
    setAutoSlug(false);
    setForm((prev) => ({ ...prev, slug }));
  }

  if (!open) return null;

  const parentOptions = categories.filter((c) => c.id !== category?.id);

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
      >
        <motion.div
          className="relative w-full max-w-lg rounded-2xl border border-border bg-surface p-6 shadow-xl"
          initial={{ opacity: 0, scale: 0.95, y: 20 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 20 }}
          transition={{ duration: 0.2 }}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-semibold text-text">
              {category ? 'Edit Category' : 'New Category'}
            </h2>
            <button
              onClick={onClose}
              className="rounded-lg p-1.5 text-text-secondary hover:bg-surface-hover transition-colors"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          <form
            onSubmit={(e) => {
              e.preventDefault();
              onSave(form);
            }}
            className="space-y-4"
          >
            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-text mb-1.5">
                Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                required
                value={form.name}
                onChange={(e) => handleNameChange(e.target.value)}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="e.g. Electronics"
              />
            </div>

            {/* Slug */}
            <div>
              <label className="block text-sm font-medium text-text mb-1.5">Slug</label>
              <input
                type="text"
                value={form.slug}
                onChange={(e) => handleSlugChange(e.target.value)}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text font-mono placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                placeholder="electronics"
              />
            </div>

            {/* Description */}
            <div>
              <label className="block text-sm font-medium text-text mb-1.5">Description</label>
              <textarea
                value={form.description}
                onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))}
                rows={3}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none"
                placeholder="Brief description of this category"
              />
            </div>

            {/* Parent Category */}
            <div>
              <label className="block text-sm font-medium text-text mb-1.5">
                Parent Category
              </label>
              <select
                value={form.parent_id}
                onChange={(e) => setForm((prev) => ({ ...prev, parent_id: e.target.value }))}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              >
                <option value="">None (top-level)</option>
                {parentOptions.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Status (edit only) */}
            {category && (
              <div>
                <label className="block text-sm font-medium text-text mb-1.5">Status</label>
                <select
                  value={form.status}
                  onChange={(e) =>
                    setForm((prev) => ({ ...prev, status: e.target.value as CategoryStatus }))
                  }
                  className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                </select>
              </div>
            )}

            {/* Actions */}
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={saving || !form.name.trim()}
                className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-50"
              >
                {saving && <Loader2 className="h-4 w-4 animate-spin" />}
                {category ? 'Update' : 'Create'}
              </button>
            </div>
          </form>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}

// ---------------------------------------------------------------------------
// Category Row (supports tree indentation)
// ---------------------------------------------------------------------------

function CategoryRow({
  category,
  depth,
  parentName,
  deleting,
  onEdit,
  onDelete,
}: {
  category: Category & { children: Category[] };
  depth: number;
  parentName?: string;
  deleting: string | null;
  onEdit: (c: Category) => void;
  onDelete: (c: Category) => void;
}) {
  return (
    <>
      <tr className="border-b border-border transition-colors last:border-b-0 hover:bg-surface-hover">
        <td className="px-6 py-4">
          <div className="flex items-center gap-3" style={{ paddingLeft: `${depth * 24}px` }}>
            {depth > 0 && (
              <ChevronRight className="h-3.5 w-3.5 text-text-muted" />
            )}
            <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-primary/10 text-sm font-bold text-primary">
              {category.name.charAt(0).toUpperCase()}
            </div>
            <div>
              <span className="text-sm font-medium text-text">{category.name}</span>
              {category.description && (
                <p className="mt-0.5 text-xs text-text-muted line-clamp-1 max-w-xs">
                  {category.description}
                </p>
              )}
            </div>
          </div>
        </td>
        <td className="px-6 py-4 text-sm text-text-secondary font-mono">
          {category.slug}
        </td>
        <td className="px-6 py-4 text-sm text-text-secondary">
          {parentName || '—'}
        </td>
        <td className="px-6 py-4">
          <span
            className={cn(
              'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
              category.status === 'active'
                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400',
            )}
          >
            {category.status}
          </span>
        </td>
        <td className="px-6 py-4 text-sm text-text-muted">
          {formatDate(category.created_at)}
        </td>
        <td className="px-6 py-4">
          <div className="flex items-center gap-2">
            <button
              onClick={() => onEdit(category)}
              className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-surface-hover hover:text-text"
            >
              <Pencil className="h-4 w-4" />
            </button>
            <button
              onClick={() => onDelete(category)}
              disabled={deleting === category.id}
              className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 disabled:opacity-50"
            >
              {deleting === category.id ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="h-4 w-4" />
              )}
            </button>
          </div>
        </td>
      </tr>
      {category.children.map((child) => (
        <CategoryRow
          key={child.id}
          category={child as Category & { children: Category[] }}
          depth={depth + 1}
          parentName={category.name}
          deleting={deleting}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      ))}
    </>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function CategoriesPage() {
  const tenantId = useAuthStore((s) => s.tenantId);
  const user = useAuthStore((s) => s.user);
  const categories = useProductStore((s) => s.categories);
  const fetchCategories = useProductStore((s) => s.fetchCategories);
  const storeAddCategory = useProductStore((s) => s.addCategory);
  const storeUpdateCategory = useProductStore((s) => s.updateCategory);
  const storeUpdateCategoryStatus = useProductStore((s) => s.updateCategoryStatus);
  const storeDeleteCategory = useProductStore((s) => s.deleteCategory);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<CategoryStatus | 'all'>('all');

  // Modal state
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);

  const loadCategories = useCallback(async () => {
    if (!tenantId) return;
    setLoading(true);
    setError(null);
    await fetchCategories(tenantId);
    setLoading(false);
  }, [tenantId, fetchCategories]);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  const filtered =
    activeTab === 'all' ? categories : categories.filter((c) => c.status === activeTab);

  const tree = buildTree(filtered);

  // Flatten tree for the "flat" rows but with depth info
  const categoryMap = new Map(categories.map((c) => [c.id, c]));

  function getParentName(parentId: string | null): string | undefined {
    if (!parentId) return undefined;
    return categoryMap.get(parentId)?.name;
  }

  function openCreate() {
    setEditingCategory(null);
    setModalOpen(true);
  }

  function openEdit(cat: Category) {
    setEditingCategory(cat);
    setModalOpen(true);
  }

  async function handleSave(data: CategoryFormData) {
    if (!tenantId || !user) return;
    setSaving(true);
    if (editingCategory) {
      const payload: UpdateCategoryRequest = {
        name: data.name,
        slug: data.slug,
        description: data.description,
        parent_id: data.parent_id || null,
        updated_by: user.id,
      };
      await storeUpdateCategory(editingCategory.id, payload, tenantId);
      if (data.status !== editingCategory.status) {
        await storeUpdateCategoryStatus(editingCategory.id, data.status, tenantId);
      }
    } else {
      const payload: CreateCategoryRequest = {
        tenant_id: tenantId,
        name: data.name,
        slug: data.slug || slugify(data.name),
        description: data.description,
        parent_id: data.parent_id || null,
        created_by: user.id,
      };
      await storeAddCategory(payload, tenantId);
    }
    setModalOpen(false);
    setSaving(false);
  }

  async function handleDelete(cat: Category) {
    if (!tenantId) return;
    const hasChildren = categories.some((c) => c.parent_id === cat.id);
    const msg = hasChildren
      ? `Delete "${cat.name}" and its subcategories? This cannot be undone.`
      : `Delete "${cat.name}"? This cannot be undone.`;
    if (!confirm(msg)) return;

    setDeleting(cat.id);
    await storeDeleteCategory(cat.id, tenantId);
    setDeleting(null);
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <motion.div
        className="flex items-center justify-between"
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <div>
          <h1 className="text-2xl font-bold text-text">Categories</h1>
          <p className="mt-1 text-sm text-text-secondary">
            {categories.length} total categories
          </p>
        </div>
        <button
          onClick={openCreate}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
        >
          <Plus className="h-4 w-4" />
          Add Category
        </button>
      </motion.div>

      {error && (
        <div className="rounded-lg bg-red-50 dark:bg-red-900/20 px-4 py-3 text-sm text-red-700 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-border">
        <nav className="-mb-px flex gap-6">
          {tabs.map((tab) => {
            const count =
              tab.value === 'all'
                ? categories.length
                : categories.filter((c) => c.status === tab.value).length;
            return (
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
                {tab.label} ({count})
              </button>
            );
          })}
        </nav>
      </div>

      {/* Table */}
      <motion.div
        className="rounded-2xl border border-border bg-surface shadow-sm"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1 }}
      >
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border text-left text-sm text-text-secondary">
                <th className="px-6 py-3 font-medium">Category</th>
                <th className="px-6 py-3 font-medium">Slug</th>
                <th className="px-6 py-3 font-medium">Parent</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Created</th>
                <th className="px-6 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-6 py-16 text-center">
                    <Loader2 className="mx-auto h-6 w-6 animate-spin text-primary" />
                    <p className="mt-2 text-sm text-text-secondary">Loading categories...</p>
                  </td>
                </tr>
              ) : tree.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-16 text-center">
                    <FolderTree className="mx-auto h-10 w-10 text-text-muted" />
                    <p className="mt-3 text-sm font-medium text-text">No categories found</p>
                    <p className="mt-1 text-sm text-text-muted">
                      Create your first category to organize products.
                    </p>
                    <button
                      onClick={openCreate}
                      className="mt-4 inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
                    >
                      <Plus className="h-4 w-4" />
                      Add Category
                    </button>
                  </td>
                </tr>
              ) : (
                tree.map((root) => (
                  <CategoryRow
                    key={root.id}
                    category={root}
                    depth={0}
                    parentName={getParentName(root.parent_id)}
                    deleting={deleting}
                    onEdit={openEdit}
                    onDelete={handleDelete}
                  />
                ))
              )}
            </tbody>
          </table>
        </div>
      </motion.div>

      {/* Modal */}
      <CategoryModal
        open={modalOpen}
        category={editingCategory}
        categories={categories}
        saving={saving}
        onClose={() => setModalOpen(false)}
        onSave={handleSave}
      />
    </div>
  );
}
