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
package checks

import (
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
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
