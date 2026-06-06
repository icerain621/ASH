package store

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"
)

const (
	// RLSSpaceSetting is the Postgres session variable holding the active space id.
	RLSSpaceSetting = "app.ash_space_id"
	// RLSBypassSetting allows migration/admin paths to bypass tenant policies when set to "on".
	RLSBypassSetting = "app.ash_rls_bypass"
	rlsPolicyPrefix  = "ash_space_"
)

type rlsSpaceContextKey struct{}
type rlsBypassContextKey struct{}

// PostgresRLSEnabled reports whether tenant RLS policies should be applied (Postgres only).
func PostgresRLSEnabled() bool {
	return strings.TrimSpace(os.Getenv("ASH_POSTGRES_RLS")) == "1"
}

// PostgresRLSForce applies FORCE ROW LEVEL SECURITY so even table owners are filtered.
func PostgresRLSForce() bool {
	return strings.TrimSpace(os.Getenv("ASH_POSTGRES_RLS_FORCE")) == "1"
}

// WithRLSSpaceContext attaches the request space for GORM RLS callbacks.
func WithRLSSpaceContext(ctx context.Context, spaceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, rlsSpaceContextKey{}, strings.TrimSpace(spaceID))
}

// RLSSpaceFromContext returns the active tenant space from context.
func RLSSpaceFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(rlsSpaceContextKey{}).(string)
	return v, ok && v != ""
}

// WithRLSBypassContext marks the context as migration/admin (bypass tenant filter).
func WithRLSBypassContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, rlsBypassContextKey{}, true)
}

// RLSBypassFromContext reports whether bypass is active for this context.
func RLSBypassFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(rlsBypassContextKey{}).(bool)
	return v
}

// PostgresRLSTable describes a tenant-scoped table and its space column.
type PostgresRLSTable struct {
	Table       string
	SpaceColumn string
}

// PostgresRLSTables returns tables protected by space_id (or id for spaces).
func PostgresRLSTables() []PostgresRLSTable {
	return []PostgresRLSTable{
		{Table: "runs", SpaceColumn: "space_id"},
		{Table: "memory_records", SpaceColumn: "space_id"},
		{Table: "memory_edges", SpaceColumn: "space_id"},
		{Table: "rag_documents", SpaceColumn: "space_id"},
		{Table: "rag_chunks", SpaceColumn: "space_id"},
		{Table: "quality_metrics", SpaceColumn: "space_id"},
		{Table: "mcp_tools", SpaceColumn: "space_id"},
		{Table: "feedback", SpaceColumn: "space_id"},
		{Table: "repo_connections", SpaceColumn: "space_id"},
		{Table: "ci_runs", SpaceColumn: "space_id"},
		{Table: "ci_jobs", SpaceColumn: "space_id"},
		{Table: "ci_diagnoses", SpaceColumn: "space_id"},
		{Table: "alert_rules", SpaceColumn: "space_id"},
		{Table: "alert_events", SpaceColumn: "space_id"},
		{Table: "alert_silences", SpaceColumn: "space_id"},
		{Table: "release_records", SpaceColumn: "space_id"},
		{Table: "release_checklist_items", SpaceColumn: "space_id"},
		{Table: "release_gate_results", SpaceColumn: "space_id"},
		{Table: "rollback_drills", SpaceColumn: "space_id"},
		{Table: "secret_records", SpaceColumn: "space_id"},
		{Table: "audit_log", SpaceColumn: "space_id"},
		{Table: "approval_requests", SpaceColumn: "space_id"},
		{Table: "resource_scopes", SpaceColumn: "space_id"},
		{Table: "audit_exports", SpaceColumn: "space_id"},
		{Table: "audit_policies", SpaceColumn: "space_id"},
		{Table: "plugin_registry", SpaceColumn: "space_id"},
		{Table: "improve_proposals", SpaceColumn: "space_id"},
		{Table: "spaces", SpaceColumn: "id"},
	}
}

// PostgresRLSExpectedPolicyCount returns installed policy cardinality when RLS is fully applied.
func PostgresRLSExpectedPolicyCount() int {
	return len(PostgresRLSTables()) + len(postgresRLSRunScopedTables())
}

// postgresRLSRunScopedTables are child tables filtered via runs.space_id (phase 2 skeleton).
func postgresRLSRunScopedTables() []string {
	return []string{
		"run_steps", "tool_calls", "agent_tasks", "artifact_index", "checkpoints", "run_events",
	}
}

func postgresRLSPolicyExpr(spaceColumn string) string {
	col := quoteIdent(spaceColumn)
	return fmt.Sprintf(`(
  current_setting('%s', true) = 'on'
  OR (
    NULLIF(current_setting('%s', true), '') IS NOT NULL
    AND %s = current_setting('%s', true)
  )
)`, RLSBypassSetting, RLSSpaceSetting, col, RLSSpaceSetting)
}

func postgresRLSRunPolicyExpr() string {
	return fmt.Sprintf(`(
  current_setting('%s', true) = 'on'
  OR EXISTS (
    SELECT 1 FROM runs r
    WHERE r.id = run_id
      AND r.space_id = current_setting('%s', true)
  )
)`, RLSBypassSetting, RLSSpaceSetting)
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// ApplyPostgresRLSPolicies enables RLS and (re)creates tenant isolation policies.
func ApplyPostgresRLSPolicies(db *DB) error {
	if db == nil || db.Dialect() != "postgres" {
		return nil
	}
	force := PostgresRLSForce()
	for _, tbl := range PostgresRLSTables() {
		if err := applyPostgresRLSPolicy(db, tbl.Table, tbl.SpaceColumn, postgresRLSPolicyExpr(tbl.SpaceColumn), force); err != nil {
			return err
		}
	}
	for _, table := range postgresRLSRunScopedTables() {
		if err := applyPostgresRLSPolicy(db, table, "run_id", postgresRLSRunPolicyExpr(), force); err != nil {
			return err
		}
	}
	return nil
}

func applyPostgresRLSPolicy(db *DB, table, spaceColumn, expr string, force bool) error {
	policy := rlsPolicyPrefix + table
	qTable := quoteIdent(table)
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", qTable)).Error; err != nil {
		return fmt.Errorf("enable rls on %s: %w", table, err)
	}
	if force {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE %s FORCE ROW LEVEL SECURITY", qTable)).Error; err != nil {
			return fmt.Errorf("force rls on %s: %w", table, err)
		}
	}
	_ = db.Exec(fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s", quoteIdent(policy), qTable))
	create := fmt.Sprintf(
		"CREATE POLICY %s ON %s USING (%s) WITH CHECK (%s)",
		quoteIdent(policy), qTable, expr, expr,
	)
	if err := db.Exec(create).Error; err != nil {
		return fmt.Errorf("create policy %s on %s (%s): %w", policy, table, spaceColumn, err)
	}
	return nil
}

// CountPostgresRLSPolicies returns ash tenant policies currently installed.
func CountPostgresRLSPolicies(db *DB) (int64, error) {
	if db == nil || db.Dialect() != "postgres" {
		return 0, nil
	}
	var count int64
	err := db.Raw(`
SELECT COUNT(*) FROM pg_policies
WHERE schemaname = 'public' AND policyname LIKE ?`, rlsPolicyPrefix+"%").Scan(&count).Error
	return count, err
}

const rlsSkipCallbackKey = "ash:rls_skip_callback"

// SetRLSSession applies transaction-local space/bypass settings on tx.
func SetRLSSession(tx *gorm.DB, spaceID string, bypass bool) error {
	if tx == nil {
		return nil
	}
	sess := tx.Session(&gorm.Session{})
	if sess.Statement != nil {
		sess.Statement.Settings.Store(rlsSkipCallbackKey, true)
	}
	if bypass {
		return sess.Exec("SELECT set_config(?, ?, true)", RLSBypassSetting, "on").Error
	}
	if strings.TrimSpace(spaceID) == "" {
		return nil
	}
	return sess.Exec("SELECT set_config(?, ?, true)", RLSSpaceSetting, strings.TrimSpace(spaceID)).Error
}

// TransactionWithRLSSpace runs fn inside a transaction scoped to spaceID.
func (db *DB) TransactionWithRLSSpace(spaceID string, fn func(tx *gorm.DB) error) error {
	if db == nil || db.DB == nil {
		return fmt.Errorf("database is nil")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if db.Dialect() == "postgres" && PostgresRLSEnabled() {
			if err := SetRLSSession(tx, spaceID, false); err != nil {
				return err
			}
		}
		return fn(tx)
	})
}

// TransactionWithRLSBypass runs fn with migration/admin bypass enabled.
func (db *DB) TransactionWithRLSBypass(fn func(tx *gorm.DB) error) error {
	if db == nil || db.DB == nil {
		return fmt.Errorf("database is nil")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if db.Dialect() == "postgres" && PostgresRLSEnabled() {
			if err := SetRLSSession(tx, "", true); err != nil {
				return err
			}
		}
		return fn(tx)
	})
}

func registerRLSCallbacks(gdb *gorm.DB) {
	inject := func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil {
			return
		}
		if _, skip := tx.Statement.Settings.Load(rlsSkipCallbackKey); skip {
			return
		}
		ctx := tx.Statement.Context
		if ctx == nil {
			return
		}
		if RLSBypassFromContext(ctx) {
			_ = SetRLSSession(tx, "", true)
			return
		}
		if space, ok := RLSSpaceFromContext(ctx); ok {
			_ = SetRLSSession(tx, space, false)
		}
	}
	cb := gdb.Callback()
	cb.Query().Before("gorm:query").Register("ash:rls", inject)
	cb.Create().Before("gorm:create").Register("ash:rls", inject)
	cb.Update().Before("gorm:update").Register("ash:rls", inject)
	cb.Delete().Before("gorm:delete").Register("ash:rls", inject)
	cb.Row().Before("gorm:row").Register("ash:rls", inject)
	cb.Raw().Before("gorm:raw").Register("ash:rls", inject)
}

// WithRLSBypassIfNeeded runs fn with migration bypass when Postgres RLS is active.
func (db *DB) WithRLSBypassIfNeeded(fn func(gdb *gorm.DB) error) error {
	if db == nil || db.DB == nil {
		return fmt.Errorf("database is nil")
	}
	if db.Dialect() == "postgres" && PostgresRLSEnabled() {
		return db.TransactionWithRLSBypass(fn)
	}
	return fn(db.DB)
}

func maybeConfigurePostgresRLS(db *DB, databaseURL string) error {
	if db == nil || db.Dialect() != "postgres" || !PostgresRLSEnabled() {
		return nil
	}
	registerRLSCallbacks(db.DB)
	if !shouldApplyPostgresRLSPolicies(databaseURL) {
		return nil
	}
	return ApplyPostgresRLSPolicies(db)
}

// shouldApplyPostgresRLSPolicies returns false for ash_app worker URLs (DDL is owner-only).
func shouldApplyPostgresRLSPolicies(databaseURL string) bool {
	app := strings.TrimSpace(os.Getenv("ASH_DATABASE_APP_URL"))
	if app == "" {
		return true
	}
	return !postgresDatabaseURLsMatch(databaseURL, app)
}

func postgresDatabaseURLsMatch(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}
