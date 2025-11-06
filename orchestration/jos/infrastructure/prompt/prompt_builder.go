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
package prompt

import (
	"fmt"
	"objectweaver/orchestration/extractor"
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// DefaultPromptBuilder builds prompts for field generation
type DefaultPromptBuilder struct {
	extractor extractor.Extractor
}

func NewDefaultPromptBuilder() *DefaultPromptBuilder {
	return &DefaultPromptBuilder{
		extractor: extractor.NewDefaultExtractor(),
	}
}

func (b *DefaultPromptBuilder) Build(task *domain.FieldTask, context *domain.PromptContext) (string, error) {
	def := task.Definition()

	// Check for override prompt
	if def.OverridePrompt != nil {
		return *def.OverridePrompt, nil
	}

	// Get base prompt from context (the user's overarching prompt)
	basePrompt := context.FirstPrompt()
	// Note: We don't fall back to def.Instruction here because that's the field-specific
	// instruction, not the overarching context. The field instruction is used separately below.

	// Build contextual information
	currentGen := ""
	if context.CurrentGen != "" {
		currentGen = fmt.Sprintf("Context:\n%s\n\n", context.CurrentGen)
	}

	// Check for narrow focus
	if def.NarrowFocus != nil {
		return b.buildNarrowFocusPrompt(def, context, basePrompt)
	}

	// Standard prompt template
	return fmt.Sprintf(`
Task:
Please return information just about the "%s" using the below instructions and context:

Instructions:

Overarching instruction: 
%s

Direct Instruction for the %s:
%s

---------
Context:
%s

%s
`,
		task.Key(),
		basePrompt,
		task.Key(),
		def.Instruction,
		currentGen,
		context.CurrentGen,
	), nil
}

func (b *DefaultPromptBuilder) BuildWithHistory(
	task *domain.FieldTask,
	context *domain.PromptContext,
	history *domain.GenerationHistory,
) (string, error) {
	// For now, delegate to standard build
	// In the future, this would incorporate generation history
	return b.Build(task, context)
}

func (b *DefaultPromptBuilder) buildNarrowFocusPrompt(
	def *jsonSchema.Definition,
	context *domain.PromptContext,
	basePrompt string,
) (string, error) {
	originalPrompt := ""
	if def.NarrowFocus.KeepOriginal {
		originalPrompt = fmt.Sprintf("Additional Contextual Information:\n%s\n\n", context.FirstPrompt())
	}

	info := context.CurrentGen
	if info == "" {
		info = basePrompt
	}

	return fmt.Sprintf(`
Instruction: 
%s 

-----
Context:
%s

%s
`,
		def.NarrowFocus.Prompt,
		info,
		originalPrompt,
	), nil
}
