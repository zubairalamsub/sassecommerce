import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatCurrency(amount: number, currency = 'BDT'): string {
  return new Intl.NumberFormat('en-BD', {
    style: 'currency',
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(amount);
}

export function formatDate(date: string): string {
  return new Date(date).toLocaleDateString('en-BD', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function formatDateTime(date: string): string {
  return new Date(date).toLocaleString('en-BD', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Build a full media URL from a relative path stored in the database.
 *
 * DB stores:   "products/abc123.jpg"
 * Docker:      "/api/media/products/abc123.jpg"  (NEXT_PUBLIC_MEDIA_URL unset or "/api/media")
 * Cloud (S3):  "https://cdn.yourstore.com/products/abc123.jpg"  (set NEXT_PUBLIC_MEDIA_URL)
 *
 * Pass-through for absolute URLs (http/https/data:) so existing external URLs still work.
 */
const MEDIA_BASE = process.env.NEXT_PUBLIC_MEDIA_URL || '/api/media';

export function mediaUrl(relativePath: string | undefined | null): string {
  if (!relativePath) return '';
  // Already an absolute URL or data URI — return as-is
  if (relativePath.startsWith('http://') || relativePath.startsWith('https://') || relativePath.startsWith('data:')) {
    return relativePath;
  }
  // Strip leading slash if present
  const clean = relativePath.startsWith('/') ? relativePath.slice(1) : relativePath;
  const base = MEDIA_BASE.endsWith('/') ? MEDIA_BASE.slice(0, -1) : MEDIA_BASE;
  return `${base}/${clean}`;
}

export function statusColor(status: string): string {
  const colors: Record<string, string> = {
    active: 'bg-green-100 text-green-800',
    pending: 'bg-yellow-100 text-yellow-800',
    confirmed: 'bg-blue-100 text-blue-800',
    shipped: 'bg-purple-100 text-purple-800',
    delivered: 'bg-green-100 text-green-800',
    cancelled: 'bg-red-100 text-red-800',
    suspended: 'bg-red-100 text-red-800',
    inactive: 'bg-gray-100 text-gray-800',
    draft: 'bg-gray-100 text-gray-800',
    archived: 'bg-gray-100 text-gray-800',
    Completed: 'bg-green-100 text-green-800',
    Pending: 'bg-yellow-100 text-yellow-800',
    Failed: 'bg-red-100 text-red-800',
    Cancelled: 'bg-red-100 text-red-800',
    Refunded: 'bg-orange-100 text-orange-800',
  };
  return colors[status] || 'bg-gray-100 text-gray-800';
}
