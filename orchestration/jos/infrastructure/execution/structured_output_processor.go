package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"
	"github.com/ObjectWeaver/ObjectWeaver/logger"
	"github.com/ObjectWeaver/ObjectWeaver/orchestration/jos/domain"
)

// StructuredOutputProcessor handles fields marked with StructuredOutput=true.
// Instead of decomposing into per-field LLM calls, it sends a single request
// with a JSON Schema response_format so the LLM returns structured JSON in one call.
// On context size errors, it returns an error so the caller can fall back to normal OW decomposition.
type StructuredOutputProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	systemPromptProvider SystemPromptProvider
}

func NewStructuredOutputProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *StructuredOutputProcessor {
	return &StructuredOutputProcessor{
		llmProvider:          llmProvider,
		promptBuilder:        promptBuilder,
		systemPromptProvider: NewDefaultSystemPromptProvider(),
	}
}

// Process sends the entire Definition subtree as a single structured output call.
// Returns the parsed result map, or an error if the call fails (including context size errors).
func (p *StructuredOutputProcessor) Process(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) (*domain.TaskResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	def := task.Definition()

	// Build the JSON Schema from the Definition subtree
	schema := DefinitionToJSONSchema(def)

	// Build the prompt
	prompt, err := p.promptBuilder.Build(task, execContext.PromptContext())
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Enhance prompt with SelectFields if specified
	if len(def.SelectFields) > 0 {
		prompt += "\n\nContext from previous generation:\n"
		for _, fieldPath := range def.SelectFields {
			if value, exists := ResolveFieldPath(fieldPath, execContext.GeneratedValues()); exists {
				formattedValue := FormatFieldValue(value)
				prompt += fmt.Sprintf("\n%s:\n%s\n", fieldPath, formattedValue)
			}
		}
	}

	// Build generation config
	sharedConfig := execContext.GenerationConfig()
	config := &domain.GenerationConfig{}
	if sharedConfig != nil {
		*config = *sharedConfig
	}
	config.Model = p.determineModel(def)

	defCopy := *def
	defCopy.ResponseSchema = schema
	config.Definition = &defCopy

	// Set system prompt
	if def.SystemPrompt != nil {
		config.SystemPrompt = *def.SystemPrompt
	} else if p.systemPromptProvider != nil {
		if systemPrompt := p.systemPromptProvider.GetSystemPrompt(def.Type); systemPrompt != nil {
			config.SystemPrompt = *systemPrompt
		}
	}

	logger.Printf("[StructuredOutput] Sending structured output request for field '%s' with schema", task.Key())

	result, metadata, err := p.llmProvider.Generate(prompt, config)
	if err != nil {
		if isContextSizeError(err) {
			logger.Printf("[StructuredOutput] Context size error for field '%s', signaling fallback: %v", task.Key(), err)
			return nil, &ContextSizeError{Err: err, Field: task.Key()}
		}
		return nil, fmt.Errorf("structured output generation failed: %w", err)
	}

	responseStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("structured output response is not a string: %T", result)
	}

	parsed, err := parseStructuredResponse(responseStr, def)
	if err != nil {
		return nil, fmt.Errorf("failed to parse structured response: %w", err)
	}

	resultMetadata := domain.NewResultMetadata()
	if metadata != nil {
		resultMetadata.Cost = metadata.Cost
		resultMetadata.TokensUsed = metadata.TokensUsed
		resultMetadata.PromptTokens = metadata.PromptTokens
		resultMetadata.CompletionTokens = metadata.CompletionTokens
		resultMetadata.ModelUsed = metadata.Model
	}

	taskResult := domain.NewTaskResult(task.ID(), task.Key(), parsed, resultMetadata)
	return taskResult.WithPath(task.Path()), nil
}

func (p *StructuredOutputProcessor) determineModel(def *jsonSchema.Definition) string {
	if def.Model != "" {
		return def.Model
	}
	return p.llmProvider.ModelType()
}

// ContextSizeError signals that the structured output call failed due to context size limits.
// The caller should catch this and fall back to normal OW decomposition.
type ContextSizeError struct {
	Err   error
	Field string
}

func (e *ContextSizeError) Error() string {
	return fmt.Sprintf("context size exceeded for field '%s': %v", e.Field, e.Err)
}

func (e *ContextSizeError) Unwrap() error {
	return e.Err
}

// isContextSizeError checks if an error is related to context/token limits.
func isContextSizeError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	contextPatterns := []string{
		"context length",
		"context_length",
		"token limit",
		"max_tokens",
		"maximum context",
		"too many tokens",
		"request too large",
		"content too large",
		"payload too large",
		"413",
		"input too long",
		"exceeds the model",
	}
	for _, pattern := range contextPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// DefinitionToJSONSchema converts a Definition subtree into a JSON Schema map
// suitable for use in OpenAI's response_format.
func DefinitionToJSONSchema(def *jsonSchema.Definition) map[string]any {
	if def == nil {
		return map[string]any{"type": "string"}
	}

	switch def.Type {
	case jsonSchema.Object:
		return objectToSchema(def)
	case jsonSchema.Array:
		return arrayToSchema(def)
	case jsonSchema.String:
		schema := map[string]any{"type": "string"}
		if def.Instruction != "" {
			schema["description"] = def.Instruction
		}
		return schema
	case jsonSchema.Number:
		schema := map[string]any{"type": "number"}
		if def.Instruction != "" {
			schema["description"] = def.Instruction
		}
		return schema
	case jsonSchema.Integer:
		schema := map[string]any{"type": "integer"}
		if def.Instruction != "" {
			schema["description"] = def.Instruction
		}
		return schema
	case jsonSchema.Boolean:
		schema := map[string]any{"type": "boolean"}
		if def.Instruction != "" {
			schema["description"] = def.Instruction
		}
		return schema
	default:
		schema := map[string]any{"type": "string"}
		if def.Instruction != "" {
			schema["description"] = def.Instruction
		}
		return schema
	}
}

func objectToSchema(def *jsonSchema.Definition) map[string]any {
	schema := map[string]any{
		"type": "object",
	}
	if def.Instruction != "" {
		schema["description"] = def.Instruction
	}

	if def.Properties != nil {
		properties := make(map[string]any)
		required := make([]string, 0, len(def.Properties))

		for key, propDef := range def.Properties {
			propDefCopy := propDef
			properties[key] = DefinitionToJSONSchema(&propDefCopy)
			required = append(required, key)
		}

		schema["properties"] = properties
		schema["required"] = required
		schema["additionalProperties"] = false
	}

	return schema
}

func arrayToSchema(def *jsonSchema.Definition) map[string]any {
	schema := map[string]any{
		"type": "array",
	}
	if def.Instruction != "" {
		schema["description"] = def.Instruction
	}

	if def.Items != nil {
		schema["items"] = DefinitionToJSONSchema(def.Items)
	} else {
		schema["items"] = map[string]any{"type": "string"}
	}

	return schema
}

// parseStructuredResponse parses a JSON string response into the structure
// defined by the Definition. Returns a map for Object types or a slice for Array types.
func parseStructuredResponse(response string, def *jsonSchema.Definition) (any, error) {
	// Clean the response — strip markdown code blocks if present
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	switch def.Type {
	case jsonSchema.Object:
		var result map[string]any
		if err := json.Unmarshal([]byte(response), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal object response: %w", err)
		}
		return result, nil
	case jsonSchema.Array:
		var result []any
		if err := json.Unmarshal([]byte(response), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal array response: %w", err)
		}
		return result, nil
	default:
		// For primitives, just return the cleaned string
		return response, nil
	}
}
