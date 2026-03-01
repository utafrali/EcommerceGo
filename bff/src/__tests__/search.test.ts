import { describe, it, expect, vi, beforeEach } from 'vitest';
import Fastify from 'fastify';
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
import { searchRoutes } from '../routes/search.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(searchRoutes);
  return app;
}

describe('Search Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  describe('GET /api/search', () => {
    it('returns 200 with search results', async () => {
      const mockData = { products: [], total: 0, query: 'shirt', page: 1, pageSize: 20 };
      mockApiRequest.mockResolvedValueOnce(mockData);

      const res = await app.inject({ method: 'GET', url: '/api/search?q=shirt' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockData);
      await app.close();
    });

    it('forwards q param to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce({ products: [], total: 0 });

      await app.inject({ method: 'GET', url: '/api/search?q=dress' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/search',
        expect.objectContaining({ query: expect.objectContaining({ q: 'dress' }) }),
      );
      await app.close();
    });

    it('forwards pageSize as per_page to gateway', async () => {
      mockApiRequest.mockResolvedValueOnce({ products: [] });

      await app.inject({ method: 'GET', url: '/api/search?q=test&pageSize=15' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/search',
        expect.objectContaining({
          query: expect.objectContaining({ per_page: '15' }),
        }),
      );
      await app.close();
    });

    it('forwards per_page directly when provided', async () => {
      mockApiRequest.mockResolvedValueOnce({ products: [] });

      await app.inject({ method: 'GET', url: '/api/search?q=test&per_page=25' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/search',
        expect.objectContaining({
          query: expect.objectContaining({ per_page: '25' }),
        }),
      );
      await app.close();
    });

    it('forwards category_id filter', async () => {
      mockApiRequest.mockResolvedValueOnce({ products: [] });

      await app.inject({ method: 'GET', url: '/api/search?category_id=cat-1' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/search',
        expect.objectContaining({
          query: expect.objectContaining({ category_id: 'cat-1' }),
        }),
      );
      await app.close();
    });

    it('uses category as category_id fallback', async () => {
      mockApiRequest.mockResolvedValueOnce({ products: [] });

      await app.inject({ method: 'GET', url: '/api/search?category=shirts' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/search',
        expect.objectContaining({
          query: expect.objectContaining({ category_id: 'shirts' }),
        }),
      );
      await app.close();
    });

    it('forwards minPrice as min_price and maxPrice as max_price', async () => {
      mockApiRequest.mockResolvedValueOnce({ products: [] });

      await app.inject({ method: 'GET', url: '/api/search?minPrice=100&maxPrice=500' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/search',
        expect.objectContaining({
          query: expect.objectContaining({ min_price: '100', max_price: '500' }),
        }),
      );
      await app.close();
    });

    it('returns 500 on unexpected error', async () => {
      mockApiRequest.mockRejectedValueOnce(new Error('Connection refused'));

      const res = await app.inject({ method: 'GET', url: '/api/search?q=test' });
      expect(res.statusCode).toBe(500);
      await app.close();
    });
  });

  describe('GET /api/search/suggest', () => {
    it('returns 200 with suggestions', async () => {
      const mockData = { suggestions: ['shirt', 'shoes'] };
      mockApiRequest.mockResolvedValueOnce(mockData);

      const res = await app.inject({ method: 'GET', url: '/api/search/suggest?q=sh' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockData);
      await app.close();
    });

    it('forwards q and limit params', async () => {
      mockApiRequest.mockResolvedValueOnce({ suggestions: [] });

      await app.inject({ method: 'GET', url: '/api/search/suggest?q=bl&limit=5' });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/search/suggest',
        expect.objectContaining({ query: expect.objectContaining({ q: 'bl', limit: '5' }) }),
      );
      await app.close();
    });

    it('returns 503 when ApiError is thrown', async () => {
      mockApiRequest.mockRejectedValueOnce(
        new ApiError(503, 'SERVICE_UNAVAILABLE', 'Search service down'),
      );

      const res = await app.inject({ method: 'GET', url: '/api/search/suggest?q=test' });
      expect(res.statusCode).toBe(503);
      await app.close();
    });
  });
});
