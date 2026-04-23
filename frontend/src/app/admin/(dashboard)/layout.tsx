'use client';

import AdminSidebar from '@/components/admin/sidebar';
import AuthGuard from '@/components/auth/auth-guard';

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard requiredRole="moderator">
      <div className="flex min-h-screen bg-surface-secondary">
        <AdminSidebar />
        <main className="ml-64 flex-1 p-8">{children}</main>
      </div>
    </AuthGuard>
  );
}
