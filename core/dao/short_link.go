package dao

import (
	"time"

	"gorm.io/gorm"
)

// ShortLink defines model for short link.
type ShortLink struct {
	ID        uint64 `gorm:"primary_key,AUTO_INCREMENT"`
	URLID     string `gorm:"column:url_id;type:varchar(20);not null;unique_index"`
	URL       string `gorm:"type:varchar(256);not null"`
	ExpireAt  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type shortLinkDao struct {
	db *gorm.DB
}

// NewShortLinkDao creates an instance of ShortLinkDao.
func NewShortLinkDao(db *gorm.DB) (ShortLinkDao, error) {
	dao := &shortLinkDao{
		db: db,
	}
	return dao, dao.migrate()
}

func (d *shortLinkDao) migrate() error {
	return d.db.AutoMigrate(&ShortLink{})
}

func (d *shortLinkDao) Create(shortLink *ShortLink) error {
	err := d.db.Create(shortLink).Error
	return err
}

func (d *shortLinkDao) GetByURLID(urlID string) (*ShortLink, error) {
	var shortLink ShortLink
	if err := d.db.Where("url_id = ?", urlID).First(&shortLink).Error; err != nil {
		return nil, err
	}
	return &shortLink, nil
}

func (d *shortLinkDao) Exists(urlID string) (bool, error) {
	var exists int
	if err :=
		d.db.
			Model(&ShortLink{}).
			Where("url_id = ?", urlID).
			Select("1 AS one").
			Limit(1).
			Scan(&exists).Error; err != nil {
		return false, err
	}

	return exists == 1, nil
}
