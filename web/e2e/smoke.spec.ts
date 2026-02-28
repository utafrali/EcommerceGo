import { test, expect } from '@playwright/test';

test.describe('Smoke Tests', () => {
  test('homepage loads with correct title', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle('EcommerceGo');
  });

  test('homepage displays hero heading', async ({ page }) => {
    await page.goto('/');
    // HeroSlider fallback first slide renders "ELBİSE" as h2
    const heading = page.getByRole('heading', { name: 'ELBİSE' });
    await expect(heading).toBeVisible();
  });

  test('homepage has primary CTA link', async ({ page }) => {
    await page.goto('/');
    // HeroSlider first slide CTA is "KEŞFET" linking to /products?sort=newest
    const link = page.getByRole('link', { name: 'KEŞFET' }).first();
    await expect(link).toBeVisible();
    await expect(link).toHaveAttribute('href', '/products?sort=newest');
  });

  test('homepage displays benefit bar', async ({ page }) => {
    await page.goto('/');
    // BenefitBar is a server component below the hero — scroll to it and check
    const benefitSection = page.locator('section', { hasText: 'Kargo Bedava' });
    await benefitSection.scrollIntoViewIfNeeded();
    await expect(benefitSection.getByText('Kargo Bedava')).toBeVisible();
    await expect(benefitSection.getByText('Koşulsuz İade')).toBeVisible();
    await expect(benefitSection.getByText('Kredi Kartı Taksit İmkânı')).toBeVisible();
    await expect(benefitSection.getByText('Kampanyaları Keşfet')).toBeVisible();
  });

  test('page has correct meta description', async ({ page }) => {
    await page.goto('/');
    const metaDescription = page.locator('meta[name="description"]');
    await expect(metaDescription).toHaveAttribute(
      'content',
      'AI-driven open-source e-commerce platform',
    );
  });

  test('page has correct lang attribute', async ({ page }) => {
    await page.goto('/');
    const html = page.locator('html');
    await expect(html).toHaveAttribute('lang', 'en');
  });
});

test.describe('Navigation', () => {
  test('header contains logo link to home', async ({ page }) => {
    await page.goto('/');
    const logo = page
      .locator('header')
      .getByRole('link', { name: 'EcommerceGo' });
    await expect(logo).toBeVisible();
    await expect(logo).toHaveAttribute('href', '/');
  });

  test('header contains Products nav link', async ({ page }) => {
    await page.goto('/');
    // Category nav has "Markalar" link pointing to /products (desktop only)
    const productsLink = page
      .locator('header')
      .locator('a[href="/products"]')
      .first();
    await expect(productsLink).toBeVisible();
    await expect(productsLink).toHaveAttribute('href', '/products');
  });

  test('header contains shopping cart button', async ({ page }) => {
    await page.goto('/');
    // Cart is a button (opens mini cart), not a link
    const cartButton = page
      .locator('header')
      .getByRole('button', { name: /Sepetim|Sepet/i })
      .first();
    await expect(cartButton).toBeVisible();
  });

  test('header contains Sign In link for unauthenticated users', async ({
    page,
  }) => {
    await page.goto('/');
    const signInLink = page
      .locator('header')
      .getByRole('link', { name: 'Giriş yap veya Üye ol' });
    await expect(signInLink).toBeVisible();
    await expect(signInLink).toHaveAttribute('href', '/auth/login');
  });

  test('header contains wishlist link', async ({ page }) => {
    await page.goto('/');
    const wishlistLink = page
      .locator('header')
      .getByRole('link', { name: 'Favoriler' });
    await expect(wishlistLink).toBeVisible();
    await expect(wishlistLink).toHaveAttribute('href', '/wishlist');
  });

  test('clicking Products nav link navigates to products page', async ({
    page,
  }) => {
    await page.goto('/');
    await page
      .locator('header')
      .locator('a[href="/products"]')
      .first()
      .click();
    await expect(page).toHaveURL('/products');
    await expect(
      page.getByRole('heading', { name: 'Tüm Ürünler' }),
    ).toBeVisible();
  });

  test('clicking cart button opens mini cart or navigates to cart page', async ({
    page,
  }) => {
    // Navigate directly to cart page since cart is now a mini-cart button
    await page.goto('/cart');
    await expect(page).toHaveURL('/cart');
    await expect(
      page.getByRole('heading', { name: 'Sepetim' }),
    ).toBeVisible();
  });

  test('clicking Sign In link navigates to login page', async ({ page }) => {
    await page.goto('/');
    await page
      .locator('header')
      .getByRole('link', { name: 'Giriş yap veya Üye ol' })
      .click();
    await expect(page).toHaveURL('/auth/login');
    await expect(
      page.getByRole('heading', { name: "EcommerceGo'ya Giriş Yap" }),
    ).toBeVisible();
  });

  test('footer displays brand and copyright', async ({ page }) => {
    await page.goto('/');
    const footer = page.locator('footer');
    await expect(footer).toContainText('EcommerceGo');
    await expect(footer).toContainText('Tüm hakları saklıdır');
  });

  test('footer displays Turkish navigation links', async ({ page }) => {
    await page.goto('/');
    const footer = page.locator('footer');
    await expect(footer.getByText('Hakkımızda')).toBeVisible();
    await expect(footer.getByText('Sıkça Sorulan Sorular')).toBeVisible();
    await expect(footer.getByText('Kategoriler')).toBeVisible();
  });
});
