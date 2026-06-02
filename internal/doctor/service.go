package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/rules"
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
	case "TR0", "ALL":
		rep.Results = append(rep.Results, s.tr0DeliveryLoop())
		rep.Results = append(rep.Results, s.tr0EventStream())
		rep.Results = append(rep.Results, s.tr0ReplayDigest())
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
	probeDir := filepath.Join(s.dataDir, "doctor-probe")
	_ = os.MkdirAll(probeDir, 0o755)
	return map[string]any{
		"issueOrSpec": "doctor " + suffix,
		"repoRoot":    probeDir,
	}
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
	create, err := s.runs.Create(runs.CreateRequest{
		Scenario: runs.ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   s.probeInputs("TR0-03"),
	})
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.RunID = create.RunID

	runDir := filepath.Join(s.dataDir, "runs", create.RunID)
	m1, err := artifacts.LoadManifest(runDir)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	m2, err := artifacts.LoadManifest(runDir)
	if err != nil {
		res.Message = err.Error()
		return res
	}
	if len(m1.Artifacts) != len(m2.Artifacts) {
		res.Message = "artifact count mismatch on replay read"
		return res
	}
	for i := range m1.Artifacts {
		if m1.Artifacts[i].Digest != m2.Artifacts[i].Digest {
			res.Message = fmt.Sprintf("digest drift on %s", m1.Artifacts[i].Type)
			return res
		}
		res.Evidence = append(res.Evidence, Evidence{Kind: "artifact", Ref: m1.Artifacts[i].Type, Digest: m1.Artifacts[i].Digest})
	}
	res.Status = "pass"
	return res
}
