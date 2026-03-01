/**
 * Checkout Service — k6 Tests
 *
 * Tests the checkout saga flow under load.
 * NOTE: Checkout is write-heavy; use low VU count to avoid overwhelming the saga.
 *
 * Run: k6 run k6/tests/checkout.js
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
  vus:        3,  // checkout is expensive; keep VU count low
  duration:   '2m',
  thresholds: {
    http_req_failed:                               [{ threshold: 'rate<0.10', abortOnFail: false }],
    'http_req_duration{name:checkout:initiate}':   ['p(95)<3000'],
    'http_req_duration{name:checkout:status}':     ['p(95)<500'],
  },
  tags: { service: 'checkout' },
};

const metrics = makeServiceMetrics('checkout');

// Fake cart/address IDs — replace with seeded data in real env
const ADDRESS_IDS = ['addr-0001', 'addr-0002', 'addr-0003'];

export function setup() {
  const tokens = [];
  for (let i = 0; i < 3; i++) {
    const t = login(i);
    if (t) tokens.push(t);
  }
  return { tokens };
}

export default function (data) {
  const { tokens } = data;
  const token = tokens.length > 0 ? tokens[__VU % tokens.length] : null;

  if (!token) {
    sleep(2);
    return;
  }

  const addressId = ADDRESS_IDS[randomInt(0, ADDRESS_IDS.length - 1)];

  group('checkout:initiate', () => {
    metrics.reqs.add(1);
    const res = http.post(
      url('/api/v1/checkout'),
      JSON.stringify({
        address_id:     addressId,
        payment_method: 'card',
        coupon_code:    null,
      }),
      { headers: jsonHeaders(token), tags: { name: 'checkout:initiate' } },
    );

    // Expect 201 Created or 400 (if cart empty / validation error)
    const ok = check(res, {
      'checkout:initiate: 2xx or 400': (r) => r.status === 201 || r.status === 400 || r.status === 409,
    });
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(res.status >= 500); // only 5xx counts as error

    // If created, check status
    if (res.status === 201) {
      let checkoutId;
      try {
        checkoutId = JSON.parse(res.body).id;
      } catch (_) {}

      if (checkoutId) {
        sleep(0.5);
        group('checkout:status', () => {
          const statusRes = http.get(
            url(`/api/v1/checkout/${checkoutId}`),
            { headers: authHeaders(token), tags: { name: 'checkout:status' } },
          );
          checkOK(statusRes, 'checkout:status');
        });
      }
    }
  });

  sleep(3); // checkout is slow; be kind to the saga
}
