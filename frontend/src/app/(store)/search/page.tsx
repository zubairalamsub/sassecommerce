'use client';

import { useState, useEffect, useCallback, Suspense } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  Search,
  SlidersHorizontal,
  X,
  Loader2,
  ShoppingCart,
  Check,
  Heart,
  ChevronLeft,
  ChevronRight,
  Filter,
} from 'lucide-react';
import { searchApi, type SearchProduct, type SearchFacets } from '@/lib/api';
import { useCartStore } from '@/stores/cart';
import { useWishlistStore } from '@/stores/wishlist';
import { formatCurrency } from '@/lib/utils';

const TENANT_ID = 'tenant_saajan';
const PAGE_SIZE = 20;

const gradients = [
  'from-rose-100 to-pink-200',
  'from-blue-100 to-indigo-200',
  'from-emerald-100 to-teal-200',
  'from-amber-100 to-orange-200',
  'from-violet-100 to-purple-200',
  'from-cyan-100 to-sky-200',
];

export default function SearchPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-[50vh] items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      }
    >
      <SearchContent />
    </Suspense>
  );
}

function SearchContent() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const initialQ = searchParams.get('q') || '';
  const initialBrand = searchParams.get('brand') || '';
  const initialMinPrice = searchParams.get('min_price') ? Number(searchParams.get('min_price')) : undefined;
  const initialMaxPrice = searchParams.get('max_price') ? Number(searchParams.get('max_price')) : undefined;
  const initialInStock = searchParams.get('in_stock') === 'true' ? true : undefined;
  const initialPage = searchParams.get('page') ? Number(searchParams.get('page')) : 1;
  const initialSort = searchParams.get('sort_by') || '';

  const [query, setQuery] = useState(initialQ);
  const [inputValue, setInputValue] = useState(initialQ);
  const [brand, setBrand] = useState(initialBrand);
  const [minPrice, setMinPrice] = useState<string>(initialMinPrice !== undefined ? String(initialMinPrice) : '');
  const [maxPrice, setMaxPrice] = useState<string>(initialMaxPrice !== undefined ? String(initialMaxPrice) : '');
  const [inStock, setInStock] = useState(initialInStock ?? false);
  const [sortBy, setSortBy] = useState(initialSort);
  const [page, setPage] = useState(initialPage);
  const [showFilters, setShowFilters] = useState(false);

  const [results, setResults] = useState<SearchProduct[]>([]);
  const [facets, setFacets] = useState<SearchFacets | null>(null);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(false);
  const [addedId, setAddedId] = useState<string | null>(null);

  const addItem = useCartStore((s) => s.addItem);
  const { toggleItem, isInWishlist } = useWishlistStore();

  const pushUrl = useCallback(
    (overrides: Record<string, string | number | boolean | undefined>) => {
      const p = new URLSearchParams();
      const merged = { q: query, brand, min_price: minPrice, max_price: maxPrice, in_stock: inStock || undefined, sort_by: sortBy, page, ...overrides };
      if (merged.q) p.set('q', String(merged.q));
      if (merged.brand) p.set('brand', String(merged.brand));
      if (merged.min_price) p.set('min_price', String(merged.min_price));
      if (merged.max_price) p.set('max_price', String(merged.max_price));
      if (merged.in_stock) p.set('in_stock', 'true');
      if (merged.sort_by) p.set('sort_by', String(merged.sort_by));
      if (merged.page && Number(merged.page) > 1) p.set('page', String(merged.page));
      router.push(`/search?${p.toString()}`, { scroll: false });
    },
    [query, brand, minPrice, maxPrice, inStock, sortBy, page, router],
  );

  const doSearch = useCallback(async () => {
    setLoading(true);
    try {
      const res = await searchApi.search({
        tenant_id: TENANT_ID,
        q: query || undefined,
        brand: brand || undefined,
        min_price: minPrice ? Number(minPrice) : undefined,
        max_price: maxPrice ? Number(maxPrice) : undefined,
        in_stock: inStock || undefined,
        sort_by: sortBy || undefined,
        sort_order: sortBy ? 'desc' : undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setResults(res.products ?? []);
      setFacets(res.facets ?? null);
      setTotal(res.total ?? 0);
      setTotalPages(res.total_pages ?? 0);
    } catch {
      setResults([]);
      setTotal(0);
      setTotalPages(0);
    } finally {
      setLoading(false);
    }
  }, [query, brand, minPrice, maxPrice, inStock, sortBy, page]);

  useEffect(() => {
    doSearch();
  }, [doSearch]);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    setPage(1);
    setQuery(inputValue);
    pushUrl({ q: inputValue, page: 1 });
  }

  function handleBrandFilter(b: string) {
    const next = brand === b ? '' : b;
    setBrand(next);
    setPage(1);
    pushUrl({ brand: next, page: 1 });
  }

  function handlePriceFilter(e: React.FormEvent) {
    e.preventDefault();
    setPage(1);
    pushUrl({ min_price: minPrice, max_price: maxPrice, page: 1 });
  }

  function handleInStock(checked: boolean) {
    setInStock(checked);
    setPage(1);
    pushUrl({ in_stock: checked || undefined, page: 1 });
  }

  function handleSort(s: string) {
    setSortBy(s);
    setPage(1);
    pushUrl({ sort_by: s, page: 1 });
  }

  function handlePage(p: number) {
    setPage(p);
    pushUrl({ page: p });
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function clearFilters() {
    setBrand('');
    setMinPrice('');
    setMaxPrice('');
    setInStock(false);
    setSortBy('');
    setPage(1);
    pushUrl({ brand: undefined, min_price: undefined, max_price: undefined, in_stock: undefined, sort_by: undefined, page: 1 });
  }

  const hasFilters = !!(brand || minPrice || maxPrice || inStock || sortBy);

  function handleAddToCart(product: SearchProduct) {
    addItem({
      productId: product.id,
      name: product.name,
      sku: product.sku,
      price: product.price,
      quantity: 1,
      image: product.images?.[0],
    });
    setAddedId(product.id);
    setTimeout(() => setAddedId(null), 1500);
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      {/* Header */}
      <div className="mb-8">
        <form onSubmit={handleSearch} className="flex gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              placeholder="Search products…"
              className="w-full rounded-xl border border-gray-200 bg-white py-3 pl-12 pr-4 text-sm shadow-sm transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
            />
            {inputValue && (
              <button
                type="button"
                onClick={() => { setInputValue(''); setQuery(''); pushUrl({ q: undefined, page: 1 }); }}
                className="absolute right-4 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
          <button
            type="submit"
            className="rounded-xl bg-primary px-6 py-3 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
          >
            Search
          </button>
          <button
            type="button"
            onClick={() => setShowFilters((v) => !v)}
            className={`flex items-center gap-2 rounded-xl border px-4 py-3 text-sm font-medium transition-colors lg:hidden ${
              showFilters ? 'border-primary bg-primary/5 text-primary' : 'border-gray-200 text-gray-700 hover:border-gray-300'
            }`}
          >
            <Filter className="h-4 w-4" />
            Filters
          </button>
        </form>

        {/* Results summary */}
        <div className="mt-4 flex items-center justify-between">
          <p className="text-sm text-gray-500">
            {loading ? (
              'Searching…'
            ) : query ? (
              <>
                <span className="font-medium text-gray-900">{total}</span> results for{' '}
                <span className="font-medium text-gray-900">&ldquo;{query}&rdquo;</span>
              </>
            ) : (
              <>
                <span className="font-medium text-gray-900">{total}</span> products
              </>
            )}
          </p>

          <div className="flex items-center gap-2">
            {hasFilters && (
              <button
                onClick={clearFilters}
                className="flex items-center gap-1 text-xs text-red-500 hover:text-red-700"
              >
                <X className="h-3.5 w-3.5" />
                Clear filters
              </button>
            )}
            <select
              value={sortBy}
              onChange={(e) => handleSort(e.target.value)}
              className="rounded-lg border border-gray-200 px-3 py-1.5 text-sm focus:border-primary focus:outline-none"
            >
              <option value="">Sort: Relevance</option>
              <option value="price">Price (Low → High)</option>
              <option value="name">Name (A → Z)</option>
            </select>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-4">
        {/* Filters sidebar */}
        <aside className={`lg:block ${showFilters ? 'block' : 'hidden'}`}>
          <div className="sticky top-8 space-y-6 rounded-xl border border-gray-200 bg-white p-5">
            <div className="flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-sm font-semibold text-gray-900">
                <SlidersHorizontal className="h-4 w-4" />
                Filters
              </h2>
              {hasFilters && (
                <button onClick={clearFilters} className="text-xs text-primary hover:underline">
                  Clear all
                </button>
              )}
            </div>

            {/* In Stock */}
            <div>
              <label className="flex cursor-pointer items-center gap-2.5">
                <input
                  type="checkbox"
                  checked={inStock}
                  onChange={(e) => handleInStock(e.target.checked)}
                  className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                />
                <span className="text-sm text-gray-700">In stock only</span>
              </label>
            </div>

            {/* Price Range */}
            <div>
              <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500">
                Price Range
              </h3>
              {facets?.price_range && (
                <p className="mb-2 text-xs text-gray-400">
                  ৳{Math.round(facets.price_range.min)} – ৳{Math.round(facets.price_range.max)}
                </p>
              )}
              <form onSubmit={handlePriceFilter} className="flex items-center gap-2">
                <input
                  type="number"
                  value={minPrice}
                  onChange={(e) => setMinPrice(e.target.value)}
                  placeholder="Min"
                  min={0}
                  className="w-full rounded-lg border border-gray-200 px-2 py-1.5 text-sm focus:border-primary focus:outline-none"
                />
                <span className="text-gray-400">–</span>
                <input
                  type="number"
                  value={maxPrice}
                  onChange={(e) => setMaxPrice(e.target.value)}
                  placeholder="Max"
                  min={0}
                  className="w-full rounded-lg border border-gray-200 px-2 py-1.5 text-sm focus:border-primary focus:outline-none"
                />
                <button
                  type="submit"
                  className="rounded-lg bg-gray-100 px-2 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-200"
                >
                  Go
                </button>
              </form>
            </div>

            {/* Brands */}
            {facets?.brands && facets.brands.length > 0 && (
              <div>
                <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500">
                  Brand
                </h3>
                <div className="space-y-2">
                  {facets.brands.slice(0, 8).map((b) => (
                    <label key={b.key} className="flex cursor-pointer items-center gap-2.5">
                      <input
                        type="checkbox"
                        checked={brand === b.key}
                        onChange={() => handleBrandFilter(b.key)}
                        className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                      />
                      <span className="flex-1 text-sm text-gray-700 capitalize">{b.key}</span>
                      <span className="text-xs text-gray-400">({b.count})</span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            {/* Tags */}
            {facets?.tags && facets.tags.length > 0 && (
              <div>
                <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500">
                  Tags
                </h3>
                <div className="flex flex-wrap gap-2">
                  {facets.tags.slice(0, 12).map((t) => (
                    <span
                      key={t.key}
                      className="rounded-full bg-gray-100 px-2.5 py-1 text-xs text-gray-600"
                    >
                      {t.key}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </aside>

        {/* Results */}
        <div className="lg:col-span-3">
          {loading ? (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {Array.from({ length: 9 }).map((_, i) => (
                <div key={i} className="animate-pulse rounded-xl border border-gray-200 bg-white p-4">
                  <div className="mb-3 h-40 rounded-lg bg-gray-100" />
                  <div className="h-4 w-3/4 rounded bg-gray-100" />
                  <div className="mt-2 h-4 w-1/2 rounded bg-gray-100" />
                </div>
              ))}
            </div>
          ) : results.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-gray-200 bg-white py-24 text-center">
              <Search className="h-12 w-12 text-gray-300" />
              <h3 className="mt-4 text-lg font-semibold text-gray-900">No products found</h3>
              <p className="mt-1 text-sm text-gray-500">
                {query ? `No results for "${query}". Try a different search term.` : 'Try searching for something or adjust your filters.'}
              </p>
              {hasFilters && (
                <button
                  onClick={clearFilters}
                  className="mt-4 text-sm font-medium text-primary hover:underline"
                >
                  Clear all filters
                </button>
              )}
            </div>
          ) : (
            <>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {results.map((product, idx) => {
                  const gradient = gradients[idx % gradients.length];
                  const inWishlist = isInWishlist(product.id);
                  const added = addedId === product.id;
                  return (
                    <div
                      key={product.id}
                      className="group flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white transition-shadow hover:shadow-md"
                    >
                      {/* Image */}
                      <Link href={`/products/${product.id}`} className="relative block overflow-hidden">
                        <div className={`flex h-44 items-center justify-center bg-gradient-to-br ${gradient}`}>
                          {product.images?.[0] ? (
                            <img
                              src={product.images[0]}
                              alt={product.name}
                              className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                            />
                          ) : (
                            <span className="text-4xl font-bold text-white/60">
                              {product.name.charAt(0)}
                            </span>
                          )}
                        </div>
                        {!product.in_stock && (
                          <span className="absolute left-2 top-2 rounded-full bg-gray-900/80 px-2.5 py-1 text-xs font-medium text-white">
                            Out of stock
                          </span>
                        )}
                      </Link>

                      {/* Info */}
                      <div className="flex flex-1 flex-col p-4">
                        {product.brand && (
                          <p className="mb-0.5 text-xs font-medium uppercase tracking-wider text-gray-400">
                            {product.brand}
                          </p>
                        )}
                        <Link href={`/products/${product.id}`}>
                          <h3 className="line-clamp-2 text-sm font-semibold text-gray-900 transition-colors hover:text-primary">
                            {product.name}
                          </h3>
                        </Link>
                        <p className="mt-1 text-base font-bold text-gray-900">
                          {formatCurrency(product.price)}
                        </p>

                        <div className="mt-auto flex gap-2 pt-3">
                          <button
                            onClick={() => handleAddToCart(product)}
                            disabled={!product.in_stock || added}
                            className={`flex flex-1 items-center justify-center gap-1.5 rounded-lg py-2 text-sm font-medium transition-all ${
                              added
                                ? 'bg-green-500 text-white'
                                : product.in_stock
                                ? 'bg-primary text-white hover:bg-primary-dark'
                                : 'cursor-not-allowed bg-gray-100 text-gray-400'
                            }`}
                          >
                            {added ? (
                              <><Check className="h-4 w-4" /> Added</>
                            ) : (
                              <><ShoppingCart className="h-4 w-4" /> Add to cart</>
                            )}
                          </button>
                          <button
                            onClick={() => toggleItem({ productId: product.id, name: product.name, price: product.price, image: product.images?.[0], slug: product.id })}
                            className={`flex h-9 w-9 items-center justify-center rounded-lg border transition-colors ${
                              inWishlist
                                ? 'border-rose-200 bg-rose-50 text-rose-500'
                                : 'border-gray-200 text-gray-400 hover:border-rose-200 hover:text-rose-500'
                            }`}
                          >
                            <Heart className={`h-4 w-4 ${inWishlist ? 'fill-current' : ''}`} />
                          </button>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="mt-8 flex items-center justify-center gap-2">
                  <button
                    onClick={() => handlePage(page - 1)}
                    disabled={page <= 1}
                    className="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 text-gray-600 transition-colors hover:border-gray-300 disabled:opacity-40"
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </button>
                  {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
                    const p = i + 1;
                    return (
                      <button
                        key={p}
                        onClick={() => handlePage(p)}
                        className={`flex h-9 w-9 items-center justify-center rounded-lg border text-sm font-medium transition-colors ${
                          page === p
                            ? 'border-primary bg-primary text-white'
                            : 'border-gray-200 text-gray-700 hover:border-gray-300'
                        }`}
                      >
                        {p}
                      </button>
                    );
                  })}
                  <button
                    onClick={() => handlePage(page + 1)}
                    disabled={page >= totalPages}
                    className="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 text-gray-600 transition-colors hover:border-gray-300 disabled:opacity-40"
                  >
                    <ChevronRight className="h-4 w-4" />
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
