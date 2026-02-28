import { defineConfig } from '@playwright/test';

/**
 * Port 3100 is used for E2E testing to avoid conflicts with other services
 * that may run on port 3000 (e.g., Grafana in Docker Compose).
 */
const E2E_PORT = Number(process.env.E2E_PORT) || 3100;

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: `http://localhost:${E2E_PORT}`,
    headless: false,
    // slowMo: 500, // Commented out - causes TypeScript error (not in official types)
    screenshot: 'only-on-failure',
  },
  workers: 1, // Run tests serially so you can watch
  webServer: {
    command: `npx next dev --port ${E2E_PORT}`,
    port: E2E_PORT,
    timeout: 120000,
    reuseExistingServer: true,
  },
});
