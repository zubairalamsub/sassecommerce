'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { motion, AnimatePresence } from 'framer-motion';
import { ChevronLeft, ChevronRight, ShoppingBag, Sparkles, Tag, Clock, ShoppingCart, Check, Heart } from 'lucide-react';
import { useProductStore } from '@/stores/products';
import { useStoreConfigStore, type StoreSection } from '@/stores/store-config';
import { useCartStore } from '@/stores/cart';
import { formatCurrency, cn } from '@/lib/utils';

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

const fadeUp = {
  hidden: { opacity: 0, y: 30 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.6, ease: [0, 0, 0.2, 1] as const } },
};

const staggerContainer = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.08 } },
};

const cardVariant = {
  hidden: { opacity: 0, y: 20, scale: 0.95 },
  visible: { opacity: 1, y: 0, scale: 1, transition: { duration: 0.4, ease: [0, 0, 0.2, 1] as const } },
};

export default function HomePage() {
  const { products, categories, fetchProducts, fetchCategories } = useProductStore();
  const { config, fetchConfig } = useStoreConfigStore();
  const addItem = useCartStore((s) => s.addItem);

  const [mounted, setMounted] = useState(false);
  const [currentBanner, setCurrentBanner] = useState(0);
  const [addedId, setAddedId] = useState<string | null>(null);

  useEffect(() => {
    setMounted(true);
    fetchProducts(TENANT_ID);
    fetchCategories(TENANT_ID);
    fetchConfig(TENANT_ID);
  }, [fetchProducts, fetchCategories, fetchConfig]);

  useEffect(() => {
    if (config.banners.length <= 1) return;
    const timer = setInterval(() => {
      setCurrentBanner((prev) => (prev + 1) % config.banners.length);
    }, 5000);
    return () => clearInterval(timer);
  }, [config.banners.length]);

  const activeProducts = products.filter((p) => p.status === 'active');
  const discountProducts = activeProducts.filter((p) => p.compare_at_price && p.compare_at_price > p.price);
  const enabledSections = config.sections.filter((s) => s.enabled).sort((a, b) => a.position - b.position);

  function handleAddToCart(product: typeof activeProducts[0]) {
    addItem({
      productId: product.id,
      name: product.name,
      price: product.price,
      quantity: 1,
      sku: product.sku,
      image: product.images?.[0] || undefined,
    });
    setAddedId(product.id);
    setTimeout(() => setAddedId(null), 1500);
  }

  if (!mounted) return null;

  return (
    <div className="bg-surface-secondary">
      {/* Announcement Bar */}
      <AnimatePresence>
        {config.announcement_bar.enabled && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            className="text-center text-sm font-medium py-2.5 px-4 overflow-hidden"
            style={{ backgroundColor: config.announcement_bar.bg_color, color: config.announcement_bar.text_color }}
          >
            {config.announcement_bar.text}
          </motion.div>
        )}
      </AnimatePresence>

      {/* Hero Banner */}
      {config.banners.length > 0 && (
        <div className="relative overflow-hidden">
          <AnimatePresence mode="wait">
            <motion.div
              key={currentBanner}
              initial={{ opacity: 0, scale: 1.05 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
              transition={{ duration: 0.7, ease: [0.4, 0, 0.2, 1] as const }}
            >
              <div className="relative py-20 sm:py-32 px-4" style={{ backgroundColor: config.banners[currentBanner].bg_color }}>
                {config.banners[currentBanner].image_url && (
                  <div className="absolute inset-0 bg-cover bg-center" style={{ backgroundImage: `url(${config.banners[currentBanner].image_url})` }} />
                )}
                <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/30 to-black/10" />
                <div className="relative mx-auto max-w-4xl text-center">
                  <motion.h1
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.2, duration: 0.5 }}
                    className="text-3xl sm:text-5xl lg:text-6xl font-bold text-white mb-4 drop-shadow-lg"
                  >
                    {config.banners[currentBanner].title}
                  </motion.h1>
                  <motion.p
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.35, duration: 0.5 }}
                    className="text-lg sm:text-xl text-white/90 mb-8 max-w-2xl mx-auto"
                  >
                    {config.banners[currentBanner].subtitle}
                  </motion.p>
                  {config.banners[currentBanner].cta_text && (
                    <motion.div
                      initial={{ opacity: 0, y: 20 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: 0.5, duration: 0.5 }}
                    >
                      <Link href={config.banners[currentBanner].cta_link || '/products'}
                        className="inline-flex items-center gap-2 rounded-full bg-white px-8 py-3.5 text-sm font-semibold shadow-lg transition-all hover:scale-105 hover:shadow-xl"
                        style={{ color: config.banners[currentBanner].bg_color }}>
                        <ShoppingBag className="h-4 w-4" />
                        {config.banners[currentBanner].cta_text}
                      </Link>
                    </motion.div>
                  )}
                </div>
              </div>
            </motion.div>
          </AnimatePresence>
          {config.banners.length > 1 && (
            <>
              <button onClick={() => setCurrentBanner((prev) => (prev - 1 + config.banners.length) % config.banners.length)}
                className="absolute left-4 top-1/2 -translate-y-1/2 rounded-full bg-white/20 p-3 text-white backdrop-blur-md hover:bg-white/30 transition-all hover:scale-110">
                <ChevronLeft className="h-5 w-5" />
              </button>
              <button onClick={() => setCurrentBanner((prev) => (prev + 1) % config.banners.length)}
                className="absolute right-4 top-1/2 -translate-y-1/2 rounded-full bg-white/20 p-3 text-white backdrop-blur-md hover:bg-white/30 transition-all hover:scale-110">
                <ChevronRight className="h-5 w-5" />
              </button>
              <div className="absolute bottom-6 left-1/2 -translate-x-1/2 flex gap-2">
                {config.banners.map((_, idx) => (
                  <button key={idx} onClick={() => setCurrentBanner(idx)}
                    className={cn('h-2.5 rounded-full transition-all duration-300', idx === currentBanner ? 'w-8 bg-white' : 'w-2.5 bg-white/40 hover:bg-white/60')} />
                ))}
              </div>
            </>
          )}
        </div>
      )}

      {/* Dynamic Sections */}
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {enabledSections.map((section) => (
          <SectionRenderer
            key={section.id}
            section={section}
            products={activeProducts}
            discountProducts={discountProducts}
            categories={categories}
            addedId={addedId}
            onAddToCart={handleAddToCart}
          />
        ))}
      </div>

      {/* CTA Section */}
      <motion.div
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: '-80px' }}
        variants={fadeUp}
        className="bg-gradient-to-br from-primary to-primary-dark py-20 mt-16"
      >
        <div className="mx-auto max-w-4xl text-center px-4">
          <h2 className="text-3xl font-bold text-white mb-4">Ready to explore?</h2>
          <p className="text-white/80 mb-8 text-lg">Browse our full collection of quality products</p>
          <Link href="/products" className="inline-flex items-center gap-2 rounded-full bg-white px-8 py-3.5 text-sm font-semibold text-primary hover:shadow-xl transition-all hover:scale-105">
            <ShoppingBag className="h-4 w-4" />
            View All Products
          </Link>
        </div>
      </motion.div>
    </div>
  );
}

/* ── Product Card ── */
function ProductCard({ product, gradient, addedId, onAddToCart }: {
  product: ReturnType<typeof useProductStore.getState>['products'][0];
  gradient: string;
  addedId: string | null;
  onAddToCart: (p: ReturnType<typeof useProductStore.getState>['products'][0]) => void;
}) {
  const discount = product.compare_at_price && product.compare_at_price > product.price
    ? Math.round(((product.compare_at_price - product.price) / product.compare_at_price) * 100)
    : null;
  const isAdded = addedId === product.id;

  return (
    <motion.div variants={cardVariant} className="group rounded-2xl border border-border bg-surface overflow-hidden transition-all duration-300 hover:shadow-xl hover:-translate-y-1">
      <Link href={`/products/${product.slug}`}>
        <div className={cn('relative h-52 bg-gradient-to-br flex items-center justify-center overflow-hidden', gradient)}>
          <span className="text-5xl font-bold text-white/30 group-hover:scale-110 transition-transform duration-500">{product.name[0]}</span>
          {/* Hover overlay */}
          <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors duration-300" />
          {discount && (
            <span className="absolute top-3 left-3 rounded-full bg-accent px-3 py-1 text-xs font-bold text-white shadow-lg">
              -{discount}%
            </span>
          )}
          {/* Quick action */}
          <div className="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-all duration-300 translate-y-1 group-hover:translate-y-0">
            <button className="flex h-8 w-8 items-center justify-center rounded-full bg-white/90 dark:bg-gray-800/90 text-text-secondary hover:text-accent transition-colors shadow-md">
              <Heart className="h-4 w-4" />
            </button>
          </div>
        </div>
      </Link>
      <div className="p-4">
        <Link href={`/products/${product.slug}`} className="block">
          <h3 className="text-sm font-semibold text-text line-clamp-1 group-hover:text-primary transition-colors">{product.name}</h3>
        </Link>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-lg font-bold text-text">{formatCurrency(product.price)}</span>
          {product.compare_at_price ? (
            <span className="text-xs text-text-muted line-through">{formatCurrency(product.compare_at_price)}</span>
          ) : null}
        </div>
        <motion.button
          onClick={() => onAddToCart(product)}
          whileTap={{ scale: 0.95 }}
          className={cn(
            'mt-3 w-full flex items-center justify-center gap-2 rounded-xl py-2.5 text-xs font-semibold transition-all duration-300',
            isAdded
              ? 'bg-green-500 text-white'
              : 'bg-primary text-white hover:bg-primary-dark hover:shadow-md',
          )}
        >
          {isAdded ? (
            <><Check className="h-3.5 w-3.5" /> Added to Cart</>
          ) : (
            <><ShoppingCart className="h-3.5 w-3.5" /> Add to Cart</>
          )}
        </motion.button>
      </div>
    </motion.div>
  );
}

/* ── Section Renderer ── */
function SectionRenderer({ section, products, discountProducts, categories, addedId, onAddToCart }: {
  section: StoreSection;
  products: ReturnType<typeof useProductStore.getState>['products'];
  discountProducts: ReturnType<typeof useProductStore.getState>['products'];
  categories: ReturnType<typeof useProductStore.getState>['categories'];
  addedId: string | null;
  onAddToCart: (product: ReturnType<typeof useProductStore.getState>['products'][0]) => void;
}) {
  const sectionIcon: Record<string, React.ReactNode> = {
    hot_products: <Sparkles className="h-5 w-5 text-orange-500" />,
    discount: <Tag className="h-5 w-5 text-red-500" />,
    new_arrivals: <Clock className="h-5 w-5 text-blue-500" />,
    category_showcase: <ShoppingBag className="h-5 w-5 text-purple-500" />,
    campaign: <Tag className="h-5 w-5 text-green-500" />,
    custom: <Sparkles className="h-5 w-5 text-text-muted" />,
  };

  let displayProducts = products;
  if (section.type === 'discount') displayProducts = discountProducts;
  if (section.type === 'new_arrivals') displayProducts = [...products].reverse();
  if (section.type === 'hot_products') displayProducts = products.slice(0, 8);

  return (
    <motion.section
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: '-60px' }}
      variants={fadeUp}
      className="py-14"
    >
      <div className="mb-8 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-surface-hover">
            {sectionIcon[section.type]}
          </div>
          <div>
            <h2 className="text-xl font-bold text-text">{section.title}</h2>
            {section.subtitle && <p className="text-sm text-text-secondary">{section.subtitle}</p>}
          </div>
        </div>
        {section.type !== 'category_showcase' && (
          <Link href="/products" className="text-sm font-medium text-primary hover:text-primary-dark transition-colors flex items-center gap-1">
            View All <ChevronRight className="h-4 w-4" />
          </Link>
        )}
      </div>

      {section.type === 'category_showcase' ? (
        <motion.div variants={staggerContainer} initial="hidden" whileInView="visible" viewport={{ once: true }} className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {categories.filter((c) => c.status === 'active').map((cat, idx) => (
            <motion.div key={cat.id} variants={cardVariant}>
              <Link href={`/products?category=${cat.slug}`}
                className="group relative block overflow-hidden rounded-2xl p-6 text-center transition-all duration-300 hover:shadow-lg hover:-translate-y-1">
                <div className={cn('absolute inset-0 bg-gradient-to-br opacity-80 group-hover:opacity-100 transition-opacity', gradients[idx % gradients.length])} />
                <div className="relative">
                  <div className="mx-auto mb-3 flex h-16 w-16 items-center justify-center rounded-2xl bg-white/80 dark:bg-gray-800/80 text-2xl font-bold text-text shadow-sm group-hover:scale-110 transition-transform duration-300">
                    {cat.name[0]}
                  </div>
                  <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{cat.name}</h3>
                  {cat.description && <p className="mt-1 text-xs text-gray-600 dark:text-gray-300 line-clamp-2">{cat.description}</p>}
                </div>
              </Link>
            </motion.div>
          ))}
          {categories.length === 0 && (
            <p className="col-span-full text-center text-sm text-text-muted py-8">No categories yet</p>
          )}
        </motion.div>
      ) : section.type === 'campaign' ? (
        <motion.div
          whileHover={{ scale: 1.01 }}
          className="rounded-2xl bg-gradient-to-r from-primary to-primary-dark p-10 text-center text-white shadow-lg overflow-hidden relative"
        >
          <div className="absolute inset-0 opacity-10">
            <div className="absolute -right-10 -top-10 h-40 w-40 rounded-full bg-white/20" />
            <div className="absolute -left-5 -bottom-5 h-32 w-32 rounded-full bg-white/15" />
          </div>
          <div className="relative">
            <h3 className="text-3xl font-bold mb-3">{section.title}</h3>
            <p className="text-white/80 mb-6 text-lg">{section.subtitle}</p>
            <Link href="/products" className="inline-flex items-center gap-2 rounded-full bg-white px-7 py-3 text-sm font-semibold text-primary hover:shadow-xl transition-all hover:scale-105">
              <ShoppingBag className="h-4 w-4" />
              Shop Now
            </Link>
          </div>
        </motion.div>
      ) : (
        <motion.div variants={staggerContainer} initial="hidden" whileInView="visible" viewport={{ once: true }} className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {displayProducts.slice(0, 8).map((product, idx) => (
            <ProductCard
              key={product.id}
              product={product}
              gradient={gradients[idx % gradients.length]}
              addedId={addedId}
              onAddToCart={onAddToCart}
            />
          ))}
          {displayProducts.length === 0 && (
            <p className="col-span-full text-center text-sm text-text-muted py-8">No products available yet</p>
          )}
        </motion.div>
      )}
    </motion.section>
  );
}
