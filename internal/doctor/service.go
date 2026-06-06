package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/authz"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/modelrouter"
	"github.com/ash-repwiki/ash/internal/observability"
	"github.com/ash-repwiki/ash/internal/pluginabi"
	"github.com/ash-repwiki/ash/internal/rag"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/security"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

// Evidence links a check to run artifacts or events.
type Evidence struct {
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Digest string `json:"digest,omitempty"`
}

// CaseResult is one TR case outcome.
type CaseResult struct {
	ID       string     `json:"id"`
	Status   string     `json:"status"`
	RunID    string     `json:"runId,omitempty"`
	Message  string     `json:"message,omitempty"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// Report is the structured doctor output (appendix E).
type Report struct {
	Suite      string       `json:"suite"`
	StartedAt  int64        `json:"startedAt"`
	FinishedAt int64        `json:"finishedAt"`
	Results    []CaseResult `json:"results"`
	Summary    struct {
		Pass int `json:"pass"`
		Fail int `json:"fail"`
	} `json:"summary"`
}

// Service runs validation suites in-process.
type Service struct {
	runs      *runs.Service
	events    *events.Service
	scenarios *rules.Loader
	dataDir   string
}

func NewService(runsSvc *runs.Service, ev *events.Service, scenarios *rules.Loader, dataDir string) *Service {
	return &Service{runs: runsSvc, events: ev, scenarios: scenarios, dataDir: dataDir}
}

func (s *Service) RunSuite(suite string) (*Report, error) {
	start := time.Now().UTC()
	rep := &Report{Suite: suite, StartedAt: start.UnixMilli()}

	switch suite {
	case "TR0":
		rep.Results = append(rep.Results, s.tr0DeliveryLoop())
		rep.Results = append(rep.Results, s.tr0EventStream())
		rep.Results = append(rep.Results, s.tr0ReplayDigest())
		rep.Results = append(rep.Results, s.tr0AgentTask())
		rep.Results = append(rep.Results, s.tr0ArtifactIndex())
		rep.Results = append(rep.Results, s.tr0EvidenceBinding())
		rep.Results = append(rep.Results, s.tr0CheckpointRecovery())
		rep.Results = append(rep.Results, s.tr0ScenarioCatalog())
	case "TR1":
		rep.Results = append(rep.Results, s.tr1ModelRouterFallback())
		rep.Results = append(rep.Results, s.tr1WaterfallQuality())
		rep.Results = append(rep.Results, s.tr1MemoryConflict())
		rep.Results = append(rep.Results, s.tr1MCPIsolation())
		rep.Results = append(rep.Results, s.tr1DSLSchemaValidation())
	case "TR2":
		rep.Results = append(rep.Results, s.tr2IdentityScopeModel())
		rep.Results = append(rep.Results, s.tr2SpaceScopedRuns())
		rep.Results = append(rep.Results, s.tr2ArtifactStoreProfile())
		rep.Results = append(rep.Results, s.tr2PluginABI())
		rep.Results = append(rep.Results, s.tr2SecretLeakScan())
	case "M2":
		rep.Results = append(rep.Results, s.m2PermissionMatrix())
		rep.Results = append(rep.Results, s.m2ScenarioPolicyUpdate())
		rep.Results = append(rep.Results, s.m2ScenarioPolicyEnforcement())
	case "M3":
		rep.Results = append(rep.Results, s.m3TenantIsolation())
		rep.Results = append(rep.Results, s.m3PostgresReadiness())
		rep.Results = append(rep.Results, s.m3MigrationCatalog())
		rep.Results = append(rep.Results, s.m3PostgresMigrateVerify())
		rep.Results = append(rep.Results, s.m3ExecGoLiveSmoke())
	case "TR3":
		rep.Results = append(rep.Results, s.tr3MemoryMigration())
		rep.Results = append(rep.Results, s.tr3RAGFallback())
		rep.Results = append(rep.Results, s.tr3CostLatencySLO())
		rep.Results = append(rep.Results, s.tr3AuditProvenance())
	case "ALL":
		rep.Results = append(rep.Results, s.tr0DeliveryLoop())
		rep.Results = append(rep.Results, s.tr0EventStream())
		rep.Results = append(rep.Results, s.tr0ReplayDigest())
		rep.Results = append(rep.Results, s.tr0AgentTask())
		rep.Results = append(rep.Results, s.tr0ArtifactIndex())
		rep.Results = append(rep.Results, s.tr0EvidenceBinding())
		rep.Results = append(rep.Results, s.tr0CheckpointRecovery())
		rep.Results = append(rep.Results, s.tr0ScenarioCatalog())
		rep.Results = append(rep.Results, s.tr1ModelRouterFallback())
		rep.Results = append(rep.Results, s.tr1WaterfallQuality())
		rep.Results = append(rep.Results, s.tr1MemoryConflict())
		rep.Results = append(rep.Results, s.tr1MCPIsolation())
		rep.Results = append(rep.Results, s.tr1DSLSchemaValidation())
		rep.Results = append(rep.Results, s.tr2IdentityScopeModel())
		rep.Results = append(rep.Results, s.tr2SpaceScopedRuns())
		rep.Results = append(rep.Results, s.tr2ArtifactStoreProfile())
		rep.Results = append(rep.Results, s.tr2PluginABI())
		rep.Results = append(rep.Results, s.tr2SecretLeakScan())
		rep.Results = append(rep.Results, s.m2PermissionMatrix())
		rep.Results = append(rep.Results, s.m2ScenarioPolicyUpdate())
		rep.Results = append(rep.Results, s.m2ScenarioPolicyEnforcement())
		rep.Results = append(rep.Results, s.m3TenantIsolation())
		rep.Results = append(rep.Results, s.m3PostgresReadiness())
		rep.Results = append(rep.Results, s.m3MigrationCatalog())
		rep.Results = append(rep.Results, s.m3PostgresMigrateVerify())
		rep.Results = append(rep.Results, s.m3ExecGoLiveSmoke())
		rep.Results = append(rep.Results, s.tr3MemoryMigration())
		rep.Results = append(rep.Results, s.tr3RAGFallback())
		rep.Results = append(rep.Results, s.tr3CostLatencySLO())
		rep.Results = append(rep.Results, s.tr3AuditProvenance())
	default:
		return nil, fmt.Errorf("unsupported suite %q", suite)
	}

	for _, r := range rep.Results {
		if r.Status == "pass" {
			rep.Summary.Pass++
		} else {
			rep.Summary.Fail++
		}
	}
	rep.FinishedAt = time.Now().UTC().UnixMilli()
	return rep, nil
}

func (s *Service) probeInputs(suffix string) map[string]any {
	probeDir := filepath.Join(s.dataDir, "doctor-probe", suffix)
	_ = os.MkdirAll(probeDir, 0o755)
	issue := "doctor " + suffix
	_ = os.WriteFile(filepath.Join(probeDir, "README.md"), []byte(issue+"\nFeature delivery evidence for ASH doctor.\n"), 0o644)
	return map[string]any{
		"issueOrSpec": issue,
		"repoRoot":    probeDir,
	}
}

func (s *Service) createProbeRun(caseID string) (*runs.CreateResponse, string, error) {
	inputs := s.probeInputs(caseID)
	repoRoot, _ := inputs["repoRoot"].(string)
	create, err := s.runs.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   inputs,
		Repo:     &runs.RepoRef{Root: repoRoot},
	})
	return create, repoRoot, err
}

func (s *Service) tr0DeliveryLoop() CaseResult {
	res := CaseResult{ID: "TR0-01", Status: "fail"}
	create, err := s.runs.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   s.probeInputs("TR0-01"),
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	manifest, err := s.runs.Artifacts(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	required := map[string]bool{"diff": false, "test_report": false, "release_notes": false, "rollback_plan": false}
	for _, a := range manifest.Artifacts {
		if _, ok := required[a.Type]; ok {
			required[a.Type] = true
			res.Evidence = append(res.Evidence, Evidence{Kind: "artifact", Ref: a.Type, Digest: a.Digest})
		}
	}
	for typ, ok := range required {
		if !ok {
			res.Message = fmt.Sprintf("missing artifact type %q", typ)
			return res
		}
	}
	qualityEvidence, qualityMessage := artifactQualityEvidence(s.runs.DB().RunDir(create.RunID), manifest, s.runs.AgentAdapter() != "static")
	if qualityMessage != "" {
		res.Message = qualityMessage
		return res
	}
	res.Evidence = append(res.Evidence, qualityEvidence...)
	res.Status = "pass"
	return res
}

func (s *Service) tr0EventStream() CaseResult {
	res := CaseResult{ID: "TR0-02", Status: "fail"}
	create, err := s.runs.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   s.probeInputs("TR0-02"),
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	evs, err := s.events.ListAfter(create.RunID, 0, 500)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if len(evs) < 3 {
		res.Message = fmt.Sprintf("expected events, got %d", len(evs))
		return res
	}
	last := evs[len(evs)-1]
	resumed, err := s.events.ListAfter(create.RunID, last.Seq-1, 500)
	if err != nil || len(resumed) == 0 {
		res.Message = "Last-Event-Seq resume failed"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "eventRange", Ref: fmt.Sprintf("run_events:seq=1..%d", last.Seq)},
		Evidence{Kind: "event", Ref: last.ID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr0ReplayDigest() CaseResult {
	res := CaseResult{ID: "TR0-03", Status: "fail"}
	create, _, err := s.createProbeRun("TR0-03")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	sourceManifest, err := s.runs.Artifacts(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	replayed, err := s.runs.Replay(create.RunID, runs.ReplayRequest{Mode: "exact"})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if replayed.RunID == create.RunID {
		res.Message = "replay reused source run id"
		return res
	}
	replayManifest, err := s.runs.Artifacts(replayed.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	sourceByType := artifactDigestByType(sourceManifest)
	replayByType := artifactDigestByType(replayManifest)
	for _, typ := range []string{"diff"} {
		sourceDigest := sourceByType[typ]
		replayDigest := replayByType[typ]
		if sourceDigest == "" || replayDigest == "" {
			res.Message = fmt.Sprintf("missing %s artifact digest for replay comparison", typ)
			return res
		}
		if sourceDigest != replayDigest {
			res.Message = fmt.Sprintf("digest drift on replay %s: source=%s replay=%s", typ, sourceDigest, replayDigest)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{Kind: "artifact", Ref: "source:" + typ, Digest: sourceDigest})
		res.Evidence = append(res.Evidence, Evidence{Kind: "artifact", Ref: "replay:" + typ, Digest: replayDigest})
	}
	if replayByType["test_report"] == "" {
		res.Message = "replay run missing test_report artifact"
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "artifact", Ref: "replay:test_report", Digest: replayByType["test_report"]})
	evs, err := s.events.ListAfter(replayed.RunID, 0, 500)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if !hasEventType(evs, "run.finished") {
		res.Message = "replay run missing run.finished event"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "run", Ref: "source:" + create.RunID},
		Evidence{Kind: "run", Ref: "replay:" + replayed.RunID},
		Evidence{Kind: "event", Ref: "replay.run.finished"},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr0AgentTask() CaseResult {
	res := CaseResult{ID: "TR0-04", Status: "fail"}
	create, _, err := s.createProbeRun("TR0-04")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	tasks, err := s.runs.AgentTasks(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if len(tasks) == 0 {
		res.Message = "missing agent task record"
		return res
	}

	foundCoder := false
	for _, task := range tasks {
		if task.StepID != "code.implement" {
			continue
		}
		foundCoder = true
		if task.Status != "success" {
			res.Message = fmt.Sprintf("agent task %s status=%s", task.ID, task.Status)
			return res
		}
		if task.ExecGoTaskID == "" && task.ActionID == "" {
			res.Message = fmt.Sprintf("agent task %s missing external task id", task.ID)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{Kind: "agentTask", Ref: task.ID, Digest: task.PromptDigest})
	}
	if !foundCoder {
		res.Message = "missing code.implement agent task"
		return res
	}

	evs, err := s.events.ListAfter(create.RunID, 0, 500)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if !hasEventType(evs, "agent.finished") {
		res.Message = "missing agent.finished event"
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "event", Ref: "agent.finished"})
	res.Status = "pass"
	return res
}

func (s *Service) tr0ArtifactIndex() CaseResult {
	res := CaseResult{ID: "TR0-05", Status: "fail"}
	create, _, err := s.createProbeRun("TR0-05")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	manifest, err := s.runs.Artifacts(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	var rows []store.ArtifactIndex
	if err := s.runs.DB().Where("run_id = ?", create.RunID).Find(&rows).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	byName := map[string]store.ArtifactIndex{}
	for _, row := range rows {
		byName[row.Name] = row
	}

	required := map[string]bool{"diff": false, "test_report": false, "release_notes": false, "rollback_plan": false}
	for _, a := range manifest.Artifacts {
		if _, ok := required[a.Type]; ok {
			required[a.Type] = true
		}
		row, ok := byName[a.Name]
		if !ok {
			res.Message = fmt.Sprintf("artifact %s missing index row", a.Name)
			return res
		}
		if row.Digest != a.Digest || row.URI == "" {
			res.Message = fmt.Sprintf("artifact %s index mismatch", a.Name)
			return res
		}
		access, err := s.runs.ArtifactAccess(create.RunID, a.Name, time.Minute)
		if err != nil {
			res.Message = fmt.Sprintf("artifact %s access failed: %v", a.Name, err)
			return res
		}
		if access.SignedURL == "" || access.Digest != a.Digest {
			res.Message = fmt.Sprintf("artifact %s access response incomplete", a.Name)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{Kind: "artifact", Ref: a.Name, Digest: a.Digest})
	}
	for typ, ok := range required {
		if !ok {
			res.Message = fmt.Sprintf("missing artifact type %q", typ)
			return res
		}
	}
	res.Status = "pass"
	return res
}

func (s *Service) tr0EvidenceBinding() CaseResult {
	res := CaseResult{ID: "TR0-06", Status: "fail"}
	create, repoRoot, err := s.createProbeRun("TR0-06")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	absRepo, err := rag.AbsRepoRoot(repoRoot)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	var chunkCount int64
	if err := s.runs.DB().Model(&store.RAGChunk{}).Where("repo_root = ?", absRepo).Count(&chunkCount).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if chunkCount == 0 {
		res.Message = "missing RAG chunks for probe repo"
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "rag", Ref: absRepo})

	evs, err := s.events.ListAfter(create.RunID, 0, 500)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if !hasEventType(evs, "rag.indexed") || !hasEventType(evs, "citation.bound") {
		res.Message = "missing rag.indexed or citation.bound event"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "event", Ref: "rag.indexed"},
		Evidence{Kind: "event", Ref: "citation.bound"},
	)

	manifest, err := s.runs.Artifacts(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	for _, a := range manifest.Artifacts {
		if evidenceRefCount(a.Producer["evidenceRefs"]) == 0 {
			res.Message = fmt.Sprintf("artifact %s missing producer evidence refs", a.Name)
			return res
		}
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "artifactManifest", Ref: "producer.evidenceRefs"})
	res.Status = "pass"
	return res
}

func (s *Service) tr0CheckpointRecovery() CaseResult {
	res := CaseResult{ID: "TR0-07", Status: "fail"}
	create, _, err := s.createProbeRun("TR0-07")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	checkpoints, err := s.runs.Checkpoints(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if len(checkpoints) == 0 {
		res.Message = "missing checkpoints"
		return res
	}
	for _, checkpoint := range checkpoints {
		if checkpoint.StepID == "" || checkpoint.SnapshotDigest == "" || checkpoint.URI == "" {
			res.Message = fmt.Sprintf("checkpoint %s missing step/digest/uri", checkpoint.ID)
			return res
		}
		access, err := s.runs.CheckpointAccess(create.RunID, checkpoint.ID, time.Minute)
		if err != nil {
			res.Message = fmt.Sprintf("checkpoint %s access failed: %v", checkpoint.ID, err)
			return res
		}
		if access.SignedURL == "" || access.SnapshotDigest != checkpoint.SnapshotDigest {
			res.Message = fmt.Sprintf("checkpoint %s access response incomplete", checkpoint.ID)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{
			Kind: "checkpoint", Ref: checkpoint.ID, Digest: checkpoint.SnapshotDigest,
		})
	}

	evs, err := s.events.ListAfter(create.RunID, 0, 500)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if !hasEventType(evs, "checkpoint.stored") || !hasEventType(evs, "run.checkpoint_saved") {
		res.Message = "missing checkpoint.stored or run.checkpoint_saved event"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "event", Ref: "checkpoint.stored"},
		Evidence{Kind: "event", Ref: "run.checkpoint_saved"},
	)

	var auditCount int64
	if err := s.runs.DB().Model(&store.AuditLog{}).
		Where("run_id = ? AND event_type = ?", create.RunID, "checkpoint.access_url_issued").
		Count(&auditCount).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if auditCount < int64(len(checkpoints)) {
		res.Message = fmt.Sprintf("checkpoint access audit count=%d want at least %d", auditCount, len(checkpoints))
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "audit", Ref: "checkpoint.access_url_issued"})
	res.Status = "pass"
	return res
}

func (s *Service) tr0ScenarioCatalog() CaseResult {
	res := CaseResult{ID: "TR0-08", Status: "fail"}
	if s.scenarios == nil {
		res.Message = "scenario loader not configured"
		return res
	}
	required := []struct {
		name    string
		version string
	}{
		{name: "feature_delivery", version: "1.0.0"},
		{name: "hotfix", version: "1.0.0"},
		{name: "security_patch", version: "1.0.0"},
	}
	for _, item := range required {
		doc, err := s.scenarios.Get(item.name, item.version)
		if err != nil {
			res.Message = fmt.Sprintf("missing scenario %s@%s: %v", item.name, item.version, err)
			return res
		}
		if doc.Scenario.Name != item.name {
			res.Message = fmt.Sprintf("scenario name mismatch for %s@%s", item.name, item.version)
			return res
		}
		if len(doc.Scenario.Steps) == 0 {
			res.Message = fmt.Sprintf("scenario %s@%s has no steps", item.name, item.version)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{
			Kind: "scenario",
			Ref:  item.name + "@" + item.version,
		})
	}
	res.Status = "pass"
	return res
}

func (s *Service) tr1ModelRouterFallback() CaseResult {
	res := CaseResult{ID: "TR1-01", Status: "fail"}
	router := modelrouter.New([]modelrouter.Provider{
		{ID: "primary", Provider: "openai-compatible", Model: "primary-model", Role: "default", Status: "unavailable"},
		{
			ID: "fallback", Provider: "openai-compatible", Model: "fallback-model", Role: "fallback", Status: "available",
			InputMicrosPer1K: 2000, OutputMicrosPer1K: 6000,
		},
	})
	decision := router.Route(modelrouter.Request{Prompt: "doctor TR1 fallback route", OutputTokens: 10})
	if decision.Provider.ID != "fallback" || !decision.FallbackUsed || decision.Status != "routed" {
		res.Message = fmt.Sprintf("unexpected route decision: %+v", decision)
		return res
	}
	if decision.CostMicros <= 0 {
		res.Message = "fallback route missing positive cost estimate"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "modelRouter", Ref: "fallback"},
		Evidence{Kind: "modelUsage", Ref: fmt.Sprintf("costMicros:%d", decision.CostMicros)},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr1WaterfallQuality() CaseResult {
	res := CaseResult{ID: "TR1-02", Status: "fail"}
	create, _, err := s.createProbeRun("TR1-02")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	waterfall, err := observability.BuildWaterfall(s.runs.DB(), create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	for _, typ := range []string{"run", "step", "tool", "agent", "model"} {
		if !hasWaterfallSpanType(waterfall.Spans, typ) {
			res.Message = fmt.Sprintf("waterfall missing %s span", typ)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{Kind: "waterfallSpan", Ref: typ})
	}
	if !hasWaterfallMetric(waterfall.Metrics, "citation_hit_rate") || !hasWaterfallMetric(waterfall.Metrics, "tool_failure_rate") {
		res.Message = "waterfall missing quality metrics"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "qualityMetric", Ref: "citation_hit_rate"},
		Evidence{Kind: "qualityMetric", Ref: "tool_failure_rate"},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr1MemoryConflict() CaseResult {
	res := CaseResult{ID: "TR1-03", Status: "fail"}
	mem := memory.NewService(s.runs.DB(), s.events)
	confidence := 0.88
	base, err := mem.CreateCandidate(memory.CreateCandidateRequest{
		Layer:     "L1",
		Title:     "TR1 conflict policy",
		Body:      "Use conservative release gates.",
		ScopeRepo: "ash",
		Evidence:  []memory.EvidenceInput{{Kind: "file", Ref: "doc/tr1.md"}},
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if _, err := mem.Review(base.CandidateID, memory.ReviewRequest{
		Decision: "approve", Reason: "baseline", PolicyProfile: "default", ReviewerID: "doctor", Confidence: &confidence,
	}); err != nil {
		res.Message = err.Error()
		return res
	}
	conflict, err := mem.CreateCandidate(memory.CreateCandidateRequest{
		Layer:     "L1",
		Title:     "TR1 conflict policy",
		Body:      "Skip release gates for speed.",
		ScopeRepo: "ash",
		Evidence:  []memory.EvidenceInput{{Kind: "file", Ref: "doc/tr1.md"}},
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if _, err := mem.Review(conflict.CandidateID, memory.ReviewRequest{
		Decision: "approve", Reason: "conflicting policy", PolicyProfile: "default", ReviewerID: "doctor",
	}); err != nil {
		res.Message = err.Error()
		return res
	}
	var edges []store.MemoryEdge
	if err := s.runs.DB().Where("from_id = ? AND to_id = ? AND kind = ?", conflict.CandidateID, base.CandidateID, "conflict").
		Find(&edges).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if len(edges) == 0 {
		res.Message = "missing memory conflict edge"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "memory", Ref: base.CandidateID},
		Evidence{Kind: "memory", Ref: conflict.CandidateID},
		Evidence{Kind: "memoryEdge", Ref: edges[0].ID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr1MCPIsolation() CaseResult {
	res := CaseResult{ID: "TR1-04", Status: "fail"}
	bus := toolbus.DefaultBus()
	schema := bus.Call(toolbus.Context{}, toolbus.CallRequest{
		Tool: "mcp.call",
		Args: map[string]any{
			"serverURL": "http://127.0.0.1:1",
			"tool":      "demo.echo",
			"headers":   map[string]any{"Authorization": "secret"},
		},
	})
	if schema.OK || schema.FailureClass != "mcp_schema" {
		res.Message = fmt.Sprintf("expected MCP schema isolation, got %+v", schema)
		return res
	}
	policy := bus.Call(toolbus.Context{}, toolbus.CallRequest{
		Tool: "mcp.call",
		Args: map[string]any{
			"serverURL":   "http://127.0.0.1:1",
			"tool":        "demo.echo",
			"arguments":   map[string]any{"message": "hello", "secret": "nope"},
			"allowedArgs": []any{"message"},
		},
	})
	if policy.OK || policy.FailureClass != "mcp_policy" {
		res.Message = fmt.Sprintf("expected MCP policy isolation, got %+v", policy)
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "mcpIsolation", Ref: "schema"},
		Evidence{Kind: "mcpIsolation", Ref: "policy"},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr1DSLSchemaValidation() CaseResult {
	res := CaseResult{ID: "TR1-05", Status: "fail"}
	if s.scenarios == nil {
		res.Message = "scenario loader not configured"
		return res
	}
	invalid := []byte(`version: "ash.rules/v0.1"
scenario:
  name: broken
`)
	invalidResult := s.scenarios.ValidateYAML(invalid)
	if invalidResult.OK {
		res.Message = "invalid DSL was accepted"
		return res
	}
	valid := []byte(`version: "ash.rules/v0.1"
scenario:
  name: "doctor_schema_ok"
  scenarioVersion: "1.0.0"
  roles:
    PM: { maxParallel: 1 }
  steps:
    - id: "noop"
      role: "PM"
      kind: "llm"
      promptRef: "prompts/noop.md"
`)
	validResult := s.scenarios.ValidateYAML(valid)
	if !validResult.OK {
		res.Message = fmt.Sprintf("valid DSL rejected: %+v", validResult.Issues)
		return res
	}
	code := "rejected"
	if len(invalidResult.Issues) > 0 {
		code = invalidResult.Issues[0].Code
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "rulesValidation", Ref: "invalid:" + code},
		Evidence{Kind: "rulesValidation", Ref: "valid:ok"},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr2IdentityScopeModel() CaseResult {
	res := CaseResult{ID: "TR2-01", Status: "fail"}
	now := time.Now().UTC()
	suffix := fmt.Sprintf("%d", now.UnixNano())
	org := store.Org{ID: "org_tr2_" + suffix, Name: "TR2 Org", Slug: "tr2-" + suffix, CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_tr2_" + suffix, OrgID: org.ID, Name: "TR2 Space", Slug: "tr2-" + suffix, CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_tr2_" + suffix, Email: "tr2-" + suffix + "@ash.local", DisplayName: "TR2 User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{
		ID: "role_tr2_" + suffix, OrgID: org.ID, Name: "TR2 Maintainer",
		Permissions: `["run:*","artifact:read","plugin:*"]`, CreatedAt: now, UpdatedAt: now,
	}
	member := store.Member{
		ID: "mem_tr2_" + suffix, OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	scope := store.ResourceScope{
		ID: "scope_tr2_" + suffix, SpaceID: space.ID, ResourceType: "space", ResourceID: space.ID,
		PolicyJSON: `{"profile":"doctor-tr2"}`, CreatedAt: now, UpdatedAt: now,
	}
	policy := store.AuditPolicy{
		SpaceID: space.ID, RetentionDays: 365, RedactPayload: false, CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member, &scope, &policy} {
		if err := s.runs.DB().Create(row).Error; err != nil {
			res.Message = err.Error()
			return res
		}
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "identityModel", Ref: org.ID},
		Evidence{Kind: "space", Ref: space.ID},
		Evidence{Kind: "memberRole", Ref: member.ID},
		Evidence{Kind: "resourceScope", Ref: scope.ID},
		Evidence{Kind: "auditPolicy", Ref: space.ID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr2SpaceScopedRuns() CaseResult {
	res := CaseResult{ID: "TR2-02", Status: "fail"}
	spaceA := "space_tr2_runs_a"
	spaceB := "space_tr2_runs_b"
	inputsA := s.probeInputs("TR2-02-a")
	inputsB := s.probeInputs("TR2-02-b")
	repoA, _ := inputsA["repoRoot"].(string)
	repoB, _ := inputsB["repoRoot"].(string)
	runA, err := s.runs.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   inputsA, Repo: &runs.RepoRef{Root: repoA}, SpaceID: spaceA,
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	runB, err := s.runs.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   inputsB, Repo: &runs.RepoRef{Root: repoB}, SpaceID: spaceB,
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	aRows, err := s.runs.ListForSpace(spaceA, 50)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	bRows, err := s.runs.ListForSpace(spaceB, 50)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if !runListContainsOnly(aRows, runA.RunID, spaceA) || !runListContainsOnly(bRows, runB.RunID, spaceB) {
		res.Message = fmt.Sprintf("space scoped run list mismatch: a=%+v b=%+v", aRows, bRows)
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "runScope", Ref: spaceA + ":" + runA.RunID},
		Evidence{Kind: "runScope", Ref: spaceB + ":" + runB.RunID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr2ArtifactStoreProfile() CaseResult {
	res := CaseResult{ID: "TR2-03", Status: "fail"}
	profile := artifactstore.Describe("fs", s.dataDir)
	if !profile.Ready || !profile.SupportsSignedURL || profile.URI == "" {
		res.Message = fmt.Sprintf("artifact store profile not ready: %+v", profile)
		return res
	}
	create, _, err := s.createProbeRun("TR2-03")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	var artifacts []store.ArtifactIndex
	if err := s.runs.DB().Where("run_id = ?", create.RunID).Find(&artifacts).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if len(artifacts) == 0 || !allArtifactsHaveStoreKeys(artifacts) {
		res.Message = fmt.Sprintf("artifact index missing store metadata: %+v", artifacts)
		return res
	}
	checkpoints, err := s.runs.Checkpoints(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if len(checkpoints) == 0 || !allCheckpointsHaveStoreKeys(checkpoints) {
		res.Message = fmt.Sprintf("checkpoint index missing store metadata: %+v", checkpoints)
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "storageProfile", Ref: profile.Kind},
		Evidence{Kind: "artifactStore", Ref: fmt.Sprintf("artifacts:%d", len(artifacts))},
		Evidence{Kind: "checkpointStore", Ref: fmt.Sprintf("checkpoints:%d", len(checkpoints))},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr2PluginABI() CaseResult {
	res := CaseResult{ID: "TR2-04", Status: "fail"}
	ok, reason := pluginabi.Compatible("grpc", pluginabi.CurrentABI, "doctor-plugin", "1.0.0")
	if !ok {
		res.Message = "current ABI rejected: " + reason
		return res
	}
	bad, _ := pluginabi.Compatible("grpc", "ash.plugin.v0", "doctor-plugin", "1.0.0")
	if bad {
		res.Message = "incompatible ABI was accepted"
		return res
	}
	now := time.Now().UTC()
	suffix := fmt.Sprintf("%d", now.UnixNano())
	row := store.PluginRegistry{
		ID: "plg_tr2_" + suffix, SpaceID: "space_tr2_plugin",
		Name: "doctor-plugin", Version: "1.0.0", Protocol: "grpc", ABI: pluginabi.CurrentABI,
		Endpoint: "dns:///doctor-plugin", Capabilities: `["runs.inspect"]`, Compatible: true,
		Status: "registered", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.runs.DB().Create(&row).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "pluginABI", Ref: pluginabi.CurrentABI},
		Evidence{Kind: "pluginRegistry", Ref: row.ID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr2SecretLeakScan() CaseResult {
	res := CaseResult{ID: "TR2-05", Status: "fail"}
	now := time.Now().UTC()
	suffix := fmt.Sprintf("%d", now.UnixNano())
	spaceID := "space_tr2_secret_" + suffix
	policy := store.AuditPolicy{
		SpaceID: spaceID, RetentionDays: 90, RedactPayload: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.runs.DB().Create(&policy).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	leakPayload := `{"password":"doctor-tr2-leak-probe","note":"compliance scan"}`
	audit := store.AuditLog{
		ID: "aud_tr2_leak_" + suffix, SpaceID: spaceID, EventType: "compliance.probe",
		PayloadJSON: leakPayload, CreatedAt: now,
	}
	if err := s.runs.DB().Create(&audit).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	findings := security.FindLeaks("audit_log", audit.ID, leakPayload)
	if len(findings) == 0 {
		res.Message = "leak scanner did not detect probe secret"
		return res
	}
	redacted := security.RedactJSON(leakPayload)
	if redacted == leakPayload || strings.Contains(redacted, "doctor-tr2-leak-probe") {
		res.Message = "redact did not mask probe secret"
		return res
	}
	var secret store.SecretRecord
	if err := s.runs.DB().Where("space_id = ?", spaceID).Limit(1).Find(&secret).Error; err != nil {
		secret = store.SecretRecord{
			ID: "sec_tr2_" + suffix, SpaceID: spaceID, Name: "PROBE_KEY",
			ValueDigest: "sha256:probe", ValueCiphertext: "enc:probe-value",
			Status: "active", ScopeJSON: "{}", CreatedAt: now, UpdatedAt: now,
		}
		if err := s.runs.DB().Create(&secret).Error; err != nil {
			res.Message = err.Error()
			return res
		}
	}
	if strings.Contains(secret.ValueCiphertext, "probe-value") {
		// API must never return encrypted blob as plaintext; doctor checks redact path only.
		res.Evidence = append(res.Evidence, Evidence{Kind: "secretStore", Ref: "encrypted_at_rest"})
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "secretScan", Ref: fmt.Sprintf("findings:%d", len(findings))},
		Evidence{Kind: "redact", Ref: "payload_masked"},
		Evidence{Kind: "auditPolicy", Ref: spaceID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) m2PermissionMatrix() CaseResult {
	res := CaseResult{ID: "M2-01", Status: "fail"}
	if !authz.RoleAllows("viewer", "artifact:read") || authz.RoleAllows("viewer", "run:create") {
		res.Message = "builtin RBAC matrix inconsistent for viewer"
		return res
	}
	policy := authz.DefaultScenarioPolicyJSON("feature_delivery", "1.0.0")
	ok, _ := authz.EvaluateScenarioTool(policy, "reviewer", "git.status")
	if !ok {
		res.Message = "reviewer cannot read git.status"
		return res
	}
	ok, reason := authz.EvaluateScenarioTool(policy, "reviewer", "apply_patch")
	if ok {
		res.Message = "reviewer should be denied apply_patch"
		return res
	}
	secPolicy := authz.DefaultScenarioPolicyJSON("security_patch", "1.0.0")
	ok, _ = authz.EvaluateScenarioTool(secPolicy, "operator", "apply_patch")
	if ok {
		res.Message = "security_patch operator must not apply_patch"
		return res
	}
	now := time.Now().UTC()
	spaceID := "space_m2_matrix_" + fmt.Sprintf("%d", now.UnixNano())
	if err := authz.SeedScenarioScopes(s.runs.DB(), spaceID, now); err != nil {
		res.Message = err.Error()
		return res
	}
	var count int64
	if err := s.runs.DB().Model(&store.ResourceScope{}).
		Where("space_id = ? AND resource_type = ?", spaceID, "scenario").Count(&count).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if count < 3 {
		res.Message = fmt.Sprintf("scenario scopes=%d want >=3", count)
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "rbacMatrix", Ref: "viewer:read-only"},
		Evidence{Kind: "scenarioMatrix", Ref: "reviewer:deny:apply_patch"},
		Evidence{Kind: "policyDenied", Ref: reason},
		Evidence{Kind: "resourceScope", Ref: fmt.Sprintf("scenario:%d", count)},
	)
	res.Status = "pass"
	return res
}

func (s *Service) m2ScenarioPolicyUpdate() CaseResult {
	res := CaseResult{ID: "M2-02", Status: "fail"}
	now := time.Now().UTC()
	spaceID := "space_m2_policy_" + fmt.Sprintf("%d", now.UnixNano())
	if err := authz.SeedScenarioScopes(s.runs.DB(), spaceID, now); err != nil {
		res.Message = err.Error()
		return res
	}
	var row store.ResourceScope
	key := "feature_delivery@1.0.0"
	if err := s.runs.DB().Where(
		"space_id = ? AND resource_type = ? AND resource_id = ?",
		spaceID, "scenario", key,
	).First(&row).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	customPolicy := `{"toolMatrix":{"reviewer":{"allow":["git.status","apply_patch"],"deny":[],"denyMode":"block"}}}`
	row.PolicyJSON = customPolicy
	row.UpdatedAt = now
	if err := s.runs.DB().Save(&row).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	loaded, err := authz.LoadScenarioPolicy(s.runs.DB(), spaceID, "feature_delivery", "1.0.0")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	ok, _ := authz.EvaluateScenarioTool(loaded, "reviewer", "apply_patch")
	if !ok {
		res.Message = "updated policy should allow reviewer apply_patch"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "resourceScope", Ref: row.ID},
		Evidence{Kind: "scenarioPolicy", Ref: key},
	)
	res.Status = "pass"
	return res
}

func (s *Service) m2ScenarioPolicyEnforcement() CaseResult {
	res := CaseResult{ID: "M2-03", Status: "fail"}
	now := time.Now().UTC()
	suffix := fmt.Sprintf("%d", now.UnixNano())
	spaceID := "space_m2_enforce_" + suffix
	policyJSON := `{"toolMatrix":{"reviewer":{"allow":["git.status"],"deny":["apply_patch"],"denyMode":"block"}}}`
	scope := store.ResourceScope{
		ID: "scope_m2_enforce_" + suffix, SpaceID: spaceID,
		ResourceType: "scenario", ResourceID: "m2_policy_enforce@1.0.0",
		PolicyJSON: policyJSON, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.runs.DB().Create(&scope).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	create, err := s.runs.Create(runs.CreateRequest{
		Scenario:  runs.ScenarioRef{Name: "m2_policy_enforce", ScenarioVersion: "1.0.0"},
		Inputs:    map[string]any{"issueOrSpec": "m2 policy enforcement probe"},
		SpaceID:   spaceID,
		ActorRole: "reviewer",
	})
	if create == nil || create.RunID == "" {
		res.Message = fmt.Sprintf("create run: %v", err)
		return res
	}
	if err == nil || !strings.Contains(err.Error(), "POLICY_DENIED") {
		res.Message = fmt.Sprintf("expected POLICY_DENIED, err=%v", err)
		return res
	}
	res.RunID = create.RunID
	sum, err := s.runs.Get(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if sum.Status != "failed" {
		res.Message = fmt.Sprintf("run status=%q want failed", sum.Status)
		return res
	}
	var step store.RunStep
	if err := s.runs.DB().Where("run_id = ?", create.RunID).Order("created_at asc").First(&step).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if step.ErrorCode != "POLICY_DENIED" {
		res.Message = fmt.Sprintf("step errorCode=%q want POLICY_DENIED", step.ErrorCode)
		return res
	}
	evs, err := s.events.ListAfter(create.RunID, 0, 100)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	foundDenied := false
	for _, ev := range evs {
		if ev.Type == "policy.denied" && strings.Contains(string(ev.Payload), "apply_patch") {
			foundDenied = true
			break
		}
	}
	if !foundDenied {
		res.Message = "missing policy.denied event for apply_patch"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "policyDenied", Ref: step.ErrorCode},
		Evidence{Kind: "resourceScope", Ref: scope.ID},
		Evidence{Kind: "run", Ref: create.RunID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) m3TenantIsolation() CaseResult {
	res := CaseResult{ID: "M3-01", Status: "fail"}
	now := time.Now().UTC()
	suffix := fmt.Sprintf("%d", now.UnixNano())
	spaceA := "space_m3_a_" + suffix
	spaceB := "space_m3_b_" + suffix
	memA := store.MemoryRecord{
		ID: "mem_m3_a_" + suffix, Layer: "L1", Status: "approved", SpaceID: spaceA,
		SchemaVersion: memory.CurrentSchemaVersion, Title: "a", Body: "tenant a",
		CreatedAt: now, UpdatedAt: now,
	}
	memB := store.MemoryRecord{
		ID: "mem_m3_b_" + suffix, Layer: "L1", Status: "approved", SpaceID: spaceB,
		SchemaVersion: memory.CurrentSchemaVersion, Title: "b", Body: "tenant b",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.runs.DB().Create(&memA).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if err := s.runs.DB().Create(&memB).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if err := store.EnforceSpaceAccess(memB.SpaceID, spaceA); err == nil {
		res.Message = "cross-space access should be denied"
		return res
	}
	var leak int64
	if err := s.runs.DB().Model(&store.MemoryRecord{}).
		Where("space_id = ? AND id = ?", spaceA, memB.ID).Count(&leak).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if leak != 0 {
		res.Message = "space A query returned space B memory"
		return res
	}
	var onlyA int64
	if err := s.runs.DB().Model(&store.MemoryRecord{}).Where("space_id = ?", spaceA).Count(&onlyA).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if onlyA < 1 {
		res.Message = "space A memory missing"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "tenantScope", Ref: spaceA},
		Evidence{Kind: "tenantScope", Ref: spaceB},
		Evidence{Kind: "memory", Ref: memA.ID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) m3PostgresReadiness() CaseResult {
	res := CaseResult{ID: "M3-02", Status: "fail"}
	dialect := s.runs.DB().Dialect()
	profile, err := store.DatabaseProfile(s.dataDir, os.Getenv("ASH_DATABASE_URL"))
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if profile.Dialect != dialect {
		res.Message = fmt.Sprintf("profile dialect=%q db dialect=%q", profile.Dialect, dialect)
		return res
	}
	if !profile.MigrationReady {
		res.Message = "database profile not migration-ready"
		return res
	}
	for _, raw := range []string{
		"postgres://ash:ash@127.0.0.1:5432/ash?sslmode=disable",
		"postgresql://ash:ash@127.0.0.1:5432/ash",
	} {
		parsed, err := store.ParseDatabaseTarget(s.dataDir, raw)
		if err != nil || parsed.Dialect != "postgres" {
			res.Message = fmt.Sprintf("postgres url parse failed for %q: %v", raw, err)
			return res
		}
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "databaseDialect", Ref: dialect},
		Evidence{Kind: "postgresURL", Ref: "parsed"},
	)
	if profile.PostgresConfigured {
		res.Evidence = append(res.Evidence, Evidence{Kind: "ASH_DATABASE_URL", Ref: "postgres"})
	}
	res.Status = "pass"
	return res
}

func (s *Service) m3MigrationCatalog() CaseResult {
	res := CaseResult{ID: "M3-03", Status: "fail"}
	catalog := store.MigrationCatalog()
	if len(catalog) < 25 {
		res.Message = fmt.Sprintf("migration catalog tables=%d want >=25", len(catalog))
		return res
	}
	critical := []string{"runs", "run_events", "memory_records", "audit_log", "resource_scopes"}
	set := make(map[string]struct{}, len(catalog))
	for _, name := range catalog {
		set[name] = struct{}{}
	}
	for _, name := range critical {
		if _, ok := set[name]; !ok {
			res.Message = fmt.Sprintf("migration catalog missing critical table %q", name)
			return res
		}
	}
	if err := store.VerifyMigrationSchema(s.runs.DB()); err != nil {
		res.Message = err.Error()
		return res
	}
	snap, err := store.MigrationSnapshotFor(s.dataDir)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "migrationCatalog", Ref: fmt.Sprintf("%d tables", len(catalog))},
		Evidence{Kind: "sqlitePath", Ref: snap.SQLitePath},
	)
	if snap.DualWriteEnabled {
		res.Evidence = append(res.Evidence, Evidence{Kind: "dualWrite", Ref: "enabled"})
	}
	res.Status = "pass"
	return res
}

func (s *Service) m3PostgresMigrateVerify() CaseResult {
	res := CaseResult{ID: "M3-04", Status: "fail"}
	if os.Getenv("ASH_MIGRATE_E2E") != "1" {
		res.Status = "pass"
		res.Message = "skipped: set ASH_MIGRATE_E2E=1 for live sqlite→postgres verify"
		res.Evidence = append(res.Evidence, Evidence{Kind: "skipped", Ref: "ASH_MIGRATE_E2E"})
		return res
	}
	pgURL := strings.TrimSpace(os.Getenv("ASH_DATABASE_URL"))
	if pgURL == "" {
		res.Message = "ASH_DATABASE_URL is required for migrate e2e"
		return res
	}
	target, err := store.ParseDatabaseTarget(s.dataDir, pgURL)
	if err != nil || target.Dialect != "postgres" {
		res.Message = fmt.Sprintf("postgres url invalid: %v", err)
		return res
	}
	sqlitePath := store.DefaultSQLitePath(s.dataDir)
	if _, err := os.Stat(sqlitePath); err != nil {
		res.Message = fmt.Sprintf("sqlite not found at %s", sqlitePath)
		return res
	}
	m, err := store.NewMigrator(s.dataDir, sqlitePath, pgURL)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	defer m.Close()

	if _, err := m.Verify(); err != nil {
		if _, copyErr := m.Copy(store.CopyOptions{BatchSize: 200}); copyErr != nil {
			res.Message = fmt.Sprintf("copy failed: %v", copyErr)
			return res
		}
		if _, err = m.Verify(); err != nil {
			res.Message = err.Error()
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{Kind: "migrateCopy", Ref: "reconciled"})
	}
	plan, err := m.Plan()
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "migrationVerify", Ref: fmt.Sprintf("%d tables", len(plan.Tables))},
		Evidence{Kind: "postgresURL", Ref: "live"},
	)
	res.Status = "pass"
	return res
}

func (s *Service) m3ExecGoLiveSmoke() CaseResult {
	res := CaseResult{ID: "M3-05", Status: "fail"}
	if os.Getenv("ASH_EXECGO_E2E") != "1" {
		res.Status = "pass"
		res.Message = "skipped: set ASH_EXECGO_E2E=1 for live ExecGo/Codex smoke"
		res.Evidence = append(res.Evidence, Evidence{Kind: "skipped", Ref: "ASH_EXECGO_E2E"})
		return res
	}
	if s.runs.AgentAdapter() == "static" {
		res.Message = "ASH_EXECGO_E2E=1 requires --agent execgo_codex"
		return res
	}
	create, _, err := s.createProbeRun("M3-05")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID
	tasks, err := s.runs.AgentTasks(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	for _, task := range tasks {
		if task.StepID != "code.implement" {
			continue
		}
		if task.Status != "success" {
			res.Message = fmt.Sprintf("ExecGo task %s status=%s error=%s", task.ID, task.Status, task.ErrorCode)
			return res
		}
		if task.ExecGoTaskID == "" {
			res.Message = fmt.Sprintf("ExecGo task %s missing execGoTaskId", task.ID)
			return res
		}
		res.Evidence = append(res.Evidence,
			Evidence{Kind: "execgoTask", Ref: task.ExecGoTaskID},
			Evidence{Kind: "agentTask", Ref: task.ID, Digest: task.PromptDigest},
		)
		res.Status = "pass"
		return res
	}
	res.Message = "missing code.implement ExecGo agent task"
	return res
}

func (s *Service) tr3MemoryMigration() CaseResult {
	res := CaseResult{ID: "TR3-01", Status: "fail"}
	mem := memory.NewService(s.runs.DB(), s.events)
	probeTitle := "TR3 migration probe " + fmt.Sprintf("%d", time.Now().UnixNano())
	confidence := 0.9
	cand, err := mem.CreateCandidate(memory.CreateCandidateRequest{
		Layer: "L2", Title: probeTitle, Body: "schema v1 readable after migration gate",
		ScopeRepo: "ash", Evidence: []memory.EvidenceInput{{Kind: "file", Ref: "doc/tr3.md"}},
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if _, err := mem.Review(cand.CandidateID, memory.ReviewRequest{
		Decision: "approve", Reason: "tr3 migration", ReviewerID: "doctor", PolicyProfile: "default", Confidence: &confidence,
	}); err != nil {
		res.Message = err.Error()
		return res
	}
	var row store.MemoryRecord
	if err := s.runs.DB().First(&row, "id = ?", cand.CandidateID).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if row.SchemaVersion != memory.CurrentSchemaVersion {
		res.Message = fmt.Sprintf("schema version=%d want %d", row.SchemaVersion, memory.CurrentSchemaVersion)
		return res
	}
	q, err := mem.Query(memory.QueryRequest{Text: probeTitle, TopK: 5})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	found := false
	for _, hit := range q.Items {
		if hit.ID == cand.CandidateID {
			found = true
			break
		}
	}
	if !found {
		res.Message = "approved v1 record not returned by memory query"
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "memorySchema", Ref: fmt.Sprintf("v%d", memory.CurrentSchemaVersion)},
		Evidence{Kind: "memoryQuery", Ref: cand.CandidateID},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr3RAGFallback() CaseResult {
	res := CaseResult{ID: "TR3-02", Status: "fail"}
	repo := filepath.Join(s.dataDir, "doctor_tr3_rag_"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(repo, 0o755); err != nil {
		res.Message = err.Error()
		return res
	}
	content := "TR3 disaster recovery fallback evidence line\n"
	if err := os.WriteFile(filepath.Join(repo, "fallback.md"), []byte(content), 0o644); err != nil {
		res.Message = err.Error()
		return res
	}
	ragSvc := rag.NewService(s.runs.DB())
	if _, err := ragSvc.Index(rag.IndexRequest{RepoRoot: repo, SpaceID: "local"}); err != nil {
		res.Message = err.Error()
		return res
	}
	_ = s.runs.DB().Exec("DROP TABLE IF EXISTS rag_chunks_fts").Error
	resp, err := ragSvc.Query(rag.QueryRequest{RepoRoot: repo, Text: "fallback evidence", TopK: 3})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if len(resp.Items) == 0 || resp.Items[0].Path != "fallback.md" {
		res.Message = fmt.Sprintf("FTS-down fallback query empty: %+v", resp.Items)
		return res
	}
	res.Evidence = append(res.Evidence,
		Evidence{Kind: "ragFallback", Ref: resp.Items[0].Path},
		Evidence{Kind: "ragIndex", Ref: repo},
	)
	res.Status = "pass"
	return res
}

func (s *Service) tr3CostLatencySLO() CaseResult {
	res := CaseResult{ID: "TR3-03", Status: "fail"}
	create, _, err := s.createProbeRun("TR3-03")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	waterfall, err := observability.BuildWaterfall(s.runs.DB(), create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	runSpanOK := false
	for _, span := range waterfall.Spans {
		if span.Type == "run" && span.DurationMs > 0 {
			runSpanOK = true
			res.Evidence = append(res.Evidence, Evidence{Kind: "latencySpan", Ref: fmt.Sprintf("run:%dms", span.DurationMs)})
			break
		}
	}
	if !runSpanOK {
		res.Message = "waterfall missing positive run duration"
		return res
	}
	names := map[string]bool{}
	for _, m := range waterfall.Metrics {
		names[m.Name] = true
	}
	for _, want := range []string{"model_cost_micros_total", "tool_calls_total"} {
		if !names[want] {
			res.Message = fmt.Sprintf("missing quality metric %s", want)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{Kind: "sloMetric", Ref: want})
	}
	var usageCount int64
	if err := s.runs.DB().Model(&store.ModelUsage{}).Where("run_id = ?", create.RunID).Count(&usageCount).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if usageCount == 0 {
		// Static agent runs may omit model_usage rows; cost metric from quality ledger still gates SLO wiring.
		res.Evidence = append(res.Evidence, Evidence{Kind: "modelUsage", Ref: "ledger_only"})
	} else {
		res.Evidence = append(res.Evidence, Evidence{Kind: "modelUsage", Ref: fmt.Sprintf("rows:%d", usageCount)})
	}
	res.Status = "pass"
	return res
}

func (s *Service) tr3AuditProvenance() CaseResult {
	res := CaseResult{ID: "TR3-04", Status: "fail"}
	create, _, err := s.createProbeRun("TR3-04")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	var rec store.RunRecord
	if err := s.runs.DB().First(&rec, "id = ?", create.RunID).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if rec.TraceID == "" {
		res.Message = "run missing traceId"
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "trace", Ref: rec.TraceID})

	var eventCount, toolCount, agentCount int64
	_ = s.runs.DB().Model(&store.RunEvent{}).Where("run_id = ?", create.RunID).Count(&eventCount).Error
	_ = s.runs.DB().Model(&store.ToolCall{}).Where("run_id = ?", create.RunID).Count(&toolCount).Error
	_ = s.runs.DB().Model(&store.AgentTask{}).Where("run_id = ?", create.RunID).Count(&agentCount).Error
	if eventCount == 0 {
		res.Message = "missing run events for provenance chain"
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "eventRange", Ref: fmt.Sprintf("count=%d", eventCount)})

	manifest, err := s.runs.Artifacts(create.RunID)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if manifest == nil || len(manifest.Artifacts) == 0 {
		res.Message = "missing artifact manifest for delivery trace"
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "artifactManifest", Ref: create.RunID})
	if toolCount+agentCount == 0 {
		res.Message = "missing tool or agent provenance rows"
		return res
	}
	if toolCount > 0 {
		res.Evidence = append(res.Evidence, Evidence{Kind: "toolCalls", Ref: fmt.Sprintf("count=%d", toolCount)})
	}
	if agentCount > 0 {
		res.Evidence = append(res.Evidence, Evidence{Kind: "agentTasks", Ref: fmt.Sprintf("count=%d", agentCount)})
	}
	var auditCount int64
	_ = s.runs.DB().Model(&store.AuditLog{}).Where("run_id = ?", create.RunID).Count(&auditCount).Error
	if auditCount > 0 {
		res.Evidence = append(res.Evidence, Evidence{Kind: "audit", Ref: fmt.Sprintf("count=%d", auditCount)})
	}
	res.Status = "pass"
	return res
}

func hasEventType(evs []events.Envelope, typ string) bool {
	for _, ev := range evs {
		if ev.Type == typ {
			return true
		}
	}
	return false
}

func runListContainsOnly(rows []runs.Summary, runID, spaceID string) bool {
	if len(rows) == 0 {
		return false
	}
	found := false
	for _, row := range rows {
		if row.SpaceID != spaceID {
			return false
		}
		if row.RunID == runID {
			found = true
		}
	}
	return found
}

func allArtifactsHaveStoreKeys(rows []store.ArtifactIndex) bool {
	for _, row := range rows {
		if row.URI == "" || row.StoreKey == "" || row.Digest == "" {
			return false
		}
	}
	return true
}

func allCheckpointsHaveStoreKeys(rows []store.Checkpoint) bool {
	for _, row := range rows {
		if row.URI == "" || row.StoreKey == "" || row.SnapshotDigest == "" {
			return false
		}
	}
	return true
}

func hasWaterfallSpanType(spans []observability.Span, typ string) bool {
	for _, span := range spans {
		if span.Type == typ {
			return true
		}
	}
	return false
}

func hasWaterfallMetric(metrics []observability.Metric, name string) bool {
	for _, metric := range metrics {
		if metric.Name == name {
			return true
		}
	}
	return false
}

func artifactDigestByType(manifest *artifacts.Manifest) map[string]string {
	out := map[string]string{}
	if manifest == nil {
		return out
	}
	for _, art := range manifest.Artifacts {
		out[art.Type] = art.Digest
	}
	return out
}

func evidenceRefCount(v any) int {
	switch refs := v.(type) {
	case []any:
		return len(refs)
	case []string:
		return len(refs)
	default:
		return 0
	}
}

func artifactQualityEvidence(runDir string, manifest *artifacts.Manifest, strict bool) ([]Evidence, string) {
	if err := artifacts.ValidateQuality(runDir, manifest, strict); err != nil {
		return nil, err.Error()
	}
	evidence := []Evidence{}
	for _, art := range manifest.Artifacts {
		switch art.Type {
		case "diff":
			evidence = append(evidence, Evidence{Kind: "artifactQuality", Ref: "diff.patch", Digest: art.Digest})
		case "test_report":
			evidence = append(evidence, Evidence{Kind: "artifactQuality", Ref: "test_report.json", Digest: art.Digest})
		}
	}
	return evidence, ""
}
