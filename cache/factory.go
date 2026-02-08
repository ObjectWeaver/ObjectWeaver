package cache

import (
	"os"
)

type Cache interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

func IsActive() bool {
	return os.Getenv("CACHE_ACTIVE") == "true"
}

func GetCache() Cache {
	if !IsActive() {
		return &NoopCache{}
	}

	// Currently defaults to Redis, but can be extended to support other cache implementations based on configuration
	return NewRedisCache()
}
