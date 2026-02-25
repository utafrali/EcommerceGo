import type { FastifyInstance } from 'fastify';
import { authMiddleware } from '../middleware/auth.js';
import { apiRequest } from '../services/http-client.js';
import type { Order, OrderListResponse, CreateOrderRequest } from '../types/index.js';

export async function orderRoutes(app: FastifyInstance): Promise<void> {
  // All order routes require authentication
  app.addHook('preHandler', authMiddleware);

  /**
   * GET /api/orders
   * List the current user's orders with pagination.
   */
  app.get<{
    Querystring: { page?: string; pageSize?: string };
  }>('/api/orders', async (request, reply) => {
    const { page, pageSize } = request.query;

    const data = await apiRequest<OrderListResponse>('/api/v1/orders', {
      token: request.authToken,
      query: { page, page_size: pageSize },
    });

    return reply.send(data);
  });

  /**
   * GET /api/orders/:id
   * Retrieve a single order by ID.
   */
  app.get<{
    Params: { id: string };
  }>('/api/orders/:id', async (request, reply) => {
    const { id } = request.params;

    const data = await apiRequest<Order>(
      `/api/v1/orders/${encodeURIComponent(id)}`,
      { token: request.authToken },
    );

    return reply.send(data);
  });

  /**
   * POST /api/orders
   * Create a new order from the current cart.
   */
  app.post<{
    Body: CreateOrderRequest;
  }>('/api/orders', async (request, reply) => {
    const data = await apiRequest<Order>('/api/v1/orders', {
      method: 'POST',
      body: request.body,
      token: request.authToken,
    });

    return reply.status(201).send(data);
  });
}
