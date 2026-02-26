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
}

export interface Brand {
  id: string;
  name: string;
  slug: string;
  logo_url?: string;
  created_at: string;
  updated_at: string;
}

// ─── Campaign Types ────────────────────────────────────────────────────────

export interface Campaign {
  id: string;
  name: string;
  description: string;
  code: string;
  type: string;           // "percentage" | "fixed_amount"
  status: string;         // "active" | "inactive" | "expired"
  discount_value: number;
  min_order_amount: number;
  max_discount_amount: number;
  max_usage_count: number;
  current_usage_count: number;
  start_date: string;
  end_date: string;
  applicable_categories: string[];
  applicable_products: string[];
  created_at: string;
  updated_at: string;
}

// ─── Order Types ───────────────────────────────────────────────────────────

export interface OrderItem {
  id: string;
  order_id: string;
  product_id: string;
  variant_id: string;
  name: string;
  sku: string;
  price: number;      // in cents
  quantity: number;
}

export interface Order {
  id: string;
  user_id: string;
  status: string;
  items: OrderItem[];
  subtotal_amount: number;
  discount_amount: number;
  shipping_amount: number;
  total_amount: number;
  currency: string;
  shipping_address: {
    full_name: string;
    address_line: string;
    city: string;
    state: string;
    postal_code: string;
    country: string;
  };
  billing_address: {
    full_name: string;
    address_line: string;
    city: string;
    state: string;
    postal_code: string;
    country: string;
  };
  created_at: string;
  updated_at: string;
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

// ─── Inventory Types ───────────────────────────────────────────────────────

export interface InventoryStock {
  product_id: string;
  variant_id: string;
  quantity: number;
  reserved_quantity: number;
  available_quantity: number;
  updated_at: string;
}

export interface LowStockItem {
  product_id: string;
  variant_id: string;
  sku: string;
  quantity: number;
  threshold: number;
}

// ─── Admin Request Types ───────────────────────────────────────────────────

export interface LoginRequest {
  email: string;
  password: string;
}

export interface CreateProductRequest {
  name: string;
  slug: string;
  description: string;
  brand_id?: string;
  category_id?: string;
  status: string;
  base_price: number;
  currency?: string;
  metadata?: Record<string, unknown>;
}

export interface UpdateProductRequest {
  name?: string;
  slug?: string;
  description?: string;
  brand_id?: string;
  category_id?: string;
  status?: string;
  base_price?: number;
  currency?: string;
  metadata?: Record<string, unknown>;
}

export interface CreateCampaignRequest {
  name: string;
  description?: string;
  code: string;
  type: string;
  status?: string;
  discount_value: number;
  min_order_amount?: number;
  max_discount_amount?: number;
  max_usage_count?: number;
  start_date: string;
  end_date: string;
  applicable_categories?: string[];
  applicable_products?: string[];
}

export interface UpdateCampaignRequest {
  name?: string;
  description?: string;
  code?: string;
  type?: string;
  status?: string;
  discount_value?: number;
  min_order_amount?: number;
  max_discount_amount?: number;
  max_usage_count?: number;
  start_date?: string;
  end_date?: string;
  applicable_categories?: string[];
  applicable_products?: string[];
}

export interface UpdateOrderStatusRequest {
  status: string;
}

// ─── Query Parameter Types ─────────────────────────────────────────────────

export interface ProductListParams {
  page?: number;
  per_page?: number;
  category_id?: string;
  brand_id?: string;
  search?: string;
  min_price?: number;
  max_price?: number;
  status?: string;
  sort?: string;
}

export interface OrderListParams {
  page?: number;
  per_page?: number;
  status?: string;
  user_id?: string;
}

// ─── Dashboard Types ───────────────────────────────────────────────────────

export interface DashboardStats {
  total_products: number;
  total_orders: number;
  active_campaigns: number;
  low_stock_items: number;
}
