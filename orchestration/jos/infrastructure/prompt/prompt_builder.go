package prompt

import (
	"firechimp/orchestration/extractor"
	"firechimp/orchestration/jos/domain"
	"fmt"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
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

	// Get base prompt
	basePrompt := context.FirstPrompt()
	if basePrompt == "" {
		basePrompt = def.Instruction
	}

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
