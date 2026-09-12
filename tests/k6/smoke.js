import http from 'k6/http';
import { check } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const headers = {
  'X-User-ID': __ENV.USER_ID || '11111111-1111-1111-1111-111111111111',
  'X-Organization-ID': __ENV.ORGANIZATION_ID || '22222222-2222-2222-2222-222222222222',
};

export const options = {
  thresholds: {
    http_req_failed: ['rate==0'],
    checks: ['rate==1'],
  },
};

export default function () {
  const response = http.get(`${baseURL}/users`, {
    headers,
    tags: { name: 'GET /users' },
  });
  check(response, {
    'list users returns 200': (res) => res.status === 200,
    'list users returns an array': (res) => {
      try {
        return Array.isArray(res.json());
      } catch (_) {
        return false;
      }
    },
  });
}
