/**
 * Inventory Service — k6 Tests
 *
 * Tests stock checks, bulk availability queries.
 * Run: k6 run k6/tests/inventory.js
 */

import http from 'k6/http';
import { sleep, group, check } from 'k6';
import {
  checkOK,
  LOAD_THRESHOLDS, url, authHeaders, jsonHeaders,
  makeServiceMetrics, randomInt,
} from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  vus:        15,
  duration:   '2m',
  thresholds: {
    ...LOAD_THRESHOLDS,
    'http_req_duration{name:inventory:stock}':       ['p(95)<400'],
    'http_req_duration{name:inventory:bulk-check}':  ['p(95)<600'],
  },
  tags: { service: 'inventory' },
};

const metrics = makeServiceMetrics('inventory');

const PRODUCT_IDS = [
  'prod-00000001', 'prod-00000002', 'prod-00000003',
  'prod-00000004', 'prod-00000005', 'prod-00000006',
];

export function setup() {
  return { token: login(0) };
}

export default function (data) {
  const { token } = data;
  const hdrs = authHeaders(token);

  group('inventory:stock', () => {
    metrics.reqs.add(1);
    const productId = PRODUCT_IDS[randomInt(0, PRODUCT_IDS.length - 1)];
    const res = http.get(
      url(`/api/v1/inventory/${productId}`),
      { headers: hdrs, tags: { name: 'inventory:stock' } },
    );
    // 200 OK or 404 (product not seeded)
    const ok = check(res, {
      'inventory:stock: 200 or 404': (r) => r.status === 200 || r.status === 404,
    });
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(res.status >= 500);
  });

  sleep(0.3);

  group('inventory:bulk-check', () => {
    metrics.reqs.add(1);
    const count     = randomInt(2, 5);
    const productIds = PRODUCT_IDS.slice(0, count);
    const items      = productIds.map(id => ({ product_id: id, quantity: randomInt(1, 3) }));

    const res = http.post(
      url('/api/v1/inventory/check'),
      JSON.stringify({ items }),
      { headers: jsonHeaders(token), tags: { name: 'inventory:bulk-check' } },
    );
    const ok = check(res, {
      'inventory:bulk-check: 2xx or 404': (r) => r.status === 200 || r.status === 404,
    });
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(res.status >= 500);
  });

  sleep(0.5);
}
