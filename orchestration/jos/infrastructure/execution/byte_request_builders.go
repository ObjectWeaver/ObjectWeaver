package execution

import (
	"fmt"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
)

// TTSRequestBuilder constructs audio generation requests from field tasks
type TTSRequestBuilder struct{}

func NewTTSRequestBuilder() *TTSRequestBuilder {
	return &TTSRequestBuilder{}
}

func (b *TTSRequestBuilder) BuildRequest(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.AudioGenerationRequest, error) {
	tts := task.Definition().TextToSpeech
	if tts == nil {
		return nil, fmt.Errorf("text-to-speech configuration is nil")
	}

	// Build the input text (combining instruction and any selected fields)
	input := tts.StringToAudio
	if task.Definition().Instruction != "" {
		input = fmt.Sprintf("%s\n%s", task.Definition().Instruction, input)
	}

	model := string(tts.Model)

	return &domain.AudioGenerationRequest{
		Model:          model,
		Input:          input,
		Voice:          string(tts.Voice),
		ResponseFormat: "mp3", // Default format
		Speed:          1.0,
	}, nil
}

// ImageRequestBuilder constructs image generation requests from field tasks
type ImageRequestBuilder struct{}

func NewImageRequestBuilder() *ImageRequestBuilder {
	return &ImageRequestBuilder{}
}

func (b *ImageRequestBuilder) BuildRequest(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.ImageGenerationRequest, error) {
	img := task.Definition().Image
	if img == nil {
		return nil, fmt.Errorf("image configuration is nil")
	}

	// Use instruction as the image prompt
	prompt := task.Definition().Instruction
	if prompt == "" {
		return nil, fmt.Errorf("image generation requires an instruction field as the prompt")
	}

	// Determine image size
	size := "1024x1024"
	if img.Size != "" {
		size = string(img.Size)
	}

	model := string(img.Model)

	return &domain.ImageGenerationRequest{
		Model:          model,
		Prompt:         prompt,
		Size:           size,
		ResponseFormat: "b64_json",
		N:              1,
	}, nil
}

// STTRequestBuilder constructs audio transcription requests from field tasks
type STTRequestBuilder struct{}

func NewSTTRequestBuilder() *STTRequestBuilder {
	return &STTRequestBuilder{}
}

func (b *STTRequestBuilder) BuildRequest(task *domain.FieldTask, context *domain.ExecutionContext) (*domain.AudioTranscriptionRequest, error) {
	stt := task.Definition().SpeechToText
	if stt == nil {
		return nil, fmt.Errorf("speech-to-text configuration is nil")
	}

	// Validate audio data is present
	if len(stt.AudioToTranscribe) == 0 {
		return nil, fmt.Errorf("speech-to-text requires audio data in AudioToTranscribe field")
	}

	// Determine the response format based on configuration
	responseFormat := "json"
	if stt.ToString {
		responseFormat = "text"
	} else if stt.ToCaptions {
		responseFormat = "srt"
	}

	model := string(stt.Model)

	return &domain.AudioTranscriptionRequest{
		Model:          model,
		AudioData:      stt.AudioToTranscribe,
		Language:       stt.Language,
		ResponseFormat: responseFormat,
		Prompt:         task.Definition().Instruction,
	}, nil
}
