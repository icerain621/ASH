package waker

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

const (
	KindStaleRun = "stale_run"

	defaultDutyIntervalMs = int64(300000)
	minDutyIntervalMs     = int64(60000)

	dutyStatusOK      = "ok"
	dutyStatusFailed  = "failed"
	dutyStatusSkipped = "skipped"
)

// ErrUnsupportedDutyKind is returned when RunDuty is asked to execute a non-stale_run kind (DX13 placeholders).
var ErrUnsupportedDutyKind = errors.New("unsupported duty kind")

// StatusResponse is GET /waker/status.
type StatusResponse struct {
	Duties      []DutyStatusView `json:"duties"`
	RecentRuns  []DutyRunView    `json:"recentRuns"`
	AllowCancel bool             `json:"allowCancel"`
	Interval    string           `json:"interval,omitempty"`
	IntervalMs  int64            `json:"intervalMs,omitempty"`
}

// DutyStatusView is one enabled/scheduled duty in Status.
type DutyStatusView struct {
	ID         string    `json:"id"`
	SpaceID    string    `json:"spaceId"`
	Kind       string    `json:"kind"`
	Enabled    bool      `json:"enabled"`
	IntervalMs int64     `json:"intervalMs"`
	NextRunAt  time.Time `json:"nextRunAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// DutyRunView is one persisted waker_duty_runs row.
type DutyRunView struct {
	ID         string    `json:"id"`
	DutyID     string    `json:"dutyId"`
	SpaceID    string    `json:"spaceId"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	Matched    int       `json:"matched"`
	Flagged    int       `json:"flagged"`
	Canceled   int       `json:"canceled"`
	Summary    string    `json:"summary"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
}

func newDutyID() string {
	return "wd_" + uuid.NewString()
}

func newDutyRunID() string {
	return "wdr_" + uuid.NewString()
}

func normalizeSpaceID(spaceID string) string {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		return "local"
	}
	return spaceID
}

func clampIntervalMs(ms int64) int64 {
	if ms < minDutyIntervalMs {
		return minDutyIntervalMs
	}
	return ms
}

func defaultIntervalMs() int64 {
	d, ok := ParseInterval(os.Getenv("ASH_WAKER_INTERVAL"))
	if !ok {
		return defaultDutyIntervalMs
	}
	return clampIntervalMs(d.Milliseconds())
}

// EnsureStaleRunDuty upserts the unique (space_id, kind=stale_run) duty.
func (s *Service) EnsureStaleRunDuty(spaceID string) (store.WakerDuty, error) {
	if s.q() == nil {
		return store.WakerDuty{}, fmt.Errorf("database unavailable")
	}
	spaceID = normalizeSpaceID(spaceID)
	var existing store.WakerDuty
	err := s.q().Where("space_id = ? AND kind = ?", spaceID, KindStaleRun).First(&existing).Error
	if err == nil {
		if existing.IntervalMs < minDutyIntervalMs {
			existing.IntervalMs = minDutyIntervalMs
			existing.UpdatedAt = time.Now().UTC()
			if saveErr := s.q().Model(&store.WakerDuty{}).Where("id = ?", existing.ID).
				Updates(map[string]any{"interval_ms": existing.IntervalMs, "updated_at": existing.UpdatedAt}).Error; saveErr != nil {
				return store.WakerDuty{}, saveErr
			}
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return store.WakerDuty{}, err
	}
	now := time.Now().UTC()
	duty := store.WakerDuty{
		ID:         newDutyID(),
		SpaceID:    spaceID,
		Kind:       KindStaleRun,
		Enabled:    true,
		IntervalMs: defaultIntervalMs(),
		ConfigJSON: "{}",
		NextRunAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.q().Create(&duty).Error; err != nil {
		var raced store.WakerDuty
		if findErr := s.q().Where("space_id = ? AND kind = ?", spaceID, KindStaleRun).First(&raced).Error; findErr == nil {
			return raced, nil
		}
		return store.WakerDuty{}, err
	}
	return duty, nil
}

// EnsureDoctorSubsetDuty upserts (space_id, kind=doctor_subset). Does not auto-run from Status.
func (s *Service) EnsureDoctorSubsetDuty(spaceID string, enabled bool) (store.WakerDuty, error) {
	return s.ensureProbeDuty(spaceID, KindDoctorSubset, `{"suite":"M4"}`, enabled)
}

// EnsureKPIDriftDuty upserts (space_id, kind=kpi_drift). Does not auto-run from Status.
func (s *Service) EnsureKPIDriftDuty(spaceID string, enabled bool) (store.WakerDuty, error) {
	return s.ensureProbeDuty(spaceID, KindKPIDrift, `{"metric":"KPI-17","threshold":50}`, enabled)
}

func (s *Service) ensureProbeDuty(spaceID, kind, defaultConfig string, enabled bool) (store.WakerDuty, error) {
	if s.q() == nil {
		return store.WakerDuty{}, fmt.Errorf("database unavailable")
	}
	spaceID = normalizeSpaceID(spaceID)
	var existing store.WakerDuty
	err := s.q().Where("space_id = ? AND kind = ?", spaceID, kind).First(&existing).Error
	if err == nil {
		updates := map[string]any{"enabled": enabled, "updated_at": time.Now().UTC()}
		if existing.IntervalMs < minDutyIntervalMs {
			updates["interval_ms"] = minDutyIntervalMs
			existing.IntervalMs = minDutyIntervalMs
		}
		if saveErr := s.q().Model(&store.WakerDuty{}).Where("id = ?", existing.ID).Updates(updates).Error; saveErr != nil {
			return store.WakerDuty{}, saveErr
		}
		existing.Enabled = enabled
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return store.WakerDuty{}, err
	}
	now := time.Now().UTC()
	duty := store.WakerDuty{
		ID: newDutyID(), SpaceID: spaceID, Kind: kind, Enabled: enabled,
		IntervalMs: defaultIntervalMs(), ConfigJSON: defaultConfig,
		NextRunAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.q().Create(&duty).Error; err != nil {
		var raced store.WakerDuty
		if findErr := s.q().Where("space_id = ? AND kind = ?", spaceID, kind).First(&raced).Error; findErr == nil {
			return raced, nil
		}
		return store.WakerDuty{}, err
	}
	return duty, nil
}

// ListDuties returns duties for a space.
func (s *Service) ListDuties(spaceID string) ([]store.WakerDuty, error) {
	if s.q() == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	spaceID = normalizeSpaceID(spaceID)
	var duties []store.WakerDuty
	if err := s.q().Where("space_id = ?", spaceID).Order("kind ASC").Find(&duties).Error; err != nil {
		return nil, err
	}
	return duties, nil
}

// Status ensures the stale_run duty and returns duties, recent runs, and ticker hints.
func (s *Service) Status(spaceID string, recent int) (StatusResponse, error) {
	if _, err := s.EnsureStaleRunDuty(spaceID); err != nil {
		return StatusResponse{}, err
	}
	duties, err := s.ListDuties(spaceID)
	if err != nil {
		return StatusResponse{}, err
	}
	if recent <= 0 {
		recent = 10
	}
	if recent > 100 {
		recent = 100
	}
	spaceID = normalizeSpaceID(spaceID)
	var rows []store.WakerDutyRun
	if err := s.q().Where("space_id = ?", spaceID).
		Order("started_at DESC").Limit(recent).Find(&rows).Error; err != nil {
		return StatusResponse{}, err
	}
	views := make([]DutyStatusView, 0, len(duties))
	for _, d := range duties {
		views = append(views, DutyStatusView{
			ID: d.ID, SpaceID: d.SpaceID, Kind: d.Kind, Enabled: d.Enabled,
			IntervalMs: d.IntervalMs, NextRunAt: d.NextRunAt, UpdatedAt: d.UpdatedAt,
		})
	}
	runViews := make([]DutyRunView, 0, len(rows))
	for _, r := range rows {
		runViews = append(runViews, DutyRunView{
			ID: r.ID, DutyID: r.DutyID, SpaceID: r.SpaceID, Kind: r.Kind,
			Status: r.Status, Matched: r.Matched, Flagged: r.Flagged, Canceled: r.Canceled,
			Summary: r.Summary, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
		})
	}
	raw := strings.TrimSpace(os.Getenv("ASH_WAKER_INTERVAL"))
	d, ok := ParseInterval(raw)
	intervalHint := raw
	intervalMs := int64(0)
	if ok {
		intervalHint = d.String()
		intervalMs = d.Milliseconds()
	} else if raw == "" {
		intervalHint = "off"
	}
	return StatusResponse{
		Duties: views, RecentRuns: runViews,
		AllowCancel: AllowCancel(),
		Interval:    intervalHint,
		IntervalMs:  intervalMs,
	}, nil
}

// RunDuty force-runs one duty (stale_run / doctor_subset / kpi_drift). Never cancels.
func (s *Service) RunDuty(dutyID string, dryRun bool) (SweepResponse, error) {
	if s.q() == nil {
		return SweepResponse{}, fmt.Errorf("database unavailable")
	}
	var duty store.WakerDuty
	if err := s.q().First(&duty, "id = ?", strings.TrimSpace(dutyID)).Error; err != nil {
		return SweepResponse{}, err
	}
	started := time.Now().UTC()
	resp, err := s.executeDuty(duty, dryRun)
	if errors.Is(err, ErrDoctorUnavailable) {
		_ = s.persistDutyRun(duty, dutyStatusSkipped, SweepResponse{Summary: err.Error()}, started)
		return SweepResponse{}, err
	}
	if errors.Is(err, ErrUnsupportedDutyKind) {
		return SweepResponse{}, err
	}
	if err != nil {
		_ = s.persistDutyRun(duty, dutyStatusFailed, SweepResponse{Summary: err.Error()}, started)
		return SweepResponse{}, err
	}
	status := dutyStatusOK
	if resp.Flagged > 0 {
		status = dutyStatusFailed
	}
	if err := s.persistDutyRun(duty, status, resp, started); err != nil {
		return SweepResponse{}, err
	}
	return resp, nil
}

func (s *Service) executeDuty(duty store.WakerDuty, dryRun bool) (SweepResponse, error) {
	switch duty.Kind {
	case KindStaleRun:
		return s.runStaleDuty(duty, dryRun)
	case KindDoctorSubset:
		return s.runDoctorSubset(duty, dryRun)
	case KindKPIDrift:
		return s.runKPIDrift(duty, dryRun)
	default:
		return SweepResponse{}, ErrUnsupportedDutyKind
	}
}

// RunDueDuties executes enabled duties whose next_run_at is due. Never cancels.
func (s *Service) RunDueDuties(now time.Time) (int, error) {
	if s.q() == nil {
		return 0, fmt.Errorf("database unavailable")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var due []store.WakerDuty
	if err := s.q().Where("enabled = ? AND next_run_at <= ?", true, now).
		Order("next_run_at ASC").Find(&due).Error; err != nil {
		return 0, err
	}
	ran := 0
	for i := range due {
		duty := due[i]
		started := time.Now().UTC()
		status := dutyStatusOK
		var resp SweepResponse
		resp, runErr := s.executeDuty(duty, false)
		if errors.Is(runErr, ErrDoctorUnavailable) || errors.Is(runErr, ErrUnsupportedDutyKind) {
			status = dutyStatusSkipped
			resp.Summary = runErr.Error()
		} else if runErr != nil {
			status = dutyStatusFailed
			resp.Summary = runErr.Error()
		} else if resp.Flagged > 0 && (duty.Kind == KindDoctorSubset || duty.Kind == KindKPIDrift) {
			status = dutyStatusFailed
		}
		if err := s.persistDutyRun(duty, status, resp, started); err != nil {
			return ran, err
		}
		intervalMs := clampIntervalMs(duty.IntervalMs)
		next := now.Add(time.Duration(intervalMs) * time.Millisecond)
		if err := s.q().Model(&store.WakerDuty{}).Where("id = ?", duty.ID).Updates(map[string]any{
			"next_run_at": next,
			"updated_at":  time.Now().UTC(),
			"interval_ms": intervalMs,
		}).Error; err != nil {
			return ran, err
		}
		ran++
	}
	return ran, nil
}

func (s *Service) runStaleDuty(duty store.WakerDuty, dryRun bool) (SweepResponse, error) {
	return s.Sweep(SweepRequest{
		SpaceID: duty.SpaceID,
		DryRun:  &dryRun,
		Action:  "report",
		Limit:   defaultMaxItems,
	})
}

func (s *Service) persistDutyRun(duty store.WakerDuty, status string, resp SweepResponse, started time.Time) error {
	row := store.WakerDutyRun{
		ID:         newDutyRunID(),
		SpaceID:    duty.SpaceID,
		DutyID:     duty.ID,
		Kind:       duty.Kind,
		Status:     status,
		Matched:    resp.Matched,
		Flagged:    resp.Flagged,
		Canceled:   0,
		Summary:    resp.Summary,
		StartedAt:  started,
		FinishedAt: time.Now().UTC(),
	}
	return s.q().Create(&row).Error
}

// knownSpaces returns distinct space_id values from run records (cheap DX12 stretch).
func (s *Service) knownSpaces() []string {
	if s.q() == nil {
		return nil
	}
	var spaces []string
	_ = s.q().Model(&store.RunRecord{}).Distinct("space_id").Pluck("space_id", &spaces).Error
	return spaces
}
