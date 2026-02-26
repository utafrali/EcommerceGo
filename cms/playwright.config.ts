import { defineConfig } from '@playwright/test';

const E2E_PORT = Number(process.env.E2E_PORT) || 3102;

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: `http://localhost:${E2E_PORT}`,
    headless: true,
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: `npx next dev --port ${E2E_PORT}`,
    port: E2E_PORT,
    timeout: 120000,
    reuseExistingServer: true,
  },
});
