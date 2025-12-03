import http from 'k6/http';
import { check } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import encoding from 'k6/encoding';

// Custom metrics
const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');
const llmCallsPerRequest = new Trend('llm_calls_per_request');

// Configuration from environment variables
const MAX_RPS = parseInt(__ENV.MAX_RPS || '100');
const DURATION = __ENV.DURATION || '30s';
const RAMP_UP_TIME = __ENV.RAMP_UP_TIME || '15s';
const BASE_URL = __ENV.BASE_URL || 'http://objectweaver:2008';
const PASSWORD = __ENV.PASSWORD || 'test-password';
const SCHEMA_TYPE = __ENV.SCHEMA_TYPE || 'simple';  // simple, nested, complex, minimal

// Estimated LLM ROUNDS per schema type (not total calls)
// ObjectWeaver makes parallel LLM calls, so depth matters more than breadth
const LLM_ROUNDS_ESTIMATE = {
  minimal: 1,   // 1 field at 1 level = 1 round
  simple: 1,    // 3 fields at 1 level = 1 round (parallel)
  nested: 1,    // nested fields but still 1 level deep = 1 round
  complex: 2    // arrays with nested objects = 2 rounds
};

// Test configuration
export const options = {
  scenarios: {
    matrix_test: {
      executor: 'ramping-arrival-rate',
      startRate: 1,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: Math.max(MAX_RPS * 3, 500),  // Scale VUs with target RPS
      stages: [
        { duration: RAMP_UP_TIME, target: MAX_RPS },
        { duration: DURATION, target: MAX_RPS },
        { duration: '5s', target: 0 },
      ],
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.20'],  // Allow up to 20% errors for stress testing
    'errors': ['rate<0.20'],
  },
};

// Schema definitions with varying complexity
const schemas = {
  // Minimal: 1 field, 1 LLM call
  minimal: {
    type: 'object',
    instruction: 'Generate a single name',
    properties: {
      name: { 
        type: 'string',
        instruction: 'A person name'
      }
    },
    required: ['name']
  },

  // Simple: 3 fields, ~3 LLM calls
  simple: {
    type: 'object',
    instruction: 'Generate a simple user profile',
    properties: {
      name: { 
        type: 'string',
        instruction: 'Full name'
      },
      age: { 
        type: 'number',
        instruction: 'Age in years'
      },
      email: { 
        type: 'string',
        instruction: 'Email address'
      }
    },
    required: ['name', 'email']
  },
  
  // Nested: ~7 fields across 2 levels, ~7 LLM calls
  nested: {
    type: 'object',
    instruction: 'Generate a user with address',
    properties: {
      id: { 
        type: 'string',
        instruction: 'User ID'
      },
      name: { 
        type: 'string',
        instruction: 'Full name'
      },
      email: { 
        type: 'string',
        instruction: 'Email'
      },
      address: {
        type: 'object',
        instruction: 'Address info',
        properties: {
          street: { 
            type: 'string',
            instruction: 'Street'
          },
          city: { 
            type: 'string',
            instruction: 'City'
          },
          zipCode: { 
            type: 'string',
            instruction: 'Zip'
          },
          country: {
            type: 'string',
            instruction: 'Country'
          }
        }
      }
    },
    required: ['id', 'name']
  },
  
  // Complex: ~15+ fields with arrays, ~15+ LLM calls
  complex: {
    type: 'object',
    instruction: 'Generate a user profile with orders',
    properties: {
      id: { 
        type: 'string',
        instruction: 'User ID'
      },
      name: { 
        type: 'string',
        instruction: 'Name'
      },
      email: { 
        type: 'string',
        instruction: 'Email'
      },
      address: {
        type: 'object',
        instruction: 'Address',
        properties: {
          street: { type: 'string', instruction: 'Street' },
          city: { type: 'string', instruction: 'City' },
          zipCode: { type: 'string', instruction: 'Zip' }
        }
      },
      orders: {
        type: 'array',
        instruction: 'Orders list',
        minItems: 2,
        maxItems: 2,
        items: {
          type: 'object',
          instruction: 'Order',
          properties: {
            orderId: { type: 'string', instruction: 'Order ID' },
            amount: { type: 'number', instruction: 'Amount' },
            status: { type: 'string', instruction: 'Status' }
          }
        }
      },
      preferences: {
        type: 'object',
        instruction: 'Preferences',
        properties: {
          newsletter: { type: 'boolean', instruction: 'Newsletter opt-in' },
          theme: { type: 'string', instruction: 'UI theme preference' }
        }
      }
    },
    required: ['id', 'name']
  }
};

// Get the schema based on env var
function getSchema() {
  const schema = schemas[SCHEMA_TYPE];
  if (!schema) {
    console.error(`Unknown schema type: ${SCHEMA_TYPE}, defaulting to simple`);
    return schemas.simple;
  }
  return schema;
}

// Main test function
export default function () {
  const schema = getSchema();
  const estimatedRounds = LLM_ROUNDS_ESTIMATE[SCHEMA_TYPE] || 1;
  
  const payload = JSON.stringify({
    prompt: 'Generate test data',
    definition: schema,
    model: 'gpt-4',
    numberOfItems: 1
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Basic ${encoding.b64encode(`user:${PASSWORD}`)}`,
    },
    timeout: '180s',  // 3 minute timeout for complex schemas with slow LLM
    tags: {
      schema_type: SCHEMA_TYPE,
      target_rps: String(MAX_RPS)
    }
  };

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/objectGen`, payload, params);
  const duration = Date.now() - startTime;

  // Record metrics
  requestDuration.add(duration);
  llmCallsPerRequest.add(estimatedRounds);
  
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response has body': (r) => r.body && r.body.length > 0,
  });

  if (success) {
    successfulRequests.add(1);
    errorRate.add(0);
  } else {
    failedRequests.add(1);
    errorRate.add(1);
    if (response.status !== 200) {
      console.warn(`Request Failed: ${response.status} - ${response.body ? response.body.substring(0, 100) : 'null'}`);
    }
  }
}

// Custom summary
export function handleSummary(data) {
  const llmLatency = parseInt(__ENV.LLM_LATENCY_MS || '5000');
  const estimatedRounds = LLM_ROUNDS_ESTIMATE[SCHEMA_TYPE] || 1;
  const theoreticalMin = (llmLatency * estimatedRounds) / 1000;
  
  // Debug: print available metrics
  console.log('Available http_req_duration values:', JSON.stringify(data.metrics.http_req_duration?.values || {}));
  
  // Safely get metrics with defaults - use avg for p50 fallback since k6 may not calculate p(50)
  const httpDuration = data.metrics.http_req_duration;
  const avg = httpDuration && httpDuration.values['avg'] ? httpDuration.values['avg'] / 1000 : 0;
  const p50 = httpDuration && httpDuration.values['p(50)'] ? httpDuration.values['p(50)'] / 1000 : avg;
  const p90 = httpDuration && httpDuration.values['p(90)'] ? httpDuration.values['p(90)'] / 1000 : 0;
  const p95 = httpDuration && httpDuration.values['p(95)'] ? httpDuration.values['p(95)'] / 1000 : 0;
  const p99 = httpDuration && httpDuration.values['p(99)'] ? httpDuration.values['p(99)'] / 1000 : 0;
  const errorPct = data.metrics.errors && data.metrics.errors.values ? (data.metrics.errors.values.rate * 100) : 0;
  const throughput = data.metrics.http_reqs && data.metrics.http_reqs.values ? data.metrics.http_reqs.values.rate : 0;
  const totalReqs = data.metrics.http_reqs && data.metrics.http_reqs.values ? data.metrics.http_reqs.values.count : 0;
  
  // Calculate ObjectWeaver overhead based on average
  const overheadMs = (avg - theoreticalMin) * 1000;
  const overheadPct = theoreticalMin > 0 ? ((avg - theoreticalMin) / theoreticalMin * 100) : 0;

  const summary = `
================================================================================
MATRIX TEST RESULTS
================================================================================
Schema Type:     ${SCHEMA_TYPE}
Target RPS:      ${MAX_RPS}
LLM Latency:     ${llmLatency}ms
LLM Rounds:      ${estimatedRounds} (parallel calls per round)
Theoretical Min: ${theoreticalMin.toFixed(1)}s (${estimatedRounds} rounds × ${llmLatency/1000}s)
================================================================================

LATENCY:
  avg:  ${avg.toFixed(2)}s
  p50:  ${p50.toFixed(2)}s
  p90:  ${p90.toFixed(2)}s
  p95:  ${p95.toFixed(2)}s

OVERHEAD:
  ${overheadMs.toFixed(0)}ms  (${overheadPct > 0 ? '+' : ''}${overheadPct.toFixed(1)}%)

THROUGHPUT:
  Achieved:      ${throughput.toFixed(1)} req/s
  Total Requests: ${totalReqs}
  
RELIABILITY:
  Error Rate:    ${errorPct.toFixed(2)}%
  
================================================================================
CSV: ${SCHEMA_TYPE},${MAX_RPS},${llmLatency},${estimatedRounds},${theoreticalMin.toFixed(1)},${avg.toFixed(2)},${p90.toFixed(2)},${p95.toFixed(2)},${throughput.toFixed(1)},${errorPct.toFixed(2)},${overheadMs.toFixed(0)}
================================================================================
`;

  return {
    'stdout': summary,
    '/results/matrix-result.txt': summary,
  };
}