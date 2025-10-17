package llmManagement

import "github.com/objectweaver/go-sdk/jsonSchema"

type Inputs struct {
	Def          *jsonSchema.Definition
	Prompt       string
	SystemPrompt string
	OutStream    chan interface{}
	Index		int // The index of the item in the heap
	Priority   int32 // Higher value means higher priority // values under 0 or lower will be considered eventually ie will be processed in a batching system
}
