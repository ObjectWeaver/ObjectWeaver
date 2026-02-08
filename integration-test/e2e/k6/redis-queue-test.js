import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import encoding from 'k6/encoding';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

const enqueueErrors = new Rate('enqueue_errors');
const pollErrors = new Rate('poll_errors');
const enqueueLatency = new Trend('enqueue_latency_ms');
const completionLatency = new Trend('completion_latency_ms');
const pollAttempts = new Trend('poll_attempts');
const completedRequests = new Counter('queued_completed');
const enqueueStatusCount = new Counter('enqueue_status');
const enqueueStatusErrors = new Counter('enqueue_status_errors');
let enqueueErrorLogCount = 0;

const QUEUED_RPS = __ENV.QUEUED_RPS || '50';
const DURATION = __ENV.DURATION || '60s';
const RAMP_UP_TIME = __ENV.RAMP_UP_TIME || '15s';
const BASE_URL = __ENV.BASE_URL || 'http://objectweaver:2008';
const PASSWORD = __ENV.PASSWORD || 'test-password';
const POLL_INTERVAL_MS = parseInt(__ENV.POLL_INTERVAL_MS || '200', 10);
const POLL_TIMEOUT_MS = parseInt(__ENV.POLL_TIMEOUT_MS || '120000', 10);
const PREALLOCATED_VUS = parseInt(__ENV.PREALLOCATED_VUS || '50', 10);
const MAX_VUS = parseInt(__ENV.MAX_VUS || '500', 10);

export const options = {
  scenarios: {
    queued_flow: {
      executor: 'ramping-arrival-rate',
      startRate: 1,
      timeUnit: '1s',
      preAllocatedVUs: PREALLOCATED_VUS,
      maxVUs: MAX_VUS,
      stages: [
        { duration: RAMP_UP_TIME, target: parseInt(QUEUED_RPS, 10) },
        { duration: DURATION, target: parseInt(QUEUED_RPS, 10) },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    enqueue_errors: ['rate<0.02'],
    poll_errors: ['rate<0.02'],
    enqueue_latency_ms: ['p(95)<200'],
    completion_latency_ms: ['p(95)<7000'],
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

function authHeaders() {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${__ENV.AUTH_TOKEN || PASSWORD}`,
  };
}

export default function () {
  const payload = JSON.stringify({
    prompt: 'Generate realistic test data',
    definition: schema,
    model: 'gpt-4',
    numberOfItems: 1,
  });

  const enqueueStart = Date.now();
  const enqueueRes = http.post(`${BASE_URL}/api/objectGenQueued`, payload, {
    headers: authHeaders(),
    timeout: '120s',
  });
  const enqueueDuration = Date.now() - enqueueStart;
  enqueueLatency.add(enqueueDuration);

  enqueueStatusCount.add(1, { status: String(enqueueRes.status) });
  const enqueueOk = check(enqueueRes, {
    'enqueue status is 202': (r) => r.status === 202 || r.status === 200,
  });

  if (!enqueueOk) {
    enqueueStatusErrors.add(1, { status: String(enqueueRes.status) });
    if (enqueueErrorLogCount < 5) {
      enqueueErrorLogCount += 1;
      console.error(`enqueue failed: status=${enqueueRes.status} body=${enqueueRes.body}`);
    }
    enqueueErrors.add(1);
    return;
  }

  let id = null;
  try {
    const data = enqueueRes.json();
    id = data && data.id ? data.id : null;
  } catch (err) {
    enqueueErrors.add(1);
    return;
  }

  const idOk = check({ id }, {
    'enqueue response has id': (p) => typeof p.id === 'string' && p.id.length > 0,
  });

  if (!idOk) {
    enqueueErrors.add(1);
    return;
  }

  const deadline = Date.now() + POLL_TIMEOUT_MS;
  let attempts = 0;
  let completed = false;

  while (Date.now() < deadline) {
    sleep(POLL_INTERVAL_MS / 1000);
    attempts += 1;

    const pollRes = http.get(`${BASE_URL}/api/getObjectQueued?id=${encodeURIComponent(id)}`, {
      headers: authHeaders(),
      timeout: '120s',
    });

    if (pollRes.status === 204) {
      continue;
    }

    if (pollRes.status !== 200) {
      pollErrors.add(1);
      break;
    }

    const completionMs = Date.now() - enqueueStart;
    completionLatency.add(completionMs);

    const parsed = check(pollRes, {
      'poll response has body': (r) => r.body && r.body.length > 0,
    });

    if (!parsed) {
      pollErrors.add(1);
      break;
    }

    let payload = null;
    try {
      payload = pollRes.json();
    } catch (err) {
      pollErrors.add(1);
      break;
    }

    const formatOk = check(payload || {}, {
      'response has data object': (p) => p && typeof p.data === 'object' && p.data !== null,
      'response has usdCost number': (p) => p && typeof p.usdCost === 'number',
    });

    if (!formatOk) {
      pollErrors.add(1);
      break;
    }

    completed = true;
    completedRequests.add(1);
    break;
  }

  pollAttempts.add(attempts);

  if (!completed) {
    pollErrors.add(1);
  }
}

export function setup() {
  const healthResponse = http.get(`${BASE_URL}/health`);
  if (healthResponse.status !== 200) {
    throw new Error(`Server health check failed: ${healthResponse.status}`);
  }
}

export function handleSummary(data) {
  const queued = data.metrics.queued_completed;
  const httpReqs = data.metrics.http_reqs;
  const queuedRate = queued && queued.values ? queued.values.rate : 0;
  const httpRate = httpReqs && httpReqs.values ? httpReqs.values.rate : 0;

  const summaryLines = [
    '',
    'Queued Throughput Summary',
    '==========================',
    `Queued completions/sec: ${queuedRate.toFixed(2)}`,
    `HTTP requests/sec:      ${httpRate.toFixed(2)}`,
    '',
  ];

  return {
    '/results/queued-summary.json': JSON.stringify(data, null, 2),
    '/results/queued-summary.txt': textSummary(data, { indent: ' ', enableColors: false }),
    stdout: summaryLines.join('\n') + textSummary(data, { indent: ' ', enableColors: true }),
  };
}
