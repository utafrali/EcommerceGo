/**
 * Search Service — k6 Tests
 *
 * Tests search query performance and correctness.
 * Run: k6 run k6/tests/search.js
 */

import http from 'k6/http';
import { sleep, group, check } from 'k6';
import {
  checkOK, LOAD_THRESHOLDS, url, authHeaders,
  makeServiceMetrics, randomChoice,
} from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  vus:        20,
  duration:   '2m',
  thresholds: {
    ...LOAD_THRESHOLDS,
    'http_req_duration{name:search:query}':    ['p(95)<1000'],
    'http_req_duration{name:search:suggest}':  ['p(95)<500'],
  },
  tags: { service: 'search' },
};

const metrics = makeServiceMetrics('search');

const SEARCH_QUERIES = [
  'shirt', 'dress', 'jacket', 'shoes', 'bag', 'jeans', 'hoodie',
  'sneakers', 'boots', 'coat', 'skirt', 'cardigan', 'blazer', 'shorts',
  'swimwear', 'activewear', 'accessories', 'hat', 'gloves', 'scarf',
];
const SORT_OPTIONS  = ['price_asc', 'price_desc', 'newest', 'popular', 'relevance'];
const PAGE_SIZES    = [10, 20, 40];

export function setup() {
  return { token: login(0) };
}

export default function (data) {
  const { token } = data;
  const hdrs = authHeaders(token);

  group('search:query', () => {
    metrics.reqs.add(1);
    const q        = randomChoice(SEARCH_QUERIES);
    const sort     = randomChoice(SORT_OPTIONS);
    const pageSize = randomChoice(PAGE_SIZES);

    const res = http.get(
      url(`/api/v1/search?q=${encodeURIComponent(q)}&sort=${sort}&page=1&page_size=${pageSize}`),
      { headers: hdrs, tags: { name: 'search:query' } },
    );

    const ok = check(res, {
      'search:query: 2xx':  (r) => r.status >= 200 && r.status < 300,
      'search:query: body': (r) => r.body && r.body.length > 0,
    });
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.5);

  group('search:category-filter', () => {
    metrics.reqs.add(1);
    const q = randomChoice(SEARCH_QUERIES);

    const res = http.get(
      url(`/api/v1/search?q=${q}&category=tops&page=1&page_size=20`),
      { headers: hdrs, tags: { name: 'search:filter' } },
    );

    const ok = checkOK(res, 'search:category-filter');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.5);

  group('search:empty-query', () => {
    // Empty query should return all products or 400
    const res = http.get(
      url('/api/v1/search?q=&page=1&page_size=20'),
      { headers: hdrs, tags: { name: 'search:empty' } },
    );
    check(res, {
      'search:empty: valid response': (r) => r.status === 200 || r.status === 400,
    });
  });

  sleep(1);
}
