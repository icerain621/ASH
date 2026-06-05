package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/doctor"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestCLIArtifactResultsIncludeAccessURLsAndEvents(t *testing.T) {
	svc, created := createCLITestRun(t)
	assertCLIRunOutputRefs(t, svc, created.RunID)
}

func TestCLIReplayProducesArtifactsAndEvents(t *testing.T) {
	svc, created := createCLITestRun(t)
	replayed, err := svc.Replay(created.RunID, runs.ReplayRequest{Mode: "exact"})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.RunID == created.RunID {
		t.Fatal("replay should create a new run")
	}
	assertCLIRunOutputRefs(t, svc, replayed.RunID)
}

func TestCLIDoctorMarkdownIncludesEvidence(t *testing.T) {
	rep := sampleDoctorReport()
	md := formatReportMD(rep)
	for _, want := range []string{
		"# ASH Doctor Report",
		"TR0",
		"Pass: **1** | Fail: **1**",
		"TR0-01",
		"runId: `run_pass`",
		"evidence: artifact `diff`",
		"sha256:diff",
		"TR0-02",
		"message: missing events",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown report missing %q:\n%s", want, md)
		}
	}
}

func TestCLIDoctorJSONIncludesEvidence(t *testing.T) {
	payload, err := json.MarshalIndent(sampleDoctorReport(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var decoded doctor.Report
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Suite != "TR0" || decoded.Summary.Pass != 1 || decoded.Summary.Fail != 1 {
		t.Fatalf("unexpected summary: suite=%s pass=%d fail=%d", decoded.Suite, decoded.Summary.Pass, decoded.Summary.Fail)
	}
	if len(decoded.Results) != 2 {
		t.Fatalf("results=%d want 2", len(decoded.Results))
	}
	if decoded.Results[0].RunID != "run_pass" || len(decoded.Results[0].Evidence) != 1 {
		t.Fatalf("missing run/evidence in JSON: %+v", decoded.Results[0])
	}
	if decoded.Results[0].Evidence[0].Digest != "sha256:diff" {
		t.Fatalf("digest=%q want sha256:diff", decoded.Results[0].Evidence[0].Digest)
	}
}

func sampleDoctorReport() *doctor.Report {
	rep := &doctor.Report{
		Suite:      "TR0",
		StartedAt:  1000,
		FinishedAt: 2000,
		Results: []doctor.CaseResult{
			{
				ID:     "TR0-01",
				Status: "pass",
				RunID:  "run_pass",
				Evidence: []doctor.Evidence{
					{Kind: "artifact", Ref: "diff", Digest: "sha256:diff"},
				},
			},
			{
				ID:      "TR0-02",
				Status:  "fail",
				RunID:   "run_failed",
				Message: "missing events",
				Evidence: []doctor.Evidence{
					{Kind: "eventRange", Ref: "run_events:seq=1..2"},
				},
			},
		},
	}
	rep.Summary.Pass = 1
	rep.Summary.Fail = 1
	return rep
}

func createCLITestRun(t *testing.T) (*runs.Service, *runs.CreateResponse) {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("cli test issue\nFeature delivery evidence for CLI test.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	db := store.OpenTest(t, filepath.Join(dir, "data"))
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	svc := runs.NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "cli test issue",
			"repoRoot":    repo,
		},
		Repo: &runs.RepoRef{Root: repo},
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, created
}

func assertCLIRunOutputRefs(t *testing.T, svc *runs.Service, runID string) {
	t.Helper()
	manifest, err := svc.Artifacts(runID)
	if err != nil {
		t.Fatal(err)
	}
	items := artifactResults(svc, runID, manifest)
	types := map[string]int{}
	for _, item := range items {
		types[item.Type]++
		if item.AccessURL == "" {
			t.Fatalf("artifact %s missing access URL: %+v", item.Name, item)
		}
		if item.URI == "" || item.Digest == "" {
			t.Fatalf("artifact %s missing URI/digest: %+v", item.Name, item)
		}
	}
	for _, typ := range []string{"diff", "test_report", "release_notes", "rollback_plan"} {
		if types[typ] == 0 {
			t.Fatalf("missing required artifact type %q in %+v", typ, items)
		}
	}
	if types["step_output"] == 0 {
		t.Fatalf("missing step_output artifact in %+v", items)
	}
	checkpoints := checkpointResults(svc, runID)
	if len(checkpoints) == 0 {
		t.Fatal("expected CLI checkpoint output refs")
	}
	for _, checkpoint := range checkpoints {
		if checkpoint.ID == "" || checkpoint.URI == "" || checkpoint.SnapshotDigest == "" {
			t.Fatalf("checkpoint missing ID/URI/digest: %+v", checkpoint)
		}
		if checkpoint.AccessURL == "" {
			t.Fatalf("checkpoint %s missing access URL: %+v", checkpoint.ID, checkpoint)
		}
	}
	evs, err := svc.Events().ListAfter(runID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) == 0 {
		t.Fatal("expected run events for CLI progress output")
	}
	if evs[len(evs)-1].Type != "run.finished" {
		t.Fatalf("last event=%q want run.finished", evs[len(evs)-1].Type)
	}
}
