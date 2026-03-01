/**
 * Stress Test — EcommerceGo
 *
 * Gradually increases load to find the system's breaking point.
 * Monitor Grafana dashboards while running to see saturation.
 *
 * Run: k6 run k6/scenarios/stress.js
 * WARNING: This test intentionally pushes the system hard. Use in staging only.
 */

import http from 'k6/http';
import { sleep, group } from 'k6';
import { checkOK, url, authHeaders } from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  stages: [
    { duration: '2m',  target: 50  },  // warm up
    { duration: '5m',  target: 100 },  // moderate load
    { duration: '5m',  target: 200 },  // heavy load
    { duration: '5m',  target: 400 },  // very heavy load
    { duration: '5m',  target: 600 },  // extreme load
    { duration: '5m',  target: 0   },  // recovery
  ],
  thresholds: {
    // Intentionally loose — we want to observe breaking point, not abort
    http_req_failed:   [{ threshold: 'rate<0.50', abortOnFail: false }],
    http_req_duration: [{ threshold: 'p(95)<5000', abortOnFail: false }],
  },
  tags: { scenario: 'stress' },
};

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

  // Mix of endpoints to stress all services
  const endpoints = [
    { path: '/api/v1/products',           name: 'product:list'    },
    { path: '/api/v1/search?q=shirt',     name: 'search:query'    },
    { path: '/api/v1/campaigns',          name: 'campaign:list'   },
    { path: '/health/live',               name: 'gateway:live'    },
    { path: '/api/v1/categories',         name: 'product:cats'    },
  ];

  const ep = endpoints[__ITER % endpoints.length];

  group(ep.name, () => {
    const res = http.get(url(ep.path), {
      headers: authHeaders(token),
      tags:    { name: ep.name },
    });
    checkOK(res, ep.name);
  });

  sleep(0.1);
}
