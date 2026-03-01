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
import { productRoutes } from '../routes/products.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(cookie, { secret: 'test-secret' });
  await app.register(productRoutes);
  return app;
}

const mockProduct = {
  id: 'prod-1',
  name: 'Test Shirt',
  slug: 'test-shirt',
  description: 'A test shirt',
  priceCents: 2999,
  currency: 'USD',
  sku: 'SHIRT-001',
  categoryId: 'cat-1',
  imageUrls: [],
  isActive: true,
  createdAt: '2025-01-01T00:00:00Z',
  updatedAt: '2025-01-01T00:00:00Z',
};

const mockProductList = {
  products: [mockProduct],
  total: 1,
  page: 1,
  pageSize: 20,
};

describe('Product Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  describe('GET /api/products', () => {
    it('returns 200 with product list', async () => {
      mockApiRequest.mockResolvedValueOnce(mockProductList);

      const res = await app.inject({ method: 'GET', url: '/api/products' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockProductList);
      await app.close();
    });

    it('forwards query parameters to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce(mockProductList);

      await app.inject({
        method: 'GET',
        url: '/api/products?page=2&pageSize=10&category=shirts&sort=price_asc',
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/products',
        expect.objectContaining({
          query: expect.objectContaining({
            page: '2',
            page_size: '10',
            category: 'shirts',
            sort: 'price_asc',
          }),
        }),
      );
      await app.close();
    });

    it('forwards price filter params', async () => {
      mockApiRequest.mockResolvedValueOnce(mockProductList);

      await app.inject({
        method: 'GET',
        url: '/api/products?min_price=1000&max_price=5000',
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/products',
        expect.objectContaining({
          query: expect.objectContaining({ min_price: '1000', max_price: '5000' }),
        }),
      );
      await app.close();
    });

    it('returns 500 on upstream error', async () => {
      mockApiRequest.mockRejectedValueOnce(new Error('Upstream error'));

      const res = await app.inject({ method: 'GET', url: '/api/products' });
      expect(res.statusCode).toBe(500);
      await app.close();
    });
  });

  describe('GET /api/products/:slug', () => {
    it('returns 200 with single product', async () => {
      mockApiRequest.mockResolvedValueOnce(mockProduct);

      const res = await app.inject({ method: 'GET', url: '/api/products/test-shirt' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockProduct);
      await app.close();
    });

    it('calls gateway with correct slug path', async () => {
      mockApiRequest.mockResolvedValueOnce(mockProduct);

      await app.inject({ method: 'GET', url: '/api/products/my-slug' });
      expect(mockApiRequest).toHaveBeenCalledWith('/api/v1/products/my-slug');
      await app.close();
    });

    it('returns 404 when product not found', async () => {
      mockApiRequest.mockRejectedValueOnce(new ApiError(404, 'NOT_FOUND', 'Product not found'));

      const res = await app.inject({ method: 'GET', url: '/api/products/nonexistent' });
      expect(res.statusCode).toBe(404);
      expect(res.json()).toEqual({ error: { code: 'NOT_FOUND', message: 'Product not found' } });
      await app.close();
    });
  });

  describe('GET /api/products/:id/reviews', () => {
    it('returns 200 with reviews list', async () => {
      const mockReviews = { reviews: [], total: 0, page: 1, perPage: 20 };
      mockApiRequest.mockResolvedValueOnce(mockReviews);

      const res = await app.inject({ method: 'GET', url: '/api/products/prod-1/reviews' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockReviews);
      await app.close();
    });

    it('forwards page and per_page to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce({ reviews: [] });

      await app.inject({ method: 'GET', url: '/api/products/prod-1/reviews?page=2&per_page=5' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/products/prod-1/reviews',
        expect.objectContaining({
          query: expect.objectContaining({ page: '2', per_page: '5' }),
        }),
      );
      await app.close();
    });
  });

  describe('POST /api/products/:id/reviews', () => {
    it('returns 401 when not authenticated', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/products/prod-1/reviews',
        payload: { rating: 5, title: 'Great', body: 'Loved it' },
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('returns 201 with created review when authenticated', async () => {
      const mockReview = {
        id: 'rev-1',
        productId: 'prod-1',
        userId: 'user-1',
        rating: 5,
        title: 'Great',
        body: 'Loved it',
        createdAt: '2025-01-01T00:00:00Z',
        updatedAt: '2025-01-01T00:00:00Z',
      };
      mockApiRequest.mockResolvedValueOnce(mockReview);

      const res = await app.inject({
        method: 'POST',
        url: '/api/products/prod-1/reviews',
        headers: { Authorization: 'Bearer test-token' },
        payload: { rating: 5, title: 'Great', body: 'Loved it' },
      });
      expect(res.statusCode).toBe(201);
      expect(res.json()).toEqual(mockReview);
      await app.close();
    });

    it('forwards auth token to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce({ id: 'rev-1' });

      await app.inject({
        method: 'POST',
        url: '/api/products/prod-1/reviews',
        headers: { Authorization: 'Bearer my-token' },
        payload: { rating: 4, title: 'Good', body: 'Nice product' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/products/prod-1/reviews',
        expect.objectContaining({ token: 'my-token', method: 'POST' }),
      );
      await app.close();
    });
  });

  describe('GET /api/categories', () => {
    it('returns 200 with category list', async () => {
      const mockCats = { categories: [{ id: 'cat-1', name: 'Shirts', slug: 'shirts', createdAt: '', updatedAt: '' }] };
      mockApiRequest.mockResolvedValueOnce(mockCats);

      const res = await app.inject({ method: 'GET', url: '/api/categories' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockCats);
      await app.close();
    });
  });

  describe('GET /api/brands', () => {
    it('returns 200 with brand list', async () => {
      const mockBrands = { brands: [{ id: 'b1', name: 'Nike', slug: 'nike', createdAt: '', updatedAt: '' }] };
      mockApiRequest.mockResolvedValueOnce(mockBrands);

      const res = await app.inject({ method: 'GET', url: '/api/brands' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockBrands);
      await app.close();
    });
  });
});
