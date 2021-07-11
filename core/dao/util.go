package dao

import "gorm.io/gorm"

// IsErrRecordNotFound checks if error equals to record not found.
func IsErrRecordNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
