'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Shield, Eye, EyeOff } from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { ApiError } from '@/lib/api';

export default function PlatformLoginPage() {
  const router = useRouter();
  const login = useAuthStore((s) => s.login);
  const setAuth = useAuthStore((s) => s.setAuth);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const result = await login(email, password, '');
      if (result.user.role === 'super_admin') {
        router.push('/platform/dashboard');
      } else {
        setError('This portal is for platform administrators only');
      }
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || 'Invalid credentials');
      } else {
        setError((err as Error).message || 'Invalid credentials');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-950 px-4">
      <div className="w-full max-w-md">
        {/* Brand */}
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-violet-600">
            <Shield className="h-7 w-7 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-white">Platform Admin</h1>
          <p className="mt-1 text-sm text-gray-400">
            SaaS management console
          </p>
        </div>

        {/* Card */}
        <div className="rounded-xl border border-gray-800 bg-gray-900 p-8 shadow-2xl">
          <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
              <div className="rounded-lg bg-red-900/30 border border-red-800 px-4 py-3 text-sm text-red-400">
                {error}
              </div>
            )}

            <div>
              <label htmlFor="email" className="mb-1.5 block text-sm font-medium text-gray-300">
                Email
              </label>
              <input
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="super@saajan.com.bd"
                className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3.5 py-2.5 text-sm text-white placeholder:text-gray-500 focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
              />
            </div>

            <div>
              <label htmlFor="password" className="mb-1.5 block text-sm font-medium text-gray-300">
                Password
              </label>
              <div className="relative">
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter your password"
                  className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3.5 py-2.5 pr-10 text-sm text-white placeholder:text-gray-500 focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full rounded-lg bg-violet-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? 'Signing in...' : 'Sign in to Platform'}
            </button>
          </form>

          <div className="mt-4 rounded-lg bg-gray-800 px-4 py-3 text-xs text-gray-400">
            <p className="mb-2 font-medium text-gray-300">Demo credentials:</p>
            <div className="flex items-center justify-between">
              <p>super@saajan.com.bd / super123</p>
              <button
                type="button"
                onClick={() => { setEmail('super@saajan.com.bd'); setPassword('super123'); }}
                className="rounded bg-violet-600/20 px-2 py-0.5 text-[10px] font-semibold text-violet-400 hover:bg-violet-600/30 transition-colors"
              >
                Fill
              </button>
            </div>
          </div>
        </div>

        <p className="mt-6 text-center text-xs text-gray-600">
          Looking for the store admin?{' '}
          <a href="/admin/login" className="text-violet-500 hover:text-violet-400">
            Sign in here
          </a>
        </p>
      </div>
    </div>
  );
}
