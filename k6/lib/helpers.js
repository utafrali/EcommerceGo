/**
 * k6 shared helpers for EcommerceGo load tests.
 */

import { check } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ── Custom metrics ────────────────────────────────────────────────────────────

export const errorRate    = new Rate('errors');
export const successRate  = new Rate('successes');

// Per-service error counters (imported individually in test files as needed)
export function makeServiceMetrics(name) {
  return {
    duration: new Trend(`${name}_req_duration`, true),
    errors:   new Rate(`${name}_error_rate`),
    reqs:     new Counter(`${name}_requests_total`),
  };
}

// ── Standard thresholds ───────────────────────────────────────────────────────

/** Default thresholds for smoke tests. */
export const SMOKE_THRESHOLDS = {
  http_req_failed:   [{ threshold: 'rate<0.01',  abortOnFail: false }],
  http_req_duration: [{ threshold: 'p(95)<500',  abortOnFail: false }],
  errors:            [{ threshold: 'rate<0.01',  abortOnFail: false }],
};

/** Default thresholds for load tests. */
export const LOAD_THRESHOLDS = {
  http_req_failed:   [{ threshold: 'rate<0.05',   abortOnFail: false }],
  http_req_duration: [
    { threshold: 'p(95)<1000',  abortOnFail: false },
    { threshold: 'p(99)<3000',  abortOnFail: false },
  ],
  errors:            [{ threshold: 'rate<0.05',   abortOnFail: false }],
};

/** Default thresholds for spike tests (looser). */
export const SPIKE_THRESHOLDS = {
  http_req_failed:   [{ threshold: 'rate<0.10',   abortOnFail: false }],
  http_req_duration: [{ threshold: 'p(95)<2000',  abortOnFail: false }],
  errors:            [{ threshold: 'rate<0.10',   abortOnFail: false }],
};

// ── Standard checks ───────────────────────────────────────────────────────────

/**
 * Assert a 2xx response and record error/success metrics.
 * Returns true if check passed.
 */
export function checkOK(res, label) {
  const ok = check(res, {
    [`${label}: status 2xx`]: (r) => r.status >= 200 && r.status < 300,
    [`${label}: has body`]:   (r) => r.body && r.body.length > 0,
  });
  errorRate.add(!ok);
  successRate.add(ok);
  return ok;
}

/**
 * Assert a specific status code.
 */
export function checkStatus(res, expectedStatus, label) {
  const ok = check(res, {
    [`${label}: status ${expectedStatus}`]: (r) => r.status === expectedStatus,
  });
  errorRate.add(!ok);
  successRate.add(ok);
  return ok;
}

/**
 * Assert JSON response with a required field.
 */
export function checkJSON(res, field, label) {
  let body;
  try {
    body = JSON.parse(res.body);
  } catch (_) {
    errorRate.add(1);
    return false;
  }
  const ok = check(res, {
    [`${label}: status 2xx`]:         (r) => r.status >= 200 && r.status < 300,
    [`${label}: has field '${field}'`]: () => body[field] !== undefined,
  });
  errorRate.add(!ok);
  successRate.add(ok);
  return ok;
}

// ── Env helpers ───────────────────────────────────────────────────────────────

/** Read BASE_URL env with fallback. */
export function baseURL() {
  return __ENV.BASE_URL || 'http://localhost:8080';
}

/** Build full URL from path. */
export function url(path) {
  return `${baseURL()}${path}`;
}

/** Standard JSON POST headers. */
export function jsonHeaders(token) {
  const h = { 'Content-Type': 'application/json' };
  if (token) h['Authorization'] = `Bearer ${token}`;
  return h;
}

/** Standard auth headers. */
export function authHeaders(token) {
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// ── Random data ───────────────────────────────────────────────────────────────

/** Generate a random email for test users. */
export function randomEmail() {
  return `k6test_${Date.now()}_${Math.floor(Math.random() * 100000)}@loadtest.local`;
}

/** Random integer between min and max (inclusive). */
export function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

/** Pick a random element from an array. */
export function randomChoice(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

/** Pause for a random duration between min and max seconds. */
export function randomSleep(minSec, maxSec) {
  const { sleep } = require('k6');
  sleep(minSec + Math.random() * (maxSec - minSec));
}
