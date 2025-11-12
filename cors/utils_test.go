package cors

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		name     string
		wildcard wildcard
		input    string
		expected bool
	}{
		{
			name:     "exact prefix and suffix match",
			wildcard: wildcard{prefix: "http://", suffix: ".example.com"},
			input:    "http://test.example.com",
			expected: true,
		},
		{
			name:     "prefix and suffix match with longer subdomain",
			wildcard: wildcard{prefix: "http://", suffix: ".example.com"},
			input:    "http://api.test.example.com",
			expected: true,
		},
		{
			name:     "no match - missing prefix",
			wildcard: wildcard{prefix: "https://", suffix: ".example.com"},
			input:    "http://test.example.com",
			expected: false,
		},
		{
			name:     "no match - missing suffix",
			wildcard: wildcard{prefix: "http://", suffix: ".example.com"},
			input:    "http://test.example.org",
			expected: false,
		},
		{
			name:     "no match - too short",
			wildcard: wildcard{prefix: "http://", suffix: ".example.com"},
			input:    "http://",
			expected: false,
		},
		{
			name:     "empty wildcard part",
			wildcard: wildcard{prefix: "http://", suffix: ".com"},
			input:    "http://.com",
			expected: true,
		},
		{
			name:     "wildcard at start",
			wildcard: wildcard{prefix: "", suffix: ".example.com"},
			input:    "anything.example.com",
			expected: true,
		},
		{
			name:     "wildcard at end",
			wildcard: wildcard{prefix: "http://example.com", suffix: ""},
			input:    "http://example.com/anything",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.wildcard.match(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v for wildcard {%q, %q} matching %q, got %v",
					tt.expected, tt.wildcard.prefix, tt.wildcard.suffix, tt.input, result)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		converter converter
		expected  []string
	}{
		{
			name:      "to upper",
			input:     []string{"get", "post", "put"},
			converter: strings.ToUpper,
			expected:  []string{"GET", "POST", "PUT"},
		},
		{
			name:      "to lower",
			input:     []string{"GET", "POST", "PUT"},
			converter: strings.ToLower,
			expected:  []string{"get", "post", "put"},
		},
		{
			name:      "canonical header key",
			input:     []string{"content-type", "authorization", "x-custom-header"},
			converter: http.CanonicalHeaderKey,
			expected:  []string{"Content-Type", "Authorization", "X-Custom-Header"},
		},
		{
			name:      "empty slice",
			input:     []string{},
			converter: strings.ToUpper,
			expected:  []string{},
		},
		{
			name:      "single element",
			input:     []string{"test"},
			converter: strings.ToUpper,
			expected:  []string{"TEST"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convert(tt.input, tt.converter)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseHeaderList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single header",
			input:    "Content-Type",
			expected: []string{"Content-Type"},
		},
		{
			name:     "multiple headers comma separated",
			input:    "Content-Type, Authorization, X-Custom-Header",
			expected: []string{"Content-Type", "Authorization", "X-Custom-Header"},
		},
		{
			name:     "headers with spaces",
			input:    "Content-Type,  Authorization  ,  X-Custom-Header",
			expected: []string{"Content-Type", "Authorization", "X-Custom-Header"},
		},
		{
			name:     "lowercase to canonical",
			input:    "content-type, authorization",
			expected: []string{"Content-Type", "Authorization"},
		},
		{
			name:     "uppercase to canonical",
			input:    "CONTENT-TYPE, AUTHORIZATION",
			expected: []string{"Content-Type", "Authorization"},
		},
		{
			name:     "mixed case to canonical",
			input:    "CoNtEnT-tYpE, AuThOrIzAtIoN",
			expected: []string{"Content-Type", "Authorization"},
		},
		{
			name:     "header with numbers",
			input:    "X-Custom-Header-123",
			expected: []string{"X-Custom-Header-123"},
		},
		{
			name:     "header with underscore",
			input:    "X_Custom_Header",
			expected: []string{"X_custom_header"},
		},
		{
			name:     "header with dot",
			input:    "X.Custom.Header",
			expected: []string{"X.custom.header"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only spaces and commas",
			input:    "  ,  ,  ",
			expected: []string{},
		},
		{
			name:     "trailing comma",
			input:    "Content-Type, Authorization,",
			expected: []string{"Content-Type", "Authorization"},
		},
		{
			name:     "leading comma",
			input:    ", Content-Type, Authorization",
			expected: []string{"Content-Type", "Authorization"},
		},
		{
			name:     "multiple commas between headers",
			input:    "Content-Type,,,Authorization",
			expected: []string{"Content-Type", "Authorization"},
		},
		{
			name:     "header with hyphen followed by uppercase",
			input:    "content-TYPE",
			expected: []string{"Content-Type"},
		},
		{
			name:     "single character headers",
			input:    "A, B, C",
			expected: []string{"A", "B", "C"},
		},
		{
			name:     "header starting with number",
			input:    "1-Custom-Header",
			expected: []string{"1-Custom-Header"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeaderList(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseHeaderListEdgeCases(t *testing.T) {
	// Test very long header name
	longHeader := strings.Repeat("X-Very-Long-Header-Name-", 10) + "End"
	result := parseHeaderList(longHeader)
	if len(result) != 1 {
		t.Errorf("Expected 1 header, got %d", len(result))
	}

	// Test multiple headers with varying lengths
	input := "A, Short, X-Medium-Length, X-Very-Long-Header-Name-With-Many-Parts"
	result = parseHeaderList(input)
	if len(result) != 4 {
		t.Errorf("Expected 4 headers, got %d", len(result))
	}
}

func TestParseHeaderListCanonicalFormat(t *testing.T) {
	// Ensure the canonical format matches HTTP header conventions
	tests := map[string]string{
		"content-type":          "Content-Type",
		"x-requested-with":      "X-Requested-With",
		"accept-encoding":       "Accept-Encoding",
		"cache-control":         "Cache-Control",
		"if-modified-since":     "If-Modified-Since",
		"x-forwarded-for":       "X-Forwarded-For",
		"x-custom-header-value": "X-Custom-Header-Value",
	}

	for input, expected := range tests {
		result := parseHeaderList(input)
		if len(result) != 1 {
			t.Errorf("Expected 1 header for %q, got %d", input, len(result))
			continue
		}
		if result[0] != expected {
			t.Errorf("For input %q, expected %q, got %q", input, expected, result[0])
		}
	}
}
