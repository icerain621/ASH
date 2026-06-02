package toolbus

import "time"

// Risk classifies tool execution policy (hooks match on this).
type Risk string

const (
	RiskSafe   Risk = "safe"
	RiskMedium Risk = "medium"
	RiskDanger Risk = "danger"
)

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
	Tool       string         `json:"tool"`
	OK         bool           `json:"ok"`
	Output     map[string]any `json:"output,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"durationMs"`
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
	}
	return res
}
