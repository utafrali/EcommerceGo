import type { FastifyInstance } from 'fastify';
import { apiRequest } from '../services/http-client.js';
import type { SearchResult } from '../types/index.js';

export async function searchRoutes(app: FastifyInstance): Promise<void> {
  /**
   * GET /api/search
   * Search products by query string with optional filters.
   */
  app.get<{
    Querystring: {
      q?: string;
      page?: string;
      pageSize?: string;
      category?: string;
      minPrice?: string;
      maxPrice?: string;
      sort?: string;
    };
  }>('/api/search', async (request, reply) => {
    const { q, page, pageSize, category, minPrice, maxPrice, sort } =
      request.query;

    const data = await apiRequest<SearchResult>('/api/v1/search', {
      query: {
        q,
        page,
        page_size: pageSize,
        category,
        min_price: minPrice,
        max_price: maxPrice,
        sort,
      },
    });

    return reply.send(data);
  });
}
