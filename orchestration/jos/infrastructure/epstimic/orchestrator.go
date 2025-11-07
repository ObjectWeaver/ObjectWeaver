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
package epstimic

import (
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/infrastructure"
)

// orchestrator will handle the overall tasks of processing

type Orchestrator struct {
	epstimicEngine EpstimicEngine
	workerCount    int
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
	// start with a buffered channel with n values - dependent on the config
	resultsChan := make(chan TempResult, o.workerCount)

	if task.Definition().Epistemic.Judges > 0 {
		o.workerCount = task.Definition().Epistemic.Judges
	}

	// start a loop of n values using the fan-out pattern to process the functions independently
	for i := 0; i < o.workerCount; i++ {
		go func() {
			if i != 0 {
				task.Definition().ModelConfig.Seed = infrastructure.GenerateSeed()
			}

			value, metadata, err := generate(task, context)
			if err != nil {
				resultsChan <- TempResult{
					Task:     task,
					Value:    nil,
					Metadata: nil,
					Error:    err,
				}
				return
			}
			resultsChan <- TempResult{
				Task:     task,
				Value:    value,
				Metadata: metadata,
				Error:    nil,
			}
		}()
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
