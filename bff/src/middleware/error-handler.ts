import type { FastifyError, FastifyReply, FastifyRequest } from 'fastify';
import { ApiError } from '../services/http-client.js';

/**
 * Global Fastify error handler.
 * Returns a consistent JSON envelope: { error: { code, message } }
 */
export function errorHandler(
  error: FastifyError,
  _request: FastifyRequest,
  reply: FastifyReply,
): void {
  const logger = _request.log;

  if (error instanceof ApiError) {
    reply.status(error.statusCode).send({
      error: {
        code: error.code,
        message: error.message,
      },
    });
    return;
  }

  // Fastify validation errors
  if (error.validation) {
    reply.status(400).send({
      error: {
        code: 'VALIDATION_ERROR',
        message: error.message,
      },
    });
    return;
  }

  // Unexpected errors
  logger.error(error, 'Unhandled error');

  const statusCode = error.statusCode ?? 500;
  reply.status(statusCode).send({
    error: {
      code: 'INTERNAL_ERROR',
      message:
        statusCode === 500
          ? 'An unexpected error occurred'
          : error.message,
    },
  });
}
