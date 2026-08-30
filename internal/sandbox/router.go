package sandbox

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Mode values align with ash.harness.profile.v1 sandbox.defaultMode.
const (
	ModeOff            = "off"
	ModeReadOnly       = "read-only"
	ModeWorkspaceWrite = "workspace-write"
	ModeIsolated       = "isolated"
)

// RouteRequest is input to sandbox routing.
type RouteRequest struct {
	Tool               string
	Risk               string
	ProfileDefaultMode string
	ModeOverride       string
	ScenarioMinMode    string
	PolicyProfile      string
	RepoRoot           string
	RunID              string
	StepID             string
}

// Decision records how a tool call should execute.
type Decision struct {
	Mode     string `json:"mode"`
	Executor string `json:"executor"` // in-process | process | docker | none
	Reason   string `json:"reason,omitempty"`
	Denied   bool   `json:"denied,omitempty"`
}

// Router selects sandbox execution strategy.
type Router interface {
	Route(req RouteRequest) (Decision, error)
}

// NoopRouter always dispatches in-process (kept for tests).
type NoopRouter struct{}

func (NoopRouter) Route(req RouteRequest) (Decision, error) {
	mode := ResolveSandboxModeExt(req.Risk, req.ProfileDefaultMode, req.ModeOverride, req.ScenarioMinMode, req.PolicyProfile)
	if err := Authorize(req.Risk, mode); err != nil {
		return Decision{Mode: mode, Executor: "none", Reason: err.Error(), Denied: true}, err
	}
	return Decision{Mode: mode, Executor: "in-process", Reason: "noop-router"}, nil
}

// DefaultRouter applies policy and picks process/docker when mode != off.
type DefaultRouter struct {
	PreferDocker bool
	DockerOK     func() bool
}

func NewDefaultRouter() DefaultRouter {
	return DefaultRouter{
		PreferDocker: strings.TrimSpace(os.Getenv("ASH_SKIP_SANDBOX")) == "",
		DockerOK:     DockerAvailable,
	}
}

func (r DefaultRouter) Route(req RouteRequest) (Decision, error) {
	mode := ResolveSandboxModeExt(req.Risk, req.ProfileDefaultMode, req.ModeOverride, req.ScenarioMinMode, req.PolicyProfile)
	if err := Authorize(req.Risk, mode); err != nil {
		return Decision{Mode: mode, Executor: "none", Reason: err.Error(), Denied: true}, err
	}
	if mode == ModeOff {
		return Decision{Mode: mode, Executor: "in-process", Reason: "mode-off"}, nil
	}
	dockerOK := r.DockerOK
	if dockerOK == nil {
		dockerOK = DockerAvailable
	}
	if r.PreferDocker && dockerOK() && (mode == ModeIsolated || mode == ModeWorkspaceWrite) {
		return Decision{Mode: mode, Executor: "docker", Reason: "docker-available"}, nil
	}
	return Decision{Mode: mode, Executor: "process", Reason: "process-jail"}, nil
}

// DispatchRequest is a sandboxed tool invocation.
type DispatchRequest struct {
	RunID       string
	StepID      string
	Tool        string
	Program     string
	Args        []string
	SandboxMode string
	RepoRoot    string
	Timeout     time.Duration
	Env         []string
}

// DispatchResult is stdout/stderr from a sandboxed command.
type DispatchResult struct {
	OK       bool
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
}

// Executor runs a command under a sandbox mode.
type Executor interface {
	Dispatch(ctx context.Context, req DispatchRequest) (*DispatchResult, error)
}

// DockerAvailable reports whether the docker CLI is usable (not skipped).
func DockerAvailable() bool {
	if strings.TrimSpace(os.Getenv("ASH_SKIP_SANDBOX")) != "" {
		return false
	}
	_, err := exec.LookPath("docker")
	return err == nil
}
