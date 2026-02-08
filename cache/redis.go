package cache

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient     *redis.Client
	redisClientOnce sync.Once
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache() Cache {
	return &RedisCache{
		client: getRedisClient(),
	}
}

func (c *RedisCache) Set(key string, value []byte) error {
	return c.client.Set(context.Background(), key, value, getCacheTTL()).Err()
}

func (c *RedisCache) Get(key string) ([]byte, error) {
	result, err := c.client.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *RedisCache) Delete(key string) error {
	return c.client.Del(context.Background(), key).Err()
}

func getRedisClient() *redis.Client {
	redisClientOnce.Do(func() {
		addr := os.Getenv("CACHE_ADDRESS")
		if addr == "" {
			addr = "localhost:6379"
		}
		password := os.Getenv("CACHE_PASSWORD")
		db := os.Getenv("CACHE_DB")
		DB, err := strconv.Atoi(db)
		if err != nil {
			DB = 0 // default to DB 0 if conversion fails
		}

		redisClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       DB,
		})
	})

	return redisClient
}

func getCacheTTL() time.Duration {
	value := os.Getenv("CACHE_TTL_SECONDS")
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func ResetRedisClientForTesting() {
	redisClient = nil
	redisClientOnce = sync.Once{}
}
