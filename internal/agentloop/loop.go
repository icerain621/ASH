package agentloop

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/harness/compact"
	"github.com/ash-repwiki/ash/internal/harness/drive"
	"github.com/ash-repwiki/ash/internal/hooks"
)

// StopReason explains why the loop ended.
type StopReason string

const (
	StopNoTool        StopReason = "no_tool"
	StopFinal         StopReason = "final"
	StopUnrecoverable StopReason = "unrecoverable"
	StopMaxTurns      StopReason = "max_turns"
	StopCanceled      StopReason = "canceled"
)

// ErrUnbounded rejects a loop that has no turn cap.
var ErrUnbounded = fmt.Errorf("maxTurns must be >= 1")

// ErrTokenBudget is returned after compaction still cannot fit the next output.
var ErrTokenBudget = errors.New("TOKEN_BUDGET_EXCEEDED")

// Step is one policy decision.
type Step struct {
	Tool  string
	Args  map[string]any
	Final string
}

// Observation is the result of one tool placement.
type Observation struct {
	Tool   string
	Output string
	Err    string
}

// Policy chooses the next step from prior observations.
type Policy interface {
	Next(turn int, observations []Observation) (Step, error)
}

// Tool is the contract a template depends on.
type Tool interface {
	Name() string
	Risk() string
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// Config runs one bounded loop.
type Config struct {
	MaxTurns    int
	Policy      Policy
	Tools       map[string]Tool
	Hooks       hooks.Config
	Drive       *drive.Drive
	TokenBudget int
}

// Result is the loop outcome.
type Result struct {
	Stop   StopReason
	Final  string
	Turns  int
	Reason string
}

// Run executes think → tool → observe until a stop reason.
func Run(ctx context.Context, cfg Config) (Result, error) {
	if cfg.MaxTurns < 1 {
		return Result{}, ErrUnbounded
	}
	if cfg.Policy == nil {
		return Result{}, fmt.Errorf("policy is required")
	}
	var observations []Observation
	ledger := tokenLedger{limit: cfg.TokenBudget}
	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		if ctx.Err() != nil {
			return Result{Stop: StopCanceled, Turns: turn - 1, Reason: ctx.Err().Error()}, nil
		}
		step, err := cfg.Policy.Next(turn, observations)
		if err != nil {
			return Result{Stop: StopUnrecoverable, Turns: turn - 1, Reason: err.Error()}, nil
		}
		if step.Tool == "" && step.Final != "" {
			return Result{Stop: StopFinal, Final: step.Final, Turns: turn}, nil
		}
		if step.Tool == "" {
			return Result{Stop: StopNoTool, Turns: turn}, nil
		}
		tool := cfg.Tools[step.Tool]
		if tool == nil {
			return Result{Stop: StopUnrecoverable, Turns: turn, Reason: "unknown tool " + step.Tool}, nil
		}
		pre := hooks.Evaluate(cfg.Hooks, hooks.EventPreToolUse, hooks.ToolContext{Tool: tool.Name(), Risk: tool.Risk()})
		if pre.Action == hooks.ActionDeny || pre.Action == hooks.ActionAsk {
			return Result{Stop: StopUnrecoverable, Turns: turn, Reason: pre.Reason}, nil
		}
		var output string
		err = place(cfg.Drive, tool.Name(), tool.Risk(), func() error {
			var callErr error
			output, callErr = tool.Execute(ctx, step.Args)
			return callErr
		})
		if err != nil {
			return Result{Stop: StopUnrecoverable, Turns: turn, Reason: err.Error()}, nil
		}
		if err := ledger.charge(output, &observations); err != nil {
			return Result{Stop: StopUnrecoverable, Turns: turn, Reason: err.Error()}, err
		}
		post := hooks.Evaluate(cfg.Hooks, hooks.EventPostToolUse, hooks.ToolContext{Tool: tool.Name(), Risk: tool.Risk()})
		observations = append(observations, Observation{Tool: tool.Name(), Output: output})
		if post.Action == hooks.ActionDeny {
			return Result{Stop: StopUnrecoverable, Turns: turn, Reason: post.Reason}, nil
		}
	}
	return Result{Stop: StopMaxTurns, Turns: cfg.MaxTurns}, nil
}

func place(d *drive.Drive, name, risk string, fn func() error) error {
	if d == nil {
		if risk == "danger" {
			return drive.ErrDangerUnapproved
		}
		return fn()
	}
	return d.Execute(name, risk, fn)
}

type tokenLedger struct {
	limit int
	used  int
}

func (l *tokenLedger) charge(text string, observations *[]Observation) error {
	if l.limit <= 0 {
		return nil
	}
	cost := compact.Estimate(text)
	if l.used+cost <= l.limit {
		l.used += cost
		return nil
	}
	summary, chunks := compact.Summarize(joinObservations(*observations), 32)
	*observations = nil
	if chunks > 0 {
		*observations = []Observation{{Tool: "compaction", Output: summary}}
	}
	l.used = compact.Estimate(summary)
	if l.used+cost <= l.limit {
		l.used += cost
		return nil
	}
	return ErrTokenBudget
}

func joinObservations(items []Observation) string {
	var b strings.Builder
	for _, item := range items {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(item.Output)
	}
	return b.String()
}
