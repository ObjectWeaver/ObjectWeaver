# ObjectWeaver Makefile
# Benchmark and testing utilities

.PHONY: help benchmark benchmark-all benchmark-field benchmark-queue benchmark-strategy benchmark-compare benchmark-profile-cpu benchmark-profile-mem benchmark-fast benchmark-detailed clean-profiles

# Default target
help:
	@echo "ObjectWeaver Benchmark Commands"
	@echo "================================"
	@echo ""
	@echo "Quick Commands:"
	@echo "  make benchmark          - Run all benchmarks with memory stats"
	@echo "  make benchmark-fast     - Quick benchmark run (shorter duration)"
	@echo "  make benchmark-detailed - Detailed run with 10 iterations"
	@echo ""
	@echo "Specific Benchmarks:"
	@echo "  make benchmark-field    - Field processing concurrency benchmarks"
	@echo "  make benchmark-queue    - Job queue and orchestrator benchmarks"
	@echo "  make benchmark-strategy - Execution strategy benchmarks"
	@echo ""
	@echo "Profiling:"
	@echo "  make benchmark-profile-cpu - Run with CPU profiling"
	@echo "  make benchmark-profile-mem - Run with memory profiling"
	@echo "  make benchmark-compare     - Run benchmarks and save for comparison"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean-profiles    - Remove all profile files"
	@echo "  make test              - Run all tests"
	@echo ""

# Run all benchmarks with memory statistics
benchmark:
	@echo "Running all concurrency benchmarks..."
	@echo "====================================="
	@echo ""
	@go test -bench=. -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		./llmManagement/LLM/ \
		| tee benchmark-results.txt
	@echo ""
	@echo "Results saved to benchmark-results.txt"

# Run all benchmarks (alias)
benchmark-all: benchmark

# Quick benchmark run (1 second each)
benchmark-fast:
	@echo "Running quick benchmarks..."
	@go test -bench=. -benchmem -benchtime=1s \
		./orchestration/jos/infrastructure/execution/ \
		./llmManagement/LLM/

# Detailed benchmark with multiple iterations
benchmark-detailed:
	@echo "Running detailed benchmarks (10 iterations)..."
	@echo "This may take several minutes..."
	@echo ""
	@go test -bench=. -benchmem -benchtime=5s -count=10 \
		./orchestration/jos/infrastructure/execution/ \
		./llmManagement/LLM/ \
		| tee benchmark-detailed-results.txt
	@echo ""
	@echo "Results saved to benchmark-detailed-results.txt"

# Field processing benchmarks only
benchmark-field:
	@echo "Running field processing benchmarks..."
	@echo "======================================"
	@echo ""
	@go test -bench=BenchmarkProcess -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee benchmark-field-results.txt
	@echo ""
	@echo "Results saved to benchmark-field-results.txt"

# Job queue and orchestrator benchmarks
benchmark-queue:
	@echo "Running job queue and orchestrator benchmarks..."
	@echo "==============================================="
	@echo ""
	@go test -bench=BenchmarkJob -benchmem -benchtime=3s \
		./llmManagement/LLM/ \
		| tee benchmark-queue-results.txt
	@echo ""
	@go test -bench=BenchmarkOrchestrator -benchmem -benchtime=3s \
		./llmManagement/LLM/ \
		| tee -a benchmark-queue-results.txt
	@echo ""
	@go test -bench=BenchmarkConcurrent -benchmem -benchtime=3s \
		./llmManagement/LLM/ \
		| tee -a benchmark-queue-results.txt
	@echo ""
	@go test -bench=BenchmarkQueue -benchmem -benchtime=3s \
		./llmManagement/LLM/ \
		| tee -a benchmark-queue-results.txt
	@echo ""
	@echo "Results saved to benchmark-queue-results.txt"

# Execution strategy benchmarks
benchmark-strategy:
	@echo "Running execution strategy benchmarks..."
	@echo "========================================"
	@echo ""
	@go test -bench=BenchmarkParallel -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee benchmark-strategy-results.txt
	@echo ""
	@go test -bench=BenchmarkSequential -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee -a benchmark-strategy-results.txt
	@echo ""
	@go test -bench=BenchmarkDependency -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee -a benchmark-strategy-results.txt
	@echo ""
	@go test -bench=BenchmarkStrategy -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee -a benchmark-strategy-results.txt
	@echo ""
	@echo "Results saved to benchmark-strategy-results.txt"

# CPU profiling
benchmark-profile-cpu:
	@echo "Running benchmarks with CPU profiling..."
	@echo "========================================"
	@echo ""
	@mkdir -p profiles
	@echo "Field Processing CPU Profile..."
	@go test -bench=BenchmarkProcessFields -cpuprofile=profiles/field-cpu.prof \
		./orchestration/jos/infrastructure/execution/
	@echo ""
	@echo "Job Queue CPU Profile..."
	@go test -bench=BenchmarkJobQueue -cpuprofile=profiles/queue-cpu.prof \
		./llmManagement/LLM/
	@echo ""
	@echo "Orchestrator CPU Profile..."
	@go test -bench=BenchmarkOrchestrator -cpuprofile=profiles/orchestrator-cpu.prof \
		./llmManagement/LLM/
	@echo ""
	@echo "Strategy CPU Profile..."
	@go test -bench=BenchmarkParallelStrategy -cpuprofile=profiles/strategy-cpu.prof \
		./orchestration/jos/infrastructure/execution/
	@echo ""
	@echo "CPU profiles saved to profiles/ directory"
	@echo ""
	@echo "Analyze with:"
	@echo "  go tool pprof profiles/field-cpu.prof"
	@echo "  go tool pprof -http=:8080 profiles/field-cpu.prof"

# Memory profiling
benchmark-profile-mem:
	@echo "Running benchmarks with memory profiling..."
	@echo "=========================================="
	@echo ""
	@mkdir -p profiles
	@echo "Field Processing Memory Profile..."
	@go test -bench=BenchmarkProcessFields -memprofile=profiles/field-mem.prof -benchmem \
		./orchestration/jos/infrastructure/execution/
	@echo ""
	@echo "Job Queue Memory Profile..."
	@go test -bench=BenchmarkJobQueue -memprofile=profiles/queue-mem.prof -benchmem \
		./llmManagement/LLM/
	@echo ""
	@echo "Orchestrator Memory Profile..."
	@go test -bench=BenchmarkOrchestrator -memprofile=profiles/orchestrator-mem.prof -benchmem \
		./llmManagement/LLM/
	@echo ""
	@echo "Strategy Memory Profile..."
	@go test -bench=BenchmarkParallelStrategy -memprofile=profiles/strategy-mem.prof -benchmem \
		./orchestration/jos/infrastructure/execution/
	@echo ""
	@echo "Memory profiles saved to profiles/ directory"
	@echo ""
	@echo "Analyze with:"
	@echo "  go tool pprof -alloc_space profiles/field-mem.prof"
	@echo "  go tool pprof -http=:8080 profiles/field-mem.prof"

# Save benchmarks for comparison (useful before/after changes)
benchmark-compare:
	@echo "Running benchmarks for comparison..."
	@echo "===================================="
	@echo ""
	@mkdir -p benchmarks
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	echo "Saving results to benchmarks/benchmark-$$TIMESTAMP.txt"; \
	go test -bench=. -benchmem -benchtime=5s -count=10 \
		./orchestration/jos/infrastructure/execution/ \
		./llmManagement/LLM/ \
		> benchmarks/benchmark-$$TIMESTAMP.txt
	@echo ""
	@echo "Compare with previous run using:"
	@echo "  benchstat benchmarks/benchmark-<old>.txt benchmarks/benchmark-<new>.txt"
	@echo ""
	@echo "Install benchstat with:"
	@echo "  go install golang.org/x/perf/cmd/benchstat@latest"

# Clean generated files
clean-profiles:
	@echo "Cleaning profile files..."
	@rm -rf profiles/
	@rm -f benchmark-*.txt
	@echo "Done!"

# Run all tests
test:
	@echo "Running all tests..."
	@go test -v ./...

# Specific concurrency benchmarks with different worker counts
benchmark-workers:
	@echo "Benchmarking different worker counts..."
	@echo "======================================="
	@echo ""
	@echo "1 Worker:"
	@go test -bench=BenchmarkJobQueue/.*1Worker -benchmem ./llmManagement/LLM/
	@echo ""
	@echo "4 Workers:"
	@go test -bench=BenchmarkJobQueue/.*4Worker -benchmem ./llmManagement/LLM/
	@echo ""
	@echo "10 Workers:"
	@go test -bench=BenchmarkJobQueue/.*10Worker -benchmem ./llmManagement/LLM/

# Concurrency scaling test
benchmark-scaling:
	@echo "Testing concurrency scaling..."
	@echo "=============================="
	@echo ""
	@go test -bench=BenchmarkConcurrencyScaling -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee benchmark-scaling-results.txt
	@echo ""
	@echo "Results saved to benchmark-scaling-results.txt"

# Sequential vs Parallel comparison
benchmark-comparison:
	@echo "Comparing Sequential vs Parallel execution..."
	@echo "============================================"
	@echo ""
	@go test -bench=BenchmarkSequentialVsParallel -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee benchmark-comparison-results.txt
	@echo ""
	@go test -bench=BenchmarkStrategyComparison -benchmem -benchtime=3s \
		./orchestration/jos/infrastructure/execution/ \
		| tee -a benchmark-comparison-results.txt
	@echo ""
	@echo "Results saved to benchmark-comparison-results.txt"

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

# Full check: format, lint, test, benchmark
check: fmt lint test benchmark-fast
	@echo ""
	@echo "All checks passed! ✓"

# Show benchmark history
benchmark-history:
	@echo "Benchmark History:"
	@echo "=================="
	@ls -lht benchmarks/ 2>/dev/null || echo "No benchmark history found. Run 'make benchmark-compare' first."

# Interactive profile viewer
profile-view-cpu:
	@echo "Starting interactive CPU profile viewer..."
	@go tool pprof -http=:8080 profiles/field-cpu.prof

profile-view-mem:
	@echo "Starting interactive memory profile viewer..."
	@go tool pprof -http=:8080 profiles/field-mem.prof
