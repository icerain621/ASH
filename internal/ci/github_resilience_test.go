package ci

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGitHubProviderRetriesThenSucceeds(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"message":"bad gateway"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workflow_runs": []map[string]any{{
				"id": 42, "name": "CI", "status": "completed", "conclusion": "success",
				"run_attempt": 1, "html_url": "https://example/runs/42",
				"head_branch": "main", "head_sha": "abc", "created_at": time.Now().UTC().Format(time.RFC3339),
			}},
		})
	}))
	defer srv.Close()

	circuit := &githubCircuit{}
	p := GitHubProvider{
		Client: srv.Client(),
		Circuit: circuit,
		Sleep:   func(time.Duration) {},
		MaxAttempts: 3,
	}
	endpoint := srv.URL + "/repos/o/r/actions/runs"
	var payload struct {
		WorkflowRuns []struct {
			ID int64 `json:"id"`
		} `json:"workflow_runs"`
	}
	if err := p.getJSON(context.Background(), endpoint, "tok", &payload); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits=%d want 3", hits.Load())
	}
	if len(payload.WorkflowRuns) != 1 || payload.WorkflowRuns[0].ID != 42 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestGitHubProviderCircuitOpensAfterFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	circuit := &githubCircuit{}
	p := GitHubProvider{
		Client:      srv.Client(),
		Circuit:     circuit,
		Sleep:       func(time.Duration) {},
		MaxAttempts: 1,
	}
	endpoint := srv.URL + "/x"
	for i := 0; i < githubCircuitThreshold; i++ {
		err := p.getJSON(context.Background(), endpoint, "tok", &map[string]any{})
		if err == nil {
			t.Fatalf("attempt %d: expected error", i)
		}
		if errors.Is(err, ErrGitHubCircuitOpen) {
			t.Fatalf("circuit opened too early at attempt %d", i)
		}
	}
	err := p.getJSON(context.Background(), endpoint, "tok", &map[string]any{})
	if !errors.Is(err, ErrGitHubCircuitOpen) {
		t.Fatalf("err=%v want ErrGitHubCircuitOpen", err)
	}
}

func TestGitHubProviderDoesNotRetryAuthErrors(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	p := GitHubProvider{
		Client:      srv.Client(),
		Circuit:     &githubCircuit{},
		Sleep:       func(time.Duration) {},
		MaxAttempts: 3,
	}
	err := p.getJSON(context.Background(), srv.URL, "bad", &map[string]any{})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if hits.Load() != 1 {
		t.Fatalf("hits=%d want 1 (no retry on 401)", hits.Load())
	}
	var httpErr *githubHTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 401 {
		t.Fatalf("err=%v want githubHTTPError 401", err)
	}
}

func TestIsRetryableGitHubError(t *testing.T) {
	if !isRetryableGitHubError(&githubHTTPError{StatusCode: 429, Status: "429"}) {
		t.Fatal("429 should retry")
	}
	if !isRetryableGitHubError(&githubHTTPError{StatusCode: 503, Status: "503"}) {
		t.Fatal("503 should retry")
	}
	if isRetryableGitHubError(&githubHTTPError{StatusCode: 404, Status: "404"}) {
		t.Fatal("404 should not retry")
	}
	if isRetryableGitHubError(ErrGitHubCircuitOpen) {
		t.Fatal("circuit open should not retry")
	}
}
