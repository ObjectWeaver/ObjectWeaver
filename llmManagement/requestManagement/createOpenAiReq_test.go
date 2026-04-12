package requestManagement

import (
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/modelConverter"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	gogpt "github.com/sashabaranov/go-openai"
)

func TestMin(t *testing.T) {
	if min(1, 2) != 1 {
		t.Error("min(1,2) should be 1")
	}
	if min(2, 1) != 1 {
		t.Error("min(2,1) should be 1")
	}
	if min(3, 3) != 3 {
		t.Error("min(3,3) should be 3")
	}
}

func TestDetectMimeType(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	if got := detectMimeType(jpegData); got != "image/jpeg" {
		t.Errorf("detectMimeType(jpeg) = %s, want image/jpeg", got)
	}

	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if got := detectMimeType(pngData); got != "image/png" {
		t.Errorf("detectMimeType(png) = %s, want image/png", got)
	}

	unknownData := []byte{0x00, 0x01, 0x02}
	if got := detectMimeType(unknownData); got != "image/jpeg" {
		t.Errorf("detectMimeType(unknown) = %s, want image/jpeg", got)
	}
}

func TestToBase64DataURL(t *testing.T) {
	data := []byte("test data")
	mimeType := "image/jpeg"
	expected := "data:image/jpeg;base64,dGVzdCBkYXRh"
	got := toBase64DataURL(data, mimeType)
	if got != expected {
		t.Errorf("toBase64DataURL = %s, want %s", got, expected)
	}
}

func TestPromptWriter(t *testing.T) {
	prompt := "Hello world"
	msg := promptWriter(prompt)
	if msg.Role != gogpt.ChatMessageRoleUser {
		t.Errorf("Role = %s, want %s", msg.Role, gogpt.ChatMessageRoleUser)
	}
	if len(msg.MultiContent) != 1 {
		t.Errorf("MultiContent length = %d, want 1", len(msg.MultiContent))
	}
	if msg.MultiContent[0].Type != gogpt.ChatMessagePartTypeText {
		t.Errorf("Type = %s, want %s", msg.MultiContent[0].Type, gogpt.ChatMessagePartTypeText)
	}
	if msg.MultiContent[0].Text != prompt {
		t.Errorf("Text = %s, want %s", msg.MultiContent[0].Text, prompt)
	}
}

func TestBuildRequest_TextOnly(t *testing.T) {
	converter := modelConverter.NewModelConverter()
	builder := NewDefaultOpenAIReqBuilder(converter)

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{ // Using jsonSchema.Definition
			Model:       "gpt4",
			Stream:      false,
			SendImage:   nil,
			ModelConfig: &jsonSchema.ModelConfig{Temperature: 0.5, Seed: nil},
		},
		Prompt:       "Test prompt",
		SystemPrompt: "System prompt",
	}

	req, err := builder.BuildRequest(inputs)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if req.Model != "gpt4" { // Since not converted
		t.Errorf("Model = %s, want gpt4", req.Model)
	}
	if req.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want 0.5", req.Temperature)
	}
	if req.Stream != false {
		t.Errorf("Stream = %v, want false", req.Stream)
	}
	if len(req.Messages) != 2 {
		t.Errorf("Messages length = %d, want 2", len(req.Messages))
	}
	if req.Messages[0].Role != gogpt.ChatMessageRoleSystem {
		t.Errorf("First message role = %s, want system", req.Messages[0].Role)
	}
	if req.Messages[1].Role != gogpt.ChatMessageRoleUser {
		t.Errorf("Second message role = %s, want user", req.Messages[1].Role)
	}
}

func TestBuildRequest_WithImages(t *testing.T) {
	converter := modelConverter.NewModelConverter()
	builder := NewDefaultOpenAIReqBuilder(converter)

	imageData := [][]byte{
		{0xFF, 0xD8, 0xFF, 0xE0}, // jpeg
	}

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model:       "gpt4",
			ModelConfig: &jsonSchema.ModelConfig{Temperature: 0.5, Seed: nil},
			Stream:      false,
			SendImage: &jsonSchema.SendImage{
				ImagesData: imageData,
			},
		},
		Prompt:       "Test prompt",
		SystemPrompt: "System prompt",
	}

	req, err := builder.BuildRequest(inputs)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if len(req.Messages) != 3 { // system, prompt, image
		t.Errorf("Messages length = %d, want 3", len(req.Messages))
	}
}

func TestBuildRequest_ReasoningModel_Stream(t *testing.T) {
	converter := modelConverter.NewModelConverter()
	builder := NewDefaultOpenAIReqBuilder(converter)

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model: "o3-mini-2025-01-31",
			ModelConfig: &jsonSchema.ModelConfig{
				Temperature:     0.5,
				Seed:            nil,
				ReasoningEffort: "medium",
			},
			Stream:    true,
			SendImage: nil,
		},
		Prompt:       "Test prompt",
		SystemPrompt: "System prompt",
	}

	req, err := builder.BuildRequest(inputs)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if req.ReasoningEffort != "medium" {
		t.Errorf("ReasoningEffort = %s, want medium", req.ReasoningEffort)
	}
	// Temperature should be set to 1.0 for reasoning models, but let's check actual value
	if req.Stream != true {
		t.Errorf("Stream = %v, want true", req.Stream)
	}
}

func TestBuildRequest_ReasoningModel_NonStream(t *testing.T) {
	converter := modelConverter.NewModelConverter()
	builder := NewDefaultOpenAIReqBuilder(converter)

	inputs := &llmManagement.Inputs{
		Def: &jsonSchema.Definition{
			Model: "o3-mini-2025-01-31",
			ModelConfig: &jsonSchema.ModelConfig{
				Temperature:     0.5,
				Seed:            nil,
				ReasoningEffort: "medium",
			},
			Stream:    false,
			SendImage: nil,
		},
		Prompt:       "Test prompt",
		SystemPrompt: "System prompt",
	}

	req, err := builder.BuildRequest(inputs)
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}

	if req.ReasoningEffort != "medium" {
		t.Errorf("ReasoningEffort = %s, want medium", req.ReasoningEffort)
	}
	if req.Stream != false {
		t.Errorf("Stream = %v, want false", req.Stream)
	}
}
