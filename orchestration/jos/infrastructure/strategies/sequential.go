package strategies

import "objectweaver/orchestration/jos/domain"

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

//Execute uses the collected tasks and then sends it through to the executor (typically the TaskComposite class) so that the data can be generated using the processors.
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