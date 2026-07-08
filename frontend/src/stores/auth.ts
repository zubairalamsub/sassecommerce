'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authApi, ApiError } from '@/lib/api';
import { DEFAULT_TENANT_ID } from '@/lib/tenant';

// Role hierarchy: super_admin > admin > moderator > customer > guest
export type UserRole = 'super_admin' | 'admin' | 'moderator' | 'customer' | 'guest';

export interface AuthUser {
  id: string;
  tenant_id: string | null;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  phone: string | null;
  avatar: string | null;
  status: 'active' | 'inactive' | 'suspended';
  role: UserRole;
  email_verified: boolean;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}

// login()/verifyTwoFactor() resolve to either a completed session or a signal
// that a second factor is still required (A04-3).
export type LoginResult =
  | { user: AuthUser; token: string }
  | { twoFactorRequired: true };

interface AuthState {
  user: AuthUser | null;
  token: string | null;
  tenantId: string | null;
  setAuth: (user: AuthUser, token: string, tenantId: string | null) => void;
  login: (email: string, password: string, tenantId: string) => Promise<LoginResult>;
  verifyTwoFactor: (code: string) => Promise<{ user: AuthUser; token: string }>;
  register: (data: { email: string; username: string; password: string; first_name: string; last_name: string; phone?: string }, tenantId: string) => Promise<LoginResult>;
  logout: () => void;
  isAuthenticated: () => boolean;
  hasRole: (role: UserRole) => boolean;
  isSuperAdmin: () => boolean;
  isTenantAdmin: () => boolean;
  isStaff: () => boolean;
  isCustomer: () => boolean;
}

const ROLE_LEVEL: Record<UserRole, number> = {
  super_admin: 100,
  admin: 80,
  moderator: 60,
  customer: 40,
  guest: 0,
};

// B03: the real JWT lives in an HttpOnly cookie the browser can't read. The
// store only keeps this non-secret marker in `token` so existing UI auth-guards
// (`if (!token)`, `user && token`) keep working. It is NOT a credential and is
// never sent as one — the BFF proxy injects the real token from the cookie.
const SESSION_TOKEN = 'cookie';

// Normalises a user payload from a server auth route into an AuthUser.
function toAuthUser(raw: Record<string, unknown>, fallbackTenant: string | null): AuthUser {
  return {
    ...(raw as unknown as AuthUser),
    tenant_id: (raw.tenant_id as string) || fallbackTenant,
    role: raw.role as UserRole,
  };
}

// Establishes a session by POSTing credentials to the server login route, which
// calls the user-service and sets the JWT as an HttpOnly cookie. Returns the
// (non-secret) user, or a 2FA-required signal when the account is enrolled in
// two-factor auth. Throws ApiError on failure.
async function serverLogin(
  email: string,
  password: string,
  tenantId: string,
): Promise<{ user: AuthUser } | { twoFactorRequired: true }> {
  const res = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: JSON.stringify({ email, password, tenant_id: tenantId }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new ApiError(res.status, err.error || 'Login failed', err);
  }
  const data = await res.json();
  if (data.twoFactorRequired) return { twoFactorRequired: true };
  return { user: toAuthUser(data.user, tenantId) };
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      tenantId: null,
      setAuth: (user, token, tenantId) => set({ user, token, tenantId }),

      login: async (email, password, tenantId) => {
        try {
          const result = await serverLogin(email, password, tenantId);
          if ('twoFactorRequired' in result) return result;
          set({ user: result.user, token: SESSION_TOKEN, tenantId: result.user.tenant_id });
          return { user: result.user, token: SESSION_TOKEN };
        } catch {
          // Backend auth failed or unreachable — fall back to demo login
          // (also cookie-based via /api/auth/demo-token).
          const demo = await demoLogin(email, password);
          if (demo) {
            set({ user: demo.user, token: SESSION_TOKEN, tenantId: demo.user.tenant_id });
            return demo;
          }
          throw new Error('Invalid email or password');
        }
      },

      // Completes a 2FA-gated login: submits the code to the server route, which
      // validates it against the challenge cookie and sets the auth cookie.
      verifyTwoFactor: async (code) => {
        const res = await fetch('/api/auth/login/2fa', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'same-origin',
          body: JSON.stringify({ code }),
        });
        if (!res.ok) {
          const err = await res.json().catch(() => ({}));
          throw new ApiError(res.status, err.error || 'Invalid code', err);
        }
        const data = await res.json();
        const authUser = toAuthUser(data.user, get().tenantId ?? DEFAULT_TENANT_ID);
        set({ user: authUser, token: SESSION_TOKEN, tenantId: authUser.tenant_id });
        return { user: authUser, token: SESSION_TOKEN };
      },

      register: async (data, tenantId) => {
        try {
          await authApi.register({ ...data, tenant_id: tenantId }, tenantId);
          // After registration, establish the session (sets the HttpOnly cookie).
          const result = await serverLogin(data.email, data.password, tenantId);
          if ('twoFactorRequired' in result) return result;
          set({ user: result.user, token: SESSION_TOKEN, tenantId: result.user.tenant_id });
          return { user: result.user, token: SESSION_TOKEN };
        } catch (err) {
          // If API is unreachable, create a demo user
          if (err instanceof ApiError) throw err;
          const newUser: AuthUser = {
            id: 'new-' + Date.now(),
            tenant_id: tenantId,
            email: data.email,
            username: data.username,
            first_name: data.first_name,
            last_name: data.last_name,
            phone: data.phone || null,
            avatar: null,
            status: 'active',
            role: 'customer',
            email_verified: false,
            last_login_at: new Date().toISOString(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          };
          // Offline demo user — no server session/cookie exists for it.
          set({ user: newUser, token: SESSION_TOKEN, tenantId });
          return { user: newUser, token: SESSION_TOKEN };
        }
      },

      logout: () => {
        // Clear the HttpOnly cookie server-side (best effort); local state is
        // cleared regardless of the request outcome.
        void fetch('/api/auth/logout', { method: 'POST', credentials: 'same-origin' }).catch(() => {});
        set({ user: null, token: null, tenantId: null });
      },
      isAuthenticated: () => !!get().token,
      hasRole: (role) => {
        const user = get().user;
        if (!user) return false;
        return ROLE_LEVEL[user.role] >= ROLE_LEVEL[role];
      },
      isSuperAdmin: () => get().user?.role === 'super_admin',
      isTenantAdmin: () => {
        const role = get().user?.role;
        return role === 'admin' || role === 'super_admin';
      },
      isStaff: () => {
        const level = ROLE_LEVEL[get().user?.role || 'guest'];
        return level >= ROLE_LEVEL.moderator;
      },
      isCustomer: () => get().user?.role === 'customer',
    }),
    {
      name: 'auth-storage',
      merge: (persisted, current) => {
        const merged = { ...current, ...(persisted as object) } as AuthState;
        // Post-B03 the only valid persisted token is the cookie-session marker.
        // Anything else — real JWTs or demo strings left in localStorage by
        // older builds — is stale and, more importantly, must not be treated as
        // a live session, since the real credential now lives in the cookie.
        if (merged.token && merged.token !== SESSION_TOKEN) {
          merged.token = null;
          merged.user = null;
          merged.tenantId = null;
        }
        return merged;
      },
    },
  ),
);

// Demo users for testing without backend
export const DEMO_USERS: Record<string, { password: string; user: AuthUser; token: string }> = {
  'super@saajan.com.bd': {
    password: 'super123',
    token: 'demo-super-token',
    user: {
      id: 'su-001',
      tenant_id: null,
      email: 'super@saajan.com.bd',
      username: 'superadmin',
      first_name: 'Platform',
      last_name: 'Admin',
      phone: '+8801700000000',
      avatar: null,
      status: 'active',
      role: 'super_admin',
      email_verified: true,
      last_login_at: new Date().toISOString(),
      created_at: '2025-01-01T00:00:00Z',
      updated_at: new Date().toISOString(),
    },
  },
  'admin@fashion.com.bd': {
    password: 'admin123',
    token: 'demo-admin-token-t1',
    user: {
      id: 'ta-001',
      tenant_id: DEFAULT_TENANT_ID,
      email: 'admin@fashion.com.bd',
      username: 'fashion_admin',
      first_name: 'Karim',
      last_name: 'Rahman',
      phone: '+8801712345678',
      avatar: null,
      status: 'active',
      role: 'admin',
      email_verified: true,
      last_login_at: new Date().toISOString(),
      created_at: '2025-06-01T00:00:00Z',
      updated_at: new Date().toISOString(),
    },
  },
  'staff@fashion.com.bd': {
    password: 'staff123',
    token: 'demo-mod-token-t1',
    user: {
      id: 'tm-001',
      tenant_id: DEFAULT_TENANT_ID,
      email: 'staff@fashion.com.bd',
      username: 'fashion_staff',
      first_name: 'Nusrat',
      last_name: 'Jahan',
      phone: '+8801812345678',
      avatar: null,
      status: 'active',
      role: 'moderator',
      email_verified: true,
      last_login_at: new Date().toISOString(),
      created_at: '2025-09-01T00:00:00Z',
      updated_at: new Date().toISOString(),
    },
  },
  'rahim@example.com': {
    password: 'customer123',
    token: 'demo-customer-token',
    user: {
      id: 'cu-001',
      tenant_id: DEFAULT_TENANT_ID,
      email: 'rahim@example.com',
      username: 'rahim_ahmed',
      first_name: 'Rahim',
      last_name: 'Ahmed',
      phone: '+8801912345678',
      avatar: null,
      status: 'active',
      role: 'customer',
      email_verified: true,
      last_login_at: new Date().toISOString(),
      created_at: '2026-01-15T00:00:00Z',
      updated_at: new Date().toISOString(),
    },
  },
};

export async function demoLogin(email: string, password: string): Promise<{ user: AuthUser; token: string } | null> {
  const entry = DEMO_USERS[email];
  if (!entry || entry.password !== password) return null;

  // The server route re-validates the credentials against its own allowlist,
  // derives the JWT claims (user id, tenant, role) server-side, and sets the
  // JWT as an HttpOnly cookie (never returned to JS). Disabled in production.
  const res = await fetch('/api/auth/demo-token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) return null;
  return { user: entry.user, token: SESSION_TOKEN };
}
