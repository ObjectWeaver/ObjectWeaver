package jsonSchema

// SubordinateFunction represents a function under the AI's control, including its name, definition, and responses.
type SubordinateFunction struct {
	Name       string      `json:"name"`       // The name of the subordinate function.
	Definition *Definition `json:"definition"` // The schema definition of the function.
}
