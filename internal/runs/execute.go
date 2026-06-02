package runs

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/artifacts"
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

	if err := s.db.Create(&rec).Error; err != nil {
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

	if _, err := s.events.Append(runID, traceID, "run.started", "info", startedPayload); err != nil {
		return nil, err
	}

	if err := s.executeSteps(&rec, req, doc, eng, now); err != nil {
		return nil, err
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

	for _, step := range doc.Scenario.Steps {
		for _, gate := range eng.GatesBeforeStep(step.ID) {
			if denied, reason := s.evaluateGate(toolCtx, gate); denied {
				_, _ = s.events.Append(runID, traceID, "policy.denied", "warn", map[string]any{
					"target": "gate", "reason": reason, "action": "deny", "ref": gate.ID,
				})
				if gate.Blocking {
					_, err := s.failRun(rec, runID, traceID, started, "GATE_BLOCKED", reason)
					return err
				}
			}
		}

		stepStart := time.Now().UTC()
		if _, err := s.events.Append(runID, traceID, "step.started", "info", map[string]any{
			"stepId": step.ID, "role": step.Role, "kind": step.Kind,
		}); err != nil {
			return err
		}

		if step.Kind == "tool_chain" {
			lastToolStep.id = step.ID
			lastToolStep.role = step.Role
			for _, item := range step.Chain {
				risk := string(s.tools.ToolRisk(item.Tool))
				ctx := map[string]any{"tool": item.Tool, "risk": risk}
				if denied, reason := eng.EvaluateHooks("tool.called", ctx); denied {
					_, _ = s.events.Append(runID, traceID, "policy.denied", "warn", map[string]any{
						"target": "tool", "reason": reason, "action": "deny", "ref": item.Tool,
					})
					continue
				}
				_, _ = s.events.Append(runID, traceID, "tool.called", "info", map[string]any{
					"tool": item.Tool, "risk": risk, "timeoutMs": 30000,
					"argsDigest": digestString(fmt.Sprintf("%v", item.Args)),
				})
				res := s.tools.Call(toolCtx, toolbus.CallRequest{Tool: item.Tool, Args: item.Args})
				_, _ = s.events.Append(runID, traceID, "tool.result", "info", map[string]any{
					"tool": item.Tool, "ok": res.OK, "durationMs": res.DurationMs,
					"output": res.Output, "error": res.Error,
				})
			}
		}

		if _, err := s.events.Append(runID, traceID, "step.finished", "info", map[string]any{
			"stepId": step.ID, "ok": true, "durationMs": time.Since(stepStart).Milliseconds(),
		}); err != nil {
			return err
		}

		if _, err := s.events.Append(runID, traceID, "run.checkpoint_saved", "info", map[string]any{
			"checkpointId": "ckpt_" + step.ID, "stepId": step.ID,
			"snapshotDigest": digestString(step.ID), "strategy": checkpointStrategy(doc),
		}); err != nil {
			return err
		}
	}

	manifest, err := artifacts.WriteBundle(runDir, artifacts.BundleMeta{
		RunID: runID, ScenarioName: req.Scenario.Name,
		ScenarioVersion: req.Scenario.ScenarioVersion,
		StepID: lastToolStep.id, Role: lastToolStep.role,
	})
	if err != nil {
		_, ferr := s.failRun(rec, runID, traceID, started, "ARTIFACT_WRITE_FAILED", err.Error())
		return ferr
	}

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
	if err := s.db.Save(rec).Error; err != nil {
		return err
	}

	artifactRefs := make([]map[string]any, 0, len(manifest.Artifacts))
	for _, a := range manifest.Artifacts {
		artifactRefs = append(artifactRefs, map[string]any{"type": a.Type, "digest": a.Digest, "uri": a.URI})
	}

	_, err = s.events.Append(runID, traceID, "run.finished", "info", map[string]any{
		"ok": true, "durationMs": finished.Sub(started).Milliseconds(),
		"artifacts": artifactRefs,
		"metrics":   map[string]any{"recovered": rec.Recovered, "steps": len(doc.Scenario.Steps)},
	})
	return err
}
