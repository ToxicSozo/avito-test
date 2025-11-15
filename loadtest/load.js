import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 5,
  duration: '1m',
  thresholds: {
    http_req_duration: ['p(99)<300'],
    http_req_failed: ['rate<0.001'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const USER_TOKEN = __ENV.USER_TOKEN || 'super-secret-user';

let prCounter = 0;

export default function () {
  const prId = `load-pr-${__VU}-${prCounter++}`;
  const payload = JSON.stringify({
    pull_request_id: prId,
    pull_request_name: 'load-test',
    author_id: 'u1',
  });
  const headers = {
    Authorization: `Bearer ${USER_TOKEN}`,
    'Content-Type': 'application/json',
  };
  const res = http.post(`${BASE_URL}/pullRequest/create`, payload, { headers });

  check(res, {
    'status is 201': (r) => r.status === 201,
  });

  sleep(0.2);
}
