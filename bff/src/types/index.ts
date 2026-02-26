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

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
}

export interface AuthResponse {
  data: {
    user: User;
    tokens: AuthTokens;
  };
}

// ─── Category ──────────────────────────────────────────────────────────────

export interface Category {
  id: string;
  name: string;
  slug: string;
  parentId?: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CategoryListResponse {
  categories: Category[];
}

// ─── Brand ─────────────────────────────────────────────────────────────────

export interface Brand {
  id: string;
  name: string;
  slug: string;
  description?: string;
  logoUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface BrandListResponse {
  brands: Brand[];
}

// ─── Review ────────────────────────────────────────────────────────────────

export interface Review {
  id: string;
  productId: string;
  userId: string;
  rating: number;
  title: string;
  body: string;
  createdAt: string;
  updatedAt: string;
}

export interface ReviewListResponse {
  reviews: Review[];
  total: number;
  page: number;
  perPage: number;
}

export interface CreateReviewRequest {
  rating: number;
  title: string;
  body: string;
}

// ─── Campaign ──────────────────────────────────────────────────────────────

export interface Campaign {
  id: string;
  name: string;
  code: string;
  discountType: string;
  discountValue: number;
  minOrderCents?: number;
  maxDiscountCents?: number;
  startsAt: string;
  endsAt: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CampaignListResponse {
  campaigns: Campaign[];
}

export interface ValidateCampaignRequest {
  code: string;
}

export interface ValidateCampaignResponse {
  valid: boolean;
  campaign?: Campaign;
  message?: string;
}

// ─── Checkout ──────────────────────────────────────────────────────────────

export interface CheckoutSession {
  id: string;
  userId: string;
  status: string;
  cartSnapshot: CartItem[];
  totalCents: number;
  currency: string;
  shippingAddress?: Address;
  campaignCode?: string;
  discountCents?: number;
  createdAt: string;
  updatedAt: string;
}

export interface InitiateCheckoutRequest {
  cartId: string;
  campaignCode?: string;
}

export interface SetShippingRequest {
  shippingAddress: Address;
}

export interface ProcessPaymentRequest {
  paymentMethodId: string;
}

export interface CheckoutResponse {
  session: CheckoutSession;
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

// ─── Fastify augmentation ───────────────────────────────────────────────────

declare module 'fastify' {
  interface FastifyRequest {
    userId?: string;
    authToken?: string;
  }
}
