import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 5 }, 
    { duration: '30s', target: 5 }, 
    { duration: '10s', target: 0 }, 
  ],
  thresholds: {
    http_req_duration: ['p(95)<300'], 
    http_req_failed: ['rate<0.001'],  
  },
};

const BASE_URL = 'http://localhost:8080';

export function setup() {
  const teamPayload = JSON.stringify({
    team_name: "load-test-team",
    members: [
      { user_id: "u1", username: "Author", is_active: true },
      { user_id: "u2", username: "Rev1", is_active: true },
      { user_id: "u3", username: "Rev2", is_active: true },
      { user_id: "u4", username: "Rev3", is_active: true },
    ]
  });
  http.post(`${BASE_URL}/team/add`, teamPayload);
}

export default function () {
  const prId = `pr-${Math.random().toString(36).substring(7)}`;
  
  const payload = JSON.stringify({
    pull_request_id: prId,
    pull_request_name: "Load Test PR",
    author_id: "u1"
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(`${BASE_URL}/pullRequest/create`, payload, params);

  check(res, {
    'is status 201': (r) => r.status === 201,
    'has reviewers': (r) => r.json('pr.assigned_reviewers').length > 0,
  });

  sleep(1); 
}