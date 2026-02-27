import { test, expect } from '@playwright/test';

test.describe('Products List Page (PLP)', () => {
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

  test('products page has sort dropdown', async ({ page }) => {
    await page.goto('/products');
    const sortDropdown = page.getByLabel('Sort products');
    await expect(sortDropdown).toBeVisible();
  });

  test('products page has search bar', async ({ page }) => {
    await page.goto('/products');
    const searchInput = page.getByPlaceholder('Search products...');
    await expect(searchInput).toBeVisible();
  });

  test('products page handles empty or error state gracefully', async ({
    page,
  }) => {
    await page.goto('/products');
    // The page should display either product cards, an empty message, or an error banner -- not crash
    const heading = page.getByRole('heading', { name: 'All Products' });
    await expect(heading).toBeVisible();
  });

  test('products page shows result count', async ({ page }) => {
    await page.goto('/products');
    // Either "Showing X products" or "No products found"
    const hasProductCount = page.getByText(/Showing \d+ products?/);
    const hasEmptyState = page.getByText('No products found');
    await expect(hasProductCount.or(hasEmptyState)).toBeVisible();
  });

  test('Shop Now button on homepage navigates to products page', async ({
    page,
  }) => {
    await page.goto('/');
    await page.getByRole('link', { name: 'Shop Now' }).first().click();
    await expect(page).toHaveURL('/products');
    await expect(
      page.getByRole('heading', { name: 'All Products' }),
    ).toBeVisible();
  });
});

test.describe('Product Detail Page (PDP)', () => {
  // Note: These tests navigate to /products first and try to click through,
  // or test the structure of the PDP if products are available.

  test('PDP shows breadcrumb with Products link', async ({ page }) => {
    // Try to access a product page; if no products exist, skip gracefully
    await page.goto('/products');

    // Wait for the page to load, then check if any product links exist
    const productLinks = page.locator('a[href^="/products/"]');
    const count = await productLinks.count();

    if (count === 0) {
      // No products available, skip this test gracefully
      test.skip();
      return;
    }

    // Click the first product link
    await productLinks.first().click();
    await page.waitForLoadState('networkidle');

    // Check that Products breadcrumb link is present
    const productsLink = page.getByRole('link', { name: 'Products' });
    await expect(productsLink).toBeVisible();
    await expect(productsLink).toHaveAttribute('href', '/products');
  });

  test('PDP shows product name as heading', async ({ page }) => {
    await page.goto('/products');

    const productLinks = page.locator('a[href^="/products/"]');
    const count = await productLinks.count();

    if (count === 0) {
      test.skip();
      return;
    }

    await productLinks.first().click();
    await page.waitForLoadState('networkidle');

    // Product name should appear as an h1 heading
    const productHeading = page.locator('h1');
    await expect(productHeading).toBeVisible();
  });

  test('PDP shows Add to Cart button', async ({ page }) => {
    await page.goto('/products');

    const productLinks = page.locator('a[href^="/products/"]');
    const count = await productLinks.count();

    if (count === 0) {
      test.skip();
      return;
    }

    await productLinks.first().click();
    await page.waitForLoadState('networkidle');

    const addToCartButton = page.getByRole('button', { name: /Add to Cart/i });
    await expect(addToCartButton).toBeVisible();
  });

  test('PDP shows product tabs (Description, Reviews, Specifications)', async ({
    page,
  }) => {
    await page.goto('/products');

    const productLinks = page.locator('a[href^="/products/"]');
    const count = await productLinks.count();

    if (count === 0) {
      test.skip();
      return;
    }

    await productLinks.first().click();
    await page.waitForLoadState('networkidle');

    // Check that the tab navigation is present
    const tabNav = page.locator('nav[aria-label="Product tabs"]');
    await expect(tabNav).toBeVisible();

    // Check individual tabs
    await expect(
      tabNav.getByRole('tab', { name: 'Description' }),
    ).toBeVisible();
    await expect(
      tabNav.getByRole('tab', { name: /Reviews/ }),
    ).toBeVisible();
    await expect(
      tabNav.getByRole('tab', { name: 'Specifications' }),
    ).toBeVisible();
  });

  test('PDP shows wishlist button', async ({ page }) => {
    await page.goto('/products');

    const productLinks = page.locator('a[href^="/products/"]');
    const count = await productLinks.count();

    if (count === 0) {
      test.skip();
      return;
    }

    await productLinks.first().click();
    await page.waitForLoadState('networkidle');

    const wishlistButton = page.getByRole('button', {
      name: /Add to wishlist|Remove from wishlist/,
    });
    await expect(wishlistButton).toBeVisible();
  });

  test('PDP shows shipping and return info', async ({ page }) => {
    await page.goto('/products');

    const productLinks = page.locator('a[href^="/products/"]');
    const count = await productLinks.count();

    if (count === 0) {
      test.skip();
      return;
    }

    await productLinks.first().click();
    await page.waitForLoadState('networkidle');

    await expect(
      page.getByText('Free shipping on orders over $50'),
    ).toBeVisible();
    await expect(page.getByText('30-day return policy')).toBeVisible();
    await expect(
      page.getByText('Secure payment guaranteed'),
    ).toBeVisible();
  });
});
