package cache

import "time"

type RemoteCache interface {
	Exists(key string) (bool, error)
	Get(key string) ([]byte, error)
	GetOrSet(key string, gen RemoteEntryGenerator) ([]byte, error)
	Set(key string, value interface{}, ttl time.Duration) error
}
