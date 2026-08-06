package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifySQLiteFileAcceptsHealthyDB(t *testing.T) {
	dir := t.TempDir()
	_ = OpenTest(t, dir)
	path := filepath.Join(dir, "ash.db")
	if err := VerifySQLiteFile(path); err != nil {
		t.Fatalf("VerifySQLiteFile: %v", err)
	}
}

func TestVerifySQLiteFileRejectsMissing(t *testing.T) {
	err := VerifySQLiteFile(filepath.Join(t.TempDir(), "missing.db"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestEnvSQLiteIntegrityVerify(t *testing.T) {
	path := strings.TrimSpace(os.Getenv("ASH_SQLITE_VERIFY_PATH"))
	if path == "" {
		t.Skip("ASH_SQLITE_VERIFY_PATH unset")
	}
	if err := VerifySQLiteFile(path); err != nil {
		t.Fatal(err)
	}
}
