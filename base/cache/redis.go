package cache

import (
	"time"

	"github.com/go-redis/redis"
)

type RemoteEntryGenerator func() ([]byte, time.Duration, error)

type redisCacheImpl struct {
	client redis.Cmdable
}

func NewRedis(client redis.Cmdable) RemoteCache {
	return &redisCacheImpl{
		client: client,
	}
}

func (c *redisCacheImpl) Exists(key string) (bool, error) {
	val, err := c.client.Exists(key).Result()
	if err != nil {
		return false, err
	}

	return val > 0, nil
}

func (c *redisCacheImpl) Get(key string) ([]byte, error) {
	return c.client.Get(key).Bytes()
}

func (c *redisCacheImpl) Set(key string, val interface{}, ttl time.Duration) error {
	return c.client.Set(key, val, ttl).Err()
}

func (c *redisCacheImpl) GetOrSet(key string, gen RemoteEntryGenerator) ([]byte, error) {
	v, err := c.client.Get(key).Bytes()
	if err != nil {
		v, ttl, err := gen()
		if err != nil {
			return nil, err
		}
		err = c.client.Set(key, v, ttl).Err()
		if err != nil {
			return nil, err
		}

		return v, nil
	}

	return v, nil
}
