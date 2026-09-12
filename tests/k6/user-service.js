import http from 'k6/http';
import { check, fail, sleep } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const userID = __ENV.USER_ID || '11111111-1111-1111-1111-111111111111';
const organizationID = __ENV.ORGANIZATION_ID || '22222222-2222-2222-2222-222222222222';
const targetVUs = Number(__ENV.VUS || 10);
const thinkTime = Number(__ENV.SLEEP_SECONDS || 1);

const headers = {
  'Content-Type': 'application/json',
  'X-User-ID': userID,
  'X-Organization-ID': organizationID,
};

export const options = {
  stages: [
    { duration: __ENV.RAMP_UP || '10s', target: targetVUs },
    { duration: __ENV.HOLD || '30s', target: targetVUs },
    { duration: __ENV.RAMP_DOWN || '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
    checks: ['rate>0.99'],
  },
};

export function setup() {
  const response = http.get(`${baseURL}/health/ready`, { tags: { name: 'GET /health/ready' } });
  if (!check(response, { 'service is ready': (res) => res.status === 200 })) {
    fail(`user-service is not ready at ${baseURL}`);
  }
}

export default function () {
  const uniqueID = `${__VU}-${__ITER}-${Date.now()}`;
  const createResponse = http.post(
    `${baseURL}/users`,
    JSON.stringify({
      name: `Load Test User ${uniqueID}`,
      email: `load-test-${uniqueID}@example.test`,
    }),
    { headers, tags: { name: 'POST /users' } },
  );

  const created = check(createResponse, {
    'create user returns 201': (res) => res.status === 201,
    'create user returns an ID': (res) => {
      try {
        return Boolean(res.json('id'));
      } catch (_) {
        return false;
      }
    },
  });
  if (!created) {
    return;
  }

  const userID = createResponse.json('id');
  const getResponse = http.get(`${baseURL}/users/${userID}`, {
    headers,
    tags: { name: 'GET /users/{id}' },
  });
  check(getResponse, {
    'get user returns 200': (res) => res.status === 200,
    'get user returns created ID': (res) => res.json('id') === userID,
  });

  const listResponse = http.get(`${baseURL}/users`, {
    headers,
    tags: { name: 'GET /users' },
  });
  check(listResponse, {
    'list users returns 200': (res) => res.status === 200,
    'list users includes created user': (res) => {
      try {
        const users = res.json();
        return Array.isArray(users) && users.some((user) => user.id === userID);
      } catch (_) {
        return false;
      }
    },
  });

  sleep(thinkTime);
}
