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
	Executor string `json:"executor"` // in-process | process | docker | landlock | none
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
}

// defaultLandlockAvailable is set by internal/sandbox/landlock init (avoids import cycle).
var defaultLandlockAvailable = func() bool { return false }

// RegisterLandlockAvailable wires landlock.Available into the default router probe.
func RegisterLandlockAvailable(fn func() bool) {
	if fn != nil {
		defaultLandlockAvailable = fn
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
// Extension point for future remote / microVM backends: implement Executor and
// wire a new Decision.Executor name in DefaultRouter / runs dispatch. v2.7 does
// not ship an E2B (or other billed cloud) client — keep adapters out-of-tree
// or behind explicit opt-in until a later release.
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
