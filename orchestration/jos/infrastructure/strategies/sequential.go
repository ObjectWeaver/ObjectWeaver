package strategies

import (
	"objectweaver/orchestration/jos/domain"
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

// Execute uses the collected tasks and then sends it through to the executor (typically the TaskComposite class) so that the data can be generated using the processors.
func (s *SequentialStrategy) Execute(plan *domain.ExecutionPlan, executor domain.TaskExecutor, context *domain.ExecutionContext) ([]*domain.TaskResult, error) {
	results := make([]*domain.TaskResult, 0)

	for _, stage := range plan.Stages {
		for _, task := range stage.Tasks {
			taskResults, err := executor.Execute(task, context)
			if err != nil {
				return nil, err
			}
			// Flatten results - executor may return multiple results from decision points
			results = append(results, taskResults...)

			// Update context with all results (primary + branches)
			for _, res := range taskResults {
				context.SetGeneratedValue(res.Key(), res.Value())
			}
		}
	}

	return results, nil
}
