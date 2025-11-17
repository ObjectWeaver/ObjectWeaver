package LLM

import (
	"os"
	"testing"
)

func TestGetEnvInt_WithValidValue(t *testing.T) {
	key := "TEST_ENV_INT_VALID"
	expectedValue := 42

	os.Setenv(key, "42")
	defer os.Unsetenv(key)

	result := getEnvInt(key, 100)
	if result != expectedValue {
		t.Errorf("Expected %d, got %d", expectedValue, result)
	}
}

func TestGetEnvInt_WithInvalidValue(t *testing.T) {
	key := "TEST_ENV_INT_INVALID"
	defaultValue := 99

	os.Setenv(key, "not_a_number")
	defer os.Unsetenv(key)

	result := getEnvInt(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default %d, got %d", defaultValue, result)
	}
}

func TestGetEnvInt_WithMissingKey(t *testing.T) {
	key := "TEST_ENV_INT_MISSING"
	defaultValue := 77

	os.Unsetenv(key)

	result := getEnvInt(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default %d, got %d", defaultValue, result)
	}
}

func TestGetEnvInt_WithEmptyString(t *testing.T) {
	key := "TEST_ENV_INT_EMPTY"
	defaultValue := 55

	os.Setenv(key, "")
	defer os.Unsetenv(key)

	result := getEnvInt(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default %d, got %d", defaultValue, result)
	}
}

func TestGetEnvInt_WithZero(t *testing.T) {
	key := "TEST_ENV_INT_ZERO"

	os.Setenv(key, "0")
	defer os.Unsetenv(key)

	result := getEnvInt(key, 100)
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}
}

func TestGetEnvInt_WithNegativeValue(t *testing.T) {
	key := "TEST_ENV_INT_NEGATIVE"
	expectedValue := -42

	os.Setenv(key, "-42")
	defer os.Unsetenv(key)

	result := getEnvInt(key, 100)
	if result != expectedValue {
		t.Errorf("Expected %d, got %d", expectedValue, result)
	}
}

func TestGetEnvInt_WithLargeValue(t *testing.T) {
	key := "TEST_ENV_INT_LARGE"
	expectedValue := 999999

	os.Setenv(key, "999999")
	defer os.Unsetenv(key)

	result := getEnvInt(key, 100)
	if result != expectedValue {
		t.Errorf("Expected %d, got %d", expectedValue, result)
	}
}

func TestGetEnvInt_WithWhitespace(t *testing.T) {
	key := "TEST_ENV_INT_WHITESPACE"
	defaultValue := 88

	os.Setenv(key, "  123  ")
	defer os.Unsetenv(key)

	// strconv.Atoi doesn't trim whitespace, so this should fail and return default
	result := getEnvInt(key, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected default %d, got %d", defaultValue, result)
	}
}

func TestGetEnvInt_MultipleCallsSameKey(t *testing.T) {
	key := "TEST_ENV_INT_MULTIPLE"

	os.Setenv(key, "123")
	defer os.Unsetenv(key)

	result1 := getEnvInt(key, 999)
	result2 := getEnvInt(key, 999)

	if result1 != result2 {
		t.Errorf("Expected consistent results, got %d and %d", result1, result2)
	}
	if result1 != 123 {
		t.Errorf("Expected 123, got %d", result1)
	}
}

func TestGetEnvInt_DifferentDefaultValues(t *testing.T) {
	key := "TEST_ENV_INT_DEFAULTS"

	os.Unsetenv(key)

	defaults := []int{0, 1, 10, 100, 1000}
	for _, def := range defaults {
		result := getEnvInt(key, def)
		if result != def {
			t.Errorf("Expected default %d, got %d", def, result)
		}
	}
}

func TestGetEnvInt_ConcurrentAccess(t *testing.T) {
	key := "TEST_ENV_INT_CONCURRENT"
	os.Setenv(key, "777")
	defer os.Unsetenv(key)

	done := make(chan int, 10)

	// Launch multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			result := getEnvInt(key, 999)
			done <- result
		}()
	}

	// Collect results
	for i := 0; i < 10; i++ {
		result := <-done
		if result != 777 {
			t.Errorf("Expected 777, got %d", result)
		}
	}
}
