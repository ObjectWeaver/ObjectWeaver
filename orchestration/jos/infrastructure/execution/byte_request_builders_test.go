package execution

import (
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
)

func TestNewTTSRequestBuilder(t *testing.T) {
	builder := NewTTSRequestBuilder()
	if builder == nil {
		t.Error("Expected non-nil TTSRequestBuilder")
	}
}

func TestTTSRequestBuilder_BuildRequest_Success(t *testing.T) {
	builder := NewTTSRequestBuilder()

	def := &jsonSchema.Definition{
		Instruction: "Speak clearly",
		TextToSpeech: &jsonSchema.TextToSpeech{
			StringToAudio: "Hello world",
			Voice:         "alloy",
			Model:         "tts-1",
		},
	}
	task := domain.NewFieldTask("test", def, nil)

	request, err := builder.BuildRequest(task, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if request.Model != "tts-1" {
		t.Errorf("Expected model 'tts-1', got %v", request.Model)
	}

	expectedInput := "Speak clearly\nHello world"
	if request.Input != expectedInput {
		t.Errorf("Expected input '%s', got '%s'", expectedInput, request.Input)
	}

	if request.Voice != "alloy" {
		t.Errorf("Expected voice 'alloy', got %v", request.Voice)
	}

	if request.ResponseFormat != "mp3" {
		t.Errorf("Expected response format 'mp3', got %v", request.ResponseFormat)
	}

	if request.Speed != 1.0 {
		t.Errorf("Expected speed 1.0, got %v", request.Speed)
	}
}

func TestTTSRequestBuilder_BuildRequest_NilConfig(t *testing.T) {
	builder := NewTTSRequestBuilder()

	def := &jsonSchema.Definition{
		TextToSpeech: nil,
	}
	task := domain.NewFieldTask("test", def, nil)

	_, err := builder.BuildRequest(task, nil)
	if err == nil {
		t.Error("Expected error for nil TTS config")
	}

	expectedMsg := "text-to-speech configuration is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestTTSRequestBuilder_BuildRequest_PassThrough(t *testing.T) {
	builder := NewTTSRequestBuilder()
	model := "any-model"
	def := &jsonSchema.Definition{
		TextToSpeech: &jsonSchema.TextToSpeech{
			StringToAudio: "test",
			Voice:         "alloy",
			Model:         model,
		},
	}
	task := domain.NewFieldTask("test", def, nil)
	request, err := builder.BuildRequest(task, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}
	if request.Model != model {
		t.Errorf("Expected model '%s', got '%s'", model, request.Model)
	}
}

func TestNewImageRequestBuilder(t *testing.T) {
	builder := NewImageRequestBuilder()
	if builder == nil {
		t.Error("Expected non-nil ImageRequestBuilder")
	}
}

func TestImageRequestBuilder_BuildRequest_Success(t *testing.T) {
	builder := NewImageRequestBuilder()

	def := &jsonSchema.Definition{
		Instruction: "A beautiful sunset",
		Image: &jsonSchema.Image{
			Size:  "512x512",
			Model: "dall-e-3",
		},
	}
	task := domain.NewFieldTask("test", def, nil)

	request, err := builder.BuildRequest(task, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if request.Model != "dall-e-3" {
		t.Errorf("Expected model 'dall-e-3', got %v", request.Model)
	}

	if request.Prompt != "A beautiful sunset" {
		t.Errorf("Expected prompt 'A beautiful sunset', got %v", request.Prompt)
	}

	if request.Size != "512x512" {
		t.Errorf("Expected size '512x512', got %v", request.Size)
	}

	if request.ResponseFormat != "b64_json" {
		t.Errorf("Expected response format 'b64_json', got %v", request.ResponseFormat)
	}

	if request.N != 1 {
		t.Errorf("Expected N=1, got %v", request.N)
	}
}

func TestImageRequestBuilder_BuildRequest_NilConfig(t *testing.T) {
	builder := NewImageRequestBuilder()

	def := &jsonSchema.Definition{
		Image: nil,
	}
	task := domain.NewFieldTask("test", def, nil)

	_, err := builder.BuildRequest(task, nil)
	if err == nil {
		t.Error("Expected error for nil Image config")
	}

	expectedMsg := "image configuration is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestImageRequestBuilder_BuildRequest_NoPrompt(t *testing.T) {
	builder := NewImageRequestBuilder()

	def := &jsonSchema.Definition{
		Instruction: "",
		Image: &jsonSchema.Image{
			Model: "dall-e-3",
		},
	}
	task := domain.NewFieldTask("test", def, nil)

	_, err := builder.BuildRequest(task, nil)
	if err == nil {
		t.Error("Expected error for missing prompt")
	}

	expectedMsg := "image generation requires an instruction field as the prompt"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestImageRequestBuilder_BuildRequest_DefaultSize(t *testing.T) {
	builder := NewImageRequestBuilder()

	def := &jsonSchema.Definition{
		Instruction: "test",
		Image: &jsonSchema.Image{
			Size:  "",
			Model: "dall-e-3",
		},
	}
	task := domain.NewFieldTask("test", def, nil)

	request, err := builder.BuildRequest(task, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if request.Size != "1024x1024" {
		t.Errorf("Expected default size '1024x1024', got %v", request.Size)
	}
}

func TestImageRequestBuilder_BuildRequest_PassThrough(t *testing.T) {
	builder := NewImageRequestBuilder()
	model := "dall-e-3"
	def := &jsonSchema.Definition{
		Instruction: "test",
		Image: &jsonSchema.Image{
			Model: model,
		},
	}
	task := domain.NewFieldTask("test", def, nil)
	request, err := builder.BuildRequest(task, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}
	if request.Model != model {
		t.Errorf("Expected model '%s', got '%s'", model, request.Model)
	}
}

func TestNewSTTRequestBuilder(t *testing.T) {
	builder := NewSTTRequestBuilder()
	if builder == nil {
		t.Error("Expected non-nil STTRequestBuilder")
	}
}

func TestSTTRequestBuilder_BuildRequest_Success(t *testing.T) {
	builder := NewSTTRequestBuilder()

	def := &jsonSchema.Definition{
		Instruction: "Transcribe clearly",
		SpeechToText: &jsonSchema.SpeechToText{
			AudioToTranscribe: []byte("audio data"),
			Language:          "en",
			ToString:          true,
			Model:             "whisper-1",
		},
	}
	task := domain.NewFieldTask("test", def, nil)

	request, err := builder.BuildRequest(task, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if request.Model != "whisper-1" {
		t.Errorf("Expected model 'whisper-1', got %v", request.Model)
	}

	if string(request.AudioData) != "audio data" {
		t.Errorf("Expected audio data 'audio data', got %v", string(request.AudioData))
	}

	if request.Language != "en" {
		t.Errorf("Expected language 'en', got %v", request.Language)
	}

	if request.ResponseFormat != "text" {
		t.Errorf("Expected response format 'text', got %v", request.ResponseFormat)
	}

	if request.Prompt != "Transcribe clearly" {
		t.Errorf("Expected prompt 'Transcribe clearly', got %v", request.Prompt)
	}
}

func TestSTTRequestBuilder_BuildRequest_NilConfig(t *testing.T) {
	builder := NewSTTRequestBuilder()

	def := &jsonSchema.Definition{
		SpeechToText: nil,
	}
	task := domain.NewFieldTask("test", def, nil)

	_, err := builder.BuildRequest(task, nil)
	if err == nil {
		t.Error("Expected error for nil STT config")
	}

	expectedMsg := "speech-to-text configuration is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestSTTRequestBuilder_BuildRequest_NoAudioData(t *testing.T) {
	builder := NewSTTRequestBuilder()

	def := &jsonSchema.Definition{
		SpeechToText: &jsonSchema.SpeechToText{
			AudioToTranscribe: []byte{},
		},
	}
	task := domain.NewFieldTask("test", def, nil)

	_, err := builder.BuildRequest(task, nil)
	if err == nil {
		t.Error("Expected error for missing audio data")
	}

	expectedMsg := "speech-to-text requires audio data in AudioToTranscribe field"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestSTTRequestBuilder_BuildRequest_ResponseFormat(t *testing.T) {
	builder := NewSTTRequestBuilder()

	tests := []struct {
		toString   bool
		toCaptions bool
		expected   string
	}{
		{false, false, "json"},
		{true, false, "text"},
		{false, true, "srt"},
		{true, true, "text"}, // ToString takes precedence
	}

	for _, test := range tests {
		def := &jsonSchema.Definition{
			SpeechToText: &jsonSchema.SpeechToText{
				AudioToTranscribe: []byte("audio"),
				ToString:          test.toString,
				ToCaptions:        test.toCaptions,
			},
		}
		task := domain.NewFieldTask("test", def, nil)

		request, err := builder.BuildRequest(task, nil)
		if err != nil {
			t.Fatalf("BuildRequest failed: %v", err)
		}

		if request.ResponseFormat != test.expected {
			t.Errorf("For ToString=%v, ToCaptions=%v, expected format '%s', got '%s'",
				test.toString, test.toCaptions, test.expected, request.ResponseFormat)
		}
	}
}

func TestSTTRequestBuilder_BuildRequest_PassThrough(t *testing.T) {
	builder := NewSTTRequestBuilder()
	model := "whisper-1"
	def := &jsonSchema.Definition{
		SpeechToText: &jsonSchema.SpeechToText{
			AudioToTranscribe: []byte("audio"),
			Model:             model,
		},
	}
	task := domain.NewFieldTask("test", def, nil)
	request, err := builder.BuildRequest(task, nil)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}
	if request.Model != model {
		t.Errorf("Expected model '%s', got '%s'", model, request.Model)
	}
}
