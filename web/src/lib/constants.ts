// ─── Pagination ────────────────────────────────────────────────────────────

export const ITEMS_PER_PAGE = 12;
export const MAX_CART_QUANTITY = 99;

// ─── Order Statuses ────────────────────────────────────────────────────────

export const ORDER_STATUSES: Record<string, { label: string; color: string }> = {
  pending: { label: 'Bekliyor', color: 'bg-yellow-100 text-yellow-800' },
  confirmed: { label: 'Onaylandı', color: 'bg-blue-100 text-blue-800' },
  processing: { label: 'İşleniyor', color: 'bg-purple-100 text-purple-800' },
  shipped: { label: 'Kargoya Verildi', color: 'bg-brand-lighter text-brand' },
  delivered: { label: 'Teslim Edildi', color: 'bg-green-100 text-green-800' },
  cancelled: { label: 'İptal Edildi', color: 'bg-red-100 text-red-800' },
  refunded: { label: 'İade Edildi', color: 'bg-stone-100 text-stone-800' },
};

// ─── Checkout Steps ────────────────────────────────────────────────────────

export const CHECKOUT_STEPS = [
  { id: 'shipping', label: 'Teslimat' },
  { id: 'review', label: 'İnceleme' },
  { id: 'payment', label: 'Ödeme' },
  { id: 'confirmation', label: 'Onay' },
] as const;

export type CheckoutStepId = (typeof CHECKOUT_STEPS)[number]['id'];

// ─── Sort Options ──────────────────────────────────────────────────────────

export const SORT_OPTIONS = [
  { value: 'newest', label: 'En Yeni' },
  { value: 'price_asc', label: 'Fiyat: Artan' },
  { value: 'price_desc', label: 'Fiyat: Azalan' },
  { value: 'name_asc', label: 'İsim: A-Z' },
  { value: 'rating', label: 'En Yüksek Puan' },
] as const;

export type SortOptionValue = (typeof SORT_OPTIONS)[number]['value'];

// ─── Hero & Navigation ─────────────────────────────────────────────────────

export const HERO_AUTOPLAY_INTERVAL = 5000;
export const MEGAMENU_CLOSE_DELAY = 300;
