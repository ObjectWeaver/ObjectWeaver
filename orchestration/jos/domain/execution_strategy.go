// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://objectweaver.dev/licensing/server-side-public-license>.
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