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

	// Get parent instruction if parent exists
	parentInstruction := ""
	if task.Parent() != nil && task.Parent().Definition() != nil {
		parentInstruction = task.Parent().Definition().Instruction
		parentInstruction = fmt.Sprintf("Overarching instruction: \n%s\n\n", parentInstruction)
	}

	// Standard prompt template
	return fmt.Sprintf(`
Task:
Please return information just about the "%s" using the below instructions and context:

%s

Direct Instruction for the %s:
%s

User information:
%s

%s
`,
		task.Key(),
		parentInstruction,
		task.Key(),
		def.Instruction,
		basePrompt,
		currentGen,
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
