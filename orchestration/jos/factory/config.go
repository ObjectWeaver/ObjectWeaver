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
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package factory

import "objectweaver/orchestration/jos/domain"

// ProcessingMode defines the operational mode
type ProcessingMode int

const (
	ModeSync ProcessingMode = iota
	ModeAsync
	ModeParallel
	ModeDependencyAware
	ModeStreaming
	ModeStreamingComplete
	ModeStreamingProgressive
	// ModeRecursive removed - now the default behavior
)

// GeneratorConfig configures generator creation
type GeneratorConfig struct {
	Mode                ProcessingMode
	MaxConcurrency      int
	EnableCache         bool
	EnableValidation    bool
	EnableObservability bool
	LLMProvider         string
	Plugins             []domain.Plugin
	Granularity         domain.StreamGranularity
	StreamChannel       chan<- *domain.StreamChunk
}

func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		Mode:                ModeParallel,
		MaxConcurrency:      10,
		EnableCache:         false,
		EnableValidation:    true,
		EnableObservability: false,
		LLMProvider:         "openai",
		Plugins:             make([]domain.Plugin, 0),
		Granularity:         domain.GranularityField,
	}
}

func (c *GeneratorConfig) WithMode(mode ProcessingMode) *GeneratorConfig {
	newConfig := *c
	newConfig.Mode = mode
	return &newConfig
}

func (c *GeneratorConfig) WithConcurrency(max int) *GeneratorConfig {
	newConfig := *c
	newConfig.MaxConcurrency = max
	return &newConfig
}

func (c *GeneratorConfig) WithCache(enabled bool) *GeneratorConfig {
	newConfig := *c
	newConfig.EnableCache = enabled
	return &newConfig
}

func (c *GeneratorConfig) WithPlugin(plugin domain.Plugin) *GeneratorConfig {
	newConfig := *c
	newConfig.Plugins = append(newConfig.Plugins, plugin)
	return &newConfig
}

func (c *GeneratorConfig) WithGranularity(granularity domain.StreamGranularity) *GeneratorConfig {
	newConfig := *c
	newConfig.Granularity = granularity
	return &newConfig
}
