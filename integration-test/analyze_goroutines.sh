#!/bin/bash
# Analyze goroutine monitoring results

set -e

RESULTS_DIR="${1:-./integration-test/results}"

echo "╔════════════════════════════════════════════╗"
echo "║  Goroutine Analysis Report                 ║"
echo "╚════════════════════════════════════════════╝"
echo ""

# Find latest log file
LATEST_LOG=$(ls -t "$RESULTS_DIR"/goroutine_monitor_*.log 2>/dev/null | head -1)
LATEST_PROFILE_DIR=$(ls -td "$RESULTS_DIR"/goroutine_profiles_* 2>/dev/null | head -1)

if [ -z "$LATEST_LOG" ]; then
    echo "❌ No monitoring logs found in $RESULTS_DIR"
    exit 1
fi

echo "📁 Analyzing: $LATEST_LOG"
echo ""

# Extract goroutine stats
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 GOROUTINE STATISTICS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Get min, max, avg goroutines
goroutine_counts=$(grep "Goroutines:" "$LATEST_LOG" | awk '{print $2}' | grep -E '^[0-9]+$')
if [ -n "$goroutine_counts" ]; then
    min=$(echo "$goroutine_counts" | sort -n | head -1)
    max=$(echo "$goroutine_counts" | sort -n | tail -1)
    avg=$(echo "$goroutine_counts" | awk '{sum+=$1; count++} END {if(count>0) print int(sum/count); else print 0}')
    
    echo "Minimum:     $min goroutines"
    echo "Maximum:     $max goroutines"
    echo "Average:     $avg goroutines"
    echo "Peak Spike:  +$((max - min)) goroutines"
    echo ""
    
    # Check for concerning patterns
    if [ "$max" -gt 50000 ]; then
        echo "⚠️  CRITICAL: Peak exceeded 50,000 goroutines!"
    elif [ "$max" -gt 30000 ]; then
        echo "⚠️  WARNING: Peak exceeded 30,000 goroutines"
    else
        echo "✅ Goroutine count within acceptable range"
    fi
    echo ""
fi

# Timeline analysis
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📈 TIMELINE (Last 20 samples)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
grep -E "^[0-9]{2}:[0-9]{2}:[0-9]{2}" "$LATEST_LOG" | tail -20
echo ""

# Analyze spike patterns
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 SPIKE DETECTION"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Find sudden increases (>10k goroutines between samples)
prev_count=""
spike_detected=false
while IFS= read -r line; do
    if [[ $line =~ Goroutines:\ ([0-9]+) ]]; then
        current_count="${BASH_REMATCH[1]}"
        if [ -n "$prev_count" ]; then
            diff=$((current_count - prev_count))
            if [ "$diff" -gt 10000 ]; then
                echo "⚠️  Spike detected: +$diff goroutines"
                spike_detected=true
            fi
        fi
        prev_count=$current_count
    fi
done < "$LATEST_LOG"

if [ "$spike_detected" = false ]; then
    echo "✅ No sudden spikes detected (threshold: 10k goroutines)"
fi
echo ""

# Analyze goroutine sources from profiles
if [ -n "$LATEST_PROFILE_DIR" ] && [ -d "$LATEST_PROFILE_DIR" ]; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔬 TOP GOROUTINE SOURCES (from profiles)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Aggregate all profiles
    cat "$LATEST_PROFILE_DIR"/*.txt 2>/dev/null | \
        grep -E "^(net/http|objectweaver|runtime|encoding|google)" | \
        grep -v "^goroutine" | \
        sort | uniq -c | sort -rn | head -30
    echo ""
    
    # Check for spike profiles
    if ls "$LATEST_PROFILE_DIR"/SPIKE_*.txt >/dev/null 2>&1; then
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "🚨 SPIKE PROFILE ANALYSIS"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "Critical spike detected during test! Analyzing..."
        echo ""
        
        latest_spike=$(ls -t "$LATEST_PROFILE_DIR"/SPIKE_*.txt | head -1)
        echo "Spike profile: $(basename "$latest_spike")"
        echo ""
        echo "Top sources during spike:"
        grep -E "^(net/http|objectweaver|runtime)" "$latest_spike" | \
            sort | uniq -c | sort -rn | head -20
        echo ""
    fi
fi

# Memory analysis
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💾 RESOURCE USAGE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
memory_values=$(grep "Memory (MB):" "$LATEST_LOG" | grep -oE '[0-9.]+' | head -20)
if [ -n "$memory_values" ]; then
    mem_max=$(echo "$memory_values" | sort -n | tail -1)
    mem_min=$(echo "$memory_values" | sort -n | head -1)
    echo "Memory Range: ${mem_min} MB - ${mem_max} MB"
fi
echo ""

# Recommendations
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💡 RECOMMENDATIONS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ "$max" -gt 40000 ]; then
    echo "❌ Goroutine count is too high. Check:"
    echo "   1. HTTP connection pool settings (MaxConnsPerHost)"
    echo "   2. IdleConnTimeout (currently should be 90s)"
    echo "   3. MaxIdleConnsPerHost (should be ~80% of MaxConnsPerHost)"
    echo ""
    echo "   Environment variables to tune:"
    echo "   export LLM_HTTP_MAX_CONNS_PER_HOST=5000"
    echo "   export LLM_HTTP_MAX_IDLE_CONNS_PER_HOST=4000"
    echo "   export LLM_HTTP_IDLE_CONN_TIMEOUT_SECONDS=120"
elif [ "$max" -gt 30000 ]; then
    echo "⚠️  Goroutine count is elevated. Monitor for:"
    echo "   - Connection pool churn (check IdleConnTimeout)"
    echo "   - Ensure MaxIdleConnsPerHost >= 80% of MaxConnsPerHost"
else
    echo "✅ Goroutine count looks healthy!"
    echo "   Current settings appear well-tuned for this load"
fi
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📂 Full logs available at:"
echo "   $LATEST_LOG"
if [ -n "$LATEST_PROFILE_DIR" ]; then
    echo "   $LATEST_PROFILE_DIR/"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
