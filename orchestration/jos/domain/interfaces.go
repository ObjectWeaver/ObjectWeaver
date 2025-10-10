package domain

import (
	"github.com/objectweaver/go-sdk/jsonSchema"
)

// Generator - Primary domain service interface
type Generator interface {
	Generate(request *GenerationRequest) (*GenerationResult, error)
	GenerateStream(request *GenerationRequest) (<-chan *StreamChunk, error)
	GenerateStreamProgressive(request *GenerationRequest) (<-chan *AccumulatedStreamChunk, error)
}

// SchemaAnalyzer - Analyzes and breaks down schemas
type SchemaAnalyzer interface {
	Analyze(schema *jsonSchema.Definition) (*SchemaAnalysis, error)
	ExtractFields(schema *jsonSchema.Definition) ([]*FieldDefinition, error)
	DetermineProcessingOrder(fields []*FieldDefinition) ([]*FieldTask, error)
}

// SchemaAnalysis represents the analyzed schema structure
type SchemaAnalysis struct {
	Fields           []*FieldDefinition
	TotalFieldCount  int
	MaxDepth         int
	HasNestedObjects bool
}

// FieldDefinition represents a field in the schema
type FieldDefinition struct {
	Key        string
	Definition *jsonSchema.Definition
	Parent     *FieldDefinition
	Required   bool
}

// TaskExecutor - Executes field generation tasks
type TaskExecutor interface {
	Execute(task *FieldTask, context *ExecutionContext) (*TaskResult, error)
	ExecuteBatch(tasks []*FieldTask, context *ExecutionContext) ([]*TaskResult, error)
}

// ExecutionContext provides context for task execution
type ExecutionContext struct {
	request          *GenerationRequest
	parentContext    *ExecutionContext
	generatedValues  map[string]interface{}
	metadata         map[string]interface{}
	promptContext    *PromptContext
	generationConfig *GenerationConfig
}

func NewExecutionContext(request *GenerationRequest) *ExecutionContext {
	return &ExecutionContext{
		request:          request,
		generatedValues:  make(map[string]interface{}),
		metadata:         make(map[string]interface{}),
		promptContext:    NewPromptContext(),
		generationConfig: DefaultGenerationConfig(),
	}
}

func (e *ExecutionContext) WithParent(parent *FieldTask) *ExecutionContext {
	return &ExecutionContext{
		request:          e.request,
		parentContext:    e,
		generatedValues:  e.copyGeneratedValues(),
		metadata:         e.copyMetadata(),
		promptContext:    e.promptContext,
		generationConfig: e.generationConfig,
	}
}

func (e *ExecutionContext) Request() *GenerationRequest             { return e.request }
func (e *ExecutionContext) GeneratedValues() map[string]interface{} { return e.generatedValues }
func (e *ExecutionContext) PromptContext() *PromptContext           { return e.promptContext }
func (e *ExecutionContext) GenerationConfig() *GenerationConfig     { return e.generationConfig }

func (e *ExecutionContext) SetGeneratedValue(key string, value interface{}) {
	e.generatedValues[key] = value
}

func (e *ExecutionContext) copyGeneratedValues() map[string]interface{} {
	copied := make(map[string]interface{})
	for k, v := range e.generatedValues {
		copied[k] = v
	}
	return copied
}

func (e *ExecutionContext) copyMetadata() map[string]interface{} {
	copied := make(map[string]interface{})
	for k, v := range e.metadata {
		copied[k] = v
	}
	return copied
}

// PromptBuilder - Builds prompts for field generation
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

// LLMProvider - Abstract LLM interaction
type LLMProvider interface {
	Generate(prompt string, config *GenerationConfig) (string, *ProviderMetadata, error)
	SupportsStreaming() bool
	ModelType() string
}

// TokenStreamingProvider - LLM that supports token streaming
type TokenStreamingProvider interface {
	LLMProvider
	GenerateStream(prompt string, config *GenerationConfig) (<-chan string, error)
	GenerateTokenStream(prompt string, config *GenerationConfig) (<-chan *TokenChunk, error)
	SupportsTokenStreaming() bool
}

// ByteOperationProvider - Provider that supports byte operations (TTS, Image, STT)
type ByteOperationProvider interface {
	GenerateAudio(request *AudioGenerationRequest) ([]byte, *ProviderMetadata, error)
	GenerateImage(request *ImageGenerationRequest) ([]byte, *ProviderMetadata, error)
	TranscribeAudio(request *AudioTranscriptionRequest) (string, *ProviderMetadata, error)
	SupportsByteOperations() bool
}

// AudioGenerationRequest contains parameters for text-to-speech
type AudioGenerationRequest struct {
	Model          string
	Input          string
	Voice          string
	ResponseFormat string
	Speed          float64
}

// ImageGenerationRequest contains parameters for image generation
type ImageGenerationRequest struct {
	Model          string
	Prompt         string
	Size           string
	ResponseFormat string
	N              int
}

// AudioTranscriptionRequest contains parameters for speech-to-text
type AudioTranscriptionRequest struct {
	Model          string
	AudioData      []byte
	Language       string
	ResponseFormat string
	Prompt         string
}

// ProviderMetadata contains metadata from LLM provider
type ProviderMetadata struct {
	TokensUsed   int
	Cost         float64
	Model        string
	FinishReason string
}

// GenerationConfig configures LLM generation
type GenerationConfig struct {
	Model         string
	Temperature   float64
	MaxTokens     int
	SystemPrompt  string
	Granularity   StreamGranularity
	BufferSize    int
	StopSequences []string
	Definition    *jsonSchema.Definition // Schema definition for this generation (includes SendImage, etc.)
}

func DefaultGenerationConfig() *GenerationConfig {
	return &GenerationConfig{
		Temperature:   0.7,
		MaxTokens:     2000,
		Granularity:   GranularityField,
		BufferSize:    10,
		StopSequences: []string{},
	}
}

// ResultAssembler - Assembles final result from tasks
type ResultAssembler interface {
	Assemble(results []*TaskResult) (*GenerationResult, error)
}

// StreamingAssembler - Assembles streaming results
type StreamingAssembler interface {
	ResultAssembler
	AssembleStreaming(results <-chan *TaskResult) (<-chan *StreamChunk, error)
}

// ProgressiveAssembler - Assembles progressive token-level results
type ProgressiveAssembler interface {
	ResultAssembler
	AssembleProgressive(tokenStream <-chan *TokenStreamChunk) (<-chan *AccumulatedStreamChunk, error)
}

// ExecutionStrategy - Determines how tasks are executed
type ExecutionStrategy interface {
	Schedule(tasks []*FieldTask) (*ExecutionPlan, error)
	Execute(plan *ExecutionPlan, executor TaskExecutor, context *ExecutionContext) ([]*TaskResult, error)
}

// ExecutionPlan represents a plan for executing tasks
type ExecutionPlan struct {
	Stages   []ExecutionStage
	Metadata map[string]interface{}
}

// ExecutionStage represents a stage in execution
type ExecutionStage struct {
	Tasks    []*FieldTask
	Parallel bool
}

// TypeProcessor - Handles different JSON schema types
type TypeProcessor interface {
	CanProcess(schemaType jsonSchema.DataType) bool
	Process(task *FieldTask, context *ExecutionContext) (*TaskResult, error)
}

// StreamingTypeProcessor - Type processor with streaming support
type StreamingTypeProcessor interface {
	TypeProcessor
	ProcessStreaming(task *FieldTask, context *ExecutionContext) (<-chan *TokenStreamChunk, error)
}

// ValidationPlugin - Validates results
type ValidationPlugin interface {
	Plugin
	Validate(result *GenerationResult, schema *jsonSchema.Definition) ([]ValidationError, error)
}

type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// Plugin - Base plugin interface
type Plugin interface {
	Name() string
	Version() string
	Initialize(config map[string]interface{}) error
}

// PreProcessorPlugin - Runs before generation
type PreProcessorPlugin interface {
	Plugin
	PreProcess(request *GenerationRequest) (*GenerationRequest, error)
}

// PostProcessorPlugin - Runs after generation
type PostProcessorPlugin interface {
	Plugin
	PostProcess(result *GenerationResult) (*GenerationResult, error)
}

// CachePlugin - Handles caching
type CachePlugin interface {
	Plugin
	Get(key string) (*GenerationResult, bool)
	Set(key string, result *GenerationResult) error
}

// ObservabilityPlugin - Handles metrics and tracing
type ObservabilityPlugin interface {
	Plugin
	RecordMetric(name string, value float64, tags map[string]string)
	StartSpan(name string) Span
}

// Span represents a tracing span
type Span interface {
	End()
	SetTag(key string, value interface{})
}
