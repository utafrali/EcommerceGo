// ─── Product ────────────────────────────────────────────────────────────────

export interface Product {
  id: string;
  name: string;
  slug: string;
  description: string;
  priceCents: number;
  currency: string;
  sku: string;
  categoryId: string;
  imageUrls: string[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ProductListResponse {
  products: Product[];
  total: number;
  page: number;
  pageSize: number;
}

// ─── Cart ───────────────────────────────────────────────────────────────────

export interface CartItem {
  id: string;
  productId: string;
  productName: string;
  priceCents: number;
  quantity: number;
  imageUrl: string;
}

export interface Cart {
  id: string;
  userId: string;
  items: CartItem[];
  totalCents: number;
  currency: string;
  updatedAt: string;
}

export interface AddCartItemRequest {
  productId: string;
  quantity: number;
}

export interface UpdateCartItemRequest {
  quantity: number;
}

// ─── Order ──────────────────────────────────────────────────────────────────

export interface OrderItem {
  id: string;
  productId: string;
  productName: string;
  priceCents: number;
  quantity: number;
}

export interface Order {
  id: string;
  userId: string;
  status: string;
  items: OrderItem[];
  totalCents: number;
  currency: string;
  shippingAddress: Address;
  createdAt: string;
  updatedAt: string;
}

export interface CreateOrderRequest {
  shippingAddress: Address;
  paymentMethodId: string;
}

export interface OrderListResponse {
  orders: Order[];
  total: number;
  page: number;
  pageSize: number;
}

// ─── User / Auth ────────────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  createdAt: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  firstName: string;
  lastName: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  user: User;
  accessToken: string;
  refreshToken: string;
}

// ─── Search ─────────────────────────────────────────────────────────────────

export interface SearchResult {
  products: Product[];
  total: number;
  query: string;
  page: number;
  pageSize: number;
}

// ─── Shared ─────────────────────────────────────────────────────────────────

export interface Address {
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postalCode: string;
  country: string;
}

export interface ApiError {
  error: {
    code: string;
    message: string;
  };
}
