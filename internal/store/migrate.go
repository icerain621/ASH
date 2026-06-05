package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TableStat is row counts for one migrated table.
type TableStat struct {
	Table      string `json:"table"`
	SourceRows int64  `json:"sourceRows"`
	TargetRows int64  `json:"targetRows"`
	Match      bool   `json:"match"`
}

// MigrationPlan summarizes a sqlite→postgres migration window.
type MigrationPlan struct {
	SourceDialect string      `json:"sourceDialect"`
	TargetDialect string      `json:"targetDialect"`
	SourceDSN     string      `json:"sourceDsn"`
	TargetDSN     string      `json:"targetDsn"`
	Tables        []TableStat `json:"tables"`
	Ready         bool        `json:"ready"`
}

// CopyOptions controls bulk copy.
type CopyOptions struct {
	BatchSize int
	DryRun    bool
	Tables    []string
}

// CopyTableResult is per-table copy output.
type CopyTableResult struct {
	Table  string `json:"table"`
	Copied int64  `json:"copied"`
	DryRun bool   `json:"dryRun,omitempty"`
}

// CopyReport aggregates a copy or sync run.
type CopyReport struct {
	StartedAt  time.Time         `json:"startedAt"`
	FinishedAt time.Time         `json:"finishedAt"`
	Incremental bool             `json:"incremental,omitempty"`
	Tables     []CopyTableResult `json:"tables"`
	TotalCopied int64            `json:"totalCopied"`
}

// SyncState tracks incremental dual-write / sync checkpoints.
type SyncState struct {
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// Migrator copies metadata between two ASH databases.
type Migrator struct {
	Source      *DB
	Target      *DB
	sqlitePath  string
	postgresURL string
}

// DefaultSQLitePath returns the dev sqlite file under a data directory.
func DefaultSQLitePath(dataDir string) string {
	return filepath.Join(dataDir, "ash.db")
}

// OpenSQLite opens an on-disk sqlite database (ignores ASH_DATABASE_URL).
func OpenSQLite(dataDir, sqlitePath string) (*DB, error) {
	if strings.TrimSpace(sqlitePath) == "" {
		sqlitePath = DefaultSQLitePath(dataDir)
	}
	return OpenWithDatabaseURL(dataDir, sqlitePath)
}

// OpenPostgres opens postgres using an explicit URL (ignores ASH_DATABASE_URL).
func OpenPostgres(dataDir, postgresURL string) (*DB, error) {
	postgresURL = strings.TrimSpace(postgresURL)
	if postgresURL == "" {
		return nil, fmt.Errorf("postgres url is required")
	}
	return OpenWithDatabaseURL(dataDir, postgresURL)
}

// NewMigrator connects to sqlite source and postgres target.
func NewMigrator(dataDir, sqlitePath, postgresURL string) (*Migrator, error) {
	src, err := OpenSQLite(dataDir, sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite source: %w", err)
	}
	if src.Dialect() != "sqlite" {
		_ = src.Close()
		return nil, fmt.Errorf("source dialect=%q want sqlite", src.Dialect())
	}
	dst, err := OpenPostgres(dataDir, postgresURL)
	if err != nil {
		_ = src.Close()
		return nil, fmt.Errorf("open postgres target: %w", err)
	}
	if dst.Dialect() != "postgres" {
		_ = src.Close()
		_ = dst.Close()
		return nil, fmt.Errorf("target dialect=%q want postgres", dst.Dialect())
	}
	return &Migrator{Source: src, Target: dst, sqlitePath: sqlitePath, postgresURL: postgresURL}, nil
}

// Close closes underlying database handles.
func (m *Migrator) Close() error {
	if m == nil {
		return nil
	}
	var first error
	if m.Source != nil {
		if err := m.Source.Close(); err != nil && first == nil {
			first = err
		}
	}
	if m.Target != nil {
		if err := m.Target.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Plan returns per-table row counts on source and target.
func (m *Migrator) Plan() (*MigrationPlan, error) {
	if m == nil || m.Source == nil || m.Target == nil {
		return nil, fmt.Errorf("migrator is not initialized")
	}
	plan := &MigrationPlan{
		SourceDialect: m.Source.Dialect(),
		TargetDialect: m.Target.Dialect(),
		SourceDSN:     m.sqlitePath,
		TargetDSN:     redactDSN(m.postgresURL),
	}
	for _, ent := range migrationEntities() {
		srcCount, err := countModel(m.Source.DB, ent.model)
		if err != nil {
			return nil, fmt.Errorf("count source %s: %w", ent.table, err)
		}
		dstCount, err := countModel(m.Target.DB, ent.model)
		if err != nil {
			return nil, fmt.Errorf("count target %s: %w", ent.table, err)
		}
		plan.Tables = append(plan.Tables, TableStat{
			Table: ent.table, SourceRows: srcCount, TargetRows: dstCount, Match: srcCount == dstCount,
		})
	}
	plan.Ready = m.Source.Dialect() == "sqlite" && m.Target.Dialect() == "postgres"
	return plan, nil
}

// Copy upserts all rows from source into target.
func (m *Migrator) Copy(opts CopyOptions) (*CopyReport, error) {
	return m.copy(opts, nil)
}

// Sync copies rows updated since the saved checkpoint (or all rows when no checkpoint).
func (m *Migrator) Sync(dataDir string, opts CopyOptions) (*CopyReport, error) {
	state, err := LoadSyncState(dataDir)
	if err != nil {
		return nil, err
	}
	report, err := m.copy(opts, state.LastSyncAt)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	state.LastSyncAt = &now
	state.UpdatedAt = now
	if err := SaveSyncState(dataDir, state); err != nil {
		return nil, err
	}
	report.Incremental = true
	return report, nil
}

func (m *Migrator) copy(opts CopyOptions, since *time.Time) (*CopyReport, error) {
	if m == nil || m.Source == nil || m.Target == nil {
		return nil, fmt.Errorf("migrator is not initialized")
	}
	batch := opts.BatchSize
	if batch <= 0 {
		batch = 200
	}
	allow := tableFilter(opts.Tables)
	started := time.Now().UTC()
	report := &CopyReport{StartedAt: started, Tables: make([]CopyTableResult, 0, len(migrationEntities()))}
	for _, ent := range migrationEntities() {
		if !allow(ent.table) {
			continue
		}
		var copied int64
		var err error
		if opts.DryRun {
			copied, err = countModelFiltered(m.Source.DB, ent.model, ent.incremental, since)
		} else {
			copied, err = copyEntity(m.Source.DB, m.Target.DB, ent, batch, since)
		}
		if err != nil {
			return nil, fmt.Errorf("copy %s: %w", ent.table, err)
		}
		report.Tables = append(report.Tables, CopyTableResult{
			Table: ent.table, Copied: copied, DryRun: opts.DryRun,
		})
		report.TotalCopied += copied
	}
	report.FinishedAt = time.Now().UTC()
	return report, nil
}

// Verify ensures source and target row counts match.
func (m *Migrator) Verify() (*MigrationPlan, error) {
	plan, err := m.Plan()
	if err != nil {
		return nil, err
	}
	for _, row := range plan.Tables {
		if !row.Match {
			return plan, fmt.Errorf("table %s source=%d target=%d", row.Table, row.SourceRows, row.TargetRows)
		}
	}
	return plan, nil
}

func countModel(db *gorm.DB, model any) (int64, error) {
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func countModelFiltered(db *gorm.DB, model any, incremental bool, since *time.Time) (int64, error) {
	q := db.Model(model)
	if incremental && since != nil {
		q = q.Where("updated_at > ?", *since)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func copyEntity(src, dst *gorm.DB, ent migrationEntity, batch int, since *time.Time) (int64, error) {
	modelType := reflect.TypeOf(ent.model).Elem()
	var total int64
	offset := 0
	for {
		slicePtr := reflect.New(reflect.SliceOf(modelType))
		q := src.Model(ent.model)
		if ent.incremental && since != nil {
			q = q.Where("updated_at > ?", *since)
		}
		if err := q.Offset(offset).Limit(batch).Find(slicePtr.Interface()).Error; err != nil {
			return total, err
		}
		sliceVal := slicePtr.Elem()
		n := sliceVal.Len()
		if n == 0 {
			break
		}
		rows := sliceVal.Interface()
		if err := dst.Session(&gorm.Session{SkipHooks: true}).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: ent.pk}},
			UpdateAll: true,
		}).CreateInBatches(rows, batch).Error; err != nil {
			return total, err
		}
		total += int64(n)
		offset += n
		if n < batch {
			break
		}
	}
	return total, nil
}

func tableFilter(only []string) func(string) bool {
	if len(only) == 0 {
		return func(string) bool { return true }
	}
	set := make(map[string]struct{}, len(only))
	for _, name := range only {
		set[strings.TrimSpace(strings.ToLower(name))] = struct{}{}
	}
	return func(table string) bool {
		_, ok := set[strings.ToLower(table)]
		return ok
	}
}

func migrationStatePath(dataDir string) string {
	return filepath.Join(dataDir, "migration", "sync-state.json")
}

func dualWriteConfigPath(dataDir string) string {
	return filepath.Join(dataDir, "migration", "dual-write.json")
}

// DualWriteConfigPath returns `.ash/migration/dual-write.json` for a data directory.
func DualWriteConfigPath(dataDir string) string {
	return dualWriteConfigPath(dataDir)
}

// LoadSyncState reads the incremental sync checkpoint.
func LoadSyncState(dataDir string) (*SyncState, error) {
	path := migrationStatePath(dataDir)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SyncState{UpdatedAt: time.Now().UTC()}, nil
		}
		return nil, err
	}
	var state SyncState
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveSyncState persists the incremental sync checkpoint.
func SaveSyncState(dataDir string, state *SyncState) error {
	if state == nil {
		state = &SyncState{}
	}
	state.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(filepath.Dir(migrationStatePath(dataDir)), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(migrationStatePath(dataDir), b, 0o644)
}

// DualWriteConfig is persisted CLI state for a dual-write window.
type DualWriteConfig struct {
	Enabled     bool      `json:"enabled"`
	PostgresURL string    `json:"postgresUrl"`
	SQLitePath  string    `json:"sqlitePath,omitempty"`
	EnabledAt   time.Time `json:"enabledAt"`
}

// LoadDualWriteConfig reads `.ash/migration/dual-write.json`.
func LoadDualWriteConfig(dataDir string) (*DualWriteConfig, error) {
	path := dualWriteConfigPath(dataDir)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DualWriteConfig{}, nil
		}
		return nil, err
	}
	var cfg DualWriteConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveDualWriteConfig writes dual-write CLI state.
func SaveDualWriteConfig(dataDir string, cfg *DualWriteConfig) error {
	if cfg == nil {
		cfg = &DualWriteConfig{}
	}
	if err := os.MkdirAll(filepath.Dir(dualWriteConfigPath(dataDir)), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dualWriteConfigPath(dataDir), b, 0o644)
}
