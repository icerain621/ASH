package agentexec

import (
	"context"
	"errors"
)

var ErrBridgeUnavailable = errors.New("agent bridge unavailable")
var ErrAgentTaskFailed = errors.New("agent task failed")
var ErrAgentOutputInvalid = errors.New("agent output invalid")

// Request describes one external-agent step execution.
type Request struct {
	RunID     string
	TraceID   string
	StepID    string
	Role      string
	RepoRoot  string
	RunDir    string
	Issue     string
	Prompt    string
	Inputs    map[string]any
	TimeoutMs int64
	Metadata  map[string]any
}

// Result is the normalized outcome of an external agent task.
type Result struct {
	TaskID        string
	ExecGoTaskID  string
	Adapter       string
	AgentID       string
	SessionID     string
	ActionID      string
	Status        string
	StdoutSummary string
	StderrSummary string
	ExitCode      *int
	DurationMs    int64
	Output        map[string]any
}

// Status describes a previously submitted external agent task.
type Status struct {
	TaskID       string
	ExecGoTaskID string
	Status       string
	Output       map[string]any
}

// Executor is the boundary between ASH orchestration and an external coding agent.
type Executor interface {
	Plan(ctx context.Context, req Request) (*Result, error)
	Execute(ctx context.Context, req Request) (*Result, error)
	Cancel(ctx context.Context, taskID string) error
	Status(ctx context.Context, taskID string) (*Status, error)
	CollectArtifacts(ctx context.Context, req Request, res Result) (map[string]string, error)
}
