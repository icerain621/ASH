package waker

import (
	"errors"
	"path/filepath"
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
	n, err := svc.RunDueDuties(time.Now().UTC())
	if err != nil || n < 1 {
		t.Fatalf("n=%d err=%v", n, err)
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

func TestRunDutyUnsupportedKind(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&store.WakerDuty{
		ID: "wd_doc", SpaceID: "local", Kind: "doctor_subset", Enabled: true,
		IntervalMs: 300000, ConfigJSON: "{}", NextRunAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	_, err := svc.RunDuty("wd_doc", true)
	if !errors.Is(err, ErrUnsupportedDutyKind) {
		t.Fatalf("want unsupported duty kind, got %v", err)
	}
}

func TestRunDueDutiesSkipsUnknownKind(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC().Add(-time.Minute)
	if err := db.Create(&store.WakerDuty{
		ID: "wd_kpi", SpaceID: "local", Kind: "kpi_drift", Enabled: true,
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
		if r.DutyID == "wd_kpi" {
			found = true
			if r.Status != "skipped" || r.Canceled != 0 {
				t.Fatalf("want skipped never-cancel: %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("want skipped kpi_drift run: %+v", st)
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
