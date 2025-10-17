package strategies

import (
	"log"
	"objectweaver/orchestration/jos/domain"
	"os"
	"sync"
)

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

//Execute uses the collected tasks and then sends it through to the executor (typically the TaskComposite class) so that the data can be generated using the processors.
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
	verboseLogs := os.Getenv("VERBOSE") == "true"

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
