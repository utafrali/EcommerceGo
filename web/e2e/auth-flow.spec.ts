import { test, expect } from '@playwright/test';

test.describe('Auth Flow - Login Page', () => {
  test('login page loads with heading', async ({ page }) => {
    await page.goto('/auth/login');
    await expect(
      page.getByRole('heading', { name: 'Sign in to EcommerceGo' }),
    ).toBeVisible();
  });

  test('login page has register link', async ({ page }) => {
    await page.goto('/auth/login');
    await expect(page.getByText("Don't have an account?")).toBeVisible();
    await expect(page.getByRole('main').getByRole('link', { name: 'Register' })).toBeVisible();
  });

  test('login form has email input', async ({ page }) => {
    await page.goto('/auth/login');
    const emailInput = page.getByLabel('Email address');
    await expect(emailInput).toBeVisible();
    await expect(emailInput).toHaveAttribute('type', 'email');
    await expect(emailInput).toHaveAttribute(
      'placeholder',
      'you@example.com',
    );
  });

  test('login form has password input', async ({ page }) => {
    await page.goto('/auth/login');
    const passwordInput = page.locator('input[name="password"]');
    await expect(passwordInput).toBeVisible();
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await expect(passwordInput).toHaveAttribute(
      'placeholder',
      'Enter your password',
    );
  });

  test('login form has submit button', async ({ page }) => {
    await page.goto('/auth/login');
    const submitButton = page.getByRole('button', { name: 'Sign In' });
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toHaveAttribute('type', 'submit');
  });

  test('email and password fields accept input', async ({ page }) => {
    await page.goto('/auth/login');
    const emailInput = page.getByLabel('Email address');
    const passwordInput = page.locator('input[name="password"]');

    await emailInput.fill('test@example.com');
    await passwordInput.fill('password123');

    await expect(emailInput).toHaveValue('test@example.com');
    await expect(passwordInput).toHaveValue('password123');
  });

  test('email field has required attribute', async ({ page }) => {
    await page.goto('/auth/login');
    const emailInput = page.getByLabel('Email address');
    await expect(emailInput).toHaveAttribute('required', '');
  });

  test('password field has required attribute', async ({ page }) => {
    await page.goto('/auth/login');
    const passwordInput = page.locator('input[name="password"]');
    await expect(passwordInput).toHaveAttribute('required', '');
  });

  // TODO: Uncomment when login form submission is wired to BFF
  // test('submitting login form with valid credentials redirects to home', async ({ page }) => {
  //   await page.goto('/auth/login');
  //   await page.getByLabel('Email address').fill('test@example.com');
  //   await page.locator('input[name="password"]').fill('password123');
  //   await page.getByRole('button', { name: 'Sign In' }).click();
  //   await expect(page).toHaveURL('/');
  // });

  // TODO: Uncomment when error handling is implemented
  // test('submitting login form with invalid credentials shows error', async ({ page }) => {
  //   await page.goto('/auth/login');
  //   await page.getByLabel('Email address').fill('wrong@example.com');
  //   await page.locator('input[name="password"]').fill('wrongpassword');
  //   await page.getByRole('button', { name: 'Sign In' }).click();
  //   await expect(page.getByText('Invalid credentials')).toBeVisible();
  // });
});

test.describe('Auth Flow - Cart Page (requires auth)', () => {
  test('cart page loads with heading', async ({ page }) => {
    await page.goto('/cart');
    await expect(
      page.getByRole('heading', { name: 'Shopping Cart' }),
    ).toBeVisible();
  });

  test('cart page shows empty cart message', async ({ page }) => {
    await page.goto('/cart');
    await expect(page.getByText('Your cart is empty')).toBeVisible();
  });

  test('cart page shows explore products link', async ({ page }) => {
    await page.goto('/cart');
    const exploreLink = page.getByRole('link', { name: 'Explore Products' });
    await expect(exploreLink).toBeVisible();
    await expect(exploreLink).toHaveAttribute('href', '/products');
  });

  test('cart page displays empty state with helpful message', async ({
    page,
  }) => {
    await page.goto('/cart');
    await expect(page.getByText('Your cart is empty')).toBeVisible();
    await expect(
      page.getByText(
        'Discover amazing products and start shopping today. Your perfect find is just a click away!',
      ),
    ).toBeVisible();
  });

  // TODO: Uncomment when cart requires authentication
  // test('unauthenticated user visiting cart is redirected to login', async ({ page }) => {
  //   await page.goto('/cart');
  //   await expect(page).toHaveURL('/auth/login?redirect=/cart');
  // });
});
