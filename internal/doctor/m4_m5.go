package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/evolve"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/harness/loop"
	"github.com/ash-repwiki/ash/internal/improve"
	"github.com/ash-repwiki/ash/internal/sandbox"
	"github.com/ash-repwiki/ash/internal/sandbox/landlock"
	"github.com/ash-repwiki/ash/internal/store"
)

func (s *Service) m4SuiteCases() []CaseResult {
	return []CaseResult{
		s.m4Har01DefaultProfileSchema(),
		s.m4Har02EventInvariant(),
		s.m4Har03ActiveUniqueness(),
		s.m4Sbx01DockerSmokeOrSkip(),
		s.m4Sbx02DangerOffDenied(),
		s.m4Sbx03PathJail(),
		s.m4Sbx04LandlockAvailable(),
		s.m4Acp01TaskSchema(),
		s.m4Acp02ProbeUnconfigured(),
	}
}

func (s *Service) m5SuiteCases() []CaseResult {
	return []CaseResult{
		s.m4Evo01FeedbackTargetTypes(),
		s.m4Evo02ReviewsQueue(),
		s.m5Evo03PromoteNeedsReview(),
		s.m5Evo04CanaryCeiling(),
	}
}

func (s *Service) harnessSvc() *harness.Service {
	if s == nil || s.runs == nil || s.runs.DB() == nil {
		return nil
	}
	return harness.NewService(s.runs.DB())
}

func (s *Service) m4Har01DefaultProfileSchema() CaseResult {
	res := CaseResult{ID: "M4-HAR-01", Status: "fail"}
	if err := harness.ValidateSpec(harness.DefaultSpec()); err != nil {
		res.Message = err.Error()
		return res
	}
	res.Status = "pass"
	res.Message = "default harness profile schema ok"
	res.Evidence = append(res.Evidence, Evidence{Kind: "harnessSchema", Ref: "ash.harness.profile.v1"})
	return res
}

func (s *Service) m4Har02EventInvariant() CaseResult {
	res := CaseResult{ID: "M4-HAR-02", Status: "fail"}
	okCovered := loop.AssertToolResultsCovered([]string{"tool.called", "tool.result"}, nil)
	okBare := loop.AssertToolResultsCovered([]string{"tool.result"}, nil)
	if !okCovered {
		res.Message = "covered sequence should pass HAR-02"
		return res
	}
	if okBare {
		res.Message = "bare tool.result without cover should fail HAR-02"
		return res
	}
	res.Status = "pass"
	res.Message = "HAR-02 event invariant holds"
	res.Evidence = append(res.Evidence, Evidence{Kind: "harnessInvariant", Ref: "tool.result ⊆ tool.called|harness.tool.completed"})
	return res
}

func (s *Service) m4Har03ActiveUniqueness() CaseResult {
	res := CaseResult{ID: "M4-HAR-03", Status: "fail"}
	hs := s.harnessSvc()
	if hs == nil {
		res.Message = "harness unavailable"
		return res
	}
	ok, err := hs.ActiveUniquenessOK("local")
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if !ok {
		res.Message = "active harness profile uniqueness violated"
		return res
	}
	view, err := hs.LoadActive("local", "default")
	if err != nil || view == nil {
		res.Message = fmt.Sprintf("LoadActive failed: %v", err)
		return res
	}
	res.Status = "pass"
	res.Message = "active profile unique; LoadActive ok"
	res.Evidence = append(res.Evidence, Evidence{Kind: "harnessActive", Ref: view.ID})
	return res
}

func (s *Service) m4Sbx01DockerSmokeOrSkip() CaseResult {
	res := CaseResult{ID: "M4-SBX-01", Status: "fail"}
	if os.Getenv("ASH_SKIP_SANDBOX") == "1" {
		res.Status = "pass"
		res.Message = "skipped: ASH_SKIP_SANDBOX=1"
		res.Evidence = append(res.Evidence, Evidence{Kind: "skipped", Ref: "ASH_SKIP_SANDBOX"})
		return res
	}
	if _, err := exec.LookPath("docker"); err != nil {
		res.Status = "pass"
		res.Message = "skipped: docker CLI not found (process-jail only)"
		res.Evidence = append(res.Evidence, Evidence{Kind: "skipped", Ref: "docker"})
		return res
	}
	res.Status = "pass"
	res.Message = "docker CLI present for sandbox smoke"
	res.Evidence = append(res.Evidence, Evidence{Kind: "sandboxDocker", Ref: "docker"})
	return res
}

func (s *Service) m4Sbx02DangerOffDenied() CaseResult {
	res := CaseResult{ID: "M4-SBX-02", Status: "fail"}
	err := sandbox.Authorize("danger", sandbox.ModeOff)
	if err == nil {
		res.Message = "danger under off should be denied"
		return res
	}
	res.Status = "pass"
	res.Message = "danger+off denied"
	res.Evidence = append(res.Evidence, Evidence{Kind: "sandboxPolicy", Ref: "Authorize(danger,off)"})
	return res
}

func (s *Service) m4Sbx03PathJail() CaseResult {
	res := CaseResult{ID: "M4-SBX-03", Status: "fail"}
	root := s.dataDir
	if root == "" {
		root = os.TempDir()
	}
	ok, err := sandbox.PathWithinRoot(root, filepath.Join(root, "ok.txt"))
	if err != nil || !ok {
		res.Message = fmt.Sprintf("in-root path should pass: ok=%v err=%v", ok, err)
		return res
	}
	escape := filepath.Join(root, "..", "escape.txt")
	ok, err = sandbox.PathWithinRoot(root, escape)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if ok {
		res.Message = "path outside repoRoot must be rejected"
		return res
	}
	res.Status = "pass"
	res.Message = "workspace-write path jail ok"
	res.Evidence = append(res.Evidence, Evidence{Kind: "pathJail", Ref: "PathWithinRoot"})
	return res
}

func (s *Service) m4Sbx04LandlockAvailable() CaseResult {
	res := CaseResult{ID: "M4-SBX-04", Status: "fail"}
	ok := landlock.Available()
	res.Evidence = append(res.Evidence, Evidence{Kind: "landlockAvailable", Ref: fmt.Sprintf("%v", ok)})
	if runtime.GOOS != "linux" {
		res.Status = "pass"
		res.Message = "skipped: landlock unavailable on this OS"
		res.Evidence = append(res.Evidence, Evidence{Kind: "skipped", Ref: runtime.GOOS})
		return res
	}
	if !ok {
		res.Status = "pass"
		res.Message = "skipped: landlock not supported by this kernel"
		res.Evidence = append(res.Evidence, Evidence{Kind: "skipped", Ref: "landlock"})
		return res
	}
	res.Status = "pass"
	res.Message = "landlock available"
	return res
}

func (s *Service) m4Acp01TaskSchema() CaseResult {
	res := CaseResult{ID: "M4-ACP-01", Status: "fail"}
	ok, err := agentexec.NewACPTaskV1("ash-acp", agentexec.Request{Prompt: "doctor probe", RunID: "run_doctor"})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if ok.Schema != agentexec.ACPTaskSchemaV1 {
		res.Message = "unexpected schema " + ok.Schema
		return res
	}
	bad := agentexec.ACPTaskV1{Schema: "other", AgentID: "a", Prompt: "x"}
	if err := bad.Validate(); err == nil {
		res.Message = "invalid schema should fail Validate"
		return res
	}
	res.Status = "pass"
	res.Message = "ash.acp.task.v1 validate ok"
	res.Evidence = append(res.Evidence, Evidence{Kind: "acpSchema", Ref: agentexec.ACPTaskSchemaV1})
	return res
}

func (s *Service) m4Acp02ProbeUnconfigured() CaseResult {
	res := CaseResult{ID: "M4-ACP-02", Status: "fail"}
	prevEndpoint := os.Getenv("ASH_ACP_ENDPOINT")
	prevURL := os.Getenv("ASH_ACP_URL")
	prevBin := os.Getenv("ASH_ACP_BIN")
	_ = os.Unsetenv("ASH_ACP_ENDPOINT")
	_ = os.Unsetenv("ASH_ACP_URL")
	_ = os.Unsetenv("ASH_ACP_BIN")
	defer func() {
		_ = os.Setenv("ASH_ACP_ENDPOINT", prevEndpoint)
		_ = os.Setenv("ASH_ACP_URL", prevURL)
		_ = os.Setenv("ASH_ACP_BIN", prevBin)
	}()
	rep := agentexec.ProbeACP(context.Background())
	if rep.OK || rep.Kind != "acp_sdk" {
		res.Message = fmt.Sprintf("unconfigured ProbeACP should be not-ok: %+v", rep)
		return res
	}
	if !strings.Contains(rep.Message, "ASH_ACP_ENDPOINT") {
		res.Message = "probe message should mention ASH_ACP_ENDPOINT: " + rep.Message
		return res
	}
	res.Status = "pass"
	res.Message = "ProbeACP unconfigured → not ok"
	res.Evidence = append(res.Evidence, Evidence{Kind: "acpProbe", Ref: "unconfigured"})
	return res
}

func (s *Service) m4Evo01FeedbackTargetTypes() CaseResult {
	res := CaseResult{ID: "M4-EVO-01", Status: "fail"}
	if _, ok := evolve.NormalizeTargetType("harness_profile"); !ok {
		res.Message = "harness_profile should be allowed"
		return res
	}
	if _, ok := evolve.NormalizeTargetType("not_a_type"); ok {
		res.Message = "unknown targetType should be rejected"
		return res
	}
	res.Status = "pass"
	res.Message = "feedback targetType enum ok"
	res.Evidence = append(res.Evidence, Evidence{Kind: "feedbackEnum", Ref: "evolve.AllowedTargetTypes"})
	return res
}

func (s *Service) m4Evo02ReviewsQueue() CaseResult {
	res := CaseResult{ID: "M4-EVO-02", Status: "fail"}
	if s.runs == nil || s.runs.DB() == nil {
		res.Message = "db unavailable"
		return res
	}
	mem := evolve.NewService(s.runs.DB(), nil, s.harnessSvc(), nil)
	q, err := mem.ListQueue("local", evolve.QueueOrchestration, 10)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if q == nil {
		res.Message = "nil queue response"
		return res
	}
	res.Status = "pass"
	res.Message = fmt.Sprintf("orchestration queue ok items=%d", len(q.Items))
	res.Evidence = append(res.Evidence, Evidence{Kind: "reviewsQueue", Ref: evolve.QueueOrchestration})
	return res
}

func (s *Service) m5Evo03PromoteNeedsReview() CaseResult {
	res := CaseResult{ID: "M5-EVO-03", Status: "fail"}
	hs := s.harnessSvc()
	if hs == nil {
		res.Message = "harness unavailable"
		return res
	}
	created, err := hs.Create(harness.CreateRequest{
		SpaceID: "local", Name: "doctor-m5-promote", Spec: harness.DefaultSpec(), CreatedBy: "doctor",
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	_, err = hs.Promote(created.ID, "doctor")
	if err == nil {
		res.Message = "promote without in_review must fail"
		return res
	}
	if !strings.Contains(err.Error(), "in_review") {
		res.Message = "unexpected promote error: " + err.Error()
		return res
	}
	res.Status = "pass"
	res.Message = "promote without review rejected"
	res.Evidence = append(res.Evidence, Evidence{Kind: "promoteGate", Ref: created.ID})
	return res
}

func (s *Service) m5Evo04CanaryCeiling() CaseResult {
	res := CaseResult{ID: "M5-EVO-04", Status: "fail"}
	if improve.MaxCanaryPercent != 10 {
		res.Message = fmt.Sprintf("MaxCanaryPercent=%d want 10", improve.MaxCanaryPercent)
		return res
	}
	if s.runs == nil || s.runs.DB() == nil {
		res.Message = "db unavailable"
		return res
	}
	svc := improve.NewService(s.runs.DB(), s.runs, s.events)
	row := store.ImproveProposal{
		ID: "imp_doctor_canary", SpaceID: "local", Title: "doctor canary",
		Status: "experimenting", BaselineRunID: "run_doctor", ChangeSummary: "probe",
		CompareJSON: "{}",
	}
	_ = s.runs.DB().Where("id = ?", row.ID).Delete(&store.ImproveProposal{})
	if err := s.runs.DB().Create(&row).Error; err != nil {
		res.Message = err.Error()
		return res
	}
	_, err := svc.StartCanary("local", row.ID, improve.CanaryRequest{Percent: 11})
	if err == nil {
		res.Message = "canary percent 11 must be rejected"
		return res
	}
	if !strings.Contains(err.Error(), "1..10") {
		res.Message = "unexpected canary error: " + err.Error()
		return res
	}
	_, err = svc.StartCanary("local", row.ID, improve.CanaryRequest{Percent: 10})
	if err != nil {
		res.Message = "canary percent 10 should pass: " + err.Error()
		return res
	}
	res.Status = "pass"
	res.Message = "canary ceiling ≤10%"
	res.Evidence = append(res.Evidence, Evidence{Kind: "canaryCeiling", Ref: fmt.Sprintf("max=%d", improve.MaxCanaryPercent)})
	return res
}
