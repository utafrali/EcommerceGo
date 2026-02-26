import type { FastifyInstance } from 'fastify';
import { authMiddleware } from '../middleware/auth.js';
import { apiRequest } from '../services/http-client.js';
import type {
  Product,
  ProductListResponse,
  CategoryListResponse,
  BrandListResponse,
  Review,
  ReviewListResponse,
  CreateReviewRequest,
} from '../types/index.js';

export async function productRoutes(app: FastifyInstance): Promise<void> {
  /**
   * GET /api/products
   * List products with optional pagination and filtering.
   */
  app.get<{
    Querystring: {
      page?: string;
      pageSize?: string;
      per_page?: string;
      category?: string;
      category_id?: string;
      brand_id?: string;
      search?: string;
      min_price?: string;
      max_price?: string;
      status?: string;
      sort?: string;
    };
  }>('/api/products', async (request, reply) => {
    const {
      page,
      pageSize,
      per_page,
      category,
      category_id,
      brand_id,
      search,
      min_price,
      max_price,
      status,
      sort,
    } = request.query;

    const data = await apiRequest<ProductListResponse>('/api/v1/products', {
      query: {
        page,
        page_size: pageSize,
        per_page,
        category,
        category_id,
        brand_id,
        search,
        min_price,
        max_price,
        status,
        sort,
      },
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

  /**
   * GET /api/products/:id/reviews
   * List reviews for a specific product.
   */
  app.get<{
    Params: { id: string };
    Querystring: { page?: string; per_page?: string };
  }>('/api/products/:id/reviews', async (request, reply) => {
    const { id } = request.params;
    const { page, per_page } = request.query;

    const data = await apiRequest<ReviewListResponse>(
      `/api/v1/products/${encodeURIComponent(id)}/reviews`,
      {
        query: { page, per_page },
      },
    );

    return reply.send(data);
  });

  /**
   * POST /api/products/:id/reviews
   * Create a review for a specific product. Requires authentication.
   */
  app.post<{
    Params: { id: string };
    Body: CreateReviewRequest;
  }>('/api/products/:id/reviews', {
    preHandler: authMiddleware,
  }, async (request, reply) => {
    const { id } = request.params;

    const data = await apiRequest<Review>(
      `/api/v1/products/${encodeURIComponent(id)}/reviews`,
      {
        method: 'POST',
        body: request.body,
        token: request.authToken,
      },
    );

    return reply.status(201).send(data);
  });

  /**
   * GET /api/categories
   * List all product categories.
   */
  app.get('/api/categories', async (_request, reply) => {
    const data = await apiRequest<CategoryListResponse>('/api/v1/categories');

    return reply.send(data);
  });

  /**
   * GET /api/brands
   * List all product brands.
   */
  app.get('/api/brands', async (_request, reply) => {
    const data = await apiRequest<BrandListResponse>('/api/v1/brands');

    return reply.send(data);
  });
}
