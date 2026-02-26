import { test, expect } from '@playwright/test';

test.describe('Products Page', () => {
  test('products page loads with heading', async ({ page }) => {
    await page.goto('/products');
    await expect(
      page.getByRole('heading', { name: 'Products' }),
    ).toBeVisible();
  });

  test('products page shows coming soon message', async ({ page }) => {
    await page.goto('/products');
    await expect(page.getByText('coming soon')).toBeVisible();
  });

  test('products page mentions BFF API endpoint', async ({ page }) => {
    await page.goto('/products');
    await expect(page.getByText('/api/products')).toBeVisible();
  });

  test('products page displays placeholder skeleton grid', async ({
    page,
  }) => {
    await page.goto('/products');
    // The scaffold renders 8 placeholder skeleton cards
    const skeletons = page.locator('.animate-pulse');
    await expect(skeletons).toHaveCount(8);
  });

  test('Browse Products button on homepage navigates to products page', async ({
    page,
  }) => {
    await page.goto('/');
    await page.getByRole('link', { name: 'Browse Products' }).click();
    await expect(page).toHaveURL('/products');
    await expect(
      page.getByRole('heading', { name: 'Products' }),
    ).toBeVisible();
  });

  // TODO: Uncomment when product listing is implemented with real data
  // test('products page displays product cards with names and prices', async ({ page }) => {
  //   await page.goto('/products');
  //   const productCards = page.locator('[data-testid="product-card"]');
  //   await expect(productCards.first()).toBeVisible();
  // });

  // TODO: Uncomment when search functionality is implemented
  // test('search input filters products', async ({ page }) => {
  //   await page.goto('/products');
  //   await page.getByPlaceholder('Search products').fill('test');
  //   await page.keyboard.press('Enter');
  //   await expect(page).toHaveURL(/q=test/);
  // });
});
