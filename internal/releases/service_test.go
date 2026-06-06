package releases

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestReleaseChecklistGateAndRollbackDrill(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	rel, err := svc.Create(CreateRequest{SpaceID: "local", Version: "v0.3.0", Title: "MVP closeout", CreatedBy: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.Checklist("local", rel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 10 {
		t.Fatalf("checklist items=%d want MVP checklist", len(items))
	}
	items, err = svc.PatchChecklist("local", rel.ID, "dev", []ChecklistUpdate{{ItemKey: items[0].ItemKey, Status: "done", EvidenceRef: "doc:11"}})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Status != "done" {
		t.Fatalf("first item=%+v want done", items[0])
	}

	seedPassingGateEvidence(t, db, now)
	gate, err := svc.EvaluateGate("local", rel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gate.Overall != "pass" {
		t.Fatalf("gate=%+v want pass", gate)
	}

	if err := db.Create(&store.AlertEvent{
		ID: "alert_block_release", SpaceID: "local", RuleName: "运行失败率",
		Severity: "critical", Status: "active", TargetType: "metric", TargetID: "run_failure_rate",
		Fingerprint: "test:block", Message: "blocking alert", EvidenceRefsJSON: "[]",
		TriggeredAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	gate, err = svc.EvaluateGate("local", rel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gate.Overall != "block" {
		t.Fatalf("gate=%+v want block with active critical alert", gate)
	}

	drill, err := svc.CreateRollbackDrill(RollbackDrillRequest{
		SpaceID: "local", ReleaseID: rel.ID, Scenario: "rollback to previous image",
		Status: "passed", DurationMs: 120000, EvidenceRefs: []string{"runbook:rollback"}, CreatedBy: "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if drill.ID == "" || drill.Status != "passed" {
		t.Fatalf("drill=%+v want persisted drill", drill)
	}
}

func seedPassingGateEvidence(t *testing.T, db *store.DB, now time.Time) {
	t.Helper()
	done := now.Add(time.Minute)
	rows := []any{
		&store.CIRun{
			ID: "ci_release_pass", SpaceID: "local", ConnectionID: "conn_release", ProviderRunID: "42",
			Workflow: "ci", Status: "completed", Conclusion: "success", Attempt: 1,
			StartedAt: &now, CompletedAt: &done, CreatedAt: now, UpdatedAt: done,
		},
		&store.AuditLog{ID: "aud_doctor_m3", SpaceID: "local", EventType: "doctor.suite_completed", PayloadJSON: `{"suite":"M3","pass":10,"fail":0}`, CreatedAt: now},
		&store.AuditLog{ID: "aud_doctor_all", SpaceID: "local", EventType: "doctor.suite_completed", PayloadJSON: `{"suite":"ALL","pass":30,"fail":0}`, CreatedAt: now},
		&store.AuditLog{ID: "aud_pg", SpaceID: "local", EventType: "postgres.e2e_completed", PayloadJSON: `{"status":"pass"}`, CreatedAt: now},
		&store.AuditLog{ID: "aud_execgo", SpaceID: "local", EventType: "execgo.live_smoke", PayloadJSON: `{"status":"pass"}`, CreatedAt: now},
		&store.Feedback{ID: "fb_release_ok", SpaceID: "local", TargetType: "release", TargetID: "rel", Rating: 5, Category: "quality", Status: "resolved", Severity: "info", Source: "test", CreatedAt: now, UpdatedAt: now},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}
