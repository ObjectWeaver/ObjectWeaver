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
package application

import (
	"objectweaver/orchestration/jos/domain"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

// PluginRegistry manages plugins for generators
type PluginRegistry struct {
	preProcessors  []domain.PreProcessorPlugin
	postProcessors []domain.PostProcessorPlugin
	validators     []domain.ValidationPlugin
	cache          domain.CachePlugin
	observability  domain.ObservabilityPlugin
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		preProcessors:  make([]domain.PreProcessorPlugin, 0),
		postProcessors: make([]domain.PostProcessorPlugin, 0),
		validators:     make([]domain.ValidationPlugin, 0),
	}
}

func (r *PluginRegistry) Register(plugin domain.Plugin) {
	switch p := plugin.(type) {
	case domain.PreProcessorPlugin:
		r.preProcessors = append(r.preProcessors, p)
	case domain.PostProcessorPlugin:
		r.postProcessors = append(r.postProcessors, p)
	case domain.ValidationPlugin:
		r.validators = append(r.validators, p)
	case domain.CachePlugin:
		r.cache = p
	case domain.ObservabilityPlugin:
		r.observability = p
	}
}

func (r *PluginRegistry) ApplyPreProcessors(req *domain.GenerationRequest) (*domain.GenerationRequest, error) {
	result := req
	for _, processor := range r.preProcessors {
		var err error
		result, err = processor.PreProcess(result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *PluginRegistry) ApplyPostProcessors(res *domain.GenerationResult) (*domain.GenerationResult, error) {
	result := res
	for _, processor := range r.postProcessors {
		var err error
		result, err = processor.PostProcess(result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *PluginRegistry) ApplyValidation(res *domain.GenerationResult, schema *jsonSchema.Definition) error {
	for _, validator := range r.validators {
		errors, err := validator.Validate(res, schema)
		if err != nil {
			return err
		}
		if len(errors) > 0 {
			// Could collect all validation errors and return them
			return err
		}
	}
	return nil
}

func (r *PluginRegistry) GetFromCache(key string) (*domain.GenerationResult, bool) {
	if r.cache != nil {
		return r.cache.Get(key)
	}
	return nil, false
}

func (r *PluginRegistry) CacheResult(key string, result *domain.GenerationResult) {
	if r.cache != nil {
		r.cache.Set(key, result)
	}
}
