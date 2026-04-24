'use client';

import { useState, useEffect } from 'react';
import { Package, AlertTriangle, Warehouse, Loader2, Search, Plus, Minus } from 'lucide-react';
import { cn } from '@/lib/utils';
import { inventoryApi, type InventoryItem, type Warehouse as WarehouseType } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

type StockStatus = 'in_stock' | 'low_stock' | 'out_of_stock';

function getStatus(item: InventoryItem): StockStatus {
  if (item.quantity_on_hand <= 0) return 'out_of_stock';
  if (item.quantity_on_hand <= item.reorder_point) return 'low_stock';
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

export default function InventoryPage() {
  const { tenantId } = useAuthStore();
  const [items, setItems] = useState<InventoryItem[]>([]);
  const [warehouses, setWarehouses] = useState<WarehouseType[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [adjustItem, setAdjustItem] = useState<InventoryItem | null>(null);
  const [adjustQty, setAdjustQty] = useState(0);
  const [adjustReason, setAdjustReason] = useState('');
  const [adjustLoading, setAdjustLoading] = useState(false);

  useEffect(() => {
    loadInventory();
  }, [tenantId]);

  async function loadInventory() {
    if (!tenantId) return;
    setLoading(true);
    try {
      const [itemsRes, warehousesRes] = await Promise.allSettled([
        inventoryApi.listItems(tenantId, 1, 200),
        inventoryApi.listWarehouses(tenantId),
      ]);
      if (itemsRes.status === 'fulfilled') setItems(itemsRes.value.data ?? []);
      if (warehousesRes.status === 'fulfilled') setWarehouses(warehousesRes.value);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }

  async function handleAdjust(type: 'add' | 'remove') {
    if (!adjustItem || !tenantId || adjustQty <= 0) return;
    setAdjustLoading(true);
    try {
      const updated = await inventoryApi.adjustStock(adjustItem.id, {
        quantity: type === 'add' ? adjustQty : -adjustQty,
        reason: adjustReason || (type === 'add' ? 'Stock received' : 'Stock adjustment'),
        type,
      }, tenantId);
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

  const warehouseMap = new Map(warehouses.map((w) => [w.id, w.name]));

  const filtered = items.filter((item) => {
    const status = getStatus(item);
    if (statusFilter && status !== statusFilter) return false;
    if (search) {
      const q = search.toLowerCase();
      return item.sku.toLowerCase().includes(q) || item.product_id.toLowerCase().includes(q);
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
    { title: 'Warehouses', value: warehouseCount, icon: Warehouse, color: 'text-primary' },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Inventory</h1>
          <p className="mt-1 text-sm text-gray-500">Track stock levels across all warehouses.</p>
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
              {search || statusFilter ? 'Try adjusting your filters' : 'Inventory items will appear here'}
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
                      <td className="px-6 py-4 text-sm font-mono text-gray-500 max-w-[140px] truncate">{item.product_id}</td>
                      <td className="px-6 py-4 text-sm text-gray-500">
                        {warehouseMap.get(item.warehouse_id) || item.warehouse_id || '—'}
                      </td>
                      <td className="px-6 py-4 text-sm">
                        <span className={cn(
                          'font-medium',
                          status === 'out_of_stock' ? 'text-red-600' : status === 'low_stock' ? 'text-yellow-600' : 'text-gray-900',
                        )}>
                          {item.quantity_on_hand}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500">{item.reorder_point}</td>
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
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-sm rounded-2xl border border-gray-200 bg-white p-6 shadow-xl">
            <h3 className="text-base font-semibold text-gray-900 mb-1">Adjust Stock</h3>
            <p className="text-xs text-gray-500 mb-4">SKU: {adjustItem.sku}</p>
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
              <button
                onClick={() => setAdjustItem(null)}
                className="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
