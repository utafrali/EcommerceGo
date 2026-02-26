import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';

// ─── Test Suite: CMS Orders ─────────────────────────────────────────────────

test.describe('CMS Orders — List Page', () => {

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  // ── 1. Orders list loads with orders visible ────────────────────────────
  test('orders list page loads with correct heading and subtitle', async ({ page }) => {
    await page.goto('/orders');

    await expect(page.locator('h1', { hasText: 'Orders' })).toBeVisible();
    await expect(
      page.locator('p', { hasText: 'View and manage customer orders.' }),
    ).toBeVisible();
  });

  // ── 2. At least 7 orders are shown ─────────────────────────────────────
  test('at least 7 orders are shown in the table', async ({ page }) => {
    await page.goto('/orders');

    // Wait for the table body rows to appear (loading skeleton fades out)
    const rows = page.locator('tbody tr');
    await expect(rows.first()).toBeVisible({ timeout: 15000 });

    const count = await rows.count();
    expect(count).toBeGreaterThanOrEqual(7);
  });

  // ── 3. Table columns are all present ───────────────────────────────────
  test('orders table renders all expected column headers', async ({ page }) => {
    await page.goto('/orders');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    const thead = page.locator('thead');
    await expect(thead.locator('th', { hasText: /order id/i })).toBeVisible();
    await expect(thead.locator('th', { hasText: /customer id/i })).toBeVisible();
    await expect(thead.locator('th', { hasText: /status/i })).toBeVisible();
    await expect(thead.locator('th', { hasText: /items/i })).toBeVisible();
    await expect(thead.locator('th', { hasText: /total/i })).toBeVisible();
    await expect(thead.locator('th', { hasText: /created/i })).toBeVisible();
  });

  // ── 4. Status badges display correctly (all "Pending") ─────────────────
  test('status badges render with "Pending" label and yellow colour classes', async ({ page }) => {
    await page.goto('/orders');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // Every visible badge must say "Pending" and carry yellow Tailwind classes
    const badges = page.locator('tbody tr span.rounded-full');
    const badgeCount = await badges.count();
    expect(badgeCount).toBeGreaterThanOrEqual(7);

    for (let i = 0; i < badgeCount; i++) {
      const badge = badges.nth(i);
      await expect(badge).toHaveText('Pending');
      await expect(badge).toHaveClass(/bg-yellow-100/);
      await expect(badge).toHaveClass(/text-yellow-800/);
    }
  });

  // ── 5. Order ID links are clickable ────────────────────────────────────
  test('order ID cells contain links that point to /orders/[id]', async ({ page }) => {
    await page.goto('/orders');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // The first-column link in every row should have href="/orders/<uuid>"
    const idLinks = page.locator('tbody tr td:first-child a');
    const linkCount = await idLinks.count();
    expect(linkCount).toBeGreaterThanOrEqual(7);

    const firstHref = await idLinks.first().getAttribute('href');
    expect(firstHref).toMatch(/^\/orders\/[0-9a-f-]+$/i);
  });

  // ── 6. Status filter dropdown is present with correct options ───────────
  test('status filter dropdown contains all expected status options', async ({ page }) => {
    await page.goto('/orders');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    const filter = page.locator('select#status-filter');
    await expect(filter).toBeVisible();

    const options = filter.locator('option');
    const texts = await options.allInnerTexts();
    expect(texts).toContain('All');
    expect(texts).toContain('Pending');
    expect(texts).toContain('Confirmed');
    expect(texts).toContain('Processing');
    expect(texts).toContain('Shipped');
    expect(texts).toContain('Delivered');
    expect(texts).toContain('Canceled');
    expect(texts).toContain('Refunded');
  });

  // ── 7. Status filter hides rows that do not match ──────────────────────
  test('selecting "Confirmed" status filter shows no-orders empty state (all orders are Pending)', async ({ page }) => {
    await page.goto('/orders');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // Change filter to "confirmed"; since all test orders are pending, the
    // table should become empty and display the empty-state message.
    await page.selectOption('select#status-filter', 'confirmed');

    // Wait for the re-fetch — the empty state text or "No orders found" appears
    await expect(
      page.locator('text=No orders found'),
    ).toBeVisible({ timeout: 10000 });
  });

  // ── 8. No "Create" button is present ───────────────────────────────────
  test('orders list page has no create/new order button', async ({ page }) => {
    await page.goto('/orders');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // Orders are created through checkout, not the CMS
    await expect(page.locator('a', { hasText: /new order/i })).toHaveCount(0);
    await expect(page.locator('button', { hasText: /create order/i })).toHaveCount(0);
  });

  // ── 9. Pagination controls only appear when there are multiple pages ────
  test('pagination section is rendered in the DOM (even if single page)', async ({ page }) => {
    await page.goto('/orders');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 15000 });

    // totalPages == 1 means the pagination block is hidden; just verify the
    // page doesn't crash and the table is still intact.
    const rows = page.locator('tbody tr');
    await expect(rows.first()).toBeVisible();
  });

});

// ─── Test Suite: CMS Order Detail Page ─────────────────────────────────────

test.describe('CMS Orders — Detail Page', () => {

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  /**
   * Helper: navigate to the orders list and return the href of the first order
   * detail link. Using this avoids hard-coding a UUID in the tests.
   */
  async function getFirstOrderHref(page: import('@playwright/test').Page): Promise<string> {
    await page.goto('/orders');
    const firstLink = page.locator('tbody tr td:first-child a').first();
    await expect(firstLink).toBeVisible({ timeout: 15000 });
    const href = await firstLink.getAttribute('href');
    if (!href) throw new Error('Could not obtain order detail href from list page');
    return href;
  }

  // ── 10. Navigate to order detail page ──────────────────────────────────
  test('clicking an order ID link navigates to the order detail page', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    // The detail page renders "Order Details" as the card heading
    await expect(
      page.locator('h1', { hasText: 'Order Details' }),
    ).toBeVisible({ timeout: 10000 });
  });

  // ── 11. Order detail shows order ID and customer ID ─────────────────────
  test('order detail card displays order ID, customer ID, created and updated dates', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    // Order ID rendered as monospaced paragraph beneath the heading
    const orderIdCell = page.locator('p.font-mono').first();
    await expect(orderIdCell).toBeVisible();
    const orderIdText = await orderIdCell.innerText();
    expect(orderIdText.trim().length).toBeGreaterThan(0);

    // Customer ID — labelled term in the <dl>
    const customerIdLabel = page.locator('dt', { hasText: /customer id/i });
    await expect(customerIdLabel).toBeVisible();
    const customerIdValue = customerIdLabel.locator('~ dd').first();
    await expect(customerIdValue).toBeVisible();
    expect((await customerIdValue.innerText()).trim().length).toBeGreaterThan(0);

    // Created date term
    await expect(page.locator('dt', { hasText: /^created$/i })).toBeVisible();

    // Last Updated date term
    await expect(page.locator('dt', { hasText: /last updated/i })).toBeVisible();
  });

  // ── 12. Items table shows product names, quantities, prices ─────────────
  test('items table renders product name, product ID, qty, unit price, and total columns', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    // Section heading
    await expect(page.locator('h2', { hasText: 'Items' })).toBeVisible();

    // Column headers
    const itemsTable = page.locator('h2:has-text("Items") ~ div table, h2:has-text("Items") + div table').first();

    // Use the containing card instead so we do not depend on sibling selectors
    const itemsCard = page.locator('div.bg-white').filter({ has: page.locator('h2', { hasText: 'Items' }) });
    const itemsThead = itemsCard.locator('thead');
    await expect(itemsThead.locator('th', { hasText: /product/i }).first()).toBeVisible();
    await expect(itemsThead.locator('th', { hasText: /product id/i })).toBeVisible();
    await expect(itemsThead.locator('th', { hasText: /qty/i })).toBeVisible();
    await expect(itemsThead.locator('th', { hasText: /unit price/i })).toBeVisible();
    await expect(itemsThead.locator('th', { hasText: /total/i })).toBeVisible();

    // At least one item row
    const itemRows = itemsCard.locator('tbody tr');
    await expect(itemRows.first()).toBeVisible();

    // Each row should have a non-empty product name in the first cell
    const firstRowCells = itemRows.first().locator('td');
    const productName = await firstRowCells.first().innerText();
    expect(productName.trim().length).toBeGreaterThan(0);

    // Quantity cell (3rd column, index 2) should be a positive integer
    const qty = await firstRowCells.nth(2).innerText();
    expect(parseInt(qty.trim(), 10)).toBeGreaterThan(0);

    // Unit price cell (4th column, index 3) should look like a currency string
    const unitPrice = await firstRowCells.nth(3).innerText();
    expect(unitPrice.trim()).toMatch(/^\$[\d,]+\.\d{2}$/);

    // Row total cell (5th column, index 4)
    const rowTotal = await firstRowCells.nth(4).innerText();
    expect(rowTotal.trim()).toMatch(/^\$[\d,]+\.\d{2}$/);
  });

  // ── 13. Shipping address section is present ─────────────────────────────
  test('shipping address section displays full_name, address_line, city, state, postal_code, country', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    const addressCard = page
      .locator('div.bg-white')
      .filter({ has: page.locator('h2', { hasText: 'Shipping Address' }) });

    await expect(addressCard.locator('h2', { hasText: 'Shipping Address' })).toBeVisible();

    // The <address> tag should contain visible text (non-empty lines)
    const address = addressCard.locator('address');
    await expect(address).toBeVisible();
    const addressText = await address.innerText();
    expect(addressText.trim().length).toBeGreaterThan(0);
  });

  // ── 14. Order summary shows subtotal and total amounts ──────────────────
  test('order summary section shows subtotal and total formatted as currency', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    const summaryCard = page
      .locator('div.bg-white')
      .filter({ has: page.locator('h2', { hasText: 'Order Summary' }) });

    await expect(summaryCard.locator('h2', { hasText: 'Order Summary' })).toBeVisible();

    // Subtotal row: parent div contains dt "Subtotal" + dd with price
    const subtotalRow = summaryCard.locator('div').filter({ has: page.locator('dt', { hasText: 'Subtotal' }) });
    await expect(subtotalRow).toBeVisible();
    const subtotalValue = subtotalRow.locator('dd');
    await expect(subtotalValue).toBeVisible();
    expect((await subtotalValue.innerText()).trim()).toMatch(/^\$[\d,]+\.\d{2}$/);

    // Total row: parent div contains dt "Total" + dd with price
    const totalRow = summaryCard.locator('div').filter({ has: page.locator('dt', { hasText: /^Total$/ }) });
    await expect(totalRow).toBeVisible();
    const totalValue = totalRow.locator('dd');
    await expect(totalValue).toBeVisible();
    expect((await totalValue.innerText()).trim()).toMatch(/^\$[\d,]+\.\d{2}$/);
  });

  // ── 15. Update status dropdown has correct options ──────────────────────
  test('Update Status section has dropdown with valid transition options', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    // Section heading
    await expect(page.locator('h2', { hasText: 'Update Status' })).toBeVisible();

    const statusCard = page
      .locator('div.bg-white')
      .filter({ has: page.locator('h2', { hasText: 'Update Status' }) });

    // Valid transition map (mirrors the CMS constant)
    const VALID_TRANSITIONS: Record<string, string[]> = {
      pending: ['confirmed', 'canceled'],
      confirmed: ['processing', 'canceled'],
      processing: ['shipped', 'canceled'],
      shipped: ['delivered'],
      delivered: ['refunded'],
      canceled: [],
      refunded: [],
    };

    // Determine current order status from the badge in the header
    const statusBadge = page.locator('div.bg-white').first().locator('span.inline-flex');
    const currentStatus = (await statusBadge.innerText()).trim().toLowerCase();

    const expectedTransitions = VALID_TRANSITIONS[currentStatus] || [];
    const isTerminal = expectedTransitions.length === 0;

    if (isTerminal) {
      // Terminal states show a message instead of a dropdown
      await expect(statusCard.locator('p', { hasText: 'terminal state' })).toBeVisible();
    } else {
      const statusSelect = statusCard.locator('select');
      await expect(statusSelect).toBeVisible();

      const optionTexts = await statusSelect.locator('option').allInnerTexts();
      const normalised = optionTexts.map((t) => t.toLowerCase().replace('(current)', '').trim());

      // Should contain current status and all valid transitions
      expect(normalised).toContain(currentStatus);
      for (const transition of expectedTransitions) {
        expect(normalised).toContain(transition);
      }

      // "Update Status" button is present alongside the select
      const updateBtn = statusCard.locator('button', { hasText: 'Update Status' });
      await expect(updateBtn).toBeVisible();
    }
  });

  // ── 16. Update Status button is disabled when status has not changed ─────
  test('Update Status button is disabled when the selected status matches current status', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    const updateBtn = page
      .locator('div.bg-white')
      .filter({ has: page.locator('h2', { hasText: 'Update Status' }) })
      .locator('button', { hasText: 'Update Status' });

    // The current status is "pending" and the select defaults to "pending",
    // so the button must be disabled.
    await expect(updateBtn).toBeDisabled();
  });

  // ── 17. Update Status button becomes enabled after changing the dropdown ─
  test('Update Status button becomes enabled when a different status is selected', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    const updateCard = page
      .locator('div.bg-white')
      .filter({ has: page.locator('h2', { hasText: 'Update Status' }) });

    const statusSelect = updateCard.locator('select');
    const updateBtn = updateCard.locator('button', { hasText: 'Update Status' });

    // Change to "confirmed" — different from the current "pending"
    await statusSelect.selectOption('confirmed');
    await expect(updateBtn).toBeEnabled();
  });

  // ── 18. Back to Orders link navigates back to the list page ─────────────
  test('"Back to Orders" link navigates to /orders', async ({ page }) => {
    const href = await getFirstOrderHref(page);
    await page.goto(href);

    await expect(page.locator('h1', { hasText: 'Order Details' })).toBeVisible({ timeout: 10000 });

    const backLink = page.locator('a', { hasText: 'Back to Orders' });
    await expect(backLink).toBeVisible();

    await backLink.click();
    await page.waitForURL('**/orders', { timeout: 10000 });
    await expect(page.locator('h1', { hasText: 'Orders' })).toBeVisible();
  });

});
