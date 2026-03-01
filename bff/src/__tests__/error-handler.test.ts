import { describe, it, expect, vi, beforeEach } from 'vitest';
import Fastify from 'fastify';
import type { FastifyInstance } from 'fastify';
import { errorHandler } from '../middleware/error-handler.js';
import { ApiError } from '../services/http-client.js';

async function buildApp(): Promise<FastifyInstance> {
  const app = Fastify({ logger: false });
  app.setErrorHandler(errorHandler);

  // Route that throws ApiError
  app.get('/throw-api-error', async () => {
    throw new ApiError(404, 'NOT_FOUND', 'Resource not found');
  });

  // Route that throws ApiError with 4xx
  app.get('/throw-api-error-400', async () => {
    throw new ApiError(400, 'BAD_REQUEST', 'Bad request');
  });

  // Route that throws generic Error
  app.get('/throw-generic', async () => {
    throw new Error('Something went wrong');
  });

  // Route that throws an error with a statusCode (Fastify-style)
  app.get('/throw-with-status', async () => {
    const err: any = new Error('Custom status error');
    err.statusCode = 422;
    throw err;
  });

  return app;
}

describe('Error Handler Middleware', () => {
  let app: FastifyInstance;

  beforeEach(async () => {
    app = await buildApp();
  });

  it('returns 404 with error envelope for ApiError 404', async () => {
    const res = await app.inject({ method: 'GET', url: '/throw-api-error' });
    expect(res.statusCode).toBe(404);
    const body = res.json();
    expect(body).toEqual({
      error: { code: 'NOT_FOUND', message: 'Resource not found' },
    });
    await app.close();
  });

  it('returns 400 with error envelope for ApiError 400', async () => {
    const res = await app.inject({ method: 'GET', url: '/throw-api-error-400' });
    expect(res.statusCode).toBe(400);
    const body = res.json();
    expect(body).toEqual({
      error: { code: 'BAD_REQUEST', message: 'Bad request' },
    });
    await app.close();
  });

  it('returns 500 with INTERNAL_ERROR for generic Error', async () => {
    const res = await app.inject({ method: 'GET', url: '/throw-generic' });
    expect(res.statusCode).toBe(500);
    const body = res.json();
    expect(body).toEqual({
      error: { code: 'INTERNAL_ERROR', message: 'An unexpected error occurred' },
    });
    await app.close();
  });

  it('returns 422 with INTERNAL_ERROR for error with non-500 statusCode', async () => {
    const res = await app.inject({ method: 'GET', url: '/throw-with-status' });
    expect(res.statusCode).toBe(422);
    const body = res.json();
    expect(body).toEqual({
      error: { code: 'INTERNAL_ERROR', message: 'Custom status error' },
    });
    await app.close();
  });
});
