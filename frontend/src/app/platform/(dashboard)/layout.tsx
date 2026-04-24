'use client';

import PlatformSidebar from '@/components/platform/sidebar';
import AuthGuard from '@/components/auth/auth-guard';

export default function PlatformLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard requiredRole="super_admin" redirectTo="/platform/login">
      <div className="flex min-h-screen bg-surface-secondary">
        <PlatformSidebar />
        <main className="ml-64 flex-1 p-8">{children}</main>
      </div>
    </AuthGuard>
  );
}
