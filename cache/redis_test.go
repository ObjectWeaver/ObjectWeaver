package cache

import (
	"bytes"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestGetCacheTTL(t *testing.T) {
	ResetRedisClientForTesting()

	t.Setenv("CACHE_TTL_SECONDS", "")
	if ttl := getCacheTTL(); ttl != 0 {
		t.Fatalf("expected ttl 0 for empty env, got %v", ttl)
	}

	t.Setenv("CACHE_TTL_SECONDS", "-5")
	if ttl := getCacheTTL(); ttl != 0 {
		t.Fatalf("expected ttl 0 for negative env, got %v", ttl)
	}

	t.Setenv("CACHE_TTL_SECONDS", "abc")
	if ttl := getCacheTTL(); ttl != 0 {
		t.Fatalf("expected ttl 0 for invalid env, got %v", ttl)
	}

	t.Setenv("CACHE_TTL_SECONDS", "30")
	if ttl := getCacheTTL(); ttl != 30*time.Second {
		t.Fatalf("expected ttl 30s, got %v", ttl)
	}
}

func TestGetRedisClient_Defaults(t *testing.T) {
	ResetRedisClientForTesting()

	t.Setenv("CACHE_ADDRESS", "")
	t.Setenv("CACHE_PASSWORD", "")
	t.Setenv("CACHE_DB", "")

	client := getRedisClient()
	if client == nil {
		t.Fatal("expected client to be initialized")
	}

	opts := client.Options()
	if opts.Addr != "localhost:6379" {
		t.Fatalf("expected default addr localhost:6379, got %s", opts.Addr)
	}
	if opts.Password != "" {
		t.Fatalf("expected empty password by default, got %q", opts.Password)
	}
	if opts.DB != 0 {
		t.Fatalf("expected default DB 0, got %d", opts.DB)
	}
}

func TestGetRedisClient_UsesEnv(t *testing.T) {
	ResetRedisClientForTesting()

	t.Setenv("CACHE_ADDRESS", "example.com:6379")
	t.Setenv("CACHE_PASSWORD", "secret")
	t.Setenv("CACHE_DB", "4")

	client := getRedisClient()
	if client == nil {
		t.Fatal("expected client to be initialized")
	}

	opts := client.Options()
	if opts.Addr != "example.com:6379" {
		t.Fatalf("expected addr example.com:6379, got %s", opts.Addr)
	}
	if opts.Password != "secret" {
		t.Fatalf("expected password secret, got %q", opts.Password)
	}
	if opts.DB != 4 {
		t.Fatalf("expected DB 4, got %d", opts.DB)
	}
}

func TestRedisCache_SetGet(t *testing.T) {
	// Arrange
	ResetRedisClientForTesting()
	miniredisServer := miniredis.RunT(t)
	t.Setenv("CACHE_ADDRESS", miniredisServer.Addr())
	t.Setenv("CACHE_PASSWORD", "")
	t.Setenv("CACHE_DB", "0")
	cache := NewRedisCache()
	key := "test-key"
	value := []byte("test-value")

	// Act
	setErr := cache.Set(key, value)
	fetched, getErr := cache.Get(key)

	// Assert
	if setErr != nil {
		t.Fatalf("expected set to succeed, got error %v", setErr)
	}
	if getErr != nil {
		t.Fatalf("expected get to succeed, got error %v", getErr)
	}
	if !bytes.Equal(fetched, value) {
		t.Fatalf("expected fetched value %q, got %q", value, fetched)
	}
}

func TestRedisCache_Delete(t *testing.T) {
	// Arrange
	ResetRedisClientForTesting()
	miniredisServer := miniredis.RunT(t)
	t.Setenv("CACHE_ADDRESS", miniredisServer.Addr())
	t.Setenv("CACHE_PASSWORD", "")
	t.Setenv("CACHE_DB", "0")
	cache := NewRedisCache()
	key := "test-key"
	value := []byte("test-value")
	if err := cache.Set(key, value); err != nil {
		t.Fatalf("expected set to succeed, got error %v", err)
	}

	// Act
	deleteErr := cache.Delete(key)
	missing, getErr := cache.Get(key)

	// Assert
	if deleteErr != nil {
		t.Fatalf("expected delete to succeed, got error %v", deleteErr)
	}
	if getErr != nil {
		t.Fatalf("expected get after delete to succeed, got error %v", getErr)
	}
	if missing != nil {
		t.Fatalf("expected nil after delete, got %q", missing)
	}
}
