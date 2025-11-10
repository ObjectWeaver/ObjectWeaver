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
package modelConverter

// ModelConverter is an interface for converting a local ModelType to its provider-specific string.
type ModelConverter interface {
	// Convert takes a local ModelType and returns the corresponding provider-specific model string.
	// If the model is not recognized, it returns an empty string.
	Convert(model string) string
}

// providerModelConverter is a concrete implementation of the ModelConverter interface.
// It maps local ModelType enums to their official string representations for various LLM providers.
type providerModelConverter struct{}

// NewModelConverter creates and returns a new instance of the providerModelConverter.
func NewModelConverter() ModelConverter {
	return &providerModelConverter{}
}

// Convert implements the logic to translate a ModelType into the correct API string.
// It handles known model constants and falls back to treating unknown models as
// pass-through strings, enabling environment variable configuration and custom models.
func (c *providerModelConverter) Convert(model string) string {
	modelStr := string(model)

	return modelStr
}
