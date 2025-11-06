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
package llm

import (
	"fmt"
	"math/rand"
	"objectweaver/orchestration/jos/domain"
	"time"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// MockProvider is a test implementation that generates random data without calling real APIs
type MockProvider struct {
	rng *rand.Rand
}

// NewMockProvider creates a new mock provider for benchmarking
func NewMockProvider() *MockProvider {
	return &MockProvider{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate creates random data matching the schema type
func (m *MockProvider) Generate(prompt string, config *domain.GenerationConfig) (string, *domain.ProviderMetadata, error) {
	if config == nil || config.Definition == nil {
		return m.randomString(10), m.metadata(), nil
	}

	result := m.generateForType(config.Definition)

	return result, m.metadata(), nil
}

// generateForType returns a random value based on the schema type
func (m *MockProvider) generateForType(def *jsonSchema.Definition) string {
	switch def.Type {
	case jsonSchema.String:
		return m.randomString(10)
	case jsonSchema.Integer:
		return fmt.Sprintf("%d", m.rng.Intn(100))
	case jsonSchema.Number:
		val := m.rng.Float64() * 100.0
		return fmt.Sprintf("%.2f", val)
	case jsonSchema.Boolean:
		if m.rng.Intn(2) == 0 {
			return "true"
		}
		return "false"
	case jsonSchema.Array:
		return "[]"
	case jsonSchema.Object:
		return "{}"
	case jsonSchema.Map:
		return "{}"
	default:
		return m.randomString(10)
	}
}

// randomString generates a random alphanumeric string
func (m *MockProvider) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[m.rng.Intn(len(charset))]
	}
	return string(b)
}

// metadata returns fake metadata
func (m *MockProvider) metadata() *domain.ProviderMetadata {
	return &domain.ProviderMetadata{
		TokensUsed:   m.rng.Intn(100) + 10,
		Cost:         0.001,
		Model:        "mock-model",
		FinishReason: "stop",
	}
}

// SupportsStreaming returns false
func (m *MockProvider) SupportsStreaming() bool {
	return false
}

// ModelType returns the model type
func (m *MockProvider) ModelType() string {
	return "mock"
}
