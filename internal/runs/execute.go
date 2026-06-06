package runs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/authz"
	"github.com/ash-repwiki/ash/internal/modelrouter"
	"github.com/ash-repwiki/ash/internal/rag"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func (s *Service) createAndExecute(req CreateRequest, opts createOptions) (*CreateResponse, error) {
	doc, err := s.scenarios.Get(req.Scenario.Name, req.Scenario.ScenarioVersion)
	if err != nil {
		return nil, fmt.Errorf("scenario not found: %w", err)
	}
	eng := rules.NewEngine(doc)
	for _, key := range eng.RequiredInputs() {
		if _, ok := req.Inputs[key]; !ok {
			return nil, fmt.Errorf("missing required input %q", key)
		}
	}

	runID := "run_" + uuid.NewString()
	traceID := "trc_" + uuid.NewString()
	policy := req.PolicyProfile
	if policy == "" {
		policy = doc.Scenario.PolicyProfile
		if policy == "" {
			policy = "default"
		}
	}

	inputsDigest, err := digestJSON(req.Inputs)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	rec := store.RunRecord{
		ID:              runID,
		TraceID:         traceID,
		ScenarioName:    req.Scenario.Name,
		ScenarioVersion: req.Scenario.ScenarioVersion,
		PolicyProfile:   policy,
		Status:          "running",
		SpaceID:         firstNonEmpty(req.SpaceID, "local"),
		ActorRole:       firstNonEmpty(req.ActorRole, "maintainer"),
		InputsDigest:    inputsDigest,
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	repoRoot := ""
	if req.Repo != nil {
		rec.RepoRoot = req.Repo.Root
		repoRoot = req.Repo.Root
	}
	if repoRoot == "" {
		if v, ok := req.Inputs["repoRoot"].(string); ok {
			repoRoot = v
			rec.RepoRoot = v
		}
	}
	if repoRoot != "" {
		if abs, err := rag.AbsRepoRoot(repoRoot); err == nil {
			repoRoot = abs
			rec.RepoRoot = abs
		}
	}

	if err := s.gdb().Create(&rec).Error; err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	runDir := s.db.RunDir(runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir run dir: %w", err)
	}

	meta := RunMeta{
		RunID:         runID,
		TraceID:       traceID,
		Scenario:      req.Scenario,
		Inputs:        req.Inputs,
		PolicyProfile: policy,
		ActorRole:     rec.ActorRole,
		Repo:          req.Repo,
		SourceRunID:   opts.sourceRunID,
		ReplayMode:    opts.replayMode,
	}
	if err := saveRunMeta(runDir, meta); err != nil {
		return nil, err
	}

	startedPayload := map[string]any{
		"scenario": map[string]string{
			"name":            req.Scenario.Name,
			"scenarioVersion": req.Scenario.ScenarioVersion,
		},
		"policyProfile": policy,
		"inputsDigest":  inputsDigest,
		"repo":          req.Repo,
	}
	if opts.sourceRunID != "" {
		startedPayload["sourceRunId"] = opts.sourceRunID
		startedPayload["replayMode"] = opts.replayMode
	}

	if _, err := s.eventsFor().Append(runID, traceID, "run.started", "info", startedPayload); err != nil {
		return nil, err
	}
	_ = s.writeAudit(runID, traceID, "run.started", startedPayload)

	if err := s.executeSteps(&rec, req, doc, eng, now); err != nil {
		resp := &CreateResponse{RunID: runID, TraceID: traceID}
		if errors.Is(err, ErrWaitingApproval) {
			return resp, nil
		}
		return resp, err
	}
	return &CreateResponse{RunID: runID, TraceID: traceID}, nil
}

func (s *Service) CreateWithOptions(req CreateRequest, opts createOptions) (*CreateResponse, error) {
	return s.createAndExecute(req, opts)
}

func (s *Service) executeSteps(rec *store.RunRecord, req CreateRequest, doc *rules.Document, eng *rules.Engine, started time.Time) error {
	runID := rec.ID
	traceID := rec.TraceID
	runDir := s.db.RunDir(runID)
	repoRoot := rec.RepoRoot

	toolCtx := toolbus.Context{
		RunID:    runID,
		TraceID:  traceID,
		RepoRoot: repoRoot,
		RunDir:   runDir,
		Inputs:   req.Inputs,
	}

	lastToolStep := struct{ id, role string }{"ship.release", "Shipper"}
	agentTaskID := ""
	issue := inputString(req.Inputs, "issueOrSpec")
	evidenceRefs := s.prepareExecutionContext(runID, traceID, rec.SpaceID, repoRoot, issue)

	for idx, step := range doc.Scenario.Steps {
		for _, gate := range eng.GatesBeforeStep(step.ID) {
			if denied, reason := s.evaluateGate(toolCtx, gate); denied {
				_, _ = s.eventsFor().Append(runID, traceID, "policy.denied", "warn", map[string]any{
					"target": "gate", "reason": reason, "action": "deny", "ref": gate.ID,
				})
				if gate.Blocking {
					_, err := s.failRun(rec, runID, traceID, started, "GATE_BLOCKED", reason)
					return err
				}
			}
		}

		stepStart := time.Now().UTC()
		stepRow := s.startStep(runID, step, idx, stepStart)
		if _, err := s.eventsFor().Append(runID, traceID, "step.started", "info", map[string]any{
			"stepId": step.ID, "role": step.Role, "kind": step.Kind,
		}); err != nil {
			return err
		}
		if step.RAG != nil {
			refs, err := s.retrieveStepEvidence(runID, traceID, rec.SpaceID, repoRoot, issue, step, req.Inputs)
			if err != nil {
				if isCitationHumanConfirm(step) {
					rec.Status = "waiting_approval"
					rec.UpdatedAt = time.Now().UTC()
					_ = s.gdb().Save(rec).Error
					s.finishStep(stepRow, "waiting_approval", stepStart, "GATE_CITATION_MISSING", err.Error())
					_, _ = s.eventsFor().Append(runID, traceID, "gate.waiting_approval", "warn", map[string]any{
						"stepId": step.ID, "reason": err.Error(), "gate": "citation",
					})
					s.requestApproval(rec, stepRow, "citation", "", err.Error(), map[string]any{
						"stepId": step.ID, "errorCode": "GATE_CITATION_MISSING",
					})
					return ErrWaitingApproval
				}
				s.finishStep(stepRow, "failed", stepStart, "GATE_CITATION_MISSING", err.Error())
				_, ferr := s.failRun(rec, runID, traceID, started, "GATE_CITATION_MISSING", err.Error())
				return ferr
			}
			evidenceRefs = appendUnique(evidenceRefs, refs...)
		}

		switch step.Kind {
		case "agent":
			lastToolStep.id = step.ID
			lastToolStep.role = step.Role
			res, err := s.executeAgentStep(runID, traceID, runDir, repoRoot, issue, step, req.Inputs)
			if res != nil {
				agentTaskID = firstNonEmpty(res.ExecGoTaskID, res.TaskID, res.ActionID)
			}
			if err != nil {
				code := agentErrorCode(err)
				s.finishStep(stepRow, "failed", stepStart, code, err.Error())
				_, ferr := s.failRun(rec, runID, traceID, started, code, err.Error())
				return ferr
			}
			evidenceRefs = append(evidenceRefs, s.captureDiffEvidence(runID, traceID, runDir, repoRoot)...)
		case "tool_chain":
			lastToolStep.id = step.ID
			lastToolStep.role = step.Role
			for _, item := range step.Chain {
				if denied, reason := s.scenarioToolDenied(rec, item.Tool); denied {
					_, _ = s.eventsFor().Append(runID, traceID, "policy.denied", "warn", map[string]any{
						"target": "tool", "reason": reason, "action": "deny", "ref": item.Tool,
						"matrix": "scenario", "actorRole": rec.ActorRole,
					})
					s.finishStep(stepRow, "failed", stepStart, "POLICY_DENIED", reason)
					_, ferr := s.failRun(rec, runID, traceID, started, "POLICY_DENIED", reason)
					return ferr
				}
				risk := string(s.tools.ToolRisk(item.Tool))
				if !s.dangerousToolAllowed(req.Inputs, step.ID, item, risk) {
					msg := fmt.Sprintf("tool %s has danger risk and requires human approval or policy allow_dangerous", item.Tool)
					rec.Status = "waiting_approval"
					rec.UpdatedAt = time.Now().UTC()
					_ = s.gdb().Save(rec).Error
					s.finishStep(stepRow, "waiting_approval", stepStart, "TOOL_DANGEROUS_APPROVAL_REQUIRED", msg)
					_, _ = s.eventsFor().Append(runID, traceID, "gate.waiting_approval", "warn", map[string]any{
						"stepId": step.ID, "gate": "tool_risk", "tool": item.Tool, "risk": risk, "reason": msg,
					})
					_, _ = s.eventsFor().Append(runID, traceID, "policy.denied", "warn", map[string]any{
						"target": "tool", "reason": msg, "action": "require_approval", "ref": item.Tool,
					})
					_ = s.writeAudit(runID, traceID, "tool.approval_required", map[string]any{
						"stepId": step.ID, "tool": item.Tool, "risk": risk, "policy": item.Policy,
					})
					s.requestApproval(rec, stepRow, "tool_risk", risk, msg, map[string]any{
						"stepId": step.ID, "tool": item.Tool, "policy": item.Policy,
					})
					return ErrWaitingApproval
				}
				if risk == string(toolbus.RiskDanger) {
					_, _ = s.eventsFor().Append(runID, traceID, "tool.approval_used", "info", map[string]any{
						"stepId": step.ID, "tool": item.Tool, "risk": risk, "policy": item.Policy,
					})
				}
				ctx := map[string]any{"tool": item.Tool, "risk": risk}
				if denied, reason := eng.EvaluateHooks("tool.called", ctx); denied {
					_, _ = s.eventsFor().Append(runID, traceID, "policy.denied", "warn", map[string]any{
						"target": "tool", "reason": reason, "action": "deny", "ref": item.Tool,
					})
					continue
				}
				res := s.callToolWithRetry(runID, traceID, step.ID, risk, toolCtx, item)
				if !res.OK {
					s.finishStep(stepRow, "failed", stepStart, "TOOL_FAILED", res.Error)
					_, ferr := s.failRun(rec, runID, traceID, started, "TOOL_FAILED", res.Error)
					return ferr
				}
			}
		case "llm":
			decision := modelrouter.NewFromEnv().Route(modelrouter.Request{
				RunID: runID, StepID: step.ID, UseCase: step.Role, Prompt: issue,
			})
			usage := modelrouter.UsageRow(decision, modelrouter.Request{
				RunID: runID, StepID: step.ID, UseCase: step.Role,
				InputTokens: decision.InputTokens, OutputTokens: decision.OutputTokens,
			})
			_ = s.gdb().Create(&usage).Error
			_, _ = s.eventsFor().Append(runID, traceID, "model.routed", "info", map[string]any{
				"stepId": step.ID, "provider": decision.Provider.ID, "model": decision.Provider.Model,
				"status": decision.Status, "fallbackUsed": decision.FallbackUsed,
				"inputTokens": decision.InputTokens, "outputTokens": decision.OutputTokens,
				"costMicros": decision.CostMicros,
			})
			if refs := s.writeTemplateStepArtifact(runDir, step, issue, evidenceRefs); len(refs) > 0 {
				evidenceRefs = append(evidenceRefs, refs...)
			}
		case "human":
			if approvedStep(req.Inputs, "_approvedHumanSteps", step.ID) {
				_, _ = s.eventsFor().Append(runID, traceID, "gate.approval_used", "info", map[string]any{
					"stepId": step.ID, "kind": "human", "ref": "approval:" + step.ID,
				})
				evidenceRefs = appendUnique(evidenceRefs, "approval:"+step.ID)
				break
			}
			rec.Status = "waiting_approval"
			rec.UpdatedAt = time.Now().UTC()
			_ = s.gdb().Save(rec).Error
			s.finishStep(stepRow, "waiting_approval", stepStart, "", "")
			_, _ = s.eventsFor().Append(runID, traceID, "gate.waiting_approval", "warn", map[string]any{"stepId": step.ID})
			s.requestApproval(rec, stepRow, "human", "", "human approval required", map[string]any{"stepId": step.ID})
			return ErrWaitingApproval
		}

		if _, err := s.eventsFor().Append(runID, traceID, "step.finished", "info", map[string]any{
			"stepId": step.ID, "ok": true, "durationMs": time.Since(stepStart).Milliseconds(),
		}); err != nil {
			return err
		}
		s.finishStep(stepRow, "finished", stepStart, "", "")

		ckptID, snapshotDigest, checkpointURI := s.saveCheckpoint(runID, traceID, step.ID, runDir, checkpointStrategy(doc))
		if _, err := s.eventsFor().Append(runID, traceID, "run.checkpoint_saved", "info", map[string]any{
			"checkpointId": ckptID, "stepId": step.ID,
			"snapshotDigest": snapshotDigest, "strategy": checkpointStrategy(doc), "uri": checkpointURI,
		}); err != nil {
			return err
		}
	}

	lastSeq := s.lastEventSeq(runID)
	manifest, err := artifacts.WriteBundle(runDir, artifacts.BundleMeta{
		RunID: runID, ScenarioName: req.Scenario.Name,
		ScenarioVersion: req.Scenario.ScenarioVersion,
		StepID:          lastToolStep.id, Role: lastToolStep.role,
		RepoRoot: repoRoot, Issue: issue,
		EventRange:  fmt.Sprintf("run_events:seq=1..%d", lastSeq),
		AgentTaskID: agentTaskID, EvidenceRefs: evidenceRefs,
	})
	if err != nil {
		_, ferr := s.failRun(rec, runID, traceID, started, "ARTIFACT_WRITE_FAILED", err.Error())
		return ferr
	}
	if err := artifacts.ValidateQuality(runDir, manifest, s.AgentAdapter() != "static"); err != nil {
		_, _ = s.eventsFor().Append(runID, traceID, "artifact.quality_failed", "error", map[string]any{
			"error": err.Error(),
		})
		s.recordQualityMetric(runID, rec.SpaceID, "artifact_quality_passed", 0, "bool")
		s.recordQualityMetric(runID, rec.SpaceID, "artifact_quality_failed_total", 1, "count")
		_, ferr := s.failRun(rec, runID, traceID, started, "ARTIFACT_QUALITY_FAILED", err.Error())
		return ferr
	}
	_ = s.indexArtifacts(runID, lastToolStep.id, runDir, manifest)

	for _, gate := range eng.GatesBeforeFinish() {
		if gate.Check.Artifact != "" {
			found := false
			for _, a := range manifest.Artifacts {
				if a.Type == gate.Check.Artifact {
					found = true
					break
				}
			}
			if !found && gate.Blocking {
				_, ferr := s.failRun(rec, runID, traceID, started, "GATE_ARTIFACT_MISSING", gate.Check.Artifact)
				return ferr
			}
		}
	}

	finished := time.Now().UTC()
	rec.Status = "finished"
	rec.FinishedAt = &finished
	rec.UpdatedAt = finished
	if err := s.gdb().Save(rec).Error; err != nil {
		return err
	}

	artifactRefs := make([]map[string]any, 0, len(manifest.Artifacts))
	for _, a := range manifest.Artifacts {
		artifactRefs = append(artifactRefs, map[string]any{"type": a.Type, "digest": a.Digest, "uri": a.URI})
	}
	s.recordQualityMetrics(runID, traceID, rec.SpaceID, len(doc.Scenario.Steps), len(manifest.Artifacts), rec.Recovered)

	_, err = s.eventsFor().Append(runID, traceID, "run.finished", "info", map[string]any{
		"ok": true, "durationMs": finished.Sub(started).Milliseconds(),
		"artifacts": artifactRefs,
		"metrics":   map[string]any{"recovered": rec.Recovered, "steps": len(doc.Scenario.Steps)},
	})
	_ = s.writeAudit(runID, traceID, "run.finished", map[string]any{"artifacts": artifactRefs})
	return err
}

func (s *Service) prepareExecutionContext(runID, traceID, spaceID, repoRoot, issue string) []string {
	var refs []string
	if repoRoot != "" {
		if resp, err := s.rag.Index(ragIndexRequest(spaceID, repoRoot)); err == nil {
			_, _ = s.eventsFor().Append(runID, traceID, "rag.indexed", "info", map[string]any{
				"repoRoot": repoRoot, "documents": resp.Documents, "chunks": resp.Chunks,
			})
		} else {
			_, _ = s.eventsFor().Append(runID, traceID, "rag.index_failed", "warn", map[string]any{
				"repoRoot": repoRoot, "error": err.Error(),
			})
		}
		if hits, err := s.rag.Query(ragQueryRequest(spaceID, repoRoot, issue)); err == nil {
			for _, hit := range hits.Items {
				refs = append(refs, hit.Ref)
			}
			_, _ = s.eventsFor().Append(runID, traceID, "rag.retrieved", "info", map[string]any{
				"query": issue, "hits": len(hits.Items), "refs": refs,
			})
		}
	}
	if strings.TrimSpace(issue) != "" {
		memories, err := s.queryExecutionMemory(spaceID, repoRoot, issue, 5)
		if err != nil {
			_, _ = s.eventsFor().Append(runID, traceID, "memory.query_failed", "warn", map[string]any{
				"error": err.Error(),
			})
			return refs
		}
		if len(memories) > 0 {
			recordIDs := make([]string, 0, len(memories))
			memoryRefs := make([]string, 0, len(memories))
			for _, mem := range memories {
				recordIDs = append(recordIDs, mem.ID)
				memoryRefs = append(memoryRefs, "memory:"+mem.ID)
			}
			refs = appendUnique(refs, memoryRefs...)
			_, _ = s.eventsFor().Append(runID, traceID, "memory.injected", "info", map[string]any{
				"count": len(recordIDs), "recordIds": recordIDs,
			})
			_, _ = s.eventsFor().Append(runID, traceID, "memory.hit_used", "info", map[string]any{
				"recordIds": recordIDs, "count": len(recordIDs),
			})
			_ = s.writeAudit(runID, traceID, "memory.hit_used", map[string]any{
				"recordIds": recordIDs, "actorId": "ash-runner",
			})
		}
	}
	return refs
}

func (s *Service) queryExecutionMemory(spaceID, repoRoot, issue string, limit int) ([]store.MemoryRecord, error) {
	if limit <= 0 {
		limit = 5
	}
	like := "%" + strings.ToLower(strings.TrimSpace(issue)) + "%"
	q := s.gdb().Where("status = ? AND layer = ? AND space_id = ?", "approved", "L1", firstNonEmpty(spaceID, "local")).
		Where("LOWER(title) LIKE ? OR LOWER(body) LIKE ?", like, like)
	if repoRoot != "" {
		q = q.Where("scope_repo = ? OR scope_repo = ?", repoRoot, "")
	}
	var rows []store.MemoryRecord
	if err := q.Order("updated_at desc").Limit(limit * 3).Find(&rows).Error; err != nil {
		return nil, err
	}
	var edges []store.MemoryEdge
	if len(rows) > 0 {
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		if err := s.gdb().Where("from_id IN ? AND kind = ?", ids, "duplicate").Find(&edges).Error; err != nil {
			return nil, err
		}
	}
	duplicated := map[string]bool{}
	for _, edge := range edges {
		duplicated[edge.FromID] = true
	}
	now := time.Now().UTC()
	out := make([]store.MemoryRecord, 0, limit)
	for _, row := range rows {
		if duplicated[row.ID] || runMemoryExpired(row, now) {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func runMemoryExpired(row store.MemoryRecord, now time.Time) bool {
	if row.TTLDays == nil || *row.TTLDays <= 0 {
		return false
	}
	return row.CreatedAt.Add(time.Duration(*row.TTLDays) * 24 * time.Hour).Before(now)
}

func (s *Service) retrieveStepEvidence(runID, traceID, spaceID, repoRoot, issue string, step rules.Step, inputs map[string]any) ([]string, error) {
	if step.RAG == nil {
		return nil, nil
	}
	if approvedStep(inputs, "_approvedCitationSteps", step.ID) {
		ref := "approval:" + step.ID
		_, _ = s.eventsFor().Append(runID, traceID, "citation.approved_without_evidence", "warn", map[string]any{
			"stepId": step.ID, "ref": ref,
		})
		return []string{ref}, nil
	}
	if repoRoot == "" {
		if step.RAG.RequireCitations {
			msg := fmt.Sprintf("step %s requires citations but repoRoot is empty", step.ID)
			_, _ = s.eventsFor().Append(runID, traceID, "citation.missing", "warn", map[string]any{"stepId": step.ID, "reason": msg})
			return nil, errors.New(msg)
		}
		return nil, nil
	}
	resp, err := s.rag.Query(rag.QueryRequest{RepoRoot: repoRoot, Text: issue, TopK: 6, SpaceID: firstNonEmpty(spaceID, "local")})
	if err != nil {
		if step.RAG.RequireCitations {
			msg := fmt.Sprintf("step %s citation query failed: %v", step.ID, err)
			_, _ = s.eventsFor().Append(runID, traceID, "citation.missing", "warn", map[string]any{"stepId": step.ID, "reason": msg})
			return nil, errors.New(msg)
		}
		return nil, nil
	}
	refs := make([]string, 0, len(resp.Items))
	for _, hit := range resp.Items {
		refs = append(refs, hit.Ref)
	}
	if len(refs) == 0 && step.RAG.RequireCitations {
		msg := fmt.Sprintf("step %s requires citations but no repo evidence matched", step.ID)
		_, _ = s.eventsFor().Append(runID, traceID, "citation.missing", "warn", map[string]any{"stepId": step.ID, "reason": msg})
		return nil, errors.New(msg)
	}
	_, _ = s.eventsFor().Append(runID, traceID, "citation.bound", "info", map[string]any{
		"stepId": step.ID, "required": step.RAG.RequireCitations, "refs": refs,
	})
	return refs, nil
}

func (s *Service) startStep(runID string, step rules.Step, order int, started time.Time) *store.RunStep {
	row := &store.RunStep{
		ID: "step_" + uuid.NewString(), RunID: runID, StepID: step.ID, StepOrder: order,
		Role: step.Role, Kind: step.Kind, Status: "running", StartedAt: &started,
		CreatedAt: started, UpdatedAt: started,
	}
	_ = s.gdb().Create(row).Error
	return row
}

func (s *Service) finishStep(row *store.RunStep, status string, started time.Time, code, msg string) {
	if row == nil || row.ID == "" {
		return
	}
	now := time.Now().UTC()
	row.Status = status
	row.FinishedAt = &now
	row.DurationMs = now.Sub(started).Milliseconds()
	row.ErrorCode = code
	row.ErrorMessage = msg
	row.UpdatedAt = now
	_ = s.gdb().Save(row).Error
}

func (s *Service) executeAgentStep(runID, traceID, runDir, repoRoot, issue string, step rules.Step, inputs map[string]any) (*agentexec.Result, error) {
	timeout := step.TimeoutMs
	if timeout <= 0 {
		timeout = 120000
	}
	prompt := ""
	if step.Agent != nil {
		prompt = step.Agent.Prompt
	}
	req := agentexec.Request{
		RunID: runID, TraceID: traceID, StepID: step.ID, Role: step.Role,
		RepoRoot: repoRoot, RunDir: runDir, Issue: issue, Prompt: prompt,
		Inputs: inputs, TimeoutMs: timeout,
	}
	now := time.Now().UTC()
	task := store.AgentTask{
		ID: "agt_" + uuid.NewString(), RunID: runID, TraceID: traceID, StepID: step.ID,
		Adapter: "codex", AgentID: "ash-codex", SessionID: runID,
		Status: "running", PromptDigest: digestString(issue + "\n" + prompt),
		TimeoutMs: timeout, CreatedAt: now, StartedAt: &now,
	}
	_ = s.gdb().Create(&task).Error
	_, _ = s.eventsFor().Append(runID, traceID, "agent.called", "info", map[string]any{
		"stepId": step.ID, "adapter": "codex", "timeoutMs": timeout,
	})
	res, err := s.agent.Execute(context.Background(), req)
	finished := time.Now().UTC()
	task.CompletedAt = &finished
	task.DurationMs = finished.Sub(now).Milliseconds()
	if res != nil {
		task.ExecGoTaskID = res.ExecGoTaskID
		task.ActionID = res.ActionID
		task.AgentID = firstNonEmpty(res.AgentID, task.AgentID)
		task.SessionID = firstNonEmpty(res.SessionID, task.SessionID)
		task.StdoutSummary = res.StdoutSummary
		task.StderrSummary = res.StderrSummary
		task.ExitCode = res.ExitCode
		task.Status = res.Status
	}
	if err != nil {
		task.Status = "failed"
		task.ErrorCode = agentErrorCode(err)
		task.ErrorMessage = err.Error()
		_ = s.gdb().Save(&task).Error
		_, _ = s.eventsFor().Append(runID, traceID, "agent.failed", "error", map[string]any{
			"stepId": step.ID, "taskId": task.ExecGoTaskID, "error": err.Error(),
		})
		return res, err
	}
	if task.Status == "" {
		task.Status = "success"
	}
	_ = s.gdb().Save(&task).Error
	_, _ = s.eventsFor().Append(runID, traceID, "agent.finished", "info", map[string]any{
		"stepId": step.ID, "taskId": firstNonEmpty(task.ExecGoTaskID, task.ActionID, task.ID),
		"status": task.Status, "durationMs": task.DurationMs,
	})
	return res, nil
}

func agentErrorCode(err error) string {
	switch {
	case errors.Is(err, agentexec.ErrBridgeUnavailable):
		return "AGENT_BRIDGE_UNAVAILABLE"
	case errors.Is(err, agentexec.ErrAgentOutputInvalid):
		return "AGENT_OUTPUT_INVALID"
	case errors.Is(err, agentexec.ErrAgentTaskFailed):
		return "AGENT_TASK_FAILED"
	default:
		return "AGENT_EXECUTION_FAILED"
	}
}

func (s *Service) callToolWithRetry(runID, traceID, stepID, risk string, ctx toolbus.Context, item rules.ToolChainItem) toolbus.Result {
	attempts := retryAttempts(item.Retry)
	backoff := retryBackoff(item.Retry)
	var last toolbus.Result
	for attempt := 1; attempt <= attempts; attempt++ {
		timeout := toolTimeoutMs(item)
		_, _ = s.eventsFor().Append(runID, traceID, "tool.called", "info", map[string]any{
			"tool": item.Tool, "risk": risk, "timeoutMs": timeout,
			"attempt": attempt, "maxAttempts": attempts,
			"argsDigest": digestString(fmt.Sprintf("%v", item.Args)),
		})
		last = s.callTool(runID, traceID, stepID, ctx, item, attempt)
		failureClass := toolFailureClass(last, timeout)
		severity := "info"
		if !last.OK {
			severity = "warn"
		}
		_, _ = s.eventsFor().Append(runID, traceID, "tool.result", severity, map[string]any{
			"tool": item.Tool, "ok": last.OK, "durationMs": last.DurationMs,
			"attempt": attempt, "maxAttempts": attempts, "failureClass": failureClass,
			"output": last.Output, "error": last.Error,
		})
		if last.OK || attempt == attempts {
			return last
		}
		_, _ = s.eventsFor().Append(runID, traceID, "tool.retry_scheduled", "warn", map[string]any{
			"tool": item.Tool, "attempt": attempt + 1, "maxAttempts": attempts,
			"backoffMs": backoff, "failureClass": failureClass,
		})
		if backoff > 0 {
			time.Sleep(time.Duration(backoff) * time.Millisecond)
		}
	}
	return last
}

func (s *Service) callTool(runID, traceID, stepID string, ctx toolbus.Context, item rules.ToolChainItem, attempt int) toolbus.Result {
	timeout := item.TimeoutMs
	if timeout <= 0 {
		timeout = 30000
	}
	risk := string(s.tools.ToolRisk(item.Tool))
	start := time.Now().UTC()
	row := store.ToolCall{
		ID: "tool_" + uuid.NewString(), RunID: runID, TraceID: traceID, StepID: stepID,
		Tool: item.Tool, Risk: risk, Status: "running",
		ArgsDigest: digestString(fmt.Sprintf("%v", item.Args)), TimeoutMs: timeout,
		Attempt: attempt, CreatedAt: start,
	}
	_ = s.gdb().Create(&row).Error
	res := s.tools.Call(ctx, toolbus.CallRequest{Tool: item.Tool, Args: item.Args})
	if timeout > 0 && res.DurationMs > timeout {
		res.OK = false
		res.Error = fmt.Sprintf("tool %s exceeded timeout %dms", item.Tool, timeout)
	}
	finished := time.Now().UTC()
	row.CompletedAt = &finished
	row.DurationMs = res.DurationMs
	if res.OK {
		row.Status = "success"
	} else {
		row.Status = "failed"
		row.Error = res.Error
	}
	if b, err := json.Marshal(res.Output); err == nil {
		row.OutputJSON = string(b)
	}
	_ = s.gdb().Save(&row).Error
	_ = s.writeAudit(runID, traceID, "tool."+row.Status, row)
	return res
}

func retryAttempts(spec *rules.RetrySpec) int {
	if spec == nil || spec.MaxAttempts <= 1 {
		return 1
	}
	if spec.MaxAttempts > 5 {
		return 5
	}
	return spec.MaxAttempts
}

func retryBackoff(spec *rules.RetrySpec) int {
	if spec == nil || spec.BackoffMs <= 0 {
		return 0
	}
	if spec.BackoffMs > 30000 {
		return 30000
	}
	return spec.BackoffMs
}

func toolTimeoutMs(item rules.ToolChainItem) int64 {
	if item.TimeoutMs > 0 {
		return item.TimeoutMs
	}
	return 30000
}

func toolFailureClass(res toolbus.Result, timeoutMs int64) string {
	if res.FailureClass != "" {
		return res.FailureClass
	}
	if timeoutMs > 0 && res.DurationMs > timeoutMs {
		return "timeout"
	}
	if res.OK {
		return ""
	}
	if strings.HasPrefix(res.Error, "unknown tool:") {
		return "unknown_tool"
	}
	return "error"
}

func (s *Service) scenarioToolDenied(rec *store.RunRecord, tool string) (bool, string) {
	policy, err := authz.LoadScenarioPolicy(s.db, rec.SpaceID, rec.ScenarioName, rec.ScenarioVersion)
	if err != nil {
		return false, ""
	}
	ok, reason := authz.EvaluateScenarioTool(policy, rec.ActorRole, tool)
	return !ok, reason
}

func (s *Service) dangerousToolAllowed(inputs map[string]any, stepID string, item rules.ToolChainItem, risk string) bool {
	if risk != string(toolbus.RiskDanger) {
		return true
	}
	if explicitDangerousToolPolicy(item.Policy) {
		return true
	}
	return approvedStep(inputs, "_approvedDangerousToolSteps", stepID)
}

func explicitDangerousToolPolicy(policy string) bool {
	switch strings.TrimSpace(strings.ToLower(policy)) {
	case "allow_dangerous", "allow-dangerous", "dangerous_allowed", "dangerous-allowed":
		return true
	default:
		return false
	}
}

func (s *Service) captureDiffEvidence(runID, traceID, runDir, repoRoot string) []string {
	if repoRoot == "" {
		return nil
	}
	res := s.tools.Call(toolbus.Context{RunID: runID, TraceID: traceID, RepoRoot: repoRoot, RunDir: runDir}, toolbus.CallRequest{Tool: "git.diff"})
	diff, _ := res.Output["diff"].(string)
	if strings.TrimSpace(diff) == "" {
		return nil
	}
	artDir := filepath.Join(runDir, "artifacts")
	_ = os.MkdirAll(artDir, 0o755)
	path := filepath.Join(artDir, "diff.patch")
	_ = os.WriteFile(path, []byte(strings.ReplaceAll(diff, "\r\n", "\n")), 0o644)
	ref := "artifact:diff.patch"
	_, _ = s.eventsFor().Append(runID, traceID, "artifact.captured", "info", map[string]any{"type": "diff", "ref": ref})
	return []string{ref}
}

func (s *Service) writeTemplateStepArtifact(runDir string, step rules.Step, issue string, refs []string) []string {
	artDir := filepath.Join(runDir, "artifacts")
	_ = os.MkdirAll(artDir, 0o755)
	name := strings.ReplaceAll(step.ID, ".", "_") + ".md"
	path := filepath.Join(artDir, name)
	var b strings.Builder
	b.WriteString("# " + step.Role + " step\n\n")
	b.WriteString("- Step: `" + step.ID + "`\n")
	if issue != "" {
		b.WriteString("- Issue/spec: " + issue + "\n")
	}
	if len(refs) > 0 {
		b.WriteString("\n## Evidence\n\n")
		for _, ref := range refs {
			b.WriteString("- `" + ref + "`\n")
		}
	}
	_ = os.WriteFile(path, []byte(b.String()), 0o644)
	return []string{"artifact:" + name}
}

func (s *Service) saveCheckpoint(runID, traceID, stepID, runDir, strategy string) (string, string, string) {
	ckptID := "ckpt_" + uuid.NewString()
	snapshotDigest := digestString(runID + ":" + stepID + ":" + time.Now().UTC().Format(time.RFC3339Nano))
	rel := filepath.Join("checkpoints", ckptID+".json")
	abs := filepath.Join(runDir, rel)
	_ = os.MkdirAll(filepath.Dir(abs), 0o755)
	payload := map[string]any{"runId": runID, "stepId": stepID, "snapshotDigest": snapshotDigest}
	b, _ := json.MarshalIndent(payload, "", "  ")
	_ = os.WriteFile(abs, append(b, '\n'), 0o644)
	uri := filepath.ToSlash(rel)
	storeKey := ""
	contentType := "application/json"
	sizeBytes := int64(len(b) + 1)
	if s.artifacts != nil {
		if f, err := os.Open(abs); err == nil {
			ref, putErr := s.artifacts.Put(context.Background(), filepath.ToSlash(filepath.Join("runs", runID, "checkpoints", ckptID+".json")), f, contentType)
			_ = f.Close()
			if putErr == nil && ref != nil {
				uri = ref.URI
				storeKey = ref.Key
				sizeBytes = ref.SizeBytes
				_, _ = s.eventsFor().Append(runID, traceID, "checkpoint.stored", "info", map[string]any{
					"checkpointId": ckptID, "stepId": stepID, "uri": uri, "sizeBytes": sizeBytes,
				})
			} else if putErr != nil {
				_, _ = s.eventsFor().Append(runID, traceID, "checkpoint.store_failed", "warn", map[string]any{
					"checkpointId": ckptID, "stepId": stepID, "error": putErr.Error(),
				})
			}
		}
	}
	_ = s.gdb().Create(&store.Checkpoint{
		ID: ckptID, RunID: runID, StepID: stepID, SnapshotDigest: snapshotDigest,
		URI: uri, StoreKey: storeKey, ContentType: contentType, SizeBytes: sizeBytes,
		Strategy: strategy, CreatedAt: time.Now().UTC(),
	}).Error
	return ckptID, snapshotDigest, uri
}

func (s *Service) indexArtifacts(runID, stepID, runDir string, manifest *artifacts.Manifest) error {
	now := time.Now().UTC()
	for i := range manifest.Artifacts {
		a := &manifest.Artifacts[i]
		uri := a.URI
		storeKey := a.StoreKey
		if s.artifacts != nil {
			localPath := filepath.Join(runDir, filepath.FromSlash(a.URI))
			if f, err := os.Open(localPath); err == nil {
				ref, putErr := s.artifacts.Put(context.Background(), filepath.ToSlash(filepath.Join("runs", runID, a.Name)), f, a.ContentType)
				_ = f.Close()
				if putErr == nil && ref != nil {
					uri = ref.URI
					storeKey = ref.Key
					a.URI = ref.URI
					a.StoreKey = ref.Key
					if ref.ContentType != "" {
						a.ContentType = ref.ContentType
					}
					if ref.SizeBytes > 0 {
						a.SizeBytes = ref.SizeBytes
					}
					_, _ = s.eventsFor().Append(runID, "", "artifact.stored", "info", map[string]any{
						"name": a.Name, "type": a.Type, "uri": uri, "storeKey": storeKey, "sizeBytes": a.SizeBytes,
					})
				} else if putErr != nil {
					_, _ = s.eventsFor().Append(runID, "", "artifact.store_failed", "warn", map[string]any{
						"name": a.Name, "type": a.Type, "error": putErr.Error(),
					})
				}
			}
		}
		row := store.ArtifactIndex{
			ID: "artidx_" + uuid.NewString(), RunID: runID, StepID: stepID, Type: a.Type,
			Name: a.Name, URI: uri, StoreKey: storeKey, Digest: a.Digest, ContentType: a.ContentType,
			SizeBytes: a.SizeBytes, CreatedAt: now,
		}
		if producer := a.Producer; producer != nil {
			if v, ok := producer["eventRange"].(string); ok {
				row.EventRange = v
			}
		}
		_ = s.gdb().Create(&row).Error
	}
	return artifacts.SaveManifest(runDir, manifest)
}

func (s *Service) lastEventSeq(runID string) int64 {
	var ev store.RunEvent
	if err := s.gdb().Where("run_id = ?", runID).Order("seq desc").First(&ev).Error; err != nil {
		return 0
	}
	return ev.Seq
}

func (s *Service) writeAudit(runID, traceID, eventType string, payload any) error {
	spaceID := "local"
	if runID != "" {
		var rec store.RunRecord
		if err := s.gdb().Select("space_id", "trace_id").First(&rec, "id = ?", runID).Error; err == nil {
			spaceID = firstNonEmpty(rec.SpaceID, "local")
			traceID = firstNonEmpty(traceID, rec.TraceID)
		}
	}
	b, _ := json.Marshal(payload)
	return s.gdb().Create(&store.AuditLog{
		ID: "aud_" + uuid.NewString(), SpaceID: spaceID, RunID: runID, TraceID: traceID,
		EventType: eventType, PayloadJSON: string(b), CreatedAt: time.Now().UTC(),
	}).Error
}

func (s *Service) recordQualityMetrics(runID, traceID, spaceID string, scenarioSteps, artifactCount int, recovered bool) {
	now := time.Now().UTC()
	spaceID = firstNonEmpty(spaceID, "local")
	metrics := []store.QualityMetric{
		{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "steps_total", Value: float64(scenarioSteps), Unit: "count", CreatedAt: now},
		{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "artifacts_total", Value: float64(artifactCount), Unit: "count", CreatedAt: now},
		{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "artifact_quality_passed", Value: 1, Unit: "bool", CreatedAt: now},
		{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "artifact_quality_failed_total", Value: 0, Unit: "count", CreatedAt: now},
	}
	if recovered {
		metrics = append(metrics, store.QualityMetric{
			ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "run_recovered", Value: 1, Unit: "bool", CreatedAt: now,
		})
	}

	var toolTotal, toolFailed int64
	_ = s.gdb().Model(&store.ToolCall{}).Where("run_id = ?", runID).Count(&toolTotal).Error
	_ = s.gdb().Model(&store.ToolCall{}).Where("run_id = ? AND status = ?", runID, "failed").Count(&toolFailed).Error
	metrics = append(metrics,
		store.QualityMetric{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "tool_calls_total", Value: float64(toolTotal), Unit: "count", CreatedAt: now},
		store.QualityMetric{
			ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "tool_failure_rate",
			Value: ratio(toolFailed, toolTotal), Unit: "ratio", CreatedAt: now,
		},
	)

	var agentTotal, agentFailed int64
	_ = s.gdb().Model(&store.AgentTask{}).Where("run_id = ?", runID).Count(&agentTotal).Error
	_ = s.gdb().Model(&store.AgentTask{}).Where("run_id = ? AND status = ?", runID, "failed").Count(&agentFailed).Error
	metrics = append(metrics,
		store.QualityMetric{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "agent_tasks_total", Value: float64(agentTotal), Unit: "count", CreatedAt: now},
		store.QualityMetric{
			ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "agent_failure_rate",
			Value: ratio(agentFailed, agentTotal), Unit: "ratio", CreatedAt: now,
		},
	)

	var modelCostMicros int64
	_ = s.gdb().Model(&store.ModelUsage{}).Where("run_id = ?", runID).Select("COALESCE(SUM(cost_micros), 0)").Scan(&modelCostMicros).Error
	metrics = append(metrics, store.QualityMetric{
		ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "model_cost_micros_total",
		Value: float64(modelCostMicros), Unit: "micros", CreatedAt: now,
	})

	var citationBound, citationMissing int64
	_ = s.gdb().Model(&store.RunEvent{}).Where("run_id = ? AND type = ?", runID, "citation.bound").Count(&citationBound).Error
	_ = s.gdb().Model(&store.RunEvent{}).Where("run_id = ? AND type = ?", runID, "citation.missing").Count(&citationMissing).Error
	metrics = append(metrics,
		store.QualityMetric{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "citation_bound_total", Value: float64(citationBound), Unit: "count", CreatedAt: now},
		store.QualityMetric{ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "citation_missing_total", Value: float64(citationMissing), Unit: "count", CreatedAt: now},
		store.QualityMetric{
			ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID, Name: "citation_hit_rate",
			Value: ratio(citationBound, citationBound+citationMissing), Unit: "ratio", CreatedAt: now,
		},
	)

	for _, metric := range metrics {
		_ = s.gdb().Create(&metric).Error
	}
	_, _ = s.eventsFor().Append(runID, traceID, "quality.metrics_recorded", "info", map[string]any{
		"count": len(metrics),
		"names": metricNames(metrics),
	})
}

func (s *Service) recordQualityMetric(runID, spaceID, name string, value float64, unit string) {
	spaceID = firstNonEmpty(spaceID, "local")
	_ = s.gdb().Create(&store.QualityMetric{
		ID: "qm_" + uuid.NewString(), RunID: runID, SpaceID: spaceID,
		Name: name, Value: value, Unit: unit, CreatedAt: time.Now().UTC(),
	}).Error
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func metricNames(metrics []store.QualityMetric) []string {
	names := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		names = append(names, metric.Name)
	}
	return names
}

func inputString(inputs map[string]any, key string) string {
	if v, ok := inputs[key].(string); ok {
		return v
	}
	return ""
}

func appendUnique(base []string, values ...string) []string {
	seen := make(map[string]bool, len(base)+len(values))
	for _, item := range base {
		seen[item] = true
	}
	for _, item := range values {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		base = append(base, item)
	}
	return base
}

func appendApprovedStep(inputs map[string]any, key, stepID string) {
	if inputs == nil || stepID == "" {
		return
	}
	steps := approvedStepList(inputs[key])
	for _, item := range steps {
		if item == stepID {
			inputs[key] = steps
			return
		}
	}
	inputs[key] = append(steps, stepID)
}

func approvedStep(inputs map[string]any, key, stepID string) bool {
	if inputs == nil || stepID == "" {
		return false
	}
	for _, item := range approvedStepList(inputs[key]) {
		if item == stepID {
			return true
		}
	}
	return false
}

func approvedStepList(v any) []string {
	switch items := v.(type) {
	case []string:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if str, ok := item.(string); ok && str != "" {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func isCitationHumanConfirm(step rules.Step) bool {
	return step.RAG != nil && step.RAG.OnMissingCitations == "human_confirm"
}

func ragIndexRequest(spaceID, repoRoot string) rag.IndexRequest {
	return rag.IndexRequest{RepoRoot: repoRoot, SpaceID: firstNonEmpty(spaceID, "local")}
}

func ragQueryRequest(spaceID, repoRoot, text string) rag.QueryRequest {
	return rag.QueryRequest{RepoRoot: repoRoot, Text: text, TopK: 6, SpaceID: firstNonEmpty(spaceID, "local")}
}
