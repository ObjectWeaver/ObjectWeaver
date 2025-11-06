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
}