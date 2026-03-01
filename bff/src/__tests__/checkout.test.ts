import { describe, it, expect, vi, beforeEach } from 'vitest';
import Fastify from 'fastify';
import cookie from '@fastify/cookie';
import type { FastifyInstance } from 'fastify';

vi.mock('../services/http-client.js', () => ({
  apiRequest: vi.fn(),
  ApiError: class ApiError extends Error {
    statusCode: number;
    code: string;
    constructor(statusCode: number, code: string, message: string) {
      super(message);
      this.statusCode = statusCode;
      this.code = code;
      this.name = 'ApiError';
    }
  },
}));

import { apiRequest, ApiError } from '../services/http-client.js';
import { checkoutRoutes } from '../routes/checkout.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(cookie, { secret: 'test-secret' });
  await app.register(checkoutRoutes);
  return app;
}

const mockCart = {
  data: {
    id: 'cart-1',
    user_id: 'user-1',
    items: [
      {
        product_id: 'prod-1',
        variant_id: 'var-1',
        name: 'Test Shirt',
        sku: 'SHIRT-001',
        price: 2999,
        quantity: 2,
        image_url: 'https://example.com/img.jpg',
      },
    ],
    currency: 'USD',
  },
};

const mockCheckoutFromApi = {
  id: 'checkout-1',
  user_id: 'user-1',
  status: 'pending',
  items: [
    {
      product_id: 'prod-1',
      variant_id: 'var-1',
      name: 'Test Shirt',
      sku: 'SHIRT-001',
      price: 2999,
      quantity: 2,
    },
  ],
  subtotal_amount: 5998,
  discount_amount: 0,
  shipping_amount: 500,
  total_amount: 6498,
  currency: 'USD',
  shipping_address: null,
  campaign_code: '',
  expires_at: '2025-01-02T00:00:00Z',
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

const mockCheckoutResp = { data: mockCheckoutFromApi };

describe('Checkout Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  describe('Authentication requirement', () => {
    it('POST /api/checkout returns 401 without auth', async () => {
      const res = await app.inject({ method: 'POST', url: '/api/checkout', payload: {} });
      expect(res.statusCode).toBe(401);
      await app.close();
    });
  });

  describe('POST /api/checkout', () => {
    it('returns 400 when cart is empty', async () => {
      mockApiRequest.mockResolvedValueOnce({ data: { ...mockCart.data, items: [] } });

      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout',
        headers: { Authorization: 'Bearer test-token' },
        payload: {},
      });
      expect(res.statusCode).toBe(400);
      const body = res.json();
      expect(body.error.code).toBe('EMPTY_CART');
      await app.close();
    });

    it('returns 400 when cart has no items array', async () => {
      mockApiRequest.mockResolvedValueOnce({ data: null });

      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout',
        headers: { Authorization: 'Bearer test-token' },
        payload: {},
      });
      expect(res.statusCode).toBe(400);
      await app.close();
    });

    it('returns 201 with transformed checkout session on success', async () => {
      mockApiRequest
        .mockResolvedValueOnce(mockCart)        // fetch cart
        .mockResolvedValueOnce(mockCheckoutResp); // initiate checkout

      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout',
        headers: { Authorization: 'Bearer test-token' },
        payload: {},
      });
      expect(res.statusCode).toBe(201);
      const body = res.json();
      expect(body.data.session_id).toBe('checkout-1');
      expect(body.data.status).toBe('pending');
      expect(body.data.subtotal).toBe(5998);
      expect(body.data.total).toBe(6498);
      expect(body.data.items).toHaveLength(1);
      expect(body.data.items[0].product_name).toBe('Test Shirt');
      expect(body.data.items[0].total_price).toBe(2999 * 2);
      await app.close();
    });

    it('transforms shipping_address correctly', async () => {
      const checkoutWithAddr = {
        ...mockCheckoutFromApi,
        shipping_address: {
          full_name: 'John Doe',
          address_line: '123 Main St',
          city: 'Anytown',
          state: 'CA',
          postal_code: '12345',
          country: 'US',
        },
      };
      mockApiRequest
        .mockResolvedValueOnce(mockCart)
        .mockResolvedValueOnce({ data: checkoutWithAddr });

      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout',
        headers: { Authorization: 'Bearer test-token' },
        payload: {},
      });
      expect(res.statusCode).toBe(201);
      const body = res.json();
      expect(body.data.shipping_address).toEqual({
        line1: '123 Main St',
        city: 'Anytown',
        state: 'CA',
        postal_code: '12345',
        country: 'US',
      });
      await app.close();
    });

    it('calls cart API then checkout API with correct body', async () => {
      mockApiRequest
        .mockResolvedValueOnce(mockCart)
        .mockResolvedValueOnce(mockCheckoutResp);

      await app.inject({
        method: 'POST',
        url: '/api/checkout',
        headers: { Authorization: 'Bearer tok' },
        payload: {},
      });

      expect(mockApiRequest).toHaveBeenNthCalledWith(
        1,
        '/api/v1/cart',
        expect.objectContaining({ token: 'tok' }),
      );
      expect(mockApiRequest).toHaveBeenNthCalledWith(
        2,
        '/api/v1/checkout',
        expect.objectContaining({
          method: 'POST',
          token: 'tok',
          body: expect.objectContaining({
            items: expect.arrayContaining([
              expect.objectContaining({ product_id: 'prod-1' }),
            ]),
          }),
        }),
      );
      await app.close();
    });

    it('returns 503 when cart service is down', async () => {
      mockApiRequest.mockRejectedValueOnce(
        new ApiError(503, 'SERVICE_UNAVAILABLE', 'Cart service down'),
      );

      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout',
        headers: { Authorization: 'Bearer test-token' },
        payload: {},
      });
      expect(res.statusCode).toBe(503);
      await app.close();
    });
  });

  describe('PUT /api/checkout/:sessionId/shipping', () => {
    it('returns 401 without auth', async () => {
      const res = await app.inject({
        method: 'PUT',
        url: '/api/checkout/sess-1/shipping',
        payload: {
          shipping_address: { line1: '1 Main', city: 'City', state: 'CA', postal_code: '11111', country: 'US' },
        },
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('returns 200 with updated checkout session', async () => {
      const updatedCheckout = {
        ...mockCheckoutFromApi,
        shipping_address: {
          full_name: 'Customer',
          address_line: '123 Elm St',
          city: 'Springfield',
          state: 'IL',
          postal_code: '62701',
          country: 'US',
        },
      };
      mockApiRequest.mockResolvedValueOnce({ data: updatedCheckout });

      const res = await app.inject({
        method: 'PUT',
        url: '/api/checkout/sess-1/shipping',
        headers: { Authorization: 'Bearer test-token' },
        payload: {
          shipping_address: {
            line1: '123 Elm St',
            city: 'Springfield',
            state: 'IL',
            postal_code: '62701',
            country: 'US',
          },
        },
      });
      expect(res.statusCode).toBe(200);
      const body = res.json();
      expect(body.data.session_id).toBe('checkout-1');
      expect(body.data.shipping_address.city).toBe('Springfield');
      await app.close();
    });

    it('maps address format and calls shipping endpoint', async () => {
      mockApiRequest.mockResolvedValueOnce({ data: mockCheckoutFromApi });

      await app.inject({
        method: 'PUT',
        url: '/api/checkout/sess-abc/shipping',
        headers: { Authorization: 'Bearer tok' },
        payload: {
          shipping_address: {
            line1: '1 Main',
            line2: 'Apt 5',
            city: 'Boston',
            state: 'MA',
            postal_code: '02101',
            country: 'US',
          },
        },
      });

      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/checkout/sess-abc/shipping',
        expect.objectContaining({
          method: 'PUT',
          token: 'tok',
          body: expect.objectContaining({
            address_line: '1 Main, Apt 5',
            city: 'Boston',
            state: 'MA',
            postal_code: '02101',
            country: 'US',
          }),
        }),
      );
      await app.close();
    });
  });

  describe('POST /api/checkout/:sessionId/pay', () => {
    it('returns 401 without auth', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout/sess-1/pay',
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('returns 200 with processed checkout', async () => {
      const processedCheckout = { ...mockCheckoutFromApi, status: 'completed' };
      mockApiRequest
        .mockResolvedValueOnce({})                           // set payment method
        .mockResolvedValueOnce({ data: processedCheckout }); // process

      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout/sess-1/pay',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(200);
      const body = res.json();
      expect(body.data.status).toBe('completed');
      await app.close();
    });

    it('calls payment and process endpoints in order', async () => {
      mockApiRequest
        .mockResolvedValueOnce({})
        .mockResolvedValueOnce({ data: mockCheckoutFromApi });

      await app.inject({
        method: 'POST',
        url: '/api/checkout/sess-xyz/pay',
        headers: { Authorization: 'Bearer tok' },
      });

      expect(mockApiRequest).toHaveBeenNthCalledWith(
        1,
        '/api/v1/checkout/sess-xyz/payment',
        expect.objectContaining({ method: 'PUT', body: { payment_method: 'mock_card' } }),
      );
      expect(mockApiRequest).toHaveBeenNthCalledWith(
        2,
        '/api/v1/checkout/sess-xyz/process',
        expect.objectContaining({ method: 'POST', token: 'tok' }),
      );
      await app.close();
    });

    it('returns 422 when payment fails', async () => {
      mockApiRequest.mockRejectedValueOnce(
        new ApiError(422, 'PAYMENT_FAILED', 'Payment declined'),
      );

      const res = await app.inject({
        method: 'POST',
        url: '/api/checkout/sess-1/pay',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(422);
      await app.close();
    });
  });
});
