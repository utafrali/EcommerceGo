/**
 * Playwright global setup — ensures the admin user exists with the expected
 * credentials before any CMS E2E test runs.
 *
 * The setup tries to register the user via the Gateway's public auth endpoint.
 * If the user already exists it attempts a login to verify the password is
 * correct. When login fails (password mismatch from a previous run / manual
 * override) the setup logs a warning but doesn't abort — the individual tests
 * will fail with clear login errors instead.
 */

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';
const ADMIN_EMAIL = 'admin@ecommerce.com';
const ADMIN_PASSWORD = 'AdminPass123';

export default async function globalSetup() {
  // Step 1 — try to register the admin user
  const registerRes = await fetch(`${GATEWAY_URL}/api/v1/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: ADMIN_EMAIL,
      password: ADMIN_PASSWORD,
      first_name: 'Admin',
      last_name: 'User',
    }),
  });

  if (registerRes.ok) {
    console.log('[global-setup] Registered admin user');
  } else {
    const body = await registerRes.json().catch(() => ({}));
    const code = (body as Record<string, Record<string, string>>)?.error?.code;
    if (code === 'ALREADY_EXISTS') {
      console.log('[global-setup] Admin user already exists — verifying login');
    } else {
      console.warn('[global-setup] Register failed:', JSON.stringify(body));
    }
  }

  // Step 2 — verify login works
  const loginRes = await fetch(`${GATEWAY_URL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }),
  });

  if (loginRes.ok) {
    console.log('[global-setup] Admin login verified ✓');
  } else {
    const body = await loginRes.json().catch(() => ({}));
    console.error(
      '[global-setup] ⚠️  Admin login FAILED — CMS tests will fail.',
      JSON.stringify(body),
    );
    console.error(
      '[global-setup] Fix: update the admin password in user_db or delete the user and re-run.',
    );
  }
}
