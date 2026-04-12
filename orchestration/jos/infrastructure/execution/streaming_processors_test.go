package execution

import (
	"testing"

	"objectweaver/orchestration/jos/domain"

	"objectweaver/jsonSchema"
)

// Mock implementations for testing
type mockTokenStreamingProvider struct {
	generateTokenStreamFunc func(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error)
}

func (m *mockTokenStreamingProvider) Generate(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
	return "", nil, nil
}

func (m *mockTokenStreamingProvider) GenerateStream(prompt string, config *domain.GenerationConfig) (<-chan any, error) {
	return nil, nil
}

func (m *mockTokenStreamingProvider) GenerateTokenStream(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
	if m.generateTokenStreamFunc != nil {
		return m.generateTokenStreamFunc(prompt, config)
	}
	return nil, nil
}

func (m *mockTokenStreamingProvider) SupportsStreaming() bool {
	return true
}

func (m *mockTokenStreamingProvider) SupportsTokenStreaming() bool {
	return true
}

func (m *mockTokenStreamingProvider) ModelType() string {
	return "gpt-4-0613"
}

func TestNewStreamingPrimitiveProcessor(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewStreamingPrimitiveProcessor(llmProvider, promptBuilder, domain.GranularityToken)

	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.granularity != domain.GranularityToken {
		t.Error("Expected granularity to be set")
	}
}

func TestStreamingPrimitiveProcessor_CanProcess(t *testing.T) {
	processor := NewStreamingPrimitiveProcessor(nil, nil, domain.GranularityToken)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.String, true},
		{jsonSchema.Number, true},
		{jsonSchema.Boolean, true},
		{jsonSchema.Integer, false},
		{jsonSchema.Object, false},
		{jsonSchema.Array, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestStreamingPrimitiveProcessor_Process(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{
		generateTokenStreamFunc: func(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
			ch := make(chan *domain.TokenChunk, 2)
			go func() {
				defer close(ch)
				ch <- &domain.TokenChunk{Token: "Hello", IsFinal: false}
				ch <- &domain.TokenChunk{Token: " world", IsFinal: true}
			}()
			return ch, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewStreamingPrimitiveProcessor(llmProvider, promptBuilder, domain.GranularityField)

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("message", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "message" {
		t.Errorf("Expected key 'message', got %v", result.Key())
	}

	value, ok := result.Value().(string)
	if !ok {
		t.Errorf("Expected string, got %T", result.Value())
	}

	if value != "Hello world" {
		t.Errorf("Expected 'Hello world', got %v", value)
	}
}

func TestStreamingPrimitiveProcessor_ProcessStreaming(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{
		generateTokenStreamFunc: func(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
			ch := make(chan *domain.TokenChunk, 3)
			go func() {
				defer close(ch)
				ch <- &domain.TokenChunk{Token: "Hi", IsFinal: false}
				ch <- &domain.TokenChunk{Token: "!", IsFinal: false}
				ch <- &domain.TokenChunk{Token: "", IsFinal: true}
			}()
			return ch, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewStreamingPrimitiveProcessor(llmProvider, promptBuilder, domain.GranularityToken)

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("greeting", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	stream, err := processor.ProcessStreaming(task, context)
	if err != nil {
		t.Fatalf("ProcessStreaming failed: %v", err)
	}

	var chunks []*domain.TokenStreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 4 {
		t.Errorf("Expected 4 chunks, got %d", len(chunks))
	}

	// Check first chunk
	if chunks[0].Key != "greeting" {
		t.Errorf("Expected key 'greeting', got %v", chunks[0].Key)
	}
	if chunks[0].Token != "Hi" {
		t.Errorf("Expected token 'Hi', got %v", chunks[0].Token)
	}
	if chunks[0].Partial != "Hi" {
		t.Errorf("Expected partial 'Hi', got %v", chunks[0].Partial)
	}
	if chunks[0].Complete {
		t.Error("Expected not complete")
	}

	// Check final chunk
	if !chunks[3].Complete {
		t.Error("Expected final chunk to be complete")
	}
	if chunks[3].Partial != "Hi!" {
		t.Errorf("Expected partial 'Hi!', got %v", chunks[3].Partial)
	}
}

func TestStreamingPrimitiveProcessor_shouldEmit(t *testing.T) {
	tests := []struct {
		granularity domain.StreamGranularity
		token       *domain.TokenChunk
		expected    bool
	}{
		{domain.GranularityToken, &domain.TokenChunk{Token: "a", IsFinal: false}, true},
		{domain.GranularityChunk, &domain.TokenChunk{Token: ".", IsFinal: false}, true},
		{domain.GranularityChunk, &domain.TokenChunk{Token: "!", IsFinal: false}, true},
		{domain.GranularityChunk, &domain.TokenChunk{Token: "?", IsFinal: false}, true},
		{domain.GranularityChunk, &domain.TokenChunk{Token: "a", IsFinal: false}, false},
		{domain.GranularityField, &domain.TokenChunk{Token: "a", IsFinal: false}, false},
		{domain.GranularityField, &domain.TokenChunk{Token: "a", IsFinal: true}, true},
	}

	for _, test := range tests {
		processor := NewStreamingPrimitiveProcessor(nil, nil, test.granularity)
		result := processor.shouldEmit(test.token)
		if result != test.expected {
			t.Errorf("shouldEmit(%v, %v) = %v, expected %v", test.granularity, test.token, result, test.expected)
		}
	}
}

func TestNewStreamingObjectProcessor(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewStreamingObjectProcessor(llmProvider, promptBuilder, domain.GranularityToken)

	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.granularity != domain.GranularityToken {
		t.Error("Expected granularity to be set")
	}
}

func TestStreamingObjectProcessor_CanProcess(t *testing.T) {
	processor := NewStreamingObjectProcessor(nil, nil, domain.GranularityToken)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Object, true},
		{jsonSchema.String, false},
		{jsonSchema.Array, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestStreamingObjectProcessor_Process(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{
		generateTokenStreamFunc: func(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
			ch := make(chan *domain.TokenChunk, 1)
			go func() {
				defer close(ch)
				ch <- &domain.TokenChunk{Token: "value", IsFinal: true}
			}()
			return ch, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewStreamingObjectProcessor(llmProvider, promptBuilder, domain.GranularityField)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"name": {Type: jsonSchema.String},
			"age":  {Type: jsonSchema.Number},
		},
	}
	task := domain.NewFieldTask("user", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "user" {
		t.Errorf("Expected key 'user', got %v", result.Key())
	}

	value, ok := result.Value().(map[string]interface{})
	if !ok {
		t.Errorf("Expected map, got %T", result.Value())
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(value))
	}
}

func TestStreamingObjectProcessor_ProcessStreaming(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{
		generateTokenStreamFunc: func(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
			ch := make(chan *domain.TokenChunk, 1)
			go func() {
				defer close(ch)
				ch <- &domain.TokenChunk{Token: "test", IsFinal: true}
			}()
			return ch, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewStreamingObjectProcessor(llmProvider, promptBuilder, domain.GranularityField)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"field1": {Type: jsonSchema.String},
		},
	}
	task := domain.NewFieldTask("obj", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	stream, err := processor.ProcessStreaming(task, context)
	if err != nil {
		t.Fatalf("ProcessStreaming failed: %v", err)
	}

	var chunks []*domain.TokenStreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Key != "field1" {
		t.Errorf("Expected key 'field1', got %v", chunks[0].Key)
	}
}

func TestStreamingObjectProcessor_extractFields(t *testing.T) {
	processor := NewStreamingObjectProcessor(nil, nil, domain.GranularityToken)

	// Test with properties
	schema := &jsonSchema.Definition{
		Type: jsonSchema.Object,
		Properties: map[string]jsonSchema.Definition{
			"name": {Type: jsonSchema.String},
			"age":  {Type: jsonSchema.Number},
		},
	}

	fields, err := processor.extractFields(schema)
	if err != nil {
		t.Fatalf("extractFields failed: %v", err)
	}

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}

	// Test with nil properties
	schemaNil := &jsonSchema.Definition{
		Type:       jsonSchema.Object,
		Properties: nil,
	}

	fieldsNil, err := processor.extractFields(schemaNil)
	if err != nil {
		t.Fatalf("extractFields failed: %v", err)
	}

	if len(fieldsNil) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(fieldsNil))
	}
}

func TestNewStreamingArrayProcessor(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewStreamingArrayProcessor(llmProvider, promptBuilder, domain.GranularityToken)

	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.granularity != domain.GranularityToken {
		t.Error("Expected granularity to be set")
	}
}

func TestStreamingArrayProcessor_CanProcess(t *testing.T) {
	processor := NewStreamingArrayProcessor(nil, nil, domain.GranularityToken)

	tests := []struct {
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{jsonSchema.Array, true},
		{jsonSchema.String, false},
		{jsonSchema.Object, false},
	}

	for _, test := range tests {
		result := processor.CanProcess(test.schemaType)
		if result != test.expected {
			t.Errorf("CanProcess(%v) = %v, expected %v", test.schemaType, result, test.expected)
		}
	}
}

func TestStreamingArrayProcessor_Process(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{
		generateTokenStreamFunc: func(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
			ch := make(chan *domain.TokenChunk, 1)
			go func() {
				defer close(ch)
				ch <- &domain.TokenChunk{Token: "item", IsFinal: true}
			}()
			return ch, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewStreamingArrayProcessor(llmProvider, promptBuilder, domain.GranularityField)

	schema := &jsonSchema.Definition{
		Type:  jsonSchema.Array,
		Items: &jsonSchema.Definition{Type: jsonSchema.String},
	}
	task := domain.NewFieldTask("list", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Key() != "list" {
		t.Errorf("Expected key 'list', got %v", result.Key())
	}

	value, ok := result.Value().([]interface{})
	if !ok {
		t.Errorf("Expected []interface{}, got %T", result.Value())
	}

	if len(value) != 6 {
		t.Errorf("Expected 6 items, got %d", len(value))
	}
}

func TestStreamingArrayProcessor_ProcessStreaming(t *testing.T) {
	llmProvider := &mockTokenStreamingProvider{
		generateTokenStreamFunc: func(prompt string, config *domain.GenerationConfig) (<-chan *domain.TokenChunk, error) {
			ch := make(chan *domain.TokenChunk, 1)
			go func() {
				defer close(ch)
				ch <- &domain.TokenChunk{Token: "value", IsFinal: true}
			}()
			return ch, nil
		},
	}
	promptBuilder := &mockPromptBuilder{}
	processor := NewStreamingArrayProcessor(llmProvider, promptBuilder, domain.GranularityField)

	schema := &jsonSchema.Definition{
		Type:  jsonSchema.Array,
		Items: &jsonSchema.Definition{Type: jsonSchema.String},
	}
	task := domain.NewFieldTask("array", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	stream, err := processor.ProcessStreaming(task, context)
	if err != nil {
		t.Fatalf("ProcessStreaming failed: %v", err)
	}

	var chunks []*domain.TokenStreamChunk
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) != 6 {
		t.Errorf("Expected 6 chunks, got %d", len(chunks))
	}

	// Check that keys are correct (2 for each index)
	keyCounts := make(map[string]int)
	for _, chunk := range chunks {
		keyCounts[chunk.Key]++
	}

	if keyCounts["array[0]"] != 2 || keyCounts["array[1]"] != 2 || keyCounts["array[2]"] != 2 {
		t.Errorf("Expected 2 chunks per key, got %v", keyCounts)
	}
}
