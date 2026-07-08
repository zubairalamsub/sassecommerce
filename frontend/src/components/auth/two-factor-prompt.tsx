'use client';

import { useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import { useAuthStore, type AuthUser } from '@/stores/auth';
import { ApiError } from '@/lib/api';

// Second step of a 2FA-gated login (A04-3). Rendered by the login pages once
// login() reports `twoFactorRequired`. Submits the TOTP/backup code to the
// server route, which validates it against the HttpOnly challenge cookie and
// swaps in the auth cookie; on success `onVerified` receives the resolved user
// so each page can run its own role-based redirect.
export function TwoFactorPrompt({
  onVerified,
  tone = 'light',
}: {
  onVerified: (user: AuthUser) => void;
  tone?: 'light' | 'dark';
}) {
  const verifyTwoFactor = useAuthStore((s) => s.verifyTwoFactor);
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const dark = tone === 'dark';

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const { user } = await verifyTwoFactor(code.trim());
      onVerified(user);
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message || 'Invalid code'
          : (err as Error).message || 'Invalid code',
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="flex items-center gap-2">
        <ShieldCheck className={`h-5 w-5 ${dark ? 'text-violet-400' : 'text-primary'}`} />
        <p className={`text-sm font-medium ${dark ? 'text-white' : 'text-text'}`}>
          Two-factor authentication
        </p>
      </div>
      <p className={`text-sm ${dark ? 'text-gray-400' : 'text-text-secondary'}`}>
        Enter the 6-digit code from your authenticator app, or a backup code.
      </p>

      {error && (
        <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}

      <div>
        <label
          htmlFor="twofa-code"
          className={`mb-1.5 block text-sm font-medium ${dark ? 'text-gray-300' : 'text-text'}`}
        >
          Authentication code
        </label>
        <input
          id="twofa-code"
          inputMode="numeric"
          autoComplete="one-time-code"
          autoFocus
          required
          value={code}
          onChange={(e) => setCode(e.target.value)}
          placeholder="123456"
          className={`w-full rounded-lg border px-3.5 py-2.5 text-sm tracking-widest focus:outline-none focus:ring-1 ${
            dark
              ? 'border-gray-700 bg-gray-900 text-white focus:border-violet-500 focus:ring-violet-500'
              : 'border-border bg-surface text-text focus:border-primary focus:ring-primary'
          }`}
        />
      </div>

      <button
        type="submit"
        disabled={loading}
        className={`w-full rounded-lg px-4 py-2.5 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
          dark ? 'bg-violet-600 hover:bg-violet-500' : 'bg-primary hover:bg-primary-dark'
        }`}
      >
        {loading ? 'Verifying…' : 'Verify'}
      </button>
    </form>
  );
}
