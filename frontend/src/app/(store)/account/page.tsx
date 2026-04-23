'use client';

import Link from 'next/link';
import { Package, Heart, MapPin, ChevronRight, Mail, Phone, Calendar } from 'lucide-react';
import AuthGuard from '@/components/auth/auth-guard';
import { useAuthStore } from '@/stores/auth';
import { formatDate } from '@/lib/utils';

function AccountContent() {
  const user = useAuthStore((s) => s.user);

  if (!user) return null;

  const initials = `${user.first_name[0] || ''}${user.last_name[0] || ''}`.toUpperCase();

  const quickLinks = [
    {
      label: 'My Orders',
      description: 'Track and manage your orders',
      href: '/account/orders',
      icon: Package,
      available: true,
    },
    {
      label: 'Wishlist',
      description: 'Items you have saved',
      href: '/wishlist',
      icon: Heart,
      available: true,
    },
    {
      label: 'Addresses',
      description: 'Manage delivery addresses',
      href: '/account/addresses',
      icon: MapPin,
      available: true,
    },
  ];

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="mb-8 text-2xl font-bold text-gray-900">My Account</h1>

      <div className="grid gap-6 md:grid-cols-3">
        {/* Profile Card */}
        <div className="md:col-span-2 rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="flex items-start gap-5">
            {/* Avatar */}
            <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-primary text-xl font-bold text-white">
              {initials}
            </div>

            <div className="flex-1 min-w-0">
              <h2 className="text-lg font-semibold text-gray-900">
                {user.first_name} {user.last_name}
              </h2>
              <p className="mt-0.5 text-sm text-gray-500">Customer</p>

              <div className="mt-4 space-y-2">
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <Mail className="h-4 w-4 text-gray-400" />
                  <span className="truncate">{user.email}</span>
                </div>
                {user.phone && (
                  <div className="flex items-center gap-2 text-sm text-gray-600">
                    <Phone className="h-4 w-4 text-gray-400" />
                    <span>{user.phone}</span>
                  </div>
                )}
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <Calendar className="h-4 w-4 text-gray-400" />
                  <span>Member since {formatDate(user.created_at)}</span>
                </div>
              </div>

              <button className="mt-5 rounded-lg border border-primary px-4 py-2 text-sm font-medium text-primary transition-colors hover:bg-primary-light">
                Edit Profile
              </button>
            </div>
          </div>
        </div>

        {/* Account Status */}
        <div className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-900">Account Status</h3>
          <div className="mt-4 space-y-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-500">Status</span>
              <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                Active
              </span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-500">Email verified</span>
              <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                user.email_verified
                  ? 'bg-green-100 text-green-800'
                  : 'bg-yellow-100 text-yellow-800'
              }`}>
                {user.email_verified ? 'Verified' : 'Not verified'}
              </span>
            </div>
            {user.last_login_at && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-gray-500">Last login</span>
                <span className="text-gray-700">{formatDate(user.last_login_at)}</span>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Quick Links */}
      <div className="mt-8">
        <h2 className="mb-4 text-lg font-semibold text-gray-900">Quick Links</h2>
        <div className="grid gap-4 sm:grid-cols-3">
          {quickLinks.map((link) => {
            const Icon = link.icon;
            return link.available ? (
              <Link
                key={link.label}
                href={link.href}
                className="group flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-5 shadow-sm transition-all hover:border-primary hover:shadow-md"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
                  <Icon className="h-5 w-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-900">{link.label}</p>
                  <p className="text-xs text-gray-500">{link.description}</p>
                </div>
                <ChevronRight className="h-4 w-4 text-gray-400 transition-transform group-hover:translate-x-0.5" />
              </Link>
            ) : (
              <div
                key={link.label}
                className="flex items-center gap-4 rounded-xl border border-gray-100 bg-gray-50 p-5 opacity-60"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-gray-100 text-gray-400">
                  <Icon className="h-5 w-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-500">{link.label}</p>
                  <p className="text-xs text-gray-400">Coming soon</p>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

export default function AccountPage() {
  return (
    <AuthGuard requiredRole="customer">
      <AccountContent />
    </AuthGuard>
  );
}
