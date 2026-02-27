import { test, expect } from '@playwright/test';

test.describe('Smoke Tests', () => {
  test('homepage loads with correct title', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle('EcommerceGo');
  });

  test('homepage displays hero heading', async ({ page }) => {
    await page.goto('/');
    // The HeroSlider fallback always renders "Discover Quality Products"
    const heading = page.getByRole('heading', {
      name: 'Discover Quality Products',
    });
    await expect(heading).toBeVisible();
  });

  test('homepage has Shop Now link', async ({ page }) => {
    await page.goto('/');
    const link = page.getByRole('link', { name: 'Shop Now' }).first();
    await expect(link).toBeVisible();
    await expect(link).toHaveAttribute('href', '/products');
  });

  test('homepage displays benefit bar', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('Free Shipping')).toBeVisible();
    await expect(page.getByText('Secure Payment')).toBeVisible();
    await expect(page.getByText('Easy Returns')).toBeVisible();
    await expect(page.getByText('24/7 Support')).toBeVisible();
  });

  test('page has correct meta description', async ({ page }) => {
    await page.goto('/');
    const metaDescription = page.locator('meta[name="description"]');
    await expect(metaDescription).toHaveAttribute(
      'content',
      'AI-driven open-source e-commerce platform',
    );
  });

  test('page has correct lang attribute', async ({ page }) => {
    await page.goto('/');
    const html = page.locator('html');
    await expect(html).toHaveAttribute('lang', 'en');
  });
});

test.describe('Navigation', () => {
  test('header contains logo link to home', async ({ page }) => {
    await page.goto('/');
    const logo = page
      .locator('header')
      .getByRole('link', { name: 'EcommerceGo' });
    await expect(logo).toBeVisible();
    await expect(logo).toHaveAttribute('href', '/');
  });

  test('header contains Products nav link', async ({ page }) => {
    await page.goto('/');
    // Products link is in the Layer 3 category nav (desktop only)
    const productsLink = page
      .locator('header nav')
      .getByRole('link', { name: 'Products' });
    await expect(productsLink).toBeVisible();
    await expect(productsLink).toHaveAttribute('href', '/products');
  });

  test('header contains shopping cart link', async ({ page }) => {
    await page.goto('/');
    const cartLink = page
      .locator('header')
      .getByRole('link', { name: 'Shopping cart' });
    await expect(cartLink).toBeVisible();
    await expect(cartLink).toHaveAttribute('href', '/cart');
  });

  test('header contains Sign In link for unauthenticated users', async ({
    page,
  }) => {
    await page.goto('/');
    const signInLink = page
      .locator('header')
      .getByRole('link', { name: 'Sign In' });
    await expect(signInLink).toBeVisible();
    await expect(signInLink).toHaveAttribute('href', '/auth/login');
  });

  test('header contains wishlist link', async ({ page }) => {
    await page.goto('/');
    const wishlistLink = page
      .locator('header')
      .getByRole('link', { name: 'Wishlist' });
    await expect(wishlistLink).toBeVisible();
    await expect(wishlistLink).toHaveAttribute('href', '/wishlist');
  });

  test('clicking Products nav link navigates to products page', async ({
    page,
  }) => {
    await page.goto('/');
    await page
      .locator('header nav')
      .getByRole('link', { name: 'Products' })
      .click();
    await expect(page).toHaveURL('/products');
    await expect(
      page.getByRole('heading', { name: 'All Products' }),
    ).toBeVisible();
  });

  test('clicking cart link navigates to cart page', async ({ page }) => {
    await page.goto('/');
    await page
      .locator('header')
      .getByRole('link', { name: 'Shopping cart' })
      .click();
    await expect(page).toHaveURL('/cart');
    await expect(
      page.getByRole('heading', { name: 'Shopping Cart' }),
    ).toBeVisible();
  });

  test('clicking Sign In link navigates to login page', async ({ page }) => {
    await page.goto('/');
    await page
      .locator('header')
      .getByRole('link', { name: 'Sign In' })
      .click();
    await expect(page).toHaveURL('/auth/login');
    await expect(
      page.getByRole('heading', { name: 'Sign in to EcommerceGo' }),
    ).toBeVisible();
  });

  test('footer displays platform tagline', async ({ page }) => {
    await page.goto('/');
    const footer = page.locator('footer');
    await expect(footer).toContainText(
      'AI-driven open-source e-commerce platform',
    );
  });

  test('footer displays newsletter section', async ({ page }) => {
    await page.goto('/');
    const footer = page.locator('footer');
    await expect(footer.getByText('Subscribe to our newsletter')).toBeVisible();
  });
});
