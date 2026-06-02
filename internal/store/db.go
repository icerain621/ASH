package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
	dataDir string
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "ash.db")
	gdb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil fmt.Errorf("open sqlite: %w", err)
	}

	db := &DB{DB: gdb, dataDir: dataDir}
	if err := db.migrate(); err != nil {
		return nil err
	}
	return db, nil
}

func (db *DB) migrate() error {
	if err := db.AutoMigrate(
		&RunRecord{},
		&RunEvent{},
		&MemoryRecord{},
		&MemoryEvidence{},
		&MemoryReview{},
		&AuditLog{},
		&SchemaMeta{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	var meta SchemaMeta
	res := db.First(&meta, "`key` = ?", "schema_version")
	if res.Error == gorm.ErrRecordNotFound {
		return db.Create(&SchemaMeta{
			Key:       "schema_version",
			Value:     "1",
			UpdatedAt: time.Now().UTC(),
		}).Error
	}
	return res.Error
}

func (db *DB) DataDir() string { return db.dataDir }

func (db *DB) RunDir(runID string) string {
	return filepath.Join(db.dataDir, "runs", runID)
}
