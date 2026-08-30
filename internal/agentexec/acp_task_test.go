package agentexec

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestACPTaskV1Validate(t *testing.T) {
	ok, err := NewACPTaskV1("ash-acp", Request{Prompt: "hi", RunID: "r1"})
	if err != nil {
		t.Fatal(err)
	}
	if ok.Schema != ACPTaskSchemaV1 || ok.SessionID != "r1" {
		t.Fatalf("%+v", ok)
	}
	_, err = NewACPTaskV1("ash-acp", Request{})
	if err == nil {
		t.Fatal("expected prompt/issue required")
	}
	bad := ACPTaskV1{Schema: "other", AgentID: "a", Prompt: "x"}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected schema error")
	}
}

func TestParseACPTaskResultV1(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"schema": ACPTaskSchemaV1, "ok": true, "taskId": "t1", "status": "success",
	})
	res, err := ParseACPTaskResultV1(raw)
	if err != nil || res.EffectiveStatus() != "success" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	_, err = ParseACPTaskResultV1([]byte(`{"ok":false,"schema":"nope"}`))
	if err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("err=%v", err)
	}
}

func TestACPSmokeAgainstEnv(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("ASH_ACP_ENDPOINT"))
	if endpoint == "" {
		t.Skip("ASH_ACP_ENDPOINT not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rep := ProbeACP(ctx)
	if !rep.OK {
		t.Fatalf("probe=%+v", rep)
	}
	e := NewACPExecutor()
	res, err := e.Execute(ctx, Request{
		RunID: "run_smoke", StepID: "s1", Prompt: "acp-smoke",
		Metadata: map[string]any{"sessionId": "sess_smoke"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.SessionID != "sess_smoke" || res.Status != "success" || res.TaskID == "" {
		t.Fatalf("res=%+v", res)
	}
}
