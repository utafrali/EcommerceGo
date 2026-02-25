import type { FastifyInstance } from 'fastify';
import { apiRequest } from '../services/http-client.js';
import type { Product, ProductListResponse } from '../types/index.js';

export async function productRoutes(app: FastifyInstance): Promise<void> {
  /**
   * GET /api/products
   * List products with optional pagination and filtering.
   */
  app.get<{
    Querystring: {
      page?: string;
      pageSize?: string;
      category?: string;
      sort?: string;
    };
  }>('/api/products', async (request, reply) => {
    const { page, pageSize, category, sort } = request.query;

    const data = await apiRequest<ProductListResponse>('/api/v1/products', {
      query: { page, page_size: pageSize, category, sort },
    });

    return reply.send(data);
  });

  /**
   * GET /api/products/:slug
   * Retrieve a single product by its URL slug.
   */
  app.get<{
    Params: { slug: string };
  }>('/api/products/:slug', async (request, reply) => {
    const { slug } = request.params;

    const data = await apiRequest<Product>(`/api/v1/products/${encodeURIComponent(slug)}`);

    return reply.send(data);
  });
}
