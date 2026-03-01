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
import { campaignRoutes } from '../routes/campaigns.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(cookie, { secret: 'test-secret' });
  await app.register(campaignRoutes);
  return app;
}

const activeCampaign = {
  id: 'c1',
  name: 'Summer Sale',
  code: 'SUMMER20',
  type: 'percentage',
  status: 'active',
  discount_value: 20,
  min_order_amount: 0,
  max_discount_amount: 100,
  start_date: new Date(Date.now() - 86400000).toISOString(), // yesterday
  end_date: new Date(Date.now() + 86400000).toISOString(),   // tomorrow
  max_usage_count: 1000,
  current_usage_count: 50,
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

describe('Campaign Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  describe('GET /api/campaigns', () => {
    it('returns 200 with campaign list', async () => {
      const mockData = { data: [activeCampaign] };
      mockApiRequest.mockResolvedValueOnce(mockData);

      const res = await app.inject({ method: 'GET', url: '/api/campaigns' });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual(mockData);
      await app.close();
    });

    it('returns 500 when upstream fails', async () => {
      mockApiRequest.mockRejectedValueOnce(new Error('Upstream error'));

      const res = await app.inject({ method: 'GET', url: '/api/campaigns' });
      expect(res.statusCode).toBe(500);
      await app.close();
    });
  });

  describe('POST /api/campaigns/validate', () => {
    it('returns 401 when not authenticated', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        payload: { code: 'SUMMER20' },
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('returns 400 when code is missing', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        headers: { Authorization: 'Bearer test-token' },
        payload: { code: '' },
      });
      expect(res.statusCode).toBe(400);
      const body = res.json();
      expect(body.error.code).toBe('BAD_REQUEST');
      await app.close();
    });

    it('returns 404 when campaign code not found', async () => {
      mockApiRequest.mockResolvedValueOnce({ data: [] });

      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        headers: { Authorization: 'Bearer test-token' },
        payload: { code: 'INVALID' },
      });
      expect(res.statusCode).toBe(404);
      const body = res.json();
      expect(body.error.code).toBe('NOT_FOUND');
      await app.close();
    });

    it('returns 404 when campaign is inactive', async () => {
      const inactiveCampaign = { ...activeCampaign, code: 'INACTIVE', status: 'inactive' };
      mockApiRequest.mockResolvedValueOnce({ data: [inactiveCampaign] });

      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        headers: { Authorization: 'Bearer test-token' },
        payload: { code: 'INACTIVE' },
      });
      expect(res.statusCode).toBe(404);
      await app.close();
    });

    it('returns 400 when campaign is expired', async () => {
      const expiredCampaign = {
        ...activeCampaign,
        start_date: new Date(Date.now() - 2 * 86400000).toISOString(),
        end_date: new Date(Date.now() - 86400000).toISOString(),
      };
      mockApiRequest.mockResolvedValueOnce({ data: [expiredCampaign] });

      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        headers: { Authorization: 'Bearer test-token' },
        payload: { code: 'SUMMER20' },
      });
      expect(res.statusCode).toBe(400);
      const body = res.json();
      expect(body.error.code).toBe('EXPIRED');
      await app.close();
    });

    it('returns 400 when campaign has not started yet', async () => {
      const futureCampaign = {
        ...activeCampaign,
        start_date: new Date(Date.now() + 86400000).toISOString(),
        end_date: new Date(Date.now() + 2 * 86400000).toISOString(),
      };
      mockApiRequest.mockResolvedValueOnce({ data: [futureCampaign] });

      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        headers: { Authorization: 'Bearer test-token' },
        payload: { code: 'SUMMER20' },
      });
      expect(res.statusCode).toBe(400);
      const body = res.json();
      expect(body.error.code).toBe('EXPIRED');
      await app.close();
    });

    it('returns 200 with campaign data for valid code', async () => {
      mockApiRequest.mockResolvedValueOnce({ data: [activeCampaign] });

      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        headers: { Authorization: 'Bearer test-token' },
        payload: { code: 'SUMMER20' },
      });
      expect(res.statusCode).toBe(200);
      const body = res.json();
      expect(body.data.code).toBe('SUMMER20');
      expect(body.data.is_active).toBe(true);
      expect(body.data.discount_value).toBe(20);
      await app.close();
    });

    it('validates code case-insensitively', async () => {
      mockApiRequest.mockResolvedValueOnce({ data: [activeCampaign] });

      const res = await app.inject({
        method: 'POST',
        url: '/api/campaigns/validate',
        headers: { Authorization: 'Bearer test-token' },
        payload: { code: 'summer20' },
      });
      expect(res.statusCode).toBe(200);
      await app.close();
    });
  });
});
