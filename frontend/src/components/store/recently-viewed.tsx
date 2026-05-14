'use client';

import { useEffect, useState } from 'react';
import { Clock } from 'lucide-react';
import { useRecentlyViewedStore } from '@/stores/recently-viewed';
import { useProductStore } from '@/stores/products';
import ProductCard, { ProductCardSkeleton } from '@/components/store/product-card';
import { cn } from '@/lib/utils';

const TENANT_ID = 'tenant_saajan';

interface RecentlyViewedProps {
  /** Exclude a product ID from the strip (typically the current PDP). */
  excludeProductId?: string;
  /** Max items to show. Defaults to 6 (matches desktop grid width). */
  limit?: number;
  /** Section heading. Defaults to "Recently viewed". */
  title?: string;
  /** Optional wrapper className. */
  className?: string;
}

/**
 * Horizontal-scroll on mobile, up-to-6-wide grid on desktop strip
 * of products the visitor has recently viewed. Renders nothing
 * when the store is empty (or only contains the excluded ID).
 */
export default function RecentlyViewed({
  excludeProductId,
  limit = 6,
  title = 'Recently viewed',
  className,
}: RecentlyViewedProps) {
  const [mounted, setMounted] = useState(false);
  const ids = useRecentlyViewedStore((s) => s.ids);
  const { products, fetchProducts } = useProductStore();

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && ids.length > 0 && products.length === 0) {
      fetchProducts(TENANT_ID);
    }
  }, [mounted, ids.length, products.length, fetchProducts]);

  if (!mounted) return null;

  const candidateIds = (excludeProductId
    ? ids.filter((id) => id !== excludeProductId)
    : ids
  ).slice(0, limit);

  if (candidateIds.length === 0) return null;

  // Preserve ordering from the recently-viewed store.
  const resolved = candidateIds
    .map((id) => products.find((p) => p.id === id))
    .filter((p): p is NonNullable<typeof p> => Boolean(p));

  // If product store hasn't hydrated yet, show skeletons so the section
  // does not pop in late.
  const stillLoading = products.length === 0 && candidateIds.length > 0;

  if (!stillLoading && resolved.length === 0) {
    // We have IDs but none of them match the product store (deleted/archived).
    return null;
  }

  return (
    <section
      className={cn('mt-12 border-t border-border pt-8 sm:mt-16 sm:pt-10', className)}
      aria-labelledby="recently-viewed-heading"
    >
      <div className="mb-4 flex items-center gap-2 sm:mb-6">
        <Clock className="h-5 w-5 text-primary" />
        <h2 id="recently-viewed-heading" className="text-xl font-bold text-text sm:text-2xl">
          {title}
        </h2>
      </div>

      {/* Mobile: horizontal scroller */}
      <div className="-mx-4 flex snap-x snap-mandatory gap-3 overflow-x-auto px-4 pb-2 sm:hidden">
        {stillLoading
          ? Array.from({ length: Math.min(3, candidateIds.length) }).map((_, i) => (
              <div key={i} className="w-44 flex-shrink-0 snap-start">
                <ProductCardSkeleton />
              </div>
            ))
          : resolved.map((product, idx) => (
              <div key={product.id} className="w-44 flex-shrink-0 snap-start">
                <ProductCard product={product} index={idx} hideQuickView />
              </div>
            ))}
      </div>

      {/* Desktop: grid up to 6 wide */}
      <div
        className={cn(
          'hidden gap-4 sm:grid sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6',
        )}
      >
        {stillLoading
          ? Array.from({ length: Math.min(limit, candidateIds.length) }).map((_, i) => (
              <ProductCardSkeleton key={i} />
            ))
          : resolved.map((product, idx) => (
              <ProductCard key={product.id} product={product} index={idx} />
            ))}
      </div>
    </section>
  );
}
