package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cachemocks "github.com/georgechang0117/url-shortener/base/cache/mocks"
	"github.com/georgechang0117/url-shortener/core/dao"
	urlshortenermocks "github.com/georgechang0117/url-shortener/core/urlshortener/mocks"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

const (
	testBaseURL = "http://localhost:8080"
	testPort    = 8080
	testURL     = "https://boards.greenhouse.io/dcard/jobs/871123?gh_src=ok8y1h1/"
	testURLID   = "abcdefghijk"
)

var testNow = time.Date(2021, 7, 1, 0, 0, 00, 0, time.UTC)

type restTestSuite struct {
	suite.Suite
	impl             *restImpl
	echo             *echo.Echo
	mockRemoteCache  *cachemocks.RemoteCache
	mockURLShortener *urlshortenermocks.URLShortener
}

func (s *restTestSuite) SetupTest() {
	s.echo = newEcho()
	s.mockURLShortener = &urlshortenermocks.URLShortener{}
	s.mockRemoteCache = &cachemocks.RemoteCache{}
	impl := NewRest(testBaseURL, testPort, s.mockURLShortener, fakeclock.NewFakeClock(testNow))
	s.impl = impl.(*restImpl)
}

func TestRestTestSuite(t *testing.T) {
	suite.Run(t, new(restTestSuite))
}

func (s *restTestSuite) TestUploadURL() {
	params := uploadURLParams{
		URL:      testURL,
		ExpireAt: "2021-07-30T00:00:00Z",
	}
	b, _ := json.Marshal(&params)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	expireAtTime, _ := parseTime(params.ExpireAt)
	mockShortLink := dao.ShortLink{
		URLID:    testURLID,
		URL:      testURL,
		ExpireAt: expireAtTime,
	}

	s.mockURLShortener.On("Upload", params.URL, expireAtTime).Return(&mockShortLink, nil).Once()

	s.Require().NoError(s.impl.uploadURL(c))
	s.Equal(http.StatusCreated, rec.Code)
	var resp uploadURLResp
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
	s.Equal(mockShortLink.URLID, resp.ID)
	s.Equal(fmt.Sprintf("%s/%s", s.impl.baseURL, testURLID), resp.ShortURL)
}

func (s *restTestSuite) TestUploadURLExpireAtNotTimeFormat() {
	params := uploadURLParams{
		URL:      testURL,
		ExpireAt: "invalid",
	}
	b, _ := json.Marshal(&params)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err := s.impl.uploadURL(c)
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, err.(*echo.HTTPError).Code)
}

func (s *restTestSuite) TestUploadURLExpireAtTooOld() {
	params := uploadURLParams{
		URL:      testURL,
		ExpireAt: s.impl.clock.Now().Add(-1).Format(time.RFC3339),
	}
	b, _ := json.Marshal(&params)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)

	err := s.impl.uploadURL(c)
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, err.(*echo.HTTPError).Code)
}

func (s *restTestSuite) TestRedirect() {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/:url_id")
	c.SetParamNames("url_id")
	c.SetParamValues(testURLID)

	shortLink := dao.ShortLink{
		URLID:    "testID",
		URL:      testURL,
		ExpireAt: s.impl.clock.Now().Add(10),
	}

	s.mockURLShortener.On("Load", testURLID).Return(&shortLink, nil).Once()

	s.Require().NoError(s.impl.redirect(c))
	s.Equal(http.StatusMovedPermanently, rec.Code)
	s.Equal(testURL, rec.HeaderMap.Get("Location"))
}

func (s *restTestSuite) TestRedirectExpired() {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := s.echo.NewContext(req, rec)
	c.SetPath("/:url_id")
	c.SetParamNames("url_id")
	c.SetParamValues(testURLID)

	shortLink := dao.ShortLink{
		URLID:    "testID",
		URL:      testURL,
		ExpireAt: s.impl.clock.Now().Add(-1),
	}

	s.mockURLShortener.On("Load", testURLID).Return(&shortLink, nil).Once()

	s.Require().NoError(s.impl.redirect(c))
	s.Equal(http.StatusNotFound, rec.Code)
}
