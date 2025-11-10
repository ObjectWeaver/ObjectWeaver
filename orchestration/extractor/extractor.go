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
package extractor

import (
	"fmt"
	"strings"
)

// Extractor handles the responsibility of extracting and joining values.
type Extractor interface {
	ExtractAndJoin(currentGen map[string]any, keys []string) string
}

func NewDefaultExtractor() Extractor {
	return &DefaultExtractor{}
}

type DefaultExtractor struct{}

func (de *DefaultExtractor) ExtractAndJoin(currentGen map[string]any, keys []string) string {
	var result []string
	for _, key := range keys {
		if value, exists := currentGen[key]; exists {
			if strValue, ok := value.(string); ok {
				result = append(result, strValue)
			} else {
				v := fmt.Sprintf("%v", value)
				if v == "" || v == "map[]" {
					continue
				}
				result = append(result, v)
			}
		}
	}
	return strings.Join(result, "\n")
}
