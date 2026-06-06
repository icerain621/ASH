package agentexec

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultExecGoURL        = "http://127.0.0.1:8080"
	defaultExecGoRuntimeURL = "http://127.0.0.1:18080"
)

// ExecGoCodexExecutor submits Codex work through execgocli and execgo-runtime.
type ExecGoCodexExecutor struct {
	CLI           string
	CodexBin      string
	ExecGoURL     string
	RuntimeURL    string
	AgentID       string
	BypassSandbox bool
}

func NewExecGoCodexExecutor() *ExecGoCodexExecutor {
	cli := firstNonEmpty(os.Getenv("EXECGO_EXECGOCLI"), "execgocli")
	codexBin := firstNonEmpty(os.Getenv("ASH_CODEX_BIN"), "codex")
	return &ExecGoCodexExecutor{
		CLI:        cli,
		CodexBin:   codexBin,
		ExecGoURL:  firstNonEmpty(os.Getenv("EXECGO_URL"), defaultExecGoURL),
		RuntimeURL: firstNonEmpty(os.Getenv("EXECGO_RUNTIME_URL"), defaultExecGoRuntimeURL),
		AgentID:    firstNonEmpty(os.Getenv("ASH_AGENT_ID"), "ash-codex"),
		BypassSandbox: envBool(
			os.Getenv("ASH_CODEX_BYPASS_SANDBOX"),
		),
	}
}

func (e *ExecGoCodexExecutor) AdapterName() string {
	return "execgo_codex"
}

func (e *ExecGoCodexExecutor) Plan(ctx context.Context, req Request) (*Result, error) {
	return e.Execute(ctx, req)
}

func (e *ExecGoCodexExecutor) Execute(ctx context.Context, req Request) (*Result, error) {
	start := time.Now()
	if err := e.health(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBridgeUnavailable, err)
	}

	timeout := req.TimeoutMs
	if timeout <= 0 {
		timeout = 120000
	}
	actionID := stableActionID(req.RunID, req.StepID)
	prompt := buildCodexPrompt(req)
	request := map[string]any{
		"adapter":    "codex",
		"agent_id":   e.AgentID,
		"session_id": req.RunID,
		"action_id":  actionID,
		"action": map[string]any{
			"kind":    "runtime.command",
			"timeout": timeout,
			"input": map[string]any{
				"program": e.CodexBin,
				"args":    e.codexArgs(prompt),
				"cwd":     req.RepoRoot,
				"limits": map[string]any{
					"wall_time_ms": timeout,
				},
				"sandbox": map[string]any{"profile": "process"},
			},
		},
		"metadata": map[string]any{
			"source":   "ash",
			"runId":    req.RunID,
			"traceId":  req.TraceID,
			"stepId":   req.StepID,
			"repoRoot": req.RepoRoot,
		},
	}

	path := filepath.Join(req.RunDir, "agent-"+req.StepID+".json")
	if err := os.MkdirAll(req.RunDir, 0o755); err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(request, "", "  ")
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return nil, err
	}

	act, stdout, stderr, err := e.runJSON(ctx, "act", "-file", path)
	if err != nil {
		return nil, classifyExecGoError(err)
	}
	taskID := firstString(act, "task_id")
	if taskID == "" {
		if ids, ok := act["task_ids"].([]any); ok && len(ids) > 0 {
			taskID, _ = ids[0].(string)
		}
	}
	if taskID == "" {
		taskID = actionID
	}

	wait, waitOut, waitErr, err := e.runJSON(ctx, "wait", "-task-ids", taskID)
	stdout = joinSummaries(stdout, waitOut)
	stderr = joinSummaries(stderr, waitErr)
	if err != nil {
		return nil, classifyExecGoError(err)
	}

	status := terminalStatus(wait)
	if status == "" {
		status = "success"
	}
	if status != "success" {
		return &Result{
			TaskID: taskID, ExecGoTaskID: taskID, Adapter: "codex", AgentID: e.AgentID,
			SessionID: req.RunID, ActionID: actionID, Status: status,
			StdoutSummary: trimSummary(stdout), StderrSummary: trimSummary(stderr),
			DurationMs: time.Since(start).Milliseconds(), Output: wait,
		}, fmt.Errorf("%w: agent task %s ended with status %s", ErrAgentTaskFailed, taskID, status)
	}

	return &Result{
		TaskID: taskID, ExecGoTaskID: taskID, Adapter: "codex", AgentID: e.AgentID,
		SessionID: req.RunID, ActionID: actionID, Status: status,
		StdoutSummary: trimSummary(stdout), StderrSummary: trimSummary(stderr),
		DurationMs: time.Since(start).Milliseconds(), Output: wait,
	}, nil
}

func (e *ExecGoCodexExecutor) codexArgs(prompt string) []string {
	args := []string{"exec", "--skip-git-repo-check"}
	if e.BypassSandbox {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}
	return append(args, prompt)
}

func (e *ExecGoCodexExecutor) Cancel(ctx context.Context, taskID string) error {
	if strings.TrimSpace(taskID) == "" {
		return nil
	}
	_, _, _, err := e.runJSON(ctx, "cancel", "-task-ids", taskID, "--wait")
	return err
}

func (e *ExecGoCodexExecutor) Status(ctx context.Context, taskID string) (*Status, error) {
	data, _, _, err := e.runJSON(ctx, "wait", "-task-ids", taskID)
	if err != nil {
		return nil, err
	}
	return &Status{TaskID: taskID, ExecGoTaskID: taskID, Status: terminalStatus(data), Output: data}, nil
}

func (e *ExecGoCodexExecutor) CollectArtifacts(_ context.Context, _ Request, _ Result) (map[string]string, error) {
	return map[string]string{}, nil
}

func (e *ExecGoCodexExecutor) health(ctx context.Context) error {
	if _, err := exec.LookPath(e.CLI); err != nil {
		return fmt.Errorf("execgocli not found: %w", err)
	}
	if _, err := exec.LookPath(e.CodexBin); err != nil {
		return fmt.Errorf("codex binary not found: %w", err)
	}
	if _, _, _, err := e.runJSON(ctx, "health"); err != nil {
		return fmt.Errorf("execgo health: %w", err)
	}
	if _, _, _, err := e.runJSON(ctx, "tools"); err != nil {
		return fmt.Errorf("execgo tools: %w", err)
	}
	return nil
}

func (e *ExecGoCodexExecutor) runJSON(ctx context.Context, args ...string) (map[string]any, string, string, error) {
	cmd := exec.CommandContext(ctx, e.CLI, args...)
	cmd.Env = append(os.Environ(),
		"EXECGO_URL="+e.ExecGoURL,
		"EXECGO_RUNTIME_URL="+e.RuntimeURL,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outText := stdout.String()
	errText := stderr.String()
	if err != nil {
		return nil, outText, errText, fmt.Errorf("%s %s: %w: %s", e.CLI, strings.Join(args, " "), err, trimSummary(errText))
	}
	var envelope struct {
		OK    bool           `json:"ok"`
		Data  map[string]any `json:"data"`
		Error *struct {
			Message    string `json:"message"`
			StatusCode int    `json:"status_code"`
			Body       string `json:"body"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		return nil, outText, errText, fmt.Errorf("parse execgocli JSON: %w: %s", err, trimSummary(outText))
	}
	if !envelope.OK {
		if envelope.Error != nil {
			return nil, outText, errText, fmt.Errorf("%s (status=%d body=%s)", envelope.Error.Message, envelope.Error.StatusCode, envelope.Error.Body)
		}
		return nil, outText, errText, fmt.Errorf("execgocli returned ok=false")
	}
	if envelope.Data == nil {
		envelope.Data = map[string]any{}
	}
	return envelope.Data, outText, errText, nil
}

func classifyExecGoError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "parse execgocli JSON") {
		return fmt.Errorf("%w: %v", ErrAgentOutputInvalid, err)
	}
	return fmt.Errorf("%w: %v", ErrAgentTaskFailed, err)
}

func buildCodexPrompt(req Request) string {
	var b strings.Builder
	b.WriteString("You are the coding executor for ASH. Implement the requested change in the repository.\n")
	b.WriteString("Return by modifying files only when necessary. Keep changes minimal, run relevant tests, and leave evidence in the working tree.\n\n")
	if req.Issue != "" {
		b.WriteString("Issue/spec:\n")
		b.WriteString(req.Issue)
		b.WriteString("\n\n")
	}
	if req.Prompt != "" {
		b.WriteString("Step prompt:\n")
		b.WriteString(req.Prompt)
		b.WriteString("\n\n")
	}
	b.WriteString("ASH run: " + req.RunID + "\nStep: " + req.StepID + "\n")
	return b.String()
}

func stableActionID(runID, stepID string) string {
	sum := sha256.Sum256([]byte(runID + ":" + stepID))
	return "ash-" + hex.EncodeToString(sum[:])[:20]
}

func terminalStatus(data map[string]any) string {
	if tasks, ok := data["tasks"].([]any); ok && len(tasks) > 0 {
		if task, ok := tasks[0].(map[string]any); ok {
			if st, ok := task["status"].(string); ok {
				return st
			}
			if st, ok := task["state"].(string); ok {
				return st
			}
		}
	}
	if st, ok := data["status"].(string); ok {
		return st
	}
	return ""
}

func firstString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func envBool(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func trimSummary(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 4000 {
		return s
	}
	return s[:4000] + "\n...<truncated>"
}

func joinSummaries(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n")
}
