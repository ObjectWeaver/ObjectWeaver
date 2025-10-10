package backoff

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewPerWorkerExponentialBackoff(t *testing.T) {
	maxBackoff := 10 * time.Second
	concurrency := 5
	b := NewPerWorkerExponentialBackoff(maxBackoff, concurrency, true)

	if b.maxBackoff != maxBackoff {
		t.Errorf("Expected maxBackoff %v, got %v", maxBackoff, b.maxBackoff)
	}
	if b.concurrency != concurrency {
		t.Errorf("Expected concurrency %d, got %d", concurrency, b.concurrency)
	}
	if b.Verbose != true {
		t.Errorf("Expected Verbose true, got %v", b.Verbose)
	}
	if len(b.workerState) != concurrency {
		t.Errorf("Expected %d worker states, got %d", concurrency, len(b.workerState))
	}
	for i := 0; i < concurrency; i++ {
		if b.workerState[i] == nil {
			t.Errorf("Worker %d state not initialized", i)
		}
		if b.workerState[i].currentBackoff != time.Second {
			t.Errorf("Worker %d initial backoff %v, expected 1s", i, b.workerState[i].currentBackoff)
		}
	}
}

func TestPerWorkerExponentialBackoff_ApplyBackoff_NoActive(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	start := time.Now()
	b.ApplyBackoff(0)
	elapsed := time.Since(start)

	if elapsed > 10*time.Millisecond {
		t.Errorf("Expected no sleep, got %v", elapsed)
	}
}

func TestPerWorkerExponentialBackoff_ApplyBackoff_Active(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	// Manually set backoffUntil
	b.mu.Lock()
	b.workerState[0].backoffUntil = time.Now().Add(500 * time.Millisecond)
	b.mu.Unlock()

	start := time.Now()
	b.ApplyBackoff(0)
	elapsed := time.Since(start)

	if elapsed < 400*time.Millisecond {
		t.Errorf("Expected at least 400ms sleep, got %v", elapsed)
	}
}

func TestPerWorkerExponentialBackoff_ApplyBackoff_InvalidWorker(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	// Worker 2 does not exist
	start := time.Now()
	b.ApplyBackoff(2)
	elapsed := time.Since(start)

	if elapsed > 10*time.Millisecond {
		t.Errorf("Expected no sleep for invalid worker, got %v", elapsed)
	}
}

func TestPerWorkerExponentialBackoff_ActivateBackoff_NoRetryAfter(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	workerID := 0
	initialBackoff := b.workerState[workerID].currentBackoff

	b.ActivateBackoff(workerID, 0)

	expectedBackoff := initialBackoff * 2
	if b.workerState[workerID].currentBackoff != expectedBackoff {
		t.Errorf("Expected currentBackoff %v, got %v", expectedBackoff, b.workerState[workerID].currentBackoff)
	}
	if time.Until(b.workerState[workerID].backoffUntil) <= 0 {
		t.Errorf("backoffUntil not set")
	}
}

func TestPerWorkerExponentialBackoff_ActivateBackoff_WithRetryAfter(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	workerID := 0
	retryAfter := 3 * time.Second

	b.ActivateBackoff(workerID, retryAfter)

	if b.workerState[workerID].currentBackoff != time.Second {
		t.Errorf("Expected currentBackoff reset to 1s, got %v", b.workerState[workerID].currentBackoff)
	}
	expectedMin := time.Now().Add(retryAfter)
	if b.workerState[workerID].backoffUntil.Before(expectedMin) {
		t.Errorf("backoffUntil should be at least %v, got %v", expectedMin, b.workerState[workerID].backoffUntil)
	}
}

func TestPerWorkerExponentialBackoff_ActivateBackoff_MaxBackoff(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(2*time.Second, 2, false)
	workerID := 0
	b.workerState[workerID].currentBackoff = 2 * time.Second

	b.ActivateBackoff(workerID, 0)

	if b.workerState[workerID].currentBackoff > 2*time.Second {
		t.Errorf("currentBackoff should not exceed max, got %v", b.workerState[workerID].currentBackoff)
	}
}

func TestPerWorkerExponentialBackoff_ActivateBackoff_AlreadyBackingOff(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	workerID := 0
	// Set future backoffUntil
	b.workerState[workerID].backoffUntil = time.Now().Add(time.Hour)
	initialBackoff := b.workerState[workerID].currentBackoff

	b.ActivateBackoff(workerID, 0)

	// Should not change
	if b.workerState[workerID].currentBackoff != initialBackoff {
		t.Errorf("Backoff should not change if already active")
	}
}

func TestPerWorkerExponentialBackoff_ResetBackoff(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	workerID := 0
	b.workerState[workerID].currentBackoff = 5 * time.Second

	b.ResetBackoff(workerID)

	if b.workerState[workerID].currentBackoff != time.Second {
		t.Errorf("Expected reset to 1s, got %v", b.workerState[workerID].currentBackoff)
	}
}

func TestPerWorkerExponentialBackoff_ResetBackoff_AlreadyMin(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, false)
	workerID := 0
	b.workerState[workerID].currentBackoff = time.Second

	b.ResetBackoff(workerID)

	if b.workerState[workerID].currentBackoff != time.Second {
		t.Errorf("Should remain 1s, got %v", b.workerState[workerID].currentBackoff)
	}
}

func TestPerWorkerExponentialBackoff_Concurrency(t *testing.T) {
	b := NewPerWorkerExponentialBackoff(10*time.Second, 5, false)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				b.ActivateBackoff(workerID, 0)
				b.ApplyBackoff(workerID)
			}
			// Check after activations
			b.mu.Lock()
			if b.workerState[workerID].currentBackoff <= time.Second {
				t.Errorf("Worker %d backoff should have increased", workerID)
			}
			b.mu.Unlock()
			b.ResetBackoff(workerID)
		}(i)
	}

	wg.Wait()
}

func TestPerWorkerExponentialBackoff_ActivateBackoff_Verbose(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(log.Writer())

	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, true)
	b.ActivateBackoff(0, 0)

	output := buf.String()
	if !strings.Contains(output, "Rate limit hit by worker 0") {
		t.Errorf("Expected verbose log, got: %s", output)
	}
}

func TestPerWorkerExponentialBackoff_ApplyBackoff_Verbose(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(log.Writer())

	b := NewPerWorkerExponentialBackoff(10*time.Second, 2, true)
	b.workerState[0].backoffUntil = time.Now().Add(100 * time.Millisecond)
	b.ApplyBackoff(0)

	output := buf.String()
	if !strings.Contains(output, "Per-worker backoff active") {
		t.Errorf("Expected verbose log, got: %s", output)
	}
}
