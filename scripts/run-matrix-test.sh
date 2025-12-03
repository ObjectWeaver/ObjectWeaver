#!/bin/bash
# Performance Matrix Test Runner
# Tests various combinations of load levels and schema complexity

set -e

# Configuration
RESULTS_DIR="integration-test/e2e/results/matrix"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="${RESULTS_DIR}/matrix_${TIMESTAMP}.csv"

# Test parameters
SCHEMA_TYPES=("minimal" "simple" "nested" "complex")
LOAD_LEVELS=(50 100 200 500 1000)  # requests per second

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "╔══════════════════════════════════════════════════════════════════════════════╗"
echo "║                    ObjectWeaver Performance Matrix Test                       ║"
echo "╚══════════════════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Create results directory
mkdir -p "${RESULTS_DIR}"

# Write CSV header
echo "schema_type,target_rps,llm_latency_ms,est_llm_calls,theoretical_min_s,p50_s,p95_s,p99_s,throughput_rps,error_pct" > "${RESULTS_FILE}"

# Check if services are running
echo -e "${YELLOW}Checking services...${NC}"
if ! curl -s http://localhost:2008/health > /dev/null 2>&1; then
    echo -e "${RED}ObjectWeaver not running. Starting services...${NC}"
    make integration-test-up
    sleep 5
fi
echo -e "${GREEN}✓ Services are running${NC}"

# Get LLM latency from docker-compose
LLM_LATENCY=$(docker exec mock-llm-server printenv MOCK_LLM_MIN_DELAY_MS 2>/dev/null || echo "5000")
echo -e "${BLUE}LLM Latency: ${LLM_LATENCY}ms${NC}"
echo ""

# Total tests
TOTAL_TESTS=$((${#SCHEMA_TYPES[@]} * ${#LOAD_LEVELS[@]}))
CURRENT_TEST=0

echo -e "${BLUE}Running ${TOTAL_TESTS} test combinations...${NC}"
echo ""

# Run matrix tests
for SCHEMA in "${SCHEMA_TYPES[@]}"; do
    for RPS in "${LOAD_LEVELS[@]}"; do
        CURRENT_TEST=$((CURRENT_TEST + 1))
        
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${YELLOW}Test ${CURRENT_TEST}/${TOTAL_TESTS}: Schema=${SCHEMA}, Target=${RPS} req/s${NC}"
        echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        
        # Calculate appropriate test duration based on RPS
        if [ $RPS -le 100 ]; then
            DURATION="30s"
            RAMP_UP="10s"
        elif [ $RPS -le 500 ]; then
            DURATION="45s"
            RAMP_UP="20s"
        else
            DURATION="60s"
            RAMP_UP="30s"
        fi
        
        # Run the test
        cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
            -e MAX_RPS=${RPS} \
            -e DURATION=${DURATION} \
            -e RAMP_UP_TIME=${RAMP_UP} \
            -e SCHEMA_TYPE=${SCHEMA} \
            -e LLM_LATENCY_MS=${LLM_LATENCY} \
            -e BASE_URL=http://objectweaver:2008 \
            -e PASSWORD=test-password \
            k6 run /scripts/matrix-test.js 2>&1 | tee /tmp/k6_output.txt
        
        cd ../..
        
        # Extract CSV line from output and append to results
        CSV_LINE=$(grep "^CSV:" /tmp/k6_output.txt | sed 's/^CSV: //' || echo "${SCHEMA},${RPS},${LLM_LATENCY},0,0,0,0,0,0,100")
        echo "${CSV_LINE}" >> "${RESULTS_FILE}"
        
        # Brief pause between tests to let system settle
        echo -e "${BLUE}Cooling down (5s)...${NC}"
        sleep 5
    done
done

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                         Matrix Test Complete!                                ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Results saved to: ${RESULTS_FILE}${NC}"
echo ""

# Display summary table
echo -e "${YELLOW}RESULTS SUMMARY:${NC}"
echo ""
column -t -s',' "${RESULTS_FILE}" | head -1
echo "────────────────────────────────────────────────────────────────────────────────────────────────"
column -t -s',' "${RESULTS_FILE}" | tail -n +2

echo ""
echo -e "${BLUE}To analyze results:${NC}"
echo "  cat ${RESULTS_FILE}"
echo "  # Or import into spreadsheet for graphing"
