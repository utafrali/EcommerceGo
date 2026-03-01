import { describe, it, expect, beforeEach } from 'vitest';
import Fastify from 'fastify';
import cookie from '@fastify/cookie';
import type { FastifyInstance } from 'fastify';
import { authMiddleware } from '../middleware/auth.js';
import '../types/index.js';

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  await app.register(cookie, { secret: 'test-secret' });

  app.get('/protected', { preHandler: authMiddleware }, async (request, reply) => {
    return reply.send({ token: request.authToken });
  });

  return app;
}

describe('Auth Middleware', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    app = await buildApp();
  });

  it('returns 401 when no token is present', async () => {
    const res = await app.inject({ method: 'GET', url: '/protected' });
    expect(res.statusCode).toBe(401);
    const body = res.json();
    expect(body).toEqual({
      error: { code: 'UNAUTHORIZED', message: 'Authentication required' },
    });
    await app.close();
  });

  it('extracts token from Authorization header', async () => {
    const res = await app.inject({
      method: 'GET',
      url: '/protected',
      headers: { Authorization: 'Bearer my-test-token' },
    });
    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.token).toBe('my-test-token');
    await app.close();
  });

  it('extracts token from Authorization header case-insensitively', async () => {
    const res = await app.inject({
      method: 'GET',
      url: '/protected',
      headers: { Authorization: 'bearer another-token' },
    });
    expect(res.statusCode).toBe(200);
    expect(res.json().token).toBe('another-token');
    await app.close();
  });

  it('returns 401 when Authorization header has no Bearer prefix', async () => {
    // Without "Bearer " prefix, the whole header value is treated as token
    const res = await app.inject({
      method: 'GET',
      url: '/protected',
      headers: { Authorization: '' },
    });
    // Empty auth header => no token resolved => 401
    expect(res.statusCode).toBe(401);
    await app.close();
  });

  it('extracts token from auth cookie', async () => {
    const res = await app.inject({
      method: 'GET',
      url: '/protected',
      cookies: { auth: 'cookie-token-value' },
    });
    expect(res.statusCode).toBe(200);
    expect(res.json().token).toBe('cookie-token-value');
    await app.close();
  });

  it('prefers cookie token over Authorization header', async () => {
    const res = await app.inject({
      method: 'GET',
      url: '/protected',
      headers: { Authorization: 'Bearer header-token' },
      cookies: { auth: 'cookie-token' },
    });
    expect(res.statusCode).toBe(200);
    // cookie takes precedence
    expect(res.json().token).toBe('cookie-token');
    await app.close();
  });
});
