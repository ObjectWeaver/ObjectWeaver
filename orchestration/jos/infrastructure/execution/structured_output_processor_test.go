package execution

import (
	"fmt"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

func TestDefinitionToJSONSchema_Object(t *testing.T) {
	def := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Extract person details",
		Properties: map[string]jsonSchema.Definition{
			"name": {
				Type:        jsonSchema.String,
				Instruction: "The person's full name",
			},
			"age": {
				Type:        jsonSchema.Integer,
				Instruction: "The person's age",
			},
			"employed": {
				Type:        jsonSchema.Boolean,
				Instruction: "Whether the person is employed",
			},
		},
	}

	schema := DefinitionToJSONSchema(def)

	if schema["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	if len(props) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(props))
	}

	nameSchema, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatal("Expected name to be a map")
	}
	if nameSchema["type"] != "string" {
		t.Errorf("Expected name type 'string', got %v", nameSchema["type"])
	}

	ageSchema, ok := props["age"].(map[string]any)
	if !ok {
		t.Fatal("Expected age to be a map")
	}
	if ageSchema["type"] != "integer" {
		t.Errorf("Expected age type 'integer', got %v", ageSchema["type"])
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required to be a string slice")
	}
	if len(required) != 3 {
		t.Errorf("Expected 3 required fields, got %d", len(required))
	}
}

func TestDefinitionToJSONSchema_Array(t *testing.T) {
	def := &jsonSchema.Definition{
		Type:        jsonSchema.Array,
		Instruction: "List of items",
		Items: &jsonSchema.Definition{
			Type: jsonSchema.Object,
			Properties: map[string]jsonSchema.Definition{
				"title": {
					Type:        jsonSchema.String,
					Instruction: "Item title",
				},
				"value": {
					Type:        jsonSchema.Number,
					Instruction: "Item value",
				},
			},
		},
	}

	schema := DefinitionToJSONSchema(def)

	if schema["type"] != "array" {
		t.Errorf("Expected type 'array', got %v", schema["type"])
	}

	items, ok := schema["items"].(map[string]any)
	if !ok {
		t.Fatal("Expected items to be a map")
	}
	if items["type"] != "object" {
		t.Errorf("Expected items type 'object', got %v", items["type"])
	}

	props, ok := items["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected items properties to be a map")
	}
	if len(props) != 2 {
		t.Errorf("Expected 2 properties in items, got %d", len(props))
	}
}

func TestDefinitionToJSONSchema_Primitives(t *testing.T) {
	tests := []struct {
		dataType jsonSchema.DataType
		expected string
	}{
		{jsonSchema.String, "string"},
		{jsonSchema.Number, "number"},
		{jsonSchema.Integer, "integer"},
		{jsonSchema.Boolean, "boolean"},
	}

	for _, tt := range tests {
		def := &jsonSchema.Definition{
			Type:        tt.dataType,
			Instruction: "test instruction",
		}
		schema := DefinitionToJSONSchema(def)
		if schema["type"] != tt.expected {
			t.Errorf("For %s: expected type '%s', got %v", tt.dataType, tt.expected, schema["type"])
		}
		if schema["description"] != "test instruction" {
			t.Errorf("For %s: expected description 'test instruction', got %v", tt.dataType, schema["description"])
		}
	}
}

func TestParseStructuredResponse_Object(t *testing.T) {
	def := &jsonSchema.Definition{Type: jsonSchema.Object}
	response := `{"name": "Alice", "age": 30}`

	result, err := parseStructuredResponse(response, def)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}
	if m["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", m["name"])
	}
}

func TestParseStructuredResponse_Array(t *testing.T) {
	def := &jsonSchema.Definition{Type: jsonSchema.Array}
	response := `[{"name": "Alice"}, {"name": "Bob"}]`

	result, err := parseStructuredResponse(response, def)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("Expected slice, got %T", result)
	}
	if len(arr) != 2 {
		t.Errorf("Expected 2 items, got %d", len(arr))
	}
}

func TestParseStructuredResponse_CodeBlocks(t *testing.T) {
	def := &jsonSchema.Definition{Type: jsonSchema.Object}
	response := "```json\n{\"name\": \"Alice\"}\n```"

	result, err := parseStructuredResponse(response, def)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}
	if m["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", m["name"])
	}
}

func TestIsContextSizeError(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected bool
	}{
		{"context length exceeded", true},
		{"maximum context reached", true},
		{"too many tokens in request", true},
		{"request too large for model", true},
		{"413 Payload Too Large", true},
		{"input too long for model", true},
		{"normal error", false},
		{"rate limit exceeded", false},
	}

	for _, tt := range tests {
		err := &ContextSizeError{Err: fmt.Errorf("%s", tt.errMsg), Field: "test"}
		_ = err // just testing the detection function
		result := isContextSizeError(fmt.Errorf("%s", tt.errMsg))
		if result != tt.expected {
			t.Errorf("For '%s': expected %v, got %v", tt.errMsg, tt.expected, result)
		}
	}
}

func TestDefinitionToJSONSchema_NestedObject(t *testing.T) {
	def := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"address": {
				Type: jsonSchema.Object,
				Properties: map[string]jsonSchema.Definition{
					"street": {Type: jsonSchema.String, Instruction: "Street name"},
					"city":   {Type: jsonSchema.String, Instruction: "City name"},
				},
			},
			"tags": {
				Type: jsonSchema.Array,
				Items: &jsonSchema.Definition{
					Type: jsonSchema.String,
				},
			},
		},
	}

	schema := DefinitionToJSONSchema(def)

	props := schema["properties"].(map[string]any)
	addr := props["address"].(map[string]any)
	if addr["type"] != "object" {
		t.Errorf("Expected nested object type, got %v", addr["type"])
	}
	addrProps := addr["properties"].(map[string]any)
	if len(addrProps) != 2 {
		t.Errorf("Expected 2 address properties, got %d", len(addrProps))
	}

	tags := props["tags"].(map[string]any)
	if tags["type"] != "array" {
		t.Errorf("Expected array type for tags, got %v", tags["type"])
	}
	tagItems := tags["items"].(map[string]any)
	if tagItems["type"] != "string" {
		t.Errorf("Expected string items for tags, got %v", tagItems["type"])
	}
}
