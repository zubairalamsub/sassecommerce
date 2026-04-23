import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: {
    default: 'Admin Dashboard',
    template: '%s | Admin - Saajan',
  },
};

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
