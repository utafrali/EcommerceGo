import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';

// ─── Categories ─────────────────────────────────────────────────────────────

test.describe('Categories page', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/categories');
    // Wait for the page heading to confirm navigation completed
    await expect(page.locator('h1')).toHaveText('Categories', { timeout: 10000 });
  });

  test('1. categories page loads with correct title', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Categories');
  });

  test('2. read-only info banner is displayed', async ({ page }) => {
    const banner = page.locator('.bg-blue-50');
    await expect(banner).toBeVisible();
    await expect(banner).toContainText('Categories are read-only in the CMS');
  });

  test('3. all 5 categories are listed', async ({ page }) => {
    // Wait until the skeleton loader disappears and the table rows appear
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });
  });

  test('4. category names match expected values', async ({ page }) => {
    const expectedNames = [
      'Electronics',
      'Clothing',
      'Home & Kitchen',
      'Sports & Outdoors',
      'Books',
    ];

    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });

    for (const name of expectedNames) {
      await expect(page.locator('tbody').getByText(name, { exact: true })).toBeVisible();
    }
  });

  test('5. slug column shows correct slugs', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });

    // Verify two representative slugs rendered in the font-mono slug cells
    const slugCells = page.locator('tbody td.font-mono');
    const slugTexts = await slugCells.allTextContents();

    expect(slugTexts).toContain('electronics');
    expect(slugTexts.some((s) => s === 'home-kitchen' || s === 'home_kitchen')).toBe(true);
  });

  test('6. no edit or delete buttons exist on the page', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });

    // Broad selectors that would match any typical edit/delete action button
    await expect(page.locator('button:has-text("Edit")')).toHaveCount(0);
    await expect(page.locator('button:has-text("Delete")')).toHaveCount(0);
    await expect(page.locator('a:has-text("Edit")')).toHaveCount(0);
    await expect(page.locator('a:has-text("Delete")')).toHaveCount(0);
    await expect(page.locator('[data-testid="edit-btn"]')).toHaveCount(0);
    await expect(page.locator('[data-testid="delete-btn"]')).toHaveCount(0);
  });
});

// ─── Brands ─────────────────────────────────────────────────────────────────

test.describe('Brands page', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/brands');
    // Wait for the page heading to confirm navigation completed
    await expect(page.locator('h1')).toHaveText('Brands', { timeout: 10000 });
  });

  test('7. brands page loads with correct title', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Brands');
  });

  test('8. read-only info banner is displayed', async ({ page }) => {
    const banner = page.locator('.bg-blue-50');
    await expect(banner).toBeVisible();
    await expect(banner).toContainText('Brands are read-only in the CMS');
  });

  test('9. all 5 brands are listed', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });
  });

  test('10. brand names match expected values', async ({ page }) => {
    const expectedNames = [
      'BookWorld',
      'HomeEssentials',
      'SportPro',
      'StyleCo',
      'TechBrand',
    ];

    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });

    for (const name of expectedNames) {
      await expect(page.locator('tbody').getByText(name, { exact: true })).toBeVisible();
    }
  });

  test('11. letter avatars are displayed with the correct first letter', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });

    // The avatar element is a div > span containing the uppercase first letter.
    // Only rows without a logo_url render the letter avatar.
    // We look for spans inside the avatar container (rounded div with bg-gray-100).
    const avatarSpans = page.locator('tbody td div.rounded.bg-gray-100 span');

    // Collect the visible avatar letters
    const letters = await avatarSpans.allTextContents();
    const visibleLetters = letters.map((l) => l.trim()).filter(Boolean);

    // Each expected brand produces a specific first letter
    const expectedLetters = ['B', 'H', 'S', 'T']; // BookWorld, HomeEssentials/SportPro/StyleCo, TechBrand
    for (const letter of expectedLetters) {
      expect(visibleLetters).toContain(letter);
    }
  });

  test('12. slug column shows correct slugs', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });

    const slugCells = page.locator('tbody td.font-mono');
    const slugTexts = await slugCells.allTextContents();

    // Verify a representative selection of expected slugs
    const expectedSlugs = [
      'bookworld',
      'homeessentials',
      'sportpro',
      'styleco',
      'techbrand',
    ];

    for (const slug of expectedSlugs) {
      expect(slugTexts.map((s) => s.trim())).toContain(slug);
    }
  });

  test('13. no edit or delete buttons exist on the page', async ({ page }) => {
    const rows = page.locator('tbody tr');
    await expect(rows).toHaveCount(5, { timeout: 15000 });

    await expect(page.locator('button:has-text("Edit")')).toHaveCount(0);
    await expect(page.locator('button:has-text("Delete")')).toHaveCount(0);
    await expect(page.locator('a:has-text("Edit")')).toHaveCount(0);
    await expect(page.locator('a:has-text("Delete")')).toHaveCount(0);
    await expect(page.locator('[data-testid="edit-btn"]')).toHaveCount(0);
    await expect(page.locator('[data-testid="delete-btn"]')).toHaveCount(0);
  });
});
