'use client';

import { useState, useEffect, useMemo, Suspense } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { motion } from 'framer-motion';
import { ShoppingCart, Tag, Loader2, Check, Heart, Search, X } from 'lucide-react';
import { useCartStore } from '@/stores/cart';
import { useProductStore, type StoreProduct } from '@/stores/products';
import { useAuthStore } from '@/stores/auth';
import { useWishlistStore } from '@/stores/wishlist';
import { formatCurrency, cn, mediaUrl } from '@/lib/utils';

const TENANT_ID = 'tenant_saajan';

const gradients = [
  'from-rose-100 to-pink-200 dark:from-rose-900/40 dark:to-pink-900/40',
  'from-blue-100 to-indigo-200 dark:from-blue-900/40 dark:to-indigo-900/40',
  'from-emerald-100 to-teal-200 dark:from-emerald-900/40 dark:to-teal-900/40',
  'from-amber-100 to-orange-200 dark:from-amber-900/40 dark:to-orange-900/40',
  'from-violet-100 to-purple-200 dark:from-violet-900/40 dark:to-purple-900/40',
  'from-cyan-100 to-sky-200 dark:from-cyan-900/40 dark:to-sky-900/40',
  'from-lime-100 to-green-200 dark:from-lime-900/40 dark:to-green-900/40',
  'from-fuchsia-100 to-pink-200 dark:from-fuchsia-900/40 dark:to-pink-900/40',
];

const stagger = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.06 } },
};

const cardAnim = {
  hidden: { opacity: 0, y: 20, scale: 0.97 },
  visible: { opacity: 1, y: 0, scale: 1, transition: { duration: 0.35, ease: [0, 0, 0.2, 1] as const } },
};

export default function ProductsPage() {
  return (
    <Suspense fallback={<div className="py-16 text-center"><Loader2 className="mx-auto h-8 w-8 animate-spin text-primary" /><p className="mt-3 text-text-muted">Loading products...</p></div>}>
      <ProductsContent />
    </Suspense>
  );
}

function ProductsContent() {
  const searchParams = useSearchParams();
  const { products, categories, loading, fetchProducts, fetchCategories } = useProductStore();
  const addItem = useCartStore((s) => s.addItem);
  const { toggleItem, isInWishlist } = useWishlistStore();
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);
  const [addedId, setAddedId] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  const auth = user && token ? { userId: user.id, tenantId: 'tenant_saajan', token } : undefined;

  const searchParam = searchParams.get('search') || '';
  const categoryParam = searchParams.get('category') || '';

  useEffect(() => { setMounted(true); }, []);
  useEffect(() => { fetchProducts(TENANT_ID); fetchCategories(TENANT_ID); }, [fetchProducts, fetchCategories]);

  const activeProducts = products.filter((p) => p.status === 'active');
  const activeCategories = categories.filter((c) => c.status === 'active');

  const filtered = useMemo(() => {
    let list = activeProducts;
    if (categoryParam) {
      const cat = activeCategories.find((c) => c.slug === categoryParam);
      if (cat) list = list.filter((p) => p.category_id === cat.id);
    }
    if (searchParam) {
      const q = searchParam.toLowerCase();
      list = list.filter((p) =>
        p.name.toLowerCase().includes(q) ||
        p.description?.toLowerCase().includes(q) ||
        p.tags?.some((t) => t.toLowerCase().includes(q)),
      );
    }
    return list;
  }, [activeProducts, activeCategories, categoryParam, searchParam]);

  const activeCatName = activeCategories.find((c) => c.slug === categoryParam)?.name;

  function handleAddToCart(product: StoreProduct) {
    addItem({
      productId: product.id,
      name: product.name,
      sku: product.sku,
      price: product.price,
      quantity: 1,
      image: product.images?.[0],
    }, auth);
    setAddedId(product.id);
    setTimeout(() => setAddedId(null), 1500);
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-text">
          {searchParam ? `Search: "${searchParam}"` : activeCatName || 'All Products'}
        </h1>
        <p className="mt-1 text-text-secondary">
          {filtered.length} product{filtered.length !== 1 ? 's' : ''} found
        </p>
      </div>

      {/* Category filters */}
      {!searchParam && activeCategories.length > 0 && (
        <div className="mb-6 flex flex-wrap gap-2">
          <Link href="/products"
            className={cn(
              'rounded-full px-4 py-1.5 text-sm font-medium transition-colors',
              !categoryParam ? 'bg-primary text-white' : 'bg-surface-hover text-text-secondary hover:text-text',
            )}>
            All
          </Link>
          {activeCategories.map((cat) => (
            <Link key={cat.id} href={`/products?category=${cat.slug}`}
              className={cn(
                'rounded-full px-4 py-1.5 text-sm font-medium transition-colors',
                categoryParam === cat.slug ? 'bg-primary text-white' : 'bg-surface-hover text-text-secondary hover:text-text',
              )}>
              {cat.name}
            </Link>
          ))}
        </div>
      )}

      {/* Active filter badge */}
      {(searchParam || categoryParam) && (
        <div className="mb-6 flex items-center gap-2">
          {searchParam && (
            <Link href={categoryParam ? `/products?category=${categoryParam}` : '/products'}
              className="inline-flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
              <Search className="h-3 w-3" /> {searchParam} <X className="h-3 w-3" />
            </Link>
          )}
          {categoryParam && (
            <Link href={searchParam ? `/products?search=${searchParam}` : '/products'}
              className="inline-flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
              {activeCatName} <X className="h-3 w-3" />
            </Link>
          )}
        </div>
      )}

      {/* Content */}
      {loading && !mounted ? (
        <div className="py-16 text-center">
          <Loader2 className="mx-auto h-8 w-8 animate-spin text-primary" />
          <p className="mt-3 text-text-muted">Loading products...</p>
        </div>
      ) : filtered.length === 0 && !loading ? (
        <div className="rounded-2xl border border-border bg-surface py-16 text-center">
          <p className="text-text-secondary">No products found.</p>
          {(searchParam || categoryParam) && (
            <Link href="/products" className="mt-3 inline-block text-sm font-medium text-primary hover:text-primary-dark">
              Clear filters
            </Link>
          )}
        </div>
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="visible" className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {filtered.map((product, index) => {
            const discount = product.compare_at_price && product.compare_at_price > product.price
              ? Math.round(((product.compare_at_price - product.price) / product.compare_at_price) * 100)
              : null;
            const isAdded = addedId === product.id;
            return (
              <motion.div key={product.id} variants={cardAnim}
                className="group rounded-2xl border border-border bg-surface overflow-hidden transition-all duration-300 hover:shadow-lg hover:-translate-y-1">
                <Link href={`/products/${product.slug}`}>
                  <div className={cn('relative h-52 bg-gradient-to-br flex items-center justify-center overflow-hidden', gradients[index % gradients.length])}>
                    {product.images && product.images.length > 0 ? (
                      <img src={mediaUrl(product.images[0])} alt={product.name} className="h-full w-full object-cover group-hover:scale-110 transition-transform duration-500" />
                    ) : (
                      <span className="text-5xl font-bold text-white/30 group-hover:scale-110 transition-transform duration-500">{product.name[0]}</span>
                    )}
                    <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors duration-300" />
                    {discount && (
                      <span className="absolute top-3 left-3 flex items-center gap-1 rounded-full bg-accent px-2.5 py-0.5 text-xs font-bold text-white">
                        <Tag className="h-3 w-3" /> {discount}% OFF
                      </span>
                    )}
                    <div className={cn('absolute top-3 right-3 transition-all duration-300', mounted && isInWishlist(product.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100')}>
                      <button
                        onClick={(e) => { e.preventDefault(); toggleItem({ productId: product.id, name: product.name, slug: product.slug, price: product.price, image: mediaUrl(product.images?.[0]) }); }}
                        className="flex h-8 w-8 items-center justify-center rounded-full bg-white/90 dark:bg-gray-800/90 text-text-secondary hover:text-accent transition-colors shadow-md">
                        <Heart className={cn('h-4 w-4', mounted && isInWishlist(product.id) && 'fill-accent text-accent')} />
                      </button>
                    </div>
                  </div>
                </Link>
                <div className="p-4">
                  <Link href={`/products/${product.slug}`}>
                    <h3 className="text-sm font-semibold text-text line-clamp-1 group-hover:text-primary transition-colors">{product.name}</h3>
                  </Link>
                  <div className="mt-2 flex items-baseline gap-2">
                    <span className="text-lg font-bold text-text">{formatCurrency(product.price)}</span>
                    {product.compare_at_price ? (
                      <span className="text-xs text-text-muted line-through">{formatCurrency(product.compare_at_price)}</span>
                    ) : null}
                  </div>
                  {product.tags && product.tags.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                      {product.tags.slice(0, 3).map((tag) => (
                        <span key={tag} className="rounded-full bg-surface-hover px-2 py-0.5 text-[10px] text-text-muted">{tag}</span>
                      ))}
                    </div>
                  )}
                  <motion.button
                    onClick={() => handleAddToCart(product)}
                    whileTap={{ scale: 0.95 }}
                    disabled={isAdded}
                    className={cn(
                      'mt-3 w-full flex items-center justify-center gap-2 rounded-xl py-2.5 text-xs font-semibold transition-all duration-300',
                      isAdded ? 'bg-green-500 text-white' : 'bg-primary text-white hover:bg-primary-dark',
                    )}>
                    {isAdded ? <><Check className="h-3.5 w-3.5" /> Added!</> : <><ShoppingCart className="h-3.5 w-3.5" /> Add to Cart</>}
                  </motion.button>
                </div>
              </motion.div>
            );
          })}
        </motion.div>
      )}
    </div>
  );
}
