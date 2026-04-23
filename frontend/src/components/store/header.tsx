'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ShoppingCart, Search, User, Menu, X, LogOut, Package, ChevronDown, Heart } from 'lucide-react';
import { useCartStore } from '@/stores/cart';
import { useAuthStore } from '@/stores/auth';
import { useProductStore } from '@/stores/products';
import { useWishlistStore } from '@/stores/wishlist';
import { useState, useRef, useEffect } from 'react';
import ThemeSwitcher from '@/components/ui/theme-switcher';
import { cn } from '@/lib/utils';

interface StoreHeaderProps {
  storeName?: string;
  logoUrl?: string;
}

export default function StoreHeader({ storeName = 'Saajan', logoUrl }: StoreHeaderProps) {
  const router = useRouter();
  const itemCount = useCartStore((s) => s.itemCount());
  const wishlistCount = useWishlistStore((s) => s.items.length);
  const { user, isAuthenticated, logout } = useAuthStore();
  const categories = useProductStore((s) => s.categories);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchFocused, setSearchFocused] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => { setMounted(true); }, []);

  const authenticated = mounted && isAuthenticated();
  const initials = user
    ? `${user.first_name[0] || ''}${user.last_name[0] || ''}`.toUpperCase()
    : '';

  const navCategories = categories.filter((c) => c.status === 'active').slice(0, 4);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false);
      }
    }
    if (userMenuOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [userMenuOpen]);

  function handleSignOut() {
    logout();
    setUserMenuOpen(false);
    router.push('/');
  }

  return (
    <header className="sticky top-0 z-50 bg-surface/95 backdrop-blur-md border-b border-border">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-16 items-center gap-4">

          {/* Logo */}
          <Link href="/" className="flex items-center gap-2.5 flex-shrink-0">
            {logoUrl ? (
              <img src={logoUrl} alt={storeName} className="h-8 w-auto max-w-[120px] object-contain" />
            ) : (
              <div className="h-9 w-9 rounded-xl bg-primary flex items-center justify-center shadow-sm">
                <span className="text-white font-bold">{storeName[0]}</span>
              </div>
            )}
            <span className="text-lg font-semibold text-text hidden sm:block">{storeName}</span>
          </Link>

          {/* Nav links */}
          <nav className="hidden lg:flex items-center gap-1 ml-4">
            <Link href="/products" className="rounded-lg px-3 py-1.5 text-sm font-medium text-text-secondary hover:text-text hover:bg-surface-hover transition-colors">
              All Products
            </Link>
            {navCategories.map((cat) => (
              <Link key={cat.id} href={`/products?category=${cat.slug}`}
                className="rounded-lg px-3 py-1.5 text-sm font-medium text-text-secondary hover:text-text hover:bg-surface-hover transition-colors">
                {cat.name}
              </Link>
            ))}
          </nav>

          {/* Search — grows to fill space */}
          <div className="hidden sm:block flex-1 max-w-sm ml-auto">
            <div className="relative">
              <Search className={cn(
                'absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 transition-colors',
                searchFocused ? 'text-primary' : 'text-text-muted',
              )} />
              <input
                type="text"
                placeholder="Search products..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={() => setSearchFocused(true)}
                onBlur={() => setSearchFocused(false)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && searchQuery.trim()) {
                    router.push(`/products?search=${encodeURIComponent(searchQuery.trim())}`);
                    setSearchQuery('');
                  }
                }}
                className={cn(
                  'w-full rounded-full border bg-surface-secondary py-2 pl-10 pr-4 text-sm text-text transition-all duration-200 placeholder:text-text-muted outline-none',
                  searchFocused
                    ? 'border-primary ring-2 ring-primary/20 bg-surface'
                    : 'border-transparent hover:border-border hover:bg-surface-hover',
                )}
              />
            </div>
          </div>

          {/* Right actions */}
          <div className="flex items-center gap-1 flex-shrink-0">
            <ThemeSwitcher compact />

            <Link href="/wishlist" className="relative p-2 rounded-lg text-text-secondary hover:text-text hover:bg-surface-hover transition-colors">
              <Heart className="h-5 w-5" />
              {mounted && wishlistCount > 0 && (
                <span className="absolute -right-0.5 -top-0.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-accent px-1 text-[10px] font-bold text-white">
                  {wishlistCount}
                </span>
              )}
            </Link>

            <Link href="/cart" className="relative p-2 rounded-lg text-text-secondary hover:text-text hover:bg-surface-hover transition-colors">
              <ShoppingCart className="h-5 w-5" />
              {mounted && itemCount > 0 && (
                <span className="absolute -right-0.5 -top-0.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-accent px-1 text-[10px] font-bold text-white">
                  {itemCount}
                </span>
              )}
            </Link>

            {authenticated && user ? (
              <div className="relative" ref={userMenuRef}>
                <button
                  onClick={() => setUserMenuOpen(!userMenuOpen)}
                  className="flex items-center gap-1.5 rounded-lg p-1.5 text-text-secondary transition-colors hover:bg-surface-hover"
                >
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-xs font-bold text-white">
                    {initials}
                  </div>
                  <ChevronDown className={cn('hidden h-3.5 w-3.5 text-text-muted transition-transform sm:block', userMenuOpen && 'rotate-180')} />
                </button>

                {userMenuOpen && (
                  <div className="absolute right-0 top-full mt-2 w-52 rounded-xl border border-border bg-surface py-1 shadow-lg">
                    <div className="border-b border-border px-4 py-3">
                      <p className="text-sm font-semibold text-text truncate">
                        {user.first_name} {user.last_name}
                      </p>
                      <p className="text-xs text-text-muted truncate">{user.email}</p>
                    </div>
                    <div className="py-1">
                      <Link href="/account" onClick={() => setUserMenuOpen(false)}
                        className="flex items-center gap-3 px-4 py-2.5 text-sm text-text-secondary hover:bg-surface-hover hover:text-text transition-colors">
                        <User className="h-4 w-4 text-text-muted" /> My Account
                      </Link>
                      <Link href="/account/orders" onClick={() => setUserMenuOpen(false)}
                        className="flex items-center gap-3 px-4 py-2.5 text-sm text-text-secondary hover:bg-surface-hover hover:text-text transition-colors">
                        <Package className="h-4 w-4 text-text-muted" /> My Orders
                      </Link>
                    </div>
                    <div className="border-t border-border py-1">
                      <button onClick={handleSignOut}
                        className="flex w-full items-center gap-3 px-4 py-2.5 text-sm text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors">
                        <LogOut className="h-4 w-4" /> Sign Out
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex items-center gap-1.5 ml-1">
                <Link href="/login" className="hidden sm:block rounded-lg px-3 py-1.5 text-sm font-medium text-text-secondary hover:text-text hover:bg-surface-hover transition-colors">
                  Sign In
                </Link>
                <Link href="/register" className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-dark transition-colors">
                  Register
                </Link>
              </div>
            )}

            <button onClick={() => setMobileMenuOpen(!mobileMenuOpen)} className="p-2 rounded-lg text-text-secondary hover:bg-surface-hover lg:hidden transition-colors">
              {mobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile menu */}
      {mobileMenuOpen && (
        <div className="border-t border-border bg-surface px-4 py-4 lg:hidden">
          {/* Mobile search */}
          <div className="mb-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-muted" />
              <input type="text" placeholder="Search products..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && searchQuery.trim()) {
                    router.push(`/products?search=${encodeURIComponent(searchQuery.trim())}`);
                    setSearchQuery('');
                    setMobileMenuOpen(false);
                  }
                }}
                className="w-full rounded-full border border-border bg-surface-secondary py-2.5 pl-10 pr-4 text-sm text-text placeholder:text-text-muted outline-none focus:border-primary focus:ring-2 focus:ring-primary/20" />
            </div>
          </div>

          <nav className="space-y-1">
            <Link href="/products" onClick={() => setMobileMenuOpen(false)}
              className="block rounded-lg px-3 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover hover:text-text transition-colors">
              All Products
            </Link>
            {navCategories.map((cat) => (
              <Link key={cat.id} href={`/products?category=${cat.slug}`} onClick={() => setMobileMenuOpen(false)}
                className="block rounded-lg px-3 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover hover:text-text transition-colors">
                {cat.name}
              </Link>
            ))}
          </nav>

          {authenticated && user ? (
            <div className="mt-4 space-y-1 border-t border-border pt-4">
              <Link href="/account" onClick={() => setMobileMenuOpen(false)}
                className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover hover:text-text transition-colors">
                <User className="h-4 w-4" /> My Account
              </Link>
              <Link href="/account/orders" onClick={() => setMobileMenuOpen(false)}
                className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover hover:text-text transition-colors">
                <Package className="h-4 w-4" /> My Orders
              </Link>
              <button onClick={() => { handleSignOut(); setMobileMenuOpen(false); }}
                className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors">
                <LogOut className="h-4 w-4" /> Sign Out
              </button>
            </div>
          ) : (
            <div className="mt-4 space-y-2 border-t border-border pt-4">
              <Link href="/login" onClick={() => setMobileMenuOpen(false)}
                className="block rounded-lg px-3 py-2.5 text-sm font-medium text-text-secondary hover:bg-surface-hover transition-colors">
                Sign In
              </Link>
              <Link href="/register" onClick={() => setMobileMenuOpen(false)}
                className="block rounded-lg bg-primary px-3 py-2.5 text-center text-sm font-medium text-white hover:bg-primary-dark transition-colors">
                Create Account
              </Link>
            </div>
          )}
        </div>
      )}
    </header>
  );
}
