package checks

import (
	"testing"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
)

func TestCheckCircularDefinitions_NoCycle(t *testing.T) {
	def := &jsonSchema.Definition{
		Type: "object",
		Properties: map[string]jsonSchema.Definition{
			"name": {Type: "string"},
			"age":  {Type: "number"},
		},
	}
	if CheckCircularDefinitions(def) {
		t.Error("Expected no circular definition")
	}
}

func TestCheckCircularDefinitions_SelfCycle(t *testing.T) {
	def := &jsonSchema.Definition{
		Type: "object",
	}
	def.Items = def
	if !CheckCircularDefinitions(def) {
		t.Error("Expected circular definition")
	}
}

func TestCheckCircularDefinitions_TwoNodeCycle(t *testing.T) {
	def1 := &jsonSchema.Definition{
		Type: "array",
	}
	def2 := &jsonSchema.Definition{
		Type: "object",
	}
	def1.Items = def2
	def2.Items = def1
	if !CheckCircularDefinitions(def1) {
		t.Error("Expected circular definition")
	}
}

func TestCheckCircularDefinitions_Nil(t *testing.T) {
	if CheckCircularDefinitions(nil) {
		t.Error("Expected no circular definition for nil")
	}
}

func TestCheckCircularDefinitions_Empty(t *testing.T) {
	def := &jsonSchema.Definition{}
	if CheckCircularDefinitions(def) {
		t.Error("Expected no circular definition for empty definition")
	}
}

func TestCheckCircularDefinitions_NestedNoCycle(t *testing.T) {
	def := &jsonSchema.Definition{
		Type: "object",
		Properties: map[string]jsonSchema.Definition{
			"items": {
				Type:  "array",
				Items: &jsonSchema.Definition{Type: "string"},
			},
		},
	}
	if CheckCircularDefinitions(def) {
		t.Error("Expected no circular definition")
	}
}
