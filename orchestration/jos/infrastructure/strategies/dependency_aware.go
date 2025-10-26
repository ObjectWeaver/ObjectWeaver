package strategies

import (
	"objectweaver/orchestration/jos/domain"
)

// DependencyAwareStrategy optimizes execution based on dependencies
type DependencyAwareStrategy struct {
	maxConcurrency int
}

func NewDependencyAwareStrategy(maxConcurrency int) *DependencyAwareStrategy {
	return &DependencyAwareStrategy{maxConcurrency: maxConcurrency}
}

// Schedule sets up and sorts the tasks which are going to be executed later down the line.
func (s *DependencyAwareStrategy) Schedule(tasks []*domain.FieldTask) (*domain.ExecutionPlan, error) {
	// Build dependency graph and create stages
	graph := NewDependencyGraph(tasks)
	stages := graph.TopoSort()

	return &domain.ExecutionPlan{
		Stages:   stages,
		Metadata: make(map[string]interface{}),
	}, nil
}

// Execute uses the collected tasks and then sends it through to the executor (typically the TaskComposite class) so that the data can be generated using the processors.
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
				taskResults, err := executor.Execute(task, context)
				if err != nil {
					return nil, err
				}
				// Flatten results - executor may return multiple results from decision points
				results = append(results, taskResults...)

				// Update context with all results
				for _, res := range taskResults {
					context.SetGeneratedValue(res.Key(), res.Value())
				}
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

// TopoSort Simple topological sort - returns stages that can be executed in parallel
func (g *DependencyGraph) TopoSort() []domain.ExecutionStage {
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
