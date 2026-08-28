package sqlmigrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/postgres/*.sql
var postgresFiles embed.FS

const (
	SchemaModeAuto     = "automigrate"
	SchemaModeSQL      = "sql"
	SchemaModeDual     = "dual"
	envSchemaMode      = "ASH_SCHEMA_MODE"
	envDisableAuto     = "ASH_DISABLE_AUTOMIGRATE"
	expectedVersion    = 24
	postgresDialect    = "postgres"
)

// Mode returns the schema application mode (automigrate|sql|dual).
func Mode() string {
	if strings.TrimSpace(os.Getenv(envDisableAuto)) == "1" {
		return SchemaModeSQL
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envSchemaMode))) {
	case SchemaModeSQL, SchemaModeAuto, SchemaModeDual:
		return strings.ToLower(strings.TrimSpace(os.Getenv(envSchemaMode)))
	default:
		return SchemaModeDual
	}
}

// SQLMigrationsEnabled reports whether versioned SQL migrations should run for dialect.
func SQLMigrationsEnabled(dialect string) bool {
	if dialect != postgresDialect {
		return false
	}
	switch Mode() {
	case SchemaModeAuto:
		return false
	default:
		return true
	}
}

// AutoMigrateEnabled reports whether GORM AutoMigrate should run after SQL migrations.
func AutoMigrateEnabled(dialect string) bool {
	switch Mode() {
	case SchemaModeSQL:
		return false
	case SchemaModeAuto:
		return true
	default:
		return true
	}
}

// ApplyPostgres runs pending golang-migrate revisions against a Postgres DSN.
func ApplyPostgres(dsn string) (version uint, err error) {
	m, err := newMigrator(dsn)
	if err != nil {
		return 0, err
	}
	defer closeMigrator(m)
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return 0, err
	}
	v, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, err
	}
	if dirty {
		return v, fmt.Errorf("schema_migrations dirty at version %d", v)
	}
	return v, nil
}

// VersionPostgres returns the applied golang-migrate version for Postgres.
func VersionPostgres(dsn string) (version uint, dirty bool, err error) {
	m, err := newMigrator(dsn)
	if err != nil {
		return 0, false, err
	}
	defer closeMigrator(m)
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, dirty, err
}

// DownPostgres rolls back one SQL migration revision.
func DownPostgres(dsn string) (version uint, err error) {
	m, err := newMigrator(dsn)
	if err != nil {
		return 0, err
	}
	defer closeMigrator(m)
	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return 0, err
	}
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if dirty {
		return v, fmt.Errorf("schema_migrations dirty at version %d", v)
	}
	return v, nil
}

// ExpectedVersion is the latest embedded SQL migration revision.
func ExpectedVersion() uint { return expectedVersion }

// ReadPostgresUpSQL returns one embedded up migration file (for catalog parity tests).
func ReadPostgresUpSQL(name string) (string, error) {
	data, err := postgresFiles.ReadFile("migrations/postgres/" + name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func newMigrator(dsn string) (*migrate.Migrate, error) {
	sub, err := fs.Sub(postgresFiles, "migrations/postgres")
	if err != nil {
		return nil, err
	}
	source, err := iofs.New(sub, ".")
	if err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return migrate.NewWithInstance("iofs", source, "postgres", driver)
}

func closeMigrator(m *migrate.Migrate) {
	if m == nil {
		return
	}
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		_ = srcErr
	}
	if dbErr != nil {
		_ = dbErr
	}
}
