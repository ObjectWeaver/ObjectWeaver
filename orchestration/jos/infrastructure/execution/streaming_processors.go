package execution

import (
	"fmt"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
	"sync"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

// StreamingPrimitiveProcessor handles primitives with token-level streaming
type StreamingPrimitiveProcessor struct {
	llmProvider   domain.TokenStreamingProvider
	promptBuilder domain.PromptBuilder
	granularity   domain.StreamGranularity
}

func NewStreamingPrimitiveProcessor(
	provider domain.TokenStreamingProvider,
	builder domain.PromptBuilder,
	granularity domain.StreamGranularity,
) *StreamingPrimitiveProcessor {
	return &StreamingPrimitiveProcessor{
		llmProvider:   provider,
		promptBuilder: builder,
		granularity:   granularity,
	}
}

func (p *StreamingPrimitiveProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.String ||
		schemaType == jsonSchema.Number ||
		schemaType == jsonSchema.Boolean
}

func (p *StreamingPrimitiveProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// For non-streaming mode, collect all tokens
	tokenStream, err := p.ProcessStreaming(task, context)
	if err != nil {
		return nil, err
	}

	accumulated := ""
	for token := range tokenStream {
		if token.Complete {
			accumulated = token.Partial
		}
	}

	metadata := domain.NewResultMetadata()
	result := domain.NewTaskResult(task.ID(), task.Key(), accumulated, metadata)
	return result.WithPath(task.Path()), nil
}

func (p *StreamingPrimitiveProcessor) ProcessStreaming(task *domain.FieldTask, context *domain.ExecutionContext) (<-chan *domain.TokenStreamChunk, error) {
	out := make(chan *domain.TokenStreamChunk, 100)

	// Use worker pool if available, otherwise spawn goroutine directly
	if context.WorkerPool() != nil {
		context.WorkerPool().Submit(func() {
			defer close(out)

			// Build prompt
			prompt, err := p.promptBuilder.Build(task, context.PromptContext())
			if err != nil {
				return
			}

			// Get token stream from LLM
			config := context.GenerationConfig()
			tokenStream, err := p.llmProvider.GenerateTokenStream(prompt, config)
			if err != nil {
				return
			}

			accumulated := ""

			for token := range tokenStream {
				accumulated += token.Token

				// Emit based on granularity
				if p.shouldEmit(token) {
					chunk := domain.NewTokenStreamChunk(task.Key(), token.Token)
					chunk.Partial = accumulated
					chunk.Complete = token.IsFinal
					chunk.Path = task.Path()

					out <- chunk
				}
			}

			// Emit final
			final := domain.NewTokenStreamChunk(task.Key(), "")
			final.Partial = accumulated
			final.MarkComplete()
			final.Path = task.Path()
			out <- final
		})
	} else {
		go func() {
			defer close(out)

			// Build prompt
			prompt, err := p.promptBuilder.Build(task, context.PromptContext())
			if err != nil {
				return
			}

			// Get token stream from LLM
			config := context.GenerationConfig()
			tokenStream, err := p.llmProvider.GenerateTokenStream(prompt, config)
			if err != nil {
				return
			}

			accumulated := ""

			for token := range tokenStream {
				accumulated += token.Token

				// Emit based on granularity
				if p.shouldEmit(token) {
					chunk := domain.NewTokenStreamChunk(task.Key(), token.Token)
					chunk.Partial = accumulated
					chunk.Complete = token.IsFinal
					chunk.Path = task.Path()

					out <- chunk
				}
			}

			// Emit final
			final := domain.NewTokenStreamChunk(task.Key(), "")
			final.Partial = accumulated
			final.MarkComplete()
			final.Path = task.Path()
			out <- final
		}()
	}

	return out, nil
}

func (p *StreamingPrimitiveProcessor) shouldEmit(token *domain.TokenChunk) bool {
	switch p.granularity {
	case domain.GranularityToken:
		return true // Emit every token
	case domain.GranularityChunk:
		// Emit on sentence boundaries
		return token.Token == "." || token.Token == "!" || token.Token == "?"
	default:
		return token.IsFinal // Emit only when complete
	}
}

// StreamingObjectProcessor handles objects with progressive field streaming
type StreamingObjectProcessor struct {
	llmProvider   domain.TokenStreamingProvider
	promptBuilder domain.PromptBuilder
	granularity   domain.StreamGranularity
}

func NewStreamingObjectProcessor(
	provider domain.TokenStreamingProvider,
	builder domain.PromptBuilder,
	granularity domain.StreamGranularity,
) *StreamingObjectProcessor {
	return &StreamingObjectProcessor{
		llmProvider:   provider,
		promptBuilder: builder,
		granularity:   granularity,
	}
}

func (p *StreamingObjectProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Object
}

func (p *StreamingObjectProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Extract nested fields
	nestedFields, err := p.extractFields(task.Definition())
	if err != nil {
		return nil, err
	}

	// Process fields and merge streams
	tokenStream := p.mergeFieldStreams(nestedFields, task, context)

	// Collect results
	nestedResults := make(map[string]interface{})
	for token := range tokenStream {
		if token.Complete {
			nestedResults[token.Key] = token.Partial
		}
	}

	metadata := domain.NewResultMetadata()
	result := domain.NewTaskResult(task.ID(), task.Key(), nestedResults, metadata)
	return result.WithPath(task.Path()), nil
}

func (p *StreamingObjectProcessor) ProcessStreaming(task *domain.FieldTask, context *domain.ExecutionContext) (<-chan *domain.TokenStreamChunk, error) {
	nestedFields, err := p.extractFields(task.Definition())
	if err != nil {
		return nil, err
	}

	return p.mergeFieldStreams(nestedFields, task, context), nil
}

func (p *StreamingObjectProcessor) mergeFieldStreams(
	fields map[string]*jsonSchema.Definition,
	parentTask *domain.FieldTask,
	context *domain.ExecutionContext,
) <-chan *domain.TokenStreamChunk {
	out := make(chan *domain.TokenStreamChunk, 100)

	// Use worker pool if available, otherwise spawn goroutine directly
	if context.WorkerPool() != nil {
		context.WorkerPool().Submit(func() {
			defer close(out)

			var wg sync.WaitGroup

			for key, def := range fields {
				wg.Add(1)

				// Use worker pool for nested goroutines too
				context.WorkerPool().Submit(func(k string, d *jsonSchema.Definition) func() {
					return func() {
						defer wg.Done()

						task := domain.NewFieldTask(k, d, parentTask)

						processor := NewStreamingPrimitiveProcessor(p.llmProvider, p.promptBuilder, p.granularity)
						tokenStream, err := processor.ProcessStreaming(task, context)
						if err != nil {
							return
						}

						// Forward tokens
						for token := range tokenStream {
							out <- token
						}
					}
				}(key, def))
			}

			wg.Wait()
		})
	} else {
		go func() {
			defer close(out)

			var wg sync.WaitGroup

			for key, def := range fields {
				wg.Add(1)

				go func(k string, d *jsonSchema.Definition) {
					defer wg.Done()

					task := domain.NewFieldTask(k, d, parentTask)

					processor := NewStreamingPrimitiveProcessor(p.llmProvider, p.promptBuilder, p.granularity)
					tokenStream, err := processor.ProcessStreaming(task, context)
					if err != nil {
						return
					}

					// Forward tokens
					for token := range tokenStream {
						out <- token
					}
				}(key, def)
			}

			wg.Wait()
		}()
	}

	return out
}

func (p *StreamingObjectProcessor) extractFields(def *jsonSchema.Definition) (map[string]*jsonSchema.Definition, error) {
	if def.Properties == nil {
		return make(map[string]*jsonSchema.Definition), nil
	}

	fields := make(map[string]*jsonSchema.Definition)
	for key, childDef := range def.Properties {
		childDefCopy := childDef
		fields[key] = &childDefCopy
	}

	return fields, nil
}

// StreamingArrayProcessor handles arrays with progressive streaming
type StreamingArrayProcessor struct {
	llmProvider   domain.TokenStreamingProvider
	promptBuilder domain.PromptBuilder
	granularity   domain.StreamGranularity
}

func NewStreamingArrayProcessor(
	provider domain.TokenStreamingProvider,
	builder domain.PromptBuilder,
	granularity domain.StreamGranularity,
) *StreamingArrayProcessor {
	return &StreamingArrayProcessor{
		llmProvider:   provider,
		promptBuilder: builder,
		granularity:   granularity,
	}
}

func (p *StreamingArrayProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	return schemaType == jsonSchema.Array
}

func (p *StreamingArrayProcessor) Process(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.TaskResult, error) {
	tokenStream, err := p.ProcessStreaming(task, context)
	if err != nil {
		return nil, err
	}

	items := make([]interface{}, 0)
	for token := range tokenStream {
		if token.Complete {
			items = append(items, token.Partial)
		}
	}

	metadata := domain.NewResultMetadata()
	result := domain.NewTaskResult(task.ID(), task.Key(), items, metadata)
	return result.WithPath(task.Path()), nil
}

func (p *StreamingArrayProcessor) ProcessStreaming(task *domain.FieldTask, context *domain.ExecutionContext) (<-chan *domain.TokenStreamChunk, error) {
	out := make(chan *domain.TokenStreamChunk, 100)

	// Use worker pool if available, otherwise spawn goroutine directly
	if context.WorkerPool() != nil {
		context.WorkerPool().Submit(func() {
			defer close(out)

			itemDef := task.Definition().Items
			if itemDef == nil {
				return
			}

			arraySize := 3 // Default

			for i := 0; i < arraySize; i++ {
				itemTask := domain.NewFieldTask(fmt.Sprintf("%s[%d]", task.Key(), i), itemDef, task)

				processor := NewStreamingPrimitiveProcessor(p.llmProvider, p.promptBuilder, p.granularity)
				tokenStream, err := processor.ProcessStreaming(itemTask, context)
				if err != nil {
					continue
				}

				for token := range tokenStream {
					out <- token
				}
			}
		})
	} else {
		go func() {
			defer close(out)

			itemDef := task.Definition().Items
			if itemDef == nil {
				return
			}

			arraySize := 3 // Default

			for i := 0; i < arraySize; i++ {
				itemTask := domain.NewFieldTask(fmt.Sprintf("%s[%d]", task.Key(), i), itemDef, task)

				processor := NewStreamingPrimitiveProcessor(p.llmProvider, p.promptBuilder, p.granularity)
				tokenStream, err := processor.ProcessStreaming(itemTask, context)
				if err != nil {
					continue
				}

				for token := range tokenStream {
					out <- token
				}
			}
		}()
	}

	return out, nil
}
