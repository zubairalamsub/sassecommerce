'use client';

import { Package, AlertTriangle, Warehouse } from 'lucide-react';
import { cn, formatCurrency } from '@/lib/utils';

const stats = [
  { title: 'Total Items', value: '1,842', icon: Package },
  { title: 'Low Stock Alerts', value: '12', icon: AlertTriangle },
  { title: 'Warehouses', value: '3', icon: Warehouse },
];

type StockStatus = 'in_stock' | 'low_stock' | 'out_of_stock';

const inventoryItems = [
  {
    id: '1',
    sku: 'SAR-JAM-001',
    product: 'Jamdani Saree',
    warehouse: 'Dhaka Central',
    qtyOnHand: 24,
    reorderPoint: 10,
    status: 'in_stock' as StockStatus,
  },
  {
    id: '2',
    sku: 'PNJ-COT-002',
    product: 'Cotton Panjabi',
    warehouse: 'Dhaka Central',
    qtyOnHand: 56,
    reorderPoint: 20,
    status: 'in_stock' as StockStatus,
  },
  {
    id: '3',
    sku: 'KNT-NAK-003',
    product: 'Nakshi Kantha',
    warehouse: 'Chittagong Port',
    qtyOnHand: 5,
    reorderPoint: 10,
    status: 'low_stock' as StockStatus,
  },
  {
    id: '4',
    sku: 'DUP-MUS-004',
    product: 'Muslin Dupatta',
    warehouse: 'Dhaka Central',
    qtyOnHand: 0,
    reorderPoint: 15,
    status: 'out_of_stock' as StockStatus,
  },
  {
    id: '5',
    sku: 'SHO-MOJ-005',
    product: 'Leather Mojari',
    warehouse: 'Chittagong Port',
    qtyOnHand: 38,
    reorderPoint: 12,
    status: 'in_stock' as StockStatus,
  },
  {
    id: '6',
    sku: 'HND-BRS-006',
    product: 'Brass Handicraft Vase',
    warehouse: 'Dhaka Central',
    qtyOnHand: 3,
    reorderPoint: 8,
    status: 'low_stock' as StockStatus,
  },
  {
    id: '7',
    sku: 'TEX-SLK-007',
    product: 'Rajshahi Silk',
    warehouse: 'Rajshahi Depot',
    qtyOnHand: 42,
    reorderPoint: 15,
    status: 'in_stock' as StockStatus,
  },
];

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

function rowBg(status: StockStatus) {
  switch (status) {
    case 'low_stock':
      return 'bg-yellow-50';
    case 'out_of_stock':
      return 'bg-red-50';
    default:
      return '';
  }
}

export default function InventoryPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Inventory</h1>
        <p className="mt-1 text-sm text-gray-500">
          Track stock levels across all warehouses.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div
              key={stat.title}
              className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm"
            >
              <div className="flex items-center gap-4">
                <span className="rounded-lg bg-primary-light p-3">
                  <Icon className="h-6 w-6 text-primary" />
                </span>
                <div>
                  <p className="text-sm text-gray-500">{stat.title}</p>
                  <p className="text-2xl font-bold text-gray-900">{stat.value}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Table */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                <th className="px-6 py-3 font-medium">SKU</th>
                <th className="px-6 py-3 font-medium">Product</th>
                <th className="px-6 py-3 font-medium">Warehouse</th>
                <th className="px-6 py-3 font-medium">Qty On Hand</th>
                <th className="px-6 py-3 font-medium">Reorder Point</th>
                <th className="px-6 py-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {inventoryItems.map((item) => {
                const badge = stockStatusLabel(item.status);
                return (
                  <tr
                    key={item.id}
                    className={cn(
                      'border-b border-gray-50 transition-colors hover:bg-gray-50',
                      rowBg(item.status),
                    )}
                  >
                    <td className="px-6 py-4 text-sm text-gray-500 font-mono">
                      {item.sku}
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">
                      {item.product}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">{item.warehouse}</td>
                    <td className="px-6 py-4 text-sm">
                      <span
                        className={cn(
                          'font-medium',
                          item.status === 'out_of_stock'
                            ? 'text-red-600'
                            : item.status === 'low_stock'
                              ? 'text-yellow-600'
                              : 'text-gray-900',
                        )}
                      >
                        {item.qtyOnHand}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {item.reorderPoint}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={cn(
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                          badge.classes,
                        )}
                      >
                        {badge.label}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
