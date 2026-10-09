package runs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/agentloop"
	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/contextpack"
	"github.com/ash-repwiki/ash/internal/harness/drive"
	"github.com/ash-repwiki/ash/internal/observability/lenses"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/rag"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

// finalOncePolicy ends the loop on the first turn with a fixed final text.
type finalOncePolicy struct {
	text string
}

func (p finalOncePolicy) Next(int, []agentloop.Observation) (agentloop.Step, error) {
	text := strings.TrimSpace(p.text)
	if text == "" {
		text = "ok"
	}
	return agentloop.Step{Final: text}, nil
}

type busTool struct {
	name string
	risk string
	fn   func(args map[string]any) (string, error)
}

func (t busTool) Name() string { return t.name }
func (t busTool) Risk() string { return t.risk }
func (t busTool) Execute(_ context.Context, args map[string]any) (string, error) {
	return t.fn(args)
}

func (s *Service) executeTemplateStep(
	execCtx context.Context,
	runID, traceID, runDir, repoRoot, issue, spaceID string,
	step rules.Step,
	evidenceRefs []string,
	toolCtx toolbus.Context,
) error {
	tplID := ""
	prompt := ""
	if step.Agent != nil {
		tplID = strings.TrimSpace(step.Agent.TemplateID)
		prompt = step.Agent.Prompt
	}
	if step.Kind == "review" && tplID == "" {
		tplID = "tpl.reviewer"
	}
	if tplID == "" {
		return fmt.Errorf("step %s missing agent.templateId", step.ID)
	}
	manifest, err := agenttpl.GetProduction(tplID)
	if err != nil {
		return err
	}
	decl := append([]string(nil), manifest.Memory...)
	if err := plugins.Global().BindProduction(decl); err != nil {
		return err
	}
	if !lenses.Admit("context.packed", lenses.LensAgent) {
		return fmt.Errorf("context.packed is not registered on the agent lens")
	}
	built, err := contextpack.Build(contextpack.Input{
		Issue:    issue,
		RAGRefs:  evidenceWithoutMemory(evidenceRefs),
		Memories: memoryHitsFromRefs(evidenceRefs),
		Memory:   decl,
	})
	if err != nil {
		return err
	}
	if err := packWithinEvidence(built.Refs, evidenceRefs); err != nil {
		return err
	}
	if prompt == "" {
		prompt = issue
	}
	now := time.Now().UTC()
	taskID := "tpl_" + uuid.NewString()
	task := store.AgentTask{
		ID: "agt_" + uuid.NewString(), RunID: runID, TraceID: traceID, StepID: step.ID,
		Adapter: "template_loop", AgentID: "ash-" + tplID, SessionID: runID,
		Status: "running", PromptDigest: digestString(issue + "\n" + prompt),
		ActionID: taskID, CreatedAt: now, StartedAt: &now,
	}
	_ = s.gdb().Create(&task).Error
	_, _ = s.eventsFor().Append(runID, traceID, "context.packed", "info", map[string]any{
		"stepId": step.ID, "refs": built.Refs, "memoryRefs": built.MemoryRefs,
		"prefixBytes": len(built.Prefix), "templateId": tplID,
	})
	_, _ = s.eventsFor().Append(runID, traceID, "agent.called", "info", map[string]any{
		"stepId": step.ID, "templateId": tplID, "adapter": "template_loop",
	})

	tools := map[string]agentloop.Tool{}
	for _, name := range manifest.Tools {
		n := name
		risk := string(s.tools.ToolRisk(n))
		tools[n] = busTool{
			name: n,
			risk: risk,
			fn: func(args map[string]any) (string, error) {
				res := s.tools.Call(toolCtx, toolbus.CallRequest{Tool: n, Args: args})
				if !res.OK {
					return "", fmt.Errorf("%s", res.Error)
				}
				raw, _ := json.Marshal(res.Output)
				return string(raw), nil
			},
		}
	}
	loopPrompt := prompt
	if built.Prefix != "" {
		loopPrompt = built.Prefix + "\n\n" + prompt
	}
	result, err := agentloop.Run(execCtx, agentloop.Config{
		MaxTurns: manifest.MaxTurns,
		Policy:   finalOncePolicy{text: loopPrompt},
		Tools:    tools,
		Drive:    &drive.Drive{},
	})
	finished := time.Now().UTC()
	task.CompletedAt = &finished
	task.DurationMs = finished.Sub(now).Milliseconds()
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		_ = s.gdb().Save(&task).Error
		_, _ = s.eventsFor().Append(runID, traceID, "agent.failed", "error", map[string]any{
			"stepId": step.ID, "templateId": tplID, "taskId": task.ActionID, "error": err.Error(),
		})
		return err
	}
	task.Status = "success"
	_ = s.gdb().Save(&task).Error
	_, _ = s.eventsFor().Append(runID, traceID, "template.finished", "info", map[string]any{
		"stepId": step.ID, "templateId": tplID, "stop": string(result.Stop),
		"turns": result.Turns, "spaceId": spaceID, "runDir": runDir,
	})
	_, _ = s.eventsFor().Append(runID, traceID, "agent.finished", "info", map[string]any{
		"stepId": step.ID, "taskId": task.ActionID, "status": task.Status, "durationMs": task.DurationMs,
		"templateId": tplID,
	})
	_ = s.writeTemplateStepArtifact(runDir, step, issue, evidenceRefs)
	return nil
}

func (s *Service) bindRAGQuerier(spaceID, repoRoot string) toolbus.RAGQuerier {
	if s.rag == nil {
		return nil
	}
	return func(text string, topK int) (map[string]any, error) {
		resp, err := s.rag.Query(rag.QueryRequest{
			SpaceID: spaceID, RepoRoot: repoRoot, Text: text, TopK: topK,
		})
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(resp.Items))
		for _, h := range resp.Items {
			items = append(items, map[string]any{
				"ref": h.Ref, "path": h.Path, "score": h.Score, "snippet": h.Snippet,
			})
		}
		return map[string]any{
			"items": items, "retrievalMode": resp.RetrievalMode, "count": len(items),
		}, nil
	}
}
