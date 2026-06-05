package store

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ErrSpaceMismatch indicates a cross-tenant access attempt.
var ErrSpaceMismatch = errors.New("space access denied: cross-tenant")

// EnforceSpaceAccess rejects reads/writes when record and request spaces differ.
func EnforceSpaceAccess(recordSpaceID, requestSpaceID string) error {
	recordSpaceID = strings.TrimSpace(recordSpaceID)
	requestSpaceID = strings.TrimSpace(requestSpaceID)
	if recordSpaceID == "" || requestSpaceID == "" {
		return nil
	}
	if recordSpaceID != requestSpaceID {
		return fmt.Errorf("%w: record=%q request=%q", ErrSpaceMismatch, recordSpaceID, requestSpaceID)
	}
	return nil
}

// SpaceWhere scopes a GORM query to a single space_id.
func SpaceWhere(db *gorm.DB, spaceID string) *gorm.DB {
	return db.Where("space_id = ?", strings.TrimSpace(spaceID))
}
