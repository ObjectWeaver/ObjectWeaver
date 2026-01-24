#!/bin/bash
# Goroutine monitoring script for load tests
# Monitors goroutine count and identifies top goroutine sources during test

set -e

# Configuration
OBJECTWEAVER_URL="${OBJECTWEAVER_URL:-http://localhost:2008}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
SAMPLE_INTERVAL="${SAMPLE_INTERVAL:-5}"  # seconds between samples
OUTPUT_DIR="${OUTPUT_DIR:-./integration-test/results}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Goroutine Monitor for Load Testing${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Configuration:"
echo "  ObjectWeaver: $OBJECTWEAVER_URL"
echo "  Prometheus:   $PROMETHEUS_URL"
echo "  Interval:     ${SAMPLE_INTERVAL}s"
echo "  Output:       $OUTPUT_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Timestamp for this run
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="$OUTPUT_DIR/goroutine_monitor_$TIMESTAMP.log"
PROFILE_DIR="$OUTPUT_DIR/goroutine_profiles_$TIMESTAMP"
mkdir -p "$PROFILE_DIR"

echo "Logging to: $LOG_FILE"
echo ""

# Function to get goroutine count from pprof
get_goroutine_count() {
    curl -s "$OBJECTWEAVER_URL/debug/pprof/goroutine?debug=1" 2>/dev/null | \
        grep -m1 "goroutine profile:" | \
        awk '{print $4}' || echo "0"
}

# Function to get top goroutine sources
get_goroutine_sources() {
    local output_file=$1
    curl -s "$OBJECTWEAVER_URL/debug/pprof/goroutine?debug=2" > "$output_file" 2>/dev/null
    
    # Analyze the profile
    echo "Top Goroutine Sources:" | tee -a "$LOG_FILE"
    echo "======================" | tee -a "$LOG_FILE"
    
    # Count goroutines by function
    grep -E "^[a-zA-Z]" "$output_file" | \
        grep -v "^goroutine" | \
        sort | uniq -c | sort -rn | head -20 | tee -a "$LOG_FILE"
    
    echo "" | tee -a "$LOG_FILE"
}

# Function to query Prometheus
query_prometheus() {
    local query=$1
    curl -s -G "$PROMETHEUS_URL/api/v1/query" \
        --data-urlencode "query=$query" 2>/dev/null | \
        grep -o '"value":\[[^]]*\]' | \
        grep -o '[0-9.]*"$' | \
        tr -d '"' || echo "N/A"
}

# Trap to ensure cleanup
trap 'echo -e "\n${YELLOW}Monitoring stopped${NC}"; exit 0' INT TERM

echo -e "${GREEN}Starting monitoring... (Press Ctrl+C to stop)${NC}"
echo ""
echo -e "${BLUE}Time                 Goroutines  HTTP_Conns  Workers  Memory(MB)  CPU(%)${NC}"
echo "------------------------------------------------------------------------------"

# Main monitoring loop
sample_count=0
while true; do
    sample_count=$((sample_count + 1))
    timestamp=$(date +"%H:%M:%S")
    
    # Get goroutine count directly
    goroutines=$(get_goroutine_count)
    
    # Query Prometheus metrics
    http_conns=$(query_prometheus 'go_http_client_requests_total')
    active_workers=$(query_prometheus 'objectweaver_worker_pool_active')
    memory_mb=$(query_prometheus 'go_memstats_alloc_bytes / 1024 / 1024')
    cpu_percent=$(query_prometheus 'rate(process_cpu_seconds_total[1m]) * 100')
    
    # Format and display
    printf "%-20s %-11s %-11s %-8s %-11s %-6s\n" \
        "$timestamp" \
        "$goroutines" \
        "${http_conns:-N/A}" \
        "${active_workers:-N/A}" \
        "${memory_mb:-N/A}" \
        "${cpu_percent:-N/A}" | tee -a "$LOG_FILE"
    
    # Log to file with full detail
    {
        echo "=== Sample $sample_count at $timestamp ==="
        echo "Goroutines: $goroutines"
        echo "HTTP Connections: ${http_conns:-N/A}"
        echo "Active Workers: ${active_workers:-N/A}"
        echo "Memory (MB): ${memory_mb:-N/A}"
        echo "CPU (%): ${cpu_percent:-N/A}"
        echo ""
    } >> "$LOG_FILE"
    
    # Every 30 seconds, capture detailed goroutine profile
    if [ $((sample_count % 6)) -eq 0 ]; then
        profile_file="$PROFILE_DIR/profile_${timestamp//:/-}.txt"
        echo "" | tee -a "$LOG_FILE"
        echo -e "${YELLOW}=== Capturing detailed profile at $timestamp ===${NC}" | tee -a "$LOG_FILE"
        get_goroutine_sources "$profile_file"
        echo "" | tee -a "$LOG_FILE"
    fi
    
    # Check for goroutine spike
    if [ "$goroutines" -gt 50000 ]; then
        echo -e "${RED}⚠️  WARNING: Goroutine count exceeded 50,000!${NC}" | tee -a "$LOG_FILE"
        # Capture emergency profile
        emergency_profile="$PROFILE_DIR/SPIKE_${timestamp//:/-}.txt"
        get_goroutine_sources "$emergency_profile"
    fi
    
    sleep "$SAMPLE_INTERVAL"
done
