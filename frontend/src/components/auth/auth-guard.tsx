'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore, type UserRole } from '@/stores/auth';

interface AuthGuardProps {
  children: React.ReactNode;
  requiredRole: UserRole;
  redirectTo?: string;
}

type GuardStatus = 'checking' | 'redirecting' | 'ok';

export default function AuthGuard({ children, requiredRole, redirectTo }: AuthGuardProps) {
  const router = useRouter();
  const { isAuthenticated, hasRole, user } = useAuthStore();
  const [status, setStatus] = useState<GuardStatus>('checking');
  const [redirectLabel, setRedirectLabel] = useState('login');

  useEffect(() => {
    if (!isAuthenticated()) {
      const loginPath = redirectTo || getLoginPath(requiredRole);
      setRedirectLabel('login');
      setStatus('redirecting');
      router.replace(loginPath);
      return;
    }
    if (!hasRole(requiredRole)) {
      let path = '/products';
      let label = 'storefront';
      if (user?.role === 'super_admin') { path = '/platform/dashboard'; label = 'platform dashboard'; }
      else if (user?.role === 'admin' || user?.role === 'moderator') { path = '/admin/dashboard'; label = 'admin dashboard'; }
      setRedirectLabel(label);
      setStatus('redirecting');
      router.replace(path);
      return;
    }
    setStatus('ok');
  }, [isAuthenticated, hasRole, requiredRole, redirectTo, router, user]);

  if (status !== 'ok') {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-primary" />
        <p className="text-sm text-gray-500">
          {status === 'redirecting' ? `Redirecting to ${redirectLabel}…` : 'Checking permissions…'}
        </p>
      </div>
    );
  }

  return <>{children}</>;
}

function getLoginPath(role: UserRole): string {
  switch (role) {
    case 'super_admin':
      return '/platform/login';
    case 'admin':
    case 'moderator':
      return '/admin/login';
    default:
      return '/login';
  }
}
