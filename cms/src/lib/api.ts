import type {
  ApiResponse,
  ApiListResponse,
  Product,
  ProductListParams,
  CreateProductRequest,
  UpdateProductRequest,
  Category,
  Brand,
  Campaign,
  CreateCampaignRequest,
  UpdateCampaignRequest,
  Order,
  OrderListParams,
  UpdateOrderStatusRequest,
  InventoryStock,
  LowStockItem,
  AuthResponse,
  LoginRequest,
  User,
} from '@/types';

// ─── Configuration ─────────────────────────────────────────────────────────

// Use the Next.js rewrite proxy (/gateway/* → gateway /api/v1/*) so browser
// requests stay same-origin and avoid CORS issues.  Falls back to direct
// gateway URL for SSR / non-browser contexts.
const GATEWAY_URL =
  typeof window !== 'undefined' ? '' : (process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080');
const API_PREFIX = typeof window !== 'undefined' ? '/gateway' : '/api/v1';

// ─── Token Management ──────────────────────────────────────────────────────

let memoryToken: string | null = null;

function getToken(): string | null {
  if (memoryToken) return memoryToken;
  if (typeof window !== 'undefined') {
    const stored = localStorage.getItem('cms_auth_token');
    if (stored) {
      memoryToken = stored;
      return stored;
    }
  }
  return null;
}

export function setToken(token: string): void {
  memoryToken = token;
  if (typeof window !== 'undefined') {
    localStorage.setItem('cms_auth_token', token);
  }
}

export function clearToken(): void {
  memoryToken = null;
  if (typeof window !== 'undefined') {
    localStorage.removeItem('cms_auth_token');
  }
}

// ─── Error Class ───────────────────────────────────────────────────────────

export class AdminApiError extends Error {
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
    this.name = 'AdminApiError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}

// ─── Core Request Helper ───────────────────────────────────────────────────

async function request<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const url = `${GATEWAY_URL}${API_PREFIX}${path}`;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  };

  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(url, {
    ...options,
    headers: { ...headers, ...(options?.headers as Record<string, string>) },
  });

  if (!res.ok) {
    const errorBody = await res
      .json()
      .catch(() => ({ error: { code: 'UNKNOWN', message: res.statusText } }));
    throw new AdminApiError(
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

// ─── Helper: Build Query String ────────────────────────────────────────────

function buildQueryString(params?: Record<string, unknown>): string {
  if (!params) return '';
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      qs.set(key, String(value));
    }
  });
  const str = qs.toString();
  return str ? `?${str}` : '';
}

// ─── Auth ──────────────────────────────────────────────────────────────────

export const authApi = {
  async login(data: LoginRequest): Promise<AuthResponse> {
    // API returns { data: { user, tokens: { access_token, refresh_token } } }
    const response = await request<{
      data: {
        user: User;
        tokens: { access_token: string; refresh_token: string };
      };
    }>('/auth/login', { method: 'POST', body: JSON.stringify(data) });

    return {
      user: response.data.user,
      access_token: response.data.tokens.access_token,
      refresh_token: response.data.tokens.refresh_token,
    };
  },

  async getMe(): Promise<User> {
    const response = await request<ApiResponse<User>>('/users/me');
    return response.data;
  },
};

// ─── Products ──────────────────────────────────────────────────────────────

export const productsApi = {
  async list(params?: ProductListParams): Promise<ApiListResponse<Product>> {
    return request<ApiListResponse<Product>>(
      `/products${buildQueryString(params as Record<string, unknown>)}`,
    );
  },

  async get(id: string): Promise<Product> {
    const response = await request<ApiResponse<Product>>(
      `/products/${encodeURIComponent(id)}`,
    );
    return response.data;
  },

  async create(data: CreateProductRequest): Promise<Product> {
    const response = await request<ApiResponse<Product>>('/products', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.data;
  },

  async update(id: string, data: UpdateProductRequest): Promise<Product> {
    const response = await request<ApiResponse<Product>>(
      `/products/${encodeURIComponent(id)}`,
      { method: 'PUT', body: JSON.stringify(data) },
    );
    return response.data;
  },

  async delete(id: string): Promise<void> {
    await request<void>(`/products/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },
};

// ─── Categories ────────────────────────────────────────────────────────────

export const categoriesApi = {
  async list(): Promise<Category[]> {
    const response = await request<ApiResponse<Category[]>>('/categories');
    return response.data;
  },
};

// ─── Brands ────────────────────────────────────────────────────────────────

export const brandsApi = {
  async list(): Promise<Brand[]> {
    const response = await request<ApiResponse<Brand[]>>('/brands');
    return response.data;
  },
};

// ─── Campaigns ─────────────────────────────────────────────────────────────

export const campaignsApi = {
  async list(): Promise<ApiListResponse<Campaign>> {
    return request<ApiListResponse<Campaign>>('/campaigns');
  },

  async get(id: string): Promise<Campaign> {
    const response = await request<ApiResponse<Campaign>>(
      `/campaigns/${encodeURIComponent(id)}`,
    );
    return response.data;
  },

  async create(data: CreateCampaignRequest): Promise<Campaign> {
    const response = await request<ApiResponse<Campaign>>('/campaigns', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    return response.data;
  },

  async update(id: string, data: UpdateCampaignRequest): Promise<Campaign> {
    const response = await request<ApiResponse<Campaign>>(
      `/campaigns/${encodeURIComponent(id)}`,
      { method: 'PUT', body: JSON.stringify(data) },
    );
    return response.data;
  },

  async delete(id: string): Promise<void> {
    await request<void>(`/campaigns/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },
};

// ─── Orders ────────────────────────────────────────────────────────────────

export const ordersApi = {
  async list(params?: OrderListParams): Promise<ApiListResponse<Order>> {
    return request<ApiListResponse<Order>>(
      `/orders${buildQueryString(params as Record<string, unknown>)}`,
    );
  },

  async get(id: string): Promise<Order> {
    const response = await request<ApiResponse<Order>>(
      `/orders/${encodeURIComponent(id)}`,
    );
    return response.data;
  },

  async updateStatus(id: string, data: UpdateOrderStatusRequest): Promise<Order> {
    const response = await request<ApiResponse<Order>>(
      `/orders/${encodeURIComponent(id)}/status`,
      { method: 'PUT', body: JSON.stringify(data) },
    );
    return response.data;
  },
};

// ─── Inventory ─────────────────────────────────────────────────────────────

export const inventoryApi = {
  async getStock(productId: string, variantId: string): Promise<InventoryStock> {
    const response = await request<ApiResponse<InventoryStock>>(
      `/inventory/${encodeURIComponent(productId)}/variants/${encodeURIComponent(variantId)}`,
    );
    return response.data;
  },

  async checkAvailability(items: { product_id: string; variant_id: string; quantity: number }[]): Promise<any> {
    return request<any>('/inventory/check', {
      method: 'POST',
      body: JSON.stringify({ items }),
    });
  },

  async lowStock(): Promise<LowStockItem[]> {
    const response = await request<ApiResponse<LowStockItem[]>>('/inventory/low-stock');
    return response.data || [];
  },
};
