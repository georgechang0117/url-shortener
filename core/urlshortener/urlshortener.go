package urlshortener

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"github.com/georgechang0117/url-shortener/base/base62"
	"github.com/georgechang0117/url-shortener/base/cache"
	"github.com/georgechang0117/url-shortener/base/lock"
	"github.com/georgechang0117/url-shortener/core/dao"

	"code.cloudfoundry.org/clock"
	"go.uber.org/zap"
)

const (
	lockerKeyPrefix = "get_url_shortener_"
	lockRetryCount  = 3
)

var (
	errNotFound      = errors.New("db record not found")
	defaultCacheTTL  = 24 * time.Hour
	cacheRandMax     = 5
	notFoundCacheTTL = 1 * time.Minute
	lockTTL          = time.Duration(10 * time.Second)
)

type urlShortenerImpl struct {
	locker       lock.DistributedLocker
	remoteCache  cache.RemoteCache
	shortLinkDao dao.ShortLinkDao
	clock        clock.Clock
}

// NewURLShortener creates an instance of URLShortener.
func NewURLShortener(
	locker lock.DistributedLocker,
	remoteCache cache.RemoteCache,
	shortLinkDao dao.ShortLinkDao,
	clock clock.Clock,
) URLShortener {
	return &urlShortenerImpl{
		locker:       locker,
		remoteCache:  remoteCache,
		shortLinkDao: shortLinkDao,
		clock:        clock,
	}
}

func (s *urlShortenerImpl) Upload(url string, expireAt time.Time) (*dao.ShortLink, error) {
	var id uint64
	var urlID string
	for used := true; used; used = s.isUsed(urlID) {
		id = rand.Uint64()
		urlID = base62.Encode(id)
	}

	shortLink := dao.ShortLink{
		URLID:    urlID,
		URL:      url,
		ExpireAt: expireAt,
	}

	if err := s.shortLinkDao.Create(&shortLink); err != nil {
		return nil, err
	}

	return &shortLink, nil
}

func (s *urlShortenerImpl) Load(urlID string) (*dao.ShortLink, error) {
	var shortLink dao.ShortLink

	b, err := s.remoteCache.Get(urlID)
	if cache.IsErrKeyNotExist(err) {
		// do noting
	} else if err != nil {
		return nil, err
	} else {
		if err := json.Unmarshal(b, &shortLink); err != nil {
			return nil, err
		}
		zap.S().Debugf("get shortLink from cache in the beginning, url_id: %s", urlID)
		return &shortLink, nil
	}

	// use distributed lock to prevent cache stampede
	lock, err := s.locker.Lock(
		lockerKeyPrefix+urlID,
		lockTTL,
		lock.DefaultRetryDelay,
		lockRetryCount,
	)
	if err != nil {
		zap.S().Warnf("fail to lock, err: %v", err)
		return nil, err
	}
	defer lock.Unlock()

	b, err = s.remoteCache.GetOrSet(urlID, s.shortLinkRemoteEntryGen(urlID))
	if err != nil {
		return nil, err
	}
	zap.S().Debugf("get shortLink from cache or db, url_id: %s", urlID)

	if err := json.Unmarshal(b, &shortLink); err != nil {
		return nil, err
	}

	return &shortLink, nil
}

func (s *urlShortenerImpl) isUsed(urlID string) bool {
	exists, err := s.shortLinkDao.Exists(urlID)
	if err != nil {
		return false
	}

	return exists
}

func (s *urlShortenerImpl) shortLinkRemoteEntryGen(urlID string) cache.RemoteEntryGenerator {
	gen := func() ([]byte, time.Duration, error) {
		shortLink, err := s.shortLinkDao.GetByURLID(urlID)
		if dao.IsErrRecordNotFound(err) {
			zap.S().Debugf("shortLink not found in db, url_id: %s", urlID)
			// handle request with non-existent shorten URL to prevent cache penetration
			emptyShortLink := dao.ShortLink{
				URLID:    urlID,
				URL:      "",
				ExpireAt: s.clock.Now().Add(time.Duration(-1)),
			}
			b, err := json.Marshal(emptyShortLink)
			if err != nil {
				return nil, 0, err
			}

			return b, notFoundCacheTTL, nil
		} else if err != nil {
			return nil, 0, err
		}

		zap.S().Debugf("get shortLink from db, url_id: %s", urlID)

		b, err := json.Marshal(shortLink)
		if err != nil {
			return nil, 0, err
		}

		// add rand time duration to cacheTTL to prevent cache expired at same time.
		cacheTTL := defaultCacheTTL + time.Duration(rand.Intn(cacheRandMax)*int(time.Minute))

		return b, cacheTTL, nil
	}

	return gen
}
