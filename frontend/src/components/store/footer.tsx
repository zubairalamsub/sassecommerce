'use client';

import Link from 'next/link';
import { useEffect } from 'react';
import { ExternalLink, ArrowUp } from 'lucide-react';
import { useStoreConfigStore } from '@/stores/store-config';

const TENANT_ID = 'tenant_saajan';

interface StoreFooterProps {
  storeName?: string;
}

export default function StoreFooter({ storeName = 'Saajan' }: StoreFooterProps) {
  const { config, fetchConfig } = useStoreConfigStore();
  const footer = config.footer;

  useEffect(() => {
    fetchConfig(TENANT_ID);
  }, [fetchConfig]);

  return (
    <footer className="border-t border-border bg-surface mt-auto">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {/* Brand / About */}
          <div>
            <div className="flex items-center gap-2 mb-4">
              <div className="h-8 w-8 rounded-lg bg-primary flex items-center justify-center">
                <span className="text-white font-bold text-sm">{storeName[0]}</span>
              </div>
              <span className="text-lg font-semibold text-text">{storeName}</span>
            </div>
            <p className="text-sm text-text-secondary leading-relaxed">
              {footer.about_text}
            </p>
            {(footer.social_facebook || footer.social_instagram || footer.social_youtube) && (
              <div className="mt-4 flex gap-2">
                {footer.social_facebook && (
                  <a href={footer.social_facebook} target="_blank" rel="noopener noreferrer"
                    className="flex h-9 items-center gap-1.5 rounded-lg bg-surface-hover px-3 text-xs font-medium text-text-muted hover:bg-primary hover:text-white transition-all duration-200">
                    <ExternalLink className="h-3.5 w-3.5" /> Facebook
                  </a>
                )}
                {footer.social_instagram && (
                  <a href={footer.social_instagram} target="_blank" rel="noopener noreferrer"
                    className="flex h-9 items-center gap-1.5 rounded-lg bg-surface-hover px-3 text-xs font-medium text-text-muted hover:bg-primary hover:text-white transition-all duration-200">
                    <ExternalLink className="h-3.5 w-3.5" /> Instagram
                  </a>
                )}
                {footer.social_youtube && (
                  <a href={footer.social_youtube} target="_blank" rel="noopener noreferrer"
                    className="flex h-9 items-center gap-1.5 rounded-lg bg-surface-hover px-3 text-xs font-medium text-text-muted hover:bg-primary hover:text-white transition-all duration-200">
                    <ExternalLink className="h-3.5 w-3.5" /> YouTube
                  </a>
                )}
              </div>
            )}
          </div>

          {/* Shop Links */}
          <div>
            <h3 className="text-sm font-semibold text-text mb-3">Shop</h3>
            <ul className="space-y-2">
              {footer.shop_links.map((link, i) => (
                <li key={i}>
                  <Link href={link.href} className="text-sm text-text-secondary hover:text-primary transition-colors">
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Account */}
          <div>
            <h3 className="text-sm font-semibold text-text mb-3">Account</h3>
            <ul className="space-y-2">
              <li><Link href="/login" className="text-sm text-text-secondary hover:text-primary transition-colors">Sign In</Link></li>
              <li><Link href="/cart" className="text-sm text-text-secondary hover:text-primary transition-colors">Cart</Link></li>
              <li><Link href="/account/orders" className="text-sm text-text-secondary hover:text-primary transition-colors">Orders</Link></li>
            </ul>
          </div>

          {/* Support / Contact */}
          <div>
            <h3 className="text-sm font-semibold text-text mb-3">Support</h3>
            <ul className="space-y-2">
              {footer.contact_email && (
                <li><span className="text-sm text-text-secondary">{footer.contact_email}</span></li>
              )}
              {footer.contact_phone && (
                <li><span className="text-sm text-text-secondary">{footer.contact_phone}</span></li>
              )}
              {footer.contact_address && (
                <li><span className="text-sm text-text-secondary">{footer.contact_address}</span></li>
              )}
              {footer.support_links.map((link, i) => (
                <li key={i}>
                  <Link href={link.href} className="text-sm text-text-secondary hover:text-primary transition-colors">
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div className="mt-8 border-t border-border pt-8 flex items-center justify-between">
          <p className="text-sm text-text-muted">
            &copy; {new Date().getFullYear()} {footer.copyright_text}
          </p>
          <button
            onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
            className="flex h-9 w-9 items-center justify-center rounded-lg bg-surface-hover text-text-muted hover:bg-primary hover:text-white transition-all duration-200"
            title="Back to top"
          >
            <ArrowUp className="h-4 w-4" />
          </button>
        </div>
      </div>
    </footer>
  );
}
