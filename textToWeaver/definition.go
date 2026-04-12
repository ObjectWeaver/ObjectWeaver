package texttoweaver

import "objectweaver/jsonSchema"

func DefinitionForDefinition() *jsonSchema.Definition {
	fieldDescriptor := &jsonSchema.Definition{
		Type:        "object",
		Instruction: "Describe one schema field with its name, type, generation instruction, and optional nested structure.",
		ProcessingOrder: []string{
			"fieldName",
			"fieldType",
			"reasoning",
			"dataInstruction",
			"isComplex",
			"nestedAnalysis",
			"nestedCount",
			"nestedFields",
		},
		SelectFields: []string{
			"fieldName",
			"fieldType",
			"isComplex",
			"nestedAnalysis",
			"nestedCount",
			"dataInstruction",
		},
		NarrowFocus: &jsonSchema.Focus{
			Prompt: "Only define the current field descriptor. Do not invent sibling or root-level fields.",
			Fields: []string{
				"fieldName",
				"fieldType",
				"isComplex",
				"nestedAnalysis",
				"nestedCount",
				"dataInstruction",
			},
			KeepOriginal: true,
		},
		Properties: map[string]jsonSchema.Definition{
			"fieldName": {
				Type:        "string",
				Instruction: "Use a semantic field name in snake_case.",
			},
			"fieldType": {
				Type:        "string",
				Instruction: "Choose one valid JSON/ObjectWeaver type such as string, number, integer, boolean, object, or array.",
			},
			"reasoning": {
				Type:        "string",
				Instruction: "Briefly explain why this field is needed based on the user prompt.",
			},
			"dataInstruction": {
				Type:        "string",
				Instruction: "Provide a precise generation instruction for this field.",
			},
			"isComplex": {
				Type:        "boolean",
				Instruction: "Set true when this field is object/array with nested structure, otherwise false.",
			},
			"nestedAnalysis": {
				Type:        "string",
				Instruction: "Optional comma-separated hint of nested field names when complex structure is required.",
			},
			"nestedCount": {
				Type:        "integer",
				Instruction: "Optional count of expected nested fields.",
			},
		},
	}

	// Recursive descriptor for nested objects/arrays.
	fieldDescriptor.Properties["nestedFields"] = jsonSchema.Definition{
		Type:        "array",
		Instruction: "Optional nested field descriptors for object/array field types.",
		Items:       fieldDescriptor,
	}

	fieldDescriptor.DecisionPoint = &jsonSchema.DecisionPoint{
		Name:     "expand_nested_fields_when_complex",
		Strategy: jsonSchema.RouteByField,
		Branches: []jsonSchema.ConditionalBranch{
			{
				Name: "expand_array_items",
				Conditions: []jsonSchema.Condition{
					{
						Field:    "isComplex",
						Operator: jsonSchema.OpEqual,
						Value:    true,
					},
					{
						Field:    "fieldType",
						Operator: jsonSchema.OpEqual,
						Value:    "array",
					},
				},
				Then: jsonSchema.Definition{
					Type:        "object",
					Instruction: "Generate nestedFields only for the array item structure. Do not add root-level or sibling fields.",
					SelectFields: []string{
						"fieldName",
						"fieldType",
						"nestedAnalysis",
						"nestedCount",
						"dataInstruction",
					},
					NarrowFocus: &jsonSchema.Focus{
						Prompt: "You are defining only array item fields for this single array property. Keep scope tight to nested item fields.",
						Fields: []string{
							"fieldName",
							"fieldType",
							"nestedAnalysis",
							"nestedCount",
							"dataInstruction",
						},
						KeepOriginal: true,
					},
					Properties: map[string]jsonSchema.Definition{
						"nestedFields": {
							Type:        "array",
							Instruction: "Generate nested item field descriptors only.",
							Items:       fieldDescriptor,
						},
					},
				},
			},
			{
				Name: "expand_object_fields",
				Conditions: []jsonSchema.Condition{
					{
						Field:    "isComplex",
						Operator: jsonSchema.OpEqual,
						Value:    true,
					},
					{
						Field:    "fieldType",
						Operator: jsonSchema.OpEqual,
						Value:    "object",
					},
				},
				Then: jsonSchema.Definition{
					Type:        "object",
					Instruction: "Generate nestedFields only for this object field. Do not add siblings or root fields.",
					SelectFields: []string{
						"fieldName",
						"fieldType",
						"nestedAnalysis",
						"nestedCount",
						"dataInstruction",
					},
					NarrowFocus: &jsonSchema.Focus{
						Prompt: "You are defining nested properties for this one object field only.",
						Fields: []string{
							"fieldName",
							"fieldType",
							"nestedAnalysis",
							"nestedCount",
							"dataInstruction",
						},
						KeepOriginal: true,
					},
					Properties: map[string]jsonSchema.Definition{
						"nestedFields": {
							Type:        "array",
							Instruction: "Generate nested object field descriptors only.",
							Items:       fieldDescriptor,
						},
					},
				},
			},
			{
				Name: "skip_nested_expansion",
				Conditions: []jsonSchema.Condition{
					{
						Field:    "isComplex",
						Operator: jsonSchema.OpEqual,
						Value:    false,
					},
				},
				Then: jsonSchema.Definition{
					Type:        "object",
					Instruction: "Keep nestedFields empty for primitive fields.",
				},
			},
		},
	}

	return &jsonSchema.Definition{
		Type:        "object",
		Instruction: "Convert the user prompt into a valid ObjectWeaver schema blueprint. Output only structured fields matching this schema.",
		Properties: map[string]jsonSchema.Definition{
			"structuralAnalysis": {
				Type:        "string",
				Instruction: "Summarize the proposed schema structure as readable field planning notes.",
			},
			"analysis": {
				Type:        "string",
				Instruction: "List core fields inferred from the prompt as a comma-separated summary.",
			},
			"rootType": {
				Type:        "string",
				Instruction: "Set to object for most prompts, or array when the root result must be a list.",
			},
			"definitionInstruction": {
				Type:        "string",
				Instruction: "Write a concise top-level instruction describing what the final schema should generate.",
			},
			"fields": {
				Type:        "array",
				Instruction: "Provide a correct number of field descriptors; include at least the correct amount of complex object/array fields when prompt allows/suggests.",
				Items:       fieldDescriptor,
			},
		},
	}
}
