/**
 * Load Test — EcommerceGo
 *
 * Sustained load simulating real traffic patterns.
 * Stages: ramp-up → steady → ramp-down
 *
 * Run: k6 run k6/scenarios/load.js
 * Run with more VUs: BASE_URL=http://staging:8080 k6 run k6/scenarios/load.js
 */

import http from 'k6/http';
import { sleep, group } from 'k6';
import {
  checkOK, checkStatus, LOAD_THRESHOLDS,
  url, authHeaders, jsonHeaders,
  randomChoice, randomInt,
} from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  stages: [
    { duration: '1m',  target: 10  }, // ramp-up
    { duration: '3m',  target: 50  }, // steady load
    { duration: '1m',  target: 100 }, // peak
    { duration: '3m',  target: 50  }, // sustained peak
    { duration: '1m',  target: 0   }, // ramp-down
  ],
  thresholds: {
    ...LOAD_THRESHOLDS,
    'http_req_duration{name:product:list}':  ['p(95)<800'],
    'http_req_duration{name:search:query}':  ['p(95)<1000'],
    'http_req_duration{name:cart:get}':      ['p(95)<500'],
    'http_req_duration{name:gateway:live}':  ['p(95)<100'],
  },
  tags: { scenario: 'load' },
};

const SEARCH_TERMS = ['shirt', 'dress', 'jacket', 'shoes', 'bag', 'jeans', 'hoodie', 'sneakers'];
const SORT_OPTIONS = ['price_asc', 'price_desc', 'newest', 'popular'];

// Each VU gets its own token via setup per-VU pattern
export function setup() {
  // Pre-login a pool of tokens (one per 10 VUs max)
  const tokens = [];
  const poolSize = 5;
  for (let i = 0; i < poolSize; i++) {
    const token = login(i);
    if (token) tokens.push(token);
  }
  return { tokens };
}

export default function (data) {
  const { tokens } = data;
  const token = tokens.length > 0
    ? tokens[__VU % tokens.length]
    : null;

  // Realistic user journey weights:
  // 60% browsing, 25% searching, 10% cart, 5% orders

  const roll = Math.random();

  if (roll < 0.60) {
    // Browse products
    browseProducts(token);
  } else if (roll < 0.85) {
    // Search
    searchProducts(token);
  } else if (roll < 0.95) {
    // Cart operations
    cartOperations(token);
  } else {
    // Order history
    viewOrders(token);
  }

  sleep(randomInt(1, 3));
}

function browseProducts(token) {
  group('browse', () => {
    // List products
    const listRes = http.get(url('/api/v1/products?page=1&page_size=20'), {
      headers: authHeaders(token),
      tags:    { name: 'product:list' },
    });
    checkOK(listRes, 'product:list');

    sleep(0.5);

    // View a category
    const catRes = http.get(url('/api/v1/categories'), {
      headers: authHeaders(token),
      tags:    { name: 'product:categories' },
    });
    checkOK(catRes, 'product:categories');

    // Check active campaigns
    const campRes = http.get(url('/api/v1/campaigns?status=active'), {
      headers: authHeaders(token),
      tags:    { name: 'campaign:active' },
    });
    checkOK(campRes, 'campaign:active');
  });
}

function searchProducts(token) {
  group('search', () => {
    const term = randomChoice(SEARCH_TERMS);
    const sort = randomChoice(SORT_OPTIONS);

    const res = http.get(url(`/api/v1/search?q=${term}&sort=${sort}&page=1&page_size=20`), {
      headers: authHeaders(token),
      tags:    { name: 'search:query' },
    });
    checkOK(res, 'search:query');
  });
}

function cartOperations(token) {
  if (!token) return; // cart requires auth

  group('cart', () => {
    // Get cart
    const getRes = http.get(url('/api/v1/cart'), {
      headers: authHeaders(token),
      tags:    { name: 'cart:get' },
    });
    checkOK(getRes, 'cart:get');
  });
}

function viewOrders(token) {
  if (!token) return;

  group('orders', () => {
    const res = http.get(url('/api/v1/orders?page=1&page_size=10'), {
      headers: authHeaders(token),
      tags:    { name: 'order:list' },
    });
    checkOK(res, 'order:list');
  });
}
