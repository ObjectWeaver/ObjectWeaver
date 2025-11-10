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
	"os"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// RecursiveLoopProcessor handles iterative refinement of field generation
type RecursiveLoopProcessor struct {
	baseProcessor     domain.TypeProcessor
	generator         domain.Generator
	decisionProcessor *DecisionProcessor
}

// iterationResult holds the result and scores for a single iteration
type iterationResult struct {
	result    *domain.TaskResult
	scores    map[string]float64
	iteration int
}

func NewRecursiveLoopProcessor(
	baseProcessor domain.TypeProcessor,
	generator domain.Generator,
	decisionProcessor *DecisionProcessor,
) *RecursiveLoopProcessor {
	return &RecursiveLoopProcessor{
		baseProcessor:     baseProcessor,
		generator:         generator,
		decisionProcessor: decisionProcessor,
	}
}

func (r *RecursiveLoopProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	// Delegate to base processor
	return r.baseProcessor.CanProcess(schemaType)
}

func (r *RecursiveLoopProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	loop := task.Definition().RecursiveLoop
	verboseLogs := os.Getenv("VERBOSE") == "true"

	if verboseLogs {
		log.Printf("[RecursiveLoop] Starting for task %s (max iterations: %d)", task.Key(), loop.MaxIterations)
	}

	var results []iterationResult

	for i := 0; i < loop.MaxIterations; i++ {
		if verboseLogs {
			log.Printf("[RecursiveLoop] Iteration %d/%d for task %s", i+1, loop.MaxIterations, task.Key())
		}

		// Generate iteration
		result, err := r.baseProcessor.Process(task, context)
		if err != nil {
			return nil, fmt.Errorf("iteration %d failed: %w", i+1, err)
		}

		iterResult := iterationResult{
			result:    result,
			iteration: i + 1,
		}

		// Score if criteria exist
		if task.Definition().ScoringCriteria != nil {
			scores, err := r.evaluateScores(result, task.Definition().ScoringCriteria, context)
			if err != nil {
				log.Printf("[RecursiveLoop] Warning: scoring failed for iteration %d: %v", i+1, err)
			} else {
				iterResult.scores = scores
				if verboseLogs {
					log.Printf("[RecursiveLoop] Iteration %d scores: %v", i+1, scores)
				}
			}
		}

		results = append(results, iterResult)

		// Check termination using DecisionPoint logic!
		if loop.TerminationPoint != nil {
			shouldStop, err := r.shouldTerminate(loop.TerminationPoint, result, context, task)
			if err != nil {
				log.Printf("[RecursiveLoop] Warning: termination check failed: %v", err)
			} else if shouldStop {
				if verboseLogs {
					log.Printf("[RecursiveLoop] Termination condition met at iteration %d", i+1)
				}
				break
			}
		}

		// Enhance context with feedback for next iteration
		if i < loop.MaxIterations-1 {
			r.enhanceContextWithFeedback(loop, iterResult, context)
		}
	}

	if verboseLogs {
		log.Printf("[RecursiveLoop] Completed %d iterations for task %s", len(results), task.Key())
	}

	// Select best result based on strategy
	return r.selectResult(results, loop.Selection, task.Key())
}

// shouldTerminate uses DecisionPoint logic to check if we should stop iterating
func (r *RecursiveLoopProcessor) shouldTerminate(
	terminationPoint *jsonSchema.DecisionPoint,
	currentResult *domain.TaskResult,
	context *domain.ExecutionContext,
	task *domain.FieldTask,
) (bool, error) {
	// If no branches defined, don't terminate
	if len(terminationPoint.Branches) == 0 {
		return false, nil
	}

	// If no decision processor available, don't terminate
	if r.decisionProcessor == nil {
		return false, nil
	}

	// Use the DecisionProcessor's branch evaluation logic
	// We iterate through branches and check if ANY match (OR logic across branches)
	for _, branch := range terminationPoint.Branches {
		matched, err := r.decisionProcessor.evaluateBranch(
			branch,
			task,
			currentResult,
			context,
			terminationPoint,
		)
		if err != nil {
			return false, fmt.Errorf("termination branch evaluation failed: %w", err)
		}

		if matched {
			return true, nil // Stop on first matching branch
		}
	}

	return false, nil
}

// evaluateScores uses the generator to score the content
func (r *RecursiveLoopProcessor) evaluateScores(
	result *domain.TaskResult,
	criteria *jsonSchema.ScoringCriteria,
	context *domain.ExecutionContext,
) (map[string]float64, error) {
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
	scoreResult, err := r.generator.Generate(request)
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
		aggregate := r.calculateAggregate(scores, criteria)
		scores["_aggregate"] = aggregate
	}

	return scores, nil
}

// calculateAggregate computes the aggregate score
func (r *RecursiveLoopProcessor) calculateAggregate(scores map[string]float64, criteria *jsonSchema.ScoringCriteria) float64 {
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

// enhanceContextWithFeedback adds feedback to context for next iteration
func (r *RecursiveLoopProcessor) enhanceContextWithFeedback(
	loop *jsonSchema.RecursiveLoop,
	previousIter iterationResult,
	context *domain.ExecutionContext,
) {
	if loop.FeedbackPrompt == "" {
		return
	}

	feedback := loop.FeedbackPrompt + "\n\n"
	feedback += "Previous attempt:\n"
	feedback += fmt.Sprintf("%v\n\n", previousIter.result.Value())

	// Add scores if available
	if len(previousIter.scores) > 0 {
		feedback += "Previous scores:\n"
		for dimension, score := range previousIter.scores {
			if dimension != "_aggregate" {
				feedback += fmt.Sprintf("- %s: %.2f\n", dimension, score)
			}
		}
		feedback += "\n"
	}

	context.PromptContext().AddPrompt(feedback)
}

// selectResult chooses the final result based on selection strategy
func (r *RecursiveLoopProcessor) selectResult(
	results []iterationResult,
	strategy jsonSchema.SelectionStrategy,
	taskKey string,
) (*domain.TaskResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to select from")
	}

	switch strategy {
	case jsonSchema.SelectHighestScore:
		return r.selectByScore(results, true), nil

	case jsonSchema.SelectLowestScore:
		return r.selectByScore(results, false), nil

	case jsonSchema.SelectLatest:
		return results[len(results)-1].result, nil

	case jsonSchema.SelectFirst:
		return results[0].result, nil

	case jsonSchema.SelectAll:
		// Combine all results into an array
		allValues := make([]interface{}, len(results))
		for i, iter := range results {
			allValues[i] = iter.result.Value()
		}
		return domain.NewTaskResult(results[0].result.TaskID(), taskKey, allValues, domain.NewResultMetadata()), nil

	default:
		return results[len(results)-1].result, nil
	}
}

// selectByScore finds the result with highest/lowest aggregate score
func (r *RecursiveLoopProcessor) selectByScore(results []iterationResult, highest bool) *domain.TaskResult {
	bestIter := results[0]
	bestScore := float64(0)
	hasScore := false

	for _, iter := range results {
		if len(iter.scores) == 0 {
			continue
		}

		aggregateScore, exists := iter.scores["_aggregate"]
		if !exists {
			// Calculate average if no aggregate
			var sum float64
			count := 0
			for key, val := range iter.scores {
				if key != "_aggregate" {
					sum += val
					count++
				}
			}
			if count > 0 {
				aggregateScore = sum / float64(count)
			}
		}

		if !hasScore {
			bestScore = aggregateScore
			bestIter = iter
			hasScore = true
		} else if (highest && aggregateScore > bestScore) || (!highest && aggregateScore < bestScore) {
			bestScore = aggregateScore
			bestIter = iter
		}
	}

	return bestIter.result
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}
