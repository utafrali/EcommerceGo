import type { FastifyInstance } from 'fastify';
import { apiRequest } from '../services/http-client.js';

export async function bannerRoutes(app: FastifyInstance): Promise<void> {
  /**
   * GET /api/banners
   * List banners with optional position filter.
   */
  app.get<{
    Querystring: {
      position?: string;
      is_active?: string;
      page?: string;
      per_page?: string;
    };
  }>('/api/banners', async (request, reply) => {
    const { position, is_active, page, per_page } = request.query;
    const data = await apiRequest('/api/v1/banners', {
      query: {
        position,
        is_active: is_active ?? 'true',
        page,
        per_page,
      },
    });
    return reply.send(data);
  });
}
