package agentexec

import (
	"context"
	"errors"
	"testing"
)

func TestExecGoRejectsMissingContextPack(t *testing.T) {
	exec := &ExecGoCodexExecutor{CLI: "unused"}
	_, err := exec.Execute(context.Background(), Request{RunID: "r", StepID: "s", Prompt: "x"})
	if !errors.Is(err, ErrEffectGate) {
		t.Fatalf("err=%v", err)
	}
}

func TestACPRejectsMissingContextPack(t *testing.T) {
	exec := &ACPExecutor{Endpoint: "http://127.0.0.1:9", AgentID: "t"}
	_, err := exec.Execute(context.Background(), Request{RunID: "r", StepID: "s", Prompt: "x"})
	if !errors.Is(err, ErrEffectGate) {
		t.Fatalf("err=%v", err)
	}
}
