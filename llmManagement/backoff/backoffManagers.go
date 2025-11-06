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
package backoff

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

// --- NoBackoff Strategy ---

type NoBackoff struct {
	Verbose bool
}

func (b *NoBackoff) ApplyBackoff(workerID int) { /* No-op */ }
func (b *NoBackoff) ActivateBackoff(workerID int, retryAfter time.Duration) {
	jitter := time.Duration(rand.Intn(250))
	pauseDuration := 200*time.Millisecond + jitter
	if b.Verbose {
		log.Printf("WARN: Rate limit hit. Worker %d pausing for %v before retry.", workerID, pauseDuration)
	}
	time.Sleep(pauseDuration)
}
func (b *NoBackoff) ResetBackoff(workerID int) { /* No-op */ }

// --- GlobalExponentialBackoff Strategy ---

type GlobalExponentialBackoff struct {
	mu             sync.Mutex
	currentBackoff time.Duration
	backoffUntil   time.Time
	maxBackoff     time.Duration
	Verbose        bool
}

func NewGlobalExponentialBackoff(maxBackoff time.Duration, verbose bool) *GlobalExponentialBackoff {
	return &GlobalExponentialBackoff{
		currentBackoff: time.Second,
		maxBackoff:     maxBackoff,
		Verbose:        verbose,
	}
}

func (b *GlobalExponentialBackoff) ApplyBackoff(workerID int) {
	b.mu.Lock()
	sleepDuration := time.Until(b.backoffUntil)
	b.mu.Unlock()

	if sleepDuration > 0 {
		if b.Verbose {
			log.Printf("Global backoff active. Worker %d pausing for %v.", workerID, sleepDuration.Round(time.Millisecond))
		}
		time.Sleep(sleepDuration)
	}
}

func (b *GlobalExponentialBackoff) ActivateBackoff(workerID int, retryAfter time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if time.Now().Before(b.backoffUntil) {
		return // A longer backoff is already active
	}

	pauseDuration := b.currentBackoff
	if retryAfter > 0 {
		pauseDuration = retryAfter
		b.currentBackoff = time.Second // Reset counter
	} else {
		b.currentBackoff *= 2
		if b.currentBackoff > b.maxBackoff {
			b.currentBackoff = b.maxBackoff
		}
	}

	jitter := time.Duration(rand.Intn(int(float64(pauseDuration) * 0.25)))
	totalPause := pauseDuration + jitter
	b.backoffUntil = time.Now().Add(totalPause)

	if b.Verbose {
		log.Printf("Rate limit hit by worker %d. Activating GLOBAL backoff for %v.", workerID, totalPause.Round(time.Millisecond))
	}
}

func (b *GlobalExponentialBackoff) ResetBackoff(workerID int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.currentBackoff > time.Second {
		b.currentBackoff = time.Second
	}
}
