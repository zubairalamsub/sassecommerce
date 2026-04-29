'use client';

import { use, useState, useEffect } from 'react';
import Link from 'next/link';
import {
  ShoppingCart,
  Minus,
  Plus,
  ArrowLeft,
  Star,
  Truck,
  Shield,
  RotateCcw,
  Loader2,
} from 'lucide-react';
import { useCartStore } from '@/stores/cart';
import { useProductStore } from '@/stores/products';
import { useReviewStore } from '@/stores/reviews';
import { useAuthStore } from '@/stores/auth';
import { useDeliveryProfileStore } from '@/stores/delivery-profiles';
import { formatCurrency, cn, mediaUrl } from '@/lib/utils';
import { recommendationApi, productApi, type ProductRecommendation } from '@/lib/api';
import type { StoreProduct } from '@/stores/products';

const TENANT_ID = 'tenant_saajan';

const gradients = [
  'from-pink-200 to-rose-100',
  'from-blue-200 to-sky-100',
  'from-amber-200 to-yellow-100',
  'from-indigo-200 to-violet-100',
  'from-orange-200 to-amber-100',
  'from-emerald-200 to-teal-100',
  'from-yellow-200 to-orange-100',
  'from-red-200 to-pink-100',
  'from-lime-200 to-green-100',
  'from-purple-200 to-fuchsia-100',
];

export default function ProductDetailPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = use(params);
  const { products, loading, fetchProducts } = useProductStore();
  const addItem = useCartStore((s) => s.addItem);
  const { addReview, fetchProductReviews, fetchSummary, getProductReviews, getAverageRating } = useReviewStore();
  const { user, token, isAuthenticated } = useAuthStore();
  const [quantity, setQuantity] = useState(1);
  const [selectedVariantIndex, setSelectedVariantIndex] = useState(0);
  const [added, setAdded] = useState(false);
  const [showReviewForm, setShowReviewForm] = useState(false);
  const [reviewRating, setReviewRating] = useState(5);
  const [reviewTitle, setReviewTitle] = useState('');
  const [reviewComment, setReviewComment] = useState('');
  const [hoverRating, setHoverRating] = useState(0);
  const [mounted, setMounted] = useState(false);
  const [recommendations, setRecommendations] = useState<ProductRecommendation[]>([]);
  const [directProduct, setDirectProduct] = useState<StoreProduct | null>(null);
  const [directLoading, setDirectLoading] = useState(false);
  const [selectedImageIndex, setSelectedImageIndex] = useState(0);

  useEffect(() => { setMounted(true); }, []);

  useEffect(() => {
    if (products.length === 0) {
      fetchProducts(TENANT_ID);
    }
  }, [products.length, fetchProducts]);

  const storeProduct = products.find((p) => p.slug === slug || p.id === slug);
  const product = storeProduct ?? directProduct;

  // If product not in store (e.g. navigated by ID from search), fetch directly
  useEffect(() => {
    if (!storeProduct && !loading && slug) {
      setDirectLoading(true);
      productApi.get(slug, TENANT_ID)
        .then((p) => setDirectProduct(p as unknown as StoreProduct))
        .catch(() => {})
        .finally(() => setDirectLoading(false));
    }
  }, [storeProduct, loading, slug]);

  // Fetch reviews and recommendations from backend when product is available
  useEffect(() => {
    if (!product) return;
    fetchProductReviews(product.id, TENANT_ID);
    fetchSummary(product.id, TENANT_ID);
    recommendationApi
      .forProduct(product.id, TENANT_ID, 6)
      .then((res) => setRecommendations(res.recommendations || []))
      .catch(() => {});
  }, [product?.id, fetchProductReviews, fetchSummary]);

  if ((loading && products.length === 0) || directLoading) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-16 text-center sm:px-6 lg:px-8">
        <Loader2 className="mx-auto h-8 w-8 animate-spin text-primary" />
        <p className="mt-3 text-gray-500">Loading...</p>
      </div>
    );
  }

  if (!product) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-16 text-center sm:px-6 lg:px-8">
        <h1 className="text-2xl font-bold text-gray-900">Product Not Found</h1>
        <p className="mt-2 text-gray-500">The product you are looking for does not exist.</p>
        <Link
          href="/products"
          className="mt-6 inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Products
        </Link>
      </div>
    );
  }

  const productIndex = products.indexOf(product);
  const gradient = gradients[Math.abs(productIndex) % gradients.length];
  const hasImages = product.images && product.images.length > 0;
  const selectedVariant =
    product.variants && product.variants.length > 0 ? product.variants[selectedVariantIndex] : null;
  const activePrice = selectedVariant ? selectedVariant.price : product.price;
  const dpProfiles = useDeliveryProfileStore((s) => s.profiles);
  const dpDefault = useDeliveryProfileStore((s) => s.getDefaultProfile);
  const deliveryProfile = product.delivery_profile_id
    ? dpProfiles.find((p) => p.id === product.delivery_profile_id) || dpDefault()
    : dpDefault();

  const auth = user && token ? { userId: user.id, tenantId: TENANT_ID, token } : undefined;

  function handleAddToCart() {
    if (!product) return;
    addItem(
      {
        productId: product.id,
        variantId: selectedVariant?.id,
        name: selectedVariant
          ? `${product.name} - ${selectedVariant.value}`
          : product.name,
        sku: selectedVariant?.sku ?? product.sku,
        price: activePrice,
        quantity,
        image: product.images?.[0],
      },
      auth,
    );
    setAdded(true);
    setTimeout(() => setAdded(false), 2000);
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Breadcrumb */}
      <nav className="mb-6 flex items-center gap-2 text-sm text-gray-500">
        <Link href="/products" className="hover:text-primary transition-colors">
          Products
        </Link>
        <span>/</span>
        <span className="text-gray-900">{product.name}</span>
      </nav>

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-2">
        {/* Image area */}
        <div className="space-y-3">
          <div className={cn('flex aspect-square items-center justify-center rounded-2xl overflow-hidden', hasImages ? 'bg-surface-secondary' : `bg-gradient-to-br ${gradient}`)}>
            {hasImages ? (
              <img src={mediaUrl(product.images![selectedImageIndex])} alt={product.name} className="h-full w-full object-cover" />
            ) : (
              <span className="text-[10rem] font-bold text-white/40">{product.name.charAt(0)}</span>
            )}
          </div>
          {hasImages && product.images!.length > 1 && (
            <div className="flex gap-2 overflow-x-auto pb-1">
              {product.images!.map((img, i) => (
                <button key={i} onClick={() => setSelectedImageIndex(i)}
                  className={cn('h-16 w-16 flex-shrink-0 rounded-lg overflow-hidden border-2 transition-colors', selectedImageIndex === i ? 'border-primary' : 'border-border hover:border-primary/50')}>
                  <img src={mediaUrl(img)} alt={`${product.name} ${i + 1}`} className="h-full w-full object-cover" />
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Product info */}
        <div>
          <h1 className="text-3xl font-bold text-gray-900">{product.name}</h1>

          {/* Price */}
          <div className="mt-4 flex items-baseline gap-3">
            <span className="text-2xl font-bold text-gray-900">
              {formatCurrency(activePrice)}
            </span>
            {product.compare_at_price && (
              <span className="text-lg text-gray-400 line-through">
                {formatCurrency(product.compare_at_price)}
              </span>
            )}
            {product.compare_at_price && product.compare_at_price > product.price && (
              <span className="rounded-full bg-accent/10 px-2.5 py-0.5 text-sm font-semibold text-accent">
                {Math.round(
                  ((product.compare_at_price - product.price) /
                    product.compare_at_price) *
                    100,
                )}
                % OFF
              </span>
            )}
          </div>

          {/* Description */}
          <p className="mt-6 leading-relaxed text-gray-600">
            {product.description || 'No description available.'}
          </p>

          {/* Variant selector */}
          {product.variants && product.variants.length > 0 && (
            <div className="mt-6">
              <label className="mb-2 block text-sm font-medium text-gray-700">
                {product.variants[0].name}
              </label>
              <div className="flex flex-wrap gap-2">
                {product.variants.map((variant, i) => (
                  <button
                    key={i}
                    onClick={() => setSelectedVariantIndex(i)}
                    className={`rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${
                      selectedVariantIndex === i
                        ? 'border-primary bg-primary-light text-primary'
                        : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300'
                    }`}
                  >
                    {variant.value}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Quantity selector */}
          <div className="mt-6">
            <label className="mb-2 block text-sm font-medium text-gray-700">
              Quantity
            </label>
            <div className="inline-flex items-center rounded-lg border border-gray-200">
              <button
                onClick={() => setQuantity((q) => Math.max(1, q - 1))}
                className="flex h-10 w-10 items-center justify-center text-gray-600 transition-colors hover:bg-gray-50"
              >
                <Minus className="h-4 w-4" />
              </button>
              <span className="flex h-10 w-12 items-center justify-center border-x border-gray-200 text-sm font-medium">
                {quantity}
              </span>
              <button
                onClick={() => setQuantity((q) => q + 1)}
                className="flex h-10 w-10 items-center justify-center text-gray-600 transition-colors hover:bg-gray-50"
              >
                <Plus className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Add to Cart */}
          <button
            onClick={handleAddToCart}
            disabled={added}
            className="mt-8 flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-6 py-3 font-medium text-white transition-colors hover:bg-primary-dark disabled:opacity-60 sm:w-auto"
          >
            <ShoppingCart className="h-5 w-5" />
            {added ? 'Added to Cart!' : 'Add to Cart'}
          </button>

          {/* Tags */}
          {product.tags && product.tags.length > 0 && (
            <div className="mt-6 flex flex-wrap gap-2">
              {product.tags.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600"
                >
                  {tag}
                </span>
              ))}
            </div>
          )}

          {/* Delivery, Return & Warranty */}
          <div className="mt-8 space-y-0 divide-y divide-gray-100 rounded-xl border border-gray-200 bg-gray-50">
            {/* Delivery */}
            <div className="flex gap-3 p-4">
              <Truck className="mt-0.5 h-5 w-5 flex-shrink-0 text-primary" />
              <div>
                <p className="text-sm font-medium text-gray-900">Delivery</p>
                <p className="mt-0.5 text-xs text-gray-600">
                  Inside Dhaka: {deliveryProfile.estimated_delivery_dhaka} ({deliveryProfile.inside_dhaka_rate === 0 ? 'Free' : `৳${deliveryProfile.inside_dhaka_rate}`})
                </p>
                <p className="text-xs text-gray-600">
                  Outside Dhaka: {deliveryProfile.estimated_delivery_outside} ({deliveryProfile.outside_dhaka_rate === 0 ? 'Free' : `৳${deliveryProfile.outside_dhaka_rate}`})
                </p>
              </div>
            </div>
            {/* Return Policy */}
            <div className="flex gap-3 p-4">
              <RotateCcw className="mt-0.5 h-5 w-5 flex-shrink-0 text-primary" />
              <div>
                <p className="text-sm font-medium text-gray-900">7-Day Easy Return</p>
                <p className="mt-0.5 text-xs text-gray-600">Return or exchange within 7 days of delivery. Item must be unused and in original packaging.</p>
              </div>
            </div>
            {/* Warranty */}
            <div className="flex gap-3 p-4">
              <Shield className="mt-0.5 h-5 w-5 flex-shrink-0 text-primary" />
              <div>
                <p className="text-sm font-medium text-gray-900">Warranty & Authenticity</p>
                <p className="mt-0.5 text-xs text-gray-600">100% authentic products. Manufacturer warranty applicable where mentioned.</p>
              </div>
            </div>
          </div>

          {/* SKU */}
          <p className="mt-6 text-xs text-gray-400">
            SKU: {selectedVariant?.sku ?? product.sku}
          </p>
        </div>
      </div>

      {/* Reviews section */}
      {mounted && product && (() => {
        const reviews = getProductReviews(product.id);
        const { average, count } = getAverageRating(product.id);
        const authenticated = isAuthenticated();

        function handleSubmitReview() {
          if (!user || !product || !reviewComment.trim()) return;
          addReview(
            {
              productId: product.id,
              userId: user.id,
              userName: `${user.first_name} ${user.last_name}`,
              rating: reviewRating,
              title: reviewTitle.trim(),
              comment: reviewComment.trim(),
            },
            TENANT_ID,
            token || undefined,
          );
          setShowReviewForm(false);
          setReviewRating(5);
          setReviewTitle('');
          setReviewComment('');
        }

        return (
          <div className="mt-16 border-t border-border pt-10">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-2xl font-bold text-text">Customer Reviews</h2>
                {count > 0 && (
                  <div className="mt-2 flex items-center gap-2">
                    <div className="flex items-center gap-0.5">
                      {[1, 2, 3, 4, 5].map((star) => (
                        <Star key={star} className={cn('h-4 w-4', star <= Math.round(average) ? 'fill-amber-400 text-amber-400' : 'text-text-muted')} />
                      ))}
                    </div>
                    <span className="text-sm font-medium text-text">{average.toFixed(1)}</span>
                    <span className="text-sm text-text-secondary">({count} review{count !== 1 ? 's' : ''})</span>
                  </div>
                )}
              </div>
              {authenticated && !showReviewForm && (
                <button onClick={() => setShowReviewForm(true)}
                  className="rounded-lg border border-primary px-4 py-2 text-sm font-medium text-primary hover:bg-primary/5 transition-colors">
                  Write a Review
                </button>
              )}
            </div>

            {showReviewForm && (
              <div className="mt-6 rounded-xl border border-border bg-surface p-6">
                <h3 className="text-lg font-semibold text-text mb-4">Write a Review</h3>
                <div className="space-y-4">
                  <div>
                    <label className="mb-1.5 block text-sm font-medium text-text-secondary">Rating</label>
                    <div className="flex gap-1">
                      {[1, 2, 3, 4, 5].map((star) => (
                        <button key={star}
                          onMouseEnter={() => setHoverRating(star)}
                          onMouseLeave={() => setHoverRating(0)}
                          onClick={() => setReviewRating(star)}
                          className="p-0.5">
                          <Star className={cn('h-6 w-6 transition-colors', (hoverRating || reviewRating) >= star ? 'fill-amber-400 text-amber-400' : 'text-text-muted')} />
                        </button>
                      ))}
                    </div>
                  </div>
                  <div>
                    <label className="mb-1.5 block text-sm font-medium text-text-secondary">Title (optional)</label>
                    <input value={reviewTitle} onChange={(e) => setReviewTitle(e.target.value)} placeholder="Summarize your review"
                      className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary" />
                  </div>
                  <div>
                    <label className="mb-1.5 block text-sm font-medium text-text-secondary">Review</label>
                    <textarea value={reviewComment} onChange={(e) => setReviewComment(e.target.value)} rows={4} placeholder="What did you think about this product?"
                      className="w-full rounded-lg border border-border bg-surface px-3.5 py-2.5 text-sm text-text placeholder:text-text-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none" />
                  </div>
                  <div className="flex gap-3">
                    <button onClick={handleSubmitReview} disabled={!reviewComment.trim()}
                      className="rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white hover:bg-primary-dark disabled:opacity-50 transition-colors">
                      Submit Review
                    </button>
                    <button onClick={() => setShowReviewForm(false)}
                      className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover transition-colors">
                      Cancel
                    </button>
                  </div>
                </div>
              </div>
            )}

            {reviews.length === 0 && !showReviewForm ? (
              <div className="mt-6 rounded-xl border border-border bg-surface-secondary p-8 text-center">
                <div className="mb-3 flex items-center justify-center gap-1">
                  {[1, 2, 3, 4, 5].map((star) => (
                    <Star key={star} className="h-5 w-5 text-text-muted" />
                  ))}
                </div>
                <p className="text-text-secondary">No reviews yet. Be the first to review this product!</p>
                {!authenticated && (
                  <Link href="/login" className="mt-3 inline-block text-sm font-medium text-primary hover:text-primary-dark">
                    Sign in to write a review
                  </Link>
                )}
              </div>
            ) : (
              <div className="mt-6 space-y-4">
                {reviews.map((review) => (
                  <div key={review.id} className="rounded-xl border border-border bg-surface p-5">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">
                          {review.userName.split(' ').map((n) => n[0]).join('').slice(0, 2)}
                        </div>
                        <div>
                          <p className="text-sm font-medium text-text">{review.userName}</p>
                          <div className="flex items-center gap-1">
                            {[1, 2, 3, 4, 5].map((star) => (
                              <Star key={star} className={cn('h-3 w-3', star <= review.rating ? 'fill-amber-400 text-amber-400' : 'text-text-muted')} />
                            ))}
                          </div>
                        </div>
                      </div>
                      <span className="text-xs text-text-muted">{new Date(review.createdAt).toLocaleDateString()}</span>
                    </div>
                    {review.title && <p className="mt-3 text-sm font-semibold text-text">{review.title}</p>}
                    <p className="mt-1 text-sm text-text-secondary leading-relaxed">{review.comment}</p>
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })()}

      {/* Recommendations section */}
      {mounted && recommendations.length > 0 && (() => {
        const recProducts = recommendations
          .map((rec) => {
            const p = products.find((pr) => pr.id === rec.product_id);
            return p ? { ...p, score: rec.score, reason: rec.reason } : null;
          })
          .filter(Boolean) as (typeof products[number] & { score: number; reason: string })[];

        if (recProducts.length === 0) return null;

        return (
          <div className="mt-16 border-t border-border pt-10">
            <h2 className="text-2xl font-bold text-text">You May Also Like</h2>
            <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {recProducts.slice(0, 4).map((rec) => {
                const recIndex = products.indexOf(products.find((p) => p.id === rec.id)!);
                const recGradient = gradients[Math.abs(recIndex) % gradients.length];
                const recHasImages = rec.images && rec.images.length > 0;
                return (
                  <Link key={rec.id} href={`/products/${rec.slug}`}
                    className="group rounded-xl border border-border bg-surface overflow-hidden transition-shadow hover:shadow-md">
                    <div className={cn('flex aspect-square items-center justify-center overflow-hidden', recHasImages ? 'bg-surface-secondary' : `bg-gradient-to-br ${recGradient}`)}>
                      {recHasImages ? (
                        <img src={rec.images![0]} alt={rec.name} className="h-full w-full object-cover transition-transform group-hover:scale-105" />
                      ) : (
                        <span className="text-5xl font-bold text-white/40">{rec.name.charAt(0)}</span>
                      )}
                    </div>
                    <div className="p-3">
                      <p className="text-sm font-medium text-text line-clamp-1">{rec.name}</p>
                      <p className="mt-1 text-sm font-bold text-text">{formatCurrency(rec.price)}</p>
                      {rec.reason && (
                        <p className="mt-1 text-xs text-text-muted line-clamp-1">{rec.reason}</p>
                      )}
                    </div>
                  </Link>
                );
              })}
            </div>
          </div>
        );
      })()}
    </div>
  );
}
