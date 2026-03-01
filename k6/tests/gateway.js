/**
 * Gateway — k6 Tests
 *
 * Tests gateway-level concerns: health, rate limiting, JWT validation, routing.
 * Run: k6 run k6/tests/gateway.js
 */

import http from 'k6/http';
import { sleep, group, check } from 'k6';
import {
  checkOK, checkStatus,
  SMOKE_THRESHOLDS, url, authHeaders, jsonHeaders,
  makeServiceMetrics,
} from '../lib/helpers.js';
import { login } from '../lib/auth.js';

export const options = {
  vus:        5,
  duration:   '1m',
  thresholds: {
    ...SMOKE_THRESHOLDS,
    'http_req_duration{name:gateway:live}':   ['p(95)<100'],
    'http_req_duration{name:gateway:ready}':  ['p(95)<200'],
  },
  tags: { service: 'gateway' },
};

const metrics = makeServiceMetrics('gateway');

export function setup() {
  return { token: login(0) };
}

export default function (data) {
  const { token } = data;

  group('gateway:health', () => {
    metrics.reqs.add(1);
    const liveRes = http.get(url('/health/live'), { tags: { name: 'gateway:live' } });
    const ok1 = checkOK(liveRes, 'gateway:liveness');
    metrics.errors.add(!ok1);
    metrics.duration.add(liveRes.timings.duration);

    const readyRes = http.get(url('/health/ready'), { tags: { name: 'gateway:ready' } });
    const ok2 = checkOK(readyRes, 'gateway:readiness');
    metrics.errors.add(!ok2);
    metrics.duration.add(readyRes.timings.duration);
  });

  sleep(0.2);

  group('gateway:auth-required', () => {
    // Without token → 401
    const res = http.get(
      url('/api/v1/products'),
      { tags: { name: 'gateway:no-auth' } },
    );
    checkStatus(res, 401, 'gateway:no-auth');
  });

  sleep(0.2);

  group('gateway:jwt-validation', () => {
    // Invalid token → 401
    const res = http.get(
      url('/api/v1/products'),
      { headers: { Authorization: 'Bearer invalid.jwt.token' }, tags: { name: 'gateway:bad-jwt' } },
    );
    checkStatus(res, 401, 'gateway:bad-jwt');
  });

  sleep(0.2);

  group('gateway:valid-routing', () => {
    if (!token) return;
    // Valid token → proxied to product service
    const res = http.get(
      url('/api/v1/products'),
      { headers: authHeaders(token), tags: { name: 'gateway:routing' } },
    );
    checkOK(res, 'gateway:routing');
  });

  sleep(0.2);

  group('gateway:not-found', () => {
    const res = http.get(url('/api/v1/nonexistent-service'), {
      headers: authHeaders(token),
      tags:    { name: 'gateway:404' },
    });
    check(res, {
      'gateway:404: not found': (r) => r.status === 404 || r.status === 502,
    });
  });

  sleep(1);
}
