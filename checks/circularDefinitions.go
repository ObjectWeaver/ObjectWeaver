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
package checks

import "github.com/objectweaver/go-sdk/jsonSchema"

// CheckCircularDefinitions checks for circular definitions in the provided Definition structure.
func CheckCircularDefinitions(def *jsonSchema.Definition) bool {
	visited := make(map[*jsonSchema.Definition]bool)
	recStack := make(map[*jsonSchema.Definition]bool)

	var dfs func(*jsonSchema.Definition) bool
	dfs = func(d *jsonSchema.Definition) bool {
		if d == nil {
			return false
		}
		// If the node is in the recursion stack, we found a cycle
		if recStack[d] {
			return true
		}
		// If the node is already visited and not part of the recursion stack
		if visited[d] {
			return false
		}

		// Mark the current node as visited and part of the recursion stack
		visited[d] = true
		recStack[d] = true

		// Recurse for all properties
		for _, prop := range d.Properties {
			if dfs(&prop) {
				return true
			}
		}

		// Recurse for items if present
		if d.Items != nil {
			if dfs(d.Items) {
				return true
			}
		}

		// Remove the node from the recursion stack
		recStack[d] = false
		return false
	}

	return dfs(def)
}
