package domain

import "github.com/objectweaver/go-sdk/jsonSchema"

// SchemaAnalyzer - Analyzes and breaks down schemas into processable components
//
// Implementation: infrastructure/analysis/schema_analyzer.go (DefaultSchemaAnalyzer)
// Created by: factory/generator_factory.go:createAnalyzer()
// Used by: All Generator implementations
//
// Responsibilities:
//   - Parse JSON schema into FieldDefinitions
//   - Calculate schema metrics (depth, complexity)
//   - Determine field dependencies from ProcessingOrder
//   - Create FieldTasks with dependency information
type SchemaAnalyzer interface {
	Analyze(schema *jsonSchema.Definition) (*SchemaAnalysis, error)
	ExtractFields(schema *jsonSchema.Definition) ([]*FieldDefinition, error)
	DetermineProcessingOrder(fields []*FieldDefinition) ([]*FieldTask, error)
}

// SchemaAnalysis represents the analyzed schema structure
type SchemaAnalysis struct {
	Fields           []*FieldDefinition
	TotalFieldCount  int
	MaxDepth         int
	HasNestedObjects bool
}

// FieldDefinition represents a field in the schema
type FieldDefinition struct {
	Key        string
	Definition *jsonSchema.Definition
	Parent     *FieldDefinition
	Required   bool
}