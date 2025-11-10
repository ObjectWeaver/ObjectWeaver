package logger

import (
	"bytes"
	"log"
	"testing"
)

func TestOut_Println(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(originalOutput)

	// Test when verbose is true
	o := &Out{verbose: true}
	o.Println("test message")
	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Errorf("expected log output when verbose is true, got %s", buf.String())
	}

	// Reset buffer
	buf.Reset()

	// Test when verbose is false
	o.verbose = false
	o.Println("test message")
	if bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Errorf("expected no log output when verbose is false, got %s", buf.String())
	}
}
