package execution

import (
	"context"
	"errors"
	"os"
	"testing"

	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// TestNewPrimitiveProcessor tests the constructor
func TestNewPrimitiveProcessor(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}

	processor := NewPrimitiveProcessor(llmProvider, promptBuilder)

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}
	if processor.llmProvider != llmProvider {
		t.Error("Expected llmProvider to be set")
	}
	if processor.promptBuilder != promptBuilder {
		t.Error("Expected promptBuilder to be set")
	}
	if processor.systemPromptProvider == nil {
		t.Error("Expected systemPromptProvider to be initialized")
	}
	if processor.maxRetries != 3 {
		t.Errorf("Expected maxRetries to be 3, got %d", processor.maxRetries)
	}
	if processor.numberExtractor == nil {
		t.Error("Expected numberExtractor to be initialized")
	}
}

// TestNewPrimitiveProcessorWithPromptProvider tests the constructor with custom prompt provider
func TestNewPrimitiveProcessorWithPromptProvider(t *testing.T) {
	llmProvider := &mockLLMProvider{}
	promptBuilder := &mockPromptBuilder{}
	promptProvider := &mockSystemPromptProvider{}

	processor := NewPrimitiveProcessorWithPromptProvider(llmProvider, promptBuilder, promptProvider)

	if processor.systemPromptProvider != promptProvider {
		t.Error("Expected custom systemPromptProvider to be set")
	}
}

// TestPrimitiveProcessor_CanProcess tests type checking
func TestPrimitiveProcessor_CanProcess(t *testing.T) {
	processor := NewPrimitiveProcessor(nil, nil)

	tests := []struct {
		name       string
		schemaType jsonSchema.DataType
		expected   bool
	}{
		{"String", jsonSchema.String, true},
		{"Number", jsonSchema.Number, true},
		{"Integer", jsonSchema.Integer, true},
		{"Boolean", jsonSchema.Boolean, true},
		{"Object", jsonSchema.Object, false},
		{"Array", jsonSchema.Array, false},
		{"Map", jsonSchema.Map, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.CanProcess(tt.schemaType)
			if result != tt.expected {
				t.Errorf("CanProcess(%v) = %v, expected %v", tt.schemaType, result, tt.expected)
			}
		})
	}
}

// TestPrimitiveProcessor_SetEpstimicOrchestrator tests setting the orchestrator
func TestPrimitiveProcessor_SetEpstimicOrchestrator(t *testing.T) {
	processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})
	orchestrator := &mockEpstimicOrchestrator{}

	processor.SetEpstimicOrchestrator(orchestrator)

	if processor.epstimicOrchestrator != orchestrator {
		t.Error("Expected epstimicOrchestrator to be set")
	}
}

// TestPrimitiveProcessor_Process_String tests string processing
func TestPrimitiveProcessor_Process_String(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "Hello World", &domain.ProviderMetadata{Cost: 0.01, TokensUsed: 10, Model: "gpt-4"}, nil
		},
	}
	processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("message", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Value() != "Hello World" {
		t.Errorf("Expected 'Hello World', got %v", result.Value())
	}
	if result.Metadata().Cost != 0.01 {
		t.Errorf("Expected cost 0.01, got %v", result.Metadata().Cost)
	}
}

// TestPrimitiveProcessor_Process_WithSelectFields tests SelectFields functionality
func TestPrimitiveProcessor_Process_WithSelectFields(t *testing.T) {
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			// Verify that the prompt contains the selected field
			if prompt == "" {
				t.Error("Expected prompt to contain selected field context")
			}
			return "Enhanced response", &domain.ProviderMetadata{}, nil
		},
	}
	processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

	schema := &jsonSchema.Definition{
		Type:         jsonSchema.String,
		SelectFields: []string{"previousField", "anotherField"},
	}
	task := domain.NewFieldTask("summary", schema, nil)

	req := domain.NewGenerationRequest("test", schema)
	context := domain.NewExecutionContext(req)
	context.SetGeneratedValue("previousField", "Previous content here")
	context.SetGeneratedValue("anotherField", "Another value")

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if result.Value() != "Enhanced response" {
		t.Errorf("Expected 'Enhanced response', got %v", result.Value())
	}
}

// TestPrimitiveProcessor_Process_ContextCancelled tests context cancellation
func TestPrimitiveProcessor_Process_ContextCancelled(t *testing.T) {
	processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	schema := &jsonSchema.Definition{Type: jsonSchema.String}
	task := domain.NewFieldTask("test", schema, nil)
	context := domain.NewExecutionContext(domain.NewGenerationRequest("test", schema))

	_, err := processor.Process(ctx, task, context)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// TestPrimitiveProcessor_ParseValue_Vector tests vector type parsing
func TestPrimitiveProcessor_ParseValue_Vector(t *testing.T) {
	processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	t.Run("float32 slice converts to []interface{}", func(t *testing.T) {
		response := []float32{1.0, 2.0, 3.0}
		result := processor.parseValue(response, jsonSchema.Vector)

		resSlice, ok := result.([]interface{})
		if !ok {
			t.Fatalf("Expected []interface{}, got %T", result)
		}

		expected := []interface{}{float64(1.0), float64(2.0), float64(3.0)}
		if len(resSlice) != len(expected) {
			t.Errorf("Expected length %d, got %d", len(expected), len(resSlice))
		}
		for i := range resSlice {
			if resSlice[i] != expected[i] {
				t.Errorf("At index %d: expected %v, got %v", i, expected[i], resSlice[i])
			}
		}
	})

	t.Run("non-float32 slice returns as-is", func(t *testing.T) {
		response := []int{1, 2, 3}
		result := processor.parseValue(response, jsonSchema.Vector)

		resSlice, ok := result.([]int)
		if !ok {
			t.Fatalf("Expected []int, got %T", result)
		}

		if len(resSlice) != 3 {
			t.Errorf("Expected length 3, got %d", len(resSlice))
		}
	})

	t.Run("non-slice returns as-is", func(t *testing.T) {
		response := "not a slice"
		result := processor.parseValue(response, jsonSchema.Vector)

		if result != response {
			t.Errorf("Expected %v, got %v", response, result)
		}
	})
}

// TestPrimitiveProcessor_BuildVectorRequest tests vector request building
func TestPrimitiveProcessor_BuildVectorRequest(t *testing.T) {
	t.Run("WithSelectFields", func(t *testing.T) {
		processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

		schema := &jsonSchema.Definition{
			Type:         jsonSchema.Vector,
			SelectFields: []string{"field1", "field2"},
		}
		task := domain.NewFieldTask("embedding", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(req)
		context.SetGeneratedValue("field1", "First value")
		context.SetGeneratedValue("field2", "Second value")

		prompt, config, err := processor.buildVectorRequest(task, context)
		if err != nil {
			t.Fatalf("buildVectorRequest failed: %v", err)
		}

		if prompt == "" {
			t.Error("Expected non-empty prompt")
		}
		if config == nil {
			t.Fatal("Expected non-nil config")
		}
	})

	t.Run("WithoutSelectFields", func(t *testing.T) {
		processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

		schema := &jsonSchema.Definition{
			Type: jsonSchema.Vector,
		}
		task := domain.NewFieldTask("embedding", schema, nil)

		req := domain.NewGenerationRequest("test prompt", schema)
		context := domain.NewExecutionContext(req)

		prompt, config, err := processor.buildVectorRequest(task, context)
		if err != nil {
			t.Fatalf("buildVectorRequest failed: %v", err)
		}

		if config == nil {
			t.Fatal("Expected non-nil config")
		}
		// Prompt comes from FirstPrompt which may be empty if no prompts added
		_ = prompt // prompt may be empty string, that's ok
	})

	t.Run("WithSystemPrompt", func(t *testing.T) {
		processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

		systemPrompt := "Custom system prompt"
		schema := &jsonSchema.Definition{
			Type:         jsonSchema.Vector,
			SystemPrompt: &systemPrompt,
		}
		task := domain.NewFieldTask("embedding", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(req)

		_, config, err := processor.buildVectorRequest(task, context)
		if err != nil {
			t.Fatalf("buildVectorRequest failed: %v", err)
		}

		if config.SystemPrompt != systemPrompt {
			t.Errorf("Expected system prompt '%s', got '%s'", systemPrompt, config.SystemPrompt)
		}
	})
}

// TestGetDefaultModelForProvider tests model selection based on provider
func TestGetDefaultModelForProvider(t *testing.T) {
	tests := []struct {
		name          string
		llmProvider   string
		llmAPIURL     string
		geminiAPIKey  string
		llmAPIKey     string
		expectedModel string
	}{
		{
			name:          "Gemini provider",
			llmProvider:   "gemini",
			expectedModel: "gemini-2.0-flash",
		},
		{
			name:          "OpenAI provider",
			llmProvider:   "openai",
			expectedModel: "gpt-4o-mini",
		},
		{
			name:          "Local provider",
			llmProvider:   "local",
			expectedModel: "gpt-4o-mini",
		},
		{
			name:          "Gemini provider uppercase",
			llmProvider:   "GEMINI",
			expectedModel: "gemini-2.0-flash",
		},
		{
			name:          "Auto-detect with LLM_API_URL",
			llmAPIURL:     "http://localhost:8000",
			expectedModel: "gpt-4o-mini",
		},
		{
			name:          "Auto-detect with GEMINI_API_KEY",
			geminiAPIKey:  "test-key",
			expectedModel: "gemini-2.0-flash",
		},
		{
			name:          "Auto-detect with LLM_API_KEY",
			llmAPIKey:     "test-key",
			expectedModel: "gemini-2.0-flash",
		},
		{
			name:          "Default fallback",
			expectedModel: "gpt-4o-mini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			os.Unsetenv("LLM_PROVIDER")
			os.Unsetenv("LLM_API_URL")
			os.Unsetenv("GEMINI_API_KEY")
			os.Unsetenv("LLM_API_KEY")

			// Set test-specific env vars
			if tt.llmProvider != "" {
				os.Setenv("LLM_PROVIDER", tt.llmProvider)
			}
			if tt.llmAPIURL != "" {
				os.Setenv("LLM_API_URL", tt.llmAPIURL)
			}
			if tt.geminiAPIKey != "" {
				os.Setenv("GEMINI_API_KEY", tt.geminiAPIKey)
			}
			if tt.llmAPIKey != "" {
				os.Setenv("LLM_API_KEY", tt.llmAPIKey)
			}

			result := getDefaultModelForProvider()
			if result != tt.expectedModel {
				t.Errorf("Expected model %s, got %s", tt.expectedModel, result)
			}

			// Cleanup
			os.Unsetenv("LLM_PROVIDER")
			os.Unsetenv("LLM_API_URL")
			os.Unsetenv("GEMINI_API_KEY")
			os.Unsetenv("LLM_API_KEY")
		})
	}
}

// TestPrimitiveProcessor_DetermineModel tests model determination
func TestPrimitiveProcessor_DetermineModel(t *testing.T) {
	// Clear env vars first
	os.Unsetenv("LLM_PROVIDER")

	processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

	t.Run("ModelSpecifiedInDefinition", func(t *testing.T) {
		def := &jsonSchema.Definition{
			Type:  jsonSchema.String,
			Model: "gpt-4",
		}

		model := processor.determineModel(def)
		if model != "gpt-4" {
			t.Errorf("Expected 'gpt-4', got %s", model)
		}
	})

	t.Run("NoModelSpecified_UsesDefault", func(t *testing.T) {
		os.Setenv("LLM_PROVIDER", "openai")
		defer os.Unsetenv("LLM_PROVIDER")

		def := &jsonSchema.Definition{
			Type: jsonSchema.String,
		}

		model := processor.determineModel(def)
		if model != "gpt-4o-mini" {
			t.Errorf("Expected 'gpt-4o-mini', got %s", model)
		}
	})
}

// TestPrimitiveProcessor_BuildRequestPieces tests request building
func TestPrimitiveProcessor_BuildRequestPieces(t *testing.T) {
	t.Run("VectorType", func(t *testing.T) {
		processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

		schema := &jsonSchema.Definition{
			Type: jsonSchema.Vector,
		}
		task := domain.NewFieldTask("embedding", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(req)

		prompt, config, err := processor.buildRequestPieces(task, context)
		if err != nil {
			t.Fatalf("buildRequestPieces failed: %v", err)
		}

		if config == nil {
			t.Fatal("Expected non-nil config")
		}
		// For vector, we should get the vector request builder behavior
		t.Logf("Prompt: %s", prompt)
	})

	t.Run("NonVectorType_WithSelectFields", func(t *testing.T) {
		processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

		schema := &jsonSchema.Definition{
			Type:         jsonSchema.String,
			SelectFields: []string{"previousValue"},
		}
		task := domain.NewFieldTask("summary", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(req)
		context.SetGeneratedValue("previousValue", "Some previous content")

		prompt, config, err := processor.buildRequestPieces(task, context)
		if err != nil {
			t.Fatalf("buildRequestPieces failed: %v", err)
		}

		if config == nil {
			t.Fatal("Expected non-nil config")
		}
		// Prompt should contain context from previous generation
		if prompt == "" {
			t.Error("Expected non-empty prompt with selected fields")
		}
	})

	t.Run("WithDefinitionSystemPrompt", func(t *testing.T) {
		processor := NewPrimitiveProcessor(&mockLLMProvider{}, &mockPromptBuilder{})

		systemPrompt := "Custom system prompt"
		schema := &jsonSchema.Definition{
			Type:         jsonSchema.String,
			SystemPrompt: &systemPrompt,
		}
		task := domain.NewFieldTask("test", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(req)

		_, config, err := processor.buildRequestPieces(task, context)
		if err != nil {
			t.Fatalf("buildRequestPieces failed: %v", err)
		}

		if config.SystemPrompt != systemPrompt {
			t.Errorf("Expected system prompt '%s', got '%s'", systemPrompt, config.SystemPrompt)
		}
	})

	t.Run("WithProviderSystemPrompt", func(t *testing.T) {
		customProvider := &mockSystemPromptProvider{}
		processor := NewPrimitiveProcessorWithPromptProvider(&mockLLMProvider{}, &mockPromptBuilder{}, customProvider)

		schema := &jsonSchema.Definition{
			Type: jsonSchema.String,
		}
		task := domain.NewFieldTask("test", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		context := domain.NewExecutionContext(req)

		_, config, err := processor.buildRequestPieces(task, context)
		if err != nil {
			t.Fatalf("buildRequestPieces failed: %v", err)
		}

		if config.SystemPrompt != "mock system prompt" {
			t.Errorf("Expected 'mock system prompt', got '%s'", config.SystemPrompt)
		}
	})
}

// TestPrimitiveProcessor_GenerateValue tests value generation
func TestPrimitiveProcessor_GenerateValue(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		llmProvider := &mockLLMProvider{
			generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
				return "Generated value", &domain.ProviderMetadata{
					Cost:       0.05,
					TokensUsed: 50,
					Model:      "gpt-4",
				}, nil
			},
		}
		processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

		schema := &jsonSchema.Definition{Type: jsonSchema.String}
		task := domain.NewFieldTask("test", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		execContext := domain.NewExecutionContext(req)

		value, metadata, err := processor.generateValue(context.Background(), task, execContext)
		if err != nil {
			t.Fatalf("generateValue failed: %v", err)
		}

		if value != "Generated value" {
			t.Errorf("Expected 'Generated value', got %v", value)
		}
		if metadata.Cost != 0.05 {
			t.Errorf("Expected cost 0.05, got %v", metadata.Cost)
		}
		if metadata.Prompt == "" {
			t.Error("Expected prompt to be set in metadata")
		}
	})

	t.Run("LLMError", func(t *testing.T) {
		llmProvider := &mockLLMProvider{
			generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
				return nil, nil, errors.New("LLM failed")
			},
		}
		processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})

		schema := &jsonSchema.Definition{Type: jsonSchema.String}
		task := domain.NewFieldTask("test", schema, nil)

		req := domain.NewGenerationRequest("test", schema)
		execContext := domain.NewExecutionContext(req)

		_, _, err := processor.generateValue(context.Background(), task, execContext)
		if err == nil {
			t.Error("Expected error from LLM failure")
		}
	})
}

// TestCleanResponse tests response cleaning
func TestCleanResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "quoted string",
			input:    `"Hello World"`,
			expected: "Hello World",
		},
		{
			name:     "unquoted string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single quotes not removed",
			input:    "'Hello'",
			expected: "'Hello'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanResponse(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestTrimQuotes tests quote trimming
func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"test"`, "test"},
		{`""`, ""},
		{`"`, `"`},
		{"test", "test"},
		{`"test`, `"test`},
		{`test"`, `test"`},
	}

	for _, tt := range tests {
		result := trimQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("trimQuotes(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestPrimitiveProcessor_Process_WithEpstimic tests epistemic validation flow
func TestPrimitiveProcessor_Process_WithEpstimic(t *testing.T) {
	orchestrator := &mockEpstimicOrchestrator{}
	llmProvider := &mockLLMProvider{
		generateFunc: func(prompt string, config *domain.GenerationConfig) (any, *domain.ProviderMetadata, error) {
			return "validated value", &domain.ProviderMetadata{}, nil
		},
	}

	processor := NewPrimitiveProcessor(llmProvider, &mockPromptBuilder{})
	processor.SetEpstimicOrchestrator(orchestrator)

	schema := &jsonSchema.Definition{
		Type: jsonSchema.String,
		Epistemic: jsonSchema.EpistemicValidation{
			Active: true,
		},
	}
	task := domain.NewFieldTask("test", schema, nil)

	req := domain.NewGenerationRequest("test", schema)
	context := domain.NewExecutionContext(req)

	result, err := processor.Process(testContext(t), task, context)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if !orchestrator.called {
		t.Error("Expected epistemic orchestrator to be called")
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}
