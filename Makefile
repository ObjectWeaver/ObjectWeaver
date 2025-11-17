# ObjectWeaver Makefile
# Testing, benchmarking, and deployment utilities

.PHONY: help test lint fmt build clean-profiles benchmark-server benchmark-server-concurrency benchmark-server-throughput benchmark-server-latency benchmark-server-memory benchmark-server-profile-cpu benchmark-server-profile-mem benchmark-server-stress benchmark-server-fast benchmark-server-compare

# Default target
help:
	@echo "ObjectWeaver Commands"
	@echo "====================="
	@echo ""
	@echo "Development:"
	@echo "  make test              - Run all tests"
	@echo "  make lint              - Run linters (vet, fmt)"
	@echo "  make fmt               - Format code"
	@echo "  make build             - Build the project"
	@echo ""
	@echo "Server Benchmarks (with Mock LLM):"
	@echo "  make benchmark-server             - Run all server benchmarks"
	@echo "  make benchmark-server-concurrency - Test concurrent request handling"
	@echo "  make benchmark-server-throughput  - Measure maximum throughput"
	@echo "  make benchmark-server-latency     - Measure request latency"
	@echo "  make benchmark-server-memory      - Test memory usage under load"
	@echo "  make benchmark-server-stress      - Stress test with 1000+ concurrent requests"
	@echo "  make benchmark-server-fast        - Quick server benchmark"
	@echo "  make benchmark-server-compare     - Save benchmarks for comparison"
	@echo "  make benchmark-server-profile-cpu - Server CPU profiling"
	@echo "  make benchmark-server-profile-mem - Server memory profiling"
	@echo ""
	@echo "Integration Testing (Real Infrastructure):"
	@echo "  make integration-test-help   - Show all integration test commands"
	@echo "  make integration-test-full   - Run complete integration test cycle"
	@echo "  make integration-test-up     - Start services (ObjectWeaver + Prometheus)"
	@echo "  make integration-test-run    - Run load test (1000 req/s)"
	@echo "  make integration-test-5k     - Run very heavy load test (5000 req/s) with monitoring"
	@echo "  make integration-test-10k    - Run ultra load test (10000 req/s)"
	@echo "  make monitor-goroutines      - Monitor goroutines during tests (standalone)"
	@echo "  make integration-test-down   - Stop all services"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean-profiles    - Remove all profile files"
	@echo ""

# Run all tests
test:
	@echo "Running all tests..."
	@go test -v ./...

# Lint and vet
lint:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Running go fmt check..."
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# Format code
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

# Build the project
build:
	@echo "Building ObjectWeaver..."
	@go build -v ./...

# Clean generated files
clean-profiles:
	@echo "Cleaning profile files..."
	@rm -rf profiles/
	@rm -f benchmark-*.txt
	@echo "Done!"

# ============================================
# SERVER BENCHMARKS WITH MOCK LLM RESPONSES
# ============================================

# Run all server benchmarks
benchmark-server:
	@echo "Running complete server benchmark suite..."
	@echo "=========================================="
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkServer -benchmem -benchtime=3s ./service/ | tee benchmark-server-results.txt
	@echo ""
	@echo "Results saved to benchmark-server-results.txt"

# Test concurrent request handling
benchmark-server-concurrency:
	@echo "Testing server concurrency handling..."
	@echo "======================================"
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkServerConcurrency -benchmem -benchtime=3s ./service/ | tee benchmark-server-concurrency-results.txt
	@echo ""
	@echo "Results saved to benchmark-server-concurrency-results.txt"

# Measure maximum throughput
benchmark-server-throughput:
	@echo "Measuring server throughput..."
	@echo "=============================="
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkServerThroughput -benchmem -benchtime=10s ./service/ | tee benchmark-server-throughput-results.txt
	@echo ""
	@echo "Results saved to benchmark-server-throughput-results.txt"

# Measure request latency
benchmark-server-latency:
	@echo "Measuring server latency..."
	@echo "==========================="
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkServerLatency -benchmem -benchtime=5s ./service/ | tee benchmark-server-latency-results.txt
	@echo ""
	@echo "Results saved to benchmark-server-latency-results.txt"

# Test memory usage under load
benchmark-server-memory:
	@echo "Testing server memory usage under load..."
	@echo "=========================================="
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkServerMemoryPressure -benchmem -benchtime=3s ./service/ | tee benchmark-server-memory-results.txt
	@echo ""
	@echo "Results saved to benchmark-server-memory-results.txt"

# Stress test with 1000+ concurrent requests
benchmark-server-stress:
	@echo "Running server stress test (1000+ concurrent)..."
	@echo "================================================="
	@echo ""
	@ENVIRONMENT=development go test -bench="BenchmarkServerConcurrency/1000_Concurrent" -benchmem -benchtime=5s ./service/ | tee benchmark-server-stress-results.txt
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkEndToEnd/Realistic_StressTest -benchmem -benchtime=3s ./service/ | tee -a benchmark-server-stress-results.txt
	@echo ""
	@echo "Results saved to benchmark-server-stress-results.txt"

# Server CPU profiling
benchmark-server-profile-cpu:
	@echo "Running server benchmarks with CPU profiling..."
	@echo "================================================"
	@echo ""
	@mkdir -p profiles
	@echo "Server Concurrency CPU Profile..."
	@ENVIRONMENT=development go test -bench=BenchmarkServerConcurrency/100_Concurrent_Simple -cpuprofile=profiles/server-concurrency-cpu.prof -benchtime=5s ./service/
	@echo ""
	@echo "Server Throughput CPU Profile..."
	@ENVIRONMENT=development go test -bench=BenchmarkServerThroughput -cpuprofile=profiles/server-throughput-cpu.prof -benchtime=5s ./service/
	@echo ""
	@echo "Server End-to-End CPU Profile..."
	@ENVIRONMENT=development go test -bench=BenchmarkEndToEnd/Realistic_HighLoad -cpuprofile=profiles/server-e2e-cpu.prof -benchtime=5s ./service/
	@echo ""
	@echo "CPU profiles saved to profiles/ directory"
	@echo ""
	@echo "Analyze with:"
	@echo "  go tool pprof profiles/server-concurrency-cpu.prof"
	@echo "  go tool pprof -http=:8080 profiles/server-concurrency-cpu.prof"

# Server memory profiling
benchmark-server-profile-mem:
	@echo "Running server benchmarks with memory profiling..."
	@echo "=================================================="
	@echo ""
	@mkdir -p profiles
	@echo "Server Concurrency Memory Profile..."
	@ENVIRONMENT=development go test -bench=BenchmarkServerConcurrency/100_Concurrent_Simple -memprofile=profiles/server-concurrency-mem.prof -benchmem -benchtime=5s ./service/
	@echo ""
	@echo "Server Memory Pressure Profile..."
	@ENVIRONMENT=development go test -bench=BenchmarkServerMemoryPressure/Memory_1000_Concurrent -memprofile=profiles/server-memory-pressure-mem.prof -benchmem -benchtime=3s ./service/
	@echo ""
	@echo "Server End-to-End Memory Profile..."
	@ENVIRONMENT=development go test -bench=BenchmarkEndToEnd/Realistic_HighLoad -memprofile=profiles/server-e2e-mem.prof -benchmem -benchtime=5s ./service/
	@echo ""
	@echo "Memory profiles saved to profiles/ directory"
	@echo ""
	@echo "Analyze with:"
	@echo "  go tool pprof -alloc_space profiles/server-concurrency-mem.prof"
	@echo "  go tool pprof -http=:8080 profiles/server-concurrency-mem.prof"

# Quick server benchmark
benchmark-server-fast:
	@echo "Running quick server benchmarks..."
	@echo "==================================="
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkServerConcurrency/100_Concurrent_Simple -benchmem -benchtime=1s ./service/
	@echo ""
	@ENVIRONMENT=development go test -bench=BenchmarkServerThroughput -benchmem -benchtime=1s ./service/

# Compare server performance over time
benchmark-server-compare:
	@echo "Running server benchmarks for comparison..."
	@echo "==========================================="
	@echo ""
	@mkdir -p benchmarks
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	echo "Saving results to benchmarks/server-benchmark-$$TIMESTAMP.txt"; \
	ENVIRONMENT=development go test -bench=BenchmarkServer -benchmem -benchtime=5s -count=5 ./service/ > benchmarks/server-benchmark-$$TIMESTAMP.txt
	@echo ""
	@echo "Compare with previous run using:"
	@echo "  benchstat benchmarks/server-benchmark-<old>.txt benchmarks/server-benchmark-<new>.txt"

# ============================================================================
# Integration Testing Targets
# ============================================================================

.PHONY: integration-test-help integration-test-up integration-test-down integration-test-run integration-test-clean integration-test-logs integration-test-metrics integration-test-full integration-test-10k

# Integration test help
integration-test-help:
	@echo "Integration Testing Commands"
	@echo "============================"
	@echo ""
	@echo "Setup:"
	@echo "  make integration-test-up    - Start all services (ObjectWeaver + Mock LLM + Prometheus)"
	@echo ""
	@echo "Running Tests:"
	@echo "  make integration-test-run   - Run load test (ramping to 1000 req/s over 60s)"
	@echo "  make integration-test-full  - Full test: up + run + results + down"
	@echo ""
	@echo "Custom Load Tests:"
	@echo "  make integration-test-light     - Light load: ramp to 100 req/s"
	@echo "  make integration-test-medium    - Medium load: ramp to 500 req/s"
	@echo "  make integration-test-heavy     - Heavy load: ramp to 1000 req/s"
	@echo "  make integration-test-5k        - Very heavy load: ramp to 5000 req/s"
	@echo "  make integration-test-extreme   - Extreme load: ramp to 2000 req/s"
	@echo "  make integration-test-10k       - Ultra load: ramp to 10000 req/s"
	@echo "  make integration-test-25k       - Massive load: ramp to 25000 req/s"
	@echo ""
	@echo "Monitoring:"
	@echo "  make integration-test-logs      - Follow logs from all services"
	@echo "  make integration-test-metrics   - Open Prometheus UI"
	@echo "  make integration-test-grafana   - Open Grafana UI"
	@echo "  make integration-test-results   - Display last test results"
	@echo ""
	@echo "Cleanup:"
	@echo "  make integration-test-down      - Stop all services"
	@echo "  make integration-test-clean     - Stop services and clean volumes"
	@echo ""
	@echo "Access URLs (once started):"
	@echo "  ObjectWeaver API:    http://localhost:2008"
	@echo "  ObjectWeaver Metrics: http://localhost:2008/metrics"
	@echo "  Mock LLM:            http://localhost:8080"
	@echo "  Prometheus:          http://localhost:9090"
	@echo "  Grafana:             http://localhost:3000 (admin/admin)"
	@echo ""

# Start all integration test services
integration-test-up:
	@echo "Starting integration test environment..."
	@echo "========================================"
	@echo ""
	@echo "Services starting:"
	@echo "  - Mock LLM Server (port 8080)"
	@echo "  - ObjectWeaver Server (port 2008)"
	@echo "  - Prometheus (port 9090)"
	@echo "  - Grafana (port 3000)"
	@echo ""
	cd integration-test && docker-compose -f docker-compose.integration.yml up -d --build
	@echo ""
	@echo "Waiting for services to be ready..."
	@echo "(Mock LLM has health checks, ObjectWeaver needs time to start)"
	@sleep 5
	@echo ""
	@echo "Testing ObjectWeaver health..."
	@curl -s http://localhost:2008/health > /dev/null || (echo "❌ ObjectWeaver not responding yet, waiting 2 more seconds..." && sleep 2)
	@curl -s http://localhost:2008/health > /dev/null && echo "✓ ObjectWeaver is healthy!" || echo "⚠️  ObjectWeaver may still be starting..."
	@echo ""
	@echo "✓ Services are running!"
	@echo ""
	@echo "Access points:"
	@echo "  ObjectWeaver API:    http://localhost:2008"
	@echo "  ObjectWeaver Health: http://localhost:2008/health"
	@echo "  ObjectWeaver Metrics: http://localhost:2008/metrics"
	@echo "  Mock LLM:            http://localhost:8080/stats"
	@echo "  Prometheus:          http://localhost:9090"
	@echo "  Grafana:             http://localhost:3000 (admin/admin)"
	@echo ""

# Stop all integration test services
integration-test-down:
	@echo "Stopping integration test environment..."
	cd integration-test && docker-compose -f docker-compose.integration.yml down
	@echo "✓ Services stopped"

# Stop and clean volumes
integration-test-clean:
	@echo "Cleaning integration test environment..."
	cd integration-test && docker-compose -f docker-compose.integration.yml down -v
	@rm -rf integration-test/results/*
	@echo "✓ Services stopped and volumes cleaned"

# Run load test with default settings (1000 req/s)
integration-test-run:
	@echo "Running integration load test..."
	@echo "================================"
	@echo ""
	@echo "Configuration:"
	@echo "  - Ramping from 1 to 1000 req/s"
	@echo "  - Duration: 60 seconds"
	@echo "  - Ramp-up time: 30 seconds"
	@echo ""
	@mkdir -p integration-test/results
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=1000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=30s \
		-e BASE_URL=http://objectweaver:2008 \
		-e PASSWORD=test-password \
		k6 run /scripts/load-test.js
	@echo ""
	@echo "✓ Load test completed!"
	@echo "Results saved to: integration-test/results/"

# Light load test (100 req/s)
integration-test-light:
	@echo "Running LIGHT load test (100 req/s)..."
	@mkdir -p integration-test/results
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=100 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=20s \
		k6 run /scripts/load-test.js

# Medium load test (500 req/s)
integration-test-medium:
	@echo "Running MEDIUM load test (500 req/s)..."
	@mkdir -p integration-test/results
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=500 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=30s \
		k6 run /scripts/load-test.js

# Heavy load test (1000 req/s)
integration-test-heavy:
	@echo "Running HEAVY load test (1000 req/s)..."
	@mkdir -p integration-test/results
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=1000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=30s \
		k6 run /scripts/load-test.js

# Very heavy load test (5000 req/s) with goroutine monitoring
integration-test-5k:
	@echo "Running VERY HEAVY load test (5,000 req/s)..."
	@echo "⚠️  WARNING: This is a very high load test!"
	@echo "⚠️  Ensure your system has sufficient resources"
	@echo ""
	@echo "Configuration:"
	@echo "  - Target: 5,000 requests per second"
	@echo "  - Duration: 60 seconds sustained load"
	@echo "  - Ramp-up: 45 seconds to reach target"
	@echo "  - Max VUs: 10,000"
	@echo "  - Monitoring: Goroutine tracking enabled"
	@echo ""
	@mkdir -p integration-test/results
	@echo "Starting goroutine monitor in background..."
	@./integration-test/monitor_goroutines.sh & \
	MONITOR_PID=$$! ; \
	echo "Monitor PID: $$MONITOR_PID" ; \
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=5000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=45s \
		k6 run /scripts/load-test.js ; \
	TEST_EXIT=$$? ; \
	echo "" ; \
	echo "Stopping goroutine monitor..." ; \
	kill $$MONITOR_PID 2>/dev/null || true ; \
	sleep 2 ; \
	echo "" ; \
	echo "═══════════════════════════════════════" ; \
	echo "Goroutine Analysis Results:" ; \
	echo "═══════════════════════════════════════" ; \
	ls -lh integration-test/results/goroutine_monitor_*.log 2>/dev/null | tail -1 || echo "No logs found" ; \
	echo "" ; \
	echo "📊 View detailed results:" ; \
	echo "   cat integration-test/results/goroutine_monitor_*.log | tail -50" ; \
	echo "   ls integration-test/results/goroutine_profiles_*/" ; \
	exit $$TEST_EXIT

# Extreme load test (2000 req/s)
integration-test-extreme:
	@echo "Running EXTREME load test (2000 req/s)..."
	@echo "WARNING: This is a very high load test!"
	@mkdir -p integration-test/results
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=2000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=40s \
		k6 run /scripts/load-test.js

# Ultra load test (10000 req/s)
integration-test-10k:
	@echo "Running ULTRA load test (10,000 req/s)..."
	@echo "⚠️  WARNING: This is an ULTRA HIGH load test!"
	@echo "⚠️  Ensure your system has sufficient resources"
	@echo ""
	@echo "Configuration:"
	@echo "  - Target: 10,000 requests per second"
	@echo "  - Duration: 60 seconds sustained load"
	@echo "  - Ramp-up: 60 seconds to reach target"
	@echo "  - Max VUs: 20,000"
	@echo "  - Monitoring: Goroutine tracking enabled"
	@echo ""
	@mkdir -p integration-test/results
	@echo "Starting goroutine monitor in background..."
	@./integration-test/monitor_goroutines.sh & \
	MONITOR_PID=$$! ; \
	echo "Monitor PID: $$MONITOR_PID" ; \
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=10000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=60s \
		k6 run /scripts/load-test.js ; \
	TEST_EXIT=$$? ; \
	echo "" ; \
	echo "Stopping goroutine monitor..." ; \
	kill $$MONITOR_PID 2>/dev/null || true ; \
	sleep 2 ; \
	echo "" ; \
	echo "═══════════════════════════════════════" ; \
	echo "Goroutine Analysis Results:" ; \
	echo "═══════════════════════════════════════" ; \
	ls -lh integration-test/results/goroutine_monitor_*.log 2>/dev/null | tail -1 || echo "No logs found" ; \
	echo "" ; \
	echo "📊 View detailed results:" ; \
	echo "   cat integration-test/results/goroutine_monitor_*.log | tail -50" ; \
	echo "   ls integration-test/results/goroutine_profiles_*/" ; \
	exit $$TEST_EXIT

# Massive load test (25000 req/s)
integration-test-25k:
	@echo "Running MASSIVE load test (25,000 req/s)..."
	@echo "🔥 WARNING: This is an EXTREMELY HIGH load test!"
	@echo "⚠️  Ensure your system has sufficient resources (CPU, Memory, Network)"
	@echo "⚠️  This may impact system stability"
	@echo ""
	@echo "Configuration:"
	@echo "  - Target: 25,000 requests per second"
	@echo "  - Duration: 60 seconds sustained load"
	@echo "  - Ramp-up: 90 seconds to reach target"
	@echo "  - Max VUs: 50,000"
	@echo "  - Monitoring: Goroutine tracking enabled"
	@echo ""
	@mkdir -p integration-test/results
	@echo "Starting goroutine monitor in background..."
	@./integration-test/monitor_goroutines.sh & \
	MONITOR_PID=$$! ; \
	echo "Monitor PID: $$MONITOR_PID" ; \
	cd integration-test && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=25000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=90s \
		k6 run /scripts/load-test.js ; \
	TEST_EXIT=$$? ; \
	echo "" ; \
	echo "Stopping goroutine monitor..." ; \
	kill $$MONITOR_PID 2>/dev/null || true ; \
	sleep 2 ; \
	echo "" ; \
	echo "═══════════════════════════════════════" ; \
	echo "Goroutine Analysis Results:" ; \
	echo "═══════════════════════════════════════" ; \
	ls -lh integration-test/results/goroutine_monitor_*.log 2>/dev/null | tail -1 || echo "No logs found" ; \
	echo "" ; \
	echo "📊 View detailed results:" ; \
	echo "   cat integration-test/results/goroutine_monitor_*.log | tail -50" ; \
	echo "   ls integration-test/results/goroutine_profiles_*/" ; \
	exit $$TEST_EXIT

# Follow logs from all services
integration-test-logs:
	@echo "Following logs from all services (Ctrl+C to stop)..."
	cd integration-test && docker-compose -f docker-compose.integration.yml logs -f

# Open Prometheus UI
integration-test-metrics:
	@echo "Opening Prometheus UI..."
	@open http://localhost:9090 || xdg-open http://localhost:9090 || echo "Please open http://localhost:9090 in your browser"

# Open Grafana UI
integration-test-grafana:
	@echo "Opening Grafana UI..."
	@echo "Default credentials: admin / admin"
	@open http://localhost:3000 || xdg-open http://localhost:3000 || echo "Please open http://localhost:3000 in your browser"

# Monitor goroutines in real-time (standalone)
monitor-goroutines:
	@echo "Starting real-time goroutine monitoring..."
	@echo "Press Ctrl+C to stop"
	@echo ""
	@./integration-test/monitor_goroutines.sh

# Analyze goroutine monitoring results
analyze-goroutines:
	@./integration-test/analyze_goroutines.sh

# Display test results
integration-test-results:
	@echo "Integration Test Results"
	@echo "========================"
	@echo ""
	@if [ -f integration-test/results/summary.txt ]; then \
		cat integration-test/results/summary.txt; \
	else \
		echo "No results found. Run 'make integration-test-run' first."; \
	fi

# Full integration test cycle
integration-test-full:
	@echo "Running FULL integration test cycle..."
	@echo "======================================"
	@echo ""
	@$(MAKE) integration-test-up
	@echo ""
	@echo "Waiting 2 seconds for all services to fully stabilize..."
	@sleep 2
	@echo ""
	@$(MAKE) integration-test-run
	@echo ""
	@$(MAKE) integration-test-results
	@echo ""
	@echo "Test complete! Services are still running."
	@echo "View metrics at: http://localhost:9090"
	@echo "Run 'make integration-test-down' to stop services."
