package waker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/store"
)

const (
	KindDoctorSubset = "doctor_subset"
	KindKPIDrift     = "kpi_drift"

	defaultDoctorSuite   = "M4"
	defaultKPIMetric     = "KPI-17"
	defaultKPIThreshold  = float64(50)
)

// ErrDoctorUnavailable is returned when doctor_subset runs without a DoctorRunner.
var ErrDoctorUnavailable = errors.New("doctor runner unavailable")

// DoctorCaseResult is one suite case outcome for Waker probes.
type DoctorCaseResult struct {
	ID     string
	Status string // pass | fail | other
}

// DoctorReport is a thin suite result for DoctorRunner.
type DoctorReport struct {
	Cases []DoctorCaseResult
}

// DoctorRunner runs a named Doctor suite (injected; avoids hard doctor import cycles in tests).
type DoctorRunner interface {
	RunSuite(suite string) (*DoctorReport, error)
}

// KPIBacklogFunc returns KPI-17 style backlog count for a space (test hook / override).
type KPIBacklogFunc func(spaceID string) (int64, error)

func errorsIsDoctorUnavailable(err error) bool {
	return errors.Is(err, ErrDoctorUnavailable)
}

// WithDoctorRunner returns a shallow copy with an injected DoctorRunner.
func (s *Service) WithDoctorRunner(r DoctorRunner) *Service {
	if s == nil {
		return s
	}
	out := *s
	out.doctor = r
	return &out
}

// WithKPIBacklog returns a shallow copy with an injected backlog counter.
func (s *Service) WithKPIBacklog(fn KPIBacklogFunc) *Service {
	if s == nil {
		return s
	}
	out := *s
	out.kpiBacklog = fn
	return &out
}

type doctorConfig struct {
	Suite   string   `json:"suite"`
	CaseIDs []string `json:"caseIds"`
}

type kpiConfig struct {
	Metric    string   `json:"metric"`
	Threshold *float64 `json:"threshold"`
	Baseline  *float64 `json:"baseline"`
	Mode      string   `json:"mode"`
}

func parseDoctorConfig(raw string) (suite string, caseIDs []string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	var cfg doctorConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return "", nil, err
	}
	suite = strings.TrimSpace(cfg.Suite)
	if suite == "" {
		suite = defaultDoctorSuite
	}
	for _, id := range cfg.CaseIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			caseIDs = append(caseIDs, id)
		}
	}
	return suite, caseIDs, nil
}

func parseKPIConfig(raw string) (metric string, threshold float64, baseline *float64, mode string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	var cfg kpiConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return "", 0, nil, "", err
	}
	metric = strings.TrimSpace(cfg.Metric)
	if metric == "" {
		metric = defaultKPIMetric
	}
	threshold = defaultKPIThreshold
	if cfg.Threshold != nil {
		threshold = *cfg.Threshold
	}
	baseline = cfg.Baseline
	mode = strings.TrimSpace(cfg.Mode)
	return metric, threshold, baseline, mode, nil
}

func (s *Service) runDoctorSubset(duty store.WakerDuty, dryRun bool) (SweepResponse, error) {
	_ = dryRun
	out := SweepResponse{OK: true, DryRun: dryRun, Action: "report"}
	if s == nil || s.doctor == nil {
		return out, ErrDoctorUnavailable
	}
	suite, caseIDs, err := parseDoctorConfig(duty.ConfigJSON)
	if err != nil {
		return out, err
	}
	rep, err := s.doctor.RunSuite(suite)
	if err != nil {
		return out, err
	}
	allow := map[string]bool{}
	for _, id := range caseIDs {
		allow[id] = true
	}
	var fails []string
	for _, c := range rep.Cases {
		if len(allow) > 0 && !allow[c.ID] {
			continue
		}
		out.Matched++
		if strings.EqualFold(c.Status, "fail") {
			out.Flagged++
			fails = append(fails, "doctor_subset:"+c.ID)
		}
	}
	out.Summary = strings.Join(fails, " ")
	if out.Summary == "" {
		out.Summary = fmt.Sprintf("doctor_subset suite=%s matched=%d flagged=0", suite, out.Matched)
	}
	return out, nil
}

func (s *Service) runKPIDrift(duty store.WakerDuty, dryRun bool) (SweepResponse, error) {
	out := SweepResponse{OK: true, DryRun: dryRun, Action: "report"}
	metric, threshold, baseline, _, err := parseKPIConfig(duty.ConfigJSON)
	if err != nil {
		return out, err
	}
	if !strings.EqualFold(metric, defaultKPIMetric) {
		return out, fmt.Errorf("unsupported kpi metric %q (DX13 supports KPI-17 only)", metric)
	}
	backlog, err := s.kpiBacklogCount(normalizeSpaceID(duty.SpaceID))
	if err != nil {
		return out, err
	}
	out.Matched = int(backlog)
	flag := false
	if baseline != nil {
		// Delta from baseline: flag when backlog - baseline >= threshold.
		if float64(backlog)-*baseline >= threshold {
			flag = true
		}
	} else if float64(backlog) >= threshold {
		flag = true
	}
	if flag {
		out.Flagged = 1
	}
	out.Summary = fmt.Sprintf("kpi_drift:%s backlog=%d threshold=%g", defaultKPIMetric, backlog, threshold)
	return out, nil
}

func (s *Service) kpiBacklogCount(spaceID string) (int64, error) {
	if s != nil && s.kpiBacklog != nil {
		return s.kpiBacklog(spaceID)
	}
	if s.q() == nil {
		return 0, fmt.Errorf("database unavailable")
	}
	var harness, patch int64
	if err := s.q().Model(&store.HarnessProfileVersion{}).
		Where("space_id = ? AND status = ?", spaceID, "in_review").
		Count(&harness).Error; err != nil {
		return 0, err
	}
	if err := s.q().Model(&store.ScenarioPatchDraft{}).
		Where("space_id = ? AND status = ?", spaceID, "in_review").
		Count(&patch).Error; err != nil {
		return 0, err
	}
	return harness + patch, nil
}
