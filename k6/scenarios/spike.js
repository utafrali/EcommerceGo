/**
 * Spike Test — EcommerceGo
 *
 * Sudden traffic spike simulation (flash sale / viral event).
 * Verifies the system handles sudden burst and recovers gracefully.
 *
 * Run: k6 run k6/scenarios/spike.js
 */

import http from 'k6/http';
import { sleep, group } from 'k6';
import {
  checkOK, checkStatus, SPIKE_THRESHOLDS,
  url, authHeaders, randomChoice,
} from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  stages: [
    { duration: '30s', target: 5   }, // baseline
    { duration: '10s', target: 200 }, // spike!
    { duration: '1m',  target: 200 }, // sustain spike
    { duration: '10s', target: 5   }, // recovery
    { duration: '1m',  target: 5   }, // back to baseline
  ],
  thresholds: {
    ...SPIKE_THRESHOLDS,
    http_req_failed: [{ threshold: 'rate<0.15', abortOnFail: false }],
  },
  tags: { scenario: 'spike' },
};

const PRODUCT_IDS = ['prod-1', 'prod-2', 'prod-3', 'prod-4', 'prod-5'];

export function setup() {
  const token = login(0);
  return { token };
}

export default function (data) {
  const { token } = data;

  // During a spike, most traffic is read-heavy (product views + search)
  const roll = Math.random();

  if (roll < 0.50) {
    group('product:list:spike', () => {
      const res = http.get(url('/api/v1/products?page=1&page_size=20'), {
        headers: authHeaders(token),
        tags:    { name: 'product:list' },
      });
      checkOK(res, 'product:list:spike');
    });
  } else if (roll < 0.80) {
    group('search:spike', () => {
      const res = http.get(url('/api/v1/search?q=sale&sort=popular'), {
        headers: authHeaders(token),
        tags:    { name: 'search:spike' },
      });
      checkOK(res, 'search:spike');
    });
  } else if (roll < 0.90) {
    group('campaign:spike', () => {
      const res = http.get(url('/api/v1/campaigns?status=active'), {
        headers: authHeaders(token),
        tags:    { name: 'campaign:spike' },
      });
      checkOK(res, 'campaign:spike');
    });
  } else {
    group('health:spike', () => {
      const res = http.get(url('/health/live'), {
        tags: { name: 'gateway:live' },
      });
      checkOK(res, 'gateway:live:spike');
    });
  }

  // Very short sleep during spike (simulates high-concurrency)
  sleep(0.1 + Math.random() * 0.5);
}
