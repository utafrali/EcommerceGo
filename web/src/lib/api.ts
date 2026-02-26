import type {
  ApiResponse,
  ApiListResponse,
  Product,
  ProductListParams,
  Category,
  Brand,
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

  async search(query: string, page = 1) {
    return this.request<ApiListResponse<Product>>(
      `/api/search?q=${encodeURIComponent(query)}&page=${page}`,
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
