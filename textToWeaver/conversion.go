package texttoweaver

import (
	"fmt"
	"objectweaver/logger"
	"strings"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// ConvertRawOutputToDefinition transforms the LLM output into a proper jsonSchema.Definition
// originalPrompt is the user's original request, used to make instructions more contextual
func ConvertRawOutputToDefinition(output *TtwRawOutput, originalPrompt string) *jsonSchema.Definition {
	rootType := jsonSchema.Object
	if strings.TrimSpace(output.RootType) == "array" {
		rootType = jsonSchema.Array
	}

	def := &jsonSchema.Definition{
		Type:        rootType,
		Instruction: strings.TrimSpace(output.DefinitionInstruction),
		Properties:  make(map[string]jsonSchema.Definition),
	}

	// FALLBACK: If fields array is empty OR malformed, parse structuralAnalysis directly
	// Malformed = all fields have same name, or field count doesn't match analysis
	shouldUseFallback := false
	if len(output.Fields) == 0 {
		shouldUseFallback = true
	} else if output.StructuralAnalysis != "" {
		// Check if fields are malformed (e.g., all have the same name)
		nameCount := make(map[string]int)
		for _, f := range output.Fields {
			nameCount[strings.TrimSpace(f.FieldName)]++
		}
		// If more than half have the same name, it's malformed
		for _, count := range nameCount {
			if count > len(output.Fields)/2 {
				logger.Printf("[TextToWeaver] Fields appear malformed (duplicate names), using fallback")
				shouldUseFallback = true
				break
			}
		}
	}

	if shouldUseFallback && output.StructuralAnalysis != "" {
		logger.Printf("[TextToWeaver] Using fallback parser for structuralAnalysis: %s", output.StructuralAnalysis)
		def.Properties = parseStructuralAnalysis(output.StructuralAnalysis, originalPrompt)
		return def
	}

	// Name Recovery: If the LLM used generic names, try to recover from analysis
	analysisFields := []string{}
	if output.Analysis != "" {
		parts := strings.Split(output.Analysis, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				analysisFields = append(analysisFields, p)
			}
		}
	}

	// Map to track which fields should be nested (to remove them from root)
	isNested := make(map[int]bool)
	fieldMap := make(map[string]FieldDescriptor)

	// First pass: build a map of all fields and identify which ones are claimed as nested
	for i := range output.Fields {
		field := &output.Fields[i]

		// Recover name if generic
		trimmedName := strings.TrimSpace(field.FieldName)
		if (trimmedName == "" || trimmedName == "fieldName" || trimmedName == "property") && i < len(analysisFields) {
			field.FieldName = analysisFields[i]
			trimmedName = field.FieldName
		}

		name := trimmedName
		if name == "" {
			continue
		}

		// If we have duplicates, we prefer the non-complex one for the map
		// so that parents can claim their children.
		if existing, ok := fieldMap[name]; !ok || existing.IsComplex {
			fieldMap[name] = *field
		}

		isComplex := field.IsComplex || sanitizeType(field.FieldType) == jsonSchema.Object || sanitizeType(field.FieldType) == jsonSchema.Array

		// If the LLM provided nestedAnalysis but empty nestedFields, we'll use it as a hint
		if isComplex && len(field.NestedFields) == 0 && field.NestedAnalysis != "" {
			parts := strings.Split(field.NestedAnalysis, ",")
			for _, p := range parts {
				pName := strings.TrimSpace(p)
				if pName == "" {
					continue
				}
				// Mark any OTHER field with this name as nested
				for j, otherField := range output.Fields {
					if i != j && strings.TrimSpace(otherField.FieldName) == pName {
						isNested[j] = true
					}
				}
			}
		}

		// Also track fields that ARE already nested in the LLM output
		for _, nf := range field.NestedFields {
			nfName := strings.TrimSpace(nf.FieldName)
			for j, otherField := range output.Fields {
				if strings.TrimSpace(otherField.FieldName) == nfName {
					isNested[j] = true
				}
			}
		}
	}

	// Second pass: build the properties, skipping those that are claimed as nested
	for i, field := range output.Fields {
		fieldName := strings.TrimSpace(field.FieldName)
		if fieldName == "" || isNested[i] {
			continue
		}

		fieldDef := convertFieldDescriptorToDefinition(field)

		isComplex := field.IsComplex || sanitizeType(field.FieldType) == jsonSchema.Object || sanitizeType(field.FieldType) == jsonSchema.Array

		// Heuristic: if this is a complex field with no properties but we have a hint from nestedAnalysis
		if isComplex && (fieldDef.Properties == nil || len(fieldDef.Properties) == 0) && field.NestedAnalysis != "" {
			parts := strings.Split(field.NestedAnalysis, ",")
			fieldDef.Properties = make(map[string]jsonSchema.Definition)
			for _, p := range parts {
				pName := strings.TrimSpace(p)
				if childField, ok := fieldMap[pName]; ok {
					// Don't add yourself as a child if you are complex
					if strings.TrimSpace(childField.FieldName) == fieldName && (childField.IsComplex || sanitizeType(childField.FieldType) == jsonSchema.Object) {
						continue
					}
					fieldDef.Properties[pName] = convertFieldDescriptorToDefinition(childField)
				}
			}
		}

		def.Properties[fieldName] = fieldDef
	}

	return def
}

// convertFieldDescriptorToDefinition converts a FieldDescriptor to a Definition
func convertFieldDescriptorToDefinition(field FieldDescriptor) jsonSchema.Definition {
	fieldType := sanitizeType(field.FieldType)

	def := jsonSchema.Definition{
		Type:        fieldType,
		Instruction: strings.TrimSpace(field.DataInstruction),
	}

	// Handle nested fields for complex types
	if len(field.NestedFields) > 0 {
		if fieldType == jsonSchema.Object {
			def.Properties = make(map[string]jsonSchema.Definition)
			for _, nested := range field.NestedFields {
				nestedName := strings.TrimSpace(nested.FieldName)
				if nestedName == "" {
					continue
				}
				nestedDef := convertFieldDescriptorToDefinition(nested)
				def.Properties[nestedName] = nestedDef
			}
		} else if fieldType == jsonSchema.Array {
			// Auto-Wrap Logic:
			// If the LLM lists properties directly under an Array, we assume it means "Array of Objects"
			// and these are the properties of the item.
			// Exception: If there is exactly 1 nested field AND it is explicitly an Object, we use it as the item definition directly.

			if len(field.NestedFields) == 1 && sanitizeType(field.NestedFields[0].FieldType) == jsonSchema.Object {
				// The LLM explicitly defined the item structure as an object
				itemDef := convertFieldDescriptorToDefinition(field.NestedFields[0])
				def.Items = &itemDef
			} else {
				// Implicit Object: Wrap the nested fields in a new Object definition
				itemDef := jsonSchema.Definition{
					Type:        jsonSchema.Object,
					Instruction: "Generated item structure",
					Properties:  make(map[string]jsonSchema.Definition),
				}
				for _, nested := range field.NestedFields {
					nestedName := strings.TrimSpace(nested.FieldName)
					if nestedName == "" {
						continue
					}
					itemDef.Properties[nestedName] = convertFieldDescriptorToDefinition(nested)
				}
				def.Items = &itemDef
			}
		}
	}

	return def
}

func sanitizeType(t string) jsonSchema.DataType {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "string", "number", "integer", "boolean", "array", "object":
		return jsonSchema.DataType(t)
	default:
		// Fallback: try to guess from content or default to string
		if strings.Contains(t, "{") {
			return jsonSchema.Object
		}
		return jsonSchema.String
	}
}

// parseStructuralAnalysis parses the LLM's structural analysis string into a Definition
// Format: "title(string), modules(array: title, lessons(array: title, content)), instructor(object: name, bio)"
func parseStructuralAnalysis(analysis string, rootInstruction string) map[string]jsonSchema.Definition {
	properties := make(map[string]jsonSchema.Definition)

	// Split by top-level commas (not inside parentheses)
	fields := splitTopLevel(analysis, ',')

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		name, def := parseFieldSpec(field, rootInstruction)
		if name != "" {
			properties[name] = def
		}
	}

	return properties
}

// splitTopLevel splits a string by a delimiter, but only at the top level (not inside parentheses)
func splitTopLevel(s string, delim rune) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, r := range s {
		switch r {
		case '(':
			depth++
			current.WriteRune(r)
		case ')':
			depth--
			current.WriteRune(r)
		case delim:
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parseFieldSpec parses a single field specification like "modules(array: title, lessons(array: title, content))"
// context is the original user prompt to make instructions more specific
func parseFieldSpec(spec string, context string) (string, jsonSchema.Definition) {
	spec = strings.TrimSpace(spec)

	// Helper to create contextual instruction
	makeInstruction := func(action, fieldName string) string {
		if context != "" {
			return fmt.Sprintf("%s %s for %s", action, fieldName, context)
		}
		return fmt.Sprintf("%s %s", action, fieldName)
	}

	// Check if there's a type specification in parentheses
	parenIdx := strings.Index(spec, "(")
	if parenIdx == -1 {
		// Simple field with no type - assume string
		return spec, jsonSchema.Definition{
			Type:        jsonSchema.String,
			Instruction: makeInstruction("Generate", spec),
		}
	}

	name := strings.TrimSpace(spec[:parenIdx])
	typeSpec := spec[parenIdx+1:]
	// Remove trailing )
	if strings.HasSuffix(typeSpec, ")") {
		typeSpec = typeSpec[:len(typeSpec)-1]
	}

	// Check if it's array or object with nested fields
	colonIdx := strings.Index(typeSpec, ":")
	if colonIdx != -1 {
		baseType := strings.TrimSpace(typeSpec[:colonIdx])
		nestedSpec := strings.TrimSpace(typeSpec[colonIdx+1:])

		if strings.ToLower(baseType) == "array" {
			// Parse nested fields for array items
			itemDef := jsonSchema.Definition{
				Type:        jsonSchema.Object,
				Instruction: makeInstruction("Generate a", name+" entry"),
				Properties:  make(map[string]jsonSchema.Definition),
			}

			nestedFields := splitTopLevel(nestedSpec, ',')
			for _, nf := range nestedFields {
				nf = strings.TrimSpace(nf)
				if nf == "" {
					continue
				}
				nfName, nfDef := parseFieldSpec(nf, context)
				if nfName != "" {
					itemDef.Properties[nfName] = nfDef
				}
			}

			return name, jsonSchema.Definition{
				Type:        jsonSchema.Array,
				Instruction: makeInstruction("Generate a comprehensive list of", name),
				Items:       &itemDef,
			}
		} else if strings.ToLower(baseType) == "object" {
			// Parse nested fields for object
			objDef := jsonSchema.Definition{
				Type:        jsonSchema.Object,
				Instruction: makeInstruction("Generate", name+" details"),
				Properties:  make(map[string]jsonSchema.Definition),
			}

			nestedFields := splitTopLevel(nestedSpec, ',')
			for _, nf := range nestedFields {
				nf = strings.TrimSpace(nf)
				if nf == "" {
					continue
				}
				nfName, nfDef := parseFieldSpec(nf, context)
				if nfName != "" {
					objDef.Properties[nfName] = nfDef
				}
			}

			return name, objDef
		}
	}

	// Simple type specification like "title(string)"
	fieldType := sanitizeType(typeSpec)
	return name, jsonSchema.Definition{
		Type:        fieldType,
		Instruction: makeInstruction("Generate", name),
	}
}