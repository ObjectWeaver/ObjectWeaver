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
package responseCleaner

import (
	"testing"
)

func TestNewDefaultResponseCleaner(t *testing.T) {
	cleaner := NewDefaultResponseCleaner()
	if cleaner == nil {
		t.Fatal("NewDefaultResponseCleaner returned nil")
	}
	_, ok := cleaner.(*DefaultResponseCleaner)
	if !ok {
		t.Fatal("NewDefaultResponseCleaner did not return a *DefaultResponseCleaner")
	}
}

func TestDefaultResponseCleaner_Clean(t *testing.T) {
	cleaner := &DefaultResponseCleaner{}

	tests := []struct {
		name     string
		response string
		key      string
		expected string
	}{
		{
			name:     "remove key prefix",
			response: "user: John Doe",
			key:      "user",
			expected: "John Doe",
		},
		{
			name:     "case insensitive key removal",
			response: "USER: admin",
			key:      "user",
			expected: "admin",
		},
		{
			name:     "transform double key",
			response: "useruser",
			key:      "user",
			expected: "User User",
		},
		{
			name:     "case insensitive double key",
			response: "USERuser",
			key:      "user",
			expected: "User User",
		},
		{
			name:     "both operations",
			response: "user: useruser",
			key:      "user",
			expected: "User User",
		},
		{
			name:     "no match",
			response: "hello world",
			key:      "key",
			expected: "hello world",
		},
		{
			name:     "empty response",
			response: "",
			key:      "user",
			expected: "",
		},
		{
			name:     "key with special chars",
			response: "key*: value",
			key:      "key*",
			expected: "value",
		},
		{
			name:     "double key with special chars",
			response: "key*key*",
			key:      "key*",
			expected: "Key* Key*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleaner.Clean(tt.response, tt.key)
			if result != tt.expected {
				t.Errorf("Clean(%q, %q) = %q; want %q", tt.response, tt.key, result, tt.expected)
			}
		})
	}
}
