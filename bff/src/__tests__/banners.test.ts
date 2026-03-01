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
import { bannerRoutes } from '../routes/banners.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(bannerRoutes);
  return app;
}

describe('Banner Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  it('GET /api/banners returns 200 with banner data', async () => {
    const mockData = { data: [{ id: '1', title: 'Banner 1', position: 'hero' }] };
    mockApiRequest.mockResolvedValueOnce(mockData);

    const res = await app.inject({ method: 'GET', url: '/api/banners' });
    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual(mockData);
    await app.close();
  });

  it('GET /api/banners with position filter passes query param', async () => {
    const mockData = { data: [] };
    mockApiRequest.mockResolvedValueOnce(mockData);

    const res = await app.inject({ method: 'GET', url: '/api/banners?position=hero' });
    expect(res.statusCode).toBe(200);
    expect(mockApiRequest).toHaveBeenCalledWith(
      '/api/v1/banners',
      expect.objectContaining({
        query: expect.objectContaining({ position: 'hero' }),
      }),
    );
    await app.close();
  });

  it('GET /api/banners defaults is_active to true', async () => {
    mockApiRequest.mockResolvedValueOnce({ data: [] });

    await app.inject({ method: 'GET', url: '/api/banners' });
    expect(mockApiRequest).toHaveBeenCalledWith(
      '/api/v1/banners',
      expect.objectContaining({
        query: expect.objectContaining({ is_active: 'true' }),
      }),
    );
    await app.close();
  });

  it('GET /api/banners with explicit is_active=false passes it', async () => {
    mockApiRequest.mockResolvedValueOnce({ data: [] });

    await app.inject({ method: 'GET', url: '/api/banners?is_active=false' });
    expect(mockApiRequest).toHaveBeenCalledWith(
      '/api/v1/banners',
      expect.objectContaining({
        query: expect.objectContaining({ is_active: 'false' }),
      }),
    );
    await app.close();
  });

  it('GET /api/banners returns 404 when ApiError is thrown', async () => {
    mockApiRequest.mockRejectedValueOnce(new ApiError(404, 'NOT_FOUND', 'Banners not found'));

    const res = await app.inject({ method: 'GET', url: '/api/banners' });
    expect(res.statusCode).toBe(404);
    expect(res.json()).toEqual({
      error: { code: 'NOT_FOUND', message: 'Banners not found' },
    });
    await app.close();
  });

  it('GET /api/banners with pagination params passes page and per_page', async () => {
    mockApiRequest.mockResolvedValueOnce({ data: [] });

    await app.inject({ method: 'GET', url: '/api/banners?page=2&per_page=10' });
    expect(mockApiRequest).toHaveBeenCalledWith(
      '/api/v1/banners',
      expect.objectContaining({
        query: expect.objectContaining({ page: '2', per_page: '10' }),
      }),
    );
    await app.close();
  });
});
