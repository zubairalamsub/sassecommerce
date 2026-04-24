'use client';

import { useState } from 'react';
import Link from 'next/link';
import {
  Package,
  Heart,
  MapPin,
  ChevronRight,
  Mail,
  Phone,
  Calendar,
  Pencil,
  X,
  Check,
  KeyRound,
  Loader2,
} from 'lucide-react';
import AuthGuard from '@/components/auth/auth-guard';
import { useAuthStore } from '@/stores/auth';
import { authApi } from '@/lib/api';
import { formatDate } from '@/lib/utils';

const TENANT_ID = 'tenant_saajan';

const quickLinks = [
  {
    label: 'My Orders',
    description: 'Track and manage your orders',
    href: '/account/orders',
    icon: Package,
  },
  {
    label: 'Wishlist',
    description: 'Items you have saved',
    href: '/wishlist',
    icon: Heart,
  },
  {
    label: 'Addresses',
    description: 'Manage delivery addresses',
    href: '/account/addresses',
    icon: MapPin,
  },
];

function AccountContent() {
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);
  const setAuth = useAuthStore((s) => s.setAuth);
  const tenantId = useAuthStore((s) => s.tenantId);

  const [editMode, setEditMode] = useState(false);
  const [firstName, setFirstName] = useState(user?.first_name ?? '');
  const [lastName, setLastName] = useState(user?.last_name ?? '');
  const [phone, setPhone] = useState(user?.phone ?? '');
  const [profileLoading, setProfileLoading] = useState(false);
  const [profileError, setProfileError] = useState('');
  const [profileSuccess, setProfileSuccess] = useState(false);

  const [showPasswordForm, setShowPasswordForm] = useState(false);
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [passwordError, setPasswordError] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  if (!user) return null;

  const initials = `${user.first_name[0] || ''}${user.last_name[0] || ''}`.toUpperCase();

  async function handleSaveProfile(e: React.FormEvent) {
    e.preventDefault();
    if (!token) return;
    if (!user) return;
    setProfileError('');
    setProfileLoading(true);
    try {
      const updated = await authApi.updateProfile(
        user.id,
        { first_name: firstName, last_name: lastName, phone: phone || undefined },
        tenantId ?? TENANT_ID,
        token,
      );
      setAuth({ ...user, ...updated } as typeof user, token, tenantId);
      setProfileSuccess(true);
      setEditMode(false);
      setTimeout(() => setProfileSuccess(false), 3000);
    } catch (err) {
      setProfileError((err as Error).message || 'Failed to update profile.');
    } finally {
      setProfileLoading(false);
    }
  }

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault();
    if (!token) return;
    setPasswordError('');

    if (newPassword.length < 8) {
      setPasswordError('New password must be at least 8 characters.');
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError('Passwords do not match.');
      return;
    }

    setPasswordLoading(true);
    try {
      await authApi.changePassword(oldPassword, newPassword, tenantId ?? TENANT_ID, token);
      setPasswordSuccess(true);
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setShowPasswordForm(false);
      setTimeout(() => setPasswordSuccess(false), 3000);
    } catch (err) {
      setPasswordError((err as Error).message || 'Failed to change password.');
    } finally {
      setPasswordLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="mb-8 text-2xl font-bold text-gray-900">My Account</h1>

      {profileSuccess && (
        <div className="mb-4 flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">
          <Check className="h-4 w-4" /> Profile updated successfully.
        </div>
      )}
      {passwordSuccess && (
        <div className="mb-4 flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700">
          <Check className="h-4 w-4" /> Password changed successfully.
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-3">
        {/* Profile Card */}
        <div className="md:col-span-2 rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
          {editMode ? (
            <form onSubmit={handleSaveProfile} className="space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-base font-semibold text-gray-900">Edit Profile</h2>
                <button
                  type="button"
                  onClick={() => { setEditMode(false); setProfileError(''); }}
                  className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              {profileError && (
                <p className="text-sm text-red-600">{profileError}</p>
              )}

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-600">First Name</label>
                  <input
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    required
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-600">Last Name</label>
                  <input
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    required
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-600">Phone</label>
                <input
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  placeholder="+880 1XXXXXXXXX"
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
              <div className="flex gap-3">
                <button
                  type="submit"
                  disabled={profileLoading}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-60"
                >
                  {profileLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
                  Save Changes
                </button>
                <button
                  type="button"
                  onClick={() => { setEditMode(false); setProfileError(''); }}
                  className="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                >
                  Cancel
                </button>
              </div>
            </form>
          ) : (
            <div className="flex items-start gap-5">
              <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-primary text-xl font-bold text-white">
                {initials}
              </div>
              <div className="flex-1 min-w-0">
                <h2 className="text-lg font-semibold text-gray-900">
                  {user.first_name} {user.last_name}
                </h2>
                <p className="mt-0.5 text-sm text-gray-500 capitalize">{user.role}</p>

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

                <div className="mt-5 flex flex-wrap gap-2">
                  <button
                    onClick={() => { setEditMode(true); setFirstName(user.first_name); setLastName(user.last_name); setPhone(user.phone ?? ''); }}
                    className="flex items-center gap-1.5 rounded-lg border border-primary px-4 py-2 text-sm font-medium text-primary transition-colors hover:bg-primary-light"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                    Edit Profile
                  </button>
                  <button
                    onClick={() => setShowPasswordForm(!showPasswordForm)}
                    className="flex items-center gap-1.5 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                  >
                    <KeyRound className="h-3.5 w-3.5" />
                    Change Password
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Change Password form */}
          {showPasswordForm && !editMode && (
            <form onSubmit={handleChangePassword} className="mt-6 space-y-4 border-t border-gray-200 pt-6">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-gray-900">Change Password</h3>
                <button
                  type="button"
                  onClick={() => { setShowPasswordForm(false); setPasswordError(''); }}
                  className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
              {passwordError && <p className="text-sm text-red-600">{passwordError}</p>}
              <input
                type="password"
                placeholder="Current password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                required
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <input
                type="password"
                placeholder="New password (min 8 characters)"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
                minLength={8}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <input
                type="password"
                placeholder="Confirm new password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
              <button
                type="submit"
                disabled={passwordLoading}
                className="flex items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-gray-700 disabled:opacity-60"
              >
                {passwordLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                Update Password
              </button>
            </form>
          )}
        </div>

        {/* Account Status */}
        <div className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-900">Account Status</h3>
          <div className="mt-4 space-y-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-500">Status</span>
              <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 capitalize">
                {user.status}
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
            return (
              <Link
                key={link.label}
                href={link.href}
                className="group flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-5 shadow-sm transition-all hover:border-primary hover:shadow-md"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Icon className="h-5 w-5" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-gray-900">{link.label}</p>
                  <p className="text-xs text-gray-500">{link.description}</p>
                </div>
                <ChevronRight className="h-4 w-4 text-gray-400 transition-transform group-hover:translate-x-0.5" />
              </Link>
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
