import type { FastifyInstance } from 'fastify';
import { apiRequest } from '../services/http-client.js';
import { authMiddleware } from '../middleware/auth.js';
import { config } from '../config.js';
import type { AuthResponse, RegisterRequest, LoginRequest, User } from '../types/index.js';

/** Cookie options for the auth token. */
const COOKIE_OPTIONS = {
  httpOnly: true,
  secure: config.environment === 'production',
  sameSite: 'lax' as const,
  path: '/',
  maxAge: 7 * 24 * 60 * 60, // 7 days in seconds
};

export async function authRoutes(app: FastifyInstance): Promise<void> {
  /**
   * POST /api/auth/register
   * Register a new user account.
   */
  app.post<{
    Body: RegisterRequest;
  }>('/api/auth/register', async (request, reply) => {
    const resp = await apiRequest<AuthResponse>('/api/v1/auth/register', {
      method: 'POST',
      body: request.body,
    });

    // Set auth cookie on successful registration
    reply.setCookie('auth', resp.data.tokens.access_token, COOKIE_OPTIONS);

    return reply.status(201).send({
      data: resp.data.user,
    });
  });

  /**
   * POST /api/auth/login
   * Authenticate an existing user and set an auth cookie.
   */
  app.post<{
    Body: LoginRequest;
  }>('/api/auth/login', async (request, reply) => {
    const resp = await apiRequest<AuthResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: request.body,
    });

    // Set auth cookie on successful login
    reply.setCookie('auth', resp.data.tokens.access_token, COOKIE_OPTIONS);

    return reply.send({
      data: resp.data.user,
    });
  });

  /**
   * GET /api/auth/me
   * Get the currently authenticated user by decoding the JWT payload.
   * The user service does not expose a /me endpoint, so we decode the
   * token directly.
   */
  app.get('/api/auth/me', {
    preHandler: authMiddleware,
  }, async (request, reply) => {
    const token = request.authToken!;
    const parts = token.split('.');
    if (parts.length !== 3) {
      return reply.status(401).send({
        error: { code: 'UNAUTHORIZED', message: 'Invalid token format' },
      });
    }

    try {
      const payload = JSON.parse(Buffer.from(parts[1], 'base64').toString());
      // Return snake_case to match frontend expectations and backend convention
      const user = {
        id: payload.user_id || payload.sub || '',
        email: payload.email || '',
        first_name: payload.first_name || payload.email?.split('@')[0] || 'User',
        last_name: payload.last_name || '',
        role: payload.role || 'customer',
        created_at: payload.iat ? new Date(payload.iat * 1000).toISOString() : '',
      };
      return reply.send({ data: user });
    } catch {
      return reply.status(401).send({
        error: { code: 'UNAUTHORIZED', message: 'Failed to decode token' },
      });
    }
  });

  /**
   * POST /api/auth/logout
   * Clear the auth cookie.
   */
  app.post('/api/auth/logout', async (_request, reply) => {
    reply.clearCookie('auth', { path: '/' });

    return reply.status(204).send();
  });

  /**
   * POST /api/auth/refresh
   * Refresh the access token using the current cookie.
   */
  app.post('/api/auth/refresh', async (request, reply) => {
    const currentToken = request.cookies?.auth;

    if (!currentToken) {
      return reply.status(401).send({
        error: {
          code: 'UNAUTHORIZED',
          message: 'No auth token present',
        },
      });
    }

    const resp = await apiRequest<AuthResponse>('/api/v1/auth/refresh', {
      method: 'POST',
      token: currentToken,
    });

    // Update cookie with the new token
    reply.setCookie('auth', resp.data.tokens.access_token, COOKIE_OPTIONS);

    return reply.send({
      user: resp.data.user,
    });
  });
}
