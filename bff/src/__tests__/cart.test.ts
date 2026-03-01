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
import { cartRoutes } from '../routes/cart.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(cookie, { secret: 'test-secret' });
  await app.register(cartRoutes);
  return app;
}

const mockCart = {
  id: 'cart-1',
  userId: 'user-1',
  items: [
    {
      id: 'item-1',
      productId: 'prod-1',
      productName: 'Test Shirt',
      priceCents: 2999,
      quantity: 2,
      imageUrl: 'https://example.com/img.jpg',
    },
  ],
  totalCents: 5998,
  currency: 'USD',
  updatedAt: '2025-01-01T00:00:00Z',
};

describe('Cart Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  // All cart routes require auth
  describe('Authentication requirement', () => {
    it('GET /api/cart returns 401 without auth', async () => {
      const res = await app.inject({ method: 'GET', url: '/api/cart' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('POST /api/cart/items returns 401 without auth', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/cart/items',
        payload: { productId: 'prod-1', quantity: 1 },
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });
  });

  describe('GET /api/cart', () => {
    it('returns 200 with cart data when authenticated', async () => {
      mockApiRequest.mockResolvedValueOnce(mockCart);

      const res = await app.inject({
        method: 'GET',
        url: '/api/cart',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockCart);
      await app.close();
    });

    it('forwards auth token to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce(mockCart);

      await app.inject({
        method: 'GET',
        url: '/api/cart',
        headers: { Authorization: 'Bearer my-cart-token' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/cart',
        expect.objectContaining({ token: 'my-cart-token' }),
      );
      await app.close();
    });

    it('returns 404 when ApiError is thrown', async () => {
      mockApiRequest.mockRejectedValueOnce(new ApiError(404, 'NOT_FOUND', 'Cart not found'));

      const res = await app.inject({
        method: 'GET',
        url: '/api/cart',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(404);
      await app.close();
    });
  });

  describe('POST /api/cart/items', () => {
    it('returns 201 with updated cart when item added', async () => {
      mockApiRequest.mockResolvedValueOnce(mockCart);

      const res = await app.inject({
        method: 'POST',
        url: '/api/cart/items',
        headers: { Authorization: 'Bearer test-token' },
        payload: { productId: 'prod-1', quantity: 2 },
      });
      expect(res.statusCode).toBe(201);
      expect(res.json()).toEqual(mockCart);
      await app.close();
    });

    it('calls gateway with POST method and body', async () => {
      mockApiRequest.mockResolvedValueOnce(mockCart);

      await app.inject({
        method: 'POST',
        url: '/api/cart/items',
        headers: { Authorization: 'Bearer tok' },
        payload: { productId: 'p-123', quantity: 3 },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/cart/items',
        expect.objectContaining({
          method: 'POST',
          body: { productId: 'p-123', quantity: 3 },
          token: 'tok',
        }),
      );
      await app.close();
    });
  });

  describe('PUT /api/cart/items/:id', () => {
    it('returns 200 with updated cart', async () => {
      const updatedCart = { ...mockCart, items: [{ ...mockCart.items[0], quantity: 5 }] };
      mockApiRequest.mockResolvedValueOnce(updatedCart);

      const res = await app.inject({
        method: 'PUT',
        url: '/api/cart/items/item-1',
        headers: { Authorization: 'Bearer test-token' },
        payload: { quantity: 5 },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(updatedCart);
      await app.close();
    });

    it('calls gateway with PUT method and item id', async () => {
      mockApiRequest.mockResolvedValueOnce(mockCart);

      await app.inject({
        method: 'PUT',
        url: '/api/cart/items/some-item-id',
        headers: { Authorization: 'Bearer tok' },
        payload: { quantity: 2 },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/cart/items/some-item-id',
        expect.objectContaining({ method: 'PUT', body: { quantity: 2 } }),
      );
      await app.close();
    });

    it('returns 409 on optimistic locking conflict', async () => {
      mockApiRequest.mockRejectedValueOnce(new ApiError(409, 'CONFLICT', 'Version conflict'));

      const res = await app.inject({
        method: 'PUT',
        url: '/api/cart/items/item-1',
        headers: { Authorization: 'Bearer test-token' },
        payload: { quantity: 3 },
      });
      expect(res.statusCode).toBe(409);
      await app.close();
    });
  });

  describe('DELETE /api/cart/items/:id', () => {
    it('returns 204 when item deleted', async () => {
      mockApiRequest.mockResolvedValueOnce(undefined);

      const res = await app.inject({
        method: 'DELETE',
        url: '/api/cart/items/item-1',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(204);
      await app.close();
    });

    it('calls gateway with DELETE method and item id', async () => {
      mockApiRequest.mockResolvedValueOnce(undefined);

      await app.inject({
        method: 'DELETE',
        url: '/api/cart/items/del-item-id',
        headers: { Authorization: 'Bearer tok' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/cart/items/del-item-id',
        expect.objectContaining({ method: 'DELETE', token: 'tok' }),
      );
      await app.close();
    });

    it('returns 404 when item not found', async () => {
      mockApiRequest.mockRejectedValueOnce(new ApiError(404, 'NOT_FOUND', 'Item not found'));

      const res = await app.inject({
        method: 'DELETE',
        url: '/api/cart/items/nonexistent',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(404);
      await app.close();
    });
  });
});
