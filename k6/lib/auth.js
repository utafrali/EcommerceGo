/**
 * k6 authentication helpers for EcommerceGo.
 *
 * Handles JWT token acquisition and caching across VUs.
 * Uses the user service /api/v1/auth/login endpoint.
 */

import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';
import { url, jsonHeaders } from './helpers.js';

// ── Predefined test accounts ──────────────────────────────────────────────────
// These accounts should exist in the test environment.
// Create them with: make seed-test-users (or via the user service directly).

const TEST_CREDENTIALS = new SharedArray('credentials', function () {
  return [
    { email: 'loadtest1@ecommercego.local', password: 'LoadTest1234!' },
    { email: 'loadtest2@ecommercego.local', password: 'LoadTest1234!' },
    { email: 'loadtest3@ecommercego.local', password: 'LoadTest1234!' },
    { email: 'loadtest4@ecommercego.local', password: 'LoadTest1234!' },
    { email: 'loadtest5@ecommercego.local', password: 'LoadTest1234!' },
  ];
});

/**
 * Login with a test credential and return the JWT token.
 * Returns null if login fails (test should handle gracefully).
 */
export function login(credentialIndex) {
  const cred = TEST_CREDENTIALS[credentialIndex % TEST_CREDENTIALS.length];

  const res = http.post(
    url('/api/v1/auth/login'),
    JSON.stringify({ email: cred.email, password: cred.password }),
    { headers: jsonHeaders(), tags: { name: 'auth:login' } },
  );

  const ok = check(res, {
    'auth:login status 200': (r) => r.status === 200,
    'auth:login has token':  (r) => {
      try {
        return JSON.parse(r.body).token !== undefined;
      } catch (_) {
        return false;
      }
    },
  });

  if (!ok) {
    return null;
  }

  try {
    return JSON.parse(res.body).token;
  } catch (_) {
    return null;
  }
}

/**
 * Register a new test user and return the JWT token.
 * Useful for isolation tests where each VU needs a unique account.
 */
export function registerAndLogin(email, password) {
  // Register
  const regRes = http.post(
    url('/api/v1/auth/register'),
    JSON.stringify({
      email,
      password,
      first_name: 'Load',
      last_name:  'Test',
    }),
    { headers: jsonHeaders(), tags: { name: 'auth:register' } },
  );

  if (regRes.status !== 201) {
    return null;
  }

  // Login to get token
  const loginRes = http.post(
    url('/api/v1/auth/login'),
    JSON.stringify({ email, password }),
    { headers: jsonHeaders(), tags: { name: 'auth:login' } },
  );

  if (loginRes.status !== 200) {
    return null;
  }

  try {
    return JSON.parse(loginRes.body).token;
  } catch (_) {
    return null;
  }
}

/**
 * Get a static test token from env (for simple scenarios where auth isn't the focus).
 * Set K6_TEST_TOKEN env var when running k6.
 */
export function staticToken() {
  return __ENV.K6_TEST_TOKEN || null;
}
