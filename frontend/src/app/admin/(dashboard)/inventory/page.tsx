'use client';

import { useState, useEffect } from 'react';
import { Package, AlertTriangle, Warehouse as WarehouseIcon, Loader2, Search, Plus, Minus, X, MapPin } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  inventoryApi, productApi,
  type InventoryItem, type Warehouse as WarehouseType,
  type CreateWarehouseRequest, type CreateInventoryItemRequest,
} from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

type StockStatus = 'in_stock' | 'low_stock' | 'out_of_stock';

function getStatus(item: InventoryItem): StockStatus {
  if (item.quantityOnHand <= 0) return 'out_of_stock';
  if (item.quantityOnHand <= item.reorderPoint) return 'low_stock';
  return 'in_stock';
}

function stockStatusLabel(status: StockStatus) {
  switch (status) {
    case 'in_stock':
      return { label: 'In Stock', classes: 'bg-green-100 text-green-800' };
    case 'low_stock':
      return { label: 'Low Stock', classes: 'bg-yellow-100 text-yellow-800' };
    case 'out_of_stock':
      return { label: 'Out of Stock', classes: 'bg-red-100 text-red-800' };
  }
}

interface SimpleProduct { id: string; name: string; sku: string; }

export default function InventoryPage() {
  const { tenantId, token, user } = useAuthStore();
  const [items, setItems] = useState<InventoryItem[]>([]);
  const [warehouses, setWarehouses] = useState<WarehouseType[]>([]);
  const [products, setProducts] = useState<SimpleProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  // Adjust stock dialog
  const [adjustItem, setAdjustItem] = useState<InventoryItem | null>(null);
  const [adjustQty, setAdjustQty] = useState(0);
  const [adjustReason, setAdjustReason] = useState('');
  const [adjustLoading, setAdjustLoading] = useState(false);

  // Add warehouse dialog
  const [showAddWarehouse, setShowAddWarehouse] = useState(false);
  const [whForm, setWhForm] = useState({ code: '', name: '', address: '', city: '', state: '', country: 'BD', postalCode: '', phone: '', email: '' });
  const [whLoading, setWhLoading] = useState(false);
  const [whError, setWhError] = useState('');

  // Add inventory item dialog
  const [showAddItem, setShowAddItem] = useState(false);
  const [itemForm, setItemForm] = useState({ productId: '', sku: '', warehouseId: '', initialQuantity: 0, reorderPoint: 10, reorderQuantity: 25 });
  const [itemLoading, setItemLoading] = useState(false);
  const [itemError, setItemError] = useState('');

  useEffect(() => {
    loadInventory();
  }, [tenantId, token]);

  async function loadInventory() {
    if (!tenantId || !token) return;
    setLoading(true);
    try {
      const [itemsRes, warehousesRes, productsRes] = await Promise.allSettled([
        inventoryApi.listItems(tenantId, token, 0, 200),
        inventoryApi.listWarehouses(tenantId, token),
        productApi.list(tenantId, 1, 200),
      ]);
      if (itemsRes.status === 'fulfilled') setItems(itemsRes.value.data ?? []);
      if (warehousesRes.status === 'fulfilled') setWarehouses(warehousesRes.value.data ?? []);
      if (productsRes.status === 'fulfilled') {
        const pList = productsRes.value.data ?? [];
        setProducts(pList.map((p) => ({ id: p.id, name: p.name, sku: p.sku })));
      }
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleAdjust(type: 'add' | 'remove') {
    if (!adjustItem || !tenantId || !token || adjustQty <= 0) return;
    setAdjustLoading(true);
    try {
      const updated = await inventoryApi.adjustStock(adjustItem.id, {
        quantity: type === 'add' ? adjustQty : -adjustQty,
        reason: adjustReason || (type === 'add' ? 'Stock received' : 'Stock adjustment'),
        type,
      }, tenantId, token);
      setItems((prev) => prev.map((i) => i.id === updated.id ? updated : i));
      setAdjustItem(null);
      setAdjustQty(0);
      setAdjustReason('');
    } catch {
      // ignore
    } finally {
      setAdjustLoading(false);
    }
  }

  async function handleAddWarehouse() {
    if (!tenantId || !token || !user) return;
    if (!whForm.code || !whForm.name || !whForm.address || !whForm.city || !whForm.state || !whForm.country || !whForm.postalCode) {
      setWhError('Please fill all required fields.');
      return;
    }
    setWhLoading(true);
    setWhError('');
    try {
      const created = await inventoryApi.createWarehouse({
        tenantId,
        code: whForm.code,
        name: whForm.name,
        address: whForm.address,
        city: whForm.city,
        state: whForm.state,
        country: whForm.country,
        postalCode: whForm.postalCode,
        phone: whForm.phone || undefined,
        email: whForm.email || undefined,
        isActive: true,
        createdBy: user.id,
      }, token);
      setWarehouses((prev) => [...prev, created]);
      setShowAddWarehouse(false);
      setWhForm({ code: '', name: '', address: '', city: '', state: '', country: 'BD', postalCode: '', phone: '', email: '' });
    } catch (err: unknown) {
      setWhError(err instanceof Error ? err.message : 'Failed to create warehouse');
    } finally {
      setWhLoading(false);
    }
  }

  async function handleAddItem() {
    if (!tenantId || !token || !user) return;
    if (!itemForm.productId || !itemForm.sku || !itemForm.warehouseId) {
      setItemError('Please select a product, enter SKU, and select a warehouse.');
      return;
    }
    setItemLoading(true);
    setItemError('');
    try {
      const created = await inventoryApi.createItem({
        tenantId,
        warehouseId: itemForm.warehouseId,
        productId: itemForm.productId,
        sku: itemForm.sku,
        initialQuantity: itemForm.initialQuantity,
        reorderPoint: itemForm.reorderPoint,
        reorderQuantity: itemForm.reorderQuantity,
        createdBy: user.id,
      }, token);
      setItems((prev) => [...prev, created]);
      setShowAddItem(false);
      setItemForm({ productId: '', sku: '', warehouseId: '', initialQuantity: 0, reorderPoint: 10, reorderQuantity: 25 });
    } catch (err: unknown) {
      setItemError(err instanceof Error ? err.message : 'Failed to create inventory item');
    } finally {
      setItemLoading(false);
    }
  }

  const warehouseMap = new Map(warehouses.map((w) => [w.id, w.name]));

  const filtered = items.filter((item) => {
    const status = getStatus(item);
    if (statusFilter && status !== statusFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return item.sku.toLowerCase().includes(q) || item.productId.toLowerCase().includes(q);
    }
    return true;
  });

  const totalItems = items.length;
  const lowStockCount = items.filter((i) => getStatus(i) === 'low_stock').length;
  const outOfStockCount = items.filter((i) => getStatus(i) === 'out_of_stock').length;
  const warehouseCount = warehouses.length;

  const statCards = [
    { title: 'Total SKUs', value: totalItems, icon: Package, color: 'text-gray-900' },
    { title: 'Low Stock', value: lowStockCount, icon: AlertTriangle, color: 'text-yellow-600' },
    { title: 'Out of Stock', value: outOfStockCount, icon: Package, color: 'text-red-600' },
    { title: 'Warehouses', value: warehouseCount, icon: WarehouseIcon, color: 'text-primary' },
  ];

  // Auto-fill SKU when product is selected in add-item form
  function onProductSelect(productId: string) {
    const product = products.find((p) => p.id === productId);
    setItemForm((f) => ({ ...f, productId, sku: product?.sku || f.sku }));
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Inventory</h1>
          <p className="mt-1 text-sm text-gray-500">Track stock levels across all warehouses.</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setShowAddWarehouse(true)}
            className="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
          >
            <MapPin className="h-4 w-4" /> Add Warehouse
          </button>
          <button
            onClick={() => { if (warehouses.length === 0) { alert('Please create a warehouse first.'); return; } setShowAddItem(true); }}
            className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
          >
            <Plus className="h-4 w-4" /> Add Item
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {statCards.map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.title} className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
              <div className="flex items-center gap-3">
                <span className="rounded-lg bg-primary-light p-2.5">
                  <Icon className="h-5 w-5 text-primary" />
                </span>
                <div>
                  <p className="text-xs text-gray-500">{stat.title}</p>
                  <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by SKU or product ID..."
            className="w-full rounded-lg border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none"
        >
          <option value="">All Status</option>
          <option value="in_stock">In Stock</option>
          <option value="low_stock">Low Stock</option>
          <option value="out_of_stock">Out of Stock</option>
        </select>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        {loading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <Package className="h-10 w-10 text-gray-300" />
            <p className="mt-3 text-sm font-medium text-gray-700">No inventory items found</p>
            <p className="mt-1 text-xs text-gray-400">
              {search || statusFilter ? 'Try adjusting your filters' : 'Add inventory items using the button above'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3 font-medium">SKU</th>
                  <th className="px-6 py-3 font-medium">Product ID</th>
                  <th className="px-6 py-3 font-medium">Warehouse</th>
                  <th className="px-6 py-3 font-medium">Qty On Hand</th>
                  <th className="px-6 py-3 font-medium">Reorder Point</th>
                  <th className="px-6 py-3 font-medium">Status</th>
                  <th className="px-6 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((item) => {
                  const status = getStatus(item);
                  const badge = stockStatusLabel(status);
                  return (
                    <tr
                      key={item.id}
                      className={cn(
                        'border-b border-gray-50 transition-colors hover:bg-gray-50',
                        status === 'low_stock' ? 'bg-yellow-50/50' : status === 'out_of_stock' ? 'bg-red-50/50' : '',
                      )}
                    >
                      <td className="px-6 py-4 text-sm font-mono text-gray-500">{item.sku}</td>
                      <td className="px-6 py-4 text-sm font-mono text-gray-500 max-w-[140px] truncate">{item.productId}</td>
                      <td className="px-6 py-4 text-sm text-gray-500">
                        {warehouseMap.get(item.warehouseId) || item.warehouseName || item.warehouseId || '—'}
                      </td>
                      <td className="px-6 py-4 text-sm">
                        <span className={cn(
                          'font-medium',
                          status === 'out_of_stock' ? 'text-red-600' : status === 'low_stock' ? 'text-yellow-600' : 'text-gray-900',
                        )}>
                          {item.quantityOnHand}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">{item.reorderPoint}</td>
                      <td className="px-6 py-4">
                        <span className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                          badge.classes,
                        )}>
                          {badge.label}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <button
                          onClick={() => { setAdjustItem(item); setAdjustQty(0); setAdjustReason(''); }}
                          className="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50"
                        >
                          Adjust
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Adjust Stock Dialog */}
      {adjustItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setAdjustItem(null)}>
          <div className="w-full max-w-sm rounded-2xl border border-gray-200 bg-white p-6 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-base font-semibold text-gray-900">Adjust Stock</h3>
                <p className="text-xs text-gray-500">SKU: {adjustItem.sku}</p>
              </div>
              <button onClick={() => setAdjustItem(null)} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"><X className="h-4 w-4" /></button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Quantity</label>
                <input
                  type="number"
                  min={1}
                  value={adjustQty || ''}
                  onChange={(e) => setAdjustQty(Math.max(0, parseInt(e.target.value) || 0))}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Reason (optional)</label>
                <input
                  value={adjustReason}
                  onChange={(e) => setAdjustReason(e.target.value)}
                  placeholder="e.g. Stock received, Damage, etc."
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>
            <div className="mt-5 flex gap-2">
              <button
                onClick={() => handleAdjust('add')}
                disabled={adjustLoading || adjustQty <= 0}
                className="flex flex-1 items-center justify-center gap-1 rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-green-700 disabled:opacity-50"
              >
                <Plus className="h-3.5 w-3.5" /> Add
              </button>
              <button
                onClick={() => handleAdjust('remove')}
                disabled={adjustLoading || adjustQty <= 0}
                className="flex flex-1 items-center justify-center gap-1 rounded-lg bg-red-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:opacity-50"
              >
                <Minus className="h-3.5 w-3.5" /> Remove
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Warehouse Dialog */}
      {showAddWarehouse && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setShowAddWarehouse(false)}>
          <div className="w-full max-w-lg rounded-2xl border border-gray-200 bg-white p-6 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-lg font-semibold text-gray-900">Add Warehouse</h3>
              <button onClick={() => setShowAddWarehouse(false)} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"><X className="h-4 w-4" /></button>
            </div>
            {whError && <p className="mb-4 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{whError}</p>}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Code *</label>
                <input value={whForm.code} onChange={(e) => setWhForm((f) => ({ ...f, code: e.target.value }))}
                  placeholder="e.g. WH-CTG-002" className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Name *</label>
                <input value={whForm.name} onChange={(e) => setWhForm((f) => ({ ...f, name: e.target.value }))}
                  placeholder="e.g. Chittagong Warehouse" className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div className="col-span-2">
                <label className="mb-1 block text-xs font-medium text-gray-700">Address *</label>
                <input value={whForm.address} onChange={(e) => setWhForm((f) => ({ ...f, address: e.target.value }))}
                  placeholder="Street address" className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">City *</label>
                <input value={whForm.city} onChange={(e) => setWhForm((f) => ({ ...f, city: e.target.value }))}
                  placeholder="e.g. Chittagong" className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">State / Division *</label>
                <input value={whForm.state} onChange={(e) => setWhForm((f) => ({ ...f, state: e.target.value }))}
                  placeholder="e.g. Chittagong Division" className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Country *</label>
                <input value={whForm.country} onChange={(e) => setWhForm((f) => ({ ...f, country: e.target.value }))}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Postal Code *</label>
                <input value={whForm.postalCode} onChange={(e) => setWhForm((f) => ({ ...f, postalCode: e.target.value }))}
                  placeholder="e.g. 4000" className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Phone</label>
                <input value={whForm.phone} onChange={(e) => setWhForm((f) => ({ ...f, phone: e.target.value }))}
                  placeholder="+880..." className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Email</label>
                <input value={whForm.email} onChange={(e) => setWhForm((f) => ({ ...f, email: e.target.value }))}
                  placeholder="warehouse@..." className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowAddWarehouse(false)}
                className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                Cancel
              </button>
              <button onClick={handleAddWarehouse} disabled={whLoading}
                className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-50">
                {whLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <MapPin className="h-4 w-4" />}
                Create Warehouse
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Inventory Item Dialog */}
      {showAddItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={() => setShowAddItem(false)}>
          <div className="w-full max-w-md rounded-2xl border border-gray-200 bg-white p-6 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-5">
              <h3 className="text-lg font-semibold text-gray-900">Add Inventory Item</h3>
              <button onClick={() => setShowAddItem(false)} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"><X className="h-4 w-4" /></button>
            </div>
            {itemError && <p className="mb-4 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{itemError}</p>}
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Product *</label>
                <select value={itemForm.productId} onChange={(e) => onProductSelect(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary">
                  <option value="">Select a product...</option>
                  {products.map((p) => (
                    <option key={p.id} value={p.id}>{p.name} ({p.sku})</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">SKU *</label>
                <input value={itemForm.sku} onChange={(e) => setItemForm((f) => ({ ...f, sku: e.target.value }))}
                  placeholder="Auto-filled from product" className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-700">Warehouse *</label>
                <select value={itemForm.warehouseId} onChange={(e) => setItemForm((f) => ({ ...f, warehouseId: e.target.value }))}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary">
                  <option value="">Select a warehouse...</option>
                  {warehouses.map((w) => (
                    <option key={w.id} value={w.id}>{w.name} ({w.code})</option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-700">Initial Qty</label>
                  <input type="number" min={0} value={itemForm.initialQuantity || ''}
                    onChange={(e) => setItemForm((f) => ({ ...f, initialQuantity: parseInt(e.target.value) || 0 }))}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-700">Reorder Pt</label>
                  <input type="number" min={0} value={itemForm.reorderPoint || ''}
                    onChange={(e) => setItemForm((f) => ({ ...f, reorderPoint: parseInt(e.target.value) || 0 }))}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-700">Reorder Qty</label>
                  <input type="number" min={0} value={itemForm.reorderQuantity || ''}
                    onChange={(e) => setItemForm((f) => ({ ...f, reorderQuantity: parseInt(e.target.value) || 0 }))}
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowAddItem(false)}
                className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                Cancel
              </button>
              <button onClick={handleAddItem} disabled={itemLoading}
                className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-50">
                {itemLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                Add Item
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
