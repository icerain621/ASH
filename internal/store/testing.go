package store

import "testing"

// OpenTest opens a SQLite database for tests and closes it during cleanup.
func OpenTest(tb testing.TB, dataDir string) *DB {
	tb.Helper()
	db, err := Open(dataDir)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		_ = db.Close()
	})
	return db
}
