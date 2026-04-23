'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { Heart, ShoppingCart, Trash2, Check } from 'lucide-react';
import { useWishlistStore } from '@/stores/wishlist';
import { useCartStore } from '@/stores/cart';
import { formatCurrency, cn } from '@/lib/utils';

const gradients = [
  'from-rose-100 to-pink-200 dark:from-rose-900/40 dark:to-pink-900/40',
  'from-blue-100 to-indigo-200 dark:from-blue-900/40 dark:to-indigo-900/40',
  'from-emerald-100 to-teal-200 dark:from-emerald-900/40 dark:to-teal-900/40',
  'from-amber-100 to-orange-200 dark:from-amber-900/40 dark:to-orange-900/40',
  'from-violet-100 to-purple-200 dark:from-violet-900/40 dark:to-purple-900/40',
];

export default function WishlistPage() {
  const { items, removeItem, clear } = useWishlistStore();
  const addCartItem = useCartStore((s) => s.addItem);
  const [addedId, setAddedId] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => { setMounted(true); }, []);

  function handleAddToCart(item: typeof items[0]) {
    addCartItem({
      productId: item.productId,
      name: item.name,
      sku: '',
      price: item.price,
      quantity: 1,
    });
    setAddedId(item.productId);
    setTimeout(() => setAddedId(null), 1500);
  }

  if (!mounted) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-text">My Wishlist</h1>
        <p className="mt-2 text-text-secondary">Loading...</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-text">My Wishlist</h1>
          <p className="mt-1 text-text-secondary">
            {items.length} item{items.length !== 1 ? 's' : ''} saved
          </p>
        </div>
        {items.length > 0 && (
          <button onClick={clear}
            className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-text-secondary hover:bg-surface-hover transition-colors">
            Clear All
          </button>
        )}
      </div>

      {items.length === 0 ? (
        <div className="rounded-2xl border border-border bg-surface py-16 text-center">
          <Heart className="mx-auto h-12 w-12 text-text-muted" />
          <p className="mt-4 text-lg font-medium text-text">Your wishlist is empty</p>
          <p className="mt-1 text-sm text-text-secondary">Save items you love for later</p>
          <Link href="/products"
            className="mt-6 inline-block rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white hover:bg-primary-dark transition-colors">
            Browse Products
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {items.map((item, index) => {
            const isAdded = addedId === item.productId;
            return (
              <div key={item.productId}
                className="group rounded-2xl border border-border bg-surface overflow-hidden transition-all duration-300 hover:shadow-lg hover:-translate-y-1">
                <Link href={`/products/${item.slug}`}>
                  <div className={cn('relative h-52 bg-gradient-to-br flex items-center justify-center overflow-hidden', gradients[index % gradients.length])}>
                    {item.image ? (
                      <img src={item.image} alt={item.name} className="h-full w-full object-cover group-hover:scale-110 transition-transform duration-500" />
                    ) : (
                      <span className="text-5xl font-bold text-white/30">{item.name[0]}</span>
                    )}
                  </div>
                </Link>
                <div className="p-4">
                  <Link href={`/products/${item.slug}`}>
                    <h3 className="text-sm font-semibold text-text line-clamp-1 group-hover:text-primary transition-colors">{item.name}</h3>
                  </Link>
                  <p className="mt-2 text-lg font-bold text-text">{formatCurrency(item.price)}</p>
                  <div className="mt-3 flex gap-2">
                    <button onClick={() => handleAddToCart(item)} disabled={isAdded}
                      className={cn('flex-1 flex items-center justify-center gap-2 rounded-xl py-2.5 text-xs font-semibold transition-all duration-300',
                        isAdded ? 'bg-green-500 text-white' : 'bg-primary text-white hover:bg-primary-dark')}>
                      {isAdded ? <><Check className="h-3.5 w-3.5" /> Added!</> : <><ShoppingCart className="h-3.5 w-3.5" /> Add to Cart</>}
                    </button>
                    <button onClick={() => removeItem(item.productId)}
                      className="flex items-center justify-center rounded-xl border border-border px-3 text-text-muted hover:text-red-500 hover:border-red-300 transition-colors">
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
