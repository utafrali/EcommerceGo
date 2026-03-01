/**
 * Cart Service — k6 Tests
 *
 * Tests cart CRUD operations, item management.
 * Run: k6 run k6/tests/cart.js
 */

import http from 'k6/http';
import { sleep, group, check } from 'k6';
import {
  checkOK, checkStatus,
  LOAD_THRESHOLDS, url, authHeaders, jsonHeaders,
  makeServiceMetrics, randomInt,
} from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  vus:        10,
  duration:   '2m',
  thresholds: {
    ...LOAD_THRESHOLDS,
    'http_req_duration{name:cart:get}':         ['p(95)<500'],
    'http_req_duration{name:cart:add-item}':    ['p(95)<700'],
    'http_req_duration{name:cart:remove-item}': ['p(95)<700'],
  },
  tags: { service: 'cart' },
};

const metrics = makeServiceMetrics('cart');

// Fake product IDs for load test (replace with real seeded IDs)
const PRODUCT_IDS = [
  'prod-00000001',
  'prod-00000002',
  'prod-00000003',
  'prod-00000004',
  'prod-00000005',
];

export function setup() {
  const tokens = [];
  for (let i = 0; i < 5; i++) {
    const t = login(i);
    if (t) tokens.push(t);
  }
  return { tokens };
}

export default function (data) {
  const { tokens } = data;
  const token = tokens.length > 0 ? tokens[__VU % tokens.length] : null;

  if (!token) {
    group('cart:no-auth', () => {
      const res = http.get(url('/api/v1/cart'), { tags: { name: 'cart:no-auth' } });
      checkStatus(res, 401, 'cart:no-auth');
    });
    sleep(1);
    return;
  }

  const hdrs = authHeaders(token);

  group('cart:get', () => {
    metrics.reqs.add(1);
    const res = http.get(
      url('/api/v1/cart'),
      { headers: hdrs, tags: { name: 'cart:get' } },
    );
    const ok = checkOK(res, 'cart:get');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.3);

  group('cart:add-item', () => {
    metrics.reqs.add(1);
    const productId = PRODUCT_IDS[randomInt(0, PRODUCT_IDS.length - 1)];
    const res = http.post(
      url('/api/v1/cart/items'),
      JSON.stringify({ product_id: productId, quantity: randomInt(1, 3) }),
      { headers: jsonHeaders(token), tags: { name: 'cart:add-item' } },
    );
    // 201 Created or 200 OK (item quantity updated)
    const ok = check(res, {
      'cart:add-item: 2xx': (r) => r.status >= 200 && r.status < 300,
    });
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.5);

  group('cart:get-after-add', () => {
    const res = http.get(
      url('/api/v1/cart'),
      { headers: hdrs, tags: { name: 'cart:get' } },
    );
    checkOK(res, 'cart:get-after-add');
  });

  sleep(1);
}
