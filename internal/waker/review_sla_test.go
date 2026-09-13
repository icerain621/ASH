package waker

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestSeedProbeDutiesCreatesReviewSLA(t *testing.T) {
	t.Setenv("ASH_WAKER_ENABLE_PROBES", "")
	db := openTestDB(t)
	svc := NewService(db)
	if err := svc.SeedProbeDuties("local"); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListDuties("local")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range list {
		if d.Kind == KindReviewSLA {
			found = true
			if d.Enabled {
				t.Fatal("review_sla should be disabled by default without ASH_WAKER_ENABLE_PROBES")
			}
			if !strings.Contains(d.ConfigJSON, "report") {
				t.Fatalf("config=%q", d.ConfigJSON)
			}
		}
	}
	if !found {
		t.Fatalf("want review_sla duty: %+v", list)
	}
}

func TestRunReviewSLAFlagsOldMemory(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	old := now.Add(-100 * time.Hour)
	fresh := now.Add(-1 * time.Hour)
	if err := db.Create(&store.MemoryRecord{
		ID: "mem_old", Layer: "L0", Status: "candidate", SpaceID: "local",
		SchemaVersion: 1, Title: "old", Body: "old body", TagsJSON: "[]",
		Sensitivity: "normal", CreatedAt: old, UpdatedAt: old,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.MemoryRecord{
		ID: "mem_fresh", Layer: "L0", Status: "candidate", SpaceID: "local",
		SchemaVersion: 1, Title: "fresh", Body: "fresh body", TagsJSON: "[]",
		Sensitivity: "normal", CreatedAt: fresh, UpdatedAt: fresh,
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	duty := store.WakerDuty{
		ID: "wd_sla", SpaceID: "local", Kind: KindReviewSLA,
		ConfigJSON: `{"action":"report"}`, Enabled: true, IntervalMs: 300000,
	}
	resp, err := svc.runReviewSLA(duty, true)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Flagged != 1 || resp.Matched != 1 {
		t.Fatalf("want only old flagged: %+v", resp)
	}
	if !strings.Contains(resp.Summary, "sla_breach:memory:mem_old") {
		t.Fatalf("summary=%q", resp.Summary)
	}
	if strings.Contains(resp.Summary, "mem_fresh") {
		t.Fatalf("fresh should not breach: %q", resp.Summary)
	}
	if !strings.Contains(resp.Summary, "review_sla breaches=1") {
		t.Fatalf("summary=%q", resp.Summary)
	}
	// dryRun must not require write; confirm no audit row yet
	var n int64
	if err := db.Model(&store.AuditLog{}).Where("event_type = ?", "review.sla_breach").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("dryRun should not write audit, got %d", n)
	}

	resp2, err := svc.runReviewSLA(duty, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Flagged != 1 {
		t.Fatalf("resp2=%+v", resp2)
	}
	if err := db.Model(&store.AuditLog{}).Where("event_type = ?", "review.sla_breach").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want audit row, got %d", n)
	}
}
