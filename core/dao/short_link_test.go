package dao

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	testURL = "https://boards.greenhouse.io/dcard/jobs/871123?gh_src=ok8y1h1/"
	testID  = "ejLqV3Wkyd6"
)

var (
	testShortLink1 = ShortLink{
		URLID:    "shortLink1",
		URL:      testURL,
		ExpireAt: time.Date(2021, 7, 1, 0, 0, 00, 0, time.UTC),
	}
)

type shortLinkTestSuite struct {
	suite.Suite
	impl *shortLinkDao
	db   *gorm.DB
}

func TestShortLinkSuite(t *testing.T) {
	suite.Run(t, new(shortLinkTestSuite))
}

func (s *shortLinkTestSuite) SetupSuite() {
	var err error
	s.db, err = gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	s.NoError(err)

	dao, err := NewShortLinkDao(s.db)
	s.NoError(err)
	s.impl = dao.(*shortLinkDao)

	s.insertDBSeeds()
}

func (s *shortLinkTestSuite) insertDBSeeds() {

	s.NoError(s.impl.Create(&testShortLink1))
}

func (s *shortLinkTestSuite) TestCreate() {
	shortLink := ShortLink{
		URLID:    testID,
		URL:      testURL,
		ExpireAt: time.Date(2021, 7, 1, 0, 0, 00, 0, time.UTC),
	}
	s.Require().NoError(s.impl.Create(&shortLink))
}

func (s *shortLinkTestSuite) TestGetByURLID() {
	shortLink, err := s.impl.GetByURLID(testShortLink1.URLID)
	s.Require().NoError(err)
	s.Equal(testShortLink1.ID, shortLink.ID)
	s.Equal(testShortLink1.URL, shortLink.URL)
	s.Equal(testShortLink1.ExpireAt.UnixNano(), shortLink.ExpireAt.UnixNano())
}

func (s *shortLinkTestSuite) TestExists() {
	exists, err := s.impl.Exists(testShortLink1.URLID)
	s.Require().NoError(err)
	s.True(exists)
}
