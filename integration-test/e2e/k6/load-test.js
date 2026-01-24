import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import encoding from 'k6/encoding';

// Custom metrics
const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');

// Configuration from environment variables
const MAX_RPS = __ENV.MAX_RPS || '10';  // Maximum requests per second to ramp up to
const DURATION = __ENV.DURATION || '30s';  // Test duration
const RAMP_UP_TIME = __ENV.RAMP_UP_TIME || '15s';  // Time to reach max RPS
const BASE_URL = __ENV.BASE_URL || 'http://objectweaver:2008';
const PASSWORD = __ENV.PASSWORD || 'test-password';

// Test configuration
// Calculate VUs needed: VUs = target_rps × expected_latency_seconds
// With 5s LLM latency, each VU can only complete 0.2 req/s
// For 2000 req/s, we need: 2000 / 0.2 = 10,000 VUs minimum
// Adding 50% buffer: 15,000 VUs
const targetRps = parseInt(MAX_RPS);
const estimatedLatency = 6;  // 5s LLM + 1s buffer
const VU_BUFFER = 1.5;  // 50% extra for burst handling
const calculatedMaxVUs = Math.ceil(targetRps * estimatedLatency * VU_BUFFER);
const maxVUs = Math.max(calculatedMaxVUs, 500);  // Minimum 500 VUs

export const options = {
  scenarios: {
    ramping_load: {
      executor: 'ramping-arrival-rate',
      startRate: 1,                    // Start with 1 req/sec
      timeUnit: '1s',
      preAllocatedVUs: Math.min(200, Math.ceil(maxVUs / 10)),  // Pre-allocate 10%
      maxVUs: maxVUs,                  // VUs = RPS × latency × buffer
      stages: [
        { duration: RAMP_UP_TIME, target: parseInt(MAX_RPS) },  // Ramp up to MAX_RPS
        { duration: DURATION, target: parseInt(MAX_RPS) },      // Hold at MAX_RPS
        { duration: '10s', target: 0 },                         // Ramp down
      ],
    },
  },
  thresholds: {
    // With fixed 5s LLM latency, measure ObjectWeaver overhead:
    // - Base: 5000ms (LLM response time)
    // - Target overhead: < 500ms (so total < 5500ms)
    // - p(95) < 6000ms: allows 1s overhead for 95% of requests
    // - p(99) < 8000ms: allows 3s overhead for tail latency
    // If these fail, it indicates queueing/bottleneck issues!
    'http_req_duration': ['p(95)<6000', 'p(99)<8000'],
    'http_req_failed': ['rate<0.05'],  // Error rate < 5%
    'errors': ['rate<0.05'],
  },
};

// Sample schema definitions for testing
// Note: ObjectWeaver expects "definition" not "schema", and each object needs an "instruction" field
const schemas = {
  simple: {
    type: 'object',
    instruction: 'Generate a simple user profile with basic information',
    properties: {
      name: { 
        type: 'string',
        instruction: 'Full name of the person'
      },
      age: { 
        type: 'number',
        instruction: 'Age in years'
      },
      email: { 
        type: 'string',
        instruction: 'Valid email address'
      }
    },
    required: ['name', 'email']
  },
  
  nested: {
    type: 'object',
    instruction: 'Generate a user profile with address information',
    properties: {
      id: { 
        type: 'string',
        instruction: 'Unique identifier for the user'
      },
      name: { 
        type: 'string',
        instruction: 'Full name of the person'
      },
      email: { 
        type: 'string',
        instruction: 'Valid email address'
      },
      address: {
        type: 'object',
        instruction: 'Physical address of the user',
        properties: {
          street: { 
            type: 'string',
            instruction: 'Street address'
          },
          city: { 
            type: 'string',
            instruction: 'City name'
          },
          zipCode: { 
            type: 'string',
            instruction: 'Postal code'
          }
        }
      }
    },
    required: ['id', 'name', 'email']
  },
  
  complex: {
    type: 'object',
    instruction: 'Generate a comprehensive user profile with orders and preferences',
    properties: {
      id: { 
        type: 'string',
        instruction: 'Unique identifier for the user'
      },
      name: { 
        type: 'string',
        instruction: 'Full name of the person'
      },
      email: { 
        type: 'string',
        instruction: 'Valid email address'
      },
      address: {
        type: 'object',
        instruction: 'Physical address of the user',
        properties: {
          street: { 
            type: 'string',
            instruction: 'Street address'
          },
          city: { 
            type: 'string',
            instruction: 'City name'
          },
          zipCode: { 
            type: 'string',
            instruction: 'Postal code'
          }
        }
      },
      orders: {
        type: 'array',
        instruction: 'List of recent orders',
        items: {
          type: 'object',
          instruction: 'Individual order details',
          properties: {
            orderId: { 
              type: 'string',
              instruction: 'Unique order identifier'
            },
            amount: { 
              type: 'number',
              instruction: 'Total order amount in USD'
            },
            items: {
              type: 'array',
              instruction: 'Items in the order',
              items: {
                type: 'object',
                instruction: 'Individual product in order',
                properties: {
                  productId: { 
                    type: 'string',
                    instruction: 'Product identifier'
                  },
                  quantity: { 
                    type: 'number',
                    instruction: 'Number of units ordered'
                  }
                }
              }
            }
          }
        }
      },
      preferences: {
        type: 'object',
        instruction: 'User preferences and settings',
        properties: {
          newsletter: { 
            type: 'boolean',
            instruction: 'Whether user subscribes to newsletter'
          },
          notifications: { 
            type: 'boolean',
            instruction: 'Whether user enables notifications'
          }
        }
      }
    },
    required: ['id', 'name', 'email']
  }
};

// Get random schema based on distribution
// For 5s LLM latency tests, use minimal schema for maximum throughput
// For mixed load tests, uncomment the distribution below
function getRandomSchema() {
  // Use minimal schema only for consistent, high-throughput testing
  // With 5s LLM latency, this gives max ~2000 req/s theoretical
  return { name: 'simple', schema: schemas.simple };
  
  // Uncomment below for mixed-load testing (lower throughput)
  // const rand = Math.random();
  // if (rand < 0.5) {
  //   return { name: 'simple', schema: schemas.simple };
  // } else if (rand < 0.8) {
  //   return { name: 'nested', schema: schemas.nested };
  // } else {
  //   return { name: 'complex', schema: schemas.complex };
  // }
}

// Main test function
export default function () {
  const schemaConfig = getRandomSchema();
  
  const payload = JSON.stringify({
    prompt: 'Generate realistic test data',
    definition: schemaConfig.schema,  // ObjectWeaver expects "definition" not "schema"
    model: 'gpt-4',
    numberOfItems: 1
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Basic ${__ENV.AUTH_TOKEN || encoding.b64encode(`user:${PASSWORD}`)}`,
    },
    timeout: '120s',  // 2 minute timeout for slow LLM responses
    tags: {
      schema_complexity: schemaConfig.name
    }
  };

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/api/objectGen`, payload, params);
  const duration = Date.now() - startTime;

  // Record metrics
  requestDuration.add(duration);
  
  const functionalSuccess = check(response, {
    'status is 200': (r) => r.status === 200,
    'response has body': (r) => r.body && r.body.length > 0,
  });

  // With 5s LLM latency, overhead should be minimal (< 500ms)
  check(response, {
    'response time < 5500ms (5s LLM + 500ms overhead)': () => duration < 5500,
  });

  if (functionalSuccess) {
    successfulRequests.add(1);
    errorRate.add(0);
  } else {
    failedRequests.add(1);
    errorRate.add(1);
    console.error(`Request failed: ${response.status} - ${response.body}`);
  }

  // Small random sleep to vary request patterns
  sleep(Math.random() * 0.1);
}

// Setup function - runs once at the start
export function setup() {
  console.log('='.repeat(80));
  console.log('Starting ObjectWeaver Integration Load Test');
  console.log('='.repeat(80));
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Max RPS: ${MAX_RPS}`);
  console.log(`Duration: ${DURATION}`);
  console.log(`Ramp Up Time: ${RAMP_UP_TIME}`);
  console.log('='.repeat(80));
  
  // Health check
  const healthResponse = http.get(`${BASE_URL}/health`);
  if (healthResponse.status !== 200) {
    throw new Error(`Server health check failed: ${healthResponse.status}`);
  }
  console.log('✓ Server health check passed');
  
  return { startTime: Date.now() };
}

// Teardown function - runs once at the end
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('='.repeat(80));
  console.log('Test completed');
  console.log(`Total duration: ${duration.toFixed(2)}s`);
  console.log('='.repeat(80));
}

// Handle summary - custom summary output
export function handleSummary(data) {
  const timestamp = new Date().toISOString();
  
  return {
    '/results/summary.json': JSON.stringify(data, null, 2),
    '/results/summary.txt': textSummary(data, { indent: ' ', enableColors: false }),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}

// Helper function for text summary
function textSummary(data, options) {
  const indent = options.indent || '';
  const colors = options.enableColors;
  
  let summary = '\n' + '='.repeat(80) + '\n';
  summary += `${indent}Load Test Summary\n`;
  summary += '='.repeat(80) + '\n\n';
  
  // Request statistics
  const httpReqs = data.metrics.http_reqs;
  const httpReqDuration = data.metrics.http_req_duration;
  const httpReqFailed = data.metrics.http_req_failed;
  
  // Safely access metrics with fallbacks
  if (httpReqs && httpReqs.values) {
    summary += `${indent}Total Requests: ${httpReqs.values.count || 0}\n`;
    summary += `${indent}Request Rate: ${(httpReqs.values.rate || 0).toFixed(2)} req/s\n`;
  }
  
  if (httpReqFailed && httpReqFailed.values) {
    summary += `${indent}Failed Requests: ${((httpReqFailed.values.rate || 0) * 100).toFixed(2)}%\n\n`;
  }
  
  if (httpReqDuration && httpReqDuration.values) {
    const v = httpReqDuration.values;
    summary += `${indent}Response Times:\n`;
    summary += `${indent}  Min: ${(v.min != null ? v.min : 'N/A').toFixed ? v.min.toFixed(2) : v.min}ms\n`;
    summary += `${indent}  Avg: ${(v.avg != null ? v.avg : 'N/A').toFixed ? v.avg.toFixed(2) : v.avg}ms\n`;
    summary += `${indent}  Med: ${(v.med != null ? v.med : 'N/A').toFixed ? v.med.toFixed(2) : v.med}ms\n`;
    summary += `${indent}  p90: ${(v['p(90)'] != null ? v['p(90)'].toFixed(2) : 'N/A')}ms\n`;
    summary += `${indent}  p95: ${(v['p(95)'] != null ? v['p(95)'].toFixed(2) : 'N/A')}ms\n`;
    summary += `${indent}  p99: ${(v['p(99)'] != null ? v['p(99)'].toFixed(2) : 'N/A')}ms\n`;
    summary += `${indent}  Max: ${(v.max != null ? v.max : 'N/A').toFixed ? v.max.toFixed(2) : v.max}ms\n\n`;
  }
  
  // Custom metrics
  if (data.metrics.successful_requests) {
    summary += `${indent}Successful Requests: ${data.metrics.successful_requests.values.count}\n`;
  }
  if (data.metrics.failed_requests) {
    summary += `${indent}Failed Requests: ${data.metrics.failed_requests.values.count}\n`;
  }
  
  summary += '\n' + '='.repeat(80) + '\n';
  
  return summary;
}
