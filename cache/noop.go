package cache

import "errors"


var ErrCacheDisabled = errors.New("cache disabled")

type NoopCache struct{}

func (n *NoopCache) Set(key string, value []byte) error {
	return ErrCacheDisabled
}

func (n *NoopCache) Get(key string) ([]byte, error) {
	return nil, ErrCacheDisabled
}

func (n *NoopCache) Delete(key string) error {
	return ErrCacheDisabled
}
