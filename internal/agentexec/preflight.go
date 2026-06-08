package agentexec

import (
	"context"
	"fmt"
	"os/exec"
)

type PreflightCheck struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type PreflightResult struct {
	Agent           string           `json:"agent"`
	DefaultExecutor string           `json:"defaultExecutor"`
	Ready           bool             `json:"ready"`
	Checks          []PreflightCheck `json:"checks"`
	Issues          []string         `json:"issues"`
}

func StaticPreflight(defaultExecutor string) PreflightResult {
	return PreflightResult{
		Agent:           "static",
		DefaultExecutor: defaultExecutor,
		Ready:           true,
		Checks: []PreflightCheck{{
			ID: "static", OK: true, Message: "static executor is local",
		}},
	}
}

func ExecGoCodexPreflight(ctx context.Context, defaultExecutor string) PreflightResult {
	executor := NewExecGoCodexExecutor()
	out := PreflightResult{
		Agent:           executor.AdapterName(),
		DefaultExecutor: defaultExecutor,
		Ready:           true,
	}
	_, cliErr := exec.LookPath(executor.CLI)
	out.addCheck("execgocli", cliErr == nil, fmt.Sprintf("%s not found", executor.CLI))
	_, codexErr := exec.LookPath(executor.CodexBin)
	out.addCheck("codex", codexErr == nil, fmt.Sprintf("%s not found", executor.CodexBin))

	if out.Ready {
		_, _, _, err := executor.runJSON(ctx, "health")
		out.addCheck("execgoHealth", err == nil, errText("execgo health", err))
	}
	if out.Ready {
		_, _, _, err := executor.runJSON(ctx, "tools")
		out.addCheck("execgoTools", err == nil, errText("execgo tools", err))
	}
	return out
}

func (r *PreflightResult) addCheck(id string, ok bool, message string) {
	check := PreflightCheck{ID: id, OK: ok}
	if !ok {
		check.Message = message
		r.Ready = false
		r.Issues = append(r.Issues, message)
	} else {
		check.Message = "ok"
	}
	r.Checks = append(r.Checks, check)
}

func errText(prefix string, err error) string {
	if err == nil {
		return "ok"
	}
	return fmt.Sprintf("%s failed: %v", prefix, err)
}
