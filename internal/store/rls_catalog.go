package store

import (
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

// RLSDeferredTable documents migration-catalog tables not yet under tenant/run/memory/org RLS.
type RLSDeferredTable struct {
	Table  string
	Reason string
}

// PostgresRLSDeferredTables lists tables intentionally excluded from RLS SQL (empty when catalog is complete).
func PostgresRLSDeferredTables() []RLSDeferredTable {
	return nil
}

// MigrationCatalogRLSGaps returns migration tables without tenant/run/memory/org/global/deferred classification.
func MigrationCatalogRLSGaps() []string {
	classified := postgresRLSClassifiedTables()
	var gaps []string
	for _, table := range MigrationCatalog() {
		if _, ok := classified[table]; !ok {
			gaps = append(gaps, table)
		}
	}
	return gaps
}

func postgresRLSClassifiedTables() map[string]struct{} {
	out := make(map[string]struct{})
	for _, name := range PostgresRLSGlobalTables() {
		out[name] = struct{}{}
	}
	for _, ent := range PostgresRLSDeferredTables() {
		out[ent.Table] = struct{}{}
	}
	for _, tbl := range PostgresRLSTables() {
		out[tbl.Table] = struct{}{}
	}
	for _, tbl := range postgresRLSRunScopedTables() {
		out[tbl] = struct{}{}
	}
	for _, tbl := range postgresRLSMemoryScopedTables() {
		out[tbl] = struct{}{}
	}
	for _, tbl := range PostgresRLSOrgScopedTables() {
		out[tbl.Table] = struct{}{}
	}
	return out
}

// VerifyRLSMigrationSQL ensures embedded RLS migrations cover the Go catalogs.
func VerifyRLSMigrationSQL() error {
	raw13, err := sqlmigrations.ReadPostgresUpSQL("000013_rls_tenant_policies.up.sql")
	if err != nil {
		return err
	}
	raw18, err := sqlmigrations.ReadPostgresUpSQL("000018_rls_run_scoped_extensions.up.sql")
	if err != nil {
		return err
	}
	raw19, err := sqlmigrations.ReadPostgresUpSQL("000019_rls_memory_scoped_extensions.up.sql")
	if err != nil {
		return err
	}
	raw20, err := sqlmigrations.ReadPostgresUpSQL("000020_rls_org_identity.up.sql")
	if err != nil {
		return err
	}
	combined := raw13 + "\n" + raw18 + "\n" + raw19 + "\n" + raw20

	for _, tbl := range PostgresRLSTables() {
		needle := "('" + tbl.Table + "'"
		if !strings.Contains(raw13, needle) {
			return fmt.Errorf("migration 000013 missing tenant table %q", tbl.Table)
		}
	}
	for _, tbl := range postgresRLSRunScopedTables() {
		if !strings.Contains(combined, "'"+tbl+"'") {
			return fmt.Errorf("RLS migrations missing run-scoped table %q", tbl)
		}
	}
	for _, tbl := range postgresRLSMemoryScopedTables() {
		if !strings.Contains(combined, "'"+tbl+"'") {
			return fmt.Errorf("RLS migrations missing memory-scoped table %q", tbl)
		}
	}
	for _, tbl := range PostgresRLSOrgScopedTables() {
		if !strings.Contains(combined, "'"+tbl.Table+"'") {
			return fmt.Errorf("RLS migrations missing org-scoped table %q", tbl.Table)
		}
	}
	for _, name := range PostgresRLSGlobalTables() {
		if strings.Contains(raw13, "('"+name+"'") {
			return fmt.Errorf("migration 000013 must not tenant-scope global table %q", name)
		}
	}
	for _, ent := range PostgresRLSDeferredTables() {
		if strings.Contains(combined, "('"+ent.Table+"'") {
			return fmt.Errorf("RLS migrations must not tenant-scope deferred table %q", ent.Table)
		}
	}
	return nil
}

// FormatRLSCatalogSummary is a compact doctor/log line for RLS classification.
func FormatRLSCatalogSummary() string {
	return fmt.Sprintf(
		"tenant=%d run=%d memory=%d org=%d global=%d deferred=%d policies=%d",
		len(PostgresRLSTables()),
		len(postgresRLSRunScopedTables()),
		len(postgresRLSMemoryScopedTables()),
		len(PostgresRLSOrgScopedTables()),
		len(PostgresRLSGlobalTables()),
		len(PostgresRLSDeferredTables()),
		PostgresRLSExpectedPolicyCount(),
	)
}
