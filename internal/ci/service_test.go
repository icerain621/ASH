package ci

import (
	"context"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestDiagnoseLogClassifiesTestFailure(t *testing.T) {
	resp := DiagnoseLog("go test ./...\n--- FAIL: TestThing (0.01s)\nFAIL\tgithub.com/acme/app\t0.1s\n")
	if resp.RootCause != "test_failure" {
		t.Fatalf("rootCause=%q want test_failure", resp.RootCause)
	}
	if resp.Confidence <= 0.8 || len(resp.EvidenceRefs) == 0 || resp.LogDigest == "" {
		t.Fatalf("resp=%+v want confident evidence with digest", resp)
	}
}

func TestDiagnoseLogClassifiesCancelAndResourceAndFrontend(t *testing.T) {
	cases := []struct {
		name string
		log  string
		want string
	}{
		{
			name: "actions cancel",
			log:  "##[error]The job was canceled because the workflow was cancelled.\n",
			want: "actions_cancel_or_runner_abort",
		},
		{
			name: "disk full",
			log:  "mkdir: cannot create directory '/tmp/x': No space left on device\n",
			want: "runner_resource_exhaustion",
		},
		{
			name: "typescript",
			log:  "src/pages/CIPage.tsx(12,5): error TS2322: Type 'string' is not assignable to type 'number'.\n",
			want: "frontend_lint_or_typecheck_failure",
		},
		{
			name: "vitest lifecycle before generic npm ERR",
			log:  "npm ERR! code ELIFECYCLE\nnpm ERR! errno 1\nnpm ERR! ash-frontend@0.0.0 test: `vitest run`\n",
			want: "frontend_lint_or_typecheck_failure",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := DiagnoseLog(tc.log)
			if resp.RootCause != tc.want {
				t.Fatalf("rootCause=%q want %q evidence=%v", resp.RootCause, tc.want, resp.EvidenceRefs)
			}
			if len(resp.EvidenceRefs) == 0 || resp.LogDigest == "" {
				t.Fatalf("resp=%+v want evidence and digest", resp)
			}
		})
	}
}

func TestServiceDiagnosePersistsLogText(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil)
	resp, err := svc.Diagnose(context.Background(), DiagnoseRequest{
		SpaceID: "local",
		LogText: "go test ./...\nundefined: MissingType\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RootCause != "go_compile_failure" {
		t.Fatalf("rootCause=%q want go_compile_failure", resp.RootCause)
	}
	var rows []store.CIDiagnosis
	if err := db.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RootCause != resp.RootCause || rows[0].LogDigest == "" {
		t.Fatalf("rows=%+v want persisted diagnosis", rows)
	}
}

func TestServiceSyncJobsDiagnoseAndDecide(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	conn := store.RepoConnection{
		ID: "conn_ci_decide", SpaceID: "local", Provider: "github", Owner: "iammm0", Repo: "ASH",
		DefaultBranch: "main", SecretID: "secret_ci", Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	run := store.CIRun{
		ID: "ci_run_decide", SpaceID: "local", ConnectionID: conn.ID, ProviderRunID: "100",
		Workflow: "ci", Status: "completed", Conclusion: "failure", Attempt: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&conn).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	provider := fakeProvider{
		jobs: []store.CIJob{{
			ID: "ci_job_decide", ProviderJobID: "200", Name: "test",
			Status: "completed", Conclusion: "failure", CreatedAt: now, UpdatedAt: now,
		}},
		logs: "go test ./...\n--- FAIL: TestThing\nFAIL\tpkg\t0.1s\n",
	}
	svc := NewService(db, func(spaceID, secretID string) (string, error) { return "ghp_test", nil }).WithProvider("github", provider)

	jobs, err := svc.ListJobs(context.Background(), "local", run.ID, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].CIRunID != run.ID || jobs[0].ProviderJobID != "200" {
		t.Fatalf("jobs=%+v want synced job", jobs)
	}

	diag, err := svc.Diagnose(context.Background(), DiagnoseRequest{SpaceID: "local", JobID: jobs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	if diag.RootCause != "test_failure" || diag.DecisionStatus != "pending" {
		t.Fatalf("diag=%+v want pending test_failure", diag)
	}
	listed, err := svc.ListDiagnoses(ListDiagnosesRequest{SpaceID: "local", DecisionStatus: "pending"})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != diag.ID {
		t.Fatalf("listed=%+v want diagnosis", listed)
	}
	decided, err := svc.DecideDiagnosis(DecideDiagnosisRequest{
		SpaceID: "local", DiagnosisID: diag.ID, Decision: "adopted", Reason: "fix matched", ActorID: "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !decided.Adopted || decided.DecisionStatus != "adopted" || decided.DecidedBy != "dev" {
		t.Fatalf("decided=%+v want adopted by dev", decided)
	}
}

type fakeProvider struct {
	runs []store.CIRun
	jobs []store.CIJob
	logs string
}

func (p fakeProvider) ListWorkflowRuns(context.Context, store.RepoConnection, string, int) ([]store.CIRun, error) {
	return p.runs, nil
}

func (p fakeProvider) GetRunJobs(context.Context, store.RepoConnection, string, string) ([]store.CIJob, error) {
	return p.jobs, nil
}

func (p fakeProvider) GetJobLogs(context.Context, store.RepoConnection, string, string) (string, error) {
	return p.logs, nil
}
