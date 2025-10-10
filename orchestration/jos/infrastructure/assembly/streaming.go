package assembly

import "firechimp/orchestration/jos/domain"

// StreamingAssembler streams individual field completions
type StreamingAssembler struct{}

func NewStreamingAssembler() *StreamingAssembler {
	return &StreamingAssembler{}
}

func (a *StreamingAssembler) Assemble(results []*domain.TaskResult) (*domain.GenerationResult, error) {
	// Fallback to default assembler
	defaultAssembler := NewDefaultAssembler()
	return defaultAssembler.Assemble(results)
}

func (a *StreamingAssembler) AssembleStreaming(results <-chan *domain.TaskResult) (<-chan *domain.StreamChunk, error) {
	out := make(chan *domain.StreamChunk, 100)

	go func() {
		defer close(out)

		for result := range results {
			if result.IsSuccess() {
				chunk := domain.NewStreamChunk(result.Key(), result.Value())
				chunk.Path = result.Path()
				out <- chunk
			}
		}

		// Send final marker
		finalChunk := domain.NewStreamChunk("", nil)
		finalChunk.MarkFinal()
		out <- finalChunk
	}()

	return out, nil
}
