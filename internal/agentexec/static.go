package agentexec

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// StaticExecutor is a deterministic executor for tests and local smoke scenarios.
// Production wiring uses ExecGoCodexExecutor by default.
type StaticExecutor struct {
	StatusValue string
}

func (e StaticExecutor) AdapterName() string {
	return "static"
}

func (e StaticExecutor) Plan(ctx context.Context, req Request) (*Result, error) {
	return e.Execute(ctx, req)
}

func (e StaticExecutor) Execute(_ context.Context, req Request) (*Result, error) {
	status := e.StatusValue
	if status == "" {
		status = "success"
	}
	if req.RunDir != "" {
		artDir := filepath.Join(req.RunDir, "artifacts")
		_ = os.MkdirAll(artDir, 0o755)
		path := filepath.Join(artDir, "diff.patch")
		if st, err := os.Stat(path); err != nil || st.Size() == 0 {
			_ = os.WriteFile(path, []byte("# Static executor produced no code changes.\n"), 0o644)
		}
	}
	return &Result{
		TaskID: "static-" + req.StepID, ExecGoTaskID: "static-" + req.StepID,
		Adapter: "static", AgentID: "static-agent", SessionID: req.RunID,
		ActionID: "static-" + req.StepID, Status: status,
		StdoutSummary: "static executor completed", DurationMs: int64(time.Millisecond),
		Output: map[string]any{"ok": true},
	}, nil
}

func (e StaticExecutor) Cancel(_ context.Context, _ string) error {
	return nil
}

func (e StaticExecutor) Status(_ context.Context, taskID string) (*Status, error) {
	status := e.StatusValue
	if status == "" {
		status = "success"
	}
	return &Status{TaskID: taskID, ExecGoTaskID: taskID, Status: status}, nil
}

func (e StaticExecutor) CollectArtifacts(_ context.Context, _ Request, _ Result) (map[string]string, error) {
	return map[string]string{}, nil
}
