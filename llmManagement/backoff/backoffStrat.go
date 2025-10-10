package backoff

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

// BackoffStrategy defines the type of backoff to use.
type BackoffStrategy int

const (
	// BackoffNone applies a simple, fixed pause.
	BackoffNone BackoffStrategy = iota
	// BackoffGlobalExponential applies an exponential backoff that pauses all workers.
	BackoffGlobalExponential
	// BackoffPerWorkerExponential applies an exponential backoff to the individual worker.
	BackoffPerWorkerExponential
)

// --- PerWorkerExponentialBackoff Strategy ---

// backoffState holds the backoff status for a single worker.
type backoffState struct {
	currentBackoff time.Duration
	backoffUntil   time.Time
}

type PerWorkerExponentialBackoff struct {
	mu          sync.Mutex
	workerState map[int]*backoffState
	maxBackoff  time.Duration
	concurrency int
	Verbose     bool
}

func NewPerWorkerExponentialBackoff(maxBackoff time.Duration, concurrency int, verbose bool) *PerWorkerExponentialBackoff {
	b := &PerWorkerExponentialBackoff{
		workerState: make(map[int]*backoffState),
		maxBackoff:  maxBackoff,
		concurrency: concurrency,
		Verbose:     verbose,
	}
	// Pre-initialize state for each worker
	for i := 0; i < concurrency; i++ {
		b.workerState[i] = &backoffState{currentBackoff: time.Second}
	}
	return b
}

func (b *PerWorkerExponentialBackoff) ApplyBackoff(workerID int) {
	b.mu.Lock()
	state, ok := b.workerState[workerID]
	if !ok { // Should not happen due to pre-initialization
		b.mu.Unlock()
		return
	}
	sleepDuration := time.Until(state.backoffUntil)
	b.mu.Unlock()

	if sleepDuration > 0 {
		if b.Verbose {
			log.Printf("Per-worker backoff active. Worker %d pausing for %v.", workerID, sleepDuration.Round(time.Millisecond))
		}
		time.Sleep(sleepDuration)
	}
}

func (b *PerWorkerExponentialBackoff) ActivateBackoff(workerID int, retryAfter time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.workerState[workerID]
	if time.Now().Before(state.backoffUntil) {
		return // Worker is already backing off.
	}

	pauseDuration := state.currentBackoff
	if retryAfter > 0 {
		pauseDuration = retryAfter
		state.currentBackoff = time.Second // Reset counter
	} else {
		state.currentBackoff *= 2
		if state.currentBackoff > b.maxBackoff {
			state.currentBackoff = b.maxBackoff
		}
	}

	jitter := time.Duration(rand.Intn(int(float64(pauseDuration) * 0.25)))
	totalPause := pauseDuration + jitter
	state.backoffUntil = time.Now().Add(totalPause)

	if b.Verbose {
		log.Printf("Rate limit hit by worker %d. Activating PER-WORKER backoff for %v.", workerID, totalPause.Round(time.Millisecond))
	}
}

func (b *PerWorkerExponentialBackoff) ResetBackoff(workerID int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	state := b.workerState[workerID]
	if state.currentBackoff > time.Second {
		state.currentBackoff = time.Second
	}
}
