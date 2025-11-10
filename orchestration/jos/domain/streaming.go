package domain

import (
	"time"
)

// StreamGranularity defines how fine-grained the streaming is
type StreamGranularity int

const (
	GranularityField StreamGranularity = iota // Complete field values only
	GranularityToken                          // Token-by-token streaming
	GranularityChunk                          // Chunk-by-chunk (sentences, etc)
)

// StreamChunk represents a chunk of streaming data
type StreamChunk struct {
	Key             string
	Value           interface{}
	Path            []string
	NewKey          string
	NewValue        interface{}
	AccumulatedData map[string]interface{}
	Progress        float64
	IsFinal         bool
	Timestamp       time.Time
}

func NewStreamChunk(key string, value interface{}) *StreamChunk {
	return &StreamChunk{
		Key:       key,
		Value:     value,
		IsFinal:   false,
		Timestamp: time.Now(),
	}
}

func (s *StreamChunk) WithAccumulatedData(data map[string]interface{}) *StreamChunk {
	s.AccumulatedData = data
	return s
}

func (s *StreamChunk) WithProgress(progress float64) *StreamChunk {
	s.Progress = progress
	return s
}

func (s *StreamChunk) MarkFinal() *StreamChunk {
	s.IsFinal = true
	return s
}

// TokenStreamChunk - Individual token from LLM
type TokenStreamChunk struct {
	Key       string
	Path      []string
	Token     string
	Partial   string // Accumulated value so far
	Complete  bool
	FieldType string
}

func NewTokenStreamChunk(key, token string) *TokenStreamChunk {
	return &TokenStreamChunk{
		Key:      key,
		Token:    token,
		Partial:  token,
		Complete: false,
	}
}

func (t *TokenStreamChunk) AppendToken(token string) {
	t.Partial += token
}

func (t *TokenStreamChunk) MarkComplete() {
	t.Complete = true
}

// AccumulatedStreamChunk - Current state of entire object with progressive values
type AccumulatedStreamChunk struct {
	CurrentMap        map[string]interface{}
	ProgressiveFields map[string]*ProgressiveValue
	NewToken          *TokenStreamChunk
	Progress          float64
	IsFinal           bool
}

// ProgressiveValue represents a value being built incrementally
type ProgressiveValue struct {
	key          string
	path         []string
	currentValue string
	isComplete   bool
	tokens       []string
	confidence   float64
}

func NewProgressiveValue(key string, path []string) *ProgressiveValue {
	return &ProgressiveValue{
		key:        key,
		path:       path,
		tokens:     make([]string, 0),
		isComplete: false,
	}
}

func (pv *ProgressiveValue) Append(token string) {
	pv.tokens = append(pv.tokens, token)
	pv.currentValue += token
}

func (pv *ProgressiveValue) CurrentValue() string {
	return pv.currentValue
}

func (pv *ProgressiveValue) IsComplete() bool {
	return pv.isComplete
}

func (pv *ProgressiveValue) MarkComplete() {
	pv.isComplete = true
}

func (pv *ProgressiveValue) Key() string {
	return pv.key
}

func (pv *ProgressiveValue) Path() []string {
	return pv.path
}

func (pv *ProgressiveValue) Tokens() []string {
	return pv.tokens
}

// TokenChunk from LLM provider
type TokenChunk struct {
	Token        string
	Delta        string // For chat models that send deltas
	IsFinal      bool
	FinishReason string
}

func NewTokenChunk(token string) *TokenChunk {
	return &TokenChunk{
		Token:   token,
		IsFinal: false,
	}
}
