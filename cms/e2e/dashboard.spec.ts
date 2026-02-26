import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';

// ─── Dashboard E2E Tests ────────────────────────────────────────────────────
//
// Prerequisites: backend running with seed data (30 products, 7 orders,
// 3+ active campaigns). The CMS proxies API calls via Next.js rewrites:
//   /gateway/* → gateway /api/v1/*

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    // loginAsAdmin already lands on /dashboard; ensure we are there.
    await page.waitForURL('**/dashboard', { timeout: 10000 });
  });

  // ─── 1. Page loads with all four stat cards visible ─────────────────────

  test('dashboard loads with stats cards visible', async ({ page }) => {
    // Page heading
    await expect(
      page.getByRole('heading', { name: 'Dashboard', exact: true }),
    ).toBeVisible();

    // Subtitle
    await expect(
      page.getByText('Overview of your store performance and recent activity.'),
    ).toBeVisible();

    // All four stat card titles must be present
    await expect(page.getByText('Total Products')).toBeVisible();
    await expect(page.getByText('Total Orders')).toBeVisible();
    await expect(page.getByText('Active Campaigns')).toBeVisible();
    await expect(page.getByText('Low Stock Items')).toBeVisible();
  });

  // ─── 2. Stats cards show meaningful numeric values ───────────────────────

  test('stats cards show numeric values — products and orders counts are greater than zero', async ({
    page,
  }) => {
    // Wait for the loading spinner to disappear, indicating data has loaded.
    await expect(page.getByText('Loading dashboard...')).not.toBeVisible({
      timeout: 15000,
    });

    // Locate the stat cards by their containing structure: the <p> that
    // holds the title is immediately followed by the value <p>.
    const productCard = page
      .locator('div.bg-white.rounded-lg')
      .filter({ hasText: 'Total Products' });
    const orderCard = page
      .locator('div.bg-white.rounded-lg')
      .filter({ hasText: 'Total Orders' });

    // Both cards must be visible.
    await expect(productCard).toBeVisible();
    await expect(orderCard).toBeVisible();

    // Extract the numeric value text from each card (the bold large number).
    const productValueText = await productCard
      .locator('p.text-2xl')
      .innerText();
    const orderValueText = await orderCard.locator('p.text-2xl').innerText();

    // Strip any locale-formatting commas before parsing.
    const productCount = parseInt(productValueText.replace(/,/g, ''), 10);
    const orderCount = parseInt(orderValueText.replace(/,/g, ''), 10);

    expect(productCount).toBeGreaterThan(0);
    expect(orderCount).toBeGreaterThan(0);
  });

  // ─── 3. Recent Orders table is present with at least one row ─────────────

  test('Recent Orders table is present with at least one row', async ({
    page,
  }) => {
    // Wait for data to load.
    await expect(page.getByText('Loading dashboard...')).not.toBeVisible({
      timeout: 15000,
    });

    // Section heading
    await expect(
      page.getByRole('heading', { name: 'Recent Orders' }),
    ).toBeVisible();

    // Table header columns
    const table = page.locator('table');
    await expect(table).toBeVisible();
    await expect(table.getByRole('columnheader', { name: /Order ID/i })).toBeVisible();
    await expect(table.getByRole('columnheader', { name: /Date/i })).toBeVisible();
    await expect(table.getByRole('columnheader', { name: /Status/i })).toBeVisible();
    await expect(table.getByRole('columnheader', { name: /Items/i })).toBeVisible();

    // At least one data row in tbody.
    const rows = table.locator('tbody tr');
    await expect(rows.first()).toBeVisible();
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThanOrEqual(1);
  });

  // ─── 4. Order ID links are clickable and navigate to order detail ─────────

  test('Order ID link navigates to the order detail page', async ({ page }) => {
    await expect(page.getByText('Loading dashboard...')).not.toBeVisible({
      timeout: 15000,
    });

    // The first Order ID link is inside the tbody.
    const firstOrderLink = page
      .locator('table tbody tr')
      .first()
      .locator('a');

    await expect(firstOrderLink).toBeVisible();

    // Capture the href to validate the target path.
    const href = await firstOrderLink.getAttribute('href');
    expect(href).toMatch(/^\/orders\//);

    // Click and verify navigation.
    await firstOrderLink.click();
    await page.waitForURL('**/orders/**', { timeout: 10000 });
    expect(page.url()).toContain('/orders/');
  });

  // ─── 5. Sidebar navigation links are all present and visible ─────────────

  test('sidebar navigation links are all present and visible', async ({
    page,
  }) => {
    const sidebar = page.locator('aside');

    // Brand / logo link
    await expect(sidebar.getByText('EcommerceGo CMS')).toBeVisible();

    // All seven nav labels
    const expectedNavLabels = [
      'Dashboard',
      'Products',
      'Categories',
      'Brands',
      'Campaigns',
      'Orders',
      'Inventory',
    ];

    for (const label of expectedNavLabels) {
      await expect(
        sidebar.getByRole('link', { name: label, exact: true }),
      ).toBeVisible();
    }
  });

  // ─── 6. Current user name and email displayed in the sidebar ─────────────

  test('current user name and email are displayed in the sidebar', async ({
    page,
  }) => {
    const sidebar = page.locator('aside');

    // The user block at the bottom of the sidebar shows the admin's full name
    // and email address.
    await expect(sidebar.getByText('Admin User')).toBeVisible();
    await expect(sidebar.getByText('admin@ecommerce.com')).toBeVisible();
  });

  // ─── 7. Quick Action links navigate to the correct pages ─────────────────

  test('Quick Actions — "Add Product" navigates to products page', async ({
    page,
  }) => {
    const addProductLink = page.getByRole('link', { name: 'Add Product' });
    await expect(addProductLink).toBeVisible();

    // The link href must point to /products/new (the product creation page).
    const href = await addProductLink.getAttribute('href');
    expect(href).toBe('/products/new');

    await addProductLink.click();
    await page.waitForURL('**/products/new', { timeout: 10000 });
    expect(page.url()).toContain('/products/new');
  });

  test('Quick Actions — "View Orders" navigates to the orders page', async ({
    page,
  }) => {
    const viewOrdersLink = page.getByRole('link', { name: 'View Orders' });
    await expect(viewOrdersLink).toBeVisible();

    const href = await viewOrdersLink.getAttribute('href');
    expect(href).toBe('/orders');

    await viewOrdersLink.click();
    await page.waitForURL('**/orders', { timeout: 10000 });
    expect(page.url()).toContain('/orders');
  });

  test('Quick Actions — "Create Campaign" navigates to the campaigns page', async ({
    page,
  }) => {
    // The actual label rendered in the component is "Create Campaign".
    const campaignLink = page.getByRole('link', { name: 'Create Campaign' });
    await expect(campaignLink).toBeVisible();

    const href = await campaignLink.getAttribute('href');
    expect(href).toBe('/campaigns/new');

    await campaignLink.click();
    await page.waitForURL('**/campaigns/new', { timeout: 10000 });
    expect(page.url()).toContain('/campaigns/new');
  });

  // ─── Bonus: "View all" link in Recent Orders section ─────────────────────

  test('"View all" link in Recent Orders section navigates to orders page', async ({
    page,
  }) => {
    const viewAllLink = page.getByRole('link', { name: 'View all' });
    await expect(viewAllLink).toBeVisible();

    const href = await viewAllLink.getAttribute('href');
    expect(href).toBe('/orders');

    await viewAllLink.click();
    await page.waitForURL('**/orders', { timeout: 10000 });
    expect(page.url()).toContain('/orders');
  });
});
