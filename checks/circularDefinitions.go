package checks

import "github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

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
