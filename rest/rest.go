package rest

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/georgechang0117/url-shortener/base/base62"
	"github.com/georgechang0117/url-shortener/base/cache"
	"github.com/georgechang0117/url-shortener/core/urlshortener"

	"code.cloudfoundry.org/clock"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type restImpl struct {
	e            *echo.Echo
	baseURL      string
	port         int
	remoteCache  cache.RemoteCache
	urlShortener urlshortener.URLShortener
	clock        clock.Clock
}

type uploadURLParams struct {
	URL      string `json:"url" validate:"required,uri"`
	ExpireAt string `json:"expireAt" validate:"required"`
}

type uploadURLResp struct {
	ID       string `json:"id"`
	ShortURL string `json:"shortUrl"`
}

type redirectParams struct {
	URLID string `param:"url_id" validate:"required"`
}

// NewRest creates an instance of Rest.
func NewRest(
	baseURL string,
	port int,
	urlshortener urlshortener.URLShortener,
	clock clock.Clock,
) Rest {
	r := &restImpl{
		e:            newEcho(),
		baseURL:      baseURL,
		port:         port,
		urlShortener: urlshortener,
		clock:        clock,
	}

	r.e.Use(requestLogger)
	apiGroup := r.e.Group("/api")
	apiV1Group := apiGroup.Group("/v1")
	apiV1Group.POST("/urls", r.uploadURL)

	r.e.GET("/:url_id", r.redirect)

	return r
}

func (r *restImpl) Start() {
	r.e.Start(fmt.Sprintf(":%d", r.port))
}

func newEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &defaultValidator{v: validator.New()}
	return e
}

type defaultValidator struct {
	v *validator.Validate
}

func (s *defaultValidator) Validate(i interface{}) error {
	return s.v.Struct(i)
}

func bindParams(c echo.Context, params interface{}) error {
	if err := c.Bind(params); err != nil {
		return err
	}

	err := c.Validate(params)
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return err
}

func parseTime(timeStr string) (time.Time, error) {
	ts, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Now(), err
	}
	return ts, nil
}

func (r *restImpl) uploadURL(c echo.Context) error {
	var params uploadURLParams
	if err := bindParams(c, &params); err != nil {
		return err
	}

	expireAtTime, err := parseTime(params.ExpireAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "expireAt is invalid")
	}
	if expireAtTime.Before(r.clock.Now()) {
		return echo.NewHTTPError(http.StatusBadRequest, "expireAt should be greater than now")
	}

	shorLink, err := r.urlShortener.Upload(params.URL, expireAtTime)
	if err != nil {
		return err
	}

	resp := uploadURLResp{
		ID:       shorLink.URLID,
		ShortURL: fmt.Sprintf("%s/%s", r.baseURL, shorLink.URLID),
	}

	return c.JSON(http.StatusCreated, resp)
}

func (r *restImpl) redirect(c echo.Context) error {
	var params redirectParams
	if err := bindParams(c, &params); err != nil {
		return err
	}

	if len(params.URLID) != 11 {
		return c.JSON(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}
	_, err := base62.Decode(params.URLID)
	if err != nil {
		return c.JSON(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	shortLink, err := r.urlShortener.Load(params.URLID)
	if err != nil {
		zap.S().Errorf("fail to load by urlID: %s, err: %v", params.URLID, err)
		return err
	}

	if shortLink.ExpireAt.Before(r.clock.Now()) {
		return c.JSON(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	return c.Redirect(http.StatusMovedPermanently, shortLink.URL)
}

func requestLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		req := c.Request()
		res := c.Response()
		start := time.Now()
		if err = next(c); err != nil {
			c.Error(err)
		}
		stop := time.Now()

		if err != nil {
			zap.S().Errorf("error: %v", err.Error())
		}

		path := req.URL.Path
		if path == "" {
			path = "/"
		}
		contentLength := req.Header.Get(echo.HeaderContentLength)
		if contentLength == "" {
			contentLength = "0"
		}
		zap.S().Infof(
			"uri=%s method=%s status=%d latency=%d",
			req.RequestURI,
			req.Method,
			res.Status,
			stop.Sub(start).Microseconds(),
		)

		return err
	}
}
