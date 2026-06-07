package store

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

func TestMigrationCatalog_matchesSQLRevisionCount(t *testing.T) {
	catalog := MigrationCatalog()
	if len(catalog) < 25 {
		t.Fatalf("catalog size=%d want >=25", len(catalog))
	}
	if sqlmigrations.ExpectedVersion() < 12 {
		t.Fatalf("expectedVersion=%d want >=12", sqlmigrations.ExpectedVersion())
	}
}
