import type { FastifyInstance } from 'fastify';
import { authMiddleware } from '../middleware/auth.js';
import { apiRequest } from '../services/http-client.js';
import type { Cart, AddCartItemRequest, UpdateCartItemRequest } from '../types/index.js';

export async function cartRoutes(app: FastifyInstance): Promise<void> {
  // All cart routes require authentication
  app.addHook('preHandler', authMiddleware);

  /**
   * GET /api/cart
   * Retrieve the current user's cart.
   */
  app.get('/api/cart', async (request, reply) => {
    const data = await apiRequest<Cart>('/api/v1/cart', {
      token: request.authToken,
    });

    return reply.send(data);
  });

  /**
   * POST /api/cart/items
   * Add an item to the cart.
   */
  app.post<{
    Body: AddCartItemRequest;
  }>('/api/cart/items', async (request, reply) => {
    const data = await apiRequest<Cart>('/api/v1/cart/items', {
      method: 'POST',
      body: request.body,
      token: request.authToken,
    });

    return reply.status(201).send(data);
  });

  /**
   * PUT /api/cart/items/:id
   * Update the quantity of a cart item.
   */
  app.put<{
    Params: { id: string };
    Body: UpdateCartItemRequest;
  }>('/api/cart/items/:id', async (request, reply) => {
    const { id } = request.params;

    const data = await apiRequest<Cart>(
      `/api/v1/cart/items/${encodeURIComponent(id)}`,
      {
        method: 'PUT',
        body: request.body,
        token: request.authToken,
      },
    );

    return reply.send(data);
  });

  /**
   * DELETE /api/cart/items/:id
   * Remove an item from the cart.
   */
  app.delete<{
    Params: { id: string };
  }>('/api/cart/items/:id', async (request, reply) => {
    const { id } = request.params;

    await apiRequest<void>(
      `/api/v1/cart/items/${encodeURIComponent(id)}`,
      {
        method: 'DELETE',
        token: request.authToken,
      },
    );

    return reply.status(204).send();
  });
}
