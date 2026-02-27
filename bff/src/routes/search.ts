import type { FastifyInstance } from 'fastify';
import { apiRequest } from '../services/http-client.js';
import type { SearchResult } from '../types/index.js';

export async function searchRoutes(app: FastifyInstance): Promise<void> {
  /**
   * GET /api/search/suggest
   * Autocomplete / typeahead suggestions for the search bar.
   */
  app.get<{
    Querystring: { q?: string; limit?: string };
  }>('/api/search/suggest', async (request, reply) => {
    const { q, limit } = request.query;

    const data = await apiRequest('/api/v1/search/suggest', {
      query: { q, limit },
    });

    return reply.send(data);
  });

  /**
   * GET /api/search
   * Search products by query string with optional filters.
   */
  app.get<{
    Querystring: {
      q?: string;
      page?: string;
      pageSize?: string;
      per_page?: string;
      category?: string;
      category_id?: string;
      brand_id?: string;
      status?: string;
      minPrice?: string;
      maxPrice?: string;
      sort?: string;
    };
  }>('/api/search', async (request, reply) => {
    const {
      q,
      page,
      pageSize,
      per_page,
      category,
      category_id,
      brand_id,
      status,
      minPrice,
      maxPrice,
      sort,
    } = request.query;

    const data = await apiRequest<SearchResult>('/api/v1/search', {
      query: {
        q,
        page,
        per_page: per_page || pageSize,
        category_id: category_id || category,
        brand_id,
        status,
        min_price: minPrice,
        max_price: maxPrice,
        sort,
      },
    });

    return reply.send(data);
  });
}
