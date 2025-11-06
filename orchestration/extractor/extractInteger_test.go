// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://objectweaver.dev/licensing/server-side-public-license>.
package extractor

import (
	"testing"
)

func TestIntegerExtractor_Extract(t *testing.T) {
	extractor := NewIntegerExtractor()

	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "Extract positive integer",
			input:       "42",
			expected:    42,
			expectError: false,
		},
		{
			name:        "Extract negative integer",
			input:       "-123",
			expected:    -123,
			expectError: false,
		},
		{
			name:        "Extract integer from string with text prefix",
			input:       "The answer is 42",
			expected:    42,
			expectError: false,
		},
		{
			name:        "Extract integer from string with text suffix",
			input:       "100 percent",
			expected:    100,
			expectError: false,
		},
		{
			name:        "Extract integer from string with text around it",
			input:       "Result: 256 items",
			expected:    256,
			expectError: false,
		},
		{
			name:        "Extract first integer from multiple integers",
			input:       "10 20 30",
			expected:    10,
			expectError: false,
		},
		{
			name:        "Extract integer with leading whitespace",
			input:       "   99",
			expected:    99,
			expectError: false,
		},
		{
			name:        "Extract integer with trailing whitespace",
			input:       "77   ",
			expected:    77,
			expectError: false,
		},
		{
			name:        "Extract integer with surrounding whitespace",
			input:       "   42   ",
			expected:    42,
			expectError: false,
		},
		{
			name:        "Extract zero",
			input:       "0",
			expected:    0,
			expectError: false,
		},
		{
			name:        "Extract negative zero",
			input:       "-0",
			expected:    0,
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    0,
			expectError: true,
		},
		{
			name:        "String with no integers",
			input:       "abc",
			expected:    0,
			expectError: true,
		},
		{
			name:        "String with only whitespace",
			input:       "   ",
			expected:    0,
			expectError: true,
		},
		{
			name:        "String with special characters only",
			input:       "!@#$%^&*()",
			expected:    0,
			expectError: true,
		},
		{
			name:        "Extract integer from mixed alphanumeric",
			input:       "abc123def",
			expected:    123,
			expectError: false,
		},
		{
			name:        "Extract negative integer from text",
			input:       "Temperature: -15 degrees",
			expected:    -15,
			expectError: false,
		},
		{
			name:        "Extract large positive integer",
			input:       "999999999",
			expected:    999999999,
			expectError: false,
		},
		{
			name:        "Extract large negative integer",
			input:       "-999999999",
			expected:    -999999999,
			expectError: false,
		},
		{
			name:        "Integer with decimal point (should extract before decimal)",
			input:       "3.14",
			expected:    3,
			expectError: false,
		},
		{
			name:        "Negative integer with decimal point",
			input:       "-2.5",
			expected:    -2,
			expectError: false,
		},
		{
			name:        "Integer in parentheses",
			input:       "(42)",
			expected:    42,
			expectError: false,
		},
		{
			name:        "Integer in brackets",
			input:       "[100]",
			expected:    100,
			expectError: false,
		},
		{
			name:        "Multiple negative signs (extracts first match -10)",
			input:       "--10",
			expected:    -10,
			expectError: false,
		},
		{
			name:        "Integer with comma separator (should extract before comma)",
			input:       "1,000",
			expected:    1,
			expectError: false,
		},
		{
			name:        "Newline before integer",
			input:       "\n42",
			expected:    42,
			expectError: false,
		},
		{
			name:        "Tab before integer",
			input:       "\t99",
			expected:    99,
			expectError: false,
		},
		{
			name:        "Mixed whitespace characters",
			input:       " \t\n 123 \n\t ",
			expected:    123,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.Extract(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil for input: %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v for input: %q", err, tt.input)
				}
				if result != tt.expected {
					t.Errorf("Expected %d but got %d for input: %q", tt.expected, result, tt.input)
				}
			}
		})
	}
}

func TestIntegerExtractor_ExtractOverflow(t *testing.T) {
	extractor := NewIntegerExtractor()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Integer overflow - very large positive number",
			input:       "999999999999999999999999",
			expectError: true,
		},
		{
			name:        "Integer overflow - very large negative number",
			input:       "-999999999999999999999999",
			expectError: true,
		},
		{
			name:        "Max int value",
			input:       "2147483647", // Max int32
			expectError: false,
		},
		{
			name:        "Min int value",
			input:       "-2147483648", // Min int32
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractor.Extract(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for overflow case: %q", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for valid int: %v", err)
			}
		})
	}
}

func TestNewIntegerExtractor(t *testing.T) {
	extractor := NewIntegerExtractor()

	if extractor == nil {
		t.Error("NewIntegerExtractor returned nil")
	}

	// Verify it implements the PrimitiveExtractor interface
	var _ PrimitiveExtractor[int] = extractor

	// Test that it works
	result, err := extractor.Extract("42")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected 42 but got %d", result)
	}
}

func TestIntegerExtractor_MultipleExtractions(t *testing.T) {
	extractor := NewIntegerExtractor()

	// Test that the extractor is stateless and can be reused
	inputs := []struct {
		input    string
		expected int
	}{
		{"10", 10},
		{"20", 20},
		{"30", 30},
	}

	for _, test := range inputs {
		result, err := extractor.Extract(test.input)
		if err != nil {
			t.Errorf("Unexpected error for input %q: %v", test.input, err)
		}
		if result != test.expected {
			t.Errorf("Expected %d but got %d for input %q", test.expected, result, test.input)
		}
	}
}

func TestIntegerExtractor_EdgeCases(t *testing.T) {
	extractor := NewIntegerExtractor()

	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
		description string
	}{
		{
			name:        "Plus sign before number",
			input:       "+42",
			expected:    42,
			expectError: false,
			description: "Regex matches digits after plus sign",
		},
		{
			name:        "Unicode digits",
			input:       "Number: 42",
			expected:    42,
			expectError: false,
			description: "Should extract ASCII digits",
		},
		{
			name:        "Scientific notation",
			input:       "1e10",
			expected:    1,
			expectError: false,
			description: "Should extract first integer (1)",
		},
		{
			name:        "Hexadecimal number",
			input:       "0x1A",
			expected:    0,
			expectError: false,
			description: "Should extract 0 from 0x prefix",
		},
		{
			name:        "Binary representation",
			input:       "0b101",
			expected:    0,
			expectError: false,
			description: "Should extract 0 from 0b prefix",
		},
		{
			name:        "Octal representation",
			input:       "0o17",
			expected:    0,
			expectError: false,
			description: "Should extract 0 from 0o prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.Extract(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil. %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v. %s", err, tt.description)
				}
				if result != tt.expected {
					t.Errorf("Expected %d but got %d. %s", tt.expected, result, tt.description)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkIntegerExtractor_SimpleInteger(b *testing.B) {
	extractor := NewIntegerExtractor()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract("42")
	}
}

func BenchmarkIntegerExtractor_TextWithInteger(b *testing.B) {
	extractor := NewIntegerExtractor()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract("The answer is 42 and that's final")
	}
}

func BenchmarkIntegerExtractor_NoInteger(b *testing.B) {
	extractor := NewIntegerExtractor()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract("no numbers here")
	}
}

func BenchmarkIntegerExtractor_LargeString(b *testing.B) {
	extractor := NewIntegerExtractor()
	largeString := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
		"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
		"The number is 12345 somewhere in the middle of this text. " +
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris."

	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract(largeString)
	}
}
