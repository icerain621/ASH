package ci

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestFixtureProviderSyncRunsAndJobs(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	conn := store.RepoConnection{
		ID: "conn_fixture", SpaceID: "local", Provider: "github", Owner: "iammm0", Repo: "ASH",
		DefaultBranch: "main", SecretID: "secret_fixture", Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&conn).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, func(spaceID, secretID string) (string, error) {
		return "ghp_fixture", nil
	}).WithProvider("github", DefaultFixtureProvider())

	if err := svc.SyncRuns(context.Background(), "local", conn.ID, 10); err != nil {
		t.Fatal(err)
	}
	runs, err := svc.ListRuns(context.Background(), "local", conn.ID, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].ProviderRunID != "fixture-run-9001" || runs[0].Conclusion != "failure" {
		t.Fatalf("runs=%+v want fixture-run-9001 failure", runs)
	}

	jobs, err := svc.ListJobs(context.Background(), "local", runs[0].ID, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) < 5 || jobs[0].ProviderJobID != "fixture-job-9101" {
		t.Fatalf("jobs=%+v want >=5 fixture jobs starting with fixture-job-9101", jobs)
	}
	byProvider := map[string]store.CIJob{}
	for _, job := range jobs {
		byProvider[job.ProviderJobID] = job
	}

	logs, err := DefaultFixtureProvider().GetJobLogs(context.Background(), conn, "tok", "fixture-job-9101")
	if err != nil {
		t.Fatal(err)
	}
	if logs == "" || !strings.Contains(logs, "FAIL: TestAPI") {
		t.Fatalf("logs=%q want test failure excerpt", logs)
	}

	diag, err := svc.Diagnose(context.Background(), DiagnoseRequest{
		SpaceID: "local", JobID: byProvider["fixture-job-9101"].ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if diag.RootCause != "test_failure" || diag.LogDigest == "" {
		t.Fatalf("diag=%+v want test_failure with digest", diag)
	}
	var jobRow store.CIJob
	if err := db.First(&jobRow, "id = ?", byProvider["fixture-job-9101"].ID).Error; err != nil {
		t.Fatal(err)
	}
	if jobRow.LogDigest == "" || jobRow.LogDigest != diag.LogDigest {
		t.Fatalf("job logDigest=%q want %q", jobRow.LogDigest, diag.LogDigest)
	}

	dockerLogs, err := DefaultFixtureProvider().GetJobLogs(context.Background(), conn, "tok", "fixture-job-9102")
	if err != nil || !strings.Contains(dockerLogs, "Docker daemon") {
		t.Fatalf("docker logs=%q err=%v", dockerLogs, err)
	}
	dockerDiag, err := svc.Diagnose(context.Background(), DiagnoseRequest{
		SpaceID: "local", JobID: byProvider["fixture-job-9102"].ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dockerDiag.RootCause != "docker_or_postgres_unavailable" {
		t.Fatalf("dockerDiag=%+v want docker_or_postgres_unavailable", dockerDiag)
	}

	wantCauses := map[string]string{
		"fixture-job-9103": "actions_cancel_or_runner_abort",
		"fixture-job-9104": "runner_resource_exhaustion",
		"fixture-job-9105": "frontend_lint_or_typecheck_failure",
	}
	for providerJobID, want := range wantCauses {
		job, ok := byProvider[providerJobID]
		if !ok {
			t.Fatalf("missing fixture job %s", providerJobID)
		}
		got, err := svc.Diagnose(context.Background(), DiagnoseRequest{
			SpaceID: "local", JobID: job.ID,
		})
		if err != nil {
			t.Fatalf("%s diagnose: %v", providerJobID, err)
		}
		if got.RootCause != want {
			t.Fatalf("%s rootCause=%q want %q", providerJobID, got.RootCause, want)
		}
	}

	if err := svc.SyncRuns(context.Background(), "local", conn.ID, 10); err != nil {
		t.Fatal(err)
	}
	runsAgain, err := svc.ListRuns(context.Background(), "local", conn.ID, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(runsAgain) != 1 || runsAgain[0].ID != runs[0].ID {
		t.Fatalf("runsAgain=%+v want same id %s after re-sync", runsAgain, runs[0].ID)
	}
}

func TestApplyFixtureProviderWhenEnabled(t *testing.T) {
	t.Setenv("ASH_CI_FIXTURE", "1")
	db := store.OpenTest(t, t.TempDir())
	svc := ApplyFixtureProvider(NewService(db, func(spaceID, secretID string) (string, error) {
		return "ghp_fixture", nil
	}))
	if svc == nil {
		t.Fatal("nil service")
	}
	now := time.Now().UTC()
	conn := store.RepoConnection{
		ID: "conn_apply", SpaceID: "local", Provider: "github", Owner: "o", Repo: "r",
		DefaultBranch: "main", SecretID: "s", Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&conn).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.SyncRuns(context.Background(), "local", conn.ID, 5); err != nil {
		t.Fatal(err)
	}
	runs, err := svc.ListRuns(context.Background(), "local", conn.ID, 5, false)
	if err != nil || len(runs) != 1 || runs[0].Workflow != "CI" {
		t.Fatalf("runs=%+v err=%v want synced fixture run", runs, err)
	}
}