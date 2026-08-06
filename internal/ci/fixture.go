package ci

import (
	"context"
	"os"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

// FixtureEnabled reports whether CI sync should use embedded fixture data instead of GitHub API.
func FixtureEnabled() bool {
	return os.Getenv("ASH_CI_FIXTURE") == "1"
}

// FixtureProvider serves deterministic workflow runs/jobs/logs for local and CI tests (H-04/H-05).
type FixtureProvider struct {
	Runs []store.CIRun
	Jobs []store.CIJob
	Logs map[string]string
}

// DefaultFixtureProvider returns a representative failed CI workflow with test job logs.
func DefaultFixtureProvider() FixtureProvider {
	now := time.Now().UTC().Add(-15 * time.Minute)
	completed := now.Add(8 * time.Minute)
	return FixtureProvider{
		Runs: []store.CIRun{{
			ProviderRunID: "fixture-run-9001",
			Workflow:      "CI",
			Status:        "completed",
			Conclusion:    "failure",
			Attempt:       1,
			CommitSHA:     "fixturesha9001",
			Branch:        "main",
			RunURL:        "https://github.com/example/ASH/actions/runs/9001",
			StartedAt:     &now,
			CompletedAt:   &completed,
		}},
		Jobs: []store.CIJob{{
			ProviderJobID: "fixture-job-9101",
			Name:          "Backend and Doctor Gates",
			Status:        "completed",
			Conclusion:    "failure",
			StartedAt:     &now,
			CompletedAt:   &completed,
		}, {
			ProviderJobID: "fixture-job-9102",
			Name:          "Postgres E2E",
			Status:        "completed",
			Conclusion:    "failure",
			StartedAt:     &now,
			CompletedAt:   &completed,
		}, {
			ProviderJobID: "fixture-job-9103",
			Name:          "Canceled workflow",
			Status:        "completed",
			Conclusion:    "cancelled",
			StartedAt:     &now,
			CompletedAt:   &completed,
		}, {
			ProviderJobID: "fixture-job-9104",
			Name:          "Runner disk full",
			Status:        "completed",
			Conclusion:    "failure",
			StartedAt:     &now,
			CompletedAt:   &completed,
		}, {
			ProviderJobID: "fixture-job-9105",
			Name:          "Frontend gate",
			Status:        "completed",
			Conclusion:    "failure",
			StartedAt:     &now,
			CompletedAt:   &completed,
		}},
		Logs: map[string]string{
			"fixture-job-9101": "go test ./...\n--- FAIL: TestAPI (0.02s)\nFAIL\tgithub.com/ash-repwiki/ash/internal/api\t0.2s\n",
			"fixture-job-9102": "Starting postgres service\nCannot connect to the Docker daemon at unix:///var/run/docker.sock\n",
			"fixture-job-9103": "##[error]The job was canceled because the workflow was cancelled.\n",
			"fixture-job-9104": "mkdir: cannot create directory '/tmp/ash-ci': No space left on device\n",
			"fixture-job-9105": "src/pages/CIPage.tsx(12,5): error TS2322: Type 'string' is not assignable to type 'number'.\n",
		},
	}
}

func (p FixtureProvider) ListWorkflowRuns(context.Context, store.RepoConnection, string, int) ([]store.CIRun, error) {
	return append([]store.CIRun(nil), p.Runs...), nil
}

func (p FixtureProvider) GetRunJobs(context.Context, store.RepoConnection, string, string) ([]store.CIJob, error) {
	return append([]store.CIJob(nil), p.Jobs...), nil
}

func (p FixtureProvider) GetJobLogs(ctx context.Context, conn store.RepoConnection, token, providerJobID string) (string, error) {
	if p.Logs == nil {
		return "", nil
	}
	return p.Logs[providerJobID], nil
}

// ApplyFixtureProvider registers the default fixture provider on github when ASH_CI_FIXTURE=1.
func ApplyFixtureProvider(svc *Service) *Service {
	if svc == nil || !FixtureEnabled() {
		return svc
	}
	return svc.WithProvider("github", DefaultFixtureProvider())
}
