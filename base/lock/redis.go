package lock

import (
	"time"

	rlock "github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

// DefaultRetryDelay defines default retry delay time for lock.
const DefaultRetryDelay = 100 * time.Millisecond

type redisLockerImpl struct {
	client redis.Cmdable
}

type lockImpl struct {
	locker *rlock.Locker
}

// NewRedis creates an instance of DistributedLocker.
func NewRedis(cmdable redis.Cmdable) DistributedLocker {
	return &redisLockerImpl{
		client: cmdable,
	}
}

func (l *redisLockerImpl) Lock(key string, ttl, retryDelay time.Duration, retryCount int) (Lock, error) {
	opt := rlock.Options{
		LockTimeout: ttl,
		RetryCount:  retryCount,
		RetryDelay:  retryDelay,
	}

	locker := rlock.New(l.client, key, &opt)
	ok, err := locker.Lock()
	if err != nil {
		return nil, errors.Wrap(err, "fail to lock")
	}
	if !ok {
		return nil, errors.New("lock timeout")
	}

	return &lockImpl{locker: locker}, nil
}

func (l *lockImpl) Unlock() error {
	return l.locker.Unlock()
}
