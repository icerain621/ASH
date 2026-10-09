package agentloop

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/harness/drive"
	"github.com/ash-repwiki/ash/internal/hooks"
)

type scriptPolicy struct {
	steps []Step
	i     int
	err   error
}

func (p *scriptPolicy) Next(int, []Observation) (Step, error) {
	if p.err != nil {
		return Step{}, p.err
	}
	if p.i >= len(p.steps) {
		return Step{}, nil
	}
	step := p.steps[p.i]
	p.i++
	return step, nil
}

type fakeTool struct {
	name  string
	risk  string
	calls int
}

func (f *fakeTool) Name() string { return f.name }
func (f *fakeTool) Risk() string { return f.risk }
func (f *fakeTool) Execute(context.Context, map[string]any) (string, error) {
	f.calls++
	return "ok", nil
}

func TestRejectsUnboundedLoop(t *testing.T) {
	_, err := Run(context.Background(), Config{MaxTurns: 0, Policy: &scriptPolicy{}})
	if !errors.Is(err, ErrUnbounded) {
		t.Fatalf("err=%v", err)
	}
}

func TestStopReasons(t *testing.T) {
	tool := &fakeTool{name: "read", risk: "safe"}
	tools := map[string]Tool{"read": tool}

	final, err := Run(context.Background(), Config{
		MaxTurns: 4,
		Policy:   &scriptPolicy{steps: []Step{{Final: "done"}}},
		Tools:    tools,
	})
	if err != nil || final.Stop != StopFinal || final.Final != "done" {
		t.Fatalf("final=%+v err=%v", final, err)
	}

	none, err := Run(context.Background(), Config{
		MaxTurns: 4,
		Policy:   &scriptPolicy{},
		Tools:    tools,
	})
	if err != nil || none.Stop != StopNoTool {
		t.Fatalf("none=%+v err=%v", none, err)
	}

	broken, err := Run(context.Background(), Config{
		MaxTurns: 4,
		Policy:   &scriptPolicy{err: errors.New("policy failed")},
		Tools:    tools,
	})
	if err != nil || broken.Stop != StopUnrecoverable {
		t.Fatalf("broken=%+v err=%v", broken, err)
	}

	capped, err := Run(context.Background(), Config{
		MaxTurns: 2,
		Policy:   &scriptPolicy{steps: []Step{{Tool: "read"}, {Tool: "read"}, {Tool: "read"}}},
		Tools:    tools,
	})
	if err != nil || capped.Stop != StopMaxTurns || capped.Turns != 2 {
		t.Fatalf("capped=%+v err=%v", capped, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stopped, err := Run(ctx, Config{MaxTurns: 3, Policy: &scriptPolicy{steps: []Step{{Final: "late"}}}, Tools: tools})
	if err != nil || stopped.Stop != StopCanceled || stopped.Turns != 0 {
		t.Fatalf("canceled=%+v err=%v", stopped, err)
	}
}

func TestDangerToolDoesNotExecuteWithoutApproval(t *testing.T) {
	bash := &fakeTool{name: "bash", risk: "danger"}
	res, err := Run(context.Background(), Config{
		MaxTurns: 2,
		Policy:   &scriptPolicy{steps: []Step{{Tool: "bash"}}},
		Tools:    map[string]Tool{"bash": bash},
		Drive:    &drive.Drive{},
	})
	if err != nil || res.Stop != StopUnrecoverable || bash.calls != 0 {
		t.Fatalf("res=%+v calls=%d err=%v", res, bash.calls, err)
	}
}

func TestPreToolHookDeniesCall(t *testing.T) {
	read := &fakeTool{name: "read", risk: "safe"}
	res, err := Run(context.Background(), Config{
		MaxTurns: 2,
		Policy:   &scriptPolicy{steps: []Step{{Tool: "read"}}},
		Tools:    map[string]Tool{"read": read},
		Hooks: hooks.Config{Rules: []hooks.Rule{{
			Event: hooks.EventPreToolUse, Tool: "read", Action: hooks.ActionDeny, Reason: "blocked",
		}}},
	})
	if err != nil || res.Stop != StopUnrecoverable || read.calls != 0 || res.Reason != "blocked" {
		t.Fatalf("res=%+v calls=%d err=%v", res, read.calls, err)
	}
}

type sizedTool struct {
	outputs []string
	i       int
}

func (s *sizedTool) Name() string { return "read" }
func (s *sizedTool) Risk() string { return "safe" }
func (s *sizedTool) Execute(context.Context, map[string]any) (string, error) {
	if s.i >= len(s.outputs) {
		return "", nil
	}
	out := s.outputs[s.i]
	s.i++
	return out, nil
}

func TestTokenBudgetCompactsThenRejects(t *testing.T) {
	tool := &sizedTool{outputs: []string{strings.Repeat("x", 20)}}
	_, err := Run(context.Background(), Config{
		MaxTurns:    2,
		TokenBudget: 2,
		Policy:      &scriptPolicy{steps: []Step{{Tool: "read"}, {Final: "late"}}},
		Tools:       map[string]Tool{"read": tool},
	})
	if !errors.Is(err, ErrTokenBudget) {
		t.Fatalf("err=%v", err)
	}
}
