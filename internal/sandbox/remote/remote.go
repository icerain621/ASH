// Package remote provides optional remote / microVM sandbox backends (v2.9).
//
// Default execution remains local (Landlock / process / docker). Remote is
// opt-in via ASH_SANDBOX_REMOTE; DefaultRouter prefers it for isolated mode
// when Available (DX39). ASH_SANDBOX_REMOTE_ON_FAIL=deny refuses instead of
// falling back to local executors.
package remote

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

const (
	envRemoteEnabled  = "ASH_SANDBOX_REMOTE"
	envRemoteBackend  = "ASH_SANDBOX_REMOTE_BACKEND"
	envRemoteURL      = "ASH_SANDBOX_REMOTE_URL"
	envRemoteAPIKey   = "ASH_SANDBOX_REMOTE_API_KEY"
	envRemoteTimeout  = "ASH_SANDBOX_REMOTE_TIMEOUT_SEC"
	BackendMock       = "mock"
	BackendE2B        = "e2b"
	defaultTimeoutSec = 60
)

// Capabilities describes optional remote features.
type Capabilities struct {
	CreateSession bool `json:"createSession"`
	Cancel        bool `json:"cancel"`
}

// Backend is a remote sandbox executor with discovery metadata.
type Backend interface {
	sandbox.Executor
	Name() string
	Available() bool
	Capabilities() Capabilities
}

// Config is loaded from environment (no secrets persisted).
type Config struct {
	Enabled bool
	Backend string
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// Status is a diagnostics snapshot for Scale / Observability (DX41).
type Status struct {
	Enabled    bool         `json:"enabled"`
	Preferred  bool         `json:"preferred"`
	DenyOnFail bool         `json:"denyOnFail"`
	Backend    string       `json:"backend,omitempty"`
	Available  bool         `json:"available"`
	Reason     string       `json:"reason,omitempty"`
	Caps       Capabilities `json:"capabilities"`
}

// ConfigFromEnv reads remote sandbox configuration.
// Remote is OFF unless ASH_SANDBOX_REMOTE is 1/true/on/auto/yes.
func ConfigFromEnv() Config {
	cfg := Config{
		Backend: strings.ToLower(strings.TrimSpace(os.Getenv(envRemoteBackend))),
		BaseURL: strings.TrimSpace(os.Getenv(envRemoteURL)),
		APIKey:  strings.TrimSpace(os.Getenv(envRemoteAPIKey)),
		Timeout: timeoutFromEnv(),
	}
	if cfg.Backend == "" {
		cfg.Backend = BackendMock
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envRemoteEnabled))) {
	case "1", "true", "on", "yes", "auto":
		cfg.Enabled = true
	default:
		cfg.Enabled = false
	}
	return cfg
}

func timeoutFromEnv() time.Duration {
	v := strings.TrimSpace(os.Getenv(envRemoteTimeout))
	if v == "" {
		return defaultTimeoutSec * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultTimeoutSec * time.Second
	}
	if n > 600 {
		n = 600
	}
	return time.Duration(n) * time.Second
}

// Resolve returns a Backend when remote is enabled; otherwise nil.
func Resolve() Backend {
	return ResolveWith(ConfigFromEnv())
}

// ResolveWith picks a backend for cfg.
func ResolveWith(cfg Config) Backend {
	if !cfg.Enabled {
		return nil
	}
	switch cfg.Backend {
	case BackendE2B:
		return NewE2BExecutor(cfg)
	case BackendMock, "":
		return NewMockExecutor(cfg)
	default:
		return unavailableBackend{name: cfg.Backend, reason: "unknown remote backend"}
	}
}

// ProbeStatus reports whether a remote backend is configured and ready.
func ProbeStatus() Status {
	cfg := ConfigFromEnv()
	st := Status{
		Enabled:    cfg.Enabled,
		Preferred:  sandbox.RemotePreferred(),
		DenyOnFail: sandbox.RemoteDenyOnFail(),
		Backend:    cfg.Backend,
	}
	if !cfg.Enabled {
		st.Reason = "ASH_SANDBOX_REMOTE disabled (default local)"
		return st
	}
	be := ResolveWith(cfg)
	if be == nil {
		st.Reason = "no backend"
		return st
	}
	st.Available = be.Available()
	st.Caps = be.Capabilities()
	st.Backend = be.Name()
	if !st.Available {
		if u, ok := be.(interface{ UnavailableReason() string }); ok {
			st.Reason = u.UnavailableReason()
		} else {
			st.Reason = "backend unavailable"
		}
	}
	return st
}

// Available is a package-level probe for Doctor / router hooks.
func Available() bool {
	return ProbeStatus().Available
}

// MockExecutor runs commands locally but labels them as remote=mock.
// Used for contract tests without cloud credentials.
type MockExecutor struct {
	cfg Config
}

func NewMockExecutor(cfg Config) *MockExecutor {
	return &MockExecutor{cfg: cfg}
}

func (m *MockExecutor) Name() string { return BackendMock }

func (m *MockExecutor) Available() bool { return true }

func (m *MockExecutor) Capabilities() Capabilities {
	return Capabilities{CreateSession: true, Cancel: true}
}

func (m *MockExecutor) Dispatch(ctx context.Context, req sandbox.DispatchRequest) (*sandbox.DispatchResult, error) {
	if m == nil {
		return nil, fmt.Errorf("remote mock executor is nil")
	}
	timeout := m.cfg.Timeout
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

	root := strings.TrimSpace(req.RepoRoot)
	if root == "" {
		root = os.TempDir()
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	program := strings.TrimSpace(req.Program)
	if program == "" {
		return &sandbox.DispatchResult{OK: false, Error: "program is required"}, nil
	}

	cmd := exec.CommandContext(ctx, program, req.Args...)
	cmd.Dir = absRoot
	cmd.Env = append(os.Environ(),
		"ASH_SANDBOX_REMOTE=1",
		"ASH_SANDBOX_REMOTE_BACKEND=mock",
		"ASH_RUN_ID="+strings.TrimSpace(req.RunID),
		"ASH_STEP_ID="+strings.TrimSpace(req.StepID),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	res := &sandbox.DispatchResult{
		Stdout: truncate(stdout.String()),
		Stderr: truncate(stderr.String()),
	}
	if err != nil {
		res.OK = false
		res.Error = err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res, nil
	}
	res.OK = true
	res.ExitCode = 0
	return res, nil
}

type unavailableBackend struct {
	name   string
	reason string
}

func (u unavailableBackend) Name() string               { return u.name }
func (u unavailableBackend) Available() bool            { return false }
func (u unavailableBackend) UnavailableReason() string  { return u.reason }
func (u unavailableBackend) Capabilities() Capabilities { return Capabilities{} }
func (u unavailableBackend) Dispatch(context.Context, sandbox.DispatchRequest) (*sandbox.DispatchResult, error) {
	return nil, fmt.Errorf("remote backend %s unavailable: %s", u.name, u.reason)
}

func truncate(s string) string {
	const max = 64 * 1024
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]"
}
