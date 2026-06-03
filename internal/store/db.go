package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
	dataDir string
	dialect string
}

func Open(dataDir string) (*DB, error) {
	return OpenWithDatabaseURL(dataDir, os.Getenv("ASH_DATABASE_URL"))
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

	db := &DB{DB: gdb, dataDir: dataDir, dialect: target.dialect}
	if err := db.migrate(); err != nil {
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
		} else if !filepath.IsAbs(rest) {
			rest = filepath.Join(dataDir, rest)
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

func (db *DB) migrate() error {
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
		&RAGDocument{},
		&RAGChunk{},
		&ModelUsage{},
		&QualityMetric{},
		&MCPTool{},
		&Feedback{},
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
		&SchemaMeta{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
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
	return filepath.Join(db.dataDir, "runs", runID)
}
