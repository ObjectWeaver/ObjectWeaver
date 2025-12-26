package service

import (
	"testing"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

func TestConvertRawOutputToDefinition(t *testing.T) {
	raw := &TtwRawOutput{
		Analysis:              "User with address",
		RootType:              "object",
		DefinitionInstruction: "Generate a user",
		Fields: []FieldDescriptor{
			{
				FieldName:       "name",
				FieldType:       "string",
				DataInstruction: "User's name",
				IsComplex:       false,
			},
			{
				FieldName:       "address",
				FieldType:       "object",
				DataInstruction: "User's address",
				IsComplex:       true,
				NestedFields: []FieldDescriptor{
					{
						FieldName:       "street",
						FieldType:       "string",
						DataInstruction: "Street name",
						IsComplex:       false,
					},
				},
			},
		},
	}

	def := convertRawOutputToDefinition(raw)

	if def.Type != jsonSchema.Object {
		t.Errorf("Expected root type object, got %v", def.Type)
	}

	if len(def.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(def.Properties))
	}

	address, ok := def.Properties["address"]
	if !ok {
		t.Fatal("Expected address property")
	}

	if address.Type != jsonSchema.Object {
		t.Errorf("Expected address type object, got %v", address.Type)
	}

	if len(address.Properties) != 1 {
		t.Errorf("Expected 1 nested property in address, got %d", len(address.Properties))
	}

	street, ok := address.Properties["street"]
	if !ok {
		t.Fatal("Expected street property in address")
	}

	if street.Type != jsonSchema.String {
		t.Errorf("Expected street type string, got %v", street.Type)
	}
}

func TestConvertRawOutputToDefinition_Sanitization(t *testing.T) {
	raw := &TtwRawOutput{
		Analysis:              "User with address\n",
		RootType:              "object\n",
		DefinitionInstruction: "Generate a user\n",
		Fields: []FieldDescriptor{
			{
				FieldName:       "name\n",
				FieldType:       "string\n",
				DataInstruction: "User's name\n",
				IsComplex:       false,
			},
			{
				FieldName:       "address\n",
				FieldType:       "object\n",
				DataInstruction: "User's address\n",
				IsComplex:       true,
				NestedFields: []FieldDescriptor{
					{
						FieldName:       "street\n",
						FieldType:       "string\n",
						DataInstruction: "Street name\n",
						IsComplex:       false,
					},
				},
			},
		},
	}

	def := convertRawOutputToDefinition(raw)

	// Check root instruction
	if def.Instruction != "Generate a user" {
		t.Errorf("Expected 'Generate a user', got '%s'", def.Instruction)
	}

	// Check field names
	if _, ok := def.Properties["name"]; !ok {
		t.Error("Expected 'name' property (sanitized)")
	}
	if _, ok := def.Properties["name\n"]; ok {
		t.Error("Did not expect 'name\\n' property")
	}

	address, ok := def.Properties["address"]
	if !ok {
		t.Fatal("Expected 'address' property (sanitized)")
	}

	// Check nested field names
	if _, ok := address.Properties["street"]; !ok {
		t.Error("Expected 'street' property in address (sanitized)")
	}

	// Check nested instruction
	street := address.Properties["street"]
	if street.Instruction != "Street name" {
		t.Errorf("Expected 'Street name', got '%s'", street.Instruction)
	}
}

func TestConvertRawOutputToDefinition_HeuristicNesting(t *testing.T) {
	raw := &TtwRawOutput{
		Analysis:              "User with address",
		RootType:              "object",
		DefinitionInstruction: "Generate a user",
		Fields: []FieldDescriptor{
			{
				FieldName:       "name",
				FieldType:       "string",
				DataInstruction: "User's name",
				IsComplex:       false,
			},
			{
				FieldName:       "address",
				FieldType:       "object",
				DataInstruction: "User's address",
				IsComplex:       true,
				NestedAnalysis:  "street, city", // Hint that these should be nested
				NestedFields:    nil,            // LLM was lazy and didn't nest them here
			},
			{
				FieldName:       "street",
				FieldType:       "string",
				DataInstruction: "Street name",
				IsComplex:       false,
			},
			{
				FieldName:       "city",
				FieldType:       "string",
				DataInstruction: "City name",
				IsComplex:       false,
			},
		},
	}

	def := convertRawOutputToDefinition(raw)

	// Root should only have 'name' and 'address'
	if len(def.Properties) != 2 {
		t.Errorf("Expected 2 root properties, got %d: %v", len(def.Properties), def.Properties)
	}

	if _, ok := def.Properties["name"]; !ok {
		t.Error("Missing 'name' at root")
	}
	address, ok := def.Properties["address"]
	if !ok {
		t.Fatal("Missing 'address' at root")
	}

	// Address should have 'street' and 'city'
	if len(address.Properties) != 2 {
		t.Errorf("Expected 2 nested properties in address, got %d", len(address.Properties))
	}

	if _, ok := address.Properties["street"]; !ok {
		t.Error("Missing 'street' inside address")
	}
	if _, ok := address.Properties["city"]; !ok {
		t.Error("Missing 'city' inside address")
	}
}

func TestConvertRawOutputToDefinition_SameNameParentChild(t *testing.T) {
	// This test simulates a case where Gemini uses the same name for a parent and a child
	// e.g. fieldName: "street" (object) and fieldName: "street" (string)
	raw := &TtwRawOutput{
		Analysis:              "User with address",
		RootType:              "object",
		DefinitionInstruction: "Generate a user",
		Fields: []FieldDescriptor{
			{
				FieldName:       "street", // Gemini used 'street' as the name for the address object
				FieldType:       "object",
				DataInstruction: "Address details",
				IsComplex:       true,
				NestedAnalysis:  "street, city", // It claims 'street' as a child
			},
			{
				FieldName:       "street", // This is the actual street string
				FieldType:       "string",
				DataInstruction: "Street name",
				IsComplex:       false,
			},
			{
				FieldName:       "city",
				FieldType:       "string",
				DataInstruction: "City name",
				IsComplex:       false,
			},
		},
	}

	def := convertRawOutputToDefinition(raw)

	// The root should have 'street' (the object)
	streetObj, ok := def.Properties["street"]
	if !ok {
		t.Fatal("Expected 'street' (object) at root")
	}
	if streetObj.Type != jsonSchema.Object {
		t.Errorf("Expected 'street' at root to be an object, got %s", streetObj.Type)
	}

	// The 'street' object should have 'street' (the string) and 'city' as properties
	if _, ok := streetObj.Properties["street"]; !ok {
		t.Error("Expected 'street' (string) inside 'street' (object)")
	}
	if streetObj.Properties["street"].Type != jsonSchema.String {
		t.Errorf("Expected nested 'street' to be a string, got %s", streetObj.Properties["street"].Type)
	}
	if _, ok := streetObj.Properties["city"]; !ok {
		t.Error("Expected 'city' inside 'street' (object)")
	}

	// Ensure 'city' was removed from root
	if _, ok := def.Properties["city"]; ok {
		t.Error("Expected 'city' to be removed from root")
	}
}

func TestConvertRawOutputToDefinition_LazyGemini(t *testing.T) {
	raw := &TtwRawOutput{
		Analysis:              "name, email, bio, address",
		RootType:              "object",
		DefinitionInstruction: "Generate a user profile",
		Fields: []FieldDescriptor{
			{
				FieldName:       "name",
				FieldType:       "string",
				DataInstruction: "User's name",
				IsComplex:       false,
			},
			{
				FieldName:       "email",
				FieldType:       "string",
				DataInstruction: "User's email",
				IsComplex:       false,
			},
			{
				FieldName:       "bio",
				FieldType:       "string",
				DataInstruction: "User's bio",
				IsComplex:       false,
			},
			{
				FieldName:       "fieldName", // Hallucinated generic name
				FieldType:       "object",
				DataInstruction: "Address details",
				IsComplex:       false, // Incorrectly false
				NestedAnalysis:  "street, city, postalCode",
				NestedFields:    nil,
			},
			{
				FieldName:       "street",
				FieldType:       "string",
				DataInstruction: "Street name",
				IsComplex:       false,
			},
			{
				FieldName:       "city",
				FieldType:       "string",
				DataInstruction: "City name",
				IsComplex:       false,
			},
			{
				FieldName:       "postalCode",
				FieldType:       "string",
				DataInstruction: "Postal code",
				IsComplex:       false,
			},
		},
	}

	def := convertRawOutputToDefinition(raw)

	// Should have 4 root properties: name, email, bio, address
	if len(def.Properties) != 4 {
		t.Errorf("Expected 4 root properties, got %d: %v", len(def.Properties), def.Properties)
	}

	// Check if 'fieldName' was recovered to 'address'
	address, ok := def.Properties["address"]
	if !ok {
		t.Fatal("Missing 'address' at root (should have been recovered from 'fieldName')")
	}

	// Check if children were nested despite isComplex=false
	if len(address.Properties) != 3 {
		t.Errorf("Expected 3 nested properties in address, got %d", len(address.Properties))
	}

	if _, ok := address.Properties["street"]; !ok {
		t.Error("Missing 'street' inside address")
	}
}
