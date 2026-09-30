package authz

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

func TestSeedAndLoadScenarioPolicies(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	if err := SeedScenarioScopes(nil, "local", now); err != nil {
		t.Fatal(err)
	}
	if err := SeedScenarioScopesTx(nil, "local", now); err != nil {
		t.Fatal(err)
	}
	if err := SeedScenarioScopes(db, "sp_seed", now); err != nil {
		t.Fatal(err)
	}
	// idempotent
	if err := SeedScenarioScopes(db, "sp_seed", now); err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return SeedScenarioScopesTx(tx, "sp_tx", now)
	}); err != nil {
		t.Fatal(err)
	}

	pol, err := LoadScenarioPolicy(nil, "sp_seed", "feature_delivery", "1.0.0")
	if err != nil || pol == "" {
		t.Fatalf("nil db default: %v %q", err, pol)
	}
	pol, err = LoadScenarioPolicy(db, "sp_seed", "feature_delivery", "1.0.0")
	if err != nil || pol == "" {
		t.Fatalf("seeded load: %v", err)
	}
	pol, err = LoadScenarioPolicy(db, "sp_seed", "missing_scenario", "9.9.9")
	if err != nil || pol == "" {
		t.Fatalf("fallback default: %v", err)
	}

	rows, err := ScenarioPoliciesForSpace(db, "sp_seed")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 3 {
		t.Fatalf("rows=%d want >=3 seeded", len(rows))
	}
	empty, err := ScenarioPoliciesForSpace(db, "no_such_space")
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) == 0 {
		t.Fatal("empty space should fall back to DefaultScenarioMatrix")
	}
}

func TestEvaluateScenarioToolFallbackAndAllow(t *testing.T) {
	policy := DefaultScenarioPolicyJSON("feature_delivery", "1.0.0")
	ok, _ := EvaluateScenarioTool(policy, "unknown-role", "git.status")
	if !ok {
		t.Fatal("unknown role should fall back to maintainer allow")
	}
	ok, reason := EvaluateScenarioTool(`{"toolMatrix":{"operator":{"allow":["git.*"]}}}`, "operator", "apply_patch")
	if ok || reason == "" {
		t.Fatalf("want allow-list deny, ok=%v reason=%q", ok, reason)
	}
	ok, _ = EvaluateScenarioTool(`{"toolMatrix":{"operator":{"allow":[]}}}`, "operator", "anything")
	if !ok {
		t.Fatal("empty allow means open")
	}
	ok, _ = EvaluateScenarioTool("{", "operator", "git.status")
	if !ok {
		t.Fatal("invalid JSON fails open")
	}
}
