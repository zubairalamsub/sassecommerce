import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Shopping Cart',
};

export default function CartLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
