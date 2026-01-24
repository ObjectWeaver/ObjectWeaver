#!/bin/bash
# Automated Performance Matrix Test Runner
# Runs all test combinations and generates a summary report

set -e

# Configuration
RESULTS_DIR="integration-test/e2e/results/matrix"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
CSV_FILE="${RESULTS_DIR}/matrix_results_${TIMESTAMP}.csv"
SUMMARY_FILE="${RESULTS_DIR}/matrix_summary_${TIMESTAMP}.txt"

# Test parameters - customize these
SCHEMA_TYPES="${SCHEMA_TYPES:-minimal simple nested complex}"
LOAD_LEVELS="${LOAD_LEVELS:-50 100 200 300 500 750 1000 1500 2000}"
TEST_DURATION="${TEST_DURATION:-60s}"      # Increased from 15s for better stabilization
RAMP_UP="${RAMP_UP:-15s}"                  # Increased from 5s for gradual ramp
LLM_LATENCY="${LLM_LATENCY:-5000}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_header() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════════════════════╗"
    echo "║              ObjectWeaver Performance Matrix Test Suite                       ║"
    echo "╚══════════════════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_config() {
    echo -e "${BLUE}Configuration:${NC}"
    echo "  Schemas:      ${SCHEMA_TYPES}"
    echo "  Load Levels:  ${LOAD_LEVELS} req/s"
    echo "  Duration:     ${TEST_DURATION}"
    echo "  Ramp-up:      ${RAMP_UP}"
    echo "  LLM Latency:  ${LLM_LATENCY}ms"
    echo "  Results:      ${CSV_FILE}"
    echo ""
}

# Ensure services are running
check_services() {
    echo -e "${YELLOW}Checking services...${NC}"
    if ! curl -s http://localhost:2008/health > /dev/null 2>&1; then
        echo -e "${YELLOW}ObjectWeaver not running. Starting services...${NC}"
        cd integration-test/e2e && docker-compose -f docker-compose.integration.yml up -d --build
        echo "Waiting for services to be ready..."
        sleep 10
        
        # Retry health check
        if ! curl -s http://localhost:2008/health > /dev/null 2>&1; then
            echo -e "${RED}Failed to start ObjectWeaver. Check logs with 'make integration-test-logs'${NC}"
            exit 1
        fi
        cd ../..
    fi
    echo -e "${GREEN}✓ Services are running${NC}"
    echo ""
}

# Create results directory and CSV header
init_results() {
    mkdir -p "${RESULTS_DIR}"
    echo "schema,target_rps,llm_latency_ms,llm_rounds,theoretical_min_s,avg_s,p90_s,p95_s,achieved_rps,error_pct,overhead_ms" > "${CSV_FILE}"
}

# Run a single test
run_test() {
    local schema=$1
    local rps=$2
    
    echo -e "${YELLOW}Testing: schema=${schema}, target=${rps} req/s${NC}"
    
    # Run k6 test and capture output
    local output
    output=$(cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
        -e MAX_RPS=${rps} \
        -e DURATION=${TEST_DURATION} \
        -e RAMP_UP_TIME=${RAMP_UP} \
        -e SCHEMA_TYPE=${schema} \
        -e LLM_LATENCY_MS=${LLM_LATENCY} \
        -e BASE_URL=http://objectweaver:2008 \
        -e PASSWORD=test-password \
        k6 run /scripts/matrix-test.js 2>&1) || true
    
    # Extract CSV line
    local csv_line
    csv_line=$(echo "$output" | grep "^CSV:" | sed 's/^CSV: //' | head -1)
    
    if [ -n "$csv_line" ]; then
        echo "$csv_line" >> "${CSV_FILE}"
        
        # Parse and display key metrics
        local achieved_rps=$(echo "$csv_line" | cut -d',' -f9)
        local error_pct=$(echo "$csv_line" | cut -d',' -f10)
        local overhead=$(echo "$csv_line" | cut -d',' -f11)
        
        if (( $(echo "$error_pct > 5" | bc -l) )); then
            echo -e "  ${RED}✗ Achieved: ${achieved_rps} req/s, Errors: ${error_pct}%, Overhead: ${overhead}ms${NC}"
        else
            echo -e "  ${GREEN}✓ Achieved: ${achieved_rps} req/s, Errors: ${error_pct}%, Overhead: ${overhead}ms${NC}"
        fi
    else
        echo -e "  ${RED}✗ Test failed - no results${NC}"
        echo "${schema},${rps},${LLM_LATENCY},0,0,0,0,0,0,100,0" >> "${CSV_FILE}"
    fi
}

# Generate summary report
generate_summary() {
    echo -e "${BLUE}Generating summary report...${NC}"
    
    cat > "${SUMMARY_FILE}" << 'EOF'
╔══════════════════════════════════════════════════════════════════════════════╗
║              ObjectWeaver Performance Matrix Test Results                     ║
╚══════════════════════════════════════════════════════════════════════════════╝

EOF

    echo "Test Date: $(date)" >> "${SUMMARY_FILE}"
    echo "LLM Latency: ${LLM_LATENCY}ms" >> "${SUMMARY_FILE}"
    echo "" >> "${SUMMARY_FILE}"
    
    echo "═══════════════════════════════════════════════════════════════════════════════" >> "${SUMMARY_FILE}"
    echo "RESULTS TABLE" >> "${SUMMARY_FILE}"
    echo "═══════════════════════════════════════════════════════════════════════════════" >> "${SUMMARY_FILE}"
    echo "" >> "${SUMMARY_FILE}"
    
    # Format CSV as table
    column -t -s',' "${CSV_FILE}" >> "${SUMMARY_FILE}"
    
    echo "" >> "${SUMMARY_FILE}"
    echo "═══════════════════════════════════════════════════════════════════════════════" >> "${SUMMARY_FILE}"
    echo "ANALYSIS BY SCHEMA TYPE" >> "${SUMMARY_FILE}"
    echo "═══════════════════════════════════════════════════════════════════════════════" >> "${SUMMARY_FILE}"
    
    for schema in $SCHEMA_TYPES; do
        echo "" >> "${SUMMARY_FILE}"
        echo "--- ${schema} ---" >> "${SUMMARY_FILE}"
        grep "^${schema}," "${CSV_FILE}" | while IFS=',' read -r s rps llm rounds theo avg p90 p95 achieved err overhead; do
            printf "  %4s req/s target → %6s req/s achieved (%.1f%% efficiency), errors: %s%%, overhead: %sms\n" \
                "$rps" "$achieved" "$(echo "scale=1; $achieved * 100 / $rps" | bc 2>/dev/null || echo "0")" "$err" "$overhead" >> "${SUMMARY_FILE}"
        done
    done
    
    echo "" >> "${SUMMARY_FILE}"
    echo "═══════════════════════════════════════════════════════════════════════════════" >> "${SUMMARY_FILE}"
    echo "KEY FINDINGS" >> "${SUMMARY_FILE}"
    echo "═══════════════════════════════════════════════════════════════════════════════" >> "${SUMMARY_FILE}"
    
    # Find max throughput per schema
    for schema in $SCHEMA_TYPES; do
        max_throughput=$(grep "^${schema}," "${CSV_FILE}" | cut -d',' -f9 | sort -n | tail -1)
        echo "  ${schema}: Max achieved throughput = ${max_throughput} req/s" >> "${SUMMARY_FILE}"
    done
    
    # Find average overhead
    avg_overhead=$(tail -n +2 "${CSV_FILE}" | cut -d',' -f11 | awk '{ sum += $1; n++ } END { if (n > 0) printf "%.1f", sum/n }')
    echo "" >> "${SUMMARY_FILE}"
    echo "  Average ObjectWeaver overhead: ${avg_overhead}ms" >> "${SUMMARY_FILE}"
    
    echo "" >> "${SUMMARY_FILE}"
    echo "Raw CSV: ${CSV_FILE}" >> "${SUMMARY_FILE}"
}

# Main execution
main() {
    print_header
    print_config
    check_services
    init_results
    
    # Count total tests
    local schema_count=$(echo $SCHEMA_TYPES | wc -w | tr -d ' ')
    local load_count=$(echo $LOAD_LEVELS | wc -w | tr -d ' ')
    local total_tests=$((schema_count * load_count))
    local current=0
    
    echo -e "${BLUE}Running ${total_tests} test combinations...${NC}"
    echo ""
    
    # Run all tests
    for schema in $SCHEMA_TYPES; do
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${CYAN}Schema: ${schema}${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        
        for rps in $LOAD_LEVELS; do
            current=$((current + 1))
            echo -e "${BLUE}[${current}/${total_tests}]${NC} "
            run_test "$schema" "$rps"
            sleep 2  # Cool-down between tests
        done
        echo ""
    done
    
    generate_summary
    
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                         Matrix Test Complete!                                ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}Results:${NC}"
    echo "  CSV:     ${CSV_FILE}"
    echo "  Summary: ${SUMMARY_FILE}"
    echo ""
    echo -e "${YELLOW}Summary:${NC}"
    cat "${SUMMARY_FILE}"
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Environment variables:"
        echo "  SCHEMA_TYPES   Space-separated list of schemas (default: minimal simple nested complex)"
        echo "  LOAD_LEVELS    Space-separated list of req/s targets (default: 50 100 200 300 500 750 1000 1500 2000)"
        echo "  TEST_DURATION  Duration of each test (default: 15s)"
        echo "  RAMP_UP        Ramp-up time (default: 5s)"
        echo "  LLM_LATENCY    Mock LLM latency in ms (default: 5000)"
        echo ""
        echo "Examples:"
        echo "  $0                                    # Run full matrix"
        echo "  SCHEMA_TYPES='minimal simple' $0     # Only test minimal and simple schemas"
        echo "  LOAD_LEVELS='100 500 1000' $0        # Only test specific load levels"
        exit 0
        ;;
    *)
        main
        ;;
esac
