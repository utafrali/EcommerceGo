import { test, expect } from '@playwright/test';

test.describe('Products Page', () => {
  test('products page loads with heading', async ({ page }) => {
    await page.goto('/products');
    await expect(
      page.getByRole('heading', { name: 'All Products' }),
    ).toBeVisible();
  });

  test('products page shows breadcrumb navigation', async ({ page }) => {
    await page.goto('/products');
    const breadcrumb = page.locator('nav[aria-label="Breadcrumb"]');
    await expect(breadcrumb).toBeVisible();
    await expect(breadcrumb.getByText('Home')).toBeVisible();
    await expect(breadcrumb.getByText('Products')).toBeVisible();
  });

  test('products page breadcrumb Home links to homepage', async ({ page }) => {
    await page.goto('/products');
    const breadcrumb = page.locator('nav[aria-label="Breadcrumb"]');
    const homeLink = breadcrumb.getByRole('link', { name: 'Home' });
    await expect(homeLink).toBeVisible();
    await expect(homeLink).toHaveAttribute('href', '/');
  });

  test('products page handles empty or error state gracefully', async ({
    page,
  }) => {
    await page.goto('/products');
    // The page should display either product cards, an empty message, or an error banner — not crash
    const heading = page.getByRole('heading', { name: 'All Products' });
    await expect(heading).toBeVisible();
  });

  test('Shop Now button on homepage navigates to products page', async ({
    page,
  }) => {
    await page.goto('/');
    await page.getByRole('link', { name: 'Shop Now' }).click();
    await expect(page).toHaveURL('/products');
    await expect(
      page.getByRole('heading', { name: 'All Products' }),
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
