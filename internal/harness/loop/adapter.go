package loop

import (
	"github.com/ash-repwiki/ash/internal/sandbox"
)

// Emitter appends run events (typically events.Service.Append).
type Emitter interface {
	Emit(runID, traceID, eventType, severity string, payload map[string]any) error
}

// EmittedEvent is used by test emitters.
type EmittedEvent struct {
	RunID, TraceID, Type, Severity string
	Payload                        map[string]any
}

// ProfileLoader resolves active harness profile defaults for a space.
type ProfileLoader interface {
	ProfileDefaultSandboxMode(spaceID, profileName string) (string, error)
}

// ToolHookContext is passed around tool routing hooks.
type ToolHookContext struct {
	RunID              string
	TraceID            string
	StepID             string
	SpaceID            string
	Tool               string
	Risk               string
	RepoRoot           string
	ProfileName        string
	ProfileDefaultMode string
	ModeOverride       string
	ScenarioMinMode    string
	PolicyProfile      string
}

// Adapter is the Harness Loop Adapter (thin hooks; does not own the run loop).
type Adapter struct {
	emit   Emitter
	router sandbox.Router
	loader ProfileLoader
}

func NewAdapter(emit Emitter, router sandbox.Router, loader ProfileLoader) *Adapter {
	if router == nil {
		router = sandbox.NoopRouter{}
	}
	return &Adapter{emit: emit, router: router, loader: loader}
}

func (a *Adapter) OnTurnStart(runID, traceID string) error {
	if a == nil || a.emit == nil {
		return nil
	}
	return a.emit.Emit(runID, traceID, "harness.turn.started", "info", map[string]any{})
}

func (a *Adapter) OnTurnEnd(runID, traceID string) error {
	if a == nil || a.emit == nil {
		return nil
	}
	return a.emit.Emit(runID, traceID, "harness.turn.ended", "info", map[string]any{})
}

func (a *Adapter) OnStepStart(runID, traceID, stepID string) error {
	if a == nil || a.emit == nil {
		return nil
	}
	return a.emit.Emit(runID, traceID, "harness.step.started", "info", map[string]any{"stepId": stepID})
}

func (a *Adapter) OnStepEnd(runID, traceID, stepID string) error {
	if a == nil || a.emit == nil {
		return nil
	}
	return a.emit.Emit(runID, traceID, "harness.step.ended", "info", map[string]any{"stepId": stepID})
}

func (a *Adapter) OnBeforeTool(ctx ToolHookContext) (sandbox.Decision, error) {
	if a == nil {
		return sandbox.NoopRouter{}.Route(sandbox.RouteRequest{
			Tool: ctx.Tool, Risk: ctx.Risk, ProfileDefaultMode: ctx.ProfileDefaultMode, ModeOverride: ctx.ModeOverride,
			ScenarioMinMode: ctx.ScenarioMinMode, PolicyProfile: ctx.PolicyProfile,
		})
	}
	mode := ctx.ProfileDefaultMode
	if mode == "" && a.loader != nil {
		name := ctx.ProfileName
		if name == "" {
			name = "default"
		}
		if m, err := a.loader.ProfileDefaultSandboxMode(ctx.SpaceID, name); err == nil {
			mode = m
		}
	}
	dec, err := a.router.Route(sandbox.RouteRequest{
		Tool:               ctx.Tool,
		Risk:               ctx.Risk,
		ProfileDefaultMode: mode,
		ModeOverride:       ctx.ModeOverride,
		ScenarioMinMode:    ctx.ScenarioMinMode,
		PolicyProfile:      ctx.PolicyProfile,
		RepoRoot:           ctx.RepoRoot,
		RunID:              ctx.RunID,
		StepID:             ctx.StepID,
	})
	if a.emit != nil {
		payload := map[string]any{
			"stepId": ctx.StepID, "tool": ctx.Tool, "risk": ctx.Risk,
			"sandboxMode": dec.Mode, "executor": dec.Executor, "reason": dec.Reason,
		}
		if dec.Denied {
			payload["denied"] = true
		}
		_ = a.emit.Emit(ctx.RunID, ctx.TraceID, "harness.tool.routed", "info", payload)
	}
	if err != nil {
		return dec, err
	}
	return dec, nil
}

func (a *Adapter) OnAfterTool(ctx ToolHookContext, dec sandbox.Decision, ok bool, errMsg string) error {
	if a == nil || a.emit == nil {
		return nil
	}
	sev := "info"
	if !ok {
		sev = "warn"
	}
	payload := map[string]any{
		"stepId": ctx.StepID, "tool": ctx.Tool, "ok": ok,
		"sandboxMode": dec.Mode, "executor": dec.Executor,
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	return a.emit.Emit(ctx.RunID, ctx.TraceID, "harness.tool.completed", sev, payload)
}

// AssertToolResultsCovered implements Doctor M4-HAR-02 (package-level):
// if tool.result appears, tool.called or harness.tool.completed must also appear.
func AssertToolResultsCovered(eventTypes []string, _ []string) bool {
	hasResult := false
	hasCover := false
	for _, et := range eventTypes {
		switch et {
		case "tool.result":
			hasResult = true
		case "tool.called", "harness.tool.completed":
			hasCover = true
		}
	}
	if !hasResult {
		return true
	}
	return hasCover
}
