package lenses

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestLensGlobalAndAgentAndMemoryLineage(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_lens_1", TraceID: "trace_lens_1", SpaceID: "local",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		Status: "running", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	mem := store.MemoryRecord{
		ID: "mem_lens_1", SpaceID: "local", Layer: "L1", Status: "candidate",
		Title: "secret body hint", Body: "SECRET_BODY", TagsJSON: "[]",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&mem).Error; err != nil {
		t.Fatal(err)
	}
	events := []store.RunEvent{
		{ID: "ev1", RunID: run.ID, Seq: 1, Type: "context.packed", TS: now.UnixMilli(), PayloadJSON: `{"prefixBytes":12}`, CreatedAt: now},
		{ID: "ev2", RunID: run.ID, Seq: 2, Type: "template.finished", TS: now.UnixMilli() + 1, PayloadJSON: `{"templateId":"tpl.react"}`, CreatedAt: now},
		{ID: "ev3", RunID: run.ID, Seq: 3, Type: "memory.injected", TS: now.UnixMilli() + 2, PayloadJSON: `{"memoryId":"mem_lens_1","body":"SECRET"}`, CreatedAt: now},
		{ID: "ev4", RunID: run.ID, Seq: 4, Type: "noise.event", TS: now.UnixMilli() + 3, PayloadJSON: `{}`, CreatedAt: now},
	}
	for _, ev := range events {
		if err := db.Create(&ev).Error; err != nil {
			t.Fatal(err)
		}
	}

	global, err := BuildGlobal(db, "local")
	if err != nil {
		t.Fatal(err)
	}
	if global.RunningRuns != 1 || global.ReviewPending != 1 {
		t.Fatalf("global=%+v", global)
	}
	if len(global.TemplateUsage) != 1 || global.TemplateUsage[0].ID != "tpl.react" {
		t.Fatalf("templates=%+v", global.TemplateUsage)
	}

	agent, err := BuildAgent(db, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if agent.RunID != run.ID || agent.Waterfall == nil {
		t.Fatalf("agent=%+v", agent)
	}
	for _, ev := range agent.Events {
		if ev.Type == "noise.event" {
			t.Fatal("noise must not enter agent lens")
		}
		if !Admit(ev.Type, LensAgent) {
			t.Fatalf("unadmitted %s", ev.Type)
		}
	}

	lineage, err := BuildMemoryLineage(db, mem.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !lineage.Redacted || lineage.MemoryID != mem.ID {
		t.Fatalf("lineage=%+v", lineage)
	}
	foundInject := false
	for _, st := range lineage.Stages {
		if st.Stage == "injected" {
			foundInject = true
		}
	}
	if !foundInject {
		t.Fatalf("stages=%+v", lineage.Stages)
	}
	for _, ev := range lineage.Events {
		if body, ok := ev.Payload["body"]; ok && body != "[redacted]" {
			t.Fatalf("body not redacted: %v", body)
		}
	}
}

func TestObserveDeepLinkIdsStable(t *testing.T) {
	if DeepLinkAgent("run_x") != "/ui/observe?lens=agent&run=run_x" {
		t.Fatal(DeepLinkAgent("run_x"))
	}
	if DeepLinkMemory("mem_y") != "/ui/observe?lens=memory&id=mem_y" {
		t.Fatal(DeepLinkMemory("mem_y"))
	}
	if DeepLinkGlobal() != "/ui/observe?lens=global" {
		t.Fatal(DeepLinkGlobal())
	}
}
