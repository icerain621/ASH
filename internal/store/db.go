package store

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

type DB struct {
	*gorm.DB
	dataDir string
	dialect string
	dsn     string // connection DSN used at open (postgres migrations must use this, not RuntimeDatabaseURL)
}

// RuntimeDatabaseURL returns the worker connection URL (prefers ASH_DATABASE_APP_URL).
func RuntimeDatabaseURL() string {
	if app := strings.TrimSpace(os.Getenv("ASH_DATABASE_APP_URL")); app != "" {
		return app
	}
	return strings.TrimSpace(os.Getenv("ASH_DATABASE_URL"))
}

func Open(dataDir string) (*DB, error) {
	return OpenWithDatabaseURL(dataDir, RuntimeDatabaseURL())
}

func OpenWithDatabaseURL(dataDir, databaseURL string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}

	target, err := resolveDatabaseTarget(dataDir, databaseURL)
	if err != nil {
		return nil, err
	}

	gdb, err := gorm.Open(target.dialector(), &gorm.Config{Logger: gormLogger()})
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", target.dialect, err)
	}

	db := &DB{DB: gdb, dataDir: dataDir, dialect: target.dialect, dsn: target.dsn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	if err := attachDualWrite(db, dataDir); err != nil {
		return nil, err
	}
	if err := maybeConfigurePostgresRLS(db, databaseURL); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

type databaseTarget struct {
	dialect string
	dsn     string
}

func (t databaseTarget) dialector() gorm.Dialector {
	switch t.dialect {
	case "postgres":
		return postgres.Open(t.dsn)
	default:
		return sqlite.Open(t.dsn)
	}
}

func resolveDatabaseTarget(dataDir, databaseURL string) (databaseTarget, error) {
	raw := strings.TrimSpace(databaseURL)
	defaultSQLite := filepath.Join(dataDir, "ash.db")
	if raw == "" {
		return databaseTarget{dialect: "sqlite", dsn: defaultSQLite}, nil
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"), looksLikePostgresKeywordDSN(lower):
		return databaseTarget{dialect: "postgres", dsn: raw}, nil
	case strings.HasPrefix(lower, "sqlite://"):
		rest := raw[len("sqlite://"):]
		if rest == "" || rest == "/" {
			rest = defaultSQLite
		} else if strings.HasPrefix(strings.ToLower(rest), "file:") {
			// Keep SQLite URI filenames intact, e.g. file::memory:?cache=shared.
		} else {
			rest = resolveSQLiteFilePath(dataDir, rest)
		}
		return databaseTarget{dialect: "sqlite", dsn: rest}, nil
	case strings.HasPrefix(lower, "sqlite:"):
		rest := raw[len("sqlite:"):]
		if rest == "" {
			rest = defaultSQLite
		} else if !strings.HasPrefix(strings.ToLower(rest), "file:") && !filepath.IsAbs(rest) {
			rest = filepath.Join(dataDir, rest)
		}
		return databaseTarget{dialect: "sqlite", dsn: rest}, nil
	case strings.HasPrefix(lower, "file:"), raw == ":memory:", filepath.IsAbs(raw), strings.HasPrefix(raw, "."), strings.HasSuffix(lower, ".db"):
		return databaseTarget{dialect: "sqlite", dsn: raw}, nil
	case strings.Contains(lower, "://"):
		return databaseTarget{}, fmt.Errorf("unsupported database URL scheme in ASH_DATABASE_URL: %s", raw)
	default:
		return databaseTarget{}, fmt.Errorf("unsupported ASH_DATABASE_URL %q: use postgres://, postgresql://, libpq keyword DSN, sqlite://, file:, or leave empty for dev SQLite", raw)
	}
}

func looksLikePostgresKeywordDSN(lower string) bool {
	return strings.Contains(lower, "host=") || strings.Contains(lower, "dbname=") || strings.Contains(lower, "sslmode=")
}

// resolveSQLiteFilePath normalizes sqlite:// paths (incl. sqlite:///C:/ on Windows).
func resolveSQLiteFilePath(dataDir, rest string) string {
	path := rest
	if strings.HasPrefix(path, "/") {
		if trimmed := strings.TrimPrefix(path, "/"); filepath.IsAbs(trimmed) {
			path = trimmed
		}
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dataDir, path)
}

// postgresMigrationDSN is the DSN used for SQL migrations. Prefer the handle's
// open-time DSN: RuntimeDatabaseURL() may point at ash_app (ASH_DATABASE_APP_URL),
// which cannot run owner DDL / CREATE ROLE grants.
func postgresMigrationDSN(db *DB) string {
	if db == nil {
		return strings.TrimSpace(RuntimeDatabaseURL())
	}
	if pgDSN := strings.TrimSpace(db.dsn); pgDSN != "" {
		return pgDSN
	}
	return strings.TrimSpace(RuntimeDatabaseURL())
}

func (db *DB) migrate() error {
	if sqlmigrations.SQLMigrationsEnabled(db.dialect) {
		pgDSN := postgresMigrationDSN(db)
		if pgDSN == "" {
			return fmt.Errorf("postgres sql migrations require ASH_DATABASE_URL")
		}
		if _, err := sqlmigrations.ApplyPostgres(pgDSN); err != nil {
			return fmt.Errorf("sql migrations: %w", err)
		}
	}
	if !sqlmigrations.AutoMigrateEnabled(db.dialect) {
		return db.ensureSchemaMeta()
	}
	if err := db.AutoMigrate(
		&RunRecord{},
		&RunStep{},
		&ToolCall{},
		&AgentTask{},
		&ArtifactIndex{},
		&Checkpoint{},
		&RunEvent{},
		&MemoryRecord{},
		&MemoryEvidence{},
		&MemoryReview{},
		&MemoryEdge{},
		&MemoryMigration{},
		&RAGDocument{},
		&RAGChunk{},
		&ModelUsage{},
		&QualityMetric{},
		&MCPTool{},
		&Feedback{},
		&RepoConnection{},
		&CIRun{},
		&CIJob{},
		&CIDiagnosis{},
		&AlertRule{},
		&AlertEvent{},
		&AlertSilence{},
		&ReleaseRecord{},
		&ReleaseChecklistItem{},
		&ReleaseGateResult{},
		&RollbackDrill{},
		&SecretRecord{},
		&AuditLog{},
		&ApprovalRequest{},
		&User{},
		&Org{},
		&Space{},
		&Member{},
		&Role{},
		&ResourceScope{},
		&AuditExport{},
		&AuditPolicy{},
		&PluginRegistry{},
		&ImproveProposal{},
		&HarnessProfileVersion{},
		&ScenarioPatchDraft{},
		&SchemaMeta{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	return db.ensureSchemaMeta()
}

func (db *DB) ensureSchemaMeta() error {
	var meta SchemaMeta
	res := db.First(&meta, "key = ?", "schema_version")
	if res.Error == gorm.ErrRecordNotFound {
		return db.Create(&SchemaMeta{
			Key:       "schema_version",
			Value:     "1",
			UpdatedAt: time.Now().UTC(),
		}).Error
	}
	return res.Error
}

func gormLogger() logger.Interface {
	return logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
	})
}

func (db *DB) DataDir() string { return db.dataDir }

func (db *DB) Dialect() string { return db.dialect }

func (db *DB) RunDir(runID string) string {
	return artifacts.RunDir(db.dataDir, runID)
}

// BindContext returns a shallow copy of DB with the GORM handle bound to ctx (for Postgres RLS session vars).
func (db *DB) BindContext(ctx context.Context) *DB {
	if db == nil || ctx == nil {
		return db
	}
	return &DB{DB: db.DB.WithContext(ctx), dataDir: db.dataDir, dialect: db.dialect, dsn: db.dsn}
}

// Close releases the underlying database connection (required on Windows before temp dirs are removed).
func (db *DB) Close() error {
	if db == nil || db.DB == nil {
		return nil
	}
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
