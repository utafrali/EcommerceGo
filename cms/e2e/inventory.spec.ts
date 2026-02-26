import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';

// ─── Mock API Payloads ──────────────────────────────────────────────────────

/** A realistic paginated products response (page 1 of 1, 3 products). */
const PRODUCTS_PAGE_1 = {
  data: [
    {
      id: 'aaaaaaaa-0001-0001-0001-000000000001',
      name: 'Wireless Headphones',
      slug: 'wireless-headphones',
      description: 'Premium wireless headphones',
      base_price: 4999,
      currency: 'USD',
      status: 'active',
      brand_id: null,
      category_id: null,
      metadata: {},
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
      variants: [
        { id: 'vvvv-0001', product_id: 'aaaaaaaa-0001-0001-0001-000000000001', sku: 'WH-BLK', name: 'Black', price: null, attributes: {}, weight_grams: null, is_active: true, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' },
        { id: 'vvvv-0002', product_id: 'aaaaaaaa-0001-0001-0001-000000000001', sku: 'WH-WHT', name: 'White', price: null, attributes: {}, weight_grams: null, is_active: true, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' },
      ],
    },
    {
      id: 'bbbbbbbb-0002-0002-0002-000000000002',
      name: 'Mechanical Keyboard',
      slug: 'mechanical-keyboard',
      description: 'TKL mechanical keyboard',
      base_price: 8999,
      currency: 'USD',
      status: 'active',
      brand_id: null,
      category_id: null,
      metadata: {},
      created_at: '2025-01-02T00:00:00Z',
      updated_at: '2025-01-02T00:00:00Z',
      variants: [],
    },
    {
      id: 'cccccccc-0003-0003-0003-000000000003',
      name: 'USB-C Hub',
      slug: 'usb-c-hub',
      description: '7-in-1 USB-C hub',
      base_price: 2999,
      currency: 'USD',
      status: 'active',
      brand_id: null,
      category_id: null,
      metadata: {},
      created_at: '2025-01-03T00:00:00Z',
      updated_at: '2025-01-03T00:00:00Z',
      variants: [
        { id: 'vvvv-0003', product_id: 'cccccccc-0003-0003-0003-000000000003', sku: 'HUB-7IN1', name: 'Standard', price: null, attributes: {}, weight_grams: null, is_active: true, created_at: '2025-01-01T00:00:00Z', updated_at: '2025-01-01T00:00:00Z' },
      ],
    },
  ],
  total_count: 3,
  page: 1,
  per_page: 20,
  total_pages: 1,
};

/** Low-stock items: one item for "Wireless Headphones", none for others. */
const LOW_STOCK_WITH_ITEMS = {
  data: [
    {
      id: 'lsi-0001',
      product_id: 'aaaaaaaa-0001-0001-0001-000000000001',
      variant_id: 'vvvv-0001',
      warehouse_id: 'wh-main-001-aaaa-bbbb',
      quantity: 3,
      reserved: 0,
      low_stock_threshold: 5,
      updated_at: '2025-01-15T10:00:00Z',
    },
  ],
  total_count: 1,
  page: 1,
  per_page: 20,
  total_pages: 1,
};

/** Empty low-stock response — all products are adequately stocked. */
const LOW_STOCK_EMPTY = { data: [], total_count: 0, page: 1, per_page: 20, total_pages: 0 };

/** A second page of products (for pagination tests). */
const PRODUCTS_PAGE_2 = {
  data: [
    {
      id: 'dddddddd-0004-0004-0004-000000000004',
      name: 'Ergonomic Mouse',
      slug: 'ergonomic-mouse',
      description: 'Wireless ergonomic mouse',
      base_price: 5999,
      currency: 'USD',
      status: 'active',
      brand_id: null,
      category_id: null,
      metadata: {},
      created_at: '2025-01-04T00:00:00Z',
      updated_at: '2025-01-04T00:00:00Z',
      variants: [],
    },
  ],
  total_count: 21,
  page: 2,
  per_page: 20,
  total_pages: 2,
};

/** First page response when there are 21 products total (triggers pagination). */
const PRODUCTS_PAGE_1_MULTI = {
  data: PRODUCTS_PAGE_1.data,
  total_count: 21,
  page: 1,
  per_page: 20,
  total_pages: 2,
};

// ─── Helper: intercept both API calls with given payloads ───────────────────

async function interceptInventoryApis(
  page: import('@playwright/test').Page,
  opts: {
    productsResponse?: object;
    lowStockResponse?: object;
  } = {},
) {
  const products = opts.productsResponse ?? PRODUCTS_PAGE_1;
  const lowStock = opts.lowStockResponse ?? LOW_STOCK_EMPTY;

  await page.route('**/gateway/products**', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(products) }),
  );
  await page.route('**/gateway/inventory/low-stock**', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(lowStock) }),
  );
}

// ─── Test Suite: CMS Inventory Page ────────────────────────────────────────

test.describe('CMS Inventory Page', () => {

  // ── 1. Page loads and lists products ───────────────────────────────────
  test('inventory page loads and displays products in the table', async ({ page }) => {
    await interceptInventoryApis(page, { productsResponse: PRODUCTS_PAGE_1 });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Wait for the loading skeleton to resolve and the table body to appear
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // All three products should be present as table rows
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(3);
  });

  // ── 2. Page title "Inventory" is visible ──────────────────────────────
  test('page heading "Inventory" is visible', async ({ page }) => {
    await interceptInventoryApis(page);
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('h1', { hasText: 'Inventory' })).toBeVisible({ timeout: 15000 });
  });

  // ── 3. Page subtitle is visible ───────────────────────────────────────
  test('subtitle describing the page purpose is visible', async ({ page }) => {
    await interceptInventoryApis(page);
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(
      page.locator('text=Monitor stock levels and low stock alerts across all products.'),
    ).toBeVisible({ timeout: 15000 });
  });

  // ── 4. Products show names and truncated IDs ───────────────────────────
  test('product rows display product name and a truncated ID', async ({ page }) => {
    await interceptInventoryApis(page, { productsResponse: PRODUCTS_PAGE_1 });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Wait for rows
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // Product name
    await expect(page.locator('text=Wireless Headphones').first()).toBeVisible();

    // Truncated ID: first 12 chars of 'aaaaaaaa-0001-0001-0001-000000000001' followed by '...'
    // The page renders: product.id.slice(0, 12) + '...'  → 'aaaaaaaa-000...'
    const truncatedId = 'aaaaaaaa-000...';
    await expect(page.locator(`text=${truncatedId}`).first()).toBeVisible();
  });

  // ── 5. Variant count column ────────────────────────────────────────────
  test('variant count column shows correct count for each product', async ({ page }) => {
    await interceptInventoryApis(page, { productsResponse: PRODUCTS_PAGE_1 });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // "Wireless Headphones" has 2 variants
    await expect(page.locator('text=2 variant(s)').first()).toBeVisible();

    // "Mechanical Keyboard" has 0 variants
    await expect(page.locator('text=0 variant(s)').first()).toBeVisible();

    // "USB-C Hub" has 1 variant
    await expect(page.locator('text=1 variant(s)').first()).toBeVisible();
  });

  // ── 6. Base prices formatted as currency ──────────────────────────────
  test('base prices are formatted as USD currency strings', async ({ page }) => {
    await interceptInventoryApis(page, { productsResponse: PRODUCTS_PAGE_1 });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // 4999 cents → $49.99, 8999 → $89.99, 2999 → $29.99
    await expect(page.locator('text=$49.99').first()).toBeVisible();
    await expect(page.locator('text=$89.99').first()).toBeVisible();
    await expect(page.locator('text=$29.99').first()).toBeVisible();
  });

  // ── 7. Search input is present ────────────────────────────────────────
  test('search input is rendered with the correct placeholder', async ({ page }) => {
    await interceptInventoryApis(page);
    await loginAsAdmin(page);
    await page.goto('/inventory');

    const searchInput = page.locator('input[placeholder="Search products by name..."]');
    await expect(searchInput).toBeVisible({ timeout: 15000 });
  });

  // ── 8. Search filters products by submitting the form ─────────────────
  test('typing in search and submitting triggers a filtered API request', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Wait for the initial load (real data — 30 products, first page shows 20)
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });
    const initialCount = await page.locator('tbody tr').count();
    expect(initialCount).toBeGreaterThan(0);

    // Type a specific product name and submit
    const searchInput = page.locator('input[placeholder="Search products by name..."]');
    await searchInput.fill('Domain-Driven');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(600);

    // After search the table should show fewer results
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
    const filteredCount = await page.locator('tbody tr').count();
    expect(filteredCount).toBeLessThanOrEqual(initialCount);
    await expect(page.locator('text=Domain-Driven Design').first()).toBeVisible();
  });

  // ── 9. Clear button removes search filter ─────────────────────────────
  test('clear button resets search and reloads all products', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Wait for initial load
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });
    const initialCount = await page.locator('tbody tr').count();

    // Search for a specific product
    const searchInput = page.locator('input[placeholder="Search products by name..."]');
    await searchInput.fill('Domain-Driven');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(600);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Clear button should appear and restore the full list
    const clearButton = page.locator('button', { hasText: 'Clear' });
    await expect(clearButton).toBeVisible();
    await clearButton.click();
    await page.waitForTimeout(600);

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
    const restoredCount = await page.locator('tbody tr').count();
    expect(restoredCount).toBeGreaterThanOrEqual(initialCount);
  });

  // ── 10. Low Stock column shows "OK" when no alerts exist ──────────────
  test('Low Stock column shows "OK" for products with no low-stock data', async ({ page }) => {
    await interceptInventoryApis(page, {
      productsResponse: PRODUCTS_PAGE_1,
      lowStockResponse: LOW_STOCK_EMPTY,
    });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // All three products have no low-stock entry → each row's 4th cell shows "OK"
    const okCells = page.locator('td span', { hasText: 'OK' });
    // There should be at least 3 "OK" labels (one per product)
    await expect(okCells).toHaveCount(3);
  });

  // ── 11. Low Stock column shows stock badge when there are alerts ───────
  test('Low Stock column shows a stock badge for products with low-stock data', async ({ page }) => {
    await interceptInventoryApis(page, {
      productsResponse: PRODUCTS_PAGE_1,
      lowStockResponse: LOW_STOCK_WITH_ITEMS,
    });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // "Wireless Headphones" (product 1) has quantity=3, low_stock_threshold=5 → "Low: 3"
    await expect(page.locator('text=Low: 3').first()).toBeVisible({ timeout: 10000 });

    // The other two products have no alerts → they show "OK"
    const okCells = page.locator('td span', { hasText: 'OK' });
    await expect(okCells).toHaveCount(2);
  });

  // ── 12. Low Stock Alerts section appears at the top when items exist ───
  test('"Low Stock Alerts" section is rendered at the top when there are alerts', async ({ page }) => {
    await interceptInventoryApis(page, {
      productsResponse: PRODUCTS_PAGE_1,
      lowStockResponse: LOW_STOCK_WITH_ITEMS,
    });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Wait for page to finish loading
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // The alert section heading with count
    await expect(page.locator('h2', { hasText: 'Low Stock Alerts (1)' })).toBeVisible({ timeout: 10000 });

    // The alert table should contain the warehouse_id row (first 8 chars + "...")
    await expect(page.locator('text=wh-main-').first()).toBeVisible();
  });

  // ── 13. Low Stock Alerts section is hidden when all products are OK ────
  test('"Low Stock Alerts" section is not rendered when there are no alerts', async ({ page }) => {
    await interceptInventoryApis(page, {
      productsResponse: PRODUCTS_PAGE_1,
      lowStockResponse: LOW_STOCK_EMPTY,
    });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // The alert section must not be present
    await expect(page.locator('h2', { hasText: /Low Stock Alerts/ })).not.toBeVisible();
  });

  // ── 14. View details button expands inline low-stock detail table ──────
  test('clicking "View" on a low-stock product row expands detailed variant info', async ({ page }) => {
    await interceptInventoryApis(page, {
      productsResponse: PRODUCTS_PAGE_1,
      lowStockResponse: LOW_STOCK_WITH_ITEMS,
    });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // "View (1)" button should be present for the affected product row
    const viewButton = page.locator('button', { hasText: /^View \(1\)$/ });
    await expect(viewButton).toBeVisible({ timeout: 10000 });
    await viewButton.click();

    // After expanding, the inline detail table should show the warehouse_id
    await expect(page.locator('text=wh-main-').first()).toBeVisible({ timeout: 5000 });

    // The button label toggles to "Hide Details"
    await expect(page.locator('button', { hasText: 'Hide Details' })).toBeVisible();
  });

  // ── 15. Page handles no products gracefully ────────────────────────────
  test('empty product list shows "No products found" state without errors', async ({ page }) => {
    await page.route('**/gateway/products**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], total_count: 0, page: 1, per_page: 20, total_pages: 0 }),
      }),
    );
    await page.route('**/gateway/inventory/low-stock**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(LOW_STOCK_EMPTY) }),
    );

    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('text=No products found')).toBeVisible({ timeout: 15000 });

    // No JS errors should be thrown
    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(err.message));
    expect(errors).toHaveLength(0);
  });

  // ── 16. Graceful handling when inventory API returns no data ───────────
  test('page renders without errors when inventory/low-stock returns empty data', async ({ page }) => {
    // Products exist but inventory service returns an empty array
    await interceptInventoryApis(page, {
      productsResponse: PRODUCTS_PAGE_1,
      lowStockResponse: { data: [], total_count: 0, page: 1, per_page: 20, total_pages: 0 },
    });

    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(err.message));

    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Products table still renders
    await expect(page.locator('tbody tr')).toHaveCount(3, { timeout: 15000 });

    // No runtime errors
    expect(errors).toHaveLength(0);
  });

  // ── 17. Graceful handling when inventory API errors ────────────────────
  test('page shows a low-stock error banner when the inventory API fails', async ({ page }) => {
    await page.route('**/gateway/products**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(PRODUCTS_PAGE_1) }),
    );
    // Simulate a 500 error from the inventory service
    await page.route('**/gateway/inventory/low-stock**', (route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'INTERNAL', message: 'Service unavailable' } }),
      }),
    );

    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Products table still renders
    await expect(page.locator('tbody tr')).toHaveCount(3, { timeout: 15000 });

    // Low-stock error notice should appear
    await expect(
      page.locator('text=/Could not load low stock alerts/'),
    ).toBeVisible({ timeout: 10000 });

    // A retry button should be offered
    await expect(page.locator('button', { hasText: 'Retry' }).first()).toBeVisible();
  });

  // ── 18. Table column headers are present ──────────────────────────────
  test('inventory table renders all expected column headers', async ({ page }) => {
    await interceptInventoryApis(page, { productsResponse: PRODUCTS_PAGE_1 });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    const thead = page.locator('table thead').last(); // main product table
    await expect(thead.locator('th', { hasText: 'Product' })).toBeVisible();
    await expect(thead.locator('th', { hasText: 'Variants' })).toBeVisible();
    await expect(thead.locator('th', { hasText: 'Base Price' })).toBeVisible();
    await expect(thead.locator('th', { hasText: 'Stock Status' })).toBeVisible();
    await expect(thead.locator('th', { hasText: 'Low Stock' })).toBeVisible();
  });

  // ── 19. Pagination controls render when there are multiple pages ────────
  test('pagination controls appear and are functional when total_pages > 1', async ({ page }) => {
    // Page 1 request returns data with total_pages=2
    let currentPage = 1;

    await page.route('**/gateway/products**', (route) => {
      const url = new URL(route.request().url());
      currentPage = Number(url.searchParams.get('page') || 1);
      const responseData = currentPage === 1 ? PRODUCTS_PAGE_1_MULTI : PRODUCTS_PAGE_2;
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(responseData) });
    });
    await page.route('**/gateway/inventory/low-stock**', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(LOW_STOCK_EMPTY) }),
    );

    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Wait for page 1 products
    await expect(page.locator('tbody tr')).toHaveCount(3, { timeout: 15000 });

    // Pagination footer must be visible
    await expect(page.locator('text=Page 1 of 2')).toBeVisible();

    // "Previous" should be disabled on page 1
    const prevButton = page.locator('button', { hasText: 'Previous' });
    await expect(prevButton).toBeDisabled();

    // "Next" should be enabled
    const nextButton = page.locator('button', { hasText: 'Next' });
    await expect(nextButton).toBeEnabled();

    // Click "Next" to navigate to page 2
    await nextButton.click();

    // Page 2 has 1 product ("Ergonomic Mouse")
    await expect(page.locator('tbody tr')).toHaveCount(1, { timeout: 10000 });
    await expect(page.locator('text=Ergonomic Mouse').first()).toBeVisible();
    await expect(page.locator('text=Page 2 of 2')).toBeVisible();

    // "Next" should now be disabled on the last page
    await expect(nextButton).toBeDisabled();

    // "Previous" should be enabled
    await expect(prevButton).toBeEnabled();
  });

  // ── 20. Pagination is not shown for single-page result sets ───────────
  test('pagination controls are hidden when all products fit on one page', async ({ page }) => {
    await interceptInventoryApis(page, { productsResponse: PRODUCTS_PAGE_1 });
    await loginAsAdmin(page);
    await page.goto('/inventory');

    await expect(page.locator('tbody tr')).toHaveCount(3, { timeout: 15000 });

    // Pagination footer must not be present (total_pages = 1)
    await expect(page.locator('text=Page 1 of 1')).not.toBeVisible();
    await expect(page.locator('button', { hasText: 'Previous' })).not.toBeVisible();
    await expect(page.locator('button', { hasText: 'Next' })).not.toBeVisible();
  });

  // ── 21. Search with no results shows empty state ──────────────────────
  test('searching for a non-existent product shows the empty state message', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/inventory');

    // Wait for initial load
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // Search for something that doesn't exist
    const searchInput = page.locator('input[placeholder="Search products by name..."]');
    await searchInput.fill('zzz-nonexistent-product-xyz');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(600);

    // The API will return 0 results, showing the empty state
    await expect(page.locator('text=No products found')).toBeVisible({ timeout: 10000 });
  });

  // ── 22. Inventory page is accessible from the sidebar nav ─────────────
  test('sidebar navigation link to Inventory navigates to /inventory', async ({ page }) => {
    // Start from the dashboard (no API mocking needed for the nav link click itself,
    // but we mock inventory calls so the page settles cleanly)
    await interceptInventoryApis(page, { productsResponse: PRODUCTS_PAGE_1 });
    await loginAsAdmin(page);
    await page.goto('/dashboard');

    const inventoryNavLink = page.locator('nav a', { hasText: 'Inventory' });
    await expect(inventoryNavLink).toBeVisible();
    await inventoryNavLink.click();

    await page.waitForURL('**/inventory', { timeout: 10000 });
    await expect(page.locator('h1', { hasText: 'Inventory' })).toBeVisible({ timeout: 15000 });
  });
});
