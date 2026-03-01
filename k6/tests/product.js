/**
 * Product Service — k6 Tests
 *
 * Tests product listing, single product, categories, brands, search.
 * Run: k6 run k6/tests/product.js
 */

import http from 'k6/http';
import { sleep, group, check } from 'k6';
import {
  checkOK, checkJSON, checkStatus,
  LOAD_THRESHOLDS, url, authHeaders, jsonHeaders,
  makeServiceMetrics, randomInt,
} from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  vus:        10,
  duration:   '2m',
  thresholds: {
    ...LOAD_THRESHOLDS,
    'http_req_duration{name:product:list}':   ['p(95)<800'],
    'http_req_duration{name:product:detail}': ['p(95)<600'],
    'http_req_duration{name:product:cats}':   ['p(95)<400'],
    'http_req_duration{name:product:brands}': ['p(95)<400'],
  },
  tags: { service: 'product' },
};

const metrics = makeServiceMetrics('product');

export function setup() {
  return { token: login(0) };
}

export default function (data) {
  const { token } = data;
  const hdrs = authHeaders(token);

  group('product:list', () => {
    metrics.reqs.add(1);
    const page = randomInt(1, 5);
    const res = http.get(
      url(`/api/v1/products?page=${page}&page_size=20`),
      { headers: hdrs, tags: { name: 'product:list' } },
    );
    const ok = checkOK(res, 'product:list');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.3);

  group('product:categories', () => {
    metrics.reqs.add(1);
    const res = http.get(
      url('/api/v1/categories'),
      { headers: hdrs, tags: { name: 'product:cats' } },
    );
    const ok = checkOK(res, 'product:categories');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.3);

  group('product:brands', () => {
    metrics.reqs.add(1);
    const res = http.get(
      url('/api/v1/brands'),
      { headers: hdrs, tags: { name: 'product:brands' } },
    );
    const ok = checkOK(res, 'product:brands');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.3);

  group('product:banners', () => {
    metrics.reqs.add(1);
    const res = http.get(
      url('/api/v1/banners'),
      { headers: hdrs, tags: { name: 'product:banners' } },
    );
    const ok = checkOK(res, 'product:banners');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(1);
}
