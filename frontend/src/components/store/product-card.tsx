'use client';

import { useState, useEffect, type ReactNode } from 'react';
import Link from 'next/link';
import { motion, AnimatePresence } from 'framer-motion';
import {
  ShoppingBag,
  Heart,
  Star,
  Eye,
  Sparkles,
  X,
  Check,
  Loader2,
} from 'lucide-react';
import { useCartStore } from '@/stores/cart';
import { useAuthStore } from '@/stores/auth';
import { useWishlistStore } from '@/stores/wishlist';
import { useReviewStore } from '@/stores/reviews';
import { toast } from '@/stores/toast';
import { formatCurrency, cn, mediaUrl } from '@/lib/utils';

const TENANT_ID = 'tenant_saajan';
const NEW_DAYS = 14;

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

/**
 * Minimal product shape consumed by the card. Compatible with both
 * the full `Product` (StoreProduct) and the lighter `SearchProduct`.
 */
export interface ProductCardProduct {
  id: string;
  name: string;
  sku?: string;
  slug?: string;
  description?: string;
  price: number;
  compare_at_price?: number | null;
  images?: string[] | null;
  tags?: string[] | null;
  brand?: string;
  // Either the search-style flag or the variants stock can drive out-of-stock.
  in_stock?: boolean;
  stock_quantity?: number;
  variants?: { id?: string; sku: string; name: string; value?: string; price: number; stock?: number }[];
  created_at?: string;
}

export interface ProductCardProps {
  product: ProductCardProduct;
  /** Used for fallback gradient when the product has no image. */
  index?: number;
  className?: string;
  /**
   * Optional callback fired after the item is added to the cart. The card
   * still performs the cart mutation; this is a hook for parents that need
   * to flash other UI (e.g. animated cart badge).
   */
  onAddToCart?: (product: ProductCardProduct) => void;
  /** Hide quick-view button (e.g. inside an already-modal context). */
  hideQuickView?: boolean;
  /** Disable wishlist button on cards that already filter by wishlist. */
  hideWishlist?: boolean;
}

function daysSince(iso?: string): number {
  if (!iso) return Number.POSITIVE_INFINITY;
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return Number.POSITIVE_INFINITY;
  return (Date.now() - t) / (1000 * 60 * 60 * 24);
}

function isOutOfStock(p: ProductCardProduct): boolean {
  if (typeof p.in_stock === 'boolean') return !p.in_stock;
  if (typeof p.stock_quantity === 'number') return p.stock_quantity <= 0;
  if (p.variants && p.variants.length > 0) {
    return p.variants.every((v) => typeof v.stock === 'number' && v.stock <= 0);
  }
  return false;
}

export default function ProductCard({
  product,
  index = 0,
  className,
  onAddToCart,
  hideQuickView = false,
  hideWishlist = false,
}: ProductCardProps) {
  const addItem = useCartStore((s) => s.addItem);
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);
  const { toggleItem, isInWishlist } = useWishlistStore();
  const getAverageRating = useReviewStore((s) => s.getAverageRating);

  const [mounted, setMounted] = useState(false);
  const [added, setAdded] = useState(false);
  const [quickOpen, setQuickOpen] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const auth = user && token ? { userId: user.id, tenantId: TENANT_ID, token } : undefined;

  const detailHref = product.slug ? `/products/${product.slug}` : `/products/${product.id}`;
  const firstImage = product.images?.[0];
  const hasImage = Boolean(firstImage);
  const oos = isOutOfStock(product);
  const isNew = daysSince(product.created_at) <= NEW_DAYS;
  const discountPct =
    product.compare_at_price && product.compare_at_price > product.price
      ? Math.round(((product.compare_at_price - product.price) / product.compare_at_price) * 100)
      : null;
  const gradient = gradients[Math.abs(index) % gradients.length];
  const inWishlist = mounted && isInWishlist(product.id);

  // Rating summary — gracefully handles missing data (returns {average: 0, count: 0}).
  const { average, count } = mounted ? getAverageRating(product.id) : { average: 0, count: 0 };

  function handleAddToCart(e?: React.MouseEvent) {
    e?.preventDefault();
    e?.stopPropagation();
    if (oos || added) return;
    addItem(
      {
        productId: product.id,
        name: product.name,
        sku: product.sku ?? product.id,
        price: product.price,
        quantity: 1,
        image: firstImage,
      },
      auth,
    );
    setAdded(true);
    toast.success(`${product.name} added to cart`, { duration: 2000 });
    setTimeout(() => setAdded(false), 1500);
    onAddToCart?.(product);
  }

  function handleWishlist(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    const wasInWishlist = isInWishlist(product.id);
    toggleItem(
      {
        productId: product.id,
        name: product.name,
        slug: product.slug ?? product.id,
        price: product.price,
        image: mediaUrl(firstImage),
      },
      auth,
    );
    if (wasInWishlist) {
      toast.info('Removed from wishlist');
    } else {
      toast.success('Added to wishlist');
    }
  }

  function handleQuickView(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    setQuickOpen(true);
  }

  return (
    <>
      <div
        className={cn(
          'group relative flex flex-col overflow-hidden rounded-2xl border border-border bg-surface transition-all duration-300 hover:-translate-y-1 hover:shadow-lg',
          className,
        )}
      >
        {/* Image area */}
        <Link href={detailHref} className="block">
          <div
            className={cn(
              'relative aspect-[4/3] w-full overflow-hidden',
              hasImage ? 'bg-surface-secondary' : `bg-gradient-to-br ${gradient}`,
            )}
          >
            {hasImage ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={mediaUrl(firstImage)}
                alt={product.name}
                loading="lazy"
                className={cn(
                  'h-full w-full object-cover transition-transform duration-500 ease-out',
                  'group-hover:scale-105',
                  oos && 'opacity-70',
                )}
              />
            ) : (
              <span className="flex h-full w-full items-center justify-center text-5xl font-bold text-white/40">
                {product.name.charAt(0).toUpperCase()}
              </span>
            )}

            {/* Hover overlay */}
            <div className="pointer-events-none absolute inset-0 bg-black/0 transition-colors duration-300 group-hover:bg-black/5" />

            {/* Top-left badges */}
            <div className="absolute left-3 top-3 flex flex-col gap-1.5">
              {discountPct !== null && (
                <span className="inline-flex items-center rounded-full bg-accent px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white shadow-sm">
                  Sale {discountPct > 0 ? `-${discountPct}%` : ''}
                </span>
              )}
              {isNew && !oos && (
                <span className="inline-flex items-center gap-1 rounded-full bg-primary px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wide text-white shadow-sm">
                  <Sparkles className="h-2.5 w-2.5" />
                  New
                </span>
              )}
            </div>

            {/* Top-right: out-of-stock or wishlist */}
            <div className="absolute right-3 top-3 flex flex-col items-end gap-1.5">
              {oos && (
                <span className="inline-flex items-center rounded-full bg-text/85 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-white shadow-sm backdrop-blur-sm">
                  Out of stock
                </span>
              )}
              {!hideWishlist && (
                <button
                  type="button"
                  onClick={handleWishlist}
                  aria-label={inWishlist ? 'Remove from wishlist' : 'Add to wishlist'}
                  className={cn(
                    'flex h-8 w-8 items-center justify-center rounded-full shadow-md transition-all duration-200',
                    'bg-surface/90 backdrop-blur-sm',
                    inWishlist
                      ? 'text-accent'
                      : 'text-text-secondary opacity-0 group-hover:opacity-100 hover:text-accent',
                  )}
                >
                  <Heart className={cn('h-4 w-4', inWishlist && 'fill-accent')} />
                </button>
              )}
            </div>

            {/* Quick view overlay button — desktop hover only */}
            {!hideQuickView && (
              <button
                type="button"
                onClick={handleQuickView}
                className={cn(
                  'absolute bottom-3 left-3 hidden items-center gap-1.5 rounded-full bg-surface/90 px-3 py-1.5 text-xs font-medium text-text shadow-md backdrop-blur-sm transition-all duration-200',
                  'opacity-0 translate-y-1 group-hover:opacity-100 group-hover:translate-y-0',
                  'hover:bg-surface md:inline-flex',
                )}
              >
                <Eye className="h-3.5 w-3.5" />
                Quick view
              </button>
            )}
          </div>
        </Link>

        {/* Info */}
        <div className="flex flex-1 flex-col p-3 sm:p-4">
          {product.brand && (
            <p className="mb-0.5 text-[10px] font-semibold uppercase tracking-wider text-text-muted">
              {product.brand}
            </p>
          )}

          <Link href={detailHref} className="block">
            <h3 className="line-clamp-2 min-h-[2.5rem] text-sm font-semibold text-text transition-colors group-hover:text-primary">
              {product.name}
            </h3>
          </Link>

          {/* Rating */}
          {mounted && count > 0 && (
            <div className="mt-1.5 flex items-center gap-1">
              <div className="flex items-center">
                {[1, 2, 3, 4, 5].map((s) => (
                  <Star
                    key={s}
                    className={cn(
                      'h-3 w-3',
                      s <= Math.round(average) ? 'fill-amber-400 text-amber-400' : 'text-text-muted',
                    )}
                  />
                ))}
              </div>
              <span className="text-[11px] text-text-muted">
                {average.toFixed(1)} ({count})
              </span>
            </div>
          )}

          {/* Price + add to cart row */}
          <div className="mt-2 flex items-end justify-between gap-2">
            <div className="flex flex-col">
              <span className="text-base font-bold text-text">{formatCurrency(product.price)}</span>
              {product.compare_at_price && product.compare_at_price > product.price && (
                <span className="text-xs text-text-muted line-through">
                  {formatCurrency(product.compare_at_price)}
                </span>
              )}
            </div>

            <motion.button
              type="button"
              onClick={handleAddToCart}
              whileTap={{ scale: 0.92 }}
              disabled={oos || added}
              aria-label={oos ? 'Out of stock' : `Add ${product.name} to cart`}
              className={cn(
                'flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full shadow-sm transition-colors sm:h-9 sm:w-9',
                oos
                  ? 'cursor-not-allowed bg-surface-hover text-text-muted'
                  : added
                  ? 'bg-emerald-500 text-white'
                  : 'bg-primary text-white hover:bg-primary-dark',
              )}
            >
              {added ? <Check className="h-4 w-4" /> : <ShoppingBag className="h-4 w-4" />}
            </motion.button>
          </div>
        </div>
      </div>

      {/* Quick view modal */}
      <AnimatePresence>
        {quickOpen && (
          <QuickViewModal
            product={product}
            onClose={() => setQuickOpen(false)}
            onAddToCart={handleAddToCart}
            added={added}
            oos={oos}
            discountPct={discountPct}
            detailHref={detailHref}
            average={average}
            ratingCount={count}
            inWishlist={inWishlist}
            onToggleWishlist={(e) => handleWishlist(e)}
          />
        )}
      </AnimatePresence>
    </>
  );
}

/* ─────────────────────────── Quick View modal ─────────────────────────── */

function QuickViewModal({
  product,
  onClose,
  onAddToCart,
  added,
  oos,
  discountPct,
  detailHref,
  average,
  ratingCount,
  inWishlist,
  onToggleWishlist,
}: {
  product: ProductCardProduct;
  onClose: () => void;
  onAddToCart: (e?: React.MouseEvent) => void;
  added: boolean;
  oos: boolean;
  discountPct: number | null;
  detailHref: string;
  average: number;
  ratingCount: number;
  inWishlist: boolean;
  onToggleWishlist: (e: React.MouseEvent) => void;
}) {
  const [selectedImage, setSelectedImage] = useState(0);
  const [selectedVariant, setSelectedVariant] = useState(0);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    document.addEventListener('keydown', onKey);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', onKey);
      document.body.style.overflow = '';
    };
  }, [onClose]);

  const images = product.images?.length ? product.images : null;
  const hasVariants = (product.variants?.length ?? 0) > 0;
  const variant = hasVariants ? product.variants![selectedVariant] : null;
  const activePrice = variant ? variant.price : product.price;

  return (
    <motion.div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={`Quick view: ${product.name}`}
    >
      <motion.div
        className="relative w-full max-w-3xl overflow-hidden rounded-2xl border border-border bg-surface shadow-2xl"
        initial={{ opacity: 0, scale: 0.95, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.95, y: 20 }}
        transition={{ duration: 0.2 }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close */}
        <button
          type="button"
          onClick={onClose}
          aria-label="Close quick view"
          className="absolute right-3 top-3 z-10 flex h-9 w-9 items-center justify-center rounded-full bg-surface/90 text-text-secondary shadow-md backdrop-blur-sm hover:text-text"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="grid max-h-[85vh] grid-cols-1 overflow-y-auto md:grid-cols-2">
          {/* Image side */}
          <div className="bg-surface-secondary">
            <div className="relative aspect-square w-full overflow-hidden">
              {images ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={mediaUrl(images[selectedImage])}
                  alt={product.name}
                  className="h-full w-full object-cover"
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-surface-hover to-surface-secondary">
                  <span className="text-7xl font-bold text-text-muted/50">
                    {product.name.charAt(0).toUpperCase()}
                  </span>
                </div>
              )}
            </div>
            {images && images.length > 1 && (
              <div className="flex gap-2 overflow-x-auto p-3">
                {images.map((img, i) => (
                  <button
                    key={i}
                    type="button"
                    onClick={() => setSelectedImage(i)}
                    aria-label={`View image ${i + 1}`}
                    className={cn(
                      'h-14 w-14 flex-shrink-0 overflow-hidden rounded-lg border-2 transition-colors',
                      selectedImage === i ? 'border-primary' : 'border-border hover:border-primary/50',
                    )}
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={mediaUrl(img)} alt="" className="h-full w-full object-cover" />
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Info side */}
          <div className="flex flex-col p-6">
            {product.brand && (
              <p className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-text-muted">
                {product.brand}
              </p>
            )}
            <h2 className="text-xl font-bold text-text">{product.name}</h2>

            {ratingCount > 0 && (
              <div className="mt-2 flex items-center gap-1">
                <div className="flex">
                  {[1, 2, 3, 4, 5].map((s) => (
                    <Star
                      key={s}
                      className={cn(
                        'h-3.5 w-3.5',
                        s <= Math.round(average)
                          ? 'fill-amber-400 text-amber-400'
                          : 'text-text-muted',
                      )}
                    />
                  ))}
                </div>
                <span className="text-xs text-text-muted">
                  {average.toFixed(1)} ({ratingCount} review{ratingCount !== 1 ? 's' : ''})
                </span>
              </div>
            )}

            <div className="mt-3 flex items-baseline gap-3">
              <span className="text-2xl font-bold text-text">{formatCurrency(activePrice)}</span>
              {product.compare_at_price && product.compare_at_price > product.price && (
                <>
                  <span className="text-sm text-text-muted line-through">
                    {formatCurrency(product.compare_at_price)}
                  </span>
                  {discountPct !== null && (
                    <span className="rounded-full bg-accent/10 px-2 py-0.5 text-xs font-semibold text-accent">
                      -{discountPct}%
                    </span>
                  )}
                </>
              )}
            </div>

            {product.description && (
              <p className="mt-4 line-clamp-5 text-sm leading-relaxed text-text-secondary">
                {product.description}
              </p>
            )}

            {/* Variants */}
            {hasVariants && (
              <div className="mt-4">
                <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-text-secondary">
                  {product.variants![0].name}
                </p>
                <div className="flex flex-wrap gap-2">
                  {product.variants!.map((v, i) => (
                    <button
                      key={i}
                      type="button"
                      onClick={() => setSelectedVariant(i)}
                      className={cn(
                        'rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors',
                        selectedVariant === i
                          ? 'border-primary bg-primary/10 text-primary'
                          : 'border-border text-text-secondary hover:border-primary/50',
                      )}
                    >
                      {v.value ?? v.name}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {product.tags && product.tags.length > 0 && (
              <div className="mt-4 flex flex-wrap gap-1.5">
                {product.tags.slice(0, 5).map((t) => (
                  <span
                    key={t}
                    className="rounded-full bg-surface-hover px-2 py-0.5 text-[10px] text-text-muted"
                  >
                    {t}
                  </span>
                ))}
              </div>
            )}

            <div className="mt-auto space-y-2 pt-6">
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={(e) => onAddToCart(e)}
                  disabled={oos || added}
                  className={cn(
                    'flex flex-1 items-center justify-center gap-2 rounded-lg py-2.5 text-sm font-semibold transition-colors',
                    oos
                      ? 'cursor-not-allowed bg-surface-hover text-text-muted'
                      : added
                      ? 'bg-emerald-500 text-white'
                      : 'bg-primary text-white hover:bg-primary-dark',
                  )}
                >
                  {oos ? (
                    'Out of stock'
                  ) : added ? (
                    <>
                      <Check className="h-4 w-4" /> Added
                    </>
                  ) : (
                    <>
                      <ShoppingBag className="h-4 w-4" /> Add to cart
                    </>
                  )}
                </button>
                <button
                  type="button"
                  onClick={onToggleWishlist}
                  aria-label={inWishlist ? 'Remove from wishlist' : 'Add to wishlist'}
                  className={cn(
                    'flex h-11 w-11 items-center justify-center rounded-lg border transition-colors',
                    inWishlist
                      ? 'border-accent/30 bg-accent/10 text-accent'
                      : 'border-border text-text-secondary hover:border-accent/50 hover:text-accent',
                  )}
                >
                  <Heart className={cn('h-4 w-4', inWishlist && 'fill-accent')} />
                </button>
              </div>
              <Link
                href={detailHref}
                onClick={onClose}
                className="block rounded-lg border border-border py-2.5 text-center text-sm font-medium text-text-secondary transition-colors hover:bg-surface-hover"
              >
                View full details
              </Link>
            </div>
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
}

/* ───────────────────────────── Skeleton ───────────────────────────── */

export function ProductCardSkeleton({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        'flex flex-col overflow-hidden rounded-2xl border border-border bg-surface',
        className,
      )}
    >
      <div className="skeleton aspect-[4/3] w-full" />
      <div className="space-y-2 p-4">
        <div className="skeleton h-3 w-1/3" />
        <div className="skeleton h-4 w-4/5" />
        <div className="skeleton h-4 w-2/3" />
        <div className="flex items-center justify-between pt-2">
          <div className="skeleton h-5 w-20" />
          <div className="skeleton h-9 w-9 rounded-full" />
        </div>
      </div>
    </div>
  );
}

/* ───────────────────────── Empty / no-results ───────────────────────── */

export function ProductsEmptyState({
  icon,
  title,
  message,
  action,
  className,
}: {
  icon?: ReactNode;
  title: string;
  message?: string;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center rounded-2xl border border-border bg-surface px-6 py-16 text-center',
        className,
      )}
    >
      <div className="mb-4 flex h-20 w-20 items-center justify-center rounded-full bg-surface-hover text-text-muted">
        {icon ?? <ShoppingBag className="h-10 w-10" />}
      </div>
      <h3 className="text-lg font-semibold text-text">{title}</h3>
      {message && <p className="mt-1 max-w-md text-sm text-text-secondary">{message}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}

/* Re-export the spinner-style loader for parity with caller pages. */
export { Loader2 };
