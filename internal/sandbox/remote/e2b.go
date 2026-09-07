package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

const (
	envRemoteTemplate = "ASH_SANDBOX_REMOTE_TEMPLATE"
	defaultE2BBaseURL = "https://api.e2b.app"
	defaultTemplate   = "base"
)

// E2BExecutor talks to an E2B-compatible HTTP control plane:
//
//	POST {base}/sandboxes
//	POST {base}/sandboxes/{id}/commands
//	DELETE {base}/sandboxes/{id}
//
// Auth: X-API-KEY / X-API-Key header. Not the default ASH executor — opt-in only.
type E2BExecutor struct {
	cfg        Config
	httpClient *http.Client
	templateID string
}

// NewE2BExecutor builds an E2B-class client from config.
func NewE2BExecutor(cfg Config) *E2BExecutor {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = defaultE2BBaseURL
	}
	cfg.BaseURL = base
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeoutSec * time.Second
	}
	tpl := strings.TrimSpace(os.Getenv(envRemoteTemplate))
	if tpl == "" {
		tpl = defaultTemplate
	}
	return &E2BExecutor{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout + 5*time.Second,
		},
		templateID: tpl,
	}
}

func (e *E2BExecutor) Name() string { return BackendE2B }

func (e *E2BExecutor) Available() bool {
	if e == nil {
		return false
	}
	// Need API key and/or explicit base URL (self-hosted / test gateway).
	return strings.TrimSpace(e.cfg.APIKey) != "" || (e.cfg.BaseURL != "" && e.cfg.BaseURL != defaultE2BBaseURL)
}

func (e *E2BExecutor) UnavailableReason() string {
	if e == nil {
		return "e2b executor is nil"
	}
	if !e.Available() {
		return "e2b: set ASH_SANDBOX_REMOTE_API_KEY (and optional ASH_SANDBOX_REMOTE_URL)"
	}
	return ""
}

func (e *E2BExecutor) Capabilities() Capabilities {
	return Capabilities{CreateSession: true, Cancel: true}
}

func (e *E2BExecutor) Dispatch(ctx context.Context, req sandbox.DispatchRequest) (*sandbox.DispatchResult, error) {
	if e == nil {
		return nil, fmt.Errorf("e2b executor is nil")
	}
	if !e.Available() {
		return nil, fmt.Errorf("%s", e.UnavailableReason())
	}
	timeout := e.cfg.Timeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	if timeout <= 0 {
		timeout = defaultTimeoutSec * time.Second
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	program := strings.TrimSpace(req.Program)
	if program == "" {
		return &sandbox.DispatchResult{OK: false, Error: "program is required"}, nil
	}
	cmd := shellJoin(program, req.Args)

	sandboxID, err := e.createSandbox(ctx, int(timeout.Seconds()), req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = e.killSandbox(context.Background(), sandboxID)
	}()

	cwd := strings.TrimSpace(req.RepoRoot)
	if cwd == "" {
		cwd = "/home/user"
	}
	out, err := e.runCommand(ctx, sandboxID, cmd, cwd)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type e2bCreateReq struct {
	TemplateID string            `json:"templateID"`
	Timeout    int               `json:"timeout,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	EnvVars    map[string]string `json:"envVars,omitempty"`
}

type e2bCreateResp struct {
	SandboxID string `json:"sandboxID"`
	ID        string `json:"id"`
}

type e2bCommandReq struct {
	Cmd     string `json:"cmd,omitempty"`
	Command string `json:"command,omitempty"`
	Cwd     string `json:"cwd,omitempty"`
}

type e2bCommandResp struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Exit     int    `json:"exit"`
	Error    string `json:"error"`
}

func (e *E2BExecutor) createSandbox(ctx context.Context, ttlSec int, req sandbox.DispatchRequest) (string, error) {
	if ttlSec < 15 {
		ttlSec = 15
	}
	body := e2bCreateReq{
		TemplateID: e.templateID,
		Timeout:    ttlSec,
		Metadata: map[string]string{
			"ash.runId":  strings.TrimSpace(req.RunID),
			"ash.stepId": strings.TrimSpace(req.StepID),
		},
		EnvVars: map[string]string{
			"ASH_SANDBOX_REMOTE": "1",
			"ASH_RUN_ID":         strings.TrimSpace(req.RunID),
			"ASH_STEP_ID":        strings.TrimSpace(req.StepID),
		},
	}
	raw, err := e.doJSON(ctx, http.MethodPost, e.cfg.BaseURL+"/sandboxes", body)
	if err != nil {
		return "", err
	}
	var resp e2bCreateResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("e2b create decode: %w", err)
	}
	id := firstNonEmpty(resp.SandboxID, resp.ID)
	if id == "" {
		return "", fmt.Errorf("e2b create: missing sandboxID")
	}
	return id, nil
}

func (e *E2BExecutor) runCommand(ctx context.Context, sandboxID, cmd, cwd string) (*sandbox.DispatchResult, error) {
	body := e2bCommandReq{Cmd: cmd, Command: cmd, Cwd: cwd}
	raw, err := e.doJSON(ctx, http.MethodPost, e.cfg.BaseURL+"/sandboxes/"+sandboxID+"/commands", body)
	if err != nil {
		return nil, err
	}
	var resp e2bCommandResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("e2b command decode: %w", err)
	}
	exit := resp.ExitCode
	if resp.Exit != 0 && exit == 0 {
		exit = resp.Exit
	}
	out := &sandbox.DispatchResult{
		Stdout:   truncate(resp.Stdout),
		Stderr:   truncate(resp.Stderr),
		ExitCode: exit,
		Error:    resp.Error,
	}
	out.OK = exit == 0 && resp.Error == ""
	if !out.OK && out.Error == "" {
		out.Error = fmt.Sprintf("remote exit %d", exit)
	}
	return out, nil
}

func (e *E2BExecutor) killSandbox(ctx context.Context, sandboxID string) error {
	if sandboxID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, e.cfg.BaseURL+"/sandboxes/"+sandboxID, nil)
	if err != nil {
		return err
	}
	e.applyAuth(req)
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("e2b kill: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (e *E2BExecutor) doJSON(ctx context.Context, method, url string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	e.applyAuth(req)
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 512 {
			msg = msg[:512]
		}
		return nil, fmt.Errorf("e2b %s %s: HTTP %d %s", method, url, resp.StatusCode, msg)
	}
	return raw, nil
}

func (e *E2BExecutor) applyAuth(req *http.Request) {
	key := strings.TrimSpace(e.cfg.APIKey)
	if key == "" {
		return
	}
	req.Header.Set("X-API-KEY", key)
	req.Header.Set("X-API-Key", key)
}

func shellJoin(program string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, shellQuote(program))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
