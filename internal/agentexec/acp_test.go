package agentexec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProbeACPNotConfigured(t *testing.T) {
	t.Setenv("ASH_ACP_ENDPOINT", "")
	t.Setenv("ASH_ACP_URL", "")
	t.Setenv("ASH_ACP_BIN", "")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rep := ProbeACP(ctx)
	if rep.OK || rep.Kind != "acp_sdk" {
		t.Fatalf("probe=%+v", rep)
	}
	if !strings.Contains(rep.Message, "ASH_ACP_ENDPOINT") {
		t.Fatalf("message=%q", rep.Message)
	}
}

func TestACPExecutorHealthAndExecute(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"taskId":"t1","status":"success","message":"done","output":{"n":1}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	e := &ACPExecutor{Endpoint: srv.URL, AgentID: "test", Client: srv.Client()}
	ctx := context.Background()
	if err := e.health(ctx); err != nil {
		t.Fatal(err)
	}
	res, err := e.Execute(ctx, Request{RunID: "r1", StepID: "s1", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TaskID != "t1" || res.Adapter != "acp_sdk" || res.Status != "success" {
		t.Fatalf("res=%+v", res)
	}

	t.Setenv("ASH_ACP_ENDPOINT", srv.URL)
	t.Setenv("ASH_ACP_URL", "")
	rep := ProbeACP(ctx)
	if !rep.OK {
		t.Fatalf("probe=%+v", rep)
	}
}
