//go:build integration

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ResetPostgresPublicSchema drops and recreates public schema for isolated e2e runs.
func integrationTempDir(prefix string) (string, error) {
	for _, base := range []string{os.Getenv("TMPDIR"), os.Getenv("TEMP"), os.Getenv("TMP")} {
		if strings.TrimSpace(base) != "" {
			return os.MkdirTemp(base, prefix)
		}
	}
	if gp := strings.TrimSpace(os.Getenv("GOPATH")); gp != "" {
		base := filepath.Join(gp, "pkg", "tmp")
		if err := os.MkdirAll(base, 0o755); err != nil {
			return "", err
		}
		return os.MkdirTemp(base, prefix)
	}
	return os.MkdirTemp("", prefix)
}

func ResetPostgresPublicSchema(pgURL string) error {
	dir, err := integrationTempDir("ash-pg-reset-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	db, err := OpenPostgres(dir, pgURL)
	if err != nil {
		return fmt.Errorf("open postgres for reset: %w", err)
	}
	defer db.Close()
	if db.Dialect() != "postgres" {
		return fmt.Errorf("reset expects postgres, got %q", db.Dialect())
	}
	sql := `
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO public;
`
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("reset schema: %w", err)
	}
	return db.migrate()
}

func resetPostgresForTest(t *testing.T, pgURL string) {
	t.Helper()
	if err := ResetPostgresPublicSchema(pgURL); err != nil {
		t.Fatalf("reset postgres: %v", err)
	}
}
