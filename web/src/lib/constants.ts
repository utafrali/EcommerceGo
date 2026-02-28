// ─── Pagination ────────────────────────────────────────────────────────────

export const ITEMS_PER_PAGE = 12;
export const MAX_CART_QUANTITY = 99;

// ─── Order Statuses ────────────────────────────────────────────────────────

export const ORDER_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: 'Pending', color: 'bg-yellow-100 text-yellow-800' },
  confirmed: { label: 'Confirmed', color: 'bg-blue-100 text-blue-800' },
  processing: { label: 'Processing', color: 'bg-purple-100 text-purple-800' },
  shipped: { label: 'Shipped', color: 'bg-brand-lighter text-brand' },
  delivered: { label: 'Delivered', color: 'bg-green-100 text-green-800' },
  cancelled: { label: 'Cancelled', color: 'bg-red-100 text-red-800' },
  refunded: { label: 'Refunded', color: 'bg-stone-100 text-stone-800' },
};

// ─── Checkout Steps ────────────────────────────────────────────────────────

export const CHECKOUT_STEPS = [
  { id: 'shipping', label: 'Shipping' },
  { id: 'review', label: 'Review' },
  { id: 'payment', label: 'Payment' },
  { id: 'confirmation', label: 'Confirmation' },
] as const;

export type CheckoutStepId = (typeof CHECKOUT_STEPS)[number]['id'];

// ─── Sort Options ──────────────────────────────────────────────────────────

export const SORT_OPTIONS = [
  { value: 'newest', label: 'Newest' },
  { value: 'price_asc', label: 'Price: Low to High' },
  { value: 'price_desc', label: 'Price: High to Low' },
  { value: 'name_asc', label: 'Name: A to Z' },
  { value: 'rating', label: 'Highest Rated' },
] as const;

export type SortOptionValue = (typeof SORT_OPTIONS)[number]['value'];

// ─── Hero & Navigation ─────────────────────────────────────────────────────

export const HERO_AUTOPLAY_INTERVAL = 5000;
export const MEGAMENU_CLOSE_DELAY = 300;
