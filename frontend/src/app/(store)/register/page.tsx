'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { UserPlus, Eye, EyeOff, Phone } from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { ApiError } from '@/lib/api';

const TENANT_ID = 'tenant_saajan';

export default function RegisterPage() {
  const router = useRouter();
  const register = useAuthStore((s) => s.register);

  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  function validate(): boolean {
    const errs: Record<string, string> = {};

    if (!firstName.trim()) errs.firstName = 'First name is required';
    if (!lastName.trim()) errs.lastName = 'Last name is required';
    if (!email.trim()) {
      errs.email = 'Email is required';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      errs.email = 'Enter a valid email address';
    }
    if (phone && !/^\d{10,11}$/.test(phone.replace(/\s|-/g, ''))) {
      errs.phone = 'Enter a valid phone number (10-11 digits)';
    }
    if (!password) {
      errs.password = 'Password is required';
    } else if (password.length < 6) {
      errs.password = 'Password must be at least 6 characters';
    }
    if (!confirmPassword) {
      errs.confirmPassword = 'Please confirm your password';
    } else if (password !== confirmPassword) {
      errs.confirmPassword = 'Passwords do not match';
    }

    setErrors(errs);
    return Object.keys(errs).length === 0;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!validate()) return;

    setLoading(true);

    try {
      await register(
        {
          email,
          username: email.split('@')[0],
          password,
          first_name: firstName,
          last_name: lastName,
          phone: phone ? '+880' + phone : undefined,
        },
        TENANT_ID,
      );
      router.push('/products');
    } catch (err) {
      if (err instanceof ApiError) {
        setErrors({ email: err.message || 'Registration failed' });
      } else {
        setErrors({ email: (err as Error).message || 'Registration failed' });
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4 py-12">
      <div className="w-full max-w-lg">
        <div className="rounded-2xl border border-gray-200 bg-white p-8 shadow-sm">
          {/* Header */}
          <div className="mb-8 text-center">
            <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-primary-light">
              <UserPlus className="h-7 w-7 text-primary" />
            </div>
            <h1 className="text-2xl font-bold text-gray-900">Create Account</h1>
            <p className="mt-1 text-sm text-gray-500">
              Join Saajan to start shopping
            </p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-5">
            {/* First + Last Name (side by side) */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="firstName" className="mb-1 block text-sm font-medium text-gray-700">
                  First Name
                </label>
                <input
                  id="firstName"
                  type="text"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm focus:outline-none focus:ring-1 ${
                    errors.firstName
                      ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-primary focus:ring-primary'
                  }`}
                  placeholder="Rahim"
                />
                {errors.firstName && (
                  <p className="mt-1 text-xs text-red-500">{errors.firstName}</p>
                )}
              </div>
              <div>
                <label htmlFor="lastName" className="mb-1 block text-sm font-medium text-gray-700">
                  Last Name
                </label>
                <input
                  id="lastName"
                  type="text"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm focus:outline-none focus:ring-1 ${
                    errors.lastName
                      ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-primary focus:ring-primary'
                  }`}
                  placeholder="Ahmed"
                />
                {errors.lastName && (
                  <p className="mt-1 text-xs text-red-500">{errors.lastName}</p>
                )}
              </div>
            </div>

            {/* Email */}
            <div>
              <label htmlFor="email" className="mb-1 block text-sm font-medium text-gray-700">
                Email Address
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={`w-full rounded-lg border px-4 py-2.5 text-sm focus:outline-none focus:ring-1 ${
                  errors.email
                    ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
                    : 'border-gray-300 focus:border-primary focus:ring-primary'
                }`}
                placeholder="rahim@example.com"
              />
              {errors.email && (
                <p className="mt-1 text-xs text-red-500">{errors.email}</p>
              )}
            </div>

            {/* Phone */}
            <div>
              <label htmlFor="phone" className="mb-1 block text-sm font-medium text-gray-700">
                Phone Number <span className="text-gray-400">(optional)</span>
              </label>
              <div className="flex">
                <span className="inline-flex items-center gap-1 rounded-l-lg border border-r-0 border-gray-300 bg-gray-50 px-3 text-sm text-gray-500">
                  <Phone className="h-3.5 w-3.5" />
                  +880
                </span>
                <input
                  id="phone"
                  type="tel"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value.replace(/[^\d]/g, ''))}
                  className={`w-full rounded-r-lg border px-4 py-2.5 text-sm focus:outline-none focus:ring-1 ${
                    errors.phone
                      ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-primary focus:ring-primary'
                  }`}
                  placeholder="1712345678"
                  maxLength={11}
                />
              </div>
              {errors.phone && (
                <p className="mt-1 text-xs text-red-500">{errors.phone}</p>
              )}
            </div>

            {/* Password */}
            <div>
              <label htmlFor="password" className="mb-1 block text-sm font-medium text-gray-700">
                Password
              </label>
              <div className="relative">
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className={`w-full rounded-lg border px-4 py-2.5 pr-10 text-sm focus:outline-none focus:ring-1 ${
                    errors.password
                      ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-primary focus:ring-primary'
                  }`}
                  placeholder="At least 6 characters"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.password && (
                <p className="mt-1 text-xs text-red-500">{errors.password}</p>
              )}
            </div>

            {/* Confirm Password */}
            <div>
              <label htmlFor="confirmPassword" className="mb-1 block text-sm font-medium text-gray-700">
                Confirm Password
              </label>
              <div className="relative">
                <input
                  id="confirmPassword"
                  type={showConfirm ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className={`w-full rounded-lg border px-4 py-2.5 pr-10 text-sm focus:outline-none focus:ring-1 ${
                    errors.confirmPassword
                      ? 'border-red-400 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-primary focus:ring-primary'
                  }`}
                  placeholder="Re-enter your password"
                />
                <button
                  type="button"
                  onClick={() => setShowConfirm(!showConfirm)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                >
                  {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.confirmPassword && (
                <p className="mt-1 text-xs text-red-500">{errors.confirmPassword}</p>
              )}
            </div>

            {/* Submit */}
            <button
              type="submit"
              disabled={loading}
              className="w-full rounded-lg bg-primary py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-dark disabled:cursor-not-allowed disabled:opacity-60"
            >
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  Creating account...
                </span>
              ) : (
                'Create Account'
              )}
            </button>
          </form>

          {/* Footer link */}
          <p className="mt-6 text-center text-sm text-gray-500">
            Already have an account?{' '}
            <Link href="/login" className="font-medium text-primary hover:text-primary-dark">
              Sign in
            </Link>
          </p>
        </div>

        <p className="mt-4 text-center text-xs text-gray-400">
          <Link href="/products" className="hover:text-primary">
            Back to store
          </Link>
        </p>
      </div>
    </div>
  );
}
