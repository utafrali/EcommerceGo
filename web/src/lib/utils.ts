import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

// ─── Tailwind Class Merger ─────────────────────────────────────────────────

/**
 * Merge Tailwind CSS class names, resolving conflicts intelligently.
 * Combines clsx (conditional classes) with tailwind-merge (conflict resolution).
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(...inputs));
}

// ─── Price Formatting ──────────────────────────────────────────────────────

/**
 * Format a price stored in cents to a display string.
 * @param cents - Price in smallest currency unit (e.g. 7999 = $79.99)
 * @param currency - ISO 4217 currency code (default: "USD")
 */
export function formatPrice(cents: number, currency = 'USD'): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
  }).format(cents / 100);
}

// ─── Date Formatting ───────────────────────────────────────────────────────

/**
 * Format an ISO date string to a human-readable date.
 * Example: "January 15, 2025"
 */
export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

/**
 * Format an ISO date string to a relative time description.
 * Examples: "Today", "Yesterday", "3 days ago", "2 weeks ago"
 */
export function formatRelativeTime(dateStr: string): string {
  const now = new Date();
  const date = new Date(dateStr);
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffDays === 0) return 'Today';
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays} days ago`;
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
  if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`;
  return formatDate(dateStr);
}

// ─── String Utilities ──────────────────────────────────────────────────────

/**
 * Convert a string to a URL-safe slug.
 * Example: "Hello World!" -> "hello-world"
 */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '');
}

/**
 * Truncate text to a maximum length, adding ellipsis if truncated.
 */
export function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength).trimEnd() + '...';
}

// ─── Product Helpers ───────────────────────────────────────────────────────

/**
 * Get the primary image URL from a product, falling back to a placeholder.
 */
export function getProductImageUrl(
  product: {
    images?: { url: string; is_primary: boolean }[];
    primary_image?: { url: string } | null;
  },
  fallback = '/placeholder-product.svg',
): string {
  // Check primary_image first (present on list responses)
  if (product.primary_image?.url) return product.primary_image.url;
  // Then check images array (present on detail responses)
  if (!product.images || product.images.length === 0) return fallback;
  const primary = product.images.find((img) => img.is_primary);
  return primary?.url || product.images[0].url;
}

// ─── Discount Calculation ──────────────────────────────────────────────────

/**
 * Calculate the discount amount in cents.
 * @param originalPrice - Original price in cents.
 * @param discountType - "percentage" or "fixed".
 * @param discountValue - The discount value (percentage points or cents).
 * @returns Discount amount in cents.
 */
export function calculateDiscount(
  originalPrice: number,
  discountType: string,
  discountValue: number,
): number {
  if (discountType === 'percentage') {
    return Math.round((originalPrice * discountValue) / 100);
  }
  if (discountType === 'fixed') {
    return Math.min(discountValue, originalPrice);
  }
  return 0;
}

// ─── Rating Helpers ────────────────────────────────────────────────────────

export type StarType = 'full' | 'half' | 'empty';

/**
 * Generate a 5-element array representing a star rating for display.
 * @param rating - Numeric rating (0-5).
 */
export function getStarRating(rating: number): StarType[] {
  const stars: StarType[] = [];
  for (let i = 1; i <= 5; i++) {
    if (rating >= i) stars.push('full');
    else if (rating >= i - 0.5) stars.push('half');
    else stars.push('empty');
  }
  return stars;
}
