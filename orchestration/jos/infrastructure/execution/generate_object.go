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
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package execution

import (
	"fmt"
	"log"
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// FieldProcessor handles processing of object properties (fields)
// This is the main orchestrator for recursive field generation
type FieldProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	generator            domain.Generator // For recursive loops and decision points
	decisionProcessor    *DecisionProcessor
	epstimicOrchestrator EpstimicOrchestrator
}

// EpstimicOrchestrator is an interface to avoid import cycle
type EpstimicOrchestrator interface {
	EpstimicValidation(task *domain.FieldTask, context *domain.ExecutionContext, generateFn func(*domain.FieldTask, *domain.ExecutionContext) (any, *domain.ProviderMetadata, error)) (*domain.TaskResult, *domain.ProviderMetadata, error)
}

func NewFieldProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *FieldProcessor {
	return &FieldProcessor{
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
	}
}

// SetGenerator sets the generator for recursive processing (circular dependency resolution)
func (fp *FieldProcessor) SetGenerator(generator domain.Generator) {
	fp.generator = generator
	// Create decision processor now that we have the generator
	fp.decisionProcessor = NewDecisionProcessor(generator)
}

// SetEpstimicOrchestrator sets the epstimic orchestrator for validation
func (fp *FieldProcessor) SetEpstimicOrchestrator(orchestrator EpstimicOrchestrator) {
	fp.epstimicOrchestrator = orchestrator
}

// ProcessFields processes all properties of an object definition
// Returns a channel of results for concurrent processing
func (fp *FieldProcessor) ProcessFields(schema *jsonSchema.Definition, parentTask *domain.FieldTask, context *domain.ExecutionContext) <-chan *domain.TaskResult {
	ch := make(chan *domain.TaskResult)

	go func() {
		defer close(ch)

		if schema == nil || schema.Properties == nil {
			log.Printf("Schema or Properties is nil")
			return
		}

		// Get ordered and remaining keys (for sequential vs concurrent processing)
		orderedKeys, remainingKeys := getOrderedKeys(schema)

		// Process ordered keys sequentially (dependencies)
		currentGen := make(map[string]interface{})
		fp.processSequentialFields(ch, schema, parentTask, context, currentGen, orderedKeys)

		// Process remaining keys concurrently
		fp.processConcurrentFields(ch, schema, parentTask, context, currentGen, remainingKeys)
	}()

	return ch
}

// processSequentialFields handles fields that must be processed in order
func (fp *FieldProcessor) processSequentialFields(
	ch chan<- *domain.TaskResult,
	schema *jsonSchema.Definition,
	parentTask *domain.FieldTask,
	context *domain.ExecutionContext,
	currentGen map[string]interface{},
	orderedKeys []string,
) {
	for _, key := range orderedKeys {
		childDef := schema.Properties[key]
		childDefCopy := childDef // Copy for goroutine safety

		// Create field task
		task := domain.NewFieldTask(key, &childDefCopy, parentTask)

		// Process the field and get all results (original + decision branches)
		results := fp.processField(task, context)

		// Add all results to current generation and send to channel
		for _, result := range results {
			if result != nil {
				currentGen[result.Key()] = result.Value()
				context.SetGeneratedValue(result.Key(), result.Value())
				ch <- result
			}
		}
	}
}

// processField processes a field and handles decision points
// Returns multiple results if decision point creates additional fields
func (fp *FieldProcessor) processField(task *domain.FieldTask, context *domain.ExecutionContext) []*domain.TaskResult {
	// Special case: Object types are processed by recursively calling ProcessFields
	// This maintains the concurrent/sequential ordering logic at every level
	if task.Definition().Type == jsonSchema.Object {
		return fp.processObjectField(task, context)
	}

	// Get appropriate processor for non-object types
	processor := fp.getProcessorForType(task.Definition())
	if processor == nil {
		log.Printf("No processor found for type %s", task.Definition().Type)
		return nil
	}

	// Process the field
	result, err := processor.Process(task, context)
	if err != nil {
		log.Printf("Error processing field %s: %v", task.Key(), err)
		return nil
	}

	if result == nil {
		return nil
	}

	// Add result to context so decision points can reference it
	context.SetGeneratedValue(result.Key(), result.Value())
	log.Printf("[FieldProcessor] Generated field '%s', value: %v", result.Key(), result.Value())

	// Score if criteria exist
	log.Printf("[FieldProcessor] Checking scoring criteria for field '%s': %v", task.Key(), task.Definition().ScoringCriteria != nil)
	if task.Definition().ScoringCriteria != nil {
		log.Printf("[FieldProcessor] Evaluating scores for field '%s'", task.Key())
		scores, err := fp.evaluateScores(result, task.Definition().ScoringCriteria, context)
		if err != nil {
			log.Printf("[FieldProcessor] Warning: scoring failed for field %s: %v", task.Key(), err)
		} else {
			// Attach scores to result metadata
			fp.attachScoresToResult(result, scores)
			log.Printf("[FieldProcessor] Field '%s' scores: %v", result.Key(), scores)
		}
	}

	// Check for decision point
	log.Printf("[FieldProcessor] Checking decision point for field '%s': %v", task.Key(), task.Definition().DecisionPoint != nil)
	if task.Definition().DecisionPoint != nil {
		log.Printf("[FieldProcessor] Processing decision point for field %s", task.Key())

		results, err := fp.decisionProcessor.ProcessDecisionPoint(task, result, context)
		if err != nil {
			log.Printf("[FieldProcessor] Decision point processing failed: %v", err)
			return []*domain.TaskResult{result} // Return original result on error
		}

		// Update context with branch results
		if len(results) > 1 {
			log.Printf("[FieldProcessor] Decision point created %d additional fields", len(results)-1)
			for _, branchResult := range results[1:] {
				context.SetGeneratedValue(branchResult.Key(), branchResult.Value())
				log.Printf("[FieldProcessor] Added branch field to context: %s = %v", branchResult.Key(), branchResult.Value())
			}
		}

		return results
	}

	// No decision point - return single result
	return []*domain.TaskResult{result}
}

// processObjectField handles object-type fields by recursively processing their properties
func (fp *FieldProcessor) processObjectField(task *domain.FieldTask, context *domain.ExecutionContext) []*domain.TaskResult {
	// Process nested fields recursively using the same FieldProcessor logic
	// This ensures concurrent/sequential ordering is maintained at every nesting level
	resultsCh := fp.ProcessFields(task.Definition(), task, context)

	// Collect all nested results into a map
	nestedResults := make(map[string]interface{})

	for result := range resultsCh {
		if result != nil {
			nestedResults[result.Key()] = result.Value()
		}
	}

	// Create result containing the nested object
	metadata := domain.NewResultMetadata()

	result := domain.NewTaskResult(task.ID(), task.Key(), nestedResults, metadata)
	result = result.WithPath(task.Path())

	return []*domain.TaskResult{result}
}

// processConcurrentFields handles fields that can be processed in parallel
func (fp *FieldProcessor) processConcurrentFields(
	ch chan<- *domain.TaskResult,
	schema *jsonSchema.Definition,
	parentTask *domain.FieldTask,
	context *domain.ExecutionContext,
	currentGen map[string]interface{},
	remainingKeys []string,
) {
	// Use a result channel to collect concurrent results
	resultCh := make(chan []*domain.TaskResult, len(remainingKeys))

	for _, key := range remainingKeys {
		childDef := schema.Properties[key]
		childDefCopy := childDef // Copy for goroutine safety

		go func(k string, def jsonSchema.Definition) {
			// Create field task
			task := domain.NewFieldTask(k, &def, parentTask)

			// Process the field with decision point support
			results := fp.processField(task, context)

			if results != nil {
				resultCh <- results
			} else {
				resultCh <- []*domain.TaskResult{}
			}
		}(key, childDefCopy)
	}

	// Collect all results
	for i := 0; i < len(remainingKeys); i++ {
		results := <-resultCh
		for _, result := range results {
			if result != nil {
				ch <- result
			}
		}
	}
}

// getProcessorForType returns the appropriate processor for a given type
func (fp *FieldProcessor) getProcessorForType(def *jsonSchema.Definition) domain.TypeProcessor {
	// Check for byte operations first (TTS, Image, STT)
	if def.TextToSpeech != nil || def.Image != nil || def.SpeechToText != nil {
		return NewByteProcessor(fp.llmProvider, fp.promptBuilder)
	}

	// Check for recursive loop
	if def.RecursiveLoop != nil && fp.generator != nil {
		baseProcessor := fp.getBaseProcessorForType(def.Type)
		decisionProcessor := NewDecisionProcessor(fp.generator)
		return NewRecursiveLoopProcessor(baseProcessor, fp.generator, decisionProcessor)
	}

	// Route by type
	return fp.getBaseProcessorForType(def.Type)
}

// getBaseProcessorForType returns the base processor for a type (without special handling)
func (fp *FieldProcessor) getBaseProcessorForType(schemaType jsonSchema.DataType) domain.TypeProcessor {
	switch schemaType {
	case jsonSchema.Array:
		return NewArrayProcessorWithFieldProcessor(fp.llmProvider, fp.promptBuilder, fp)
	case jsonSchema.Map:
		return NewMapProcessorWithFieldProcessor(fp.llmProvider, fp.promptBuilder, fp)
	case jsonSchema.Boolean:
		processor := NewBooleanProcessor(fp.llmProvider, fp.promptBuilder)
		// Propagate epstimic orchestrator if set
		if fp.epstimicOrchestrator != nil {
			processor.SetEpstimicOrchestrator(fp.epstimicOrchestrator)
		}
		return processor
	case jsonSchema.Number, jsonSchema.Integer:
		processor := NewNumberProcessor(fp.llmProvider, fp.promptBuilder)
		// Propagate epstimic orchestrator if set
		if fp.epstimicOrchestrator != nil {
			processor.SetEpstimicOrchestrator(fp.epstimicOrchestrator)
		}
		return processor
	default:
		processor := NewPrimitiveProcessor(fp.llmProvider, fp.promptBuilder)
		// Propagate epstimic orchestrator if set
		if fp.epstimicOrchestrator != nil {
			processor.SetEpstimicOrchestrator(fp.epstimicOrchestrator)
		}
		return processor
	}
}

// getOrderedKeys returns keys that need sequential processing and keys that can be parallel
func getOrderedKeys(schema *jsonSchema.Definition) ([]string, []string) {
	var remainingKeys []string
	allKeys := make(map[string]struct{})
	for key := range schema.Properties {
		allKeys[key] = struct{}{}
	}

	if schema.ProcessingOrder != nil {
		for _, key := range schema.ProcessingOrder {
			delete(allKeys, key)
		}
	}

	for key := range allKeys {
		remainingKeys = append(remainingKeys, key)
	}

	return schema.ProcessingOrder, remainingKeys
}

// evaluateScores uses the generator to score the generated content
func (fp *FieldProcessor) evaluateScores(
	result *domain.TaskResult,
	criteria *jsonSchema.ScoringCriteria,
	context *domain.ExecutionContext,
) (map[string]float64, error) {
	if fp.generator == nil {
		return nil, fmt.Errorf("generator not set, cannot evaluate scores")
	}

	// Build scoring schema
	properties := make(map[string]jsonSchema.Definition)
	for dimensionName, dimension := range criteria.Dimensions {
		var fieldType jsonSchema.DataType
		switch dimension.Type {
		case jsonSchema.ScoreNumeric:
			fieldType = jsonSchema.Number
		case jsonSchema.ScoreBoolean:
			fieldType = jsonSchema.Boolean
		default:
			fieldType = jsonSchema.String
		}

		instruction := dimension.Description
		if dimension.Scale != nil {
			instruction += fmt.Sprintf(" (Range: %d-%d)", dimension.Scale.Min, dimension.Scale.Max)
		}

		properties[dimensionName] = jsonSchema.Definition{
			Type:        fieldType,
			Instruction: instruction,
		}
	}

	scoringSchema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Properties:  properties,
		Instruction: "Evaluate the content according to the specified dimensions",
	}

	// Build evaluation prompt
	prompt := "Evaluate the following content:\n\n"
	prompt += fmt.Sprintf("%v\n\n", result.Value())
	prompt += "Provide scores for each dimension as requested."

	// Override model if specified
	if criteria.EvaluationModel != "" {
		scoringSchema.Model = criteria.EvaluationModel
	}

	// Generate scores
	request := domain.NewGenerationRequest(prompt, scoringSchema)
	scoreResult, err := fp.generator.Generate(request)
	if err != nil {
		return nil, fmt.Errorf("score generation failed: %w", err)
	}

	// Extract scores
	scoreData := scoreResult.Data()
	scores := make(map[string]float64)
	for key, value := range scoreData {
		if numValue, ok := toFloat64(value); ok {
			scores[key] = numValue
		}
	}

	// Calculate aggregate if needed
	if criteria.AggregationMethod != "" {
		aggregate := fp.calculateAggregate(scores, criteria)
		scores["_aggregate"] = aggregate
	}

	return scores, nil
}

// calculateAggregate computes the aggregate score
func (fp *FieldProcessor) calculateAggregate(scores map[string]float64, criteria *jsonSchema.ScoringCriteria) float64 {
	switch criteria.AggregationMethod {
	case jsonSchema.AggregateWeightedAverage:
		var sum, totalWeight float64
		for dimension, score := range scores {
			if dimDef, exists := criteria.Dimensions[dimension]; exists {
				weight := dimDef.Weight
				if weight == 0 {
					weight = 1.0 / float64(len(criteria.Dimensions))
				}
				sum += score * weight
				totalWeight += weight
			}
		}
		if totalWeight > 0 {
			return sum / totalWeight
		}
		return 0

	case jsonSchema.AggregateMinimum:
		min := float64(100)
		for _, score := range scores {
			if score < min {
				min = score
			}
		}
		return min

	case jsonSchema.AggregateMaximum:
		max := float64(0)
		for _, score := range scores {
			if score > max {
				max = score
			}
		}
		return max

	default:
		// Default to average
		var sum float64
		for _, score := range scores {
			sum += score
		}
		return sum / float64(len(scores))
	}
}

// attachScoresToResult attaches scores to the result's metadata
func (fp *FieldProcessor) attachScoresToResult(result *domain.TaskResult, scores map[string]float64) {
	if result == nil || result.Metadata() == nil {
		return
	}

	// Convert scores to Choice format for storage in metadata
	// We use a single Choice with the aggregate score
	if aggregateScore, hasAggregate := scores["_aggregate"]; hasAggregate {
		choice := domain.Choice{
			Score: int(aggregateScore),
		}
		result.Metadata().Choices = append(result.Metadata().Choices, choice)
	}
}
