package service

import (
	"encoding/json"
	"net/http"
	"objectweaver/logger"
	"strings"

	"github.com/objectweaver/go-sdk/client"
	"github.com/objectweaver/go-sdk/jsonSchema"
)

type TtwRequest struct {
	Prompt string
}

// TtwRawOutput is the intermediate format from the LLM before post-processing
type TtwRawOutput struct {
	Analysis              string            `json:"analysis"`
	RootType              string            `json:"rootType"`
	DefinitionInstruction string            `json:"definitionInstruction"`
	Fields                []FieldDescriptor `json:"fields"`
}

// FieldDescriptor represents a single field in the generated schema
type FieldDescriptor struct {
	FieldName       string            `json:"fieldName"`
	FieldType       string            `json:"fieldType"`
	Reasoning       string            `json:"reasoning"`
	DataInstruction string            `json:"dataInstruction"`
	IsComplex       bool              `json:"isComplex"`
	NestedAnalysis  string            `json:"nestedAnalysis,omitempty"`
	NestedCount     int               `json:"nestedCount,omitempty"`
	NestedFields    []FieldDescriptor `json:"nestedFields,omitempty"`
}

type TtwResponse struct {
	Definition jsonSchema.Definition `json:"definition"`
}

func (s *Server) TextToWeaver(w http.ResponseWriter, r *http.Request) {

	//get the data out of the request
	req := &TtwRequest{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	//create the request body
	body := client.RequestBody{
		Prompt:     req.Prompt,
		Definition: definitionForDefinition(),
	}

	response, err := processObjectGenRequest(body, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Marshal response.Data to JSON bytes
	bytes, err := json.Marshal(response.Data)
	if err != nil {
		http.Error(w, "Failed to marshal response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Unmarshal into our intermediate type
	rawOutput := &TtwRawOutput{}
	if err := json.Unmarshal(bytes, rawOutput); err != nil {
		http.Error(w, "Failed to parse LLM output: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Post-process to convert to proper Definition structure
	definition := convertRawOutputToDefinition(rawOutput)

	// Build the final response
	res := &TtwResponse{
		Definition: *definition,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		logger.Printf("Error encoding response (context error: %v): %v", r.Context().Err(), err)
	}
}

func definitionForDefinition() *jsonSchema.Definition {
	// Define what a single field looks like (this is the recursive unit)
	var fieldDescriptor jsonSchema.Definition
	fieldDescriptor = jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "Define ONE property from the decomposed list. Do NOT define the root object.",
		Properties: map[string]jsonSchema.Definition{
			"fieldName": {
				Type:        jsonSchema.String,
				Instruction: "The property name. MUST be a specific attribute (e.g. 'email', 'age'). \n\nCRITICAL: Do NOT use the name of the ROOT object (e.g. 'user', 'profile' if the request is for a user profile). Use the names identified in the structuralAnalysis. NEVER use generic names like 'fieldName' or 'property'.",
			},
			"reasoning": {
				Type:        jsonSchema.String,
				Instruction: "Analyze the field name. Is it a simple value (string/number) or a complex structure (object/array)? \nExample: 'address' is complex because it contains street and city.",
			},
			"fieldType": {
				Type:        jsonSchema.String,
				Instruction: "The JSON data type. \n\n- 'string' (for text, email, date)\n- 'number' (for float)\n- 'integer' (for int)\n- 'boolean' (true/false)\n- 'object' (for nested structures)\n- 'array' (for lists)\n\nDo NOT use 'object' for simple strings like email.",
			},
			"isComplex": {
				Type:        jsonSchema.Boolean,
				Instruction: "CRITICAL: If fieldType is 'object' or 'array', this MUST be TRUE. Set to FALSE only for string, number, integer, boolean.",
			},
			"dataInstruction": {
				Type:        jsonSchema.String,
				Instruction: "Write a specific command for the generator. \n\nCRITICAL: If this is an object, the instruction should be about the object as a whole, NOT its children. \nExample: if fieldName='address', instruction='Generate address details'. Do NOT say 'Generate street and city'.",
			},
			"nestedAnalysis": {
				Type:        jsonSchema.String,
				Instruction: "If isComplex is true, list the attributes for THIS nested object ONLY. Example: if fieldName is 'address', list 'street, city, zip'. Do NOT list root fields here. Otherwise leave empty.",
			},
			"nestedCount": {
				Type:        jsonSchema.Integer,
				Instruction: "Count the attributes in nestedAnalysis. 0 if isComplex is false.",
			},
			"nestedFields": {
				Type:        jsonSchema.Array,
				Instruction: "If isComplex is true, generate exactly [nestedCount] items. Otherwise leave empty.",
				Items:       &fieldDescriptor, // Recursive reference!
			},
		},
		ProcessingOrder: []string{"fieldName", "reasoning", "fieldType", "isComplex", "dataInstruction", "nestedAnalysis", "nestedCount", "nestedFields"},
	}

	// The main structure generates an array of fields with recursive capability
	return &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Instruction: "You are a Schema Flattener. The user wants an object. You must list its INTERNAL properties.\n\nInput: 'Create a User'\n\nWRONG Output:\n- Field: 'user' (Type: object)\n\nRIGHT Output:\n- Field: 'username' (Type: string)\n- Field: 'email' (Type: string)",
		Properties: map[string]jsonSchema.Definition{
			// Step 0: Structural Analysis
			"structuralAnalysis": {
				Type:        jsonSchema.String,
				Instruction: "Analyze the request. Identify which fields are simple (string/number) and which are complex (objects/arrays). \nExample: 'User with address' -> 'name (simple), email (simple), address (complex: street, city)'.",
			},
			// Step 1: Analysis phase
			"analysis": {
				Type:        jsonSchema.String,
				Instruction: "List ONLY the ROOT level fields requested. \n\nCRITICAL: If a field is nested (e.g. 'street' inside 'address'), ONLY list the parent ('address'). Do NOT list 'street' at this level. \nExample: 'User with name and address (street, city)' -> 'name, address'.",
			},
			// Step 1.5: Count
			"fieldCount": {
				Type:        jsonSchema.Integer,
				Instruction: "Count the number of items in your ROOT level analysis list.",
			},
			// Step 2: Determine root type
			"rootType": {
				Type:        jsonSchema.String,
				Instruction: "The root type of the schema. MUST be either 'object' or 'array'.",
			},
			// Step 3: Main instruction for the generated definition
			"definitionInstruction": {
				Type:        jsonSchema.String,
				Instruction: "A command to generate the whole object. Example: 'Generate a realistic user profile'.",
			},
			// Step 4: Generate the list of fields (this is where recursion happens)
			"fields": {
				Type:        jsonSchema.Array,
				Instruction: "Generate exactly [fieldCount] fields as identified in the analysis. \n\nCRITICAL: If a field is complex (object/array), define it ONCE here. Its children will be defined inside its 'nestedFields' property.",
				Items:       &fieldDescriptor,
			},
		},
		ProcessingOrder: []string{"structuralAnalysis", "analysis", "fieldCount", "rootType", "definitionInstruction", "fields"},
	}
}

// convertRawOutputToDefinition transforms the LLM output into a proper jsonSchema.Definition
func convertRawOutputToDefinition(output *TtwRawOutput) *jsonSchema.Definition {
	rootType := jsonSchema.Object
	if strings.TrimSpace(output.RootType) == "array" {
		rootType = jsonSchema.Array
	}

	def := &jsonSchema.Definition{
		Type:        rootType,
		Instruction: strings.TrimSpace(output.DefinitionInstruction),
		Properties:  make(map[string]jsonSchema.Definition),
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
		} else if fieldType == jsonSchema.Array && len(field.NestedFields) > 0 {
			// For arrays, use the first nested field descriptor as the Items definition
			itemDef := convertFieldDescriptorToDefinition(field.NestedFields[0])
			def.Items = &itemDef
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
