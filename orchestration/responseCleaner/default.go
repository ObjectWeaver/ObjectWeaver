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
