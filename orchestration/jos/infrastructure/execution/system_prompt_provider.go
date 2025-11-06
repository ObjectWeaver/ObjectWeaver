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
// <https://objectweaver.dev/licensing/server-side-public-license>.
package execution

import (
	"github.com/objectweaver/go-sdk/jsonSchema"
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
		return nil
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
