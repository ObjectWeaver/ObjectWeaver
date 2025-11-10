package domain

// PromptBuilder - Builds prompts for field generation with context awareness
//
// Implementation: infrastructure/prompt/prompt_builder.go (DefaultPromptBuilder)
// Created by: factory/generator_factory.go:createPromptBuilder()
// Used by: TypeProcessor implementations
//
// Responsibilities:
//   - Construct contextual prompts for LLM
//   - Include parent context and previously generated values
//   - Handle retry scenarios with generation history
//   - Format prompts according to field type
type PromptBuilder interface {
	Build(task *FieldTask, context *PromptContext) (string, error)
	BuildWithHistory(task *FieldTask, context *PromptContext, history *GenerationHistory) (string, error)
}

// PromptContext contains context for prompt building
type PromptContext struct {
	Prompts          []string
	CurrentGen       string
	ParentGen        string
	ExistingSubLists []string
}

func NewPromptContext() *PromptContext {
	return &PromptContext{
		Prompts:          make([]string, 0),
		ExistingSubLists: make([]string, 0),
	}
}

func (p *PromptContext) AddPrompt(prompt string) {
	p.Prompts = append(p.Prompts, prompt)
}

func (p *PromptContext) FirstPrompt() string {
	if len(p.Prompts) > 0 {
		return p.Prompts[0]
	}
	return ""
}

// GenerationHistory tracks generation history
type GenerationHistory struct {
	attempts   int
	lastPrompt string
	lastResult string
}
