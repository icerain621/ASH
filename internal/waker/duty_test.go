package waker

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "ash.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedStaleRunningRun(t *testing.T, db *store.DB) {
	t.Helper()
	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: "run_stale", TraceID: "tr_stale",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: runs.StatusRunning, SpaceID: "local",
		RepoRoot: ".", StartedAt: now.Add(-3 * time.Hour),
		CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestStatusIntervalOffWhenEnvEmpty(t *testing.T) {
	t.Setenv("ASH_WAKER_INTERVAL", "")
	db := openTestDB(t)
	svc := NewService(db)
	st, err := svc.Status("local", 5)
	if err != nil {
		t.Fatal(err)
	}
	if st.Interval != "off" || st.IntervalMs != 0 {
		t.Fatalf("want ticker off: interval=%q intervalMs=%d", st.Interval, st.IntervalMs)
	}
	if len(st.Duties) < 1 || st.Duties[0].IntervalMs < 60000 {
		t.Fatalf("duty cadence stays on duty row: %+v", st.Duties)
	}
}

func TestEnsureStaleRunDutyPreservesNextRunAt(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db)
	future := time.Now().UTC().Add(2 * time.Hour)
	if _, err := svc.EnsureStaleRunDuty("local"); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&store.WakerDuty{}).
		Where("space_id = ? AND kind = ?", "local", KindStaleRun).
		Update("next_run_at", future).Error; err != nil {
		t.Fatal(err)
	}
	got, err := svc.EnsureStaleRunDuty("local")
	if err != nil {
		t.Fatal(err)
	}
	if !got.NextRunAt.Equal(future) {
		t.Fatalf("NextRunAt changed: want %v got %v", future, got.NextRunAt)
	}
}

func TestEnsureStaleRunDutyIdempotent(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db)
	a, err := svc.EnsureStaleRunDuty("local")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.EnsureStaleRunDuty("local")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" || a.ID != b.ID || a.Kind != KindStaleRun {
		t.Fatalf("idempotent ensure failed: %+v vs %+v", a, b)
	}
}

func TestRunDueDutiesWritesDutyRun(t *testing.T) {
	t.Setenv("ASH_WAKER_RUN_TTL", "1h")
	db := openTestDB(t)
	seedStaleRunningRun(t, db)
	svc := NewService(db)
	if _, err := svc.EnsureStaleRunDuty("local"); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&store.WakerDuty{}).
		Where("space_id = ? AND kind = ?", "local", KindStaleRun).
		Update("next_run_at", past).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	n, err := svc.RunDueDuties(now)
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	var duty store.WakerDuty
	if err := db.First(&duty, "space_id = ? AND kind = ?", "local", KindStaleRun).Error; err != nil {
		t.Fatal(err)
	}
	if !duty.NextRunAt.After(now) {
		t.Fatalf("next_run_at not advanced: now=%v next=%v", now, duty.NextRunAt)
	}
	st, err := svc.Status("local", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.RecentRuns) < 1 || st.RecentRuns[0].Kind != KindStaleRun {
		t.Fatalf("want duty run: %+v", st)
	}
}

func TestEnsureStaleRunDutyClampsInterval(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&store.WakerDuty{
		ID: "wd_low", SpaceID: "local", Kind: KindStaleRun, Enabled: true,
		IntervalMs: 1000, ConfigJSON: "{}", NextRunAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	got, err := svc.EnsureStaleRunDuty("local")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "wd_low" || got.IntervalMs < 60000 {
		t.Fatalf("want clamp >= 60000: %+v", got)
	}
}

func TestRunDutyDoctorSubset(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db).WithDoctorRunner(fakeDoctor{failIDs: []string{"M4-HAR-01"}})
	duty, err := svc.EnsureDoctorSubsetDuty("local", true)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := svc.RunDuty(duty.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Flagged < 1 {
		t.Fatalf("want flagged: %+v", resp)
	}
}

func TestRunDueDutiesExecutesDoctorSubset(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db).WithDoctorRunner(fakeDoctor{failIDs: []string{"M4-HAR-01"}})
	duty, err := svc.EnsureDoctorSubsetDuty("local", true)
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&store.WakerDuty{}).Where("id = ?", duty.ID).Update("next_run_at", past).Error; err != nil {
		t.Fatal(err)
	}
	n, err := svc.RunDueDuties(time.Now().UTC())
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	st, err := svc.Status("local", 5)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range st.RecentRuns {
		if r.DutyID == duty.ID {
			found = true
			if r.Flagged < 1 || r.Canceled != 0 {
				t.Fatalf("want flagged never-cancel: %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("want doctor duty run: %+v", st)
	}
}

func TestQueueIncludesDoctorFindings(t *testing.T) {
	db := openTestDB(t)
	svc := NewService(db).WithDoctorRunner(fakeDoctor{failIDs: []string{"M4-HAR-01"}})
	duty, err := svc.EnsureDoctorSubsetDuty("local", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunDuty(duty.ID, false); err != nil {
		t.Fatal(err)
	}
	q, err := svc.Queue("local", "", 50)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range q.Items {
		if it.Kind == KindDoctorSubset && strings.Contains(it.Reason, "doctor_subset:") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want doctor queue item: %+v", q.Items)
	}
}

func TestStatusAutoSeedsProbesDisabled(t *testing.T) {
	t.Setenv("ASH_WAKER_ENABLE_PROBES", "")
	db := openTestDB(t)
	svc := NewService(db)
	st, err := svc.Status("local", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !st.ProbesAvailable {
		t.Fatal("want probesAvailable after Status seed")
	}
	list, err := svc.ListDuties("local")
	if err != nil {
		t.Fatal(err)
	}
	var doctor, kpi store.WakerDuty
	for _, d := range list {
		switch d.Kind {
		case KindDoctorSubset:
			doctor = d
		case KindKPIDrift:
			kpi = d
		}
	}
	if doctor.ID == "" || kpi.ID == "" {
		t.Fatalf("want seeded probe duties: %+v", list)
	}
	if doctor.Enabled || kpi.Enabled {
		t.Fatalf("seeded probes must be disabled by default: doctor=%v kpi=%v", doctor.Enabled, kpi.Enabled)
	}
	// Second Status must not flip enabled after explicit enable.
	if _, err := svc.SetDutyEnabled(doctor.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Status("local", 5); err != nil {
		t.Fatal(err)
	}
	list2, err := svc.ListDuties("local")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range list2 {
		if d.Kind == KindDoctorSubset && !d.Enabled {
			t.Fatal("Status re-seed must not disable an enabled probe duty")
		}
	}
}

func TestSetDutyEnabled(t *testing.T) {
	t.Setenv("ASH_WAKER_ENABLE_PROBES", "")
	db := openTestDB(t)
	svc := NewService(db)
	if err := svc.SeedProbeDuties("local"); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListDuties("local")
	if err != nil {
		t.Fatal(err)
	}
	var doctorID string
	for _, d := range list {
		if d.Kind == KindDoctorSubset {
			doctorID = d.ID
		}
	}
	if doctorID == "" {
		t.Fatal("missing doctor duty")
	}
	duty, err := svc.SetDutyEnabled(doctorID, true)
	if err != nil || !duty.Enabled {
		t.Fatalf("enable: %+v err=%v", duty, err)
	}
	duty, err = svc.SetDutyEnabled(doctorID, false)
	if err != nil || duty.Enabled {
		t.Fatalf("disable: %+v err=%v", duty, err)
	}
}

func TestProbesEnabledOnBoot(t *testing.T) {
	t.Setenv("ASH_WAKER_ENABLE_PROBES", "1")
	db := openTestDB(t)
	svc := NewService(db)
	if err := svc.SeedProbeDuties("local"); err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListDuties("local")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range list {
		if (d.Kind == KindDoctorSubset || d.Kind == KindKPIDrift) && !d.Enabled {
			t.Fatalf("want probes enabled on create when env set: %+v", d)
		}
	}
}

func TestRunDutyUnsupportedKind(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&store.WakerDuty{
		ID: "wd_future", SpaceID: "local", Kind: "future_probe", Enabled: true,
		IntervalMs: 300000, ConfigJSON: "{}", NextRunAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	_, err := svc.RunDuty("wd_future", true)
	if !errors.Is(err, ErrUnsupportedDutyKind) {
		t.Fatalf("want unsupported duty kind, got %v", err)
	}
}

func TestRunDueDutiesSkipsUnknownKind(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC().Add(-time.Minute)
	if err := db.Create(&store.WakerDuty{
		ID: "wd_future2", SpaceID: "local", Kind: "future_probe", Enabled: true,
		IntervalMs: 300000, ConfigJSON: "{}", NextRunAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	n, err := svc.RunDueDuties(time.Now().UTC())
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	st, err := svc.Status("local", 5)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range st.RecentRuns {
		if r.DutyID == "wd_future2" {
			found = true
			if r.Status != "skipped" || r.Canceled != 0 {
				t.Fatalf("want skipped never-cancel: %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("want skipped future_probe run: %+v", st)
	}
}

func TestRunDueDutiesNeverCancels(t *testing.T) {
	t.Setenv("ASH_WAKER_RUN_TTL", "1h")
	t.Setenv("ASH_WAKER_ALLOW_CANCEL", "1")
	db := openTestDB(t)
	seedStaleRunningRun(t, db)
	svc := NewService(db)
	if _, err := svc.EnsureStaleRunDuty("local"); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&store.WakerDuty{}).
		Where("space_id = ? AND kind = ?", "local", KindStaleRun).
		Update("next_run_at", past).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunDueDuties(time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	var rec store.RunRecord
	if err := db.First(&rec, "id = ?", "run_stale").Error; err != nil {
		t.Fatal(err)
	}
	if rec.Status != runs.StatusRunning {
		t.Fatalf("background must not cancel, status=%s", rec.Status)
	}
	st, err := svc.Status("local", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.RecentRuns) < 1 || st.RecentRuns[0].Canceled != 0 {
		t.Fatalf("want canceled=0: %+v", st)
	}
}
