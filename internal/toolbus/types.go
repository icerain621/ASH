package toolbus

import (
	"sort"
	"time"
)

// Risk classifies tool execution policy (hooks match on this).
type Risk string

const (
	RiskSafe   Risk = "safe"
	RiskMedium Risk = "medium"
	RiskDanger Risk = "danger"
)

// ToolRiskEntry is one row in the published dangerous-ops / risk catalog.
type ToolRiskEntry struct {
	Name           string `json:"name"`
	Risk           Risk   `json:"risk"`
	DefaultDeny    bool   `json:"defaultDeny"`
	Label          string `json:"label"`
	MinSandboxMode string `json:"minSandboxMode,omitempty"`
}

// Context carries run-scoped execution state for tools.
type Context struct {
	RunID    string
	TraceID  string
	RepoRoot string
	RunDir   string
	Inputs   map[string]any
}

// CallRequest is a single tool invocation.
type CallRequest struct {
	Tool string
	Args map[string]any
	Risk Risk
}

// Result is the outcome of a tool call.
type Result struct {
	Tool         string         `json:"tool"`
	OK           bool           `json:"ok"`
	Output       map[string]any `json:"output,omitempty"`
	Error        string         `json:"error,omitempty"`
	FailureClass string         `json:"failureClass,omitempty"`
	DurationMs   int64          `json:"durationMs"`
}

// ToolFunc executes a named tool.
type ToolFunc func(ctx Context, args map[string]any) (map[string]any, error)

// Registry maps tool names to implementations and default risk.
type Registry struct {
	tools map[string]ToolFunc
	risk  map[string]Risk
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolFunc),
		risk:  make(map[string]Risk),
	}
}

func (r *Registry) Register(name string, risk Risk, fn ToolFunc) {
	r.tools[name] = fn
	r.risk[name] = risk
}

func (r *Registry) Risk(name string) Risk {
	if v, ok := r.risk[name]; ok {
		return v
	}
	return RiskMedium
}

func (r *Registry) Has(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// Catalog returns registered tools sorted by name with risk metadata.
func (r *Registry) Catalog() []ToolRiskEntry {
	if r == nil {
		return nil
	}
	names := make([]string, 0, len(r.risk))
	for name := range r.risk {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ToolRiskEntry, 0, len(names))
	for _, name := range names {
		risk := r.Risk(name)
		out = append(out, ToolRiskEntry{
			Name:           name,
			Risk:           risk,
			DefaultDeny:    risk == RiskDanger,
			Label:          riskLabel(name, risk),
			MinSandboxMode: minSandboxModeForRisk(risk),
		})
	}
	return out
}

func minSandboxModeForRisk(risk Risk) string {
	switch risk {
	case RiskDanger:
		return "isolated"
	case RiskMedium:
		return "workspace-write"
	default:
		return "off"
	}
}

func riskLabel(name string, risk Risk) string {
	switch risk {
	case RiskDanger:
		return name + "（危险：默认需人工批准或 allow_dangerous）"
	case RiskSafe:
		return name + "（安全）"
	default:
		return name + "（中等）"
	}
}

// Bus dispatches tool calls through a registry.
type Bus struct {
	reg *Registry
}

func NewBus(reg *Registry) *Bus {
	return &Bus{reg: reg}
}

func (b *Bus) ToolRisk(name string) Risk {
	return b.reg.Risk(name)
}

// Catalog returns the bus risk catalog (dangerous-ops product surface).
func (b *Bus) Catalog() []ToolRiskEntry {
	if b == nil || b.reg == nil {
		return nil
	}
	return b.reg.Catalog()
}

func (b *Bus) Call(ctx Context, req CallRequest) Result {
	start := time.Now()
	risk := req.Risk
	if risk == "" {
		risk = b.reg.Risk(req.Tool)
	}
	_ = risk

	fn, ok := b.reg.tools[req.Tool]
	if !ok {
		return Result{
			Tool:       req.Tool,
			OK:         false,
			Error:      "unknown tool: " + req.Tool,
			DurationMs: time.Since(start).Milliseconds(),
		}
	}
	out, err := fn(ctx, req.Args)
	res := Result{
		Tool:       req.Tool,
		OK:         err == nil,
		Output:     out,
		DurationMs: time.Since(start).Milliseconds(),
	}
	if err != nil {
		res.Error = err.Error()
		if classified, ok := err.(interface{ FailureClass() string }); ok {
			res.FailureClass = classified.FailureClass()
		}
	}
	return res
}
