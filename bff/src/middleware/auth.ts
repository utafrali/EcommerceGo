import type { FastifyRequest, FastifyReply } from 'fastify';

/**
 * Fastify preHandler that extracts a JWT from the `auth` cookie or the
 * Authorization header (`Bearer <token>`) and attaches it to the request
 * so downstream handlers can forward it to the gateway.
 */
export async function authMiddleware(
  request: FastifyRequest,
  reply: FastifyReply,
): Promise<void> {
  const cookieToken = request.cookies?.auth;
  const headerToken = request.headers.authorization?.replace(/^Bearer\s+/i, '');

  const token = cookieToken || headerToken;

  if (!token) {
    reply.status(401).send({
      error: {
        code: 'UNAUTHORIZED',
        message: 'Authentication required',
      },
    });
    return;
  }

  request.authToken = token;
}
