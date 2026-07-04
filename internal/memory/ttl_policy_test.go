package memory

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestEffectiveTTLDaysEnvOverride(t *testing.T) {
	t.Setenv("ASH_MEMORY_TTL_L1_DAYS", "45")
	t.Setenv("ASH_MEMORY_TTL_L2_DAYS", "180")
	if got := EffectiveTTLDaysL1(); got != 45 {
		t.Fatalf("L1=%d want 45", got)
	}
	if got := EffectiveTTLDaysL2(); got != 180 {
		t.Fatalf("L2=%d want 180", got)
	}
	if d := DefaultTTLForLayer("L1"); d == nil || *d != 45 {
		t.Fatalf("DefaultTTLForLayer L1=%v", d)
	}
}

func TestEffectiveTTLDaysInvalidEnvUsesBuiltin(t *testing.T) {
	t.Setenv("ASH_MEMORY_TTL_L1_DAYS", "0")
	if got := EffectiveTTLDaysL1(); got != BuiltinTTLDaysL1 {
		t.Fatalf("L1=%d want builtin %d", got, BuiltinTTLDaysL1)
	}
}

func TestMigrateV1ToV2RespectsTTLEnv(t *testing.T) {
	t.Setenv("ASH_MEMORY_TTL_L1_DAYS", "30")
	svc, _, _ := newTestMemory(t)
	now := time.Now().UTC()
	if err := svc.gdb().Create(&store.SchemaMeta{
		Key: MemoryCatalogMetaKey, Value: "1", UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	id := "mem_env_ttl"
	if err := svc.gdb().Create(&store.MemoryRecord{
		ID: id, Layer: "L1", Status: "approved", SpaceID: "local",
		SchemaVersion: 1, Title: "env ttl", Body: "probe",
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunMigrations(RunMigrationRequest{}); err != nil {
		t.Fatal(err)
	}
	var row store.MemoryRecord
	if err := svc.gdb().First(&row, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if row.TTLDays == nil || *row.TTLDays != 30 {
		t.Fatalf("ttl=%v want 30 from env", row.TTLDays)
	}
}
