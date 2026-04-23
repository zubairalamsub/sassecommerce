'use client';

import SuperAdminSidebar from '@/components/super-admin/sidebar';
import AuthGuard from '@/components/auth/auth-guard';

export default function SuperAdminPanelLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard requiredRole="super_admin">
      <div className="flex min-h-screen">
        <SuperAdminSidebar />
        <main className="ml-64 flex-1 p-8">{children}</main>
      </div>
    </AuthGuard>
  );
}
