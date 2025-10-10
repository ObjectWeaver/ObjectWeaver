package execution

import (
	"log"
	"objectGeneration/orchestration/jos/domain"
	"os"
	"sync"
)

// SequentialStrategy executes tasks one by one
type SequentialStrategy struct{}

func NewSequentialStrategy() *SequentialStrategy {
	return &SequentialStrategy{}
}

func (s *SequentialStrategy) Schedule(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
	return &domain.ExecutionPlan{
		Stages: []domain.ExecutionStage{
			{Tasks: tasks, Parallel: false},
		},
	}, nil
}

func (s *SequentialStrategy) Execute(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, 0)

	for _, stage := range plan.Stages {
		for _, task := range stage.Tasks {
			result, err := executor.Execute(task, context)
			if err != nil {
				return nil, err
			}
			results = append(results, result)

			// Update context with result
			context.SetGeneratedValue(task.Key(), result.Value())
		}
	}

	return results, nil
}

// ParallelStrategy executes independent tasks in parallel
type ParallelStrategy struct {
	maxConcurrency int
}

func NewParallelStrategy(maxConcurrency int) *ParallelStrategy {
	return &ParallelStrategy{maxConcurrency: maxConcurrency}
}

func (s *ParallelStrategy) Schedule(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
	// For parallel, we can execute all tasks at once if they're independent
	return &domain.ExecutionPlan{
		Stages: []domain.ExecutionStage{
			{Tasks: tasks, Parallel: true},
		},
	}, nil
}

func (s *ParallelStrategy) Execute(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, 0)

	for _, stage := range plan.Stages {
		if stage.Parallel {
			stageResults, err := s.executeParallel(stage.Tasks, executor, context)
			if err != nil {
				return nil, err
			}
			results = append(results, stageResults...)
		} else {
			for _, task := range stage.Tasks {
				result, err := executor.Execute(task, context)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
			}
		}
	}

	return results, nil
}

func (s *ParallelStrategy) executeParallel(tasks []*domain.FieldTask, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	verboseLogs := os.Getenv("VERBOSE_LOGS") == "true"

	if verboseLogs {
		log.Printf("[ParallelStrategy] Executing %d tasks in parallel with max concurrency %d", len(tasks), s.maxConcurrency)
	}

	sem := make(chan struct{}, s.maxConcurrency)
	resultChan := make(chan *domain.TaskResult, len(tasks))
	errChan := make(chan error, len(tasks))

	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t *domain.FieldTask) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if verboseLogs {
				log.Printf("[ParallelStrategy] Starting task: %s", t.Key())
			}

			result, err := executor.Execute(t, context)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- result

			if verboseLogs {
				log.Printf("[ParallelStrategy] Completed task: %s", t.Key())
			}
		}(task)
	}

	wg.Wait()
	close(resultChan)
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	results := make([]*domain.TaskResult, 0, len(resultChan))
	for result := range resultChan {
		results = append(results, result)
	}

	if verboseLogs {
		log.Printf("[ParallelStrategy] All %d tasks completed", len(results))
	}

	return results, nil
}

// DependencyAwareStrategy optimizes execution based on dependencies
type DependencyAwareStrategy struct {
	maxConcurrency int
}

func NewDependencyAwareStrategy(maxConcurrency int) *DependencyAwareStrategy {
	return &DependencyAwareStrategy{maxConcurrency: maxConcurrency}
}

func (s *DependencyAwareStrategy) Schedule(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
	// Build dependency graph and create stages
	graph := NewDependencyGraph(tasks)
	stages := graph.TopoSort()

	return &domain.ExecutionPlan{
		Stages:   stages,
		Metadata: make(map[string]interface{}),
	}, nil
}

func (s *DependencyAwareStrategy) Execute(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, 0)

	for _, stage := range plan.Stages {
		if stage.Parallel && len(stage.Tasks) > 1 {
			// Use parallel strategy for this stage
			parallelStrat := NewParallelStrategy(s.maxConcurrency)
			stageResults, err := parallelStrat.executeParallel(stage.Tasks, executor, context)
			if err != nil {
				return nil, err
			}
			results = append(results, stageResults...)
		} else {
			// Execute sequentially
			for _, task := range stage.Tasks {
				result, err := executor.Execute(task, context)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
				context.SetGeneratedValue(task.Key(), result.Value())
			}
		}
	}

	return results, nil
}

// DependencyGraph helper for dependency analysis
type DependencyGraph struct {
	tasks []*domain.FieldTask
	graph map[string][]string
}

func NewDependencyGraph(tasks []*domain.FieldTask) *DependencyGraph {
	g := &DependencyGraph{
		tasks: tasks,
		graph: make(map[string][]string),
	}

	// Build adjacency list
	for _, task := range tasks {
		g.graph[task.ID()] = task.Dependencies()
	}

	return g
}

func (g *DependencyGraph) TopoSort() []domain.ExecutionStage {
	// Simple topological sort - returns stages that can be executed in parallel
	stages := make([]domain.ExecutionStage, 0)

	completed := make(map[string]bool)
	remaining := make([]*domain.FieldTask, len(g.tasks))
	copy(remaining, g.tasks)

	for len(remaining) > 0 {
		// Find tasks with all dependencies satisfied
		stageTasks := make([]*domain.FieldTask, 0)
		newRemaining := make([]*domain.FieldTask, 0)

		for _, task := range remaining {
			canExecute := true
			for _, dep := range task.Dependencies() {
				if !completed[dep] {
					canExecute = false
					break
				}
			}

			if canExecute {
				stageTasks = append(stageTasks, task)
				completed[task.ID()] = true
			} else {
				newRemaining = append(newRemaining, task)
			}
		}

		if len(stageTasks) == 0 && len(newRemaining) > 0 {
			// Circular dependency or error - add first task
			stageTasks = append(stageTasks, newRemaining[0])
			completed[newRemaining[0].ID()] = true
			newRemaining = newRemaining[1:]
		}

		if len(stageTasks) > 0 {
			stages = append(stages, domain.ExecutionStage{
				Tasks:    stageTasks,
				Parallel: len(stageTasks) > 1,
			})
		}

		remaining = newRemaining
	}

	return stages
}
