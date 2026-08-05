package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// OpenTest opens a SQLite database for tests and closes it during cleanup.
func OpenTest(tb testing.TB, dataDir string) *DB {
	tb.Helper()
	tb.Setenv("ASH_SCHEMA_MODE", "dual")
	db, err := OpenWithDatabaseURL(dataDir, "")
	if err != nil {
		tb.Fatal(err)
	}
	if sqlDB, err := db.DB.DB(); err == nil {
		// Single connection avoids Windows tempfile unlock races with modernc sqlite.
		sqlDB.SetMaxOpenConns(1)
	}
	tb.Cleanup(func() {
		if sqlDB, err := db.DB.DB(); err == nil {
			_, _ = sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
			sqlDB.SetMaxOpenConns(0)
		}
		_ = db.Close()
		// Windows may keep the file locked briefly after Close; remove before TempDir cleanup.
		base := filepath.Join(dataDir, "ash.db")
		for _, p := range []string{base, base + "-wal", base + "-shm", base + "-journal"} {
			for i := 0; i < 25; i++ {
				err := os.Remove(p)
				if err == nil || os.IsNotExist(err) {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
		}
	})
	return db
}
