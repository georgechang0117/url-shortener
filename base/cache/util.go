package cache

import "github.com/go-redis/redis"

// IsErrKeyNotExist checks if key does not exist.
func IsErrKeyNotExist(err error) bool {
	return err == redis.Nil
}
