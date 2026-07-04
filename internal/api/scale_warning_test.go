package api

import (
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestScaleReadinessWarningsRLSPolicyDrift(t *testing.T) {
	w := scaleReadinessWarnings(store.DatabaseProfileInfo{
		Dialect:                   "postgres",
		PostgresRLSEnabled:        true,
		PostgresRLSPolicyCount:    10,
		PostgresRLSPolicyExpected: 41,
	}, store.MigrationSnapshot{})
	if len(w) != 1 || !strings.Contains(w[0], "RLS policies=10 want >=41") {
		t.Fatalf("warnings=%v", w)
	}
}

func TestScaleReadinessWarningsSQLVersionDrift(t *testing.T) {
	w := scaleReadinessWarnings(store.DatabaseProfileInfo{
		Dialect:              "postgres",
		SQLMigrationVersion:  18,
		SQLMigrationExpected: 20,
	}, store.MigrationSnapshot{})
	if len(w) != 1 || !strings.Contains(w[0], "sql migration version=18 want 20") {
		t.Fatalf("warnings=%v", w)
	}
}
