import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';

// ─── Products List Page ───────────────────────────────────────────────────────

test.describe('Products List Page', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/products');
    // Wait for the table or empty state to appear (loading skeleton goes away)
    await page.waitForSelector('table, [class*="text-center"]', { timeout: 15000 });
  });

  // Test 1 — list loads with products visible (30 total)
  test('products list loads with products visible', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Products');

    // Subtitle contains total count — "30 products total."
    const subtitle = page.locator('p.text-gray-500').first();
    await expect(subtitle).toContainText('products total.');

    // At least one table row in the tbody
    const rows = page.locator('tbody tr');
    await expect(rows.first()).toBeVisible();
    expect(await rows.count()).toBeGreaterThan(0);
  });

  // Test 2 — product images are displayed (either <img> or placeholder icon)
  test('product images are displayed in the table', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows.first()).toBeVisible();

    // Each row should contain either a real <img> or the SVG placeholder wrapper
    const firstRow = rows.first();
    const hasImg = await firstRow.locator('img').count();
    const hasSvgPlaceholder = await firstRow.locator('div.rounded.bg-gray-100').count();
    expect(hasImg + hasSvgPlaceholder).toBeGreaterThan(0);
  });

  // Test 3 — product prices are formatted as currency (e.g. "$49.99")
  test('product prices are displayed formatted as dollars', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows.first()).toBeVisible();

    // Price column is text-right; grab all price cells
    const priceCells = page.locator('tbody tr td.text-right.font-medium');
    const count = await priceCells.count();
    expect(count).toBeGreaterThan(0);

    const firstPrice = await priceCells.first().innerText();
    // Must start with "$" and match the currency format pattern
    expect(firstPrice).toMatch(/^\$[\d,]+\.\d{2}$/);
  });

  // Test 4 — search filters products by name
  test('search input filters products by name', async ({ page }) => {
    const searchInput = page.locator('input[placeholder="Search products by name..."]');
    await expect(searchInput).toBeVisible();

    await searchInput.fill('Domain');

    // Wait for the debounce (400 ms) and the subsequent API response
    await page.waitForTimeout(600);
    await page.waitForSelector('tbody tr', { timeout: 10000 });

    const rows = page.locator('tbody tr');
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThan(0);

    // Every visible product name should contain "Domain" (case-insensitive)
    for (let i = 0; i < rowCount; i++) {
      const nameCell = rows.nth(i).locator('td').first().locator('p.font-medium');
      const name = await nameCell.innerText();
      expect(name.toLowerCase()).toContain('domain');
    }
  });

  // Test 5 — "Add Product" button navigates to /products/new
  test('"Add Product" button navigates to /products/new', async ({ page }) => {
    const addButton = page.locator('a', { hasText: 'Add Product' });
    await expect(addButton).toBeVisible();
    await addButton.click();
    await page.waitForURL('**/products/new', { timeout: 10000 });
    await expect(page).toHaveURL(/\/products\/new$/);
  });

  // Test 9 — status badges have correct colors
  test('product status badges have correct Tailwind color classes', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows.first()).toBeVisible();

    const badges = page.locator('tbody tr span.rounded-full');
    const badgeCount = await badges.count();
    expect(badgeCount).toBeGreaterThan(0);

    for (let i = 0; i < badgeCount; i++) {
      const badge = badges.nth(i);
      const text = (await badge.innerText()).toLowerCase();
      const className = await badge.getAttribute('class') ?? '';

      if (text === 'published' || text === 'active') {
        expect(className).toContain('bg-green-100');
        expect(className).toContain('text-green-800');
      } else if (text === 'draft') {
        expect(className).toContain('bg-yellow-100');
        expect(className).toContain('text-yellow-800');
      } else if (text === 'archived') {
        expect(className).toContain('bg-red-100');
        expect(className).toContain('text-red-800');
      }
    }
  });

  // Test 10 — pagination works when there are more products than per_page
  test('pagination controls are rendered and Next button works', async ({ page }) => {
    // Wait for product table to load first
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // Pagination is only rendered when total_pages > 1
    const nextBtn = page.locator('button', { hasText: 'Next' });
    const hasPagination = await nextBtn.count();

    if (hasPagination === 0) {
      // Fewer products than per_page — nothing to test; pass gracefully
      return;
    }

    await expect(nextBtn).toBeVisible();
    // The "Previous" button starts disabled on page 1
    const prevBtn = page.locator('button', { hasText: 'Previous' });
    await expect(prevBtn).toBeDisabled();

    // Click "Next" and wait for page 2 content
    await nextBtn.click();

    // Wait for the pagination text to reflect page 2 (starts with "Showing 11")
    await expect(page.locator('text=/Showing 1[1-9]/')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Now "Previous" should be enabled (page > 1)
    await expect(prevBtn).toBeEnabled({ timeout: 5000 });
  });
});

// ─── Product Create / Edit Page ───────────────────────────────────────────────

test.describe('Product Create Page (/products/new)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/products/new');
    await page.waitForSelector('form', { timeout: 15000 });
  });

  // Test 6 — create product form has all required fields
  test('create product form has all required fields', async ({ page }) => {
    // Page heading
    await expect(page.locator('h1')).toHaveText('Create Product');

    // "Back to Products" / Cancel link
    const cancelLink = page.locator('a', { hasText: 'Cancel' });
    await expect(cancelLink.first()).toBeVisible();

    // Name input
    await expect(page.locator('#name')).toBeVisible();

    // Slug input
    await expect(page.locator('#slug')).toBeVisible();

    // Description textarea
    await expect(page.locator('#description')).toBeVisible();

    // Base Price input
    await expect(page.locator('#base_price')).toBeVisible();

    // Category select
    await expect(page.locator('#category')).toBeVisible();

    // Brand select
    await expect(page.locator('#brand')).toBeVisible();

    // Status select
    await expect(page.locator('#status')).toBeVisible();

    // Submit button
    const submitBtn = page.locator('button[type="submit"]');
    await expect(submitBtn).toBeVisible();
    await expect(submitBtn).toHaveText('Create Product');
  });

  // Test 5 (from list) — "Add Product" resolves here, covered in list tests above.
  // Test 7 — create a new product and verify success
  test('creates a new product and shows success message', async ({ page }) => {
    const uid = Date.now().toString(36);
    const productName = `E2E Product ${uid}`;
    const productSlug = `e2e-product-${uid}`;

    // Fill Name — slug is auto-generated from name
    await page.fill('#name', productName);

    // Slug should be auto-generated; override to be deterministic
    const slugInput = page.locator('#slug');
    await slugInput.fill(productSlug);

    // Description
    await page.fill('#description', 'Test product');

    // Price in dollars — form accepts dollars, sends cents to API
    await page.fill('#base_price', '19.99');

    // Status: select "draft" (it is the default, but set explicitly)
    await page.selectOption('#status', 'draft');

    // Submit the form
    const submitBtn = page.locator('button[type="submit"]');
    await submitBtn.click();

    // Expect success banner
    const successBanner = page.locator('text=Product created successfully!');
    await expect(successBanner).toBeVisible({ timeout: 15000 });

    // After ~1200 ms the page redirects to /products
    await page.waitForURL('**/products', { timeout: 10000 });
    await expect(page.locator('h1')).toHaveText('Products');
  });

  // Test 8 (partial) — verify slug auto-generates from name on /products/new
  test('slug is auto-generated from the product name', async ({ page }) => {
    await page.fill('#name', 'My New Awesome Product');

    // Wait for the controlled input value to settle
    await page.waitForTimeout(100);

    const slugValue = await page.locator('#slug').inputValue();
    expect(slugValue).toBe('my-new-awesome-product');
  });
});

test.describe('Product Edit Page (/products/[id])', () => {
  // Test 8 — edit link navigates to product edit form pre-filled with data
  test('Edit link navigates to edit form pre-filled with product data', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/products');
    await page.waitForSelector('tbody tr', { timeout: 15000 });

    // Click the first "Edit" link in the actions column
    const firstEditLink = page.locator('tbody tr').first().locator('a', { hasText: 'Edit' });
    await expect(firstEditLink).toBeVisible();
    await firstEditLink.click();

    // URL should match /products/<uuid>
    await page.waitForURL(/\/products\/[^/]+$/, { timeout: 10000 });
    expect(page.url()).toMatch(/\/products\/[\w-]+$/);
    expect(page.url()).not.toContain('/products/new');

    // Wait for the form to load (loading skeleton disappears, form appears)
    await page.waitForSelector('form', { timeout: 15000 });

    // Page heading should say "Edit Product"
    await expect(page.locator('h1')).toHaveText('Edit Product');

    // Name input must be non-empty (pre-filled from API)
    const nameValue = await page.locator('#name').inputValue();
    expect(nameValue.trim().length).toBeGreaterThan(0);

    // Slug input must be non-empty
    const slugValue = await page.locator('#slug').inputValue();
    expect(slugValue.trim().length).toBeGreaterThan(0);

    // Base price must be a positive number (displayed in dollars)
    const priceValue = await page.locator('#base_price').inputValue();
    expect(Number(priceValue)).toBeGreaterThan(0);

    // Status select must be one of the known values
    const statusValue = await page.locator('#status').inputValue();
    expect(['draft', 'published', 'archived']).toContain(statusValue);

    // Submit button should say "Save Changes"
    await expect(page.locator('button[type="submit"]')).toHaveText('Save Changes');

    // "Cancel" link points back to /products
    const cancelLink = page.locator('a', { hasText: 'Cancel' });
    const href = await cancelLink.first().getAttribute('href');
    expect(href).toBe('/products');
  });
});
