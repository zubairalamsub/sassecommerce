import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: {
    default: 'Platform Admin',
    template: '%s | Platform Admin - Saajan',
  },
};

export default function SuperAdminLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
