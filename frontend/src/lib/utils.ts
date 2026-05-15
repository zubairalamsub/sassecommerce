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

// Minimal HTML sanitizer for rich-text content saved by the admin editor.
// Strips <script>/<style>/<iframe>/<object>/<embed> tags, on*= event handlers,
// and javascript: URLs in href/src. Anything else is kept verbatim.
export function sanitizeHtml(html: string): string {
  if (!html) return '';
  let out = html;
  out = out.replace(/<\/?(script|style|iframe|object|embed|link|meta)\b[^>]*>/gi, '');
  out = out.replace(/\son[a-z]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)/gi, '');
  out = out.replace(/(href|src)\s*=\s*("\s*javascript:[^"]*"|'\s*javascript:[^']*'|javascript:[^\s>]+)/gi, '$1="#"');
  return out;
}

// Strip all HTML tags and decode common entities — used for snippets/cards
// where rendering full HTML would break the layout.
export function stripHtml(html: string): string {
  if (!html) return '';
  return html
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/\s+/g, ' ')
    .trim();
}

export function statusColor(status: string): string {
  const colors: Record<string, string> = {
    active: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
    confirmed: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
    shipped: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
    delivered: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
    cancelled: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
    suspended: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
    inactive: 'bg-surface-hover text-text-secondary',
    draft: 'bg-surface-hover text-text-secondary',
    archived: 'bg-surface-hover text-text-secondary',
    Completed: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
    Pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
    Failed: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
    Cancelled: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
    Refunded: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
  };
  return colors[status] || 'bg-surface-hover text-text-secondary';
}
