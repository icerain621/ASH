package sandbox

import (
	"context"
	"fmt"
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
	// PreferRemote overrides env prefer for this call: "remote"/"1"/"prefer" forces
	// prefer when remote is enabled; "local"/"0"/"never" skips remote; empty uses env.
	PreferRemote string
}

// Decision records how a tool call should execute.
type Decision struct {
	Mode     string `json:"mode"`
	Executor string `json:"executor"` // in-process | process | docker | landlock | remote | none
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
	// LandlockOK overrides landlock availability (tests); nil uses RegisterLandlockAvailable hook.
	LandlockOK func() bool
	// RemoteOK overrides remote availability (tests); nil uses RegisterRemoteAvailable hook.
	RemoteOK func() bool
}

// defaultLandlockAvailable is set by internal/sandbox/landlock init (avoids import cycle).
var defaultLandlockAvailable = func() bool { return false }

// defaultRemoteAvailable is set by internal/sandbox/remote init (avoids import cycle).
var defaultRemoteAvailable = func() bool { return false }

// RegisterLandlockAvailable wires landlock.Available into the default router probe.
func RegisterLandlockAvailable(fn func() bool) {
	if fn != nil {
		defaultLandlockAvailable = fn
	}
}

// RegisterRemoteAvailable wires remote.Available into the default router probe.
func RegisterRemoteAvailable(fn func() bool) {
	if fn != nil {
		defaultRemoteAvailable = fn
	}
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
	// DX39: isolated (+ danger/network floors) may prefer remote when opt-in + Available.
	if mode == ModeIsolated && remotePreferred(req.PreferRemote) {
		if r.remoteOK() {
			return Decision{Mode: mode, Executor: "remote", Reason: "remote-preferred"}, nil
		}
		if remoteDenyOnFail() {
			reason := "remote preferred but unavailable (ASH_SANDBOX_REMOTE_ON_FAIL=deny)"
			return Decision{Mode: mode, Executor: "none", Reason: reason, Denied: true}, fmt.Errorf("%w: %s", ErrRemoteUnavailable, reason)
		}
		// Preferred but unavailable: fall through to landlock/docker/process.
	}
	if mode == ModeIsolated && landlockPreferred() {
		if r.landlockOK() {
			return Decision{Mode: mode, Executor: "landlock", Reason: "landlock-available"}, nil
		}
		// Preferred but unavailable: fall through to docker/process (no deny).
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

func (r DefaultRouter) landlockOK() bool {
	if r.LandlockOK != nil {
		return r.LandlockOK()
	}
	return defaultLandlockAvailable()
}

func (r DefaultRouter) remoteOK() bool {
	if r.RemoteOK != nil {
		return r.RemoteOK()
	}
	return defaultRemoteAvailable()
}

// landlockPreferred reports whether isolated mode should try Landlock first.
// Default ON (DX21). Opt out with ASH_SANDBOX_LANDLOCK=0|false|off|no.
func landlockPreferred() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ASH_SANDBOX_LANDLOCK")))
	switch v {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// LandlockPreferred is exported for Doctor / diagnostics.
func LandlockPreferred() bool { return landlockPreferred() }

// remotePreferred reports whether isolated mode should try remote first (DX39).
// Requires ASH_SANDBOX_REMOTE enabled; per-call PreferRemote can force prefer or skip.
func remotePreferred(preferOverride string) bool {
	switch strings.ToLower(strings.TrimSpace(preferOverride)) {
	case "0", "false", "off", "no", "local", "never":
		return false
	case "1", "true", "on", "yes", "remote", "prefer":
		return remoteEnabled()
	}
	return remoteEnabled()
}

func remoteEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ASH_SANDBOX_REMOTE"))) {
	case "1", "true", "on", "yes", "auto":
		return true
	default:
		return false
	}
}

// RemotePreferred is exported for Doctor / diagnostics (env only; no per-call override).
func RemotePreferred() bool { return remotePreferred("") }

// remoteDenyOnFail reports whether unavailable remote should deny instead of fallback.
// Default: fallback. Opt-in deny via ASH_SANDBOX_REMOTE_ON_FAIL=deny|fail|refuse.
func remoteDenyOnFail() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ASH_SANDBOX_REMOTE_ON_FAIL"))) {
	case "deny", "fail", "refuse", "error":
		return true
	default:
		return false
	}
}

// RemoteDenyOnFail is exported for Doctor / diagnostics.
func RemoteDenyOnFail() bool { return remoteDenyOnFail() }

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
//
// Extension point for remote / microVM backends (v2.9): implement Executor (or
// remote.Backend). DefaultRouter prefers remote for isolated when
// ASH_SANDBOX_REMOTE is enabled and Available (DX39); ON_FAIL=deny refuses
// instead of falling back to Landlock/docker/process.
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
