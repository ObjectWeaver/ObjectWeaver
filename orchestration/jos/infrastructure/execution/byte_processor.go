package execution

import (
	"context"
	"encoding/base64"
	"fmt"
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// ByteProcessor handles byte data type for TTS, Image Generation, and STT
// It constructs requests using builders and delegates to ByteOperationProvider
type ByteProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	systemPromptProvider SystemPromptProvider
	ttsBuilder           *TTSRequestBuilder
	imageBuilder         *ImageRequestBuilder
	sttBuilder           *STTRequestBuilder
}

func NewByteProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *ByteProcessor {
	return &ByteProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: NewDefaultSystemPromptProvider(),
		ttsBuilder:           NewTTSRequestBuilder(),
		imageBuilder:         NewImageRequestBuilder(),
		sttBuilder:           NewSTTRequestBuilder(),
	}
}

func NewByteProcessorWithPromptProvider(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder, promptProvider SystemPromptProvider) *ByteProcessor {
	return &ByteProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: promptProvider,
		ttsBuilder:           NewTTSRequestBuilder(),
		imageBuilder:         NewImageRequestBuilder(),
		sttBuilder:           NewSTTRequestBuilder(),
	}
}

func (p *ByteProcessor) CanProcess(schemaType jsonSchema.DataType) bool {
	// ByteProcessor only handles byte type for type-based selection
	// STT fields (type: string) are routed here by CompositeTaskExecutor's special check
	return schemaType == jsonSchema.Byte
}

func (p *ByteProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	def := task.Definition()

	// Check if provider supports byte operations
	byteProvider, ok := p.llmProvider.(domain.ByteOperationProvider)
	if !ok || !byteProvider.SupportsByteOperations() {
		return nil, fmt.Errorf("LLM provider does not support byte operations")
	}

	// Handle Text-to-Speech (TTS)
	if def.TextToSpeech != nil {
		return p.processTextToSpeech(task, execContext, byteProvider)
	}

	// Handle Image Generation
	if def.Image != nil {
		return p.processImageGeneration(task, execContext, byteProvider)
	}

	// Handle Speech-to-Text (STT)
	if def.SpeechToText != nil {
		return p.processSpeechToText(task, execContext, byteProvider)
	}

	// Fallback: if no specific byte operation is defined, return error
	return nil, fmt.Errorf("byte type requires TextToSpeech, Image, or SpeechToText configuration")
}

// processTextToSpeech handles text-to-speech conversion
func (p *ByteProcessor) processTextToSpeech(task *domain.FieldTask, context *domain.ExecutionContext, byteProvider domain.ByteOperationProvider) (*domain.TaskResult, error) {
	// Build request using the builder
	request, err := p.ttsBuilder.BuildRequest(task, context)
	if err != nil {
		return nil, fmt.Errorf("failed to build TTS request: %w", err)
	}

	// Delegate to the byte operation provider
	audioBytes, metadata, err := byteProvider.GenerateAudio(request)
	if err != nil {
		return nil, fmt.Errorf("TTS generation failed: %w", err)
	}

	// Encode to base64 for JSON compatibility
	base64Audio := base64.StdEncoding.EncodeToString(audioBytes)

	// Convert provider metadata to result metadata
	resultMetadata := domain.NewResultMetadata()
	resultMetadata.ModelUsed = metadata.Model
	resultMetadata.Cost = metadata.Cost
	resultMetadata.TokensUsed = metadata.TokensUsed

	result := domain.NewTaskResult(task.ID(), task.Key(), base64Audio, resultMetadata)
	return result.WithPath(task.Path()), nil
}

// processImageGeneration handles image generation
func (p *ByteProcessor) processImageGeneration(task *domain.FieldTask, context *domain.ExecutionContext, byteProvider domain.ByteOperationProvider) (*domain.TaskResult, error) {
	// Build request using the builder
	request, err := p.imageBuilder.BuildRequest(task, context)
	if err != nil {
		return nil, fmt.Errorf("failed to build image request: %w", err)
	}

	// Delegate to the byte operation provider
	imageBytes, metadata, err := byteProvider.GenerateImage(request)
	if err != nil {
		return nil, fmt.Errorf("image generation failed: %w", err)
	}

	// Encode to base64 for JSON compatibility
	base64Image := base64.StdEncoding.EncodeToString(imageBytes)

	// Convert provider metadata to result metadata
	resultMetadata := domain.NewResultMetadata()
	resultMetadata.ModelUsed = metadata.Model
	resultMetadata.Cost = metadata.Cost
	resultMetadata.TokensUsed = metadata.TokensUsed

	result := domain.NewTaskResult(task.ID(), task.Key(), base64Image, resultMetadata)
	return result.WithPath(task.Path()), nil
}

// processSpeechToText handles speech-to-text conversion
func (p *ByteProcessor) processSpeechToText(task *domain.FieldTask, context *domain.ExecutionContext, byteProvider domain.ByteOperationProvider) (*domain.TaskResult, error) {
	// Build request using the builder
	request, err := p.sttBuilder.BuildRequest(task, context)
	if err != nil {
		return nil, fmt.Errorf("failed to build STT request: %w", err)
	}

	// Delegate to the byte operation provider
	transcriptionText, metadata, err := byteProvider.TranscribeAudio(request)
	if err != nil {
		return nil, fmt.Errorf("speech-to-text failed: %w", err)
	}

	// Convert provider metadata to result metadata
	resultMetadata := domain.NewResultMetadata()
	resultMetadata.ModelUsed = metadata.Model
	resultMetadata.Cost = metadata.Cost
	resultMetadata.TokensUsed = metadata.TokensUsed
	resultMetadata.VerboseData = metadata.VerboseData // Preserve verbose data from provider

	result := domain.NewTaskResult(task.ID(), task.Key(), transcriptionText, resultMetadata)
	return result.WithPath(task.Path()), nil
}
