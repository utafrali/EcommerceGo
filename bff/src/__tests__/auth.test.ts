import { describe, it, expect, vi, beforeEach } from 'vitest';
import Fastify from 'fastify';
import cookie from '@fastify/cookie';
import type { FastifyInstance } from 'fastify';

vi.mock('../services/http-client.js', () => ({
  apiRequest: vi.fn(),
  ApiError: class ApiError extends Error {
    statusCode: number;
    code: string;
    constructor(statusCode: number, code: string, message: string) {
      super(message);
      this.statusCode = statusCode;
      this.code = code;
      this.name = 'ApiError';
    }
  },
}));

import { apiRequest, ApiError } from '../services/http-client.js';
import { authRoutes } from '../routes/auth.js';
import { errorHandler } from '../middleware/error-handler.js';

const mockApiRequest = vi.mocked(apiRequest);

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);
  await app.register(cookie, { secret: 'test-secret' });
  await app.register(authRoutes);
  return app;
}

const mockUser = {
  id: 'user-1',
  email: 'test@example.com',
  firstName: 'John',
  lastName: 'Doe',
  createdAt: '2025-01-01T00:00:00Z',
};

const mockAuthResponse = {
  data: {
    user: mockUser,
    tokens: {
      access_token: 'access-token-123',
      refresh_token: 'refresh-token-456',
    },
  },
};

describe('Auth Routes', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    vi.clearAllMocks();
    app = await buildApp();
  });

  describe('POST /api/auth/register', () => {
    it('returns 201 with user data on successful registration', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/register',
        payload: {
          email: 'test@example.com',
          password: 'password123',
          firstName: 'John',
          lastName: 'Doe',
        },
      });
      expect(res.statusCode).toBe(201);
      expect(res.json()).toEqual({ data: mockUser });
      await app.close();
    });

    it('sets auth cookie on successful registration', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/register',
        payload: { email: 'test@example.com', password: 'pw', firstName: 'A', lastName: 'B' },
      });
      const cookies = res.cookies;
      expect(cookies.some((c) => c.name === 'auth')).toBe(true);
      const authCookie = cookies.find((c) => c.name === 'auth');
      expect(authCookie?.value).toBe('access-token-123');
      await app.close();
    });

    it('calls gateway with POST to /api/v1/auth/register', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);
      const body = { email: 'e@e.com', password: 'pass', firstName: 'X', lastName: 'Y' };

      await app.inject({ method: 'POST', url: '/api/auth/register', payload: body });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/auth/register',
        expect.objectContaining({ method: 'POST', body }),
      );
      await app.close();
    });

    it('returns 409 on duplicate email', async () => {
      mockApiRequest.mockRejectedValueOnce(new ApiError(409, 'CONFLICT', 'Email already exists'));

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/register',
        payload: { email: 'dup@example.com', password: 'pw', firstName: 'A', lastName: 'B' },
      });
      expect(res.statusCode).toBe(409);
      expect(res.json()).toEqual({ error: { code: 'CONFLICT', message: 'Email already exists' } });
      await app.close();
    });
  });

  describe('POST /api/auth/login', () => {
    it('returns 200 with user data on successful login', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/login',
        payload: { email: 'test@example.com', password: 'password123' },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual({ data: mockUser });
      await app.close();
    });

    it('sets auth cookie on successful login', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/login',
        payload: { email: 'test@example.com', password: 'password123' },
      });
      const authCookie = res.cookies.find((c) => c.name === 'auth');
      expect(authCookie?.value).toBe('access-token-123');
      expect(authCookie?.httpOnly).toBe(true);
      await app.close();
    });

    it('returns 401 on invalid credentials', async () => {
      mockApiRequest.mockRejectedValueOnce(
        new ApiError(401, 'UNAUTHORIZED', 'Invalid credentials'),
      );

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/login',
        payload: { email: 'wrong@example.com', password: 'wrongpass' },
      });
      expect(res.statusCode).toBe(401);
      expect(res.json()).toEqual({
        error: { code: 'UNAUTHORIZED', message: 'Invalid credentials' },
      });
      await app.close();
    });

    it('calls gateway with POST to /api/v1/auth/login', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      await app.inject({
        method: 'POST',
        url: '/api/auth/login',
        payload: { email: 'e@e.com', password: 'p' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/auth/login',
        expect.objectContaining({ method: 'POST' }),
      );
      await app.close();
    });
  });

  describe('POST /api/auth/logout', () => {
    it('returns 204 and clears auth cookie', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/logout',
      });
      expect(res.statusCode).toBe(204);
      // Cookie should be cleared (set to empty or expires in the past)
      const authCookie = res.cookies.find((c) => c.name === 'auth');
      // The clearCookie sets maxAge=0 or expires to epoch
      expect(authCookie).toBeDefined();
      await app.close();
    });

    it('returns 204 even without auth token', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/logout',
      });
      expect(res.statusCode).toBe(204);
      await app.close();
    });
  });

  describe('POST /api/auth/refresh', () => {
    it('returns 401 when no auth cookie present', async () => {
      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/refresh',
      });
      expect(res.statusCode).toBe(401);
      expect(res.json()).toEqual({
        error: { code: 'UNAUTHORIZED', message: 'No auth token present' },
      });
      await app.close();
    });

    it('returns 200 with user data on successful refresh', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/refresh',
        cookies: { auth: 'old-token' },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json()).toEqual({ user: mockUser });
      await app.close();
    });

    it('sets new auth cookie on successful refresh', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/refresh',
        cookies: { auth: 'old-token' },
      });
      const authCookie = res.cookies.find((c) => c.name === 'auth');
      expect(authCookie?.value).toBe('access-token-123');
      await app.close();
    });

    it('returns 401 when refresh token is expired', async () => {
      mockApiRequest.mockRejectedValueOnce(
        new ApiError(401, 'UNAUTHORIZED', 'Token expired'),
      );

      const res = await app.inject({
        method: 'POST',
        url: '/api/auth/refresh',
        cookies: { auth: 'expired-token' },
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('calls gateway with POST to /api/v1/auth/refresh with token', async () => {
      mockApiRequest.mockResolvedValueOnce(mockAuthResponse);

      await app.inject({
        method: 'POST',
        url: '/api/auth/refresh',
        cookies: { auth: 'current-token' },
      });
      expect(mockApiRequest).toHaveBeenCalledWith(
        '/api/v1/auth/refresh',
        expect.objectContaining({ method: 'POST', token: 'current-token' }),
      );
      await app.close();
    });
  });

  describe('GET /api/auth/me', () => {
    it('returns 401 when not authenticated', async () => {
      const res = await app.inject({ method: 'GET', url: '/api/auth/me' });
      expect(res.statusCode).toBe(401);
      await app.close();
    });

    it('returns 401 when token has invalid format', async () => {
      const res = await app.inject({
        method: 'GET',
        url: '/api/auth/me',
        headers: { Authorization: 'Bearer notajwt' },
      });
      expect(res.statusCode).toBe(401);
      const body = res.json();
      expect(body.error.code).toBe('UNAUTHORIZED');
      await app.close();
    });

    it('returns 200 with decoded user from valid JWT', async () => {
      // Build a minimal JWT: header.payload.signature
      const payload = {
        user_id: 'user-123',
        email: 'me@example.com',
        first_name: 'Jane',
        last_name: 'Smith',
        role: 'customer',
        iat: Math.floor(Date.now() / 1000),
      };
      const b64Header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
      const b64Payload = Buffer.from(JSON.stringify(payload)).toString('base64url');
      const fakeJwt = `${b64Header}.${b64Payload}.fakesignature`;

      const res = await app.inject({
        method: 'GET',
        url: '/api/auth/me',
        headers: { Authorization: `Bearer ${fakeJwt}` },
      });
      expect(res.statusCode).toBe(200);
      const body = res.json();
      expect(body.data.id).toBe('user-123');
      expect(body.data.email).toBe('me@example.com');
      expect(body.data.first_name).toBe('Jane');
      expect(body.data.last_name).toBe('Smith');
      expect(body.data.role).toBe('customer');
      await app.close();
    });

    it('falls back to sub when user_id not in payload', async () => {
      const payload = {
        sub: 'sub-user-id',
        email: 'sub@example.com',
        iat: Math.floor(Date.now() / 1000),
      };
      const b64Header = Buffer.from(JSON.stringify({ alg: 'HS256' })).toString('base64url');
      const b64Payload = Buffer.from(JSON.stringify(payload)).toString('base64url');
      const fakeJwt = `${b64Header}.${b64Payload}.sig`;

      const res = await app.inject({
        method: 'GET',
        url: '/api/auth/me',
        headers: { Authorization: `Bearer ${fakeJwt}` },
      });
      expect(res.statusCode).toBe(200);
      expect(res.json().data.id).toBe('sub-user-id');
      await app.close();
    });

    it('returns 401 when payload is not valid JSON', async () => {
      // Create a token where the payload part is not valid base64-encoded JSON
      const fakeJwt = 'header.!!!invalid_base64!!!.signature';

      const res = await app.inject({
        method: 'GET',
        url: '/api/auth/me',
        headers: { Authorization: `Bearer ${fakeJwt}` },
      });
      expect(res.statusCode).toBe(401);
      await app.close();
    });
  });
});
