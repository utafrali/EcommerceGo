import type { FastifyInstance } from 'fastify';
import { authMiddleware } from '../middleware/auth.js';
import { apiRequest } from '../services/http-client.js';

// ── Raw API types (from checkout service) ─────────────────────────────────

interface CartFromApi {
  id: string;
  user_id: string;
  items: {
    product_id: string;
    variant_id: string;
    name: string;
    sku: string;
    price: number;
    quantity: number;
    image_url: string;
  }[];
  currency: string;
}

interface CheckoutFromApi {
  id: string;
  user_id: string;
  status: string;
  items: {
    product_id: string;
    variant_id: string;
    name: string;
    sku: string;
    price: number;
    quantity: number;
  }[];
  subtotal_amount: number;
  discount_amount: number;
  shipping_amount: number;
  total_amount: number;
  currency: string;
  shipping_address?: {
    full_name: string;
    address_line: string;
    city: string;
    state: string;
    postal_code: string;
    country: string;
  } | null;
  campaign_code?: string;
  expires_at: string;
  created_at: string;
  updated_at: string;
}

// ── Transform to frontend-expected format ──────────────────────────────────

function transformCheckoutSession(raw: CheckoutFromApi) {
  return {
    session_id: raw.id,
    status: raw.status,
    user_id: raw.user_id,
    items: (raw.items || []).map((item) => ({
      product_id: item.product_id,
      product_name: item.name,
      quantity: item.quantity,
      unit_price: item.price,
      total_price: item.price * item.quantity,
    })),
    subtotal: raw.subtotal_amount,
    discount: raw.discount_amount,
    shipping_cost: raw.shipping_amount,
    total: raw.total_amount,
    shipping_address: raw.shipping_address
      ? {
          line1: raw.shipping_address.address_line,
          city: raw.shipping_address.city,
          state: raw.shipping_address.state,
          postal_code: raw.shipping_address.postal_code,
          country: raw.shipping_address.country,
        }
      : null,
    campaign_code: raw.campaign_code || '',
    created_at: raw.created_at,
    updated_at: raw.updated_at,
  };
}

// ── Routes ────────────────────────────────────────────────────────────────

export async function checkoutRoutes(app: FastifyInstance): Promise<void> {
  // All checkout routes require authentication
  app.addHook('preHandler', authMiddleware);

  /**
   * POST /api/checkout
   * Initiate a new checkout session from the current cart.
   * Fetches cart items and sends them to the checkout service.
   */
  app.post<{
    Body: { campaign_code?: string };
  }>('/api/checkout', async (request, reply) => {
    // 1. Fetch the user's cart to get items
    const cartResp = await apiRequest<{ data: CartFromApi }>('/api/v1/cart', {
      token: request.authToken,
    });

    const cart = cartResp?.data;
    if (!cart || !cart.items || cart.items.length === 0) {
      return reply.status(400).send({
        error: { code: 'EMPTY_CART', message: 'Your cart is empty' },
      });
    }

    // 2. Build the checkout request with cart items
    const checkoutBody = {
      items: cart.items.map((item) => ({
        product_id: item.product_id,
        variant_id: item.variant_id,
        name: item.name,
        sku: item.sku,
        price: item.price,
        quantity: item.quantity,
      })),
      currency: cart.currency || 'USD',
    };

    // 3. Initiate checkout
    const resp = await apiRequest<{ data: CheckoutFromApi }>('/api/v1/checkout', {
      method: 'POST',
      body: checkoutBody,
      token: request.authToken,
    });

    // 4. Transform and return
    return reply.status(201).send({
      data: transformCheckoutSession(resp.data),
    });
  });

  /**
   * PUT /api/checkout/:sessionId/shipping
   * Set the shipping address for a checkout session.
   * Maps frontend address format to checkout service format.
   */
  app.put<{
    Params: { sessionId: string };
    Body: {
      shipping_address: {
        line1: string;
        line2?: string;
        city: string;
        state: string;
        postal_code: string;
        country: string;
      };
    };
  }>('/api/checkout/:sessionId/shipping', async (request, reply) => {
    const { sessionId } = request.params;
    const addr = request.body.shipping_address;

    // Map to checkout service expected format
    const shippingBody = {
      full_name: 'Customer',
      address_line: addr.line1 + (addr.line2 ? ', ' + addr.line2 : ''),
      city: addr.city,
      state: addr.state,
      postal_code: addr.postal_code,
      country: addr.country,
      phone: '',
    };

    const resp = await apiRequest<{ data: CheckoutFromApi }>(
      `/api/v1/checkout/${sessionId}/shipping`,
      {
        method: 'PUT',
        body: shippingBody,
        token: request.authToken,
      },
    );

    return reply.send({
      data: transformCheckoutSession(resp.data),
    });
  });

  /**
   * POST /api/checkout/:sessionId/pay
   * Process payment for a checkout session.
   * First sets mock payment method, then processes.
   */
  app.post<{
    Params: { sessionId: string };
  }>('/api/checkout/:sessionId/pay', async (request, reply) => {
    const { sessionId } = request.params;

    // 1. Set payment method (mock)
    await apiRequest<any>(
      `/api/v1/checkout/${sessionId}/payment`,
      {
        method: 'PUT',
        body: { payment_method: 'mock_card' },
        token: request.authToken,
      },
    );

    // 2. Process the checkout
    const resp = await apiRequest<{ data: CheckoutFromApi }>(
      `/api/v1/checkout/${sessionId}/process`,
      {
        method: 'POST',
        body: {},
        token: request.authToken,
      },
    );

    return reply.send({
      data: transformCheckoutSession(resp.data),
    });
  });
}
