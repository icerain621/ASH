package remote

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

func TestE2BExecutorAgainstGateway(t *testing.T) {
	var created, commanded, killed atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/sandboxes":
			created.Add(1)
			if r.Header.Get("X-API-Key") == "" && r.Header.Get("X-API-KEY") == "" {
				http.Error(w, "missing key", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"sandboxID": "sbx_test_1"})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/sandboxes/") && strings.HasSuffix(r.URL.Path, "/commands"):
			commanded.Add(1)
			body, _ := io.ReadAll(r.Body)
			var req e2bCommandReq
			_ = json.Unmarshal(body, &req)
			if req.Cmd == "" && req.Command == "" {
				http.Error(w, "missing cmd", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stdout":   "hello-remote\n",
				"stderr":   "",
				"exitCode": 0,
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/sandboxes/"):
			killed.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := Config{
		Enabled: true,
		Backend: BackendE2B,
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	be := NewE2BExecutor(cfg)
	if !be.Available() {
		t.Fatal("want available")
	}
	res, err := be.Dispatch(context.Background(), sandbox.DispatchRequest{
		Program:  "echo",
		Args:     []string{"hello-remote"},
		RepoRoot: "/tmp",
		RunID:    "run_1",
		StepID:   "step_1",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.OK || !strings.Contains(res.Stdout, "hello-remote") {
		t.Fatalf("res=%+v", res)
	}
	if created.Load() != 1 || commanded.Load() != 1 || killed.Load() != 1 {
		t.Fatalf("created=%d commanded=%d killed=%d", created.Load(), commanded.Load(), killed.Load())
	}
}

func TestE2BDispatchCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/sandboxes" {
			time.Sleep(2 * time.Second)
			_ = json.NewEncoder(w).Encode(map[string]string{"sandboxID": "sbx_slow"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	be := NewE2BExecutor(Config{BaseURL: srv.URL, APIKey: "k", Timeout: time.Second})
	_, err := be.Dispatch(ctx, sandbox.DispatchRequest{Program: "true"})
	if err == nil {
		t.Fatal("expected cancel/timeout error")
	}
}

// TestLiveE2B hits a real E2B-compatible control plane when explicitly enabled.
// Default CI: skipped. No API key with LIVE=1 → skip pass (DX40).
func TestLiveE2B(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ASH_SANDBOX_REMOTE_LIVE")) != "1" {
		t.Skip("optional live: set ASH_SANDBOX_REMOTE_LIVE=1")
	}
	key := strings.TrimSpace(os.Getenv(envRemoteAPIKey))
	if key == "" {
		t.Skip("no ASH_SANDBOX_REMOTE_API_KEY; skip pass")
	}
	t.Setenv(envRemoteEnabled, "1")
	t.Setenv(envRemoteBackend, BackendE2B)
	be := Resolve()
	if be == nil || !be.Available() {
		t.Fatalf("live e2b unavailable: %+v", ProbeStatus())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	res, err := be.Dispatch(ctx, sandbox.DispatchRequest{
		Program: "echo",
		Args:    []string{"ash-remote-live"},
		Timeout: 60 * time.Second,
		RunID:   "live_smoke",
		StepID:  "dx40",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.OK {
		t.Fatalf("live dispatch failed: %+v", res)
	}
}
