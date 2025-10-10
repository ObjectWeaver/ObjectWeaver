package execution

import (
	"fmt"
	"firechimp/orchestration/jos/domain"

	"github.com/henrylamb/object-generation-golang/jsonSchema"
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

	// Map model type to actual OpenAI model name
	var model string
	modelStr := string(tts.Model)
	switch modelStr {
	case "tts", "OpenAiTTS":
		model = "tts-1" // Standard quality
	case "tts-hd", "OpenAiTTSHD":
		model = "tts-1-hd" // High definition
	case "tts-1", "tts-1-hd":
		// Already in correct format
		model = modelStr
	default:
		// Default to standard TTS
		model = "tts-1"
	}

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

	// Determine model
	var model string
	switch img.Model {
	case jsonSchema.OpenAiDalle2:
		model = "dall-e-2"
	case jsonSchema.OpenAiDalle3:
		model = "dall-e-3"
	default:
		return nil, fmt.Errorf("unsupported image model: %s", img.Model)
	}

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

	// Map model type to actual OpenAI model name
	var model string
	modelStr := string(stt.Model)
	switch modelStr {
	case "OpenAiWhisper", "whisper":
		model = "whisper-1"
	case "whisper-1":
		model = modelStr
	default:
		// Default to whisper-1
		model = "whisper-1"
	}

	return &domain.AudioTranscriptionRequest{
		Model:          model,
		AudioData:      stt.AudioToTranscribe,
		Language:       stt.Language,
		ResponseFormat: responseFormat,
		Prompt:         task.Definition().Instruction,
	}, nil
}
