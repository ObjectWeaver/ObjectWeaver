package epstimic

import (
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// orchestrator will handle the overall tasks of processing

type Orchestrator struct {
	epstimicEngine EpstimicEngine
	workerCount    int
}

// ensureModelConfig ensures that the ModelConfig is initialized
func ensureModelConfig(def *jsonSchema.Definition) {
	if def.ModelConfig == nil {
		def.ModelConfig = &jsonSchema.ModelConfig{}
	}
}

type TempResult struct {
	Task     *domain.FieldTask
	Value    any
	Metadata *domain.ProviderMetadata
	Error    error
}

// EpstimicValidation(func(a, b))
func (o *Orchestrator) EpstimicValidation(
	task *domain.FieldTask,
	context *domain.ExecutionContext,
	generate func(task *domain.FieldTask, context *domain.ExecutionContext) (any, *domain.ProviderMetadata, error),
) (*domain.TaskResult, *domain.ProviderMetadata, error) {
	// Ensure ModelConfig is initialized at the start
	ensureModelConfig(task.Definition())

	// start with a buffered channel with n values - dependent on the config
	resultsChan := make(chan TempResult, o.workerCount)

	if task.Definition().Epistemic.Judges > 0 {
		o.workerCount = task.Definition().Epistemic.Judges
	}

	// start a loop of n values using the fan-out pattern to process the functions independently
	for i := 0; i < o.workerCount; i++ {
		go func(workerIndex int) {
			// Create a local copy of the task to avoid race conditions
			localTask := task
			// For workers after the first, generate a new random seed
			if workerIndex != 0 {
				// Create a copy of the definition to avoid modifying shared state
				defCopy := *task.Definition()
				if defCopy.ModelConfig != nil {
					modelConfigCopy := *defCopy.ModelConfig
					modelConfigCopy.Seed = infrastructure.GenerateSeed()
					defCopy.ModelConfig = &modelConfigCopy
				}
				// Create a new task with the copied definition
				localTask = domain.NewFieldTask(task.Key(), &defCopy, task.Parent())
			}

			value, metadata, err := generate(localTask, context)
			if err != nil {
				resultsChan <- TempResult{
					Task:     localTask,
					Value:    nil,
					Metadata: nil,
					Error:    err,
				}
				return
			}
			resultsChan <- TempResult{
				Task:     localTask,
				Value:    value,
				Metadata: metadata,
				Error:    nil,
			}
		}(i)
	}

	// consume the results chan and handle errors / results accordingly
	var results []TempResult
	for i := 0; i < o.workerCount; i++ {
		result := <-resultsChan
		if result.Error != nil {
			// handle error
			continue
		}
		results = append(results, result)
	}

	// send the list into the Epstimic engine for validation - this will return the single best result with the metadata containing all other information
	bestResult, metadata, err := o.epstimicEngine.Validate(results)
	if err != nil {
		// handle error
		return nil, nil, err
	}

	// Create TaskResult from the best value
	resultMetadata := domain.NewResultMetadata()
	resultMetadata.Cost = metadata.Cost
	resultMetadata.TokensUsed = metadata.TokensUsed
	resultMetadata.ModelUsed = metadata.Model
	resultMetadata.Choices = metadata.Choices

	taskResult := domain.NewTaskResult(task.ID(), task.Key(), bestResult.Value, resultMetadata)
	return taskResult, &metadata, nil
}
