package dao

// ShortLinkDao defines interface of ShortLink operations.
type ShortLinkDao interface {
	Create(shortLink *ShortLink) error
	GetByURLID(urlID string) (*ShortLink, error)
	Exists(id string) (bool, error)
}
