package domain

// ExecutionStrategy - Determines how tasks are scheduled and executed
//
// Implementations:
//   - infrastructure/strategies/sequential.go: SequentialStrategy (one-by-one execution)
//   - infrastructure/strategies/parrell.go: ParallelStrategy (concurrent execution)
//   - infrastructure/strategies/dependency_aware.go: DependencyAwareStrategy (topological sort)
//
// Created by: factory/generator_factory.go:createStrategy()
// Used by: All Generator implementations
//
// Responsibilities:
//   - Schedule tasks into execution stages
//   - Respect field dependencies (ProcessingOrder)
//   - Execute tasks using provided TaskExecutor
//   - Optimize for parallelism where possible
//   - Return all TaskResults in order
type ExecutionStrategy interface {
	Schedule(tasks []*FieldTask) (*ExecutionPlan, error)
	Execute(plan *ExecutionPlan, executor TaskExecutor, context *ExecutionContext) ([]*TaskResult, error)
}

// ExecutionPlan represents a plan for executing tasks
type ExecutionPlan struct {
	Stages   []ExecutionStage
	Metadata map[string]interface{}
}

// ExecutionStage represents a stage in execution
type ExecutionStage struct {
	Tasks    []*FieldTask
	Parallel bool
}