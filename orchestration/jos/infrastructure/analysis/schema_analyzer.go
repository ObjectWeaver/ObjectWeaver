package analysis

import (
	"fmt"
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// DefaultSchemaAnalyzer analyzes JSON schemas
type DefaultSchemaAnalyzer struct{}

func NewDefaultSchemaAnalyzer() *DefaultSchemaAnalyzer {
	return &DefaultSchemaAnalyzer{}
}

// Analyze analyzes the schema structure
func (a *DefaultSchemaAnalyzer) Analyze(schema *jsonSchema.Definition) (*domain.SchemaAnalysis, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	fields, err := a.ExtractFields(schema)
	if err != nil {
		return nil, err
	}

	return &domain.SchemaAnalysis{
		Fields:           fields,
		TotalFieldCount:  len(fields),
		MaxDepth:         a.calculateMaxDepth(fields),
		HasNestedObjects: a.hasNestedObjects(fields),
	}, nil
}

// ExtractFields extracts all fields from the schema
func (a *DefaultSchemaAnalyzer) ExtractFields(schema *jsonSchema.Definition) ([]*domain.FieldDefinition, error) {
	if schema == nil || schema.Properties == nil {
		return []*domain.FieldDefinition{}, nil
	}

	fields := make([]*domain.FieldDefinition, 0)

	for key, childDef := range schema.Properties {
		field := &domain.FieldDefinition{
			Key:        key,
			Definition: &childDef,
			Parent:     nil,
			Required:   a.isRequired(key, schema),
		}
		fields = append(fields, field)
	}

	return fields, nil
}

// DetermineProcessingOrder determines the order in which fields should be processed
// Uses ProcessingOrder from schema to establish dependencies between fields.
//
// ProcessingOrder workflow:
//  1. Each field's Definition.ProcessingOrder contains a list of field keys that must be
//     processed BEFORE this field can be processed (dependencies)
//  2. Fields with no ProcessingOrder or empty ProcessingOrder have no dependencies
//  3. The strategy (Sequential/Parallel/DependencyAware) will use these dependencies
//     to determine optimal execution order
//
// Example:
//
//	Field "address" has ProcessingOrder: ["city", "country"]
//	-> "city" and "country" must be generated before "address" can be generated
//	-> This allows "address" generation to access values from "city" and "country"
//
// The DependencyAwareStrategy will use topological sorting to:
// - Execute independent fields in parallel
// - Ensure dependencies are satisfied before dependent fields execute
// - Optimize overall execution time while respecting constraints
func (a *DefaultSchemaAnalyzer) DetermineProcessingOrder(fields []*domain.FieldDefinition) ([]*domain.FieldTask, error) {
	tasks := make([]*domain.FieldTask, 0, len(fields))
	fieldMap := make(map[string]*domain.FieldDefinition)

	// Create a map for quick field lookup
	for _, field := range fields {
		fieldMap[field.Key] = field
	}

	// Create tasks with dependencies based on ProcessingOrder
	for _, field := range fields {
		task := domain.NewFieldTask(field.Key, field.Definition, nil)

		// Add dependencies from ProcessingOrder
		// ProcessingOrder contains keys of fields that must be processed first
		if len(field.Definition.ProcessingOrder) > 0 {
			for _, depKey := range field.Definition.ProcessingOrder {
				// Only add as dependency if the field exists in the schema
				// This prevents errors from typos or removed fields
				if _, exists := fieldMap[depKey]; exists {
					task = task.WithDependency(depKey)
				}
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (a *DefaultSchemaAnalyzer) calculateMaxDepth(fields []*domain.FieldDefinition) int {
	maxDepth := 1
	for _, field := range fields {
		if field.Definition.Type == jsonSchema.Object && field.Definition.Properties != nil {
			depth := a.calculateDepthRecursive(field.Definition, 1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}
	return maxDepth
}

func (a *DefaultSchemaAnalyzer) calculateDepthRecursive(def *jsonSchema.Definition, currentDepth int) int {
	if def.Properties == nil {
		return currentDepth
	}

	maxDepth := currentDepth
	for _, child := range def.Properties {
		if child.Type == jsonSchema.Object {
			depth := a.calculateDepthRecursive(&child, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}
	return maxDepth
}

func (a *DefaultSchemaAnalyzer) hasNestedObjects(fields []*domain.FieldDefinition) bool {
	for _, field := range fields {
		if field.Definition.Type == jsonSchema.Object {
			return true
		}
	}
	return false
}

func (a *DefaultSchemaAnalyzer) isRequired(key string, schema *jsonSchema.Definition) bool {
	// Note: jsonSchema.Definition doesn't have a Required field
	// This could be added in the future or derived from other fields
	return false
}
