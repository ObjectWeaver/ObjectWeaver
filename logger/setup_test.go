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
// <https://objectweaver.dev/licensing/server-side-public-license>.
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
