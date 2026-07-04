package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestTR0Suite(t *testing.T) {
	svc := newTestDoctor(t)

	rep, err := svc.RunSuite("TR0")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Fail > 0 {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("TR0 failed: pass=%d fail=%d", rep.Summary.Pass, rep.Summary.Fail)
	}
	var replay CaseResult
	for _, r := range rep.Results {
		if r.ID == "TR0-03" {
			replay = r
			break
		}
	}
	if replay.ID == "" {
		t.Fatal("missing TR0-03 replay case")
	}
	foundReplayRun := false
	foundReplayArtifact := false
	for _, ev := range replay.Evidence {
		if ev.Kind == "run" && len(ev.Ref) > len("replay:") && ev.Ref[:len("replay:")] == "replay:" {
			foundReplayRun = true
		}
		if ev.Kind == "artifact" && len(ev.Ref) > len("replay:") && ev.Ref[:len("replay:")] == "replay:" && ev.Digest != "" {
			foundReplayArtifact = true
		}
	}
	if !foundReplayRun || !foundReplayArtifact {
		t.Fatalf("TR0-03 evidence=%+v want replay run and replay artifact digest", replay.Evidence)
	}
	var checkpoint CaseResult
	for _, r := range rep.Results {
		if r.ID == "TR0-07" {
			checkpoint = r
			break
		}
	}
	if checkpoint.ID == "" {
		t.Fatal("missing TR0-07 checkpoint case")
	}
	foundCheckpoint := false
	foundCheckpointEvent := false
	foundCheckpointAudit := false
	for _, ev := range checkpoint.Evidence {
		if ev.Kind == "checkpoint" && ev.Ref != "" && ev.Digest != "" {
			foundCheckpoint = true
		}
		if ev.Kind == "event" && (ev.Ref == "checkpoint.stored" || ev.Ref == "run.checkpoint_saved") {
			foundCheckpointEvent = true
		}
		if ev.Kind == "audit" && ev.Ref == "checkpoint.access_url_issued" {
			foundCheckpointAudit = true
		}
	}
	if !foundCheckpoint || !foundCheckpointEvent || !foundCheckpointAudit {
		t.Fatalf("TR0-07 evidence=%+v want checkpoint, event, and audit evidence", checkpoint.Evidence)
	}
}

func TestTR1Suite(t *testing.T) {
	svc := newTestDoctor(t)

	rep, err := svc.RunSuite("TR1")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Fail > 0 {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("TR1 failed: pass=%d fail=%d", rep.Summary.Pass, rep.Summary.Fail)
	}
	if rep.Summary.Pass != 5 {
		t.Fatalf("TR1 pass=%d want 5", rep.Summary.Pass)
	}
	assertCaseEvidence(t, rep, "TR1-01", "modelRouter")
	assertCaseEvidence(t, rep, "TR1-02", "waterfallSpan")
	assertCaseEvidence(t, rep, "TR1-03", "memoryEdge")
	assertCaseEvidence(t, rep, "TR1-04", "mcpIsolation")
	assertCaseEvidence(t, rep, "TR1-05", "rulesValidation")
}

func TestTR2Suite(t *testing.T) {
	svc := newTestDoctor(t)

	rep, err := svc.RunSuite("TR2")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Fail > 0 {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("TR2 failed: pass=%d fail=%d", rep.Summary.Pass, rep.Summary.Fail)
	}
	if rep.Summary.Pass != 5 {
		t.Fatalf("TR2 pass=%d want 5", rep.Summary.Pass)
	}
	assertCaseEvidence(t, rep, "TR2-01", "identityModel")
	assertCaseEvidence(t, rep, "TR2-02", "runScope")
	assertCaseEvidence(t, rep, "TR2-03", "storageProfile")
	assertCaseEvidence(t, rep, "TR2-04", "pluginABI")
	assertCaseEvidence(t, rep, "TR2-05", "secretScan")
}

func TestTR3Suite(t *testing.T) {
	svc := newTestDoctor(t)

	rep, err := svc.RunSuite("TR3")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Fail > 0 {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("TR3 failed: pass=%d fail=%d", rep.Summary.Pass, rep.Summary.Fail)
	}
	if rep.Summary.Pass != 7 {
		t.Fatalf("TR3 pass=%d want 7", rep.Summary.Pass)
	}
	assertCaseEvidence(t, rep, "TR3-01", "memorySchema")
	assertCaseEvidence(t, rep, "TR3-02", "ragFallback")
	assertCaseEvidence(t, rep, "TR3-06", "skipped")
	assertCaseEvidence(t, rep, "TR3-03", "sloMetric")
	assertCaseEvidence(t, rep, "TR3-04", "trace")
	assertCaseEvidence(t, rep, "TR3-05", "metricsParity")
	assertCaseEvidence(t, rep, "TR3-07", "pluginExport")
}

func TestM2Suite(t *testing.T) {
	svc := newTestDoctor(t)
	rep, err := svc.RunSuite("M2")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Pass != 3 {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("M2 pass=%d fail=%d want pass=3", rep.Summary.Pass, rep.Summary.Fail)
	}
	assertCaseEvidence(t, rep, "M2-01", "scenarioMatrix")
	assertCaseEvidence(t, rep, "M2-02", "scenarioPolicy")
	assertCaseEvidence(t, rep, "M2-03", "policyDenied")
}

func TestM3SuiteRunsMigrateVerifyFirstWhenE2E(t *testing.T) {
	t.Setenv("ASH_MIGRATE_E2E", "1")
	svc := newTestDoctor(t)
	rep, err := svc.RunSuite("M3")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) == 0 || rep.Results[0].ID != "M3-04" {
		t.Fatalf("first M3 case=%v want M3-04 first when ASH_MIGRATE_E2E=1", firstCaseID(rep))
	}
}

func TestM3Suite(t *testing.T) {
	svc := newTestDoctor(t)
	rep, err := svc.RunSuite("M3")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Pass != 9 {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("M3 pass=%d fail=%d want pass=9", rep.Summary.Pass, rep.Summary.Fail)
	}
	assertCaseEvidence(t, rep, "M3-01", "tenantScope")
	assertCaseEvidence(t, rep, "M3-02", "databaseDialect")
	assertCaseEvidence(t, rep, "M3-03", "migrationCatalog")
	assertCaseEvidence(t, rep, "M3-04", "skipped")
	assertCaseEvidence(t, rep, "M3-06", "skipped")
	assertCaseEvidence(t, rep, "M3-07", "skipped")
	assertCaseEvidence(t, rep, "M3-05", "skipped")
	assertCaseEvidence(t, rep, "M3-08", "skipped")
	assertCaseEvidence(t, rep, "M3-09", "readyzDialect")
}

func TestALLSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("ALL suite is slow; run without -short")
	}
	svc := newTestDoctor(t)

	rep, err := svc.RunSuite("ALL")
	if err != nil {
		t.Fatal(err)
	}
	want := 37
	if rep.Summary.Pass != want {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("ALL pass=%d fail=%d want pass=%d", rep.Summary.Pass, rep.Summary.Fail, want)
	}
	if len(rep.Results) != want {
		t.Fatalf("results=%d want %d", len(rep.Results), want)
	}
}

func newTestDoctor(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	db := store.OpenTest(t, dir)
	scenariosDir := filepath.Join("..", "..", "scenarios")
	if _, err := os.Stat(scenariosDir); err != nil {
		scenariosDir = filepath.Join("scenarios")
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})
	return NewService(runsSvc, ev, loader, dir)
}

func assertCaseEvidence(t *testing.T, rep *Report, caseID, kind string) {
	t.Helper()
	for _, result := range rep.Results {
		if result.ID != caseID {
			continue
		}
		for _, evidence := range result.Evidence {
			if evidence.Kind == kind {
				return
			}
		}
		t.Fatalf("%s evidence=%+v missing kind %q", caseID, result.Evidence, kind)
	}
	t.Fatalf("missing case %s", caseID)
}

func TestArtifactQualityEvidenceRejectsPlaceholdersInStrictMode(t *testing.T) {
	runDir := t.TempDir()
	artDir := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "diff.patch"), []byte("# No working tree diff was produced.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "test_report.json"), []byte(`{"ok":false,"error":"test.run did not produce a report"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &artifacts.Manifest{Artifacts: []artifacts.Entry{
		{Type: "diff", Name: "diff.patch", URI: "artifacts/diff.patch", Digest: "sha256:diff"},
		{Type: "test_report", Name: "test_report.json", URI: "artifacts/test_report.json", Digest: "sha256:test"},
	}}

	_, msg := artifactQualityEvidence(runDir, manifest, true)
	if msg == "" || msg != "diff.patch is placeholder; real delivery did not produce a working tree diff" {
		t.Fatalf("msg=%q want placeholder diff rejection", msg)
	}

	if err := os.WriteFile(filepath.Join(artDir, "diff.patch"), []byte("diff --git a/README.md b/README.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, msg = artifactQualityEvidence(runDir, manifest, true)
	if msg == "" || msg != "test_report.json is placeholder; test.run did not produce a real report" {
		t.Fatalf("msg=%q want placeholder test report rejection", msg)
	}
}

func firstCaseID(rep *Report) string {
	if rep == nil || len(rep.Results) == 0 {
		return ""
	}
	return rep.Results[0].ID
}

func TestArtifactQualityEvidenceAllowsStaticPlaceholders(t *testing.T) {
	runDir := t.TempDir()
	artDir := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "diff.patch"), []byte("# Static executor produced no code changes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "test_report.json"), []byte(`{"ok":false,"error":"test.run did not produce a report"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &artifacts.Manifest{Artifacts: []artifacts.Entry{
		{Type: "diff", Name: "diff.patch", URI: "artifacts/diff.patch", Digest: "sha256:diff"},
		{Type: "test_report", Name: "test_report.json", URI: "artifacts/test_report.json", Digest: "sha256:test"},
	}}

	evidence, msg := artifactQualityEvidence(runDir, manifest, false)
	if msg != "" {
		t.Fatalf("msg=%q want static placeholders allowed", msg)
	}
	if len(evidence) != 2 {
		t.Fatalf("evidence=%+v want diff and test_report quality evidence", evidence)
	}
}
