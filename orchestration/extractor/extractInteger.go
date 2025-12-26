package extractor

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)


type PrimitiveExtractor[T any] interface {
	// Extract extracts a value of type T from the given completion string.
	Extract(completion string) (T, error)
}

type IntegerExtractor struct{}

func NewIntegerExtractor() PrimitiveExtractor[int] {
	return &IntegerExtractor{}
}

// Extract extracts an integer value from the given completion string.
func (e *IntegerExtractor) Extract(completion string) (int, error) {
	// Define a regular expression to match any integer
	re := regexp.MustCompile(`-?\d+`)

	// Find the first occurrence of an integer
	match := re.FindString(strings.TrimSpace(completion))
	if match == "" {
		return 0, fmt.Errorf("no valid integer value found in the completion string")
	}

	// Convert the matched string to an integer
	value, err := strconv.Atoi(match)
	if err != nil {
		return 0, fmt.Errorf("error converting string to integer: %v", err)
	}

	return value, nil
}
