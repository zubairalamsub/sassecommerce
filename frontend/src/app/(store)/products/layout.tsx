import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Products',
  description: 'Browse our collection of premium fashion and lifestyle products from Bangladesh.',
};

export default function ProductsLayout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
