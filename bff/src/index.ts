import Fastify from 'fastify';
import cors from '@fastify/cors';
import cookie from '@fastify/cookie';
import { config } from './config.js';
import { errorHandler } from './middleware/error-handler.js';
import { productRoutes } from './routes/products.js';
import { cartRoutes } from './routes/cart.js';
import { orderRoutes } from './routes/orders.js';
import { authRoutes } from './routes/auth.js';
import { searchRoutes } from './routes/search.js';
import { campaignRoutes } from './routes/campaigns.js';
import { checkoutRoutes } from './routes/checkout.js';
import { bannerRoutes } from './routes/banners.js';
import { wishlistRoutes } from './routes/wishlist.js';

async function main(): Promise<void> {
  const app = Fastify({
    logger: {
      level: config.environment === 'production' ? 'info' : 'debug',
      transport:
        config.environment !== 'production'
          ? { target: 'pino-pretty', options: { colorize: true } }
          : undefined,
    },
  });

  // ── Plugins ──────────────────────────────────────────────────────────────

  await app.register(cors, {
    origin: config.environment === 'production'
      ? ['https://ecommercego.com']
      : [
          'http://localhost:3000', 'http://127.0.0.1:3000',
          'http://localhost:3002', 'http://127.0.0.1:3002',
          'http://localhost:3003', 'http://127.0.0.1:3003',
        ],
    credentials: true,
  });

  await app.register(cookie, {
    secret: config.cookieSecret,
  });

  // ── Error handling ───────────────────────────────────────────────────────

  app.setErrorHandler(errorHandler);

  // ── Health check ─────────────────────────────────────────────────────────

  app.get('/health', async () => ({ status: 'ok', service: 'bff' }));

  // ── Routes ───────────────────────────────────────────────────────────────

  await app.register(productRoutes);
  await app.register(cartRoutes);
  await app.register(orderRoutes);
  await app.register(authRoutes);
  await app.register(searchRoutes);
  await app.register(campaignRoutes);
  await app.register(checkoutRoutes);
  await app.register(bannerRoutes);
  await app.register(wishlistRoutes);

  // ── Start server ─────────────────────────────────────────────────────────

  try {
    await app.listen({ port: config.port, host: '0.0.0.0' });
    app.log.info(`BFF listening on http://0.0.0.0:${config.port}`);
  } catch (err) {
    app.log.error(err);
    process.exit(1);
  }
}

main();
