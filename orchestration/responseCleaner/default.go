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
package responseCleaner

import (
	"regexp"
	"strings"
)

func NewDefaultResponseCleaner() IResponseCleaner {
	return &DefaultResponseCleaner{}
}

type DefaultResponseCleaner struct{}

func (d *DefaultResponseCleaner) Clean(response, key string) string {
	// Remove "key: " format (case insensitive)
	keyPattern := "(?i)" + regexp.QuoteMeta(key) + ":\\s*"
	re := regexp.MustCompile(keyPattern)
	cleaned := re.ReplaceAllString(response, "")

	// Transform "keyKey" to "Key Key" (case insensitive matching, but preserve original case)
	doubleKeyPattern := "(?i)" + regexp.QuoteMeta(key) + regexp.QuoteMeta(key)
	re2 := regexp.MustCompile(doubleKeyPattern)

	// Find matches and replace with properly formatted version
	cleaned = re2.ReplaceAllStringFunc(cleaned, func(match string) string {
		// Split the match in half and capitalize first letter of each half
		halfLen := len(match) / 2
		firstHalf := strings.Title(strings.ToLower(match[:halfLen]))
		secondHalf := strings.Title(strings.ToLower(match[halfLen:]))
		return firstHalf + " " + secondHalf
	})

	return cleaned
}
