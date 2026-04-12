package domain

import (
	"context"
	"time"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

// GenerationRequest - Immutable value object representing a generation request
type GenerationRequest struct {
	prompt      string
	schema      *jsonSchema.Definition
	context     context.Context
	metadata    map[string]interface{}
	constraints *Constraints
}

func NewGenerationRequest(prompt string, schema *jsonSchema.Definition) *GenerationRequest {
	return &GenerationRequest{
		prompt:      prompt,
		schema:      schema,
		context:     context.Background(),
		metadata:    make(map[string]interface{}),
		constraints: DefaultConstraints(),
	}
}

// Getters
func (r *GenerationRequest) Prompt() string                   { return r.prompt }
func (r *GenerationRequest) Schema() *jsonSchema.Definition   { return r.schema }
func (r *GenerationRequest) Context() context.Context         { return r.context }
func (r *GenerationRequest) Metadata() map[string]interface{} { return r.metadata }
func (r *GenerationRequest) Constraints() *Constraints        { return r.constraints }

// Fluent builder pattern
func (r *GenerationRequest) WithContext(ctx context.Context) *GenerationRequest {
	return &GenerationRequest{
		prompt:      r.prompt,
		schema:      r.schema,
		context:     ctx,
		metadata:    r.copyMetadata(),
		constraints: r.constraints,
	}
}

func (r *GenerationRequest) WithMetadata(key string, value interface{}) *GenerationRequest {
	newMetadata := r.copyMetadata()
	newMetadata[key] = value

	return &GenerationRequest{
		prompt:      r.prompt,
		schema:      r.schema,
		context:     r.context,
		metadata:    newMetadata,
		constraints: r.constraints,
	}
}

func (r *GenerationRequest) WithConstraints(constraints *Constraints) *GenerationRequest {
	return &GenerationRequest{
		prompt:      r.prompt,
		schema:      r.schema,
		context:     r.context,
		metadata:    r.copyMetadata(),
		constraints: constraints,
	}
}

func (r *GenerationRequest) copyMetadata() map[string]interface{} {
	newMetadata := make(map[string]interface{})
	for k, v := range r.metadata {
		newMetadata[k] = v
	}
	return newMetadata
}

// FieldResultWithMetadata - Contains both value and metadata for a single field
type FieldResultWithMetadata struct {
	Value    interface{}
	Metadata *ResultMetadata
}

func NewFieldResultWithMetadata(value interface{}, metadata *ResultMetadata) *FieldResultWithMetadata {
	return &FieldResultWithMetadata{
		Value:    value,
		Metadata: metadata,
	}
}

// GenerationResult - Immutable result
type GenerationResult struct {
	data                map[string]interface{}
	detailedData        map[string]*FieldResultWithMetadata
	metadata            *ResultMetadata
	errors              []error
	includeDetailedData bool
}

func NewGenerationResult(data map[string]interface{}, metadata *ResultMetadata) *GenerationResult {
	return &GenerationResult{
		data:                data,
		detailedData:        nil,
		metadata:            metadata,
		errors:              make([]error, 0),
		includeDetailedData: false,
	}
}

func NewGenerationResultWithDetailedData(data map[string]interface{}, detailedData map[string]*FieldResultWithMetadata, metadata *ResultMetadata) *GenerationResult {
	return &GenerationResult{
		data:                data,
		detailedData:        detailedData,
		metadata:            metadata,
		errors:              make([]error, 0),
		includeDetailedData: true,
	}
}

func NewGenerationResultWithError(err error) *GenerationResult {
	return &GenerationResult{
		data:                nil,
		detailedData:        nil,
		metadata:            nil,
		errors:              []error{err},
		includeDetailedData: false,
	}
}

func (r *GenerationResult) IsSuccess() bool {
	return len(r.errors) == 0
}

func (r *GenerationResult) Data() map[string]interface{} {
	return r.data
}

func (r *GenerationResult) DetailedData() map[string]*FieldResultWithMetadata {
	return r.detailedData
}

func (r *GenerationResult) Metadata() *ResultMetadata {
	return r.metadata
}

func (r *GenerationResult) Errors() []error {
	return r.errors
}

func (r *GenerationResult) HasDetailedData() bool {
	return r.includeDetailedData && r.detailedData != nil
}

// ResultMetadata contains metrics about the generation
type ResultMetadata struct {
	TokensUsed  int            `json:"tokensUsed"`
	Cost        float64        `json:"cost"`
	Duration    time.Duration  `json:"duration,omitempty"`
	ModelUsed   string         `json:"modelUsed,omitempty"`
	FieldCount  int            `json:"fieldCount,omitempty"`
	Choices     []Choice       `json:"choices,omitempty"`
	VerboseData map[string]any `json:"verboseData,omitempty"` // Optional verbose data from providers (e.g., STT segments, timestamps)
}

func NewResultMetadata() *ResultMetadata {
	return &ResultMetadata{
		Duration:   0,
		ModelUsed:  "",
		FieldCount: 0,
	}
}

func (m *ResultMetadata) AddCost(cost float64) {
	m.Cost += cost
}

func (m *ResultMetadata) AddTokens(tokens int) {
	m.TokensUsed += tokens
}

func (m *ResultMetadata) IncrementFieldCount() {
	m.FieldCount++
}

// Constraints defines generation constraints
type Constraints struct {
	MaxRetries     int
	Timeout        time.Duration
	MaxConcurrency int
	VoteQuality    int
}

func DefaultConstraints() *Constraints {
	return &Constraints{
		MaxRetries:     3,
		Timeout:        5 * time.Minute,
		MaxConcurrency: 10,
		VoteQuality:    85,
	}
}

// FieldTask represents a single field generation task
type FieldTask struct {
	id           string
	key          string
	definition   *jsonSchema.Definition
	parent       *FieldTask
	dependencies []string
	path         []string
	priority     int
}

func NewFieldTask(key string, definition *jsonSchema.Definition, parent *FieldTask) *FieldTask {
	path := []string{key}
	if parent != nil {
		path = append(parent.Path(), key)
	}

	return &FieldTask{
		id:           generateTaskID(key, path),
		key:          key,
		definition:   definition,
		parent:       parent,
		dependencies: make([]string, 0),
		path:         path,
		priority:     0,
	}
}

func (f *FieldTask) ID() string                         { return f.id }
func (f *FieldTask) Key() string                        { return f.key }
func (f *FieldTask) Definition() *jsonSchema.Definition { return f.definition }
func (f *FieldTask) Parent() *FieldTask                 { return f.parent }
func (f *FieldTask) Dependencies() []string             { return f.dependencies }
func (f *FieldTask) Path() []string                     { return f.path }
func (f *FieldTask) Priority() int                      { return f.priority }
func (f *FieldTask) HasDependencies() bool              { return len(f.dependencies) > 0 }

func (f *FieldTask) WithDependency(dep string) *FieldTask {
	newDeps := append([]string{}, f.dependencies...)
	newDeps = append(newDeps, dep)

	newTask := *f
	newTask.dependencies = newDeps
	return &newTask
}

func (f *FieldTask) WithPriority(priority int) *FieldTask {
	newTask := *f
	newTask.priority = priority
	return &newTask
}

func generateTaskID(key string, path []string) string {
	// Simple ID generation - could be made more sophisticated
	id := key
	for _, p := range path {
		id += "_" + p
	}
	return id
}

// TaskResult represents the result of a field task
type TaskResult struct {
	taskID   string
	key      string
	value    interface{}
	metadata *ResultMetadata
	path     []string
	err      error
}

type TaskChoice struct {
	Value      interface{}
	Embedding  []float64
	Score      float64
	Confidence float64
}

func NewTaskResult(taskID, key string, value interface{}, metadata *ResultMetadata) *TaskResult {
	return &TaskResult{
		taskID:   taskID,
		key:      key,
		value:    value,
		metadata: metadata,
		err:      nil,
	}
}

func NewTaskResultWithError(taskID, key string, err error) *TaskResult {
	return &TaskResult{
		taskID: taskID,
		key:    key,
		err:    err,
	}
}

func (t *TaskResult) TaskID() string            { return t.taskID }
func (t *TaskResult) Key() string               { return t.key }
func (t *TaskResult) Value() interface{}        { return t.value }
func (t *TaskResult) Metadata() *ResultMetadata { return t.metadata }
func (t *TaskResult) Path() []string            { return t.path }
func (t *TaskResult) Error() error              { return t.err }
func (t *TaskResult) IsSuccess() bool           { return t.err == nil }

func (t *TaskResult) WithPath(path []string) *TaskResult {
	newResult := *t
	newResult.path = path
	return &newResult
}
