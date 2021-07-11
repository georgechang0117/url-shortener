package lock

import "time"

// DistributedLocker defines an interface for distributed lock.
type DistributedLocker interface {
	Lock(key string, ttl, retryDelay time.Duration, retryCount int) (Lock, error)
}

// Lock defines an interface fo lock.
type Lock interface {
	Unlock() error
}
