package llmManagement

import "github.com/objectweaver/go-sdk/jsonSchema"

type Inputs struct {
	Def          *jsonSchema.Definition
	Prompt       string
	SystemPrompt string
	OutStream    chan interface{}
}
