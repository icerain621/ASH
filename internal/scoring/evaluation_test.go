package scoring_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/scoring"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestEvaluate_dimensionsAndScenarios(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := scoring.NewService(db).WithProfiles(scoring.DefaultProfiles())
	now := time.Now().UTC()
	start := now.Add(-2 * time.Hour)
	done := now.Add(-time.Hour)

	runs := []store.RunRecord{
		{
			ID: "run_fd_1", TraceID: "tr1", SpaceID: "local",
			ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			Status: "finished", StartedAt: start, FinishedAt: &done, CreatedAt: start, UpdatedAt: done,
		},
		{
			ID: "run_fd_2", TraceID: "tr2", SpaceID: "local",
			ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			Status: "failed", StartedAt: start, FinishedAt: &done, CreatedAt: start, UpdatedAt: done,
		},
		{
			ID: "run_hf_1", TraceID: "tr3", SpaceID: "local",
			ScenarioName: "hotfix", ScenarioVersion: "1.0.0",
			Status: "finished", StartedAt: start, FinishedAt: &done, CreatedAt: start, UpdatedAt: done,
		},
	}
	for _, r := range runs {
		if err := db.Create(&r).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, ev := range []store.RunEvent{
		{ID: "e1", RunID: "run_fd_1", Seq: 1, TS: now.UnixMilli(), Type: "memory.hit_used", Severity: "info",
			PayloadJSON: `{"recordIds":["m1"],"count":1}`, CreatedAt: now},
		{ID: "e2", RunID: "run_fd_1", Seq: 2, TS: now.UnixMilli(), Type: "memory.query", Severity: "info",
			PayloadJSON: `{"layersKey":"L1"}`, CreatedAt: now},
		{ID: "e3", RunID: "run_fd_2", Seq: 1, TS: now.UnixMilli(), Type: "citation.missing", Severity: "warn",
			PayloadJSON: `{"reason":"none"}`, CreatedAt: now},
		{ID: "e4", RunID: "run_hf_1", Seq: 1, TS: now.UnixMilli(), Type: "interaction.thread_sealed", Severity: "info",
			PayloadJSON: `{"threadId":"th_hf"}`, CreatedAt: now},
	} {
		if err := db.Create(&ev).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&store.InteractionThread{
		ID: "th_fd", SpaceID: "local", RunID: "run_fd_1", Kind: "main", Status: "sealed",
		Digest: "thd_x", HeadSeq: 2, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.InteractionThread{
		ID: "th_fd2", SpaceID: "local", RunID: "run_fd_2", Kind: "main", Status: "open",
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.InteractionThread{
		ID: "th_hf", SpaceID: "local", RunID: "run_hf_1", Kind: "main", Status: "sealed",
		Digest: "thd_y", HeadSeq: 1, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	out, err := svc.Evaluate(scoring.EvaluationRequest{
		SpaceID: "local", From: now.Add(-24 * time.Hour), To: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Dimensions) != 4 {
		t.Fatalf("dimensions=%d", len(out.Dimensions))
	}
	byID := map[string]scoring.DimensionScore{}
	for _, d := range out.Dimensions {
		byID[d.ID] = d
	}
	if byID["quality"].Score <= 0 || byID["quality"].Score > 1 {
		t.Fatalf("quality=%v", byID["quality"])
	}
	if out.Health.ThreadSealRate.Numerator != 2 || out.Health.ThreadSealRate.Denominator != 3 {
		t.Fatalf("seal=%+v", out.Health.ThreadSealRate)
	}
	if out.Health.CitationMissingTotal != 1 {
		t.Fatalf("citation=%d", out.Health.CitationMissingTotal)
	}
	if out.Health.MemoryLinksByType["hit_used"] != 1 {
		t.Fatalf("links=%v", out.Health.MemoryLinksByType)
	}
	if len(out.Scenarios) < 2 {
		t.Fatalf("scenarios=%+v", out.Scenarios)
	}
	// Ranked by weighted score; hotfix (all finished + sealed) should rank high.
	if out.Scenarios[0].RankScore < out.Scenarios[len(out.Scenarios)-1].RankScore {
		t.Fatalf("expected descending rank: %+v", out.Scenarios)
	}
}

func TestLoadProfiles_yaml(t *testing.T) {
	profiles, err := scoring.LoadProfiles("../../config/scenario-eval-profiles.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 3 {
		t.Fatalf("profiles=%d", len(profiles))
	}
	if profiles[0].Weights["quality"] <= 0 {
		t.Fatalf("weights=%v", profiles[0].Weights)
	}
}

func TestEvaluate_excludesVoidedScoreEvents(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := scoring.NewService(db)
	now := time.Now().UTC()

	keep, err := svc.RecordScore("local", "memory", "m_keep", "", scoring.ReviewRubric{
		Correctness: 5, Safety: 5, Citable: 5, Efficiency: 5,
	}, "r1", "keep")
	if err != nil {
		t.Fatal(err)
	}
	voidMe, err := svc.RecordScore("local", "memory", "m_void", "", scoring.ReviewRubric{
		Correctness: 1, Safety: 1, Citable: 1, Efficiency: 1,
	}, "r1", "void-me")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AuditLog{
		ID: "aud_void_test", SpaceID: "local", ActorID: "reviewer",
		EventType: "score.voided",
		PayloadJSON: `{"scoreEventId":"` + voidMe.ID + `","targetType":"memory","targetId":"m_void","reason":"bad"}`,
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if !scoring.IsVoided(db.DB, voidMe.ID) {
		t.Fatal("expected voided score to be IsVoided")
	}
	if scoring.IsVoided(db.DB, keep.ID) {
		t.Fatal("keep score must not be voided")
	}
	voidSet, err := scoring.VoidedIDs(db.DB, "local")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := voidSet[voidMe.ID]; !ok {
		t.Fatalf("VoidedIDs missing %s: %v", voidMe.ID, voidSet)
	}
	if _, ok := voidSet[keep.ID]; ok {
		t.Fatalf("VoidedIDs should not include keep: %v", voidSet)
	}

	out, err := svc.Evaluate(scoring.EvaluationRequest{
		SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	var rubricNote *scoring.DataQualityNote
	for i := range out.DataQuality {
		if out.DataQuality[i].MetricID == "rubric" {
			rubricNote = &out.DataQuality[i]
			break
		}
	}
	if rubricNote == nil || rubricNote.Status != "ok" {
		t.Fatalf("dataQuality=%+v", out.DataQuality)
	}
	if !strings.Contains(rubricNote.Message, "active=1") || !strings.Contains(rubricNote.Message, "voided_excluded=1") {
		t.Fatalf("expected void exclusion in message: %s", rubricNote.Message)
	}

	var avgSig *scoring.SignalMetric
	for _, d := range out.Dimensions {
		if d.ID != "quality" {
			continue
		}
		for i := range d.Signals {
			if d.Signals[i].ID == "rubric_composite_avg" {
				avgSig = &d.Signals[i]
			}
		}
	}
	if avgSig == nil {
		t.Fatalf("missing rubric_composite_avg signal: %+v", out.Dimensions)
	}
	if avgSig.Value != 5 || avgSig.Numerator != 1 || avgSig.Denominator != 2 {
		t.Fatalf("avg signal=%+v want composite 5 from keep-only (num=1 den=2)", avgSig)
	}
}

func TestEvaluate_allScoreEventsVoided(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := scoring.NewService(db)
	now := time.Now().UTC()
	ev, err := svc.RecordScore("local", "memory", "m1", "", scoring.ReviewRubric{
		Correctness: 4, Safety: 4, Citable: 4, Efficiency: 4,
	}, "r1", "soon-void")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AuditLog{
		ID: "aud_all_void", SpaceID: "local", ActorID: "reviewer",
		EventType:   "score.voided",
		PayloadJSON: `{"scoreEventId":"` + ev.ID + `"}`,
		CreatedAt:   now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	out, err := svc.Evaluate(scoring.EvaluationRequest{
		SpaceID: "local", From: now.Add(-time.Hour), To: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range out.DataQuality {
		if n.MetricID == "rubric" && n.Status == "empty" && strings.Contains(n.Message, "voided") {
			found = true
		}
	}
	if !found {
		t.Fatalf("dataQuality=%+v", out.DataQuality)
	}
	for _, d := range out.Dimensions {
		if d.ID != "quality" {
			continue
		}
		for _, sig := range d.Signals {
			if sig.ID == "rubric_composite_avg" {
				t.Fatalf("voided-only must not blend rubric signal: %+v", d.Signals)
			}
		}
	}
}

func TestEvaluate_emptyWindow(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := scoring.NewService(db)
	now := time.Now().UTC()
	out, err := svc.Evaluate(scoring.EvaluationRequest{
		SpaceID: "local", From: now.Add(-time.Hour), To: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Scenarios) != 0 {
		t.Fatalf("scenarios=%v", out.Scenarios)
	}
	found := false
	for _, n := range out.DataQuality {
		if n.MetricID == "runs" && n.Status == "empty" {
			found = true
		}
	}
	if !found {
		t.Fatalf("dataQuality=%v", out.DataQuality)
	}
}

