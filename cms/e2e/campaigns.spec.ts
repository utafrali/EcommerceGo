import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';

// ─── Date Helpers ───────────────────────────────────────────────────────────

/** Returns today's date in YYYY-MM-DD format (for date inputs). */
function todayIso(): string {
  return new Date().toISOString().split('T')[0];
}

/** Returns a date N days from today in YYYY-MM-DD format. */
function futureDateIso(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().split('T')[0];
}

// ─── Test Suite: CMS Campaigns ──────────────────────────────────────────────

test.describe('CMS Campaigns', () => {

  // ── 1. Campaigns list page loads with correct heading and subtitle ─────────
  test('campaigns list loads with heading and subtitle', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('h1', { hasText: 'Campaigns' })).toBeVisible();
    await expect(
      page.locator('p', { hasText: 'Manage discount campaigns and promotional codes.' }),
    ).toBeVisible();
  });

  // ── 2. Existing campaigns are visible in the table ────────────────────────
  test('existing campaigns are visible in the list', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    // Wait for the table to finish loading (no skeleton pulse divs remain)
    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // All three seed campaigns should appear as rows
    await expect(page.locator('td', { hasText: 'Free Shipping' })).toBeVisible();
    await expect(page.locator('tbody').getByText('Summer Sale', { exact: true })).toBeVisible();
    await expect(page.locator('td', { hasText: 'Welcome Discount' })).toBeVisible();
  });

  // ── 3. Campaign promo codes are displayed in the table ────────────────────
  test('campaign codes are displayed correctly', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // Codes are rendered inside <code> elements
    await expect(page.locator('code', { hasText: 'FREESHIP' })).toBeVisible();
    await expect(page.locator('code', { hasText: 'SUMMER20' })).toBeVisible();
    await expect(page.locator('code', { hasText: 'WELCOME10' })).toBeVisible();
  });

  // ── 4. Campaign types are displayed as human-readable labels ──────────────
  test('campaign types are shown as Percentage and Fixed Amount', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // "Percentage" appears for SUMMER20 and WELCOME10
    const percentageCells = page.locator('td', { hasText: 'Percentage' });
    await expect(percentageCells.first()).toBeVisible();

    // "Fixed Amount" appears for FREESHIP
    await expect(page.locator('td', { hasText: 'Fixed Amount' })).toBeVisible();
  });

  // ── 5. Status badges are color-coded per campaign status ─────────────────
  test('status badges are color-coded', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // Active badge: bg-green-100 text-green-800
    const activeBadge = page
      .locator('span.rounded-full', { hasText: 'Active' })
      .first();
    await expect(activeBadge).toBeVisible();
    await expect(activeBadge).toHaveClass(/bg-green-100/);
    await expect(activeBadge).toHaveClass(/text-green-800/);

    // Inactive badge: bg-yellow-100 text-yellow-800
    const inactiveBadge = page
      .locator('span.rounded-full', { hasText: 'Inactive' })
      .first();
    if (await inactiveBadge.count() > 0) {
      await expect(inactiveBadge).toHaveClass(/bg-yellow-100/);
      await expect(inactiveBadge).toHaveClass(/text-yellow-800/);
    }

    // Expired badge: bg-red-100 text-red-800
    const expiredBadge = page
      .locator('span.rounded-full', { hasText: 'Expired' })
      .first();
    if (await expiredBadge.count() > 0) {
      await expect(expiredBadge).toHaveClass(/bg-red-100/);
      await expect(expiredBadge).toHaveClass(/text-red-800/);
    }
  });

  // ── 6. Table has all required column headers ──────────────────────────────
  test('table has expected column headers', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    const headers = page.locator('thead th');
    await expect(headers.filter({ hasText: 'Name' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Code' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Type' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Discount' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Status' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Dates' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Actions' })).toBeVisible();
  });

  // ── 7. "Create Campaign" button navigates to /campaigns/new ──────────────
  test('"Create Campaign" button navigates to /campaigns/new', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    const createBtn = page.locator('a', { hasText: 'Create Campaign' });
    await expect(createBtn).toBeVisible();
    await createBtn.click();

    await page.waitForURL('**/campaigns/new', { timeout: 10000 });
    await expect(page.locator('h1', { hasText: 'Create Campaign' })).toBeVisible();
  });

  // ── 8. Create campaign form renders all required fields ───────────────────
  test('create campaign form has all required fields', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns/new');

    await expect(page.locator('h1', { hasText: 'Create Campaign' })).toBeVisible();

    // Name field
    await expect(page.locator('#name')).toBeVisible();

    // Description textarea
    await expect(page.locator('#description')).toBeVisible();

    // Promo code field
    await expect(page.locator('#code')).toBeVisible();

    // Discount type select
    const typeSelect = page.locator('#type');
    await expect(typeSelect).toBeVisible();
    await expect(typeSelect.locator('option[value="percentage"]')).toHaveCount(1);
    await expect(typeSelect.locator('option[value="fixed_amount"]')).toHaveCount(1);

    // Discount value field
    await expect(page.locator('#discount_value')).toBeVisible();

    // Minimum order amount field
    await expect(page.locator('#min_order_amount')).toBeVisible();

    // Max usage count field
    await expect(page.locator('#max_usage_count')).toBeVisible();

    // Start date and end date fields
    await expect(page.locator('#start_date')).toBeVisible();
    await expect(page.locator('#end_date')).toBeVisible();

    // Status select with active/inactive options
    const statusSelect = page.locator('#status');
    await expect(statusSelect).toBeVisible();
    await expect(statusSelect.locator('option[value="active"]')).toHaveCount(1);
    await expect(statusSelect.locator('option[value="inactive"]')).toHaveCount(1);

    // Save / submit button
    await expect(
      page.locator('button[type="submit"]', { hasText: 'Create Campaign' }),
    ).toBeVisible();

    // "Back to Campaigns" link
    await expect(page.locator('a', { hasText: 'Back to Campaigns' })).toBeVisible();
  });

  // ── 9. Successfully create a new campaign ─────────────────────────────────
  test('creates a new campaign and returns to the list', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns/new');

    await expect(page.locator('h1', { hasText: 'Create Campaign' })).toBeVisible();

    const uid = Date.now().toString(36).toUpperCase();
    const campaignName = `E2E Campaign ${uid}`;
    const campaignCode = `E2E${uid}`;

    // Fill in all required fields
    await page.fill('#name', campaignName);
    await page.fill('#code', campaignCode);
    await page.selectOption('#type', 'percentage');
    await page.fill('#discount_value', '15');
    await page.fill('#start_date', todayIso());
    await page.fill('#end_date', futureDateIso(30));
    await page.selectOption('#status', 'active');

    // Submit the form
    await page.click('button[type="submit"]');

    // On success the page redirects back to the campaigns list
    await page.waitForURL('**/campaigns', { timeout: 15000 });
    await expect(page.locator('h1', { hasText: 'Campaigns' })).toBeVisible();

    // The newly created campaign should appear in the table
    await expect(
      page.locator('td', { hasText: campaignName }),
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('code', { hasText: campaignCode })).toBeVisible();
  });

  // ── 10. Edit link for each campaign navigates to the edit form ────────────
  test('edit link navigates to the campaign edit form', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // Click the first "Edit" link in the actions column
    const firstEditLink = page.locator('a', { hasText: 'Edit' }).first();
    await expect(firstEditLink).toBeVisible();
    await firstEditLink.click();

    // Should land on /campaigns/<uuid>
    await page.waitForURL(/\/campaigns\/[^/]+$/, { timeout: 10000 });
    await expect(page.locator('h1', { hasText: 'Edit Campaign' })).toBeVisible();
  });

  // ── 11. Edit form is pre-filled with existing campaign data ───────────────
  test('edit form pre-fills existing campaign data', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // Navigate to the edit form of the first campaign in the list
    const firstEditLink = page.locator('a', { hasText: 'Edit' }).first();
    await firstEditLink.click();
    await page.waitForURL(/\/campaigns\/[^/]+$/, { timeout: 10000 });

    // The form should have non-empty values in the key fields
    const nameInput = page.locator('#name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    const nameValue = await nameInput.inputValue();
    expect(nameValue.length).toBeGreaterThan(0);

    const codeInput = page.locator('#code');
    const codeValue = await codeInput.inputValue();
    expect(codeValue.length).toBeGreaterThan(0);

    const discountInput = page.locator('#discount_value');
    const discountValue = await discountInput.inputValue();
    expect(Number(discountValue)).toBeGreaterThan(0);

    // Save Changes button should be present (not "Create Campaign")
    await expect(
      page.locator('button[type="submit"]', { hasText: 'Save Changes' }),
    ).toBeVisible();
  });

  // ── 12. "Back to Campaigns" link navigates back to the list ──────────────
  test('"Back to Campaigns" link navigates back to the list', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns/new');

    await expect(page.locator('h1', { hasText: 'Create Campaign' })).toBeVisible();

    const backLink = page.locator('a', { hasText: 'Back to Campaigns' });
    await expect(backLink).toBeVisible();
    await backLink.click();

    await page.waitForURL('**/campaigns', { timeout: 10000 });
    await expect(page.locator('h1', { hasText: 'Campaigns' })).toBeVisible();
  });

  // ── 13. Campaign dates are displayed in a human-readable format ───────────
  test('campaign dates are displayed in a readable format', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // Dates are rendered via formatShortDate → "Jan 15, 2025" style.
    // We verify that at least one cell in the Dates column contains a
    // recognisable month abbreviation followed by a 4-digit year.
    const dateCell = page.locator('tbody td').filter({
      hasText: /[A-Z][a-z]{2}\s+\d{1,2},\s+\d{4}/,
    });
    await expect(dateCell.first()).toBeVisible();
  });

  // ── 14. "Deactivate" button is visible for active campaigns ───────────────
  test('Deactivate button is visible for active campaigns', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns');

    await expect(page.locator('table')).toBeVisible({ timeout: 15000 });

    // At least one active campaign should show a Deactivate button
    const deactivateBtn = page.locator('button', { hasText: 'Deactivate' }).first();
    await expect(deactivateBtn).toBeVisible();
  });

  // ── 15. Form validation prevents saving without required fields ────────────
  test('create form shows validation errors for missing required fields', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/campaigns/new');

    await expect(page.locator('h1', { hasText: 'Create Campaign' })).toBeVisible();

    // Submit without filling anything
    await page.click('button[type="submit"]');

    // Should stay on /campaigns/new — validation blocks navigation
    await expect(page).toHaveURL(/\/campaigns\/new/);

    // At least one field-level error message should appear
    const errorMsg = page.locator('.text-red-600');
    await expect(errorMsg.first()).toBeVisible();
  });

});
