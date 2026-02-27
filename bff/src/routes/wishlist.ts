import type { FastifyInstance } from 'fastify';
import { authMiddleware } from '../middleware/auth.js';
import { apiRequest } from '../services/http-client.js';

export async function wishlistRoutes(app: FastifyInstance): Promise<void> {
  // All wishlist routes require authentication
  app.addHook('preHandler', authMiddleware);

  /**
   * GET /api/wishlist
   * Retrieve the current user's wishlist (paginated).
   */
  app.get<{
    Querystring: { page?: string; per_page?: string };
  }>('/api/wishlist', async (request, reply) => {
    const { page, per_page } = request.query;
    const data = await apiRequest('/api/v1/users/wishlist', {
      token: request.authToken,
      query: { page, per_page },
    });

    return reply.send(data);
  });

  /**
   * POST /api/wishlist/:productId
   * Add a product to the wishlist.
   */
  app.post<{
    Params: { productId: string };
  }>('/api/wishlist/:productId', async (request, reply) => {
    const { productId } = request.params;

    const data = await apiRequest(
      `/api/v1/users/wishlist/${encodeURIComponent(productId)}`,
      {
        method: 'POST',
        token: request.authToken,
      },
    );

    return reply.status(201).send(data);
  });

  /**
   * DELETE /api/wishlist/:productId
   * Remove a product from the wishlist.
   */
  app.delete<{
    Params: { productId: string };
  }>('/api/wishlist/:productId', async (request, reply) => {
    const { productId } = request.params;

    await apiRequest<void>(
      `/api/v1/users/wishlist/${encodeURIComponent(productId)}`,
      {
        method: 'DELETE',
        token: request.authToken,
      },
    );

    return reply.status(204).send();
  });

  /**
   * GET /api/wishlist/:productId/exists
   * Check whether a product is in the user's wishlist.
   */
  app.get<{
    Params: { productId: string };
  }>('/api/wishlist/:productId/exists', async (request, reply) => {
    const { productId } = request.params;

    const data = await apiRequest(
      `/api/v1/users/wishlist/${encodeURIComponent(productId)}`,
      {
        token: request.authToken,
      },
    );

    return reply.send(data);
  });
}
