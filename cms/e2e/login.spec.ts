import { test, expect } from '@playwright/test';
import { ADMIN_EMAIL, ADMIN_PASSWORD, loginAsAdmin } from './helpers';

// ─── Test Suite: CMS Login Flow ────────────────────────────────────────────

test.describe('CMS Login Flow', () => {

  // ── 1. Login page renders correctly ──────────────────────────────────────
  test('login page renders all required elements', async ({ page }) => {
    await page.goto('/login');

    // Title elements
    await expect(page.locator('h1')).toHaveText('EcommerceGo');
    await expect(page.locator('p', { hasText: 'Admin Login' })).toBeVisible();

    // Email field
    const emailInput = page.locator('#email');
    await expect(emailInput).toBeVisible();
    await expect(emailInput).toHaveAttribute('type', 'email');
    await expect(emailInput).toHaveAttribute('required', '');

    // Password field
    const passwordInput = page.locator('#password');
    await expect(passwordInput).toBeVisible();
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await expect(passwordInput).toHaveAttribute('required', '');

    // Submit button
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toHaveText('Sign In');

    // Footer
    await expect(
      page.locator('text=EcommerceGo CMS — Administration Panel'),
    ).toBeVisible();
  });

  // ── 2. Successful login redirects to dashboard ────────────────────────────
  test('successful login with valid credentials redirects to dashboard', async ({ page }) => {
    await page.goto('/login');

    await page.fill('#email', ADMIN_EMAIL);
    await page.fill('#password', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // Should land on dashboard
    await page.waitForURL('**/dashboard', { timeout: 10000 });
    await expect(page.locator('h1', { hasText: 'Dashboard' })).toBeVisible();

    // Token should be persisted in localStorage
    const token = await page.evaluate(() =>
      localStorage.getItem('cms_auth_token'),
    );
    expect(token).toBeTruthy();
  });

  // ── 3. Invalid credentials shows error message ────────────────────────────
  test('invalid credentials shows an error message', async ({ page }) => {
    await page.goto('/login');

    await page.fill('#email', ADMIN_EMAIL);
    await page.fill('#password', 'WrongPassword999');
    await page.click('button[type="submit"]');

    // Error box should appear; URL must stay on /login
    const errorBox = page.locator('.bg-red-50');
    await expect(errorBox).toBeVisible({ timeout: 8000 });
    await expect(errorBox).not.toBeEmpty();

    await expect(page).toHaveURL(/\/login/);
  });

  // ── 4. Empty form submission triggers native HTML5 validation ─────────────
  test('submitting an empty form triggers required-field validation', async ({ page }) => {
    await page.goto('/login');

    // Click submit without filling anything
    await page.click('button[type="submit"]');

    // HTML5 required validation prevents submission and keeps us on /login
    await expect(page).toHaveURL(/\/login/);

    // Both inputs should be flagged as invalid by the browser
    const emailValidity = await page.evaluate(
      () =>
        (document.querySelector('#email') as HTMLInputElement).validity.valueMissing,
    );
    const passwordValidity = await page.evaluate(
      () =>
        (document.querySelector('#password') as HTMLInputElement).validity
          .valueMissing,
    );
    expect(emailValidity).toBe(true);
    expect(passwordValidity).toBe(true);
  });

  // ── 5. Logout clears session and redirects to /login ─────────────────────
  test('logout button clears session and redirects to login page', async ({ page }) => {
    // Start from an authenticated state
    await loginAsAdmin(page);

    // The logout button lives in the top-bar header
    const logoutButton = page.locator('button', { hasText: 'Logout' });
    await expect(logoutButton).toBeVisible();
    await logoutButton.click();

    // Should be back on /login
    await page.waitForURL('**/login', { timeout: 8000 });
    await expect(page.locator('h1')).toHaveText('EcommerceGo');

    // Token must be gone
    const token = await page.evaluate(() =>
      localStorage.getItem('cms_auth_token'),
    );
    expect(token).toBeNull();
  });

  // ── 6. Unauthenticated access to /dashboard redirects to /login ───────────
  test('unauthenticated visit to /dashboard redirects to /login', async ({ page }) => {
    // Ensure no stale token is present
    await page.goto('/login');
    await page.evaluate(() => localStorage.removeItem('cms_auth_token'));

    await page.goto('/dashboard');
    await page.waitForURL('**/login', { timeout: 8000 });
    await expect(page.locator('h1')).toHaveText('EcommerceGo');
  });

  // ── 7. Already-authenticated user on /login redirects to /dashboard ────────
  test('authenticated user visiting /login is redirected to dashboard', async ({ page }) => {
    // First, establish an authenticated session
    await loginAsAdmin(page);

    // Verify token exists
    const token = await page.evaluate(() => localStorage.getItem('cms_auth_token'));
    expect(token).toBeTruthy();

    // Now navigate back to the login page
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    // AuthContext should redirect to /dashboard (may go through /login briefly)
    await page.waitForURL('**/dashboard', { timeout: 15000 });
  });
});
