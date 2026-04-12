package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ObjectWeaver/ObjectWeaver/cache"

	"github.com/alicebob/miniredis/v2"
)

func TestGetObjectQueued_UsesCache(t *testing.T) {
	miniredisServer := miniredis.RunT(t)

	t.Setenv("CACHE_ACTIVE", "true")
	t.Setenv("CACHE_ADDRESS", miniredisServer.Addr())
	t.Setenv("CACHE_DB", "0")
	cache.ResetRedisClientForTesting()

	c := cache.GetCache()
	id := "queued-test-id"
	response := &Response{
		Data:    map[string]any{"hello": "world"},
		UsdCost: 0.01,
	}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	if err := c.Set(id, payload); err != nil {
		t.Fatalf("failed to set cache value: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/getObjectQueued?id="+id, nil)
	w := httptest.NewRecorder()

	server := &Server{}
	server.GetObjectQueued(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected status 200, got %d: %s", res.StatusCode, string(body))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(body) != string(payload) {
		t.Fatalf("unexpected response body: %s", string(body))
	}

	result, err := c.Get(id)
	if err != nil {
		t.Fatalf("failed to get cache value after delete: %v", err)
	}
	if result != nil {
		t.Fatalf("expected cached value to be deleted")
	}
}
