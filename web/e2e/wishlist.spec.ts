import { test, expect } from '@playwright/test';

test.describe('Wishlist Page', () => {
  test('wishlist page is accessible at /wishlist', async ({ page }) => {
    await page.goto('/wishlist');
    await expect(
      page.getByRole('heading', { name: 'My Wishlist' }),
    ).toBeVisible();
  });

  test('unauthenticated user sees sign-in prompt', async ({ page }) => {
    await page.goto('/wishlist');
    await expect(
      page.getByText('Sign in to view your wishlist'),
    ).toBeVisible();
    const signInLink = page.getByRole('link', { name: 'Sign In' });
    await expect(signInLink).toBeVisible();
    await expect(signInLink).toHaveAttribute('href', '/auth/login');
  });

  test('wishlist sign-in link navigates to login page', async ({ page }) => {
    await page.goto('/wishlist');
    await page.getByRole('link', { name: 'Sign In' }).click();
    await expect(page).toHaveURL('/auth/login');
    await expect(
      page.getByRole('heading', { name: 'Sign in to EcommerceGo' }),
    ).toBeVisible();
  });
});

test.describe('Wishlist Button on Product Cards', () => {
  test('product cards show heart/wishlist button', async ({ page }) => {
    await page.goto('/products');

    // Check if there are any product cards on the page
    const wishlistButtons = page.getByRole('button', {
      name: 'Add to wishlist',
    });
    const productLinks = page.locator('a[href^="/products/"]');
    const productCount = await productLinks.count();

    if (productCount === 0) {
      // No products available, skip gracefully
      test.skip();
      return;
    }

    // If products exist, there should be wishlist buttons
    await expect(wishlistButtons.first()).toBeVisible();
  });
});

test.describe('Wishlist Link in Header', () => {
  test('header wishlist link navigates to wishlist page', async ({ page }) => {
    await page.goto('/');
    const wishlistLink = page
      .locator('header')
      .getByRole('link', { name: 'Wishlist' });
    await expect(wishlistLink).toBeVisible();
    await wishlistLink.click();
    await expect(page).toHaveURL('/wishlist');
    await expect(
      page.getByRole('heading', { name: 'My Wishlist' }),
    ).toBeVisible();
  });
});
