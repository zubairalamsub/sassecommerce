'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore, type UserRole } from '@/stores/auth';

interface AuthGuardProps {
  children: React.ReactNode;
  requiredRole: UserRole;
  redirectTo?: string;
}

export default function AuthGuard({ children, requiredRole, redirectTo }: AuthGuardProps) {
  const router = useRouter();
  const { isAuthenticated, hasRole, user } = useAuthStore();
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    if (!isAuthenticated()) {
      const loginPath = redirectTo || getLoginPath(requiredRole);
      router.replace(loginPath);
      return;
    }
    if (!hasRole(requiredRole)) {
      // Authenticated but wrong role — redirect to their appropriate area
      if (user?.role === 'super_admin') {
        router.replace('/platform/dashboard');
      } else if (user?.role === 'admin' || user?.role === 'moderator') {
        router.replace('/admin/dashboard');
      } else {
        router.replace('/products');
      }
      return;
    }
    setChecked(true);
  }, [isAuthenticated, hasRole, requiredRole, redirectTo, router, user]);

  if (!checked) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-primary" />
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
