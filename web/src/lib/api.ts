import type {
  ApiResponse,
  ApiListResponse,
  Product,
  ProductListParams,
  Category,
  Brand,
  Banner,
  WishlistItem,
  Review,
  ReviewListResponse,
  CreateReviewRequest,
  LoginRequest,
  RegisterRequest,
  User,
  Cart,
  AddCartItemRequest,
  Order,
  CheckoutSession,
  InitiateCheckoutRequest,
  SetShippingRequest,
  Campaign,
} from '@/types';

// ─── Configuration ─────────────────────────────────────────────────────────

const BFF_URL = process.env.NEXT_PUBLIC_BFF_URL || 'http://localhost:3001';
const INTERNAL_BFF_URL = process.env.BFF_INTERNAL_URL || BFF_URL;

/**
 * Returns the appropriate base URL depending on execution context.
 * Server-side (SSR / RSC): uses the internal Docker-resolvable URL.
 * Client-side (browser): uses the public BFF URL.
 */
function getBaseUrl(): string {
  if (typeof window === 'undefined') return INTERNAL_BFF_URL;
  return BFF_URL;
}

// ─── Error Class ───────────────────────────────────────────────────────────

export class ApiRequestError extends Error {
  public readonly status: number;
  public readonly code: string;
  public readonly fields?: Record<string, string>;

  constructor(
    status: number,
    code: string,
    message: string,
    fields?: Record<string, string>,
  ) {
    super(message);
    this.name = 'ApiRequestError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}

// ─── API Client ────────────────────────────────────────────────────────────

export class ApiClient {
  private baseUrl: string;
  private token?: string;

  constructor(token?: string) {
    this.baseUrl = getBaseUrl();
    this.token = token;
  }

  /**
   * Low-level request helper. Adds auth headers, handles errors, and
   * deserialises the JSON response.
   */
  private async request<T>(
    path: string,
    options?: RequestInit,
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const res = await fetch(url, {
      ...options,
      headers: { ...headers, ...(options?.headers as Record<string, string>) },
      credentials: 'include',
    });

    if (!res.ok) {
      const errorBody = await res
        .json()
        .catch(() => ({ error: { code: 'UNKNOWN', message: res.statusText } }));
      throw new ApiRequestError(
        res.status,
        errorBody?.error?.code || 'UNKNOWN',
        errorBody?.error?.message || res.statusText,
        errorBody?.error?.fields,
      );
    }

    // 204 No Content
    if (res.status === 204) {
      return undefined as T;
    }

    return res.json() as Promise<T>;
  }

  // ── Products ───────────────────────────────────────────────────────────

  async getProducts(params?: ProductListParams) {
    const qs = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          qs.set(key, String(value));
        }
      });
    }
    const query = qs.toString();
    return this.request<ApiListResponse<Product>>(
      `/api/products${query ? '?' + query : ''}`,
    );
  }

  async getProduct(idOrSlug: string) {
    return this.request<ApiResponse<Product>>(
      `/api/products/${encodeURIComponent(idOrSlug)}`,
    );
  }

  async getCategories() {
    return this.request<ApiResponse<Category[]>>('/api/categories');
  }

  async getBrands() {
    return this.request<ApiResponse<Brand[]>>('/api/brands');
  }

  // ── Reviews ────────────────────────────────────────────────────────────

  async getProductReviews(productId: string, page = 1) {
    return this.request<ReviewListResponse>(
      `/api/products/${productId}/reviews?page=${page}`,
    );
  }

  async createReview(productId: string, data: CreateReviewRequest) {
    return this.request<ApiResponse<Review>>(
      `/api/products/${productId}/reviews`,
      { method: 'POST', body: JSON.stringify(data) },
    );
  }

  // ── Auth ───────────────────────────────────────────────────────────────

  async login(data: LoginRequest) {
    return this.request<ApiResponse<User>>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async register(data: RegisterRequest) {
    return this.request<ApiResponse<User>>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getMe() {
    return this.request<ApiResponse<User>>('/api/auth/me');
  }

  // ── Cart ───────────────────────────────────────────────────────────────

  async getCart() {
    return this.request<ApiResponse<Cart>>('/api/cart');
  }

  async addToCart(data: AddCartItemRequest) {
    return this.request<ApiResponse<Cart>>('/api/cart/items', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateCartItem(productId: string, quantity: number) {
    return this.request<ApiResponse<Cart>>(
      `/api/cart/items/${productId}`,
      { method: 'PUT', body: JSON.stringify({ quantity }) },
    );
  }

  async removeFromCart(productId: string) {
    return this.request<ApiResponse<Cart>>(
      `/api/cart/items/${productId}`,
      { method: 'DELETE' },
    );
  }

  // ── Orders ─────────────────────────────────────────────────────────────

  async getOrders(page = 1) {
    return this.request<ApiListResponse<Order>>(
      `/api/orders?page=${page}`,
    );
  }

  async getOrder(id: string) {
    return this.request<ApiResponse<Order>>(`/api/orders/${id}`);
  }

  // ── Checkout ───────────────────────────────────────────────────────────

  async initiateCheckout(data?: InitiateCheckoutRequest) {
    return this.request<ApiResponse<CheckoutSession>>('/api/checkout', {
      method: 'POST',
      body: JSON.stringify(data || {}),
    });
  }

  async setShipping(sessionId: string, data: SetShippingRequest) {
    return this.request<ApiResponse<CheckoutSession>>(
      `/api/checkout/${sessionId}/shipping`,
      { method: 'PUT', body: JSON.stringify(data) },
    );
  }

  async processPayment(sessionId: string) {
    return this.request<ApiResponse<CheckoutSession>>(
      `/api/checkout/${sessionId}/pay`,
      { method: 'POST', body: JSON.stringify({}) },
    );
  }

  // ── Campaigns ──────────────────────────────────────────────────────────

  async getCampaigns() {
    return this.request<ApiListResponse<Campaign>>('/api/campaigns');
  }

  async validateCoupon(code: string) {
    return this.request<ApiResponse<Campaign>>('/api/campaigns/validate', {
      method: 'POST',
      body: JSON.stringify({ code }),
    });
  }

  // ── Search ─────────────────────────────────────────────────────────────

  async search(params: {
    q?: string;
    page?: number;
    per_page?: number;
    category_id?: string;
    brand_id?: string;
    min_price?: number;
    max_price?: number;
    sort?: string;
    status?: string;
  }) {
    const qs = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        qs.set(key, String(value));
      }
    });
    const query = qs.toString();

    // Search API returns { data: { products: [], total, page, per_page } }
    const raw = await this.request<any>(
      `/api/search${query ? '?' + query : ''}`,
    );

    // Normalize to ApiListResponse<Product> format
    const searchData = raw?.data || raw;
    const rawProducts = searchData?.products || [];
    const total = searchData?.total || 0;
    const pg = searchData?.page || 1;
    const perPage = searchData?.per_page || 20;

    // Map search-specific fields to Product format
    // Search results have `image_url` (flat string) but ProductCard expects `primary_image`
    const products = rawProducts.map((p: any) => ({
      ...p,
      primary_image: p.image_url ? { url: p.image_url } : null,
    }));

    return {
      data: products,
      total_count: total,
      page: pg,
      per_page: perPage,
      total_pages: Math.ceil(total / perPage),
    } as ApiListResponse<Product>;
  }

  async searchSuggest(query: string, limit = 5) {
    return this.request<{ data: { suggestions: string[] } }>(
      `/api/search/suggest?q=${encodeURIComponent(query)}&limit=${limit}`,
    );
  }

  // ── Banners ─────────────────────────────────────────────────────────────

  async getBanners(params?: { position?: string; is_active?: string }) {
    const qs = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) qs.set(key, value);
      });
    }
    const query = qs.toString();
    return this.request<{ data: Banner[]; total_count: number }>(
      `/api/banners${query ? '?' + query : ''}`,
    );
  }

  // ── Category Tree ───────────────────────────────────────────────────────

  async getCategoryTree() {
    return this.request<ApiResponse<Category[]>>('/api/categories?tree=true');
  }

  // ── Wishlist ────────────────────────────────────────────────────────────

  async getWishlist(page = 1) {
    return this.request<ApiResponse<{ items: WishlistItem[]; total: number; page: number; per_page: number }>>(
      `/api/wishlist?page=${page}`,
    );
  }

  async addToWishlist(productId: string) {
    return this.request<ApiResponse<{ product_id: string }>>(
      `/api/wishlist/${productId}`,
      { method: 'POST' },
    );
  }

  async removeFromWishlist(productId: string) {
    return this.request<ApiResponse<{ product_id: string }>>(
      `/api/wishlist/${productId}`,
      { method: 'DELETE' },
    );
  }

  async wishlistExists(productId: string) {
    return this.request<ApiResponse<{ exists: boolean }>>(
      `/api/wishlist/${productId}`,
    );
  }
}

// ─── Convenience Exports ───────────────────────────────────────────────────

/** Default singleton API client (no auth token). */
export const api = new ApiClient();

/** Create an authenticated API client. */
export function createApiClient(token?: string): ApiClient {
  return new ApiClient(token);
}
