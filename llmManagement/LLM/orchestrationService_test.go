package LLM

import (
	"os"
	"testing"
)

func TestGetVerbose_True(t *testing.T) {
	os.Setenv("VERBOSE", "true")
	defer os.Unsetenv("VERBOSE")

	if !getVerbose() {
		t.Error("Expected getVerbose() to return true when VERBOSE=true")
	}
}

func TestGetVerbose_False(t *testing.T) {
	os.Setenv("VERBOSE", "false")
	defer os.Unsetenv("VERBOSE")

	if getVerbose() {
		t.Error("Expected getVerbose() to return false when VERBOSE=false")
	}
}

func TestGetVerbose_NotSet(t *testing.T) {
	os.Unsetenv("VERBOSE")

	if getVerbose() {
		t.Error("Expected getVerbose() to return false when VERBOSE is not set")
	}
}

func TestGetBoolFromEnv_True(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"true", "true"},
		{"TRUE", "TRUE"},
		{"True", "True"},
		{"1", "1"},
		{"yes", "yes"},
		{"YES", "YES"},
		{"Yes", "Yes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL", tt.value)
			defer os.Unsetenv("TEST_BOOL")

			result := getBoolFromEnv("TEST_BOOL", false)
			if !result {
				t.Errorf("Expected getBoolFromEnv to return true for value '%s'", tt.value)
			}
		})
	}
}

func TestGetBoolFromEnv_False(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"false", "false"},
		{"FALSE", "FALSE"},
		{"0", "0"},
		{"no", "no"},
		{"random", "random"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				os.Unsetenv("TEST_BOOL")
			} else {
				os.Setenv("TEST_BOOL", tt.value)
				defer os.Unsetenv("TEST_BOOL")
			}

			result := getBoolFromEnv("TEST_BOOL", false)
			if result {
				t.Errorf("Expected getBoolFromEnv to return false for value '%s'", tt.value)
			}
		})
	}
}

func TestGetBoolFromEnv_DefaultValue(t *testing.T) {
	os.Unsetenv("TEST_BOOL")

	// Test with default true
	result := getBoolFromEnv("TEST_BOOL", true)
	if !result {
		t.Error("Expected getBoolFromEnv to return default value true")
	}

	// Test with default false
	result = getBoolFromEnv("TEST_BOOL", false)
	if result {
		t.Error("Expected getBoolFromEnv to return default value false")
	}
}

func TestGetInt32FromEnv_ValidValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int32
	}{
		{"zero", "0", 0},
		{"positive", "42", 42},
		{"negative", "-10", -10},
		{"max_int32", "2147483647", 2147483647},
		{"min_int32", "-2147483648", -2147483648},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TEST_INT", tt.value)
			defer os.Unsetenv("TEST_INT")

			result := getInt32FromEnv("TEST_INT", 100)
			if result != tt.expected {
				t.Errorf("Expected getInt32FromEnv to return %d for value '%s', got %d", tt.expected, tt.value, result)
			}
		})
	}
}

func TestGetInt32FromEnv_InvalidValues(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue int32
	}{
		{"not a number", "abc", 100},
		{"float", "3.14", 100},
		{"too large", "999999999999999999999", 100},
		{"empty", "", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				os.Unsetenv("TEST_INT")
			} else {
				os.Setenv("TEST_INT", tt.value)
				defer os.Unsetenv("TEST_INT")
			}

			result := getInt32FromEnv("TEST_INT", tt.defaultValue)
			if result != tt.defaultValue {
				t.Errorf("Expected getInt32FromEnv to return default value %d for invalid value '%s', got %d", tt.defaultValue, tt.value, result)
			}
		})
	}
}

func TestGetInt32FromEnv_NotSet(t *testing.T) {
	os.Unsetenv("TEST_INT")

	defaultValue := int32(999)
	result := getInt32FromEnv("TEST_INT", defaultValue)

	if result != defaultValue {
		t.Errorf("Expected getInt32FromEnv to return default value %d when not set, got %d", defaultValue, result)
	}
}

func TestGetInt32FromEnv_WithSpaces(t *testing.T) {
	os.Setenv("TEST_INT", " 42 ")
	defer os.Unsetenv("TEST_INT")

	result := getInt32FromEnv("TEST_INT", 100)
	// Leading/trailing spaces should cause parse error
	if result != 100 {
		t.Error("Expected getInt32FromEnv to return default value for value with spaces")
	}
}

func TestGetBoolFromEnv_CaseInsensitive(t *testing.T) {
	// Test that true/yes/1 work in various cases
	values := []string{"TRUE", "True", "tRuE", "YES", "Yes", "yEs"}

	for _, val := range values {
		t.Run(val, func(t *testing.T) {
			os.Setenv("TEST_BOOL", val)
			defer os.Unsetenv("TEST_BOOL")

			result := getBoolFromEnv("TEST_BOOL", false)
			if !result {
				t.Errorf("Expected true for value '%s'", val)
			}
		})
	}
}

func TestGetInt32FromEnv_Zero(t *testing.T) {
	os.Setenv("TEST_INT", "0")
	defer os.Unsetenv("TEST_INT")

	result := getInt32FromEnv("TEST_INT", 100)
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}
}

func TestGetInt32FromEnv_Negative(t *testing.T) {
	os.Setenv("TEST_INT", "-42")
	defer os.Unsetenv("TEST_INT")

	result := getInt32FromEnv("TEST_INT", 100)
	if result != -42 {
		t.Errorf("Expected -42, got %d", result)
	}
}

func TestGetBoolFromEnv_MixedCase(t *testing.T) {
	// Test case insensitivity
	os.Setenv("TEST_BOOL", "TrUe")
	defer os.Unsetenv("TEST_BOOL")

	result := getBoolFromEnv("TEST_BOOL", false)
	if !result {
		t.Error("Expected true for mixed case 'TrUe'")
	}
}
