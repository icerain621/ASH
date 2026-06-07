package store

import (
	"strings"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

// SchemaProfileInfo summarizes golang-migrate vs AutoMigrate mode for ops.
type SchemaProfileInfo struct {
	Mode                 string `json:"schemaMode"`
	SQLMigrationsEnabled bool   `json:"sqlMigrationsEnabled"`
	AutoMigrateEnabled   bool   `json:"autoMigrateEnabled"`
	SQLMigrationVersion  uint   `json:"sqlMigrationVersion,omitempty"`
	SQLMigrationExpected uint   `json:"sqlMigrationExpected"`
}

// SchemaProfile returns the active schema application mode for dialect.
func SchemaProfile(dialect, databaseURL string) SchemaProfileInfo {
	info := SchemaProfileInfo{
		Mode:                 sqlmigrations.Mode(),
		SQLMigrationsEnabled: sqlmigrations.SQLMigrationsEnabled(dialect),
		AutoMigrateEnabled:   sqlmigrations.AutoMigrateEnabled(dialect),
		SQLMigrationExpected: sqlmigrations.ExpectedVersion(),
	}
	if dialect != "postgres" || !info.SQLMigrationsEnabled {
		return info
	}
	dsn := strings.TrimSpace(databaseURL)
	if dsn == "" {
		dsn = strings.TrimSpace(RuntimeDatabaseURL())
	}
	if dsn == "" {
		return info
	}
	v, dirty, err := sqlmigrations.VersionPostgres(dsn)
	if err == nil && !dirty {
		info.SQLMigrationVersion = v
	}
	return info
}
