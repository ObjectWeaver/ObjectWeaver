//go:build nolog
// +build nolog

package logger

// Printf is a no-op when built with -tags nolog
func Printf(format string, v ...interface{}) {}

// Println is a no-op when built with -tags nolog
func Println(v ...interface{}) {}
