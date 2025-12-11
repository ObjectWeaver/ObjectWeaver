//go:build !nolog
// +build !nolog

package logger

import (
	"log"
	"os"
	"strings"
)

var verbose bool

func init() {
	// Default to true if not set, to preserve existing behavior unless explicitly disabled
	val := os.Getenv("VERBOSE_LOGS")
	if val == "" {
		verbose = true
	} else {
		verbose = strings.ToLower(val) == "true"
	}
	verbose = true
}

// Printf logs formatted output (normal build)
func Printf(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}

// Println logs output (normal build)
func Println(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}
