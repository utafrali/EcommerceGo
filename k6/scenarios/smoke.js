/**
 * Smoke Test — EcommerceGo
 *
 * Quick sanity check: 1 VU, 2 minutes.
 * Validates all services are reachable and responding correctly.
 * Run: k6 run k6/scenarios/smoke.js
 */

import http from 'k6/http';
import { sleep } from 'k6';
import { group } from 'k6';
import { checkOK, checkStatus, SMOKE_THRESHOLDS, url, authHeaders } from '../lib/helpers.js';
import { staticToken, login } from '../lib/auth.js';

export const options = {
  vus:        1,
  duration:   '2m',
  thresholds: SMOKE_THRESHOLDS,
  tags:       { scenario: 'smoke' },
};

export function setup() {
  // Try to get auth token
  const token = staticToken() || login(0);
  return { token };
}

export default function (data) {
  const token = data.token;

  // ── Gateway health ─────────────────────────────────────────────────────────
  group('gateway', () => {
    const live  = http.get(url('/health/live'),  { tags: { name: 'gateway:liveness' } });
    const ready = http.get(url('/health/ready'), { tags: { name: 'gateway:readiness' } });
    checkOK(live,  'gateway:liveness');
    checkOK(ready, 'gateway:readiness');
  });

  sleep(0.5);

  // ── Product service ────────────────────────────────────────────────────────
  group('product', () => {
    const res = http.get(url('/api/v1/products'), {
      headers: authHeaders(token),
      tags:    { name: 'product:list' },
    });
    checkOK(res, 'product:list');
  });

  sleep(0.5);

  // ── Search service ─────────────────────────────────────────────────────────
  group('search', () => {
    const res = http.get(url('/api/v1/search?q=shirt'), {
      headers: authHeaders(token),
      tags:    { name: 'search:query' },
    });
    checkOK(res, 'search:query');
  });

  sleep(0.5);

  // ── Cart service ───────────────────────────────────────────────────────────
  group('cart', () => {
    const res = http.get(url('/api/v1/cart'), {
      headers: authHeaders(token),
      tags:    { name: 'cart:get' },
    });
    // 401 is acceptable if no token, 200 with token
    if (token) {
      checkOK(res, 'cart:get');
    } else {
      checkStatus(res, 401, 'cart:get:no-auth');
    }
  });

  sleep(0.5);

  // ── Order service ──────────────────────────────────────────────────────────
  group('order', () => {
    const res = http.get(url('/api/v1/orders'), {
      headers: authHeaders(token),
      tags:    { name: 'order:list' },
    });
    if (token) {
      checkOK(res, 'order:list');
    } else {
      checkStatus(res, 401, 'order:list:no-auth');
    }
  });

  sleep(0.5);

  // ── Campaign service ───────────────────────────────────────────────────────
  group('campaign', () => {
    const res = http.get(url('/api/v1/campaigns'), {
      headers: authHeaders(token),
      tags:    { name: 'campaign:list' },
    });
    checkOK(res, 'campaign:list');
  });

  sleep(0.5);

  // ── Notification service ───────────────────────────────────────────────────
  group('notification', () => {
    const res = http.get(url('/api/v1/notifications'), {
      headers: authHeaders(token),
      tags:    { name: 'notification:list' },
    });
    if (token) {
      checkOK(res, 'notification:list');
    } else {
      checkStatus(res, 401, 'notification:list:no-auth');
    }
  });

  sleep(1);
}
