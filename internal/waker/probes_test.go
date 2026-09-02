package waker

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

type fakeDoctor struct {
	failIDs []string
	err     error
}

func (f fakeDoctor) RunSuite(suite string) (*DoctorReport, error) {
	if f.err != nil {
		return nil, f.err
	}
	fail := map[string]bool{}
	for _, id := range f.failIDs {
		fail[id] = true
	}
	cases := []DoctorCaseResult{
		{ID: "M4-HAR-01", Status: "pass"},
		{ID: "M4-HAR-02", Status: "pass"},
	}
	for i := range cases {
		if fail[cases[i].ID] {
			cases[i].Status = "fail"
		}
	}
	_ = suite
	return &DoctorReport{Cases: cases}, nil
}

func TestParseDoctorConfigDefaults(t *testing.T) {
	suite, ids, err := parseDoctorConfig("{}")
	if err != nil || suite != "M4" || len(ids) != 0 {
		t.Fatalf("suite=%q ids=%v err=%v", suite, ids, err)
	}
}

func TestParseDoctorConfigCaseIDs(t *testing.T) {
	suite, ids, err := parseDoctorConfig(`{"suite":"M5","caseIds":["M5-01"]}`)
	if err != nil || suite != "M5" || len(ids) != 1 || ids[0] != "M5-01" {
		t.Fatalf("suite=%q ids=%v err=%v", suite, ids, err)
	}
}

func TestParseKPIConfigDefaults(t *testing.T) {
	metric, threshold, baseline, mode, err := parseKPIConfig("{}")
	if err != nil || metric != "KPI-17" || threshold != 50 || baseline != nil || mode != "" {
		t.Fatalf("metric=%q thr=%v base=%v mode=%q err=%v", metric, threshold, baseline, mode, err)
	}
}

func TestRunDoctorSubsetFlagsFailures(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db).WithDoctorRunner(fakeDoctor{failIDs: []string{"M4-HAR-01"}})
	duty := store.WakerDuty{
		ID: "wd_d", SpaceID: "local", Kind: KindDoctorSubset,
		ConfigJSON: `{"suite":"M4"}`, Enabled: true, IntervalMs: 300000,
	}
	resp, err := svc.runDoctorSubset(duty, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Flagged < 1 || resp.Matched < 1 {
		t.Fatalf("want flagged failures: %+v", resp)
	}
	if !strings.Contains(resp.Summary, "doctor_subset:M4-HAR-01") {
		t.Fatalf("summary=%q", resp.Summary)
	}
}

func TestRunDoctorSubsetUnavailable(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db)
	duty := store.WakerDuty{ID: "wd_d2", SpaceID: "local", Kind: KindDoctorSubset, ConfigJSON: `{}`}
	_, err := svc.runDoctorSubset(duty, false)
	if !errorsIsDoctorUnavailable(err) {
		t.Fatalf("want doctor unavailable, got %v", err)
	}
}

func TestRunKPIDriftFlagsOverThreshold(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&store.HarnessProfileVersion{
		ID: "hp1", SpaceID: "local", Name: "default", Version: 1,
		Status: "in_review", SpecJSON: "{}", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	duty := store.WakerDuty{
		ID: "wd_k", SpaceID: "local", Kind: KindKPIDrift,
		ConfigJSON: `{"metric":"KPI-17","threshold":0}`, Enabled: true, IntervalMs: 300000,
	}
	resp, err := svc.runKPIDrift(duty, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Flagged < 1 || resp.Matched < 1 {
		t.Fatalf("want flag: %+v", resp)
	}
	if !strings.Contains(resp.Summary, "kpi_drift:KPI-17") {
		t.Fatalf("summary=%q", resp.Summary)
	}
}

func TestRunKPIDriftWithBacklogHook(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db).WithKPIBacklog(func(spaceID string) (int64, error) {
		return 60, nil
	})
	duty := store.WakerDuty{
		ID: "wd_k2", SpaceID: "local", Kind: KindKPIDrift,
		ConfigJSON: `{"metric":"KPI-17","threshold":50}`,
	}
	resp, err := svc.runKPIDrift(duty, true)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Flagged != 1 || resp.Matched != 60 {
		t.Fatalf("resp=%+v", resp)
	}
}
