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
package execution

import (
	"fmt"
	"log"
	"objectweaver/orchestration/jos/domain"
	"sort"
	"strings"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// DecisionProcessor wraps task results and evaluates decision points
// This is injected into CompositeTaskExecutor to handle post-generation decision logic
type DecisionProcessor struct {
	generator domain.Generator
}

func NewDecisionProcessor(generator domain.Generator) *DecisionProcessor {
	return &DecisionProcessor{
		generator: generator,
	}
}

// ProcessDecisionPoint evaluates a decision point after content generation
// Returns a slice of TaskResults: [original result, ...additional branch results]
// The additional results should be added as sibling fields in the parent object
// This processor isn't suited for streaming generation. Therefore, condintional logic should occur in a non streamed manner.
func (d *DecisionProcessor) ProcessDecisionPoint(
	task *domain.FieldTask,
	result *domain.TaskResult,
	context *domain.ExecutionContext,
) ([]*domain.TaskResult, error) {
	decisionPoint := task.Definition().DecisionPoint

	// No decision point - return original result only
	if decisionPoint == nil {
		log.Printf("[DecisionProcessor] No decision point for task %s, skipping", task.Key())
		return []*domain.TaskResult{result}, nil
	}

	// Generator not set - log warning and return original
	if d.generator == nil {
		log.Printf("[DecisionProcessor] Warning: generator not set, skipping decision point for %s", task.Key())
		return []*domain.TaskResult{result}, nil
	}

	log.Printf("[DecisionProcessor] Evaluating decision point '%s' for task %s", decisionPoint.Name, task.Key())

	// Sort branches by priority (highest first)
	sortedBranches := make([]jsonSchema.ConditionalBranch, len(decisionPoint.Branches))
	copy(sortedBranches, decisionPoint.Branches)
	sort.Slice(sortedBranches, func(i, j int) bool {
		return sortedBranches[i].Priority > sortedBranches[j].Priority
	})

	// Evaluate each branch in priority order
	for _, branch := range sortedBranches {
		matched, err := d.evaluateBranch(branch, task, result, context, decisionPoint)
		if err != nil {
			return nil, fmt.Errorf("branch evaluation failed for '%s': %w", branch.Name, err)
		}

		if matched {
			log.Printf("[DecisionProcessor] Branch '%s' matched, executing Then definition", branch.Name)
			branchResults, err := d.executeBranch(task, branch.Then, context)
			if err != nil {
				return nil, err
			}
			// Return original result + branch results
			allResults := append([]*domain.TaskResult{result}, branchResults...)
			return allResults, nil
		}
	}

	log.Printf("[DecisionProcessor] No branch matched, using original result")
	return []*domain.TaskResult{result}, nil
}

// evaluateBranch checks if all conditions in a branch are satisfied
func (d *DecisionProcessor) evaluateBranch(
	branch jsonSchema.ConditionalBranch,
	task *domain.FieldTask,
	originalResult *domain.TaskResult,
	context *domain.ExecutionContext,
	decisionPoint *jsonSchema.DecisionPoint,
) (bool, error) {
	if len(branch.Conditions) == 0 {
		return false, nil
	}

	// Build a schema to extract condition values using the generator
	conditionSchema := d.buildConditionEvaluationSchema(branch, decisionPoint, originalResult)

	// Create evaluation prompt
	prompt := d.buildEvaluationPrompt(decisionPoint, originalResult, branch)

	// Generate evaluation using the generator
	request := domain.NewGenerationRequest(prompt, conditionSchema)
	result, err := d.generator.Generate(request)
	if err != nil {
		return false, fmt.Errorf("condition evaluation generation failed: %w", err)
	}

	// Extract evaluation data
	evaluationData := result.Data()
	log.Println("[DecisionProcessor] Evaluation data:", evaluationData)
	if evaluationData == nil {
		return false, fmt.Errorf("evaluation result data is nil")
	}

	// Check all conditions (AND logic)
	for _, condition := range branch.Conditions {
		conditionMet, err := d.evaluateCondition(condition, evaluationData, context)
		if err != nil {
			return false, fmt.Errorf("condition evaluation error: %w", err)
		}
		if !conditionMet {
			log.Printf("[DecisionProcessor] Branch '%s' condition failed: %s %s %v",
				branch.Name, condition.Field, condition.Operator, condition.Value)
			return false, nil
		}
	}

	return true, nil
}

// buildConditionEvaluationSchema creates a schema to extract values needed for condition evaluation
func (d *DecisionProcessor) buildConditionEvaluationSchema(
	branch jsonSchema.ConditionalBranch,
	decisionPoint *jsonSchema.DecisionPoint,
	originalResult *domain.TaskResult,
) *jsonSchema.Definition {
	properties := make(map[string]jsonSchema.Definition)

	for _, condition := range branch.Conditions {
		// Determine the appropriate type for extraction
		fieldType := inferTypeFromValue(condition.Value)

		properties[condition.Field] = jsonSchema.Definition{
			Type:        fieldType,
			Instruction: fmt.Sprintf("Extract or evaluate the value for %s from the content", condition.Field),
		}
	}

	return &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Properties:  properties,
		Instruction: fmt.Sprintf("Evaluate the following conditions for branch '%s'", branch.Name),
	}
}

// buildEvaluationPrompt constructs the prompt for condition evaluation
func (d *DecisionProcessor) buildEvaluationPrompt(
	decisionPoint *jsonSchema.DecisionPoint,
	originalResult *domain.TaskResult,
	branch jsonSchema.ConditionalBranch,
) string {
	prompt := ""

	if decisionPoint.EvaluationPrompt != "" {
		prompt += decisionPoint.EvaluationPrompt + "\n\n"
	}

	prompt += "Content to evaluate:\n"
	prompt += fmt.Sprintf("%v\n\n", originalResult.Value())

	prompt += fmt.Sprintf("Evaluate conditions for branch '%s':\n", branch.Name)
	for _, condition := range branch.Conditions {
		prompt += fmt.Sprintf("- %s (for comparison: %s %v)\n", condition.Field, condition.Operator, condition.Value)
	}

	return prompt
}

// evaluateCondition checks if a single condition is satisfied
func (d *DecisionProcessor) evaluateCondition(
	condition jsonSchema.Condition,
	evaluationData map[string]interface{},
	context *domain.ExecutionContext,
) (bool, error) {
	// Get left-hand side value
	var lhs interface{}

	if condition.FieldPath != "" {
		// Extract from context using field path
		lhs = context.GeneratedValues()[condition.FieldPath]
	} else {
		// Get from evaluation data
		lhs = evaluationData[condition.Field]
	}

	// Get right-hand side value
	rhs := condition.Value

	// Evaluate based on operator
	return compareValues(lhs, condition.Operator, rhs)
}

// executeBranch executes the Then definition of a matching branch
// Returns an array of TaskResults representing the fields generated by the branch
func (d *DecisionProcessor) executeBranch(
	parentTask *domain.FieldTask,
	branchDef jsonSchema.Definition,
	context *domain.ExecutionContext,
) ([]*domain.TaskResult, error) {
	// Enhance context with SelectFields if specified
	log.Printf("BranchDef: %v", branchDef)
	log.Printf("Context %v", context.GeneratedValues()) //this is empty ie it doesn't contain the previsouly generated information

	    // Build the prompt starting with the instruction
    prompt := branchDef.Instruction
    if branchDef.OverridePrompt != nil {
        prompt = *branchDef.OverridePrompt
    }

    // Enhance prompt with SelectFields if specified
    if len(branchDef.SelectFields) > 0 {
        // Add selected field values directly to the prompt
        prompt += "\n\nContext from previous generation:\n"
        for _, fieldPath := range branchDef.SelectFields {
            log.Printf("[DecisionProcessor] Looking for field '%s' in context", fieldPath)
            if value, exists := context.GeneratedValues()[fieldPath]; exists {
                prompt += fmt.Sprintf("\n%s:\n%v\n", fieldPath, value)
                log.Printf("[DecisionProcessor] Added field '%s' to branch prompt (length: %d chars)", fieldPath, len(fmt.Sprintf("%v", value)))
            } else {
				log.Printf("[DecisionProcessor] Field '%s' not found in context", fieldPath)
			}
        }
    }
	
	request := domain.NewGenerationRequest(prompt, &branchDef)
	result, err := d.generator.Generate(request)
	if err != nil {
		return nil, fmt.Errorf("branch generation failed: %w", err)
	}

	// Convert generated data to TaskResults
	// If the branch returns an object, extract each property as a separate TaskResult
	resultData := result.Data()
	taskResults := make([]*domain.TaskResult, 0)

	metadata := domain.NewResultMetadata()
	if result.Metadata() != nil {
		metadata = result.Metadata()
	}

	// Result.Data() returns map[string]interface{} for objects
	// Create a TaskResult for each property in the object
	if len(resultData) > 0 {
		for key, value := range resultData {
			taskResult := domain.NewTaskResult(
				parentTask.ID()+"-"+key,
				key, // Use the property key name (e.g., "refined_content", "seo_description")
				value,
				metadata,
			)
			// Set the path to be at the same level as the parent task
			taskResult = taskResult.WithPath([]string{key})
			taskResults = append(taskResults, taskResult)

			log.Printf("[DecisionProcessor] Created branch result for field '%s'", key)
		}
	} else {
		// Empty result - return empty array
		log.Printf("[DecisionProcessor] Branch returned empty result")
	}

	return taskResults, nil
}

// Helper functions

func inferTypeFromValue(value interface{}) jsonSchema.DataType {
	switch value.(type) {
	case bool:
		return jsonSchema.Boolean
	case int, int32, int64:
		return jsonSchema.Integer
	case float32, float64:
		return jsonSchema.Number
	default:
		return jsonSchema.String
	}
}

func compareValues(lhs interface{}, operator jsonSchema.ComparisonOperator, rhs interface{}) (bool, error) {
	// Normalize operator to handle both short forms (gt, gte, etc.) and long forms (greater_than, etc.)
	normalizedOp := normalizeOperator(operator)

	switch normalizedOp {
	case jsonSchema.OpEqual:
		return fmt.Sprintf("%v", lhs) == fmt.Sprintf("%v", rhs), nil

	case jsonSchema.OpNotEqual:
		return fmt.Sprintf("%v", lhs) != fmt.Sprintf("%v", rhs), nil

	case jsonSchema.OpGreaterThan:
		lhsFloat, lhsOk := toFloat64(lhs)
		rhsFloat, rhsOk := toFloat64(rhs)
		if !lhsOk || !rhsOk {
			return false, fmt.Errorf("cannot compare non-numeric values with >")
		}
		return lhsFloat > rhsFloat, nil

	case jsonSchema.OpLessThan:
		lhsFloat, lhsOk := toFloat64(lhs)
		rhsFloat, rhsOk := toFloat64(rhs)
		if !lhsOk || !rhsOk {
			return false, fmt.Errorf("cannot compare non-numeric values with <")
		}
		return lhsFloat < rhsFloat, nil

	case jsonSchema.OpGreaterThanOrEqual:
		lhsFloat, lhsOk := toFloat64(lhs)
		rhsFloat, rhsOk := toFloat64(rhs)
		if !lhsOk || !rhsOk {
			return false, fmt.Errorf("cannot compare non-numeric values with >=")
		}
		return lhsFloat >= rhsFloat, nil

	case jsonSchema.OpLessThanOrEqual:
		lhsFloat, lhsOk := toFloat64(lhs)
		rhsFloat, rhsOk := toFloat64(rhs)
		if !lhsOk || !rhsOk {
			return false, fmt.Errorf("cannot compare non-numeric values with <=")
		}
		return lhsFloat <= rhsFloat, nil

	case jsonSchema.OpContains:
		lhsStr := fmt.Sprintf("%v", lhs)
		rhsStr := fmt.Sprintf("%v", rhs)
		return strings.Contains(lhsStr, rhsStr), nil

	default:
		return false, fmt.Errorf("unsupported operator: %s (normalized to: %s)", operator, normalizedOp)
	}
}

// normalizeOperator converts various operator formats to the canonical form
func normalizeOperator(op jsonSchema.ComparisonOperator) jsonSchema.ComparisonOperator {
	opStr := string(op)

	// Map of alternative operator formats to canonical forms
	operatorMap := map[string]jsonSchema.ComparisonOperator{
		// Short forms (canonical)
		"eq":       jsonSchema.OpEqual,
		"neq":      jsonSchema.OpNotEqual,
		"gt":       jsonSchema.OpGreaterThan,
		"lt":       jsonSchema.OpLessThan,
		"gte":      jsonSchema.OpGreaterThanOrEqual,
		"lte":      jsonSchema.OpLessThanOrEqual,
		"in":       jsonSchema.OpIn,
		"nin":      jsonSchema.OpNotIn,
		"contains": jsonSchema.OpContains,

		// Long forms (snake_case)
		"equal":                 jsonSchema.OpEqual,
		"not_equal":             jsonSchema.OpNotEqual,
		"greater_than":          jsonSchema.OpGreaterThan,
		"less_than":             jsonSchema.OpLessThan,
		"greater_than_or_equal": jsonSchema.OpGreaterThanOrEqual,
		"less_than_or_equal":    jsonSchema.OpLessThanOrEqual,

		// Alternative formats
		"==":                 jsonSchema.OpEqual,
		"!=":                 jsonSchema.OpNotEqual,
		">":                  jsonSchema.OpGreaterThan,
		"<":                  jsonSchema.OpLessThan,
		">=":                 jsonSchema.OpGreaterThanOrEqual,
		"<=":                 jsonSchema.OpLessThanOrEqual,
		"equals":             jsonSchema.OpEqual,
		"greaterThan":        jsonSchema.OpGreaterThan,
		"lessThan":           jsonSchema.OpLessThan,
		"greaterThanOrEqual": jsonSchema.OpGreaterThanOrEqual,
		"lessThanOrEqual":    jsonSchema.OpLessThanOrEqual,
	}

	// Convert to lowercase for case-insensitive matching
	opLower := strings.ToLower(opStr)

	if canonical, exists := operatorMap[opLower]; exists {
		return canonical
	}

	// Return original if no mapping found
	return op
}
