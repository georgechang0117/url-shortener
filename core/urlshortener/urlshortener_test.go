package urlshortener

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	cachemocks "github.com/georgechang0117/url-shortener/base/cache/mocks"
	"github.com/georgechang0117/url-shortener/base/lock"
	lockmocks "github.com/georgechang0117/url-shortener/base/lock/mocks"
	"github.com/georgechang0117/url-shortener/core/dao"
	daomocks "github.com/georgechang0117/url-shortener/core/dao/mocks"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

const (
	testUploadURL = "https://boards.greenhouse.io/dcard/jobs/871123?gh_src=ok8y1h1/"
	testURLID     = "ejLqV3Wkyd6"
)

var testNow = time.Date(2021, 7, 1, 0, 0, 00, 0, time.UTC)

type urlShortenerTestSuite struct {
	suite.Suite
	impl             *urlShortenerImpl
	mockLocker       *lockmocks.DistributedLocker
	mockRemoteCache  *cachemocks.RemoteCache
	mockShortLinkDao *daomocks.ShortLinkDao
}

func (s *urlShortenerTestSuite) SetupSuite() {
	rand.Seed(time.Now().UnixNano())
	s.mockLocker = &lockmocks.DistributedLocker{}
	s.mockRemoteCache = &cachemocks.RemoteCache{}
	s.mockShortLinkDao = &daomocks.ShortLinkDao{}
	impl := NewURLShortener(
		s.mockLocker,
		s.mockRemoteCache,
		s.mockShortLinkDao,
		fakeclock.NewFakeClock(testNow),
	)
	s.impl = impl.(*urlShortenerImpl)
}

func TestURLShortenerTestSuite(t *testing.T) {
	suite.Run(t, new(urlShortenerTestSuite))
}

func (s *urlShortenerTestSuite) TestUpload() {
	expireAt := time.Date(2021, 7, 30, 0, 0, 00, 0, time.UTC)

	s.mockShortLinkDao.On("Exists", mock.Anything).Return(false, nil).Once()
	s.mockShortLinkDao.On("Create", mock.Anything).Return(nil).Once()

	shortLink, err := s.impl.Upload(testUploadURL, expireAt)
	s.NoError(err)
	s.NotNil(shortLink.ID)
	s.NotNil(shortLink.URL)
	s.Equal(expireAt.Unix(), shortLink.ExpireAt.Unix())
}

func (s *urlShortenerTestSuite) TestLoad() {
	shortLink := dao.ShortLink{
		URLID:    testURLID,
		URL:      testUploadURL,
		ExpireAt: time.Date(2021, 7, 30, 0, 0, 00, 0, time.UTC),
	}
	b, _ := json.Marshal(shortLink)
	s.mockRemoteCache.On("Get", testURLID).Return(nil, redis.Nil).Once()
	mockLock := lockmocks.Lock{}
	s.mockLocker.On("Lock", lockerKeyPrefix+testURLID, lockTTL, lock.DefaultRetryDelay, lockRetryCount).Return(&mockLock, nil).Once()
	mockLock.On("Unlock").Return(nil).Once()
	s.mockRemoteCache.On("GetOrSet", testURLID, mock.AnythingOfType("cache.RemoteEntryGenerator")).Return(b, nil).Once()
	s.mockRemoteCache.On("Set", testURLID, &shortLink, mock.AnythingOfType("int64")).Return(nil).Once()

	sl, err := s.impl.Load(testURLID)
	s.NoError(err)
	s.Equal(shortLink.URL, sl.URL)
}

func (s *urlShortenerTestSuite) TestLoadCache() {
	shortLink := dao.ShortLink{
		URLID:    testURLID,
		URL:      testUploadURL,
		ExpireAt: time.Date(2021, 7, 30, 0, 0, 00, 0, time.UTC),
	}
	b, _ := json.Marshal(shortLink)
	s.mockRemoteCache.On("Get", testURLID).Return(b, nil)

	sl, err := s.impl.Load(testURLID)
	s.NoError(err)
	s.Equal(shortLink.URL, sl.URL)
}

func (s *urlShortenerTestSuite) TestRemoteEntryGen() {

	shortLink := dao.ShortLink{
		URLID:    testURLID,
		URL:      testUploadURL,
		ExpireAt: time.Date(2021, 7, 30, 0, 0, 00, 0, time.UTC),
	}
	b, _ := json.Marshal(shortLink)

	s.mockShortLinkDao.On("GetByURLID", testURLID).Return(&shortLink, nil).Once()

	gen := s.impl.shortLinkRemoteEntryGen(testURLID)
	v, ttl, err := gen()
	s.NoError(err)
	s.GreaterOrEqual(ttl, defaultCacheTTL)
	s.Equal(b, v)
}

func (s *urlShortenerTestSuite) TestRemoteEntryGenRecordNotFound() {
	shortLink := dao.ShortLink{
		URLID:    testURLID,
		URL:      "",
		ExpireAt: s.impl.clock.Now().Add(time.Duration(-1)),
	}
	b, _ := json.Marshal(shortLink)

	s.mockShortLinkDao.On("GetByURLID", testURLID).Return(nil, gorm.ErrRecordNotFound).Once()

	gen := s.impl.shortLinkRemoteEntryGen(testURLID)
	v, ttl, err := gen()
	s.NoError(err)
	s.Equal(notFoundCacheTTL, ttl)
	s.Equal(b, v)
}
