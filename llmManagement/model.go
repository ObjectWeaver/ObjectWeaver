package llmManagement

import "github.com/henrylamb/object-generation-golang/jsonSchema"

type Inputs struct {
	Def          *jsonSchema.Definition
	Prompt       string
	SystemPrompt string
	OutStream    chan interface{}
}
