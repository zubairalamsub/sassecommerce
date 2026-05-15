'use client';

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Search,
  Loader2,
  X,
  ArrowRight,
  Clock,
  TrendingUp,
  CornerDownLeft,
  ChevronRight,
  Command as CommandIcon,
  Sparkles,
} from 'lucide-react';
import { productApi, type Product } from '@/lib/api';
import { useProductStore } from '@/stores/products';
import { useRecentSearchesStore } from '@/stores/recent-searches';
import { cn, formatCurrency, mediaUrl } from '@/lib/utils';
import { DEFAULT_TENANT_ID as TENANT_ID } from '@/lib/tenant';
const DEBOUNCE_MS = 200;
const MIN_QUERY_LEN = 2;
const PREVIEW_LIMIT = 8;
const LOW_STOCK_THRESHOLD = 5;

const fallbackGradients = [
  'from-rose-200 to-pink-300 dark:from-rose-900/50 dark:to-pink-900/50',
  'from-blue-200 to-indigo-300 dark:from-blue-900/50 dark:to-indigo-900/50',
  'from-emerald-200 to-teal-300 dark:from-emerald-900/50 dark:to-teal-900/50',
  'from-amber-200 to-orange-300 dark:from-amber-900/50 dark:to-orange-900/50',
  'from-violet-200 to-purple-300 dark:from-violet-900/50 dark:to-purple-900/50',
  'from-cyan-200 to-sky-300 dark:from-cyan-900/50 dark:to-sky-900/50',
];

const PLACEHOLDER_SUGGESTIONS = ['t-shirt', 'kurta', 'shoes', 'watch', 'saree'];

function stockState(p: Product): 'in' | 'low' | 'out' {
  if (p.variants && p.variants.length > 0) {
    const totals = p.variants
      .map((v) => (typeof v.stock === 'number' ? v.stock : null))
      .filter((s): s is number => s !== null);
    if (totals.length > 0) {
      const total = totals.reduce((a, b) => a + b, 0);
      if (total <= 0) return 'out';
      if (total <= LOW_STOCK_THRESHOLD) return 'low';
      return 'in';
    }
  }
  if (p.status === 'archived' || p.status === 'inactive') return 'out';
  return 'in';
}

function productHref(p: Product): string {
  return `/products/${p.slug || p.id}`;
}

function isMacLike(): boolean {
  if (typeof navigator === 'undefined') return false;
  return /Mac|iPod|iPhone|iPad/.test(navigator.platform || navigator.userAgent || '');
}

/* ─────────────────────────────────────────────────────────────────────────
 * <InstantSearchTrigger />
 *
 * Header-mounted pill. Looks like an input, behaves like a button.
 * ────────────────────────────────────────────────────────────────────── */

export interface InstantSearchTriggerProps {
  onClick: () => void;
  className?: string;
}

export function InstantSearchTrigger({ onClick, className }: InstantSearchTriggerProps) {
  const [mac, setMac] = useState(false);
  useEffect(() => setMac(isMacLike()), []);

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Open search"
      aria-keyshortcuts="Meta+K Control+K"
      className={cn(
        'group flex w-full items-center gap-2 rounded-full border border-transparent bg-surface-secondary px-3.5 py-2 text-left text-sm text-text-muted ring-1 ring-transparent transition-all hover:border-border hover:bg-surface-hover hover:text-text focus:outline-none focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/30',
        className,
      )}
    >
      <Search className="h-4 w-4 flex-shrink-0 text-text-muted transition-colors group-hover:text-text-secondary" />
      <span className="flex-1 truncate">Search products…</span>
      <kbd
        aria-hidden
        className="hidden items-center gap-0.5 rounded-md border border-border bg-surface px-1.5 py-0.5 font-mono text-[10px] font-semibold text-text-muted sm:inline-flex"
      >
        {mac ? <CommandIcon className="h-3 w-3" /> : <span>Ctrl</span>}
        <span>K</span>
      </kbd>
    </button>
  );
}

/* ─────────────────────────────────────────────────────────────────────────
 * <InstantSearchPanel />
 *
 * The centered command-palette overlay. Fully controlled via `open`.
 * ────────────────────────────────────────────────────────────────────── */

export interface InstantSearchPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function InstantSearchPanel({ open, onOpenChange }: InstantSearchPanelProps) {
  const router = useRouter();
  const categories = useProductStore((s) => s.categories);
  const {
    queries: recentQueries,
    push: pushRecent,
    remove: removeRecent,
    clear: clearRecent,
  } = useRecentSearchesStore();

  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Product[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [activeIdx, setActiveIdx] = useState(-1);

  const inputRef = useRef<HTMLInputElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const trimmed = query.trim();
  const isSearching = trimmed.length >= MIN_QUERY_LEN;

  const activeCategories = useMemo(
    () => categories.filter((c) => c.status === 'active'),
    [categories],
  );

  // Filter matching categories when typing
  const matchingCategories = useMemo(() => {
    if (!isSearching) return activeCategories.slice(0, 6);
    const q = trimmed.toLowerCase();
    return activeCategories.filter((c) => c.name.toLowerCase().includes(q)).slice(0, 4);
  }, [activeCategories, isSearching, trimmed]);

  const preview = useMemo(() => results.slice(0, PREVIEW_LIMIT), [results]);
  const navigableCount = preview.length;
  const showEmpty = isSearching && !loading && results.length === 0;

  /* ── Reset state on close ───────────────────────────────────────────── */
  useEffect(() => {
    if (!open) {
      // Defer to avoid React batching with parent setOpen
      const id = setTimeout(() => {
        setQuery('');
        setResults([]);
        setTotal(0);
        setLoading(false);
        setActiveIdx(-1);
      }, 0);
      return () => clearTimeout(id);
    }
  }, [open]);

  /* ── Autofocus input when panel opens ───────────────────────────────── */
  useEffect(() => {
    if (!open) return;
    const id = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(id);
  }, [open]);

  /* ── Body scroll lock while open ────────────────────────────────────── */
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, [open]);

  /* ── Debounced fetch ────────────────────────────────────────────────── */
  useEffect(() => {
    if (!open) return;

    if (abortRef.current) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }

    if (!isSearching) {
      const id = setTimeout(() => {
        setResults([]);
        setTotal(0);
        setLoading(false);
        setActiveIdx(-1);
      }, 0);
      return () => clearTimeout(id);
    }

    debounceRef.current = setTimeout(async () => {
      const controller = new AbortController();
      abortRef.current = controller;
      setLoading(true);
      try {
        const res = await productApi.search(TENANT_ID, trimmed);
        if (controller.signal.aborted) return;
        const data = res.data || [];
        setResults(data);
        setTotal(res.pagination?.total_items ?? res.total ?? data.length);
        setActiveIdx(data.length > 0 ? 0 : -1);
      } catch (err) {
        if (controller.signal.aborted) return;
        if (err instanceof DOMException && err.name === 'AbortError') return;
        setResults([]);
        setTotal(0);
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
        if (abortRef.current === controller) abortRef.current = null;
      }
    }, DEBOUNCE_MS);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [trimmed, isSearching, open]);

  /* ── Cleanup on unmount ─────────────────────────────────────────────── */
  useEffect(() => {
    return () => {
      if (abortRef.current) abortRef.current.abort();
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  /* ── Scroll active row into view ────────────────────────────────────── */
  useEffect(() => {
    if (activeIdx < 0 || !listRef.current) return;
    const el = listRef.current.children[activeIdx] as HTMLElement | undefined;
    el?.scrollIntoView({ block: 'nearest' });
  }, [activeIdx]);

  /* ── Navigation helpers ─────────────────────────────────────────────── */
  const close = useCallback(() => onOpenChange(false), [onOpenChange]);

  const goToSearchPage = useCallback(
    (q: string) => {
      const value = q.trim();
      if (!value) return;
      pushRecent(value);
      router.push(`/search?q=${encodeURIComponent(value)}`);
      close();
    },
    [router, pushRecent, close],
  );

  const goToProduct = useCallback(
    (p: Product) => {
      if (trimmed) pushRecent(trimmed);
      router.push(productHref(p));
      close();
    },
    [router, trimmed, pushRecent, close],
  );

  const goToCategory = useCallback(
    (slug: string) => {
      router.push(`/products?category=${slug}`);
      close();
    },
    [router, close],
  );

  /* ── Keyboard handling on the input ─────────────────────────────────── */
  function handleKeyDown(e: ReactKeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (navigableCount === 0) return;
      setActiveIdx((i) => (i + 1) % navigableCount);
      return;
    }

    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (navigableCount === 0) return;
      setActiveIdx((i) => (i <= 0 ? navigableCount - 1 : i - 1));
      return;
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      if (activeIdx >= 0 && activeIdx < preview.length) {
        goToProduct(preview[activeIdx]);
      } else if (trimmed) {
        goToSearchPage(trimmed);
      }
    }
  }

  /* ── Focus trap inside panel ────────────────────────────────────────── */
  useEffect(() => {
    if (!open) return;
    function onTab(e: KeyboardEvent) {
      if (e.key !== 'Tab' || !panelRef.current) return;
      const focusables = panelRef.current.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
      );
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      const active = document.activeElement as HTMLElement | null;
      if (e.shiftKey) {
        if (active === first || !panelRef.current.contains(active)) {
          e.preventDefault();
          last.focus();
        }
      } else if (active === last) {
        e.preventDefault();
        first.focus();
      }
    }
    document.addEventListener('keydown', onTab);
    return () => document.removeEventListener('keydown', onTab);
  }, [open]);

  /* ── Render ─────────────────────────────────────────────────────────── */
  const popularQueries = recentQueries.length > 0 ? recentQueries.slice(0, 5) : null;

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          key="instant-search-overlay"
          role="dialog"
          aria-modal="true"
          aria-label="Search products"
          className="fixed inset-0 z-[60] flex items-stretch justify-center md:items-start md:pt-[10vh]"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.15 }}
        >
          {/* Backdrop */}
          <button
            type="button"
            aria-label="Close search"
            tabIndex={-1}
            onClick={close}
            className="absolute inset-0 bg-black/40 backdrop-blur-sm md:bg-black/50"
          />

          {/* Panel */}
          <motion.div
            ref={panelRef}
            initial={{ opacity: 0, y: 8, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.98 }}
            transition={{ type: 'spring', stiffness: 400, damping: 32 }}
            className="relative z-10 flex h-full w-full flex-col overflow-hidden bg-surface shadow-2xl md:h-auto md:max-h-[78vh] md:w-full md:max-w-[720px] md:rounded-2xl md:border md:border-border"
            onClick={(e) => e.stopPropagation()}
          >
            {/* ─── Input row ────────────────────────────────────────── */}
            <div className="flex items-center gap-2 border-b border-border px-3 py-3 md:gap-3 md:px-4 md:py-3.5">
              <Search className="h-5 w-5 flex-shrink-0 text-text-muted" />
              <input
                ref={inputRef}
                type="text"
                role="combobox"
                aria-autocomplete="list"
                aria-expanded
                aria-controls="instant-search-list"
                placeholder="Search products…"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={handleKeyDown}
                className="min-w-0 flex-1 bg-transparent text-base text-text outline-none placeholder:text-text-muted md:text-[15px]"
              />
              {loading && (
                <Loader2 className="h-4 w-4 flex-shrink-0 animate-spin text-primary" />
              )}
              {query && !loading && (
                <button
                  type="button"
                  aria-label="Clear search"
                  onClick={() => {
                    setQuery('');
                    inputRef.current?.focus();
                  }}
                  className="rounded-md p-1 text-text-muted transition-colors hover:bg-surface-hover hover:text-text"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
              {/* Desktop: Esc kbd | Mobile: Cancel button */}
              <kbd
                aria-hidden
                className="hidden flex-shrink-0 items-center rounded-md border border-border bg-surface-secondary px-1.5 py-0.5 font-mono text-[10px] font-semibold text-text-muted md:inline-flex"
              >
                Esc
              </kbd>
              <button
                type="button"
                onClick={close}
                className="flex-shrink-0 text-sm font-medium text-primary hover:underline md:hidden"
              >
                Cancel
              </button>
            </div>

            {/* ─── Body ─────────────────────────────────────────────── */}
            <div className="flex-1 overflow-y-auto md:max-h-[58vh]">
              <div className="flex flex-col md:flex-row">
                {/* ── Left column: results ───────────────────────── */}
                <div className="flex-1 md:border-r md:border-border">
                  {/* Loading skeletons */}
                  {isSearching && loading && results.length === 0 && (
                    <ul className="space-y-1 p-2">
                      {Array.from({ length: 4 }).map((_, i) => (
                        <li
                          key={i}
                          className="flex items-center gap-3 rounded-lg px-3 py-2.5"
                        >
                          <div className="h-16 w-16 flex-shrink-0 animate-pulse rounded-lg bg-surface-secondary md:h-20 md:w-20" />
                          <div className="flex flex-1 flex-col gap-2">
                            <div className="h-3.5 w-3/4 animate-pulse rounded bg-surface-secondary" />
                            <div className="h-3 w-1/3 animate-pulse rounded bg-surface-secondary" />
                          </div>
                        </li>
                      ))}
                    </ul>
                  )}

                  {/* Result rows */}
                  {isSearching && preview.length > 0 && (
                    <>
                      <SectionLabel className="px-4 pt-3">
                        <Sparkles className="h-3 w-3" />
                        Products
                      </SectionLabel>
                      <ul
                        id="instant-search-list"
                        ref={listRef}
                        role="listbox"
                        className="p-2"
                      >
                        {preview.map((p, idx) => (
                          <SearchResultRow
                            key={p.id}
                            product={p}
                            index={idx}
                            active={idx === activeIdx}
                            onHover={() => setActiveIdx(idx)}
                            onSelect={() => goToProduct(p)}
                          />
                        ))}
                      </ul>
                      {(total > preview.length || results.length >= PREVIEW_LIMIT) && (
                        <div className="border-t border-border bg-surface-secondary">
                          <button
                            type="button"
                            onClick={() => goToSearchPage(trimmed)}
                            className="flex w-full items-center justify-between gap-2 px-4 py-3 text-sm font-medium text-primary transition-colors hover:bg-surface-hover"
                          >
                            <span>
                              View all {total || results.length} result
                              {(total || results.length) !== 1 ? 's' : ''}
                            </span>
                            <ArrowRight className="h-4 w-4" />
                          </button>
                        </div>
                      )}
                    </>
                  )}

                  {/* Empty-query state — popular searches */}
                  {!isSearching && (
                    <div className="p-2">
                      <SectionLabel className="px-3 pt-2">
                        <TrendingUp className="h-3 w-3" />
                        {popularQueries ? 'Recent searches' : 'Try searching'}
                      </SectionLabel>
                      <ul className="mt-1 space-y-0.5">
                        {(popularQueries ?? PLACEHOLDER_SUGGESTIONS).map((q) => {
                          // The row itself can't be a <button> because we nest a
                          // <button> Remove inside — invalid HTML. Use a div with
                          // role=button + keyboard handlers instead.
                          const selectQuery = () => {
                            setQuery(q);
                            inputRef.current?.focus();
                          };
                          return (
                            <li key={q}>
                              <div
                                role="button"
                                tabIndex={0}
                                onClick={selectQuery}
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter' || e.key === ' ') {
                                    e.preventDefault();
                                    selectQuery();
                                  }
                                }}
                                className="group flex w-full cursor-pointer items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-surface-hover focus-visible:bg-surface-hover focus-visible:outline-none"
                              >
                                {popularQueries ? (
                                  <Clock className="h-4 w-4 flex-shrink-0 text-text-muted" />
                                ) : (
                                  <Search className="h-4 w-4 flex-shrink-0 text-text-muted" />
                                )}
                                <span className="flex-1 truncate text-sm text-text">{q}</span>
                                {popularQueries && (
                                  <button
                                    type="button"
                                    aria-label={`Remove ${q} from recent searches`}
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      e.preventDefault();
                                      removeRecent(q);
                                    }}
                                    className="rounded p-0.5 text-text-muted opacity-0 transition-all hover:bg-surface hover:text-text group-hover:opacity-100"
                                  >
                                    <X className="h-3.5 w-3.5" />
                                  </button>
                                )}
                                <ChevronRight className="h-4 w-4 flex-shrink-0 text-text-muted opacity-0 transition-opacity group-hover:opacity-100" />
                              </div>
                            </li>
                          );
                        })}
                      </ul>
                      {popularQueries && (
                        <div className="mt-1 px-3 pb-1 pt-2">
                          <button
                            type="button"
                            onClick={() => clearRecent()}
                            className="text-[11px] font-medium text-text-muted transition-colors hover:text-text"
                          >
                            Clear recent searches
                          </button>
                        </div>
                      )}
                    </div>
                  )}

                  {/* No-results state */}
                  {showEmpty && (
                    <div className="p-6 md:p-8">
                      <p className="text-base font-semibold text-text">
                        We couldn&apos;t find anything for &ldquo;{trimmed}&rdquo;
                      </p>
                      <p className="mt-1 text-sm text-text-muted">
                        Try a different keyword or browse a category below.
                      </p>
                      {activeCategories.length > 0 && (
                        <div className="mt-5">
                          <SectionLabel>
                            <TrendingUp className="h-3 w-3" />
                            Popular categories
                          </SectionLabel>
                          <div className="mt-2 flex flex-wrap gap-1.5">
                            {activeCategories.slice(0, 6).map((c) => {
                              const img = c.image || c.image_url || '';
                              return (
                                <button
                                  key={c.id}
                                  type="button"
                                  onClick={() => goToCategory(c.slug)}
                                  className={cn(
                                    'inline-flex items-center gap-1.5 rounded-full border border-border bg-surface-secondary py-1.5 text-xs font-medium text-text-secondary transition-colors hover:bg-surface-hover hover:text-text',
                                    img ? 'pl-1 pr-3' : 'px-3',
                                  )}
                                >
                                  {img && (
                                    // eslint-disable-next-line @next/next/no-img-element
                                    <img
                                      src={mediaUrl(img)}
                                      alt=""
                                      className="h-5 w-5 rounded-full object-cover"
                                      loading="lazy"
                                    />
                                  )}
                                  {c.name}
                                </button>
                              );
                            })}
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>

                {/* ── Right column: categories + recent (desktop) ── */}
                <div className="md:w-[260px] md:flex-shrink-0">
                  {matchingCategories.length > 0 && (
                    <div className="p-2 md:p-3">
                      <SectionLabel className="px-2 pt-1">
                        Categories
                      </SectionLabel>
                      <ul className="mt-1 space-y-0.5">
                        {matchingCategories.map((c) => {
                          const img = c.image || c.image_url || '';
                          return (
                            <li key={c.id}>
                              <button
                                type="button"
                                onClick={() => goToCategory(c.slug)}
                                className="group flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left transition-colors hover:bg-surface-hover"
                              >
                                {img ? (
                                  // eslint-disable-next-line @next/next/no-img-element
                                  <img
                                    src={mediaUrl(img)}
                                    alt=""
                                    className="h-6 w-6 flex-shrink-0 rounded object-cover"
                                    loading="lazy"
                                  />
                                ) : (
                                  <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded bg-primary/10 text-[10px] font-bold text-primary">
                                    {c.name.charAt(0).toUpperCase()}
                                  </div>
                                )}
                                <span className="flex-1 truncate text-sm text-text">
                                  {c.name}
                                </span>
                                <ChevronRight className="h-3.5 w-3.5 flex-shrink-0 text-text-muted opacity-0 transition-opacity group-hover:opacity-100" />
                              </button>
                            </li>
                          );
                        })}
                      </ul>
                    </div>
                  )}

                  {/* Recent on right column when typing */}
                  {isSearching && popularQueries && (
                    <div className="border-t border-border p-2 md:p-3">
                      <SectionLabel className="px-2 pt-1">
                        <Clock className="h-3 w-3" />
                        Recent
                      </SectionLabel>
                      <ul className="mt-1 space-y-0.5">
                        {popularQueries.slice(0, 5).map((q) => (
                          <li key={q}>
                            <button
                              type="button"
                              onClick={() => goToSearchPage(q)}
                              className="flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-left transition-colors hover:bg-surface-hover"
                            >
                              <span className="flex-1 truncate text-sm text-text-secondary">
                                {q}
                              </span>
                            </button>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* ─── Footer (desktop only) ────────────────────────────── */}
            <div className="hidden items-center justify-between gap-4 border-t border-border bg-surface-secondary px-4 py-2 text-[11px] text-text-muted md:flex">
              <div className="flex items-center gap-3">
                <span className="inline-flex items-center gap-1">
                  <KbdKey>↑</KbdKey>
                  <KbdKey>↓</KbdKey>
                  navigate
                </span>
                <span className="inline-flex items-center gap-1">
                  <KbdKey>
                    <CornerDownLeft className="h-2.5 w-2.5" />
                  </KbdKey>
                  open
                </span>
                <span className="inline-flex items-center gap-1">
                  <KbdKey>Esc</KbdKey>
                  close
                </span>
              </div>
              <Link
                href="/products"
                onClick={close}
                className="font-medium text-text-secondary transition-colors hover:text-text"
              >
                Browse all products →
              </Link>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

/* ─────────────────────────── Helpers ─────────────────────────── */

function SectionLabel({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex items-center gap-1.5 px-1 text-[11px] font-semibold uppercase tracking-wider text-text-muted',
        className,
      )}
    >
      {children}
    </div>
  );
}

function KbdKey({ children }: { children: React.ReactNode }) {
  return (
    <kbd className="inline-flex h-4 min-w-[16px] items-center justify-center rounded border border-border bg-surface px-1 font-mono text-[10px] font-semibold text-text-secondary">
      {children}
    </kbd>
  );
}

/* ─────────────────────────── Result row ─────────────────────────── */

function SearchResultRow({
  product,
  index,
  active,
  onHover,
  onSelect,
}: {
  product: Product;
  index: number;
  active: boolean;
  onHover: () => void;
  onSelect: () => void;
}) {
  const categories = useProductStore((s) => s.categories);
  const firstImage = product.images?.[0];
  const stock = stockState(product);
  const discounted =
    product.compare_at_price != null && product.compare_at_price > product.price;
  const gradient = fallbackGradients[index % fallbackGradients.length];
  const categoryName = useMemo(
    () => categories.find((c) => c.id === product.category_id)?.name,
    [categories, product.category_id],
  );

  return (
    <li
      role="option"
      aria-selected={active}
      onMouseEnter={onHover}
      onMouseDown={(e) => {
        e.preventDefault();
        onSelect();
      }}
      className={cn(
        'flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2.5 transition-colors',
        active ? 'bg-surface-hover' : 'hover:bg-surface-hover',
      )}
    >
      {/* Thumbnail */}
      <div
        className={cn(
          'flex h-16 w-16 flex-shrink-0 items-center justify-center overflow-hidden rounded-lg md:h-20 md:w-20',
          firstImage ? 'bg-surface-secondary' : `bg-gradient-to-br ${gradient}`,
        )}
      >
        {firstImage ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={mediaUrl(firstImage)}
            alt=""
            loading="lazy"
            className="h-full w-full object-cover"
          />
        ) : (
          <span className="text-2xl font-bold text-white/80">
            {product.name.charAt(0).toUpperCase()}
          </span>
        )}
      </div>

      {/* Info */}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-semibold text-text">{product.name}</p>
        <div className="mt-1 flex items-center gap-2 text-xs">
          {categoryName && (
            <span className="truncate text-text-secondary">{categoryName}</span>
          )}
          {categoryName && (
            <span className="text-text-muted" aria-hidden>
              ·
            </span>
          )}
          <span className="font-semibold text-text">
            {formatCurrency(product.price)}
          </span>
          {discounted && (
            <span className="text-text-muted line-through">
              {formatCurrency(product.compare_at_price!)}
            </span>
          )}
        </div>
        {product.sku && (
          <p className="mt-0.5 truncate font-mono text-[10px] uppercase tracking-wide text-text-muted">
            {product.sku}
          </p>
        )}
      </div>

      {/* Stock pill */}
      {stock !== 'in' && (
        <span
          className={cn(
            'flex-shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide',
            stock === 'out'
              ? 'bg-accent/10 text-accent'
              : 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
          )}
        >
          {stock === 'out' ? 'Out' : 'Low'}
        </span>
      )}
    </li>
  );
}

/* ─────────────────────────── Default export (back-compat) ─────────────────
 * Some old call sites import the file as default. Re-export the panel
 * controlled by an internal piece of state so legacy callers don't blow up.
 * New code should import { InstantSearchTrigger, InstantSearchPanel } above.
 * ────────────────────────────────────────────────────────────────────── */

export default function InstantSearch() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <InstantSearchTrigger onClick={() => setOpen(true)} />
      <InstantSearchPanel open={open} onOpenChange={setOpen} />
    </>
  );
}
