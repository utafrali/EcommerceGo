import { type Page, expect } from '@playwright/test';

export const ADMIN_EMAIL = 'admin@ecommerce.com';
export const ADMIN_PASSWORD = 'AdminPass123';

/**
 * Log in as admin via the CMS login page.
 * After this call the page will be on /dashboard.
 */
export async function loginAsAdmin(page: Page) {
  await page.goto('/login');
  await page.fill('#email', ADMIN_EMAIL);
  await page.fill('#password', ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/dashboard', { timeout: 10000 });
}
