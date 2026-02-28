import { test, expect } from '@playwright/test';

test.describe('Auth Flow - Login Page', () => {
  test('login page loads with heading', async ({ page }) => {
    await page.goto('/auth/login');
    await expect(
      page.getByRole('heading', { name: "EcommerceGo'ya Giriş Yap" }),
    ).toBeVisible();
  });

  test('login page has register link', async ({ page }) => {
    await page.goto('/auth/login');
    await expect(page.getByText('Hesabınız yok mu?')).toBeVisible();
    await expect(page.getByRole('main').getByRole('link', { name: 'Üye Ol' })).toBeVisible();
  });

  test('login form has email input', async ({ page }) => {
    await page.goto('/auth/login');
    const emailInput = page.getByLabel('E-posta adresi');
    await expect(emailInput).toBeVisible();
    await expect(emailInput).toHaveAttribute('type', 'email');
    await expect(emailInput).toHaveAttribute(
      'placeholder',
      'siz@ornek.com',
    );
  });

  test('login form has password input', async ({ page }) => {
    await page.goto('/auth/login');
    const passwordInput = page.locator('input[name="password"]');
    await expect(passwordInput).toBeVisible();
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await expect(passwordInput).toHaveAttribute(
      'placeholder',
      'Şifrenizi girin',
    );
  });

  test('login form has submit button', async ({ page }) => {
    await page.goto('/auth/login');
    const submitButton = page.getByRole('button', { name: 'Giriş Yap' });
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toHaveAttribute('type', 'submit');
  });

  test('email and password fields accept input', async ({ page }) => {
    await page.goto('/auth/login');
    const emailInput = page.getByLabel('E-posta adresi');
    const passwordInput = page.locator('input[name="password"]');

    await emailInput.fill('test@example.com');
    await passwordInput.fill('password123');

    await expect(emailInput).toHaveValue('test@example.com');
    await expect(passwordInput).toHaveValue('password123');
  });

  test('email field has required attribute', async ({ page }) => {
    await page.goto('/auth/login');
    const emailInput = page.getByLabel('E-posta adresi');
    await expect(emailInput).toHaveAttribute('required', '');
  });

  test('password field has required attribute', async ({ page }) => {
    await page.goto('/auth/login');
    const passwordInput = page.locator('input[name="password"]');
    await expect(passwordInput).toHaveAttribute('required', '');
  });
});

test.describe('Auth Flow - Cart Page (requires auth)', () => {
  test('cart page loads with heading', async ({ page }) => {
    await page.goto('/cart');
    await expect(
      page.getByRole('heading', { name: 'Sepetim' }),
    ).toBeVisible();
  });

  test('cart page shows empty cart message', async ({ page }) => {
    await page.goto('/cart');
    await expect(page.getByText('Sepetiniz boş')).toBeVisible();
  });

  test('cart page shows explore products link', async ({ page }) => {
    await page.goto('/cart');
    const exploreLink = page.getByRole('link', { name: 'Ürünleri Keşfet' });
    await expect(exploreLink).toBeVisible();
    await expect(exploreLink).toHaveAttribute('href', '/products');
  });

  test('cart page displays empty state with helpful message', async ({
    page,
  }) => {
    await page.goto('/cart');
    await expect(page.getByText('Sepetiniz boş')).toBeVisible();
    await expect(
      page.getByText(
        'Harika ürünleri keşfedin ve alışverişe başlayın. Aradığınız şey bir tık uzağınızda!',
      ),
    ).toBeVisible();
  });
});
