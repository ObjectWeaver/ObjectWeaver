package LLM

import (
	"os"
	"testing"
)

// TestMain runs before all tests in this package
// It sets SKIP_LLM_INIT to prevent the orchestrator from starting during tests
func TestMain(m *testing.M) {
	// Set environment variable to skip LLM initialization during tests
	os.Setenv("SKIP_LLM_INIT", "true")

	// Run all tests
	code := m.Run()

	// Exit with the test result code
	os.Exit(code)
}
