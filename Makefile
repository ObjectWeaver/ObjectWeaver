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

.PHONY: integration-test-help integration-test-up integration-test-down integration-test-run integration-test-clean integration-test-logs integration-test-metrics integration-test-full integration-test-10k integration-test-matrix integration-test-matrix-single

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
	@echo "Matrix Tests (Load × Complexity):"
	@echo "  make integration-test-matrix           - Run full matrix (all schemas × 50-2000 req/s)"
	@echo "  make integration-test-matrix-quick     - Quick matrix (3 schemas × 4 load levels)"
	@echo "  make integration-test-matrix-light     - Light matrix (2 schemas × 2 load levels)"
	@echo "  make integration-test-matrix-minimal   - Test minimal schema at all load levels"
	@echo "  make integration-test-matrix-simple    - Test simple schema at all load levels"
	@echo "  make integration-test-matrix-nested    - Test nested schema at all load levels"
	@echo "  make integration-test-matrix-complex   - Test complex schema at all load levels"
	@echo "  make integration-test-matrix-100       - Test all schemas at 100 req/s"
	@echo "  make integration-test-matrix-500       - Test all schemas at 500 req/s"
	@echo "  make integration-test-matrix-1000      - Test all schemas at 1000 req/s"
	@echo "  make integration-test-matrix-2000      - Test all schemas at 2000 req/s"
	@echo "  make integration-test-matrix-single    - Single test (SCHEMA=x RPS=y)"
	@echo "  make integration-test-matrix-results   - View latest matrix results"
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
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml up -d --build
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
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml down
	@echo "✓ Services stopped"

# Stop and clean volumes
integration-test-clean:
	@echo "Cleaning integration test environment..."
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml down -v
	@rm -rf integration-test/e2e/results/*
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
	@mkdir -p integration-test/e2e/results
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=1000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=30s \
		-e BASE_URL=http://objectweaver:2008 \
		-e PASSWORD=test-password \
		k6 run /scripts/load-test.js
	@echo ""
	@echo "✓ Load test completed!"
	@echo "Results saved to: integration-test/e2e/results/"

# Light load test (100 req/s)
integration-test-light:
	@echo "Running LIGHT load test (100 req/s)..."
	@mkdir -p integration-test/e2e/results
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=100 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=20s \
		k6 run /scripts/load-test.js

# Medium load test (500 req/s)
integration-test-medium:
	@echo "Running MEDIUM load test (500 req/s)..."
	@mkdir -p integration-test/e2e/results
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=500 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=30s \
		k6 run /scripts/load-test.js

# Heavy load test (1000 req/s)
integration-test-heavy:
	@echo "Running HEAVY load test (1000 req/s)..."
	@mkdir -p integration-test/e2e/results
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
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
	@mkdir -p integration-test/e2e/results
	@echo "Starting goroutine monitor in background..."
	@./integration-test/e2e/monitor_goroutines.sh & \
	MONITOR_PID=$$! ; \
	echo "Monitor PID: $$MONITOR_PID" ; \
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
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
	ls -lh integration-test/e2e/results/goroutine_monitor_*.log 2>/dev/null | tail -1 || echo "No logs found" ; \
	echo "" ; \
	echo "📊 View detailed results:" ; \
	echo "   cat integration-test/e2e/results/goroutine_monitor_*.log | tail -50" ; \
	echo "   ls integration-test/e2e/results/goroutine_profiles_*/" ; \
	exit $$TEST_EXIT

# Extreme load test (2000 req/s)
integration-test-extreme:
	@echo "Running EXTREME load test (2000 req/s)..."
	@echo "WARNING: This is a very high load test!"
	@mkdir -p integration-test/e2e/results
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
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
	@mkdir -p integration-test/e2e/results
	@echo "Starting goroutine monitor in background..."
	@./integration-test/e2e/monitor_goroutines.sh & \
	MONITOR_PID=$$! ; \
	echo "Monitor PID: $$MONITOR_PID" ; \
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
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
	ls -lh integration-test/e2e/results/goroutine_monitor_*.log 2>/dev/null | tail -1 || echo "No logs found" ; \
	echo "" ; \
	echo "📊 View detailed results:" ; \
	echo "   cat integration-test/e2e/results/goroutine_monitor_*.log | tail -50" ; \
	echo "   ls integration-test/e2e/results/goroutine_profiles_*/" ; \
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
	@mkdir -p integration-test/e2e/results
	@echo "Starting goroutine monitor in background..."
	@./integration-test/e2e/monitor_goroutines.sh & \
	MONITOR_PID=$$! ; \
	echo "Monitor PID: $$MONITOR_PID" ; \
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
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
	ls -lh integration-test/e2e/results/goroutine_monitor_*.log 2>/dev/null | tail -1 || echo "No logs found" ; \
	echo "" ; \
	echo "📊 View detailed results:" ; \
	echo "   cat integration-test/e2e/results/goroutine_monitor_*.log | tail -50" ; \
	echo "   ls integration-test/e2e/results/goroutine_profiles_*/" ; \
	exit $$TEST_EXIT

# Follow logs from all services
integration-test-logs:
	@echo "Following logs from all services (Ctrl+C to stop)..."
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml logs -f

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
	@./integration-test/e2e/monitor_goroutines.sh

# Analyze goroutine monitoring results
analyze-goroutines:
	@./integration-test/e2e/analyze_goroutines.sh

# Display test results
integration-test-results:
	@echo "Integration Test Results"
	@echo "========================"
	@echo ""
	@if [ -f integration-test/e2e/results/summary.txt ]; then \
		cat integration-test/e2e/results/summary.txt; \
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

# ============================================
# MATRIX TESTS (Load × Schema Complexity)
# ============================================

# Full matrix test - all schemas × all load levels (up to 2000 req/s)
# Estimated time: ~30-45 minutes
integration-test-matrix:
	@echo "Running FULL performance matrix test..."
	@chmod +x scripts/matrix-test-runner.sh
	@./scripts/matrix-test-runner.sh

# Quick matrix - subset for faster iteration (~10 min)
integration-test-matrix-quick:
	@echo "Running QUICK matrix test (subset)..."
	@chmod +x scripts/matrix-test-runner.sh
	@SCHEMA_TYPES="minimal simple complex" LOAD_LEVELS="100 300 500 1000" ./scripts/matrix-test-runner.sh

# Light matrix - even faster (~5 min)
integration-test-matrix-light:
	@echo "Running LIGHT matrix test..."
	@chmod +x scripts/matrix-test-runner.sh
	@SCHEMA_TYPES="minimal complex" LOAD_LEVELS="100 500" TEST_DURATION=10s ./scripts/matrix-test-runner.sh

# Matrix by schema type - test one schema at all load levels
integration-test-matrix-minimal:
	@chmod +x scripts/matrix-test-runner.sh
	@SCHEMA_TYPES="minimal" ./scripts/matrix-test-runner.sh

integration-test-matrix-simple:
	@chmod +x scripts/matrix-test-runner.sh
	@SCHEMA_TYPES="simple" ./scripts/matrix-test-runner.sh

integration-test-matrix-nested:
	@chmod +x scripts/matrix-test-runner.sh
	@SCHEMA_TYPES="nested" ./scripts/matrix-test-runner.sh

integration-test-matrix-complex:
	@chmod +x scripts/matrix-test-runner.sh
	@SCHEMA_TYPES="complex" ./scripts/matrix-test-runner.sh

# Matrix by load level - test all schemas at specific load
integration-test-matrix-100:
	@chmod +x scripts/matrix-test-runner.sh
	@LOAD_LEVELS="100" ./scripts/matrix-test-runner.sh

integration-test-matrix-500:
	@chmod +x scripts/matrix-test-runner.sh
	@LOAD_LEVELS="500" ./scripts/matrix-test-runner.sh

integration-test-matrix-1000:
	@chmod +x scripts/matrix-test-runner.sh
	@LOAD_LEVELS="1000" ./scripts/matrix-test-runner.sh

integration-test-matrix-2000:
	@chmod +x scripts/matrix-test-runner.sh
	@LOAD_LEVELS="2000" ./scripts/matrix-test-runner.sh

# Single matrix test - run with SCHEMA and RPS environment variables
# Example: SCHEMA=simple RPS=100 make integration-test-matrix-single
integration-test-matrix-single:
	@echo "Running single matrix test..."
	@echo "============================="
	@echo "Schema: $(or $(SCHEMA),simple)"
	@echo "Target RPS: $(or $(RPS),100)"
	@echo ""
	@mkdir -p integration-test/e2e/results/matrix
	cd integration-test/e2e && docker-compose -f docker-compose.integration.yml run --rm \
		-e MAX_RPS=$(or $(RPS),100) \
		-e DURATION=20s \
		-e RAMP_UP_TIME=5s \
		-e SCHEMA_TYPE=$(or $(SCHEMA),simple) \
		-e LLM_LATENCY_MS=5000 \
		-e BASE_URL=http://objectweaver:2008 \
		-e PASSWORD=test-password \
		k6 run /scripts/matrix-test.js

# View latest matrix results
integration-test-matrix-results:
	@echo "Latest Matrix Test Results:"
	@echo "==========================="
	@ls -t integration-test/e2e/results/matrix/*.txt 2>/dev/null | head -1 | xargs cat 2>/dev/null || echo "No results found. Run 'make integration-test-matrix' first."

# ============================================================================
# Distributed Load Testing (50k-200k+ req/s)
# ============================================================================
# Uses multiple parallel k6 instances to achieve higher load than single k6

.PHONY: distributed-test-help distributed-test-50k distributed-test-100k distributed-test-200k distributed-test

distributed-test-help:
	@echo "Distributed Load Testing Commands"
	@echo "=================================="
	@echo ""
	@echo "These use multiple parallel k6 instances to generate higher load."
	@echo ""
	@echo "⚠️  IMPORTANT: With 5s LLM latency and default worker pool (10k):"
	@echo "    Max theoretical throughput = 10,000 workers / 5s = 2,000 req/s"
	@echo ""
	@echo "    To test higher loads, either:"
	@echo "    1. Reduce LLM latency (edit docker-compose MOCK_LLM_MIN_DELAY_MS)"
	@echo "    2. Increase worker pool (set WORKER_POOL_SIZE env var)"
	@echo ""
	@echo "Realistic Targets (with 5s LLM latency):"
	@echo "  make distributed-test INSTANCES=2 SCHEMA=minimal RPS=1000  # 2k total"
	@echo "  make distributed-test INSTANCES=3 SCHEMA=minimal RPS=500   # 1.5k total"
	@echo ""
	@echo "High Load Targets (requires tuning worker pool):"
	@echo "  make distributed-test-50k   - 5 instances × 10k RPS = 50k req/s target"
	@echo "  make distributed-test-100k  - 10 instances × 10k RPS = 100k req/s target"
	@echo ""
	@echo "Custom Target:"
	@echo "  make distributed-test INSTANCES=5 SCHEMA=simple RPS=8000"
	@echo ""
	@echo "Parameters:"
	@echo "  INSTANCES - Number of parallel k6 containers (default: 5)"
	@echo "  SCHEMA    - Schema type: minimal, simple, nested, complex (default: minimal)"
	@echo "  RPS       - Requests per second per instance (default: 10000)"
	@echo ""

# Distributed test: 50k req/s (5 instances × 10k each)
distributed-test-50k:
	@echo "Starting distributed load test: 50k req/s"
	@echo "=========================================="
	@echo "Configuration: 5 instances × 10,000 req/s each"
	@chmod +x scripts/distributed-load-test.sh
	./scripts/distributed-load-test.sh 5 minimal 10000

# Distributed test: 100k req/s (10 instances × 10k each)
distributed-test-100k:
	@echo "Starting distributed load test: 100k req/s"
	@echo "==========================================="
	@echo "Configuration: 10 instances × 10,000 req/s each"
	@chmod +x scripts/distributed-load-test.sh
	./scripts/distributed-load-test.sh 10 minimal 10000

# Distributed test: 200k req/s (20 instances × 10k each)
distributed-test-200k:
	@echo "Starting distributed load test: 200k req/s"
	@echo "==========================================="
	@echo "Configuration: 20 instances × 10,000 req/s each"
	@chmod +x scripts/distributed-load-test.sh
	./scripts/distributed-load-test.sh 20 minimal 10000

# Custom distributed test
# Usage: make distributed-test INSTANCES=5 SCHEMA=simple RPS=8000
distributed-test:
	@echo "Starting custom distributed load test"
	@echo "======================================"
	@echo "Instances: $(or $(INSTANCES),5)"
	@echo "Schema: $(or $(SCHEMA),minimal)"
	@echo "RPS per instance: $(or $(RPS),10000)"
	@echo "Total target RPS: $$(( $(or $(INSTANCES),5) * $(or $(RPS),10000) ))"
	@chmod +x scripts/distributed-load-test.sh
	./scripts/distributed-load-test.sh $(or $(INSTANCES),5) $(or $(SCHEMA),minimal) $(or $(RPS),10000)

# ============================================================================
# Fast LLM Testing (100ms latency - test true server capacity)
# ============================================================================
.PHONY: fast-llm-test-up fast-llm-test-down fast-llm-test-5k fast-llm-test-10k fast-llm-test-matrix

fast-llm-test-help:
	@echo "Fast LLM Testing (100ms latency)"
	@echo "================================="
	@echo ""
	@echo "Tests ObjectWeaver with 100ms LLM latency instead of 5000ms."
	@echo "This removes the LLM bottleneck to show TRUE server capacity."
	@echo ""
	@echo "With 100ms latency, theoretical max = 10,000 workers / 0.1s = 100,000 req/s"
	@echo ""
	@echo "Commands:"
	@echo "  make fast-llm-test-up      - Start services with 100ms LLM"
	@echo "  make fast-llm-test-down    - Stop services"
	@echo "  make fast-llm-test-5k      - Test 5,000 req/s"
	@echo "  make fast-llm-test-10k     - Test 10,000 req/s"
	@echo "  make fast-llm-test-matrix  - Test all schemas at 5000 req/s"
	@echo ""

fast-llm-test-up:
	@echo "Starting fast LLM test environment (100ms latency)..."
	cd integration-test/e2e && docker-compose -f docker-compose.fast-llm.yml up -d --build
	@sleep 5
	@curl -s http://localhost:2008/health > /dev/null && echo "✓ Services ready!" || echo "⚠️  Still starting..."

fast-llm-test-down:
	@echo "Stopping fast LLM test environment..."
	cd integration-test/e2e && docker-compose -f docker-compose.fast-llm.yml down

fast-llm-test-5k:
	@echo "Running 5k req/s test with 100ms LLM latency..."
	@$(MAKE) fast-llm-test-up
	@sleep 3
	cd integration-test/e2e && docker-compose -f docker-compose.fast-llm.yml run --rm \
		-e MAX_RPS=5000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=30s \
		-e SCHEMA_TYPE=minimal \
		k6 run /scripts/matrix-test.js

fast-llm-test-10k:
	@echo "Running 10k req/s test with 100ms LLM latency..."
	@$(MAKE) fast-llm-test-up
	@sleep 3
	cd integration-test/e2e && docker-compose -f docker-compose.fast-llm.yml run --rm \
		-e MAX_RPS=10000 \
		-e DURATION=60s \
		-e RAMP_UP_TIME=45s \
		-e SCHEMA_TYPE=minimal \
		k6 run /scripts/matrix-test.js

fast-llm-test-matrix:
	@echo "Running matrix test with 100ms LLM latency..."
	@$(MAKE) fast-llm-test-up
	@sleep 3
	@for schema in minimal simple nested complex; do \
		echo ""; \
		echo "Testing $$schema at 5000 req/s with 100ms LLM..."; \
		cd integration-test/e2e && docker-compose -f docker-compose.fast-llm.yml run --rm \
			-e MAX_RPS=5000 \
			-e DURATION=30s \
			-e RAMP_UP_TIME=10s \
			-e SCHEMA_TYPE=$$schema \
			-e LLM_LATENCY_MS=100 \
			k6 run /scripts/matrix-test.js; \
	done
	@$(MAKE) fast-llm-test-down
