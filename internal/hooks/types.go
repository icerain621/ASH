package hooks

const SchemaVersion = "ash.hooks.v1"

type Event string

const (
	EventPreToolUse Event = "PreToolUse"
)

type DecisionAction string

const (
	ActionAllow DecisionAction = "allow"
	ActionDeny  DecisionAction = "deny"
	ActionAsk   DecisionAction = "ask"
)

type Rule struct {
	Event  Event          `json:"event"`
	Tool   string         `json:"tool,omitempty"` // name or glob; empty = any
	Risk   string         `json:"risk,omitempty"` // low|medium|high|critical
	Action DecisionAction `json:"action"`
	Reason string         `json:"reason,omitempty"`
}

type Config struct {
	Version string `json:"version"` // ash.hooks.v1
	Rules   []Rule `json:"rules"`
}

type ToolContext struct {
	Tool    string
	Risk    string
	SpaceID string
	RunID   string
	StepID  string
}

type Decision struct {
	Action    DecisionAction
	Reason    string
	RuleIndex int // -1 if default allow (no match)
}
