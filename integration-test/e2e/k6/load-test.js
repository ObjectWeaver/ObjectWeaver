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
export const options = {
  scenarios: {
    ramping_load: {
      executor: 'ramping-arrival-rate',
      startRate: 1,                    // Start with 1 req/sec
      timeUnit: '1s',
      preAllocatedVUs: 50,             // Pre-allocate VUs
      maxVUs: parseInt(MAX_RPS) * 2,   // Max VUs based on target RPS
      stages: [
        { duration: RAMP_UP_TIME, target: parseInt(MAX_RPS) },  // Ramp up to MAX_RPS
        { duration: DURATION, target: parseInt(MAX_RPS) },      // Hold at MAX_RPS
        { duration: '10s', target: 0 },                         // Ramp down
      ],
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],  // 95% < 500ms, 99% < 1s
    'http_req_failed': ['rate<0.05'],                   // Error rate < 5%
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
function getRandomSchema() {
  const rand = Math.random();
  if (rand < 0.5) {
    return { name: 'simple', schema: schemas.simple };
  } else if (rand < 0.8) {
    return { name: 'nested', schema: schemas.nested };
  } else {
    return { name: 'complex', schema: schemas.complex };
  }
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
    timeout: '30s',
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

  check(response, {
    'response time < 1000ms': () => duration < 1000,
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
  
  summary += `${indent}Total Requests: ${httpReqs.values.count}\n`;
  summary += `${indent}Request Rate: ${httpReqs.values.rate.toFixed(2)} req/s\n`;
  summary += `${indent}Failed Requests: ${(httpReqFailed.values.rate * 100).toFixed(2)}%\n\n`;
  
  summary += `${indent}Response Times:\n`;
  summary += `${indent}  Min: ${httpReqDuration.values.min.toFixed(2)}ms\n`;
  summary += `${indent}  Avg: ${httpReqDuration.values.avg.toFixed(2)}ms\n`;
  summary += `${indent}  Med: ${httpReqDuration.values.med.toFixed(2)}ms\n`;
  summary += `${indent}  p90: ${httpReqDuration.values['p(90)'].toFixed(2)}ms\n`;
  summary += `${indent}  p95: ${httpReqDuration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `${indent}  p99: ${httpReqDuration.values['p(99)'].toFixed(2)}ms\n`;
  summary += `${indent}  Max: ${httpReqDuration.values.max.toFixed(2)}ms\n\n`;
  
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
