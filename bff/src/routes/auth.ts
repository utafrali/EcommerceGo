import type { FastifyInstance } from 'fastify';
import { apiRequest } from '../services/http-client.js';
import { config } from '../config.js';
import type { AuthResponse, RegisterRequest, LoginRequest } from '../types/index.js';

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
    const data = await apiRequest<AuthResponse>('/api/v1/auth/register', {
      method: 'POST',
      body: request.body,
    });

    // Set auth cookie on successful registration
    reply.setCookie('auth', data.accessToken, COOKIE_OPTIONS);

    return reply.status(201).send({
      user: data.user,
    });
  });

  /**
   * POST /api/auth/login
   * Authenticate an existing user and set an auth cookie.
   */
  app.post<{
    Body: LoginRequest;
  }>('/api/auth/login', async (request, reply) => {
    const data = await apiRequest<AuthResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: request.body,
    });

    // Set auth cookie on successful login
    reply.setCookie('auth', data.accessToken, COOKIE_OPTIONS);

    return reply.send({
      user: data.user,
    });
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

    const data = await apiRequest<AuthResponse>('/api/v1/auth/refresh', {
      method: 'POST',
      token: currentToken,
    });

    // Update cookie with the new token
    reply.setCookie('auth', data.accessToken, COOKIE_OPTIONS);

    return reply.send({
      user: data.user,
    });
  });
}
