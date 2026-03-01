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
import { orderRoutes } from '../routes/orders.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(cookie, { secret: 'test-secret' });
  await app.register(orderRoutes);
  return app;
}

const mockOrder = {
  id: 'order-1',
  userId: 'user-1',
  status: 'confirmed',
  items: [
    {
      id: 'oi-1',
      productId: 'prod-1',
      productName: 'Test Shirt',
      priceCents: 2999,
      quantity: 2,
    },
  ],
  totalCents: 5998,
  currency: 'USD',
  shippingAddress: {
    line1: '123 Main St',
    city: 'Anytown',
    state: 'CA',
    postalCode: '12345',
    country: 'US',
  },
  createdAt: '2025-01-01T00:00:00Z',
  updatedAt: '2025-01-01T00:00:00Z',
};

const mockOrderList = {
  orders: [mockOrder],
  total: 1,
  page: 1,
  pageSize: 20,
};

describe('Order Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  describe('Authentication requirement', () => {
    it('GET /api/orders returns 401 without auth', async () => {
      const res = await app.inject({ method: 'GET', url: '/api/orders' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('GET /api/orders/:id returns 401 without auth', async () => {
      const res = await app.inject({ method: 'GET', url: '/api/orders/order-1' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });
  });

  describe('GET /api/orders', () => {
    it('returns 200 with order list when authenticated', async () => {
      mockApiRequest.mockResolvedValueOnce(mockOrderList);

      const res = await app.inject({
        method: 'GET',
        url: '/api/orders',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockOrderList);
      await app.close();
    });

    it('forwards pagination params to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce(mockOrderList);

      await app.inject({
        method: 'GET',
        url: '/api/orders?page=2&pageSize=10',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/orders',
        expect.objectContaining({
          query: expect.objectContaining({ page: '2', page_size: '10' }),
        }),
      );
      await app.close();
    });

    it('forwards auth token to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce(mockOrderList);

      await app.inject({
        method: 'GET',
        url: '/api/orders',
        headers: { Authorization: 'Bearer my-order-token' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/orders',
        expect.objectContaining({ token: 'my-order-token' }),
      );
      await app.close();
    });

    it('returns 500 on upstream error', async () => {
      mockApiRequest.mockRejectedValueOnce(new Error('Connection failed'));

      const res = await app.inject({
        method: 'GET',
        url: '/api/orders',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(500);
      await app.close();
    });
  });

  describe('GET /api/orders/:id', () => {
    it('returns 200 with single order', async () => {
      mockApiRequest.mockResolvedValueOnce(mockOrder);

      const res = await app.inject({
        method: 'GET',
        url: '/api/orders/order-1',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockOrder);
      await app.close();
    });

    it('calls gateway with correct order id path', async () => {
      mockApiRequest.mockResolvedValueOnce(mockOrder);

      await app.inject({
        method: 'GET',
        url: '/api/orders/my-order-id',
        headers: { Authorization: 'Bearer tok' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/orders/my-order-id',
        expect.objectContaining({ token: 'tok' }),
      );
      await app.close();
    });

    it('returns 404 when order not found', async () => {
      mockApiRequest.mockRejectedValueOnce(new ApiError(404, 'NOT_FOUND', 'Order not found'));

      const res = await app.inject({
        method: 'GET',
        url: '/api/orders/nonexistent',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(404);
      expect(res.json()).toEqual({ error: { code: 'NOT_FOUND', message: 'Order not found' } });
      await app.close();
    });

    it('returns 403 when user does not own the order', async () => {
      mockApiRequest.mockRejectedValueOnce(new ApiError(403, 'FORBIDDEN', 'Access denied'));

      const res = await app.inject({
        method: 'GET',
        url: '/api/orders/other-user-order',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(403);
      await app.close();
    });
  });

  describe('POST /api/orders', () => {
    it('returns 401 without auth', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/orders',
        payload: {
          shippingAddress: { line1: '123 Main', city: 'City', state: 'CA', postalCode: '12345', country: 'US' },
          paymentMethodId: 'pm-1',
        },
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('returns 201 with created order', async () => {
      mockApiRequest.mockResolvedValueOnce(mockOrder);

      const res = await app.inject({
        method: 'POST',
        url: '/api/orders',
        headers: { Authorization: 'Bearer test-token' },
        payload: {
          shippingAddress: { line1: '123 Main', city: 'City', state: 'CA', postalCode: '12345', country: 'US' },
          paymentMethodId: 'pm-1',
        },
      });
      expect(res.statusCode).toBe(201);
      expect(res.json()).toEqual(mockOrder);
      await app.close();
    });

    it('calls gateway with POST and body', async () => {
      mockApiRequest.mockResolvedValueOnce(mockOrder);
      const body = {
        shippingAddress: { line1: '1 Main', city: 'Town', state: 'TX', postalCode: '77777', country: 'US' },
        paymentMethodId: 'pm-xyz',
      };

      await app.inject({
        method: 'POST',
        url: '/api/orders',
        headers: { Authorization: 'Bearer tok' },
        payload: body,
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/orders',
        expect.objectContaining({ method: 'POST', body, token: 'tok' }),
      );
      await app.close();
    });
  });
});
