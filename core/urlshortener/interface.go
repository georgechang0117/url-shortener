package urlshortener

import (
	"time"

	"github.com/georgechang0117/url-shortener/core/dao"
)

// URLShortener defines interface of URL shortener operations.
type URLShortener interface {
	Upload(url string, expireAt time.Time) (*dao.ShortLink, error)
	Load(urlID string) (*dao.ShortLink, error)
}
