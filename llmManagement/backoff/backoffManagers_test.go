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
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package backoff

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNoBackoff_ApplyBackoff(t *testing.T) {
	b := &NoBackoff{}
	// Should be no-op
	b.ApplyBackoff(1)
}

func TestNoBackoff_ActivateBackoff(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(log.Writer()) // Reset after test

	b := &NoBackoff{Verbose: true}
	start := time.Now()
	b.ActivateBackoff(1, 0)
	elapsed := time.Since(start)

	// Check sleep duration (200ms + jitter up to 250ms)
	minSleep := 200 * time.Millisecond
	if elapsed < minSleep {
		t.Errorf("Expected at least %v sleep, got %v", minSleep, elapsed)
	}

	// Check log output
	output := buf.String()
	if !strings.Contains(output, "Rate limit hit") {
		t.Errorf("Expected log message containing 'Rate limit hit', got: %s", output)
	}
}

func TestNoBackoff_ResetBackoff(t *testing.T) {
	b := &NoBackoff{}
	// Should be no-op
	b.ResetBackoff(1)
}

func TestNewGlobalExponentialBackoff(t *testing.T) {
	maxBackoff := 10 * time.Second
	b := NewGlobalExponentialBackoff(maxBackoff, true)

	if b.currentBackoff != time.Second {
		t.Errorf("Expected currentBackoff to be 1s, got %v", b.currentBackoff)
	}
	if b.maxBackoff != maxBackoff {
		t.Errorf("Expected maxBackoff to be %v, got %v", maxBackoff, b.maxBackoff)
	}
	if b.Verbose != true {
		t.Errorf("Expected Verbose to be true, got %v", b.Verbose)
	}
}

func TestGlobalExponentialBackoff_ApplyBackoff_NoActiveBackoff(t *testing.T) {
	b := NewGlobalExponentialBackoff(10*time.Second, false)
	start := time.Now()
	b.ApplyBackoff(1)
	elapsed := time.Since(start)

	// Should not sleep
	if elapsed > 10*time.Millisecond {
		t.Errorf("Expected no significant sleep, got %v", elapsed)
	}
}

func TestGlobalExponentialBackoff_ApplyBackoff_ActiveBackoff(t *testing.T) {
	b := NewGlobalExponentialBackoff(10*time.Second, false)
	// Manually set backoffUntil to future
	b.backoffUntil = time.Now().Add(500 * time.Millisecond)

	start := time.Now()
	b.ApplyBackoff(1)
	elapsed := time.Since(start)

	// Should sleep for at least 400ms (allowing some tolerance)
	if elapsed < 400*time.Millisecond {
		t.Errorf("Expected at least 400ms sleep, got %v", elapsed)
	}
}

func TestGlobalExponentialBackoff_ActivateBackoff_NoRetryAfter(t *testing.T) {
	b := NewGlobalExponentialBackoff(10*time.Second, false)
	initialBackoff := b.currentBackoff

	b.ActivateBackoff(1, 0)

	// currentBackoff should double
	expectedBackoff := initialBackoff * 2
	if b.currentBackoff != expectedBackoff {
		t.Errorf("Expected currentBackoff to be %v, got %v", expectedBackoff, b.currentBackoff)
	}

	// backoffUntil should be set
	if time.Until(b.backoffUntil) <= 0 {
		t.Errorf("backoffUntil should be in the future")
	}
}

func TestGlobalExponentialBackoff_ActivateBackoff_WithRetryAfter(t *testing.T) {
	b := NewGlobalExponentialBackoff(10*time.Second, false)
	retryAfter := 3 * time.Second

	b.ActivateBackoff(1, retryAfter)

	// currentBackoff should reset to 1s
	if b.currentBackoff != time.Second {
		t.Errorf("Expected currentBackoff to reset to 1s, got %v", b.currentBackoff)
	}

	// backoffUntil should be set to now + retryAfter + jitter
	expectedMin := time.Now().Add(retryAfter)
	if b.backoffUntil.Before(expectedMin) {
		t.Errorf("backoffUntil should be at least %v, got %v", expectedMin, b.backoffUntil)
	}
}

func TestGlobalExponentialBackoff_ActivateBackoff_MaxBackoff(t *testing.T) {
	b := NewGlobalExponentialBackoff(2*time.Second, false)
	// Set currentBackoff to max
	b.currentBackoff = 2 * time.Second

	b.ActivateBackoff(1, 0)

	// Should not exceed maxBackoff
	if b.currentBackoff > 2*time.Second {
		t.Errorf("currentBackoff should not exceed maxBackoff, got %v", b.currentBackoff)
	}
}

func TestGlobalExponentialBackoff_ResetBackoff(t *testing.T) {
	b := NewGlobalExponentialBackoff(10*time.Second, false)
	b.currentBackoff = 5 * time.Second

	b.ResetBackoff(1)

	if b.currentBackoff != time.Second {
		t.Errorf("Expected currentBackoff to reset to 1s, got %v", b.currentBackoff)
	}
}

func TestGlobalExponentialBackoff_ResetBackoff_AlreadyMin(t *testing.T) {
	b := NewGlobalExponentialBackoff(10*time.Second, false)
	b.currentBackoff = time.Second

	b.ResetBackoff(1)

	if b.currentBackoff != time.Second {
		t.Errorf("Expected currentBackoff to remain 1s, got %v", b.currentBackoff)
	}
}

func TestGlobalExponentialBackoff_Concurrency(t *testing.T) {
	b := NewGlobalExponentialBackoff(10*time.Second, false)
	var wg sync.WaitGroup

	// Simulate multiple workers activating backoff
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			b.ActivateBackoff(workerID, 0)
			b.ApplyBackoff(workerID)
		}(i)
	}

	wg.Wait()

	// After all, backoff should have increased
	if b.currentBackoff <= time.Second {
		t.Errorf("Expected currentBackoff to increase, got %v", b.currentBackoff)
	}
}
