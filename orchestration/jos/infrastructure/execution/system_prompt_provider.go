package execution

import (
	"objectweaver/jsonSchema"
)

// SystemPromptProvider provides type-specific system prompts
type SystemPromptProvider interface {
	GetSystemPrompt(dataType jsonSchema.DataType) *string
}

// DefaultSystemPromptProvider provides default system prompts for primitive types
type DefaultSystemPromptProvider struct{}

func NewDefaultSystemPromptProvider() *DefaultSystemPromptProvider {
	return &DefaultSystemPromptProvider{}
}

func (p *DefaultSystemPromptProvider) GetSystemPrompt(dataType jsonSchema.DataType) *string {
	switch dataType {
	case jsonSchema.Number, jsonSchema.Integer:
		prompt := "The value being generated for this is a number. The value returned must be a number. No other values can be returned."
		return &prompt
	case jsonSchema.Boolean:
		prompt := "The value being generated for this is a boolean. The value returned must be either true or false. No other values can be returned."
		return &prompt
	case jsonSchema.String:
		prompt := "The value being generated for this is a string. Return only the string value without any additional formatting or quotes."
		return &prompt
	case jsonSchema.Byte:
		prompt := "The value being generated for this is a byte value. Return only the byte value."
		return &prompt
	default:
		prompt := "The value being generated for this is of type " + string(dataType) + ". Return only the value without any additional formatting."
		return &prompt
		//return nil
	}
}

// NoSystemPromptProvider provides no system prompts (returns nil for all types)
type NoSystemPromptProvider struct{}

func NewNoSystemPromptProvider() *NoSystemPromptProvider {
	return &NoSystemPromptProvider{}
}

func (p *NoSystemPromptProvider) GetSystemPrompt(dataType jsonSchema.DataType) *string {
	return nil
}

// CustomSystemPromptProvider allows custom system prompts to be set per type
type CustomSystemPromptProvider struct {
	prompts map[jsonSchema.DataType]string
}

func NewCustomSystemPromptProvider() *CustomSystemPromptProvider {
	return &CustomSystemPromptProvider{
		prompts: make(map[jsonSchema.DataType]string),
	}
}

func (p *CustomSystemPromptProvider) SetPrompt(dataType jsonSchema.DataType, prompt string) {
	p.prompts[dataType] = prompt
}

func (p *CustomSystemPromptProvider) GetSystemPrompt(dataType jsonSchema.DataType) *string {
	if prompt, exists := p.prompts[dataType]; exists {
		return &prompt
	}
	return nil
}
