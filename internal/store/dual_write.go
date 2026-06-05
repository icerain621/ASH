package store

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

// DualWriteSource describes where the shadow postgres URL was resolved from.
type DualWriteSource string

const (
	DualWriteSourceNone DualWriteSource = ""
	DualWriteSourceEnv  DualWriteSource = "env"
	DualWriteSourceFile DualWriteSource = "file"
)

// ResolveDualWritePostgresURL prefers ASH_DUAL_WRITE_POSTGRES_URL, then enabled dual-write.json.
func ResolveDualWritePostgresURL(dataDir string) (url string, source DualWriteSource) {
	if u := strings.TrimSpace(os.Getenv("ASH_DUAL_WRITE_POSTGRES_URL")); u != "" {
		return u, DualWriteSourceEnv
	}
	cfg, err := LoadDualWriteConfig(dataDir)
	if err != nil || cfg == nil || !cfg.Enabled {
		return "", DualWriteSourceNone
	}
	u := strings.TrimSpace(cfg.PostgresURL)
	if u == "" {
		return "", DualWriteSourceNone
	}
	return u, DualWriteSourceFile
}

// attachDualWrite mirrors sqlite writes to postgres when env or dual-write.json is configured.
func attachDualWrite(primary *DB, dataDir string) error {
	shadowURL, _ := ResolveDualWritePostgresURL(dataDir)
	if shadowURL == "" || primary == nil || primary.Dialect() != "sqlite" {
		return nil
	}
	shadow, err := OpenPostgres(dataDir, shadowURL)
	if err != nil {
		return fmt.Errorf("dual-write shadow db: %w", err)
	}
	registerDualWriteCallbacks(primary.DB, shadow.DB)
	return nil
}

func registerDualWriteCallbacks(primary, shadow *gorm.DB) {
	primary.Callback().Create().After("gorm:after_create").Register("ash:dual_write:create", func(tx *gorm.DB) {
		mirrorWrite(tx, shadow, func(db *gorm.DB, dest any) error {
			return db.Session(&gorm.Session{SkipHooks: true}).Create(dest).Error
		})
	})
	primary.Callback().Update().After("gorm:after_update").Register("ash:dual_write:update", func(tx *gorm.DB) {
		mirrorWrite(tx, shadow, func(db *gorm.DB, dest any) error {
			return db.Session(&gorm.Session{SkipHooks: true}).Save(dest).Error
		})
	})
	primary.Callback().Delete().After("gorm:after_delete").Register("ash:dual_write:delete", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Dest == nil {
			return
		}
		_ = shadow.Session(&gorm.Session{SkipHooks: true}).Delete(tx.Statement.Dest).Error
	})
}

func mirrorWrite(tx *gorm.DB, shadow *gorm.DB, fn func(*gorm.DB, any) error) {
	if tx == nil || tx.Error != nil || tx.Statement == nil || tx.Statement.Dest == nil {
		return
	}
	_ = fn(shadow, tx.Statement.Dest)
}
