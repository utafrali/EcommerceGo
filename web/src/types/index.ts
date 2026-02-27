// ─── API Wrapper Types ─────────────────────────────────────────────────────

/** Standard API response wrapping a single resource. */
export interface ApiResponse<T> {
  data: T;
}

/** Standard API response wrapping a paginated list. */
export interface ApiListResponse<T> {
  data: T[];
  total_count: number;
  page: number;
  per_page: number;
  total_pages: number;
}

/** Standard API error envelope. */
export interface ApiError {
  error: {
    code: string;
    message: string;
    fields?: Record<string, string>;
  };
}

// ─── Product Types ─────────────────────────────────────────────────────────

export interface Product {
  id: string;
  name: string;
  slug: string;
  description: string;
  brand_id: string | null;
  category_id: string | null;
  status: string;
  base_price: number;
  currency: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  // Present on list responses (single primary image)
  primary_image?: ProductImage | null;
  // Optional enriched fields (present on single-product responses)
  images?: ProductImage[];
  variants?: ProductVariant[];
  category?: Category | null;
  brand?: Brand | null;
}

export interface ProductImage {
  id: string;
  product_id: string;
  url: string;
  alt_text: string;
  sort_order: number;
  is_primary: boolean;
  created_at: string;
}

export interface ProductVariant {
  id: string;
  product_id: string;
  sku: string;
  name: string;
  price: number | null;
  stock_quantity: number;
  attributes: Record<string, string>;
  weight_grams: number | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Category {
  id: string;
  name: string;
  slug: string;
  parent_id: string | null;
  sort_order: number;
  is_active: boolean;
  image_url?: string | null;
  icon_url?: string | null;
  description?: string | null;
  level?: number;
  product_count?: number;
  children?: Category[];
}

export interface Brand {
  id: string;
  name: string;
  slug: string;
  logo_url: string;
}

// ─── Review Types ──────────────────────────────────────────────────────────

export interface Review {
  id: string;
  product_id: string;
  user_id: string;
  rating: number;
  title: string;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface ReviewSummary {
  average_rating: number;
  total_count: number;
}

export interface ReviewListResponse extends ApiListResponse<Review> {
  summary: ReviewSummary;
}

// ─── Cart Types ────────────────────────────────────────────────────────────

export interface CartItem {
  product_id: string;
  quantity: number;
}

export interface Cart {
  user_id: string;
  items: CartItem[];
  updated_at: string;
}

// ─── Order Types ───────────────────────────────────────────────────────────

export interface OrderItem {
  id: string;
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

export interface Order {
  id: string;
  user_id: string;
  status: string;
  items: OrderItem[];
  total_amount: number;
  currency: string;
  shipping_address: Address;
  created_at: string;
  updated_at: string;
}

// ─── Campaign Types ────────────────────────────────────────────────────────

export interface Campaign {
  id: string;
  name: string;
  code: string;
  type: string;
  discount_value: number;
  min_order_amount: number;
  max_uses: number;
  current_uses: number;
  is_active: boolean;
  starts_at: string;
  ends_at: string;
}

// ─── Checkout Types ────────────────────────────────────────────────────────

export interface CheckoutSession {
  session_id: string;
  status: string;
  user_id: string;
  items: CheckoutItem[];
  subtotal: number;
  discount: number;
  shipping_cost: number;
  total: number;
  shipping_address: Address | null;
  campaign_code: string;
  created_at: string;
  updated_at: string;
}

export interface CheckoutItem {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

// ─── User / Auth Types ─────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  created_at: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
}

// ─── Shared Types ──────────────────────────────────────────────────────────

export interface Address {
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
}

// ─── Request Types ─────────────────────────────────────────────────────────

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
}

export interface AddCartItemRequest {
  product_id: string;
  variant_id: string;
  name: string;
  sku: string;
  price: number;
  quantity: number;
  image_url?: string;
}

export interface UpdateCartItemRequest {
  quantity: number;
}

export interface CreateReviewRequest {
  rating: number;
  title: string;
  body: string;
}

export interface InitiateCheckoutRequest {
  campaign_code?: string;
}

export interface SetShippingRequest {
  shipping_address: Address;
}

// ─── Banner Types ─────────────────────────────────────────────────────────

export interface Banner {
  id: string;
  title: string;
  subtitle?: string | null;
  image_url: string;
  link_url: string;
  link_type: 'internal' | 'external';
  position: 'hero_slider' | 'mid_banner' | 'category_banner';
  sort_order: number;
  is_active: boolean;
  starts_at?: string | null;
  ends_at?: string | null;
  created_at: string;
  updated_at: string;
}

// ─── Wishlist Types ───────────────────────────────────────────────────────

export interface WishlistItem {
  user_id: string;
  product_id: string;
  created_at: string;
}

// ─── Query Parameter Types ─────────────────────────────────────────────────

export interface ProductListParams {
  page?: number;
  per_page?: number;
  category_id?: string | string[];  // Support multi-select via array or comma-separated string
  brand_id?: string | string[];     // Support multi-select via array or comma-separated string
  search?: string;
  min_price?: number;
  max_price?: number;
  status?: string;
  sort?: string;
}
