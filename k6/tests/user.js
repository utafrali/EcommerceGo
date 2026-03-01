/**
 * User Service — k6 Tests
 *
 * Tests auth flows, profile management.
 * Run: k6 run k6/tests/user.js
 */

import http from 'k6/http';
import { sleep, group, check } from 'k6';
import {
  checkOK, checkStatus,
  LOAD_THRESHOLDS, url, authHeaders, jsonHeaders,
  makeServiceMetrics, randomEmail,
} from '../lib/helpers.js';
import { login, registerAndLogin } from '../lib/auth.js';

export const options = {
  vus:        5,
  duration:   '2m',
  thresholds: {
    ...LOAD_THRESHOLDS,
    'http_req_duration{name:auth:login}':    ['p(95)<500'],
    'http_req_duration{name:auth:register}': ['p(95)<700'],
    'http_req_duration{name:user:profile}':  ['p(95)<400'],
  },
  tags: { service: 'user' },
};

const metrics = makeServiceMetrics('user');

export function setup() {
  return { token: login(0) };
}

export default function (data) {
  const { token } = data;

  group('auth:register+login', () => {
    // Each iteration: register a unique user and log in
    const email    = randomEmail();
    const password = 'Test1234!k6';

    metrics.reqs.add(1);
    const regRes = http.post(
      url('/api/v1/auth/register'),
      JSON.stringify({ email, password, first_name: 'Load', last_name: 'Test' }),
      { headers: jsonHeaders(), tags: { name: 'auth:register' } },
    );
    const regOk = check(regRes, {
      'auth:register: 201': (r) => r.status === 201,
    });
    metrics.errors.add(!regOk);
    metrics.duration.add(regRes.timings.duration);

    if (!regOk) {
      sleep(1);
      return;
    }

    sleep(0.2);

    const loginRes = http.post(
      url('/api/v1/auth/login'),
      JSON.stringify({ email, password }),
      { headers: jsonHeaders(), tags: { name: 'auth:login' } },
    );
    const loginOk = check(loginRes, {
      'auth:login: 200':    (r) => r.status === 200,
      'auth:login: token':  (r) => {
        try { return JSON.parse(r.body).token != null; } catch (_) { return false; }
      },
    });
    metrics.errors.add(!loginOk);
    metrics.duration.add(loginRes.timings.duration);
  });

  sleep(0.5);

  group('user:profile', () => {
    if (!token) return;

    metrics.reqs.add(1);
    const res = http.get(
      url('/api/v1/users/me'),
      { headers: authHeaders(token), tags: { name: 'user:profile' } },
    );
    const ok = checkOK(res, 'user:profile');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(0.5);

  group('user:addresses', () => {
    if (!token) return;

    metrics.reqs.add(1);
    const res = http.get(
      url('/api/v1/users/me/addresses'),
      { headers: authHeaders(token), tags: { name: 'user:addresses' } },
    );
    const ok = checkOK(res, 'user:addresses');
    metrics.duration.add(res.timings.duration);
    metrics.errors.add(!ok);
  });

  sleep(1);
}
