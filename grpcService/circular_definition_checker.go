package grpcService

import (
	"github.com/objectweaver/go-sdk/jsonSchema"
	"objectweaver/checks"
)

// DefaultCircularDefinitionChecker is the default implementation
type DefaultCircularDefinitionChecker struct{}

// NewDefaultCircularDefinitionChecker creates a new checker
func NewDefaultCircularDefinitionChecker() CircularDefinitionChecker {
	return &DefaultCircularDefinitionChecker{}
}

// Check checks for circular definitions in the schema
func (c *DefaultCircularDefinitionChecker) Check(definition *jsonSchema.Definition) bool {
	return checks.CheckCircularDefinitions(definition)
}
