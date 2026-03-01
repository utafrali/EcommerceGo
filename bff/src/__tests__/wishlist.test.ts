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
import { wishlistRoutes } from '../routes/wishlist.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(cookie, { secret: 'test-secret' });
  await app.register(wishlistRoutes);
  return app;
}

const mockWishlist = {
  data: [
    { id: 'wl-1', productId: 'prod-1', createdAt: '2025-01-01T00:00:00Z' },
  ],
  total: 1,
  page: 1,
  perPage: 20,
};

describe('Wishlist Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  describe('Authentication requirement', () => {
    it('GET /api/wishlist returns 401 without auth', async () => {
      const res = await app.inject({ method: 'GET', url: '/api/wishlist' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('POST /api/wishlist/:productId returns 401 without auth', async () => {
      const res = await app.inject({ method: 'POST', url: '/api/wishlist/prod-1' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('DELETE /api/wishlist/:productId returns 401 without auth', async () => {
      const res = await app.inject({ method: 'DELETE', url: '/api/wishlist/prod-1' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });
  });

  describe('GET /api/wishlist', () => {
    it('returns 200 with wishlist data', async () => {
      mockApiRequest.mockResolvedValueOnce(mockWishlist);

      const res = await app.inject({
        method: 'GET',
        url: '/api/wishlist',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockWishlist);
      await app.close();
    });

    it('forwards auth token and pagination params', async () => {
      mockApiRequest.mockResolvedValueOnce(mockWishlist);

      await app.inject({
        method: 'GET',
        url: '/api/wishlist?page=2&per_page=10',
        headers: { Authorization: 'Bearer my-tok' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/users/wishlist',
        expect.objectContaining({
          token: 'my-tok',
          query: expect.objectContaining({ page: '2', per_page: '10' }),
        }),
      );
      await app.close();
    });

    it('returns 500 on upstream error', async () => {
      mockApiRequest.mockRejectedValueOnce(new Error('Upstream failure'));

      const res = await app.inject({
        method: 'GET',
        url: '/api/wishlist',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(500);
      await app.close();
    });
  });

  describe('POST /api/wishlist/:productId', () => {
    it('returns 201 when product added to wishlist', async () => {
      const mockResult = { id: 'wl-2', productId: 'prod-2' };
      mockApiRequest.mockResolvedValueOnce(mockResult);

      const res = await app.inject({
        method: 'POST',
        url: '/api/wishlist/prod-2',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(201);
      expect(res.json()).toEqual(mockResult);
      await app.close();
    });

    it('calls gateway with correct product id and POST method', async () => {
      mockApiRequest.mockResolvedValueOnce({ id: 'wl-3' });

      await app.inject({
        method: 'POST',
        url: '/api/wishlist/my-product-id',
        headers: { Authorization: 'Bearer tok' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/users/wishlist/my-product-id',
        expect.objectContaining({ method: 'POST', token: 'tok' }),
      );
      await app.close();
    });

    it('returns 409 when product already in wishlist', async () => {
      mockApiRequest.mockRejectedValueOnce(
        new ApiError(409, 'CONFLICT', 'Product already in wishlist'),
      );

      const res = await app.inject({
        method: 'POST',
        url: '/api/wishlist/prod-1',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(409);
      await app.close();
    });
  });

  describe('DELETE /api/wishlist/:productId', () => {
    it('returns 204 when product removed from wishlist', async () => {
      mockApiRequest.mockResolvedValueOnce(undefined);

      const res = await app.inject({
        method: 'DELETE',
        url: '/api/wishlist/prod-1',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(204);
      await app.close();
    });

    it('calls gateway with DELETE method and correct product id', async () => {
      mockApiRequest.mockResolvedValueOnce(undefined);

      await app.inject({
        method: 'DELETE',
        url: '/api/wishlist/del-product-id',
        headers: { Authorization: 'Bearer tok' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/users/wishlist/del-product-id',
        expect.objectContaining({ method: 'DELETE', token: 'tok' }),
      );
      await app.close();
    });

    it('returns 404 when product not in wishlist', async () => {
      mockApiRequest.mockRejectedValueOnce(
        new ApiError(404, 'NOT_FOUND', 'Product not in wishlist'),
      );

      const res = await app.inject({
        method: 'DELETE',
        url: '/api/wishlist/nonexistent',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(404);
      await app.close();
    });
  });

  describe('GET /api/wishlist/:productId/exists', () => {
    it('returns 401 without auth', async () => {
      const res = await app.inject({ method: 'GET', url: '/api/wishlist/prod-1/exists' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('returns 200 when checking if product is in wishlist', async () => {
      const mockResult = { exists: true };
      mockApiRequest.mockResolvedValueOnce(mockResult);

      const res = await app.inject({
        method: 'GET',
        url: '/api/wishlist/prod-1/exists',
        headers: { Authorization: 'Bearer test-token' },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockResult);
      await app.close();
    });

    it('calls gateway with correct product id', async () => {
      mockApiRequest.mockResolvedValueOnce({ exists: false });

      await app.inject({
        method: 'GET',
        url: '/api/wishlist/check-prod/exists',
        headers: { Authorization: 'Bearer tok' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/users/wishlist/check-prod',
        expect.objectContaining({ token: 'tok' }),
      );
      await app.close();
    });
  });
});
