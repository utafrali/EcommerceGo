export const config = {
  port: parseInt(process.env.BFF_PORT || '3001', 10),
  gatewayUrl: process.env.GATEWAY_URL || 'http://localhost:8080',
  cookieSecret: process.env.COOKIE_SECRET || 'change-me-in-production',
  environment: process.env.NODE_ENV || 'development',
} as const;
