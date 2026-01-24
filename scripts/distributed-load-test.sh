#!/bin/bash
# Distributed K6 Load Test Runner
# Runs multiple k6 instances in parallel to generate ultra-high load

set -e

# Disable Docker Content Trust for k6 image
export DOCKER_CONTENT_TRUST=0

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
RESULTS_DIR="${PROJECT_ROOT}/integration-test/e2e/results/distributed"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="${RESULTS_DIR}/distributed_${TIMESTAMP}.txt"

# Parse command line arguments
if [ $# -ge 1 ]; then
    NUM_K6_INSTANCES=$1
fi
if [ $# -ge 2 ]; then
    SCHEMA=$2
fi
if [ $# -ge 3 ]; then
    RPS_PER_INSTANCE=$3
fi

# Test parameters (with defaults)
NUM_K6_INSTANCES=${NUM_K6_INSTANCES:-5}
RPS_PER_INSTANCE=${RPS_PER_INSTANCE:-10000}
TOTAL_TARGET_RPS=$((NUM_K6_INSTANCES * RPS_PER_INSTANCE))
SCHEMA=${SCHEMA:-minimal}
DURATION=${DURATION:-30s}
WORKER_POOL=${WORKER_POOL_SIZE:-50000}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔══════════════════════════════════════════════════════════════════════════════╗"
echo "║              Distributed K6 Ultra-High Load Test                              ║"
echo "╚══════════════════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

echo -e "${BLUE}Configuration:${NC}"
echo "  K6 Instances:     ${NUM_K6_INSTANCES}"
echo "  RPS per Instance: ${RPS_PER_INSTANCE}"
echo "  Total Target RPS: ${TOTAL_TARGET_RPS}"
echo "  Schema:           ${SCHEMA}"
echo "  Duration:         ${DURATION}"
echo "  Worker Pool:      ${WORKER_POOL}"
echo ""

mkdir -p "${RESULTS_DIR}"

# Change to integration-test/e2e directory
E2E_DIR="${PROJECT_ROOT}/integration-test/e2e"
cd "${E2E_DIR}"

# Start infrastructure with higher resource allocation
echo -e "${YELLOW}Starting infrastructure with distributed k6...${NC}"

# Stop any existing containers
docker-compose -f docker-compose.distributed.yml down --remove-orphans 2>/dev/null || true

# Start services with high worker pool
WORKER_POOL_SIZE=${WORKER_POOL} \
LLM_CONCURRENCY=$((WORKER_POOL * 2)) \
docker-compose -f docker-compose.distributed.yml up -d mock-llm-server objectweaver prometheus

echo "Waiting for services to be ready..."
sleep 10

# Verify ObjectWeaver is running
if ! curl -s http://localhost:2008/health > /dev/null 2>&1; then
    echo -e "${RED}ObjectWeaver failed to start!${NC}"
    docker-compose -f docker-compose.distributed.yml logs objectweaver
    exit 1
fi
echo -e "${GREEN}✓ ObjectWeaver is ready${NC}"

# Run distributed k6 instances
echo ""
echo -e "${YELLOW}Launching ${NUM_K6_INSTANCES} k6 instances, each targeting ${RPS_PER_INSTANCE} req/s...${NC}"
echo -e "${YELLOW}Total target: ${TOTAL_TARGET_RPS} req/s${NC}"
echo ""

# Create a combined results file
echo "Distributed Load Test Results - ${TIMESTAMP}" > "${RESULTS_FILE}"
echo "=============================================" >> "${RESULTS_FILE}"
echo "K6 Instances: ${NUM_K6_INSTANCES}" >> "${RESULTS_FILE}"
echo "RPS per Instance: ${RPS_PER_INSTANCE}" >> "${RESULTS_FILE}"
echo "Total Target RPS: ${TOTAL_TARGET_RPS}" >> "${RESULTS_FILE}"
echo "Schema: ${SCHEMA}" >> "${RESULTS_FILE}"
echo "Worker Pool: ${WORKER_POOL}" >> "${RESULTS_FILE}"
echo "" >> "${RESULTS_FILE}"

# Run k6 instances using docker run (not docker-compose run)
PIDS=()
for i in $(seq 1 ${NUM_K6_INSTANCES}); do
    echo -e "${BLUE}Starting k6 instance ${i}...${NC}"
    
    docker run --rm \
        --network e2e_loadtest \
        --name "k6-parallel-${i}-${TIMESTAMP}" \
        -v "${E2E_DIR}/k6:/scripts:ro" \
        -e MAX_RPS=${RPS_PER_INSTANCE} \
        -e DURATION=${DURATION} \
        -e RAMP_UP_TIME=15s \
        -e SCHEMA_TYPE=${SCHEMA} \
        -e LLM_LATENCY_MS=5000 \
        -e K6_INSTANCE=${i} \
        -e BASE_URL=http://objectweaver-distributed:2008 \
        -e PASSWORD=test-password \
        grafana/k6:latest run /scripts/matrix-test.js > "/tmp/k6_instance_${i}.log" 2>&1 &
    
    PIDS+=($!)
done

echo ""
echo -e "${YELLOW}All ${NUM_K6_INSTANCES} k6 instances started. Waiting for completion...${NC}"
echo ""

# Wait for all instances to complete
FAILED=0
for i in "${!PIDS[@]}"; do
    instance=$((i + 1))
    if wait ${PIDS[$i]}; then
        echo -e "${GREEN}✓ Instance ${instance} completed${NC}"
    else
        echo -e "${RED}✗ Instance ${instance} failed${NC}"
        FAILED=$((FAILED + 1))
    fi
done

# Collect and aggregate results
echo "" >> "${RESULTS_FILE}"
echo "Individual Instance Results:" >> "${RESULTS_FILE}"
echo "============================" >> "${RESULTS_FILE}"

TOTAL_ACHIEVED_RPS=0
TOTAL_REQUESTS=0
TOTAL_ERRORS=0

for i in $(seq 1 ${NUM_K6_INSTANCES}); do
    echo "" >> "${RESULTS_FILE}"
    echo "--- Instance ${i} ---" >> "${RESULTS_FILE}"
    
    if [ -f "/tmp/k6_instance_${i}.log" ]; then
        # Extract key metrics
        CSV_LINE=$(grep "^CSV:" "/tmp/k6_instance_${i}.log" | sed 's/^CSV: //' | head -1)
        if [ -n "$CSV_LINE" ]; then
            ACHIEVED=$(echo "$CSV_LINE" | cut -d',' -f9)
            ERRORS=$(echo "$CSV_LINE" | cut -d',' -f10)
            
            echo "Achieved RPS: ${ACHIEVED}" >> "${RESULTS_FILE}"
            echo "Error %: ${ERRORS}" >> "${RESULTS_FILE}"
            
            TOTAL_ACHIEVED_RPS=$(echo "${TOTAL_ACHIEVED_RPS} + ${ACHIEVED}" | bc 2>/dev/null || echo "0")
        fi
        
        # Copy summary section
        grep -A 20 "MATRIX TEST RESULTS" "/tmp/k6_instance_${i}.log" >> "${RESULTS_FILE}" 2>/dev/null || true
    fi
done

echo "" >> "${RESULTS_FILE}"
echo "=========================================" >> "${RESULTS_FILE}"
echo "AGGREGATE RESULTS" >> "${RESULTS_FILE}"
echo "=========================================" >> "${RESULTS_FILE}"
echo "Total Target RPS:   ${TOTAL_TARGET_RPS}" >> "${RESULTS_FILE}"
echo "Total Achieved RPS: ${TOTAL_ACHIEVED_RPS}" >> "${RESULTS_FILE}"
echo "Failed Instances:   ${FAILED}" >> "${RESULTS_FILE}"

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                    Distributed Test Complete!                                ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Results:${NC}"
echo "  Total Target RPS:   ${TOTAL_TARGET_RPS}"
echo "  Total Achieved RPS: ${TOTAL_ACHIEVED_RPS}"
echo "  Failed Instances:   ${FAILED}/${NUM_K6_INSTANCES}"
echo ""
echo "  Full results: ${RESULTS_FILE}"
echo ""

cat "${RESULTS_FILE}"
