package jobSubmitter

import (
	"context"
	"testing"

	"objectweaver/llmManagement"
	"objectweaver/llmManagement/LLM"
	"objectweaver/llmManagement/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
	"github.com/sashabaranov/go-openai"
)

// MockJobSubmitter implements JobSumitter interface for testing
type MockJobSubmitter struct {
	response string
	usage    *openai.Usage
	err      error
}

func (m *MockJobSubmitter) SubmitJob(job *LLM.Job, workerChannel chan *LLM.Job) (any, *openai.Usage, error) {
	if m.err != nil {
		return "", nil, m.err
	}
	return m.response, m.usage, nil
}

// TestableJobEntryPoint is a testable version that accepts a submitter
type TestableJobEntryPoint struct {
	submitter LLM.JobSumitter
}

func NewTestableJobEntryPoint(submitter LLM.JobSumitter) JobEntryPoint {
	return &TestableJobEntryPoint{submitter: submitter}
}

func (t *TestableJobEntryPoint) SubmitJob(ctx context.Context, model string, def *jsonSchema.Definition, newPrompt, systemPrompt string, outStream chan interface{}) (any, *openai.Usage, error) {
	// If def is nil, create a minimal definition with the model
	if def == nil {
		def = &jsonSchema.Definition{
			Model: model,
		}
	} else {
		def.Model = model
	}

	if def.SendImage == nil {
		def.SendImage = &jsonSchema.SendImage{}
		def.SendImage.ImagesData = nil
	}

	job := &LLM.Job{
		Inputs: &llmManagement.Inputs{
			Ctx:          ctx,
			Prompt:       newPrompt,
			SystemPrompt: systemPrompt,
			Def:          def,
		},
		Result:   make(chan *domain.JobResult, 1),
		Tokens:   0,
		Priority: def.Priority,
	}

	return t.submitter.SubmitJob(job, nil)
}

func TestDefaultJobEntryPoint_SubmitJob(t *testing.T) {
	usage := &openai.Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}
	mockSubmitter := &MockJobSubmitter{
		response: "Test response",
		usage:    usage,
		err:      nil,
	}
	entryPoint := NewTestableJobEntryPoint(mockSubmitter)
	model := string("gpt-3.5-turbo")
	def := &jsonSchema.Definition{
		Model: model,
	}
	newPrompt := "Test prompt"
	systemPrompt := "System prompt"
	outStream := make(chan interface{}, 1)

	response, usageResult, err := entryPoint.SubmitJob(context.Background(), model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if response != "Test response" {
		t.Errorf("Expected 'Test response', got %s", response)
	}
	if usageResult.TotalTokens != 30 {
		t.Errorf("Expected 30 total tokens, got %d", usageResult.TotalTokens)
	}
}

func TestDefaultJobEntryPoint_SubmitJob_DefNil(t *testing.T) {
	usage := &openai.Usage{}
	mockSubmitter := &MockJobSubmitter{
		response: "Response",
		usage:    usage,
		err:      nil,
	}
	entryPoint := NewTestableJobEntryPoint(mockSubmitter)
	model := string("gpt-3.5-turbo")
	var def *jsonSchema.Definition = nil
	newPrompt := "Test prompt"
	systemPrompt := "System prompt"
	outStream := make(chan interface{}, 1)

	response, _, err := entryPoint.SubmitJob(context.Background(), model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if response != "Response" {
		t.Errorf("Expected 'Response', got %s", response)
	}
}

func TestDefaultJobEntryPoint_SubmitJob_SetsModel(t *testing.T) {
	usage := &openai.Usage{}
	mockSubmitter := &MockJobSubmitter{
		response: "Ok",
		usage:    usage,
		err:      nil,
	}
	entryPoint := NewTestableJobEntryPoint(mockSubmitter)
	model := string("gpt-4")
	def := &jsonSchema.Definition{
		// Model not set initially
	}
	newPrompt := "Prompt"
	systemPrompt := "Sys"
	outStream := make(chan interface{}, 1)

	_, _, err := entryPoint.SubmitJob(context.Background(), model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	if def.Model != model {
		t.Errorf("Expected model %s, got %s", model, def.Model)
	}
}

func TestDefaultJobEntryPoint_SubmitJob_InitializesSendImage(t *testing.T) {
	usage := &openai.Usage{}
	mockSubmitter := &MockJobSubmitter{
		response: "Response",
		usage:    usage,
		err:      nil,
	}
	entryPoint := NewTestableJobEntryPoint(mockSubmitter)
	model := string("gpt-3.5-turbo")
	def := &jsonSchema.Definition{
		Model: model,
		// SendImage nil
	}
	newPrompt := "Test prompt"
	systemPrompt := "System prompt"
	outStream := make(chan interface{}, 1)

	_, _, err := entryPoint.SubmitJob(context.Background(), model, def, newPrompt, systemPrompt, outStream)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if def.SendImage == nil {
		t.Error("Expected SendImage to be initialized")
	}
}
