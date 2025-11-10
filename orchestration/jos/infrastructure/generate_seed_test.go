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
package infrastructure

import (
	"testing"
)

func TestGenerateSeed_ReturnsNonNil(t *testing.T) {
	seed := GenerateSeed()

	if seed == nil {
		t.Fatal("GenerateSeed() returned nil, expected non-nil pointer")
	}
}

func TestGenerateSeed_ReturnsPointer(t *testing.T) {
	seed := GenerateSeed()

	// Verify we can dereference it
	seedValue := *seed

	// The value should be an integer (any value is valid)
	if seedValue < 0 {
		t.Errorf("Expected non-negative seed value, got %d", seedValue)
	}
}

func TestGenerateSeed_ValueInRange(t *testing.T) {
	seed := GenerateSeed()

	if *seed < 0 {
		t.Errorf("Seed value %d is negative, expected >= 0", *seed)
	}

	if *seed >= 1000000 {
		t.Errorf("Seed value %d is >= 1000000, expected < 1000000", *seed)
	}
}

func TestGenerateSeed_GeneratesUniqueValues(t *testing.T) {
	// Generate multiple seeds and check they're not all identical
	const numSeeds = 10
	seeds := make([]int, numSeeds)

	for i := 0; i < numSeeds; i++ {
		seed := GenerateSeed()
		if seed == nil {
			t.Fatalf("GenerateSeed() returned nil at iteration %d", i)
		}
		seeds[i] = *seed
	}

	// Check that at least some values are different
	// (statistically very unlikely all 10 would be the same)
	allSame := true
	firstSeed := seeds[0]
	for i := 1; i < numSeeds; i++ {
		if seeds[i] != firstSeed {
			allSame = false
			break
		}
	}

	if allSame {
		t.Errorf("All %d generated seeds were identical (%d), expected some variation", numSeeds, firstSeed)
	}
}

func TestGenerateSeed_Concurrent(t *testing.T) {
	// Test that concurrent calls don't panic or cause issues
	const numGoroutines = 50
	results := make(chan *int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			seed := GenerateSeed()
			results <- seed
		}()
	}

	// Collect all results
	seeds := make([]*int, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		seed := <-results
		if seed == nil {
			t.Error("GenerateSeed() returned nil in concurrent execution")
			continue
		}
		seeds = append(seeds, seed)
	}

	if len(seeds) != numGoroutines {
		t.Errorf("Expected %d seeds, got %d", numGoroutines, len(seeds))
	}

	// Verify all seeds are in valid range
	for i, seed := range seeds {
		if *seed < 0 || *seed >= 1000000 {
			t.Errorf("Seed %d at index %d is out of range: %d", i, i, *seed)
		}
	}
}

func TestGenerateSeed_MultipleCallsReturnDifferentPointers(t *testing.T) {
	seed1 := GenerateSeed()
	seed2 := GenerateSeed()

	// The pointers themselves should be different
	if seed1 == seed2 {
		t.Error("GenerateSeed() returned the same pointer twice, expected different pointers")
	}
}

func TestGenerateSeed_CanBeUsedInStruct(t *testing.T) {
	// Test that the returned seed can be used in a struct
	type Config struct {
		Seed *int
	}

	config := Config{
		Seed: GenerateSeed(),
	}

	if config.Seed == nil {
		t.Fatal("Config.Seed is nil")
	}

	if *config.Seed < 0 || *config.Seed >= 1000000 {
		t.Errorf("Config.Seed value %d is out of range", *config.Seed)
	}
}

func TestGenerateSeed_Distribution(t *testing.T) {
	// Test that seeds are reasonably distributed across the range
	// This is a statistical test that may occasionally fail due to randomness
	const numSamples = 1000
	const numBuckets = 10
	buckets := make([]int, numBuckets)
	bucketSize := 1000000 / numBuckets

	for i := 0; i < numSamples; i++ {
		seed := GenerateSeed()
		if seed == nil {
			t.Fatal("GenerateSeed() returned nil")
		}

		bucketIndex := *seed / bucketSize
		if bucketIndex >= numBuckets {
			bucketIndex = numBuckets - 1
		}
		buckets[bucketIndex]++
	}

	// Check that no bucket is completely empty (would be very unlikely)
	emptyBuckets := 0
	for i, count := range buckets {
		if count == 0 {
			emptyBuckets++
			t.Logf("Bucket %d is empty", i)
		}
	}

	// Allow up to 2 empty buckets as this is random
	if emptyBuckets > 2 {
		t.Errorf("Too many empty buckets (%d/%d), distribution may be poor", emptyBuckets, numBuckets)
	}

	// Log distribution for debugging
	t.Logf("Seed distribution across %d buckets (total samples: %d):", numBuckets, numSamples)
	for i, count := range buckets {
		t.Logf("  Bucket %d (range %d-%d): %d samples", i, i*bucketSize, (i+1)*bucketSize-1, count)
	}
}

func TestGenerateSeed_NilCheck(t *testing.T) {
	// Ensure multiple rapid calls don't return nil
	for i := 0; i < 100; i++ {
		seed := GenerateSeed()
		if seed == nil {
			t.Fatalf("GenerateSeed() returned nil at iteration %d", i)
		}
	}
}

func BenchmarkGenerateSeed(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = GenerateSeed()
	}
}

func BenchmarkGenerateSeed_Parallel(b *testing.B) {
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = GenerateSeed()
		}
	})
}
