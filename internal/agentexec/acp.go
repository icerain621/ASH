package agentexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ACPExecutor is the Agent Client Protocol (ACP) control-plane bridge skeleton (Sprint DW).
// Configure ASH_ACP_ENDPOINT (HTTP base). Optional ASH_ACP_BIN is used only for probe hints.
type ACPExecutor struct {
	Endpoint string
	Bin      string
	AgentID  string
	Client   *http.Client
}

// NewACPExecutor builds an ACP executor from env.
func NewACPExecutor() *ACPExecutor {
	return &ACPExecutor{
		Endpoint: strings.TrimRight(strings.TrimSpace(firstNonEmpty(os.Getenv("ASH_ACP_ENDPOINT"), os.Getenv("ASH_ACP_URL"))), "/"),
		Bin:      firstNonEmpty(os.Getenv("ASH_ACP_BIN"), ""),
		AgentID:  firstNonEmpty(os.Getenv("ASH_ACP_AGENT_ID"), "ash-acp"),
		Client:   &http.Client{Timeout: 3 * time.Second},
	}
}

func (e *ACPExecutor) AdapterName() string {
	return "acp_sdk"
}

func (e *ACPExecutor) Plan(ctx context.Context, req Request) (*Result, error) {
	return e.Execute(ctx, req)
}

func (e *ACPExecutor) Execute(ctx context.Context, req Request) (*Result, error) {
	start := time.Now()
	if err := e.health(ctx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBridgeUnavailable, err)
	}
	timeout := req.TimeoutMs
	if timeout <= 0 {
		timeout = 120000
	}
	body := map[string]any{
		"schema":    "ash.acp.task.v1",
		"agentId":   e.AgentID,
		"runId":     req.RunID,
		"traceId":   req.TraceID,
		"stepId":    req.StepID,
		"role":      req.Role,
		"repoRoot":  req.RepoRoot,
		"prompt":    req.Prompt,
		"issue":     req.Issue,
		"timeoutMs": timeout,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Endpoint+"/v1/tasks", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := e.client().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: acp task post: %v", ErrBridgeUnavailable, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("%w: acp task HTTP %d: %s", ErrAgentTaskFailed, resp.StatusCode, truncate(msg, 240))
	}
	var parsed struct {
		OK      bool           `json:"ok"`
		TaskID  string         `json:"taskId"`
		Status  string         `json:"status"`
		Output  map[string]any `json:"output"`
		Message string         `json:"message"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("%w: acp task decode: %v", ErrAgentOutputInvalid, err)
	}
	status := parsed.Status
	if status == "" {
		if parsed.OK {
			status = "success"
		} else {
			status = "failed"
		}
	}
	if !parsed.OK && status != "success" {
		msg := parsed.Message
		if msg == "" {
			msg = "acp task not ok"
		}
		return nil, fmt.Errorf("%w: %s", ErrAgentTaskFailed, msg)
	}
	taskID := parsed.TaskID
	if taskID == "" {
		taskID = "acp-" + req.StepID
	}
	return &Result{
		TaskID: taskID, ExecGoTaskID: taskID,
		Adapter: e.AdapterName(), AgentID: e.AgentID, SessionID: req.RunID,
		ActionID: taskID, Status: status,
		StdoutSummary: firstNonEmpty(parsed.Message, "acp task completed"),
		DurationMs:    time.Since(start).Milliseconds(),
		Output:        parsed.Output,
	}, nil
}

func (e *ACPExecutor) Cancel(ctx context.Context, taskID string) error {
	if strings.TrimSpace(e.Endpoint) == "" || strings.TrimSpace(taskID) == "" {
		return nil
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Endpoint+"/v1/tasks/"+taskID+"/cancel", nil)
	if err != nil {
		return err
	}
	resp, err := e.client().Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	return nil
}

func (e *ACPExecutor) Status(ctx context.Context, taskID string) (*Status, error) {
	if strings.TrimSpace(e.Endpoint) == "" {
		return nil, fmt.Errorf("%w: ASH_ACP_ENDPOINT not set", ErrBridgeUnavailable)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, e.Endpoint+"/v1/tasks/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := e.client().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBridgeUnavailable, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: acp status HTTP %d", ErrAgentTaskFailed, resp.StatusCode)
	}
	var parsed struct {
		TaskID string         `json:"taskId"`
		Status string         `json:"status"`
		Output map[string]any `json:"output"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentOutputInvalid, err)
	}
	id := parsed.TaskID
	if id == "" {
		id = taskID
	}
	return &Status{TaskID: id, ExecGoTaskID: id, Status: parsed.Status, Output: parsed.Output}, nil
}

func (e *ACPExecutor) CollectArtifacts(_ context.Context, _ Request, _ Result) (map[string]string, error) {
	return map[string]string{}, nil
}

func (e *ACPExecutor) client() *http.Client {
	if e != nil && e.Client != nil {
		return e.Client
	}
	return &http.Client{Timeout: 3 * time.Second}
}

func (e *ACPExecutor) health(ctx context.Context) error {
	if e == nil || strings.TrimSpace(e.Endpoint) == "" {
		return fmt.Errorf("ASH_ACP_ENDPOINT not set")
	}
	paths := []string{"/readyz", "/health", "/"}
	var last error
	for _, p := range paths {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, e.Endpoint+p, nil)
		if err != nil {
			return err
		}
		resp, err := e.client().Do(httpReq)
		if err != nil {
			last = err
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return nil
		}
		last = fmt.Errorf("HTTP %d from %s", resp.StatusCode, p)
	}
	if last != nil {
		return last
	}
	return fmt.Errorf("acp endpoint unreachable")
}

// ProbeACP reports ACP control-plane readiness (no task execution).
func ProbeACP(ctx context.Context) ProbeReport {
	now := time.Now().UTC().Unix()
	e := NewACPExecutor()
	if strings.TrimSpace(e.Endpoint) == "" {
		msg := "acp_sdk not configured (set ASH_ACP_ENDPOINT)"
		if strings.TrimSpace(e.Bin) != "" {
			if _, err := exec.LookPath(e.Bin); err == nil {
				msg = "ASH_ACP_BIN present; set ASH_ACP_ENDPOINT for live probe/execute"
			} else {
				msg = "ASH_ACP_BIN not found on PATH; set ASH_ACP_ENDPOINT"
			}
		}
		return ProbeReport{
			Adapter: e.AdapterName(), Kind: "acp_sdk", OK: false,
			Message: msg, CheckedAt: now,
		}
	}
	if err := e.health(ctx); err != nil {
		return ProbeReport{
			Adapter: e.AdapterName(), Kind: "acp_sdk", OK: false,
			Message: err.Error(), CheckedAt: now,
		}
	}
	return ProbeReport{
		Adapter: e.AdapterName(), Kind: "acp_sdk", OK: true,
		Message: "acp endpoint reachable", CheckedAt: now,
	}
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
