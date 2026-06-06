package alerts

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestEvaluateLowFeedbackCreatesAlert(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	rows := []store.Feedback{
		{ID: "fb_low_1", SpaceID: "local", TargetType: "run", TargetID: "run_1", Rating: 1, Category: "quality", Status: "open", Severity: "warn", Source: "ui", CreatedAt: now, UpdatedAt: now},
		{ID: "fb_low_2", SpaceID: "local", TargetType: "ci", TargetID: "ci_1", Rating: 2, Category: "ci", Status: "open", Severity: "warn", Source: "ui", CreatedAt: now, UpdatedAt: now},
		{ID: "fb_ok", SpaceID: "local", TargetType: "run", TargetID: "run_2", Rating: 5, Category: "quality", Status: "resolved", Severity: "info", Source: "ui", CreatedAt: now, UpdatedAt: now},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	resp, err := svc.Evaluate("local")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range resp.Results {
		if item.Metric == "low_feedback_rate" && item.Status == "alert" {
			found = true
		}
	}
	if !found || len(resp.Events) == 0 {
		t.Fatalf("resp=%+v want low feedback alert", resp)
	}
	var count int64
	if err := db.Model(&store.AlertEvent{}).Where("status = ?", "active").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected persisted active alert")
	}
}

func TestRecordLowFeedbackAndPrometheusText(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	fb := store.Feedback{
		ID: "fb_record_low", SpaceID: "local", TargetType: "run", TargetID: "run_low",
		Rating: 1, Category: "quality", Status: "open", Severity: "warn", Source: "ui",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&fb).Error; err != nil {
		t.Fatal(err)
	}
	alert, err := svc.RecordLowFeedback(fb)
	if err != nil {
		t.Fatal(err)
	}
	if alert.ID == "" || alert.TargetID != fb.ID {
		t.Fatalf("alert=%+v want feedback alert", alert)
	}
	text, err := svc.PrometheusText()
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"run_total", "ci_diagnoses_total", "feedback_low_score_total 1", "alerts_active 1"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("prometheus text missing %q:\n%s", needle, text)
		}
	}
}

func TestTraceViewLinksRunRecords(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_trace", TraceID: "trace_123", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunEvent{ID: "evt_trace", RunID: run.ID, Seq: 1, TS: now.UnixMilli(), Type: "run.started", PayloadJSON: "{}", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.ToolCall{ID: "tool_trace", RunID: run.ID, TraceID: run.TraceID, Tool: "shell", Risk: "low", Status: "ok", OutputJSON: "{}", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AgentTask{ID: "agent_trace", RunID: run.ID, TraceID: run.TraceID, Adapter: "static", Status: "succeeded", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AuditLog{ID: "audit_trace", SpaceID: "local", TraceID: run.TraceID, RunID: run.ID, EventType: "trace.test", PayloadJSON: "{}", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	view, err := svc.Trace("local", run.TraceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Runs) != 1 || len(view.Events) != 1 || len(view.ToolCalls) != 1 || len(view.AgentTasks) != 1 || len(view.AuditLogs) != 1 {
		t.Fatalf("view=%+v want linked records", view)
	}
}
