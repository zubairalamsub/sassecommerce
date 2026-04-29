'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authApi, ApiError } from '@/lib/api';

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

interface AuthState {
  user: AuthUser | null;
  token: string | null;
  tenantId: string | null;
  setAuth: (user: AuthUser, token: string, tenantId: string | null) => void;
  login: (email: string, password: string, tenantId: string) => Promise<{ user: AuthUser; token: string }>;
  register: (data: { email: string; username: string; password: string; first_name: string; last_name: string; phone?: string }, tenantId: string) => Promise<{ user: AuthUser; token: string }>;
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

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      tenantId: null,
      setAuth: (user, token, tenantId) => set({ user, token, tenantId }),

      login: async (email, password, tenantId) => {
        try {
          const res = await authApi.login({ tenant_id: tenantId, email, password }, tenantId);
          const authUser: AuthUser = {
            ...res.user,
            tenant_id: res.user.tenant_id || tenantId,
            role: res.user.role as UserRole,
          };
          set({ user: authUser, token: res.token, tenantId: authUser.tenant_id });
          return { user: authUser, token: res.token };
        } catch {
          // Backend auth failed or unreachable — fall back to demo login
          const demo = await demoLogin(email, password);
          if (demo) {
            set({ user: demo.user, token: demo.token, tenantId: demo.user.tenant_id });
            return demo;
          }
          throw new Error('Invalid email or password');
        }
      },

      register: async (data, tenantId) => {
        try {
          const user = await authApi.register({ ...data, tenant_id: tenantId }, tenantId);
          // After registration, log in to get token
          const loginRes = await authApi.login({ tenant_id: tenantId, email: data.email, password: data.password }, tenantId);
          const authUser: AuthUser = {
            ...loginRes.user,
            tenant_id: loginRes.user.tenant_id || tenantId,
            role: loginRes.user.role as UserRole,
          };
          set({ user: authUser, token: loginRes.token, tenantId: authUser.tenant_id });
          return { user: authUser, token: loginRes.token };
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
          const token = 'demo-new-token-' + Date.now();
          set({ user: newUser, token, tenantId });
          return { user: newUser, token };
        }
      },

      logout: () => set({ user: null, token: null, tenantId: null }),
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
    { name: 'auth-storage' },
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
      tenant_id: 'tenant_saajan',
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
      tenant_id: 'tenant_saajan',
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
      tenant_id: 'tenant_saajan',
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

  try {
    const res = await fetch('/api/auth/demo-token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: entry.user.id,
        tenant_id: entry.user.tenant_id || '',
        email: entry.user.email,
        role: entry.user.role,
      }),
    });
    if (res.ok) {
      const data = await res.json();
      return { user: entry.user, token: data.token };
    }
  } catch {
    // API route unavailable — use fallback token
  }
  return { user: entry.user, token: entry.token };
}
