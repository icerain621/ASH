package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/review"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestCLIReviewEval(t *testing.T) {
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_cli_eval", SpaceID: "local", TraceID: "tr", Status: "succeeded",
		ScenarioName: "x", ScenarioVersion: "1", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunEvent{
		ID: "ev_cli_eval", RunID: run.ID, Seq: 1, TS: now.UnixMilli(),
		Type: "memory.feedback", PayloadJSON: `{"kind":"wrong_update","recordId":"m1"}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	got, err := review.LoadMemoryEvalCounts(db, "local", 100)
	if err != nil || got.WrongUpdate != 1 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestCLIDoctorQualityMigrated(t *testing.T) {
	msg, ok := doctorQualityMigrated("QUALITY")
	if !ok || !strings.Contains(msg, "ash review") {
		t.Fatalf("msg=%q ok=%v", msg, ok)
	}
	if _, ok := doctorQualityMigrated("BOOT"); ok {
		t.Fatal("BOOT must stay on doctor")
	}
}

func TestCLIReviewListAndPassK(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ASH_DATA_DIR", dir)
	t.Setenv("ASH_DATABASE_URL", "")
	db := store.OpenTest(t, dir)
	manifest := `{"id":"tpl.cli.review","version":"1.0.0","loop":"react","tools":["read"],"compaction":"threshold","sandbox":"workspace-write","maxTurns":2,"memory":["mem.retrieve","mem.inject"]}`
	if _, err := db.SaveAgentTemplateCandidate("local", "tpl.cli.review", "1.0.0", manifest, "tester"); err != nil {
		t.Fatal(err)
	}
	item, err := db.CreateReviewItem("local", "agent_template", "tpl.cli.review", "cli")
	if err != nil {
		t.Fatal(err)
	}
	md := formatReviewListMD([]store.ReviewItemRow{*item})
	if !strings.Contains(md, item.ID) || !strings.Contains(md, "tpl.cli.review") {
		t.Fatalf("md=%s", md)
	}

	agenttpl.ResetCatalogForTest()
	t.Cleanup(agenttpl.ResetCatalogForTest)
	agenttpl.ClearApprovedLoaderForTest()
	t.Cleanup(agenttpl.ClearApprovedLoaderForTest)
	installCLIApprovedLoader(db)
	if _, err := agenttpl.GetProduction("tpl.cli.review"); err == nil {
		t.Fatal("candidate must not be production")
	}
	if _, err := db.ApproveAgentTemplate("local", "tpl.cli.review", "tester"); err != nil {
		t.Fatal(err)
	}
	got, err := agenttpl.GetProduction("tpl.cli.review")
	if err != nil || got.ID != "tpl.cli.review" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
