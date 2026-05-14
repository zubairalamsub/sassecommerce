'use client';

import { useEffect, useState } from 'react';
import { Sparkles } from 'lucide-react';
import { recommendationApi } from '@/lib/api';
import type { StoreProduct } from '@/stores/products';
import { useProductStore } from '@/stores/products';
import ProductCard, { ProductCardSkeleton } from '@/components/store/product-card';
import { cn } from '@/lib/utils';

const TENANT_ID = 'tenant_saajan';
const TARGET = 6;

interface RelatedProductsProps {
  /** The current product — drives recommendations + same-category fallback. */
  product: StoreProduct;
  className?: string;
}

/**
 * "You might also like" — calls the recommendation service first;
 * if that returns nothing (or fails), falls back to other products
 * in the same category from the product store.
 */
export default function RelatedProducts({ product, className }: RelatedProductsProps) {
  const allProducts = useProductStore((s) => s.products);
  const [mounted, setMounted] = useState(false);
  const [related, setRelated] = useState<StoreProduct[] | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!product?.id) return;

    let cancelled = false;
    setLoading(true);

    (async () => {
      // 1. Try recommendation service
      try {
        const res = await recommendationApi.forProduct(product.id, TENANT_ID, TARGET);
        const recs = res?.recommendations ?? [];
        if (recs.length > 0) {
          const matched = recs
            .map((r) => allProducts.find((p) => p.id === r.product_id))
            .filter((p): p is StoreProduct => Boolean(p) && p!.id !== product.id);
          if (matched.length > 0) {
            if (!cancelled) {
              setRelated(matched.slice(0, TARGET));
              setLoading(false);
            }
            return;
          }
        }
      } catch {
        // fall through to category-based fallback
      }

      // 2. Fallback: same-category products
      const sameCategory = allProducts.filter(
        (p) =>
          p.id !== product.id &&
          p.category_id === product.category_id &&
          p.status === 'active',
      );
      if (!cancelled) {
        setRelated(sameCategory.slice(0, TARGET));
        setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
    // Re-run when the product or the product catalog changes so the
    // category fallback resolves correctly once the store has loaded.
  }, [product?.id, product?.category_id, allProducts]);

  if (!mounted) return null;

  // Empty: don't render anything — keeps the PDP tidy when nothing relevant exists.
  if (!loading && (!related || related.length === 0)) return null;

  return (
    <section
      className={cn('mt-12 border-t border-border pt-8 sm:mt-16 sm:pt-10', className)}
      aria-labelledby="related-products-heading"
    >
      <div className="mb-4 flex items-center gap-2 sm:mb-6">
        <Sparkles className="h-5 w-5 text-accent" />
        <h2 id="related-products-heading" className="text-xl font-bold text-text sm:text-2xl">
          You might also like
        </h2>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4">
        {loading
          ? Array.from({ length: 4 }).map((_, i) => <ProductCardSkeleton key={i} />)
          : related!.map((p, idx) => <ProductCard key={p.id} product={p} index={idx} />)}
      </div>
    </section>
  );
}
