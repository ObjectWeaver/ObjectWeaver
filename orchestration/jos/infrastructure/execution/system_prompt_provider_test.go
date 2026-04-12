package execution

import (
	"testing"

	"objectweaver/jsonSchema"
)

func TestNewDefaultSystemPromptProvider(t *testing.T) {
	provider := NewDefaultSystemPromptProvider()
	if provider == nil {
		t.Error("Expected provider to be created")
	}
}

func TestDefaultSystemPromptProvider_GetSystemPrompt_Number(t *testing.T) {
	provider := NewDefaultSystemPromptProvider()
	prompt := provider.GetSystemPrompt(jsonSchema.Number)
	if prompt == nil {
		t.Error("Expected prompt for Number")
	}
	expected := "The value being generated for this is a number. The value returned must be a number. No other values can be returned."
	if *prompt != expected {
		t.Errorf("Expected %q, got %q", expected, *prompt)
	}
}

func TestDefaultSystemPromptProvider_GetSystemPrompt_Integer(t *testing.T) {
	provider := NewDefaultSystemPromptProvider()
	prompt := provider.GetSystemPrompt(jsonSchema.Integer)
	if prompt == nil {
		t.Error("Expected prompt for Integer")
	}
	expected := "The value being generated for this is a number. The value returned must be a number. No other values can be returned."
	if *prompt != expected {
		t.Errorf("Expected %q, got %q", expected, *prompt)
	}
}

func TestDefaultSystemPromptProvider_GetSystemPrompt_Boolean(t *testing.T) {
	provider := NewDefaultSystemPromptProvider()
	prompt := provider.GetSystemPrompt(jsonSchema.Boolean)
	if prompt == nil {
		t.Error("Expected prompt for Boolean")
	}
	expected := "The value being generated for this is a boolean. The value returned must be either true or false. No other values can be returned."
	if *prompt != expected {
		t.Errorf("Expected %q, got %q", expected, *prompt)
	}
}

func TestDefaultSystemPromptProvider_GetSystemPrompt_String(t *testing.T) {
	provider := NewDefaultSystemPromptProvider()
	prompt := provider.GetSystemPrompt(jsonSchema.String)
	if prompt == nil {
		t.Error("Expected prompt for String")
	}
	expected := "The value being generated for this is a string. Return only the string value without any additional formatting or quotes."
	if *prompt != expected {
		t.Errorf("Expected %q, got %q", expected, *prompt)
	}
}

func TestDefaultSystemPromptProvider_GetSystemPrompt_Byte(t *testing.T) {
	provider := NewDefaultSystemPromptProvider()
	prompt := provider.GetSystemPrompt(jsonSchema.Byte)
	if prompt == nil {
		t.Error("Expected prompt for Byte")
	}
	expected := "The value being generated for this is a byte value. Return only the byte value."
	if *prompt != expected {
		t.Errorf("Expected %q, got %q", expected, *prompt)
	}
}

func TestDefaultSystemPromptProvider_GetSystemPrompt_Unsupported(t *testing.T) {
	provider := NewDefaultSystemPromptProvider()
	prompt := provider.GetSystemPrompt(jsonSchema.Object)
	if prompt == nil {
		t.Error("Expected non-nil for unsupported type")
	}
	expected := "The value being generated for this is of type object. Return only the value without any additional formatting."
	if *prompt != expected {
		t.Errorf("Expected %q, got %q", expected, *prompt)
	}
}

func TestNewNoSystemPromptProvider(t *testing.T) {
	provider := NewNoSystemPromptProvider()
	if provider == nil {
		t.Error("Expected provider to be created")
	}
}

func TestNoSystemPromptProvider_GetSystemPrompt(t *testing.T) {
	provider := NewNoSystemPromptProvider()
	tests := []jsonSchema.DataType{
		jsonSchema.Number,
		jsonSchema.Boolean,
		jsonSchema.String,
		jsonSchema.Object,
		jsonSchema.Array,
	}

	for _, dataType := range tests {
		prompt := provider.GetSystemPrompt(dataType)
		if prompt != nil {
			t.Errorf("Expected nil for type %v, got %q", dataType, *prompt)
		}
	}
}

func TestNewCustomSystemPromptProvider(t *testing.T) {
	provider := NewCustomSystemPromptProvider()
	if provider == nil {
		t.Error("Expected provider to be created")
	}
	if provider.prompts == nil {
		t.Error("Expected prompts map to be initialized")
	}
}

func TestCustomSystemPromptProvider_SetPrompt(t *testing.T) {
	provider := NewCustomSystemPromptProvider()
	customPrompt := "Custom prompt for testing"
	provider.SetPrompt(jsonSchema.String, customPrompt)

	if provider.prompts[jsonSchema.String] != customPrompt {
		t.Errorf("Expected prompt to be set")
	}
}

func TestCustomSystemPromptProvider_GetSystemPrompt_Set(t *testing.T) {
	provider := NewCustomSystemPromptProvider()
	customPrompt := "Custom prompt for testing"
	provider.SetPrompt(jsonSchema.String, customPrompt)

	prompt := provider.GetSystemPrompt(jsonSchema.String)
	if prompt == nil {
		t.Error("Expected prompt to be returned")
	}
	if *prompt != customPrompt {
		t.Errorf("Expected %q, got %q", customPrompt, *prompt)
	}
}

func TestCustomSystemPromptProvider_GetSystemPrompt_NotSet(t *testing.T) {
	provider := NewCustomSystemPromptProvider()

	prompt := provider.GetSystemPrompt(jsonSchema.Number)
	if prompt != nil {
		t.Error("Expected nil for unset prompt")
	}
}

func TestCustomSystemPromptProvider_GetSystemPrompt_Overwrite(t *testing.T) {
	provider := NewCustomSystemPromptProvider()
	provider.SetPrompt(jsonSchema.Boolean, "First prompt")
	provider.SetPrompt(jsonSchema.Boolean, "Second prompt")

	prompt := provider.GetSystemPrompt(jsonSchema.Boolean)
	if prompt == nil {
		t.Error("Expected prompt to be returned")
	}
	if *prompt != "Second prompt" {
		t.Errorf("Expected overwritten prompt, got %q", *prompt)
	}
}
