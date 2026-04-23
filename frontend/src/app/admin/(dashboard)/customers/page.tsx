'use client';

import { Users, UserCheck, UserPlus } from 'lucide-react';
import { cn, formatCurrency, formatDate, statusColor } from '@/lib/utils';

const stats = [
  { title: 'Total Customers', value: '1,245', icon: Users },
  { title: 'Active', value: '1,089', icon: UserCheck },
  { title: 'New This Month', value: '47', icon: UserPlus },
];

const customers = [
  {
    id: '1',
    name: 'Rahim Uddin',
    email: 'rahim.uddin@example.com',
    phone: '+880 1711-234567',
    orders: 12,
    totalSpent: 45600,
    status: 'active',
    joined: '2025-08-14',
  },
  {
    id: '2',
    name: 'Fatima Akter',
    email: 'fatima.akter@example.com',
    phone: '+880 1812-345678',
    orders: 8,
    totalSpent: 92300,
    status: 'active',
    joined: '2025-10-03',
  },
  {
    id: '3',
    name: 'Kamal Hossain',
    email: 'kamal.hossain@example.com',
    phone: '+880 1912-456789',
    orders: 3,
    totalSpent: 12800,
    status: 'active',
    joined: '2026-01-18',
  },
  {
    id: '4',
    name: 'Nusrat Jahan',
    email: 'nusrat.jahan@example.com',
    phone: '+880 1612-567890',
    orders: 15,
    totalSpent: 67400,
    status: 'active',
    joined: '2025-06-22',
  },
  {
    id: '5',
    name: 'Shakib Ahmed',
    email: 'shakib.ahmed@example.com',
    phone: '+880 1512-678901',
    orders: 1,
    totalSpent: 2100,
    status: 'inactive',
    joined: '2026-03-01',
  },
  {
    id: '6',
    name: 'Tahmina Begum',
    email: 'tahmina.begum@example.com',
    phone: '+880 1412-789012',
    orders: 6,
    totalSpent: 34500,
    status: 'active',
    joined: '2025-12-10',
  },
];

export default function CustomersPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Customers</h1>
        <p className="mt-1 text-sm text-gray-500">
          View and manage your customer base.
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
                <th className="px-6 py-3 font-medium">Name</th>
                <th className="px-6 py-3 font-medium">Email</th>
                <th className="px-6 py-3 font-medium">Phone</th>
                <th className="px-6 py-3 font-medium">Orders</th>
                <th className="px-6 py-3 font-medium">Total Spent</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium">Joined</th>
              </tr>
            </thead>
            <tbody>
              {customers.map((customer) => (
                <tr
                  key={customer.id}
                  className="border-b border-gray-50 transition-colors hover:bg-gray-50"
                >
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full bg-primary-light text-sm font-semibold text-primary">
                        {customer.name
                          .split(' ')
                          .map((n) => n[0])
                          .join('')}
                      </div>
                      <span className="text-sm font-medium text-gray-900">
                        {customer.name}
                      </span>
                    </div>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">{customer.email}</td>
                  <td className="px-6 py-4 text-sm text-gray-500">{customer.phone}</td>
                  <td className="px-6 py-4 text-sm text-gray-900">{customer.orders}</td>
                  <td className="px-6 py-4 text-sm text-gray-900">
                    {formatCurrency(customer.totalSpent)}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={cn(
                        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                        statusColor(customer.status),
                      )}
                    >
                      {customer.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {formatDate(customer.joined)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
