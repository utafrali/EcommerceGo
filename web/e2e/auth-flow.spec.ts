import { test, expect } from '@playwright/test';

test.describe('Auth Flow - Login Page', () => {
  test('login page loads with heading', async ({ page }) => {
    await page.goto('/auth/login');
    await expect(
      page.getByRole('heading', { name: 'Sign in to your account' }),
    ).toBeVisible();
  });

  test('login page has create account link', async ({ page }) => {
    await page.goto('/auth/login');
    await expect(page.getByText('create a new account')).toBeVisible();
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
    const passwordInput = page.getByLabel('Password');
    await expect(passwordInput).toBeVisible();
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await expect(passwordInput).toHaveAttribute(
      'placeholder',
      'Enter your password',
    );
  });

  test('login form has submit button', async ({ page }) => {
    await page.goto('/auth/login');
    const submitButton = page.getByRole('button', { name: 'Sign in' });
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toHaveAttribute('type', 'submit');
  });

  test('email and password fields accept input', async ({ page }) => {
    await page.goto('/auth/login');
    const emailInput = page.getByLabel('Email address');
    const passwordInput = page.getByLabel('Password');

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
    const passwordInput = page.getByLabel('Password');
    await expect(passwordInput).toHaveAttribute('required', '');
  });

  // TODO: Uncomment when login form submission is wired to BFF
  // test('submitting login form with valid credentials redirects to home', async ({ page }) => {
  //   await page.goto('/auth/login');
  //   await page.getByLabel('Email address').fill('test@example.com');
  //   await page.getByLabel('Password').fill('password123');
  //   await page.getByRole('button', { name: 'Sign in' }).click();
  //   await expect(page).toHaveURL('/');
  // });

  // TODO: Uncomment when error handling is implemented
  // test('submitting login form with invalid credentials shows error', async ({ page }) => {
  //   await page.goto('/auth/login');
  //   await page.getByLabel('Email address').fill('wrong@example.com');
  //   await page.getByLabel('Password').fill('wrongpassword');
  //   await page.getByRole('button', { name: 'Sign in' }).click();
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

  test('cart page shows coming soon message', async ({ page }) => {
    await page.goto('/cart');
    await expect(page.getByText('coming soon')).toBeVisible();
  });

  test('cart page mentions BFF API endpoint', async ({ page }) => {
    await page.goto('/cart');
    await expect(page.getByText('/api/cart')).toBeVisible();
  });

  test('cart page displays placeholder skeleton items', async ({ page }) => {
    await page.goto('/cart');
    // The scaffold renders 3 placeholder skeleton cart items
    const skeletons = page.locator('.animate-pulse');
    await expect(skeletons).toHaveCount(3);
  });

  // TODO: Uncomment when cart requires authentication
  // test('unauthenticated user visiting cart is redirected to login', async ({ page }) => {
  //   await page.goto('/cart');
  //   await expect(page).toHaveURL('/auth/login?redirect=/cart');
  // });
});
