package modelConverter

import "github.com/henrylamb/object-generation-golang/jsonSchema"

// providerModelConverter is a concrete implementation of the ModelConverter interface.
// It maps local ModelType enums to their official string representations for various LLM providers.
type OpenAiModelConverter struct{}

// NewModelConverter creates and returns a new instance of the providerModelConverter.
func NewOpenAiModelConverter() ModelConverter {
	return &OpenAiModelConverter{}
}

// Convert implements the logic to translate a ModelType into the correct API string.
func (c *OpenAiModelConverter) Convert(model jsonSchema.ModelType) string {
	return "gpt-4-turbo"
}
