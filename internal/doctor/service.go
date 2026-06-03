package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/modelrouter"
	"github.com/ash-repwiki/ash/internal/observability"
	"github.com/ash-repwiki/ash/internal/pluginabi"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
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
	case "TR1":
		rep.Results = append(rep.Results, s.tr1ModelRouterFallback())
		rep.Results = append(rep.Results, s.tr1WaterfallQuality())
		rep.Results = append(rep.Results, s.tr1MemoryConflict())
		rep.Results = append(rep.Results, s.tr1MCPIsolation())
	case "TR2":
		rep.Results = append(rep.Results, s.tr2IdentityScopeModel())
		rep.Results = append(rep.Results, s.tr2SpaceScopedRuns())
		rep.Results = append(rep.Results, s.tr2ArtifactStoreProfile())
		rep.Results = append(rep.Results, s.tr2PluginABI())
	case "ALL":
		rep.Results = append(rep.Results, s.tr0DeliveryLoop())
		rep.Results = append(rep.Results, s.tr0EventStream())
		rep.Results = append(rep.Results, s.tr0ReplayDigest())
		rep.Results = append(rep.Results, s.tr0AgentTask())
		rep.Results = append(rep.Results, s.tr0ArtifactIndex())
		rep.Results = append(rep.Results, s.tr0EvidenceBinding())
		rep.Results = append(rep.Results, s.tr0CheckpointRecovery())
		rep.Results = append(rep.Results, s.tr1ModelRouterFallback())
		rep.Results = append(rep.Results, s.tr1WaterfallQuality())
		rep.Results = append(rep.Results, s.tr1MemoryConflict())
		rep.Results = append(rep.Results, s.tr1MCPIsolation())
		rep.Results = append(rep.Results, s.tr2IdentityScopeModel())
		rep.Results = append(rep.Results, s.tr2SpaceScopedRuns())
		rep.Results = append(rep.Results, s.tr2ArtifactStoreProfile())
		rep.Results = append(rep.Results, s.tr2PluginABI())
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

	var chunkCount int64
	if err := s.runs.DB().Model(&store.RAGChunk{}).Where("repo_root = ?", repoRoot).Count(&chunkCount).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	if chunkCount == 0 {
		res.Message = "missing RAG chunks for probe repo"
		return res
	}
	res.Evidence = append(res.Evidence, Evidence{Kind: "rag", Ref: repoRoot})

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
