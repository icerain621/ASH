package runs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func testRunsService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join("..", "..", "scenarios")
	if _, err := os.Stat(scenariosDir); err != nil {
		scenariosDir = "scenarios"
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	return NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{}), scenariosDir
}

func repoWithEvidence(t *testing.T, issue string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(issue+"\nASH feature_delivery citation evidence.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReplayExact(t *testing.T) {
	svc, _ := testRunsService(t)
	createReq := CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "replay test",
			"repoRoot":    repoWithEvidence(t, "replay test"),
		},
	}
	orig, err := svc.Create(createReq)
	if err != nil {
		t.Fatal(err)
	}

	replayed, err := svc.Replay(orig.RunID, ReplayRequest{Mode: "exact"})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.RunID == orig.RunID {
		t.Fatal("replay should create a new run id")
	}

	meta, err := loadRunMeta(svc.db.RunDir(replayed.RunID))
	if err != nil {
		t.Fatal(err)
	}
	if meta.SourceRunID != orig.RunID {
		t.Fatalf("sourceRunId=%q want %q", meta.SourceRunID, orig.RunID)
	}
	if meta.ReplayMode != "exact" {
		t.Fatalf("replayMode=%q want exact", meta.ReplayMode)
	}

	sum, err := svc.Get(replayed.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "finished" {
		t.Fatalf("status=%q want finished", sum.Status)
	}
}

func TestResumeFailedRun(t *testing.T) {
	svc, _ := testRunsService(t)
	createReq := CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "resume test",
			"repoRoot":    repoWithEvidence(t, "resume test"),
		},
	}
	created, err := svc.Create(createReq)
	if err != nil {
		t.Fatal(err)
	}

	var rec store.RunRecord
	if err := svc.db.First(&rec, "id = ?", created.RunID).Error; err != nil {
		t.Fatal(err)
	}
	rec.Status = "failed"
	rec.ErrorCode = "TEST_FAIL"
	rec.ErrorMessage = "simulated failure"
	if err := svc.db.Save(&rec).Error; err != nil {
		t.Fatal(err)
	}

	resp, err := svc.Resume(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if resp.RunID != created.RunID {
		t.Fatalf("resume runId=%q want same %q", resp.RunID, created.RunID)
	}
	if resp.Status != "finished" {
		t.Fatalf("status=%q want finished", resp.Status)
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	foundResumed := false
	for _, ev := range evs {
		if ev.Type == "run.resumed" {
			foundResumed = true
			break
		}
	}
	if !foundResumed {
		t.Fatal("expected run.resumed event")
	}
}

func TestResumeNotResumable(t *testing.T) {
	svc, _ := testRunsService(t)
	createReq := CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "not resumable",
			"repoRoot":    repoWithEvidence(t, "not resumable"),
		},
	}
	created, err := svc.Create(createReq)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Resume(created.RunID)
	if err == nil {
		t.Fatal("expected error for finished run")
	}
	if !errors.Is(err, ErrRunNotResumable) {
		t.Fatalf("err=%v want ErrRunNotResumable", err)
	}
}

func TestRunRecordsQualityMetrics(t *testing.T) {
	svc, _ := testRunsService(t)
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "quality metrics",
			"repoRoot":    repoWithEvidence(t, "quality metrics"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var rows []store.QualityMetric
	if err := svc.db.Where("run_id = ?", created.RunID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	metrics := map[string]store.QualityMetric{}
	for _, row := range rows {
		metrics[row.Name] = row
	}
	for _, name := range []string{
		"steps_total",
		"artifacts_total",
		"artifact_quality_passed",
		"artifact_quality_failed_total",
		"tool_calls_total",
		"tool_failure_rate",
		"agent_tasks_total",
		"agent_failure_rate",
		"model_cost_micros_total",
		"citation_bound_total",
		"citation_missing_total",
		"citation_hit_rate",
	} {
		if _, ok := metrics[name]; !ok {
			t.Fatalf("missing quality metric %q in %+v", name, metrics)
		}
	}
	if metrics["steps_total"].Value <= 0 {
		t.Fatalf("steps_total=%v want positive", metrics["steps_total"].Value)
	}
	if metrics["artifacts_total"].Value <= 0 {
		t.Fatalf("artifacts_total=%v want positive", metrics["artifacts_total"].Value)
	}
	if metrics["artifact_quality_passed"].Value != 1 || metrics["artifact_quality_failed_total"].Value != 0 {
		t.Fatalf("artifact quality metrics pass=%v fail=%v want 1/0", metrics["artifact_quality_passed"].Value, metrics["artifact_quality_failed_total"].Value)
	}
	if metrics["tool_calls_total"].Value <= 0 {
		t.Fatalf("tool_calls_total=%v want positive", metrics["tool_calls_total"].Value)
	}
	if metrics["agent_tasks_total"].Value <= 0 {
		t.Fatalf("agent_tasks_total=%v want positive", metrics["agent_tasks_total"].Value)
	}
	if metrics["citation_bound_total"].Value <= 0 {
		t.Fatalf("citation_bound_total=%v want positive", metrics["citation_bound_total"].Value)
	}
	if metrics["citation_hit_rate"].Value <= 0 {
		t.Fatalf("citation_hit_rate=%v want positive", metrics["citation_hit_rate"].Value)
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range evs {
		if ev.Type == "quality.metrics_recorded" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected quality.metrics_recorded event")
	}
}

func TestFeatureDeliveryManifestIncludesStepOutputs(t *testing.T) {
	svc, _ := testRunsService(t)
	issue := "step output manifest"
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": issue,
			"repoRoot":    repoWithEvidence(t, issue),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := svc.Artifacts(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"pm_clarify.md":     "PM",
		"arch_design.md":    "Architect",
		"review_quality.md": "Reviewer",
		"ship_release.md":   "Shipper",
	}
	for _, art := range manifest.Artifacts {
		if art.Type != "step_output" {
			continue
		}
		role, ok := want[art.Name]
		if !ok {
			continue
		}
		if art.Producer["role"] != role || art.Producer["stepId"] == "" {
			t.Fatalf("step output %s producer=%+v want role=%s and stepId", art.Name, art.Producer, role)
		}
		delete(want, art.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing step output artifacts: %+v", want)
	}
}

func TestRunInjectsApprovedMemoryAndRecordsHitUsed(t *testing.T) {
	svc, _ := testRunsService(t)
	issue := "memory injection"
	repo := repoWithEvidence(t, issue)
	now := time.Now().UTC()
	mem := store.MemoryRecord{
		ID: "mem_run_inject", Layer: "L1", Status: "approved", SpaceID: "local",
		SchemaVersion: 1, Title: issue, Body: "Use memory injection evidence during delivery.",
		ScopeRepo: repo, Sensitivity: "normal", Confidence: 0.91,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := svc.db.Create(&mem).Error; err != nil {
		t.Fatal(err)
	}

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": issue,
			"repoRoot":    repo,
		},
		Repo: &RepoRef{Root: repo},
	})
	if err != nil {
		t.Fatal(err)
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	foundInjected := false
	foundHitUsed := false
	for _, ev := range evs {
		if ev.Type == "memory.injected" && strings.Contains(string(ev.Payload), mem.ID) {
			foundInjected = true
		}
		if ev.Type == "memory.hit_used" && strings.Contains(string(ev.Payload), mem.ID) {
			foundHitUsed = true
		}
	}
	if !foundInjected {
		t.Fatal("expected memory.injected event with memory id")
	}
	if !foundHitUsed {
		t.Fatal("expected memory.hit_used event with memory id")
	}

	var audit store.AuditLog
	if err := svc.db.Where("run_id = ? AND event_type = ?", created.RunID, "memory.hit_used").First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if audit.SpaceID != "local" || !strings.Contains(audit.PayloadJSON, mem.ID) {
		t.Fatalf("unexpected memory hit audit: %+v", audit)
	}

	manifest, err := svc.Artifacts(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	foundEvidenceRef := false
	for _, artifact := range manifest.Artifacts {
		refs, _ := artifact.Producer["evidenceRefs"].([]any)
		for _, ref := range refs {
			if ref == "memory:"+mem.ID {
				foundEvidenceRef = true
			}
		}
	}
	if !foundEvidenceRef {
		t.Fatal("expected manifest producer evidenceRefs to include memory ref")
	}
}

func TestRunIndexesAndQueriesRAGInRunSpace(t *testing.T) {
	svc, _ := testRunsService(t)
	issue := "space scoped rag evidence"
	repo := repoWithEvidence(t, issue)
	spaceID := "space_rag_run_scope"

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": issue,
			"repoRoot":    repo,
		},
		SpaceID: spaceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.SpaceID != spaceID {
		t.Fatalf("spaceId=%q want %q", sum.SpaceID, spaceID)
	}

	inRunSpace, err := svc.rag.Query(ragQueryRequest(spaceID, repo, issue))
	if err != nil {
		t.Fatal(err)
	}
	if len(inRunSpace.Items) == 0 {
		t.Fatal("expected RAG hits in run space")
	}
	inLocal, err := svc.rag.Query(ragQueryRequest("local", repo, issue))
	if err != nil {
		t.Fatal(err)
	}
	if len(inLocal.Items) != 0 {
		t.Fatalf("local RAG hits=%+v want none", inLocal.Items)
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	foundCitation := false
	for _, ev := range evs {
		if ev.Type == "citation.bound" && strings.Contains(string(ev.Payload), "README.md") {
			foundCitation = true
		}
	}
	if !foundCitation {
		t.Fatal("expected citation.bound event from run-space RAG")
	}
}

func TestRunStoresCheckpointsInArtifactStore(t *testing.T) {
	svc, _ := testRunsService(t)
	issue := "checkpoint object store"
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": issue,
			"repoRoot":    repoWithEvidence(t, issue),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var checkpoints []store.Checkpoint
	if err := svc.db.Where("run_id = ?", created.RunID).Order("created_at asc").Find(&checkpoints).Error; err != nil {
		t.Fatal(err)
	}
	if len(checkpoints) == 0 {
		t.Fatal("expected checkpoints")
	}
	listed, err := svc.Checkpoints(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != len(checkpoints) || listed[0].ID != checkpoints[0].ID {
		t.Fatalf("listed checkpoints=%+v want persisted checkpoints", listed)
	}
	for _, ckpt := range checkpoints {
		if !strings.HasPrefix(ckpt.URI, "fs://") {
			t.Fatalf("checkpoint URI=%q want fs:// object URI", ckpt.URI)
		}
		if ckpt.StoreKey == "" || ckpt.ContentType != "application/json" || ckpt.SizeBytes <= 0 {
			t.Fatalf("checkpoint object metadata missing: %+v", ckpt)
		}
		if _, err := os.Stat(strings.TrimPrefix(ckpt.URI, "fs://")); err != nil {
			t.Fatalf("checkpoint object missing for %s: %v", ckpt.URI, err)
		}
	}
	access, err := svc.CheckpointAccess(created.RunID, checkpoints[0].ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if access.CheckpointID != checkpoints[0].ID || access.SnapshotDigest != checkpoints[0].SnapshotDigest {
		t.Fatalf("access=%+v want checkpoint metadata", access)
	}
	if !strings.HasPrefix(access.SignedURL, "fs://") || access.ExpiresAt <= time.Now().UTC().UnixMilli() {
		t.Fatalf("access=%+v want signed fs URL with future expiry", access)
	}
	var audit store.AuditLog
	if err := svc.db.Where("run_id = ? AND event_type = ?", created.RunID, "checkpoint.access_url_issued").First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audit.PayloadJSON, checkpoints[0].ID) || !strings.Contains(audit.PayloadJSON, checkpoints[0].SnapshotDigest) {
		t.Fatalf("audit=%+v missing checkpoint metadata", audit)
	}
	evs, err := svc.events.ListAfter(created.RunID, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	foundStored := false
	foundSavedURI := false
	for _, ev := range evs {
		if ev.Type == "checkpoint.stored" {
			foundStored = true
		}
		if ev.Type == "run.checkpoint_saved" && strings.Contains(string(ev.Payload), `"uri":"fs://`) {
			foundSavedURI = true
		}
	}
	if !foundStored {
		t.Fatal("expected checkpoint.stored event")
	}
	if !foundSavedURI {
		t.Fatal("expected run.checkpoint_saved event with fs URI")
	}
}

func TestRunArtifactManifestIndexAndAccessUseObjectStoreKey(t *testing.T) {
	svc, _ := testRunsService(t)
	svc.WithArtifactStore(artifactstore.NewFSStore(filepath.Join(t.TempDir(), "objects")))
	issue := "artifact object store"
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": issue,
			"repoRoot":    repoWithEvidence(t, issue),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := svc.Artifacts(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Artifacts) == 0 {
		t.Fatal("expected manifest artifacts")
	}
	for _, art := range manifest.Artifacts {
		if !strings.HasPrefix(art.URI, "fs://") {
			t.Fatalf("artifact %s URI=%q want fs:// object URI", art.Name, art.URI)
		}
		if art.StoreKey == "" {
			t.Fatalf("artifact %s missing storeKey in manifest", art.Name)
		}
		var row store.ArtifactIndex
		if err := svc.db.Where("run_id = ? AND name = ?", created.RunID, art.Name).First(&row).Error; err != nil {
			t.Fatalf("artifact_index missing for %s: %v", art.Name, err)
		}
		if row.URI != art.URI || row.StoreKey != art.StoreKey || row.Digest != art.Digest {
			t.Fatalf("artifact_index mismatch for %s: row=%+v manifest=%+v", art.Name, row, art)
		}
		if _, err := os.Stat(strings.TrimPrefix(art.URI, "fs://")); err != nil {
			t.Fatalf("artifact object missing for %s: %v", art.Name, err)
		}
	}
	access, err := svc.ArtifactAccess(created.RunID, "release_notes.md", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(access.SignedURL, "fs://") {
		t.Fatalf("signed URL=%q want fs://", access.SignedURL)
	}
	if access.Digest == "" || access.URI == "" || access.SizeBytes <= 0 {
		t.Fatalf("artifact access metadata incomplete: %+v", access)
	}
	evs, err := svc.events.ListAfter(created.RunID, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	stored := 0
	for _, ev := range evs {
		if ev.Type == "artifact.stored" && strings.Contains(string(ev.Payload), `"storeKey":"runs/`) && strings.Contains(string(ev.Payload), `"uri":"fs://`) {
			stored++
		}
	}
	if stored == 0 {
		t.Fatal("expected artifact.stored events with object store metadata")
	}
}

func TestToolChainRetriesTransientFailure(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "tool_retry"
  scenarioVersion: "1.0.0"
  roles:
    QA: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "qa.retry"
      role: "QA"
      kind: "tool_chain"
      chain:
        - tool: "flaky.tool"
          timeoutMs: 30000
          retry: { maxAttempts: 2, backoffMs: 1 }
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "tool_retry.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	attempts := 0
	reg := toolbus.NewRegistry()
	reg.Register("flaky.tool", toolbus.RiskSafe, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		attempts++
		if attempts == 1 {
			return nil, fmt.Errorf("transient failure")
		}
		return map[string]any{"ok": true, "attempts": attempts}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "tool_retry", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "retry transient tool",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts=%d want 2", attempts)
	}

	calls, err := svc.ToolCalls(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("tool calls=%d want 2: %+v", len(calls), calls)
	}
	if calls[0].Attempt != 1 || calls[0].Status != "failed" {
		t.Fatalf("first call = attempt %d status %s", calls[0].Attempt, calls[0].Status)
	}
	if calls[1].Attempt != 2 || calls[1].Status != "success" {
		t.Fatalf("second call = attempt %d status %s", calls[1].Attempt, calls[1].Status)
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundRetry := false
	foundSecondResult := false
	for _, ev := range evs {
		if ev.Type == "tool.retry_scheduled" {
			foundRetry = true
		}
		if ev.Type == "tool.result" && strings.Contains(string(ev.Payload), `"attempt":2`) {
			foundSecondResult = true
		}
	}
	if !foundRetry {
		t.Fatal("expected tool.retry_scheduled event")
	}
	if !foundSecondResult {
		t.Fatal("expected second tool.result event")
	}
}

func TestDangerousToolRequiresApprovalBeforeExecution(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "danger_tool"
  scenarioVersion: "1.0.0"
  roles:
    Admin: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "ops.danger"
      role: "Admin"
      kind: "tool_chain"
      chain:
        - tool: "danger.tool"
          timeoutMs: 30000
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "danger_tool.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	calls := 0
	reg := toolbus.NewRegistry()
	reg.Register("danger.tool", toolbus.RiskDanger, func(_ toolbus.Context, _ map[string]any) (map[string]any, error) {
		calls++
		return map[string]any{"ok": true}, nil
	})
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.NewBus(reg)).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "danger_tool", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "needs dangerous tool"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("danger tool executed before approval: calls=%d", calls)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "waiting_approval" {
		t.Fatalf("status=%q want waiting_approval", sum.Status)
	}
	var steps []store.RunStep
	if err := svc.db.Where("run_id = ?", created.RunID).Order("created_at asc").Find(&steps).Error; err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 || steps[0].ErrorCode != "TOOL_DANGEROUS_APPROVAL_REQUIRED" {
		t.Fatalf("steps=%+v want dangerous approval error", steps)
	}
	var approvals []store.ApprovalRequest
	if err := svc.db.Where("run_id = ?", created.RunID).Find(&approvals).Error; err != nil {
		t.Fatal(err)
	}
	if len(approvals) != 1 {
		t.Fatalf("approvals=%d want 1", len(approvals))
	}
	if approvals[0].Status != "pending" || approvals[0].Gate != "tool_risk" || approvals[0].StepID != "ops.danger" {
		t.Fatalf("approval=%+v want pending tool_risk ops.danger", approvals[0])
	}

	if _, err := svc.Approve(created.RunID, ApproveRequest{ActorID: "tester", Reason: "allow scoped dangerous tool"}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("danger tool calls=%d want 1 after approval", calls)
	}
	sum, err = svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "finished" {
		t.Fatalf("status=%q want finished", sum.Status)
	}
	meta, err := loadRunMeta(svc.db.RunDir(created.RunID))
	if err != nil {
		t.Fatal(err)
	}
	if !approvedStep(meta.Inputs, "_approvedDangerousToolSteps", "ops.danger") {
		t.Fatal("expected dangerous tool approval to be persisted")
	}
	if err := svc.db.First(&approvals[0], "id = ?", approvals[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if approvals[0].Status != "approved" || approvals[0].DecidedBy != "tester" {
		t.Fatalf("approval=%+v want approved by tester", approvals[0])
	}
	evs, err := svc.events.ListAfter(created.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	foundRequired := false
	foundUsed := false
	foundToolApproval := false
	for _, ev := range evs {
		if ev.Type == "policy.denied" && strings.Contains(string(ev.Payload), "require_approval") {
			foundRequired = true
		}
		if ev.Type == "tool.approval_used" {
			foundUsed = true
		}
		if ev.Type == "gate.approved" && strings.Contains(string(ev.Payload), `"kind":"tool"`) {
			foundToolApproval = true
		}
	}
	if !foundRequired {
		t.Fatal("expected policy.denied require_approval event")
	}
	if !foundUsed {
		t.Fatal("expected tool.approval_used event")
	}
	if !foundToolApproval {
		t.Fatal("expected gate.approved kind=tool event")
	}
}

func TestRuntimeCommandRequiresApprovalThenExecutesViaExecGo(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "runtime_tool"
  scenarioVersion: "1.0.0"
  roles:
    Ops: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "ops.runtime"
      role: "Ops"
      kind: "tool_chain"
      chain:
        - tool: "runtime.command"
          args:
            program: "/bin/echo"
            args: ["hello"]
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "runtime_tool.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	cli := writeFakeRunsExecGoCLI(t, `#!/bin/sh
case "$1" in
  health) echo '{"ok":true,"data":{"status":"ok"}}' ;;
  tools) echo '{"ok":true,"data":{"schema_version":"adapter.v1","tools":["runtime.command"]}}' ;;
  act) echo '{"ok":true,"data":{"task_id":"task_runtime_runs"}}' ;;
  wait) echo '{"ok":true,"data":{"tasks":[{"status":"success"}]}}' ;;
  *) echo '{"ok":false,"error":{"message":"unexpected","status_code":400,"body":"bad"}}' ;;
esac
`)
	t.Setenv("EXECGO_EXECGOCLI", cli)
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "runtime_tool", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "runtime command gate"},
	})
	if err != nil {
		t.Fatal(err)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "waiting_approval" {
		t.Fatalf("status=%q want waiting_approval", sum.Status)
	}
	calls, err := svc.ToolCalls(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("runtime.command executed before approval: %+v", calls)
	}

	if _, err := svc.Approve(created.RunID, ApproveRequest{ActorID: "tester", Reason: "allow runtime command"}); err != nil {
		t.Fatal(err)
	}
	calls, err = svc.ToolCalls(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].Tool != "runtime.command" || calls[0].Status != "success" {
		t.Fatalf("tool calls=%+v want one successful runtime.command", calls)
	}
	if !strings.Contains(calls[0].OutputJSON, "task_runtime_runs") {
		t.Fatalf("output=%s missing execgo task id", calls[0].OutputJSON)
	}
}

func TestRunFailsNonStaticPlaceholderArtifacts(t *testing.T) {
	svc, _ := testRunsService(t)
	svc.WithAgentExecutor(nonStaticPlaceholderExecutor{})
	_, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "real artifact quality gate",
			"repoRoot":    repoWithEvidence(t, "real artifact quality gate"),
		},
	})
	if err == nil {
		t.Fatal("expected artifact quality failure")
	}
	if !strings.Contains(err.Error(), "ARTIFACT_QUALITY_FAILED") {
		t.Fatalf("err=%v want ARTIFACT_QUALITY_FAILED", err)
	}

	var rec store.RunRecord
	if err := svc.db.Order("created_at desc").First(&rec).Error; err != nil {
		t.Fatal(err)
	}
	if rec.Status != "failed" || rec.ErrorCode != "ARTIFACT_QUALITY_FAILED" {
		t.Fatalf("run=%+v want failed ARTIFACT_QUALITY_FAILED", rec)
	}
	var metrics []store.QualityMetric
	if err := svc.db.Where("run_id = ?", rec.ID).Find(&metrics).Error; err != nil {
		t.Fatal(err)
	}
	byName := map[string]store.QualityMetric{}
	for _, metric := range metrics {
		byName[metric.Name] = metric
	}
	if byName["artifact_quality_passed"].Value != 0 || byName["artifact_quality_failed_total"].Value != 1 {
		t.Fatalf("metrics=%+v want artifact quality failed markers", byName)
	}
	evs, err := svc.events.ListAfter(rec.ID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range evs {
		if ev.Type == "artifact.quality_failed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("events=%+v missing artifact.quality_failed", evs)
	}
}

type nonStaticPlaceholderExecutor struct {
	agentexec.StaticExecutor
}

func (nonStaticPlaceholderExecutor) AdapterName() string {
	return "execgo_codex"
}

func TestMCPFailureClassIsPreservedInToolResultEvent(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "mcp_policy"
  scenarioVersion: "1.0.0"
  roles:
    Ops: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec]
  steps:
    - id: "ops.mcp"
      role: "Ops"
      kind: "tool_chain"
      chain:
        - tool: "mcp.call"
          args:
            serverURL: "http://127.0.0.1:1"
            tool: "demo.echo"
            timeoutMs: 120001
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "mcp_policy.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})

	_, err = svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "mcp_policy", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "mcp policy should fail isolated"},
	})
	if err == nil {
		t.Fatal("expected mcp policy failure")
	}
	if !strings.Contains(err.Error(), "TOOL_FAILED") {
		t.Fatalf("err=%v want TOOL_FAILED", err)
	}

	var rec store.RunRecord
	if err := svc.db.First(&rec, "scenario_name = ?", "mcp_policy").Error; err != nil {
		t.Fatal(err)
	}
	evs, err := svc.events.ListAfter(rec.ID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range evs {
		if ev.Type == "tool.result" && strings.Contains(string(ev.Payload), `"failureClass":"mcp_policy"`) {
			found = true
		}
	}
	if !found {
		t.Fatal("expected tool.result failureClass=mcp_policy")
	}
}

func TestRequireCitationsBlocksWithoutEvidence(t *testing.T) {
	svc, _ := testRunsService(t)
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("unrelated content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "citation gate should block",
			"repoRoot":    repo,
		},
	})
	if err == nil {
		t.Fatal("expected citation gate error")
	}
	if !strings.Contains(err.Error(), "GATE_CITATION_MISSING") {
		t.Fatalf("err=%v want GATE_CITATION_MISSING", err)
	}
}

func TestRequireCitationsCanDegradeToHumanConfirm(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	scenario := `version: "ash.rules/v0.1"
scenario:
  name: "citation_human"
  scenarioVersion: "1.0.0"
  roles:
    Architect: { maxParallel: 1 }
  inputs:
    required: [issueOrSpec, repoRoot]
  steps:
    - id: "arch.design"
      role: "Architect"
      kind: "llm"
      promptRef: "prompts/arch.md"
      rag:
        sources: ["code"]
        requireCitations: true
        onMissingCitations: "human_confirm"
`
	if err := os.WriteFile(filepath.Join(scenariosDir, "citation_human.yaml"), []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	svc := NewService(db, ev, loader, toolbus.DefaultBus()).WithAgentExecutor(agentexec.StaticExecutor{})
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("unrelated content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "citation_human", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "missing citation should wait",
			"repoRoot":    repo,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "waiting_approval" {
		t.Fatalf("status=%q want waiting_approval", sum.Status)
	}

	if _, err := svc.Approve(created.RunID, ApproveRequest{ActorID: "tester", Reason: "manual evidence accepted"}); err != nil {
		t.Fatal(err)
	}
	sum, err = svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "finished" {
		t.Fatalf("status=%q want finished", sum.Status)
	}

	meta, err := loadRunMeta(svc.db.RunDir(created.RunID))
	if err != nil {
		t.Fatal(err)
	}
	if !approvedStep(meta.Inputs, "_approvedCitationSteps", "arch.design") {
		t.Fatal("expected approved citation step to be persisted in run meta")
	}

	evs, err := svc.events.ListAfter(created.RunID, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	foundApproved := false
	foundCitationBypass := false
	for _, ev := range evs {
		if ev.Type == "gate.approved" {
			foundApproved = true
		}
		if ev.Type == "citation.approved_without_evidence" {
			foundCitationBypass = true
		}
	}
	if !foundApproved {
		t.Fatal("expected gate.approved event")
	}
	if !foundCitationBypass {
		t.Fatal("expected citation.approved_without_evidence event")
	}
}

func writeFakeRunsExecGoCLI(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "execgocli")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
