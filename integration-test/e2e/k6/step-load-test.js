import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import encoding from 'k6/encoding';

const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');

const BASE_URL = __ENV.BASE_URL || 'http://objectweaver:2008';
const PASSWORD = __ENV.PASSWORD || 'test-password';
const STEP_1_RPS = __ENV.STEP_1_RPS || '2000';
const STEP_2_RPS = __ENV.STEP_2_RPS || '3000';
const STEP_3_RPS = __ENV.STEP_3_RPS || '4000';
const STEP_DURATION = __ENV.STEP_DURATION || '60s';
const RAMP_TIME = __ENV.RAMP_TIME || '20s';

const targetRps = parseInt(STEP_3_RPS, 10);
const estimatedLatency = 6;
const VU_BUFFER = 1.5;
const calculatedMaxVUs = Math.ceil(targetRps * estimatedLatency * VU_BUFFER);
const maxVUs = Math.max(calculatedMaxVUs, 500);

export const options = {
  scenarios: {
    stepped_load: {
      executor: 'ramping-arrival-rate',
      startRate: 1,
      timeUnit: '1s',
      preAllocatedVUs: Math.min(500, Math.ceil(maxVUs / 10)),
      maxVUs,
      stages: [
        { duration: RAMP_TIME, target: parseInt(STEP_1_RPS, 10) },
        { duration: STEP_DURATION, target: parseInt(STEP_1_RPS, 10) },
        { duration: RAMP_TIME, target: parseInt(STEP_2_RPS, 10) },
        { duration: STEP_DURATION, target: parseInt(STEP_2_RPS, 10) },
        { duration: RAMP_TIME, target: parseInt(STEP_3_RPS, 10) },
        { duration: STEP_DURATION, target: parseInt(STEP_3_RPS, 10) },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<8000', 'p(99)<12000'],
    'http_req_failed': ['rate<0.05'],
    'errors': ['rate<0.05'],
  },
};

const schema = {
  type: 'object',
  instruction: 'Generate a simple user profile with basic information',
  properties: {
    name: { type: 'string', instruction: 'Full name of the person' },
    age: { type: 'number', instruction: 'Age in years' },
    email: { type: 'string', instruction: 'Valid email address' },
  },
  required: ['name', 'email'],
};

export default function () {
  const payload = JSON.stringify({
    prompt: 'Generate realistic test data',
    definition: schema,
    model: 'gpt-4',
    numberOfItems: 1,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Basic ${__ENV.AUTH_TOKEN || encoding.b64encode(`user:${PASSWORD}`)}`,
    },
    timeout: '120s',
  };

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/objectGen`, payload, params);
  const duration = Date.now() - startTime;

  requestDuration.add(duration);

  const ok = check(response, {
    'status is 200': (r) => r.status === 200,
    'response has body': (r) => r.body && r.body.length > 0,
  });

  if (ok) {
    successfulRequests.add(1);
    errorRate.add(0);
  } else {
    failedRequests.add(1);
    errorRate.add(1);
  }

  sleep(Math.random() * 0.1);
}

export function setup() {
  const healthResponse = http.get(`${BASE_URL}/health`);
  if (healthResponse.status !== 200) {
    throw new Error(`Server health check failed: ${healthResponse.status}`);
  }
}
