# ObjectWeaver Integration Testing

Load testing with Docker Compose, mock LLM server, and Prometheus monitoring.

## Quick Start

```bash
# Start all services
make integration-test-up

# Run load test (choose one)
make integration-test-light    # 100 req/s
make integration-test-medium   # 500 req/s
make integration-test-heavy    # 1000 req/s
make integration-test-extreme  # 2000 req/s
make integration-test-2k       # 2000 req/s
make integration-test-3k       # 3000 req/s
make integration-test-4k       # 4000 req/s
make integration-test-5k 
make integration-test-10k

# Run Redis queued-cache integration test
make integration-test-cache

# Separate load tests (2k / 3k / 4k req/s)
make integration-test-2k
make integration-test-3k
make integration-test-4k

# View results
make integration-test-results

# Stop services
make integration-test-down
```

## Redis Queued-Cache Test Suite

The queued endpoints now store results in Redis and serve them via
`/api/getObjectQueued`. This test validates that:

1. Queue submission is fast (enqueue latency).
2. Results are retrieved successfully from Redis.
3. End-to-end completion time matches expected LLM latency.

Defaults (override with environment variables):

```bash
QUEUED_RPS=50
DURATION=60s
RAMP_UP_TIME=15s
POLL_INTERVAL_MS=200
POLL_TIMEOUT_MS=120000
```

## Test Results

Integration tests must be run and passed before submitting a PR. These tests have been crucial for identifying goroutine leaks and other performance-degrading bugs.

**Test Environment:**
- Device: MacBook Pro
- CPU: M1 Max
- RAM: 64GB

**Results:**

| Test | Status | Notes |
|------|--------|-------|
| `make integration-test-light` | Pass | 100 req/s |
| `make integration-test-medium` | Pass | 500 req/s |
| `make integration-test-heavy` | Pass | 1000 req/s |
| `make integration-test-extreme` | Pass | 2000 req/s |
| `make integration-test-5k` | Fail | Response times start slowing |
| `make integration-test-10k` | Fail | Just Breaks it! |

**Note:** For the 5k and 10k test whilst the server is able to handle the load in part it starts to take a toll on the response times. With the 5k test leading to around a 7 second response time adn the 10k test can take up to 30 seconds. 

---

## Monitoring

**Prometheus UI**: http://localhost:9090  
**Grafana**: http://localhost:3000 (admin/admin)  
**Metrics Endpoint**: http://localhost:2008/metrics

### Key Metrics

- `http_requests_total` - Total requests
- `http_request_duration_seconds` - Latency
- `active_requests` - Concurrent requests
- `goroutines` - Goroutine count
- `memory_usage_bytes` - Memory usage

## Troubleshooting

```bash
# Check service health
curl http://localhost:2008/health
curl http://localhost:8080/health

# View logs
cd integration-test
docker-compose -f docker-compose.integration.yml logs

# Rebuild containers
docker-compose -f docker-compose.integration.yml up -d --build
```

## Discliamer 

All the files in the integration-test folder have been completely creating using "AI"/LLMs as CBA to create a new server and all myself for an integration test. However, it prove essentail as before this testing started the ObjectWeaver server was hot garbage in handling high traffic levels. The tests revealed bottlenecks and a lot of poor code I had and quite a few bugs! 