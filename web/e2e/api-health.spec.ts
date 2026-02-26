import { test, expect } from '@playwright/test';

// These tests verify that the backend services are reachable.
// They will fail if the backend is not running.
// Run with: docker compose --profile backend up
// Skip these in CI if backend is not available.

test.describe('API Health Checks', () => {
  test.skip(
    !process.env.CI_BACKEND_RUNNING,
    'Skipped: backend services are not running (set CI_BACKEND_RUNNING=1 to enable)',
  );

  test('API gateway health endpoint is reachable', async ({ request }) => {
    const response = await request.get('http://localhost:8080/health/live');
    expect(response.ok()).toBeTruthy();
  });

  test('BFF health endpoint is reachable', async ({ request }) => {
    const response = await request.get('http://localhost:3001/health');
    expect(response.ok()).toBeTruthy();
  });

  test('products endpoint responds via gateway', async ({ request }) => {
    const response = await request.get(
      'http://localhost:8080/api/v1/products',
    );
    expect(response.ok()).toBeTruthy();
  });

  test('search endpoint responds via gateway', async ({ request }) => {
    const response = await request.get(
      'http://localhost:8080/api/v1/search?q=test',
    );
    expect(response.ok()).toBeTruthy();
  });
});
