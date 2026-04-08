package texttoweaver

// TtwRawOutput is the intermediate format from the LLM before post-processing
type TtwRawOutput struct {
	StructuralAnalysis    string            `json:"structuralAnalysis"`
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