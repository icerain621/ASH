package observability

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

type Span struct {
	ID         string         `json:"id"`
	ParentID   string         `json:"parentId,omitempty"`
	RunID      string         `json:"runId"`
	Type       string         `json:"type"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	StartTs    int64          `json:"startTs,omitempty"`
	EndTs      int64          `json:"endTs,omitempty"`
	DurationMs int64          `json:"durationMs,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type FailureAttribution struct {
	Type    string `json:"type"`
	Ref     string `json:"ref"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type Metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type Waterfall struct {
	RunID       string               `json:"runId"`
	TraceID     string               `json:"traceId"`
	Status      string               `json:"status"`
	GeneratedAt int64                `json:"generatedAt"`
	Spans       []Span               `json:"spans"`
	Failures    []FailureAttribution `json:"failures,omitempty"`
	Metrics     []Metric             `json:"metrics,omitempty"`
}

func BuildWaterfall(db *store.DB, runID string) (*Waterfall, error) {
	var run store.RunRecord
	if err := db.First(&run, "id = ?", runID).Error; err != nil {
		return nil, err
	}

	out := &Waterfall{
		RunID: run.ID, TraceID: run.TraceID, Status: run.Status,
		GeneratedAt: time.Now().UTC().UnixMilli(),
	}
	out.Spans = append(out.Spans, runSpan(run))
	if run.ErrorCode != "" || run.ErrorMessage != "" {
		out.Failures = append(out.Failures, FailureAttribution{
			Type: "run", Ref: run.ID, Code: run.ErrorCode, Message: run.ErrorMessage,
		})
	}

	var steps []store.RunStep
	if err := db.Where("run_id = ?", runID).Order("step_order asc, created_at asc").Find(&steps).Error; err != nil {
		return nil, err
	}
	for _, step := range steps {
		out.Spans = append(out.Spans, stepSpan(run.ID, step))
		if step.Status == "failed" || step.Status == "waiting_approval" {
			out.Failures = append(out.Failures, FailureAttribution{
				Type: "step", Ref: step.StepID, Code: step.ErrorCode, Message: step.ErrorMessage,
			})
		}
	}

	var tools []store.ToolCall
	if err := db.Where("run_id = ?", runID).Order("created_at asc").Find(&tools).Error; err != nil {
		return nil, err
	}
	for _, tool := range tools {
		out.Spans = append(out.Spans, toolSpan(run.ID, tool))
		if tool.Status == "failed" {
			out.Failures = append(out.Failures, FailureAttribution{
				Type: "tool", Ref: tool.Tool, Message: tool.Error,
			})
		}
	}

	var agents []store.AgentTask
	if err := db.Where("run_id = ?", runID).Order("created_at asc").Find(&agents).Error; err != nil {
		return nil, err
	}
	for _, task := range agents {
		out.Spans = append(out.Spans, agentSpan(run.ID, task))
		if task.Status == "failed" {
			out.Failures = append(out.Failures, FailureAttribution{
				Type: "agent", Ref: firstNonEmpty(task.ExecGoTaskID, task.ActionID, task.ID),
				Code: task.ErrorCode, Message: task.ErrorMessage,
			})
		}
	}

	var modelUsages []store.ModelUsage
	if err := db.Where("run_id = ?", runID).Order("created_at asc").Find(&modelUsages).Error; err != nil {
		return nil, err
	}
	for _, usage := range modelUsages {
		out.Spans = append(out.Spans, modelSpan(run.ID, usage))
		if usage.Status != "routed" && usage.Status != "success" {
			out.Failures = append(out.Failures, FailureAttribution{
				Type: "model", Ref: usage.Provider + "/" + usage.Model, Code: usage.Status,
			})
		}
	}
	eventFailures, err := buildEventFailures(db, runID)
	if err != nil {
		return nil, err
	}
	out.Failures = append(out.Failures, eventFailures...)

	var metrics []store.QualityMetric
	if err := db.Where("run_id = ?", runID).Order("created_at asc").Find(&metrics).Error; err != nil {
		return nil, err
	}
	for _, metric := range metrics {
		out.Metrics = append(out.Metrics, Metric{Name: metric.Name, Value: metric.Value, Unit: metric.Unit})
	}
	return out, nil
}

func buildEventFailures(db *store.DB, runID string) ([]FailureAttribution, error) {
	var evs []store.RunEvent
	types := []string{"artifact.quality_failed", "artifact.store_failed", "citation.missing", "policy.denied"}
	if err := db.Where("run_id = ? AND type IN ?", runID, types).Order("seq asc").Find(&evs).Error; err != nil {
		return nil, err
	}
	out := make([]FailureAttribution, 0, len(evs))
	for _, ev := range evs {
		out = append(out, eventFailure(ev))
	}
	return out, nil
}

func eventFailure(ev store.RunEvent) FailureAttribution {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(ev.PayloadJSON), &payload)
	ref := firstNonEmpty(stringField(payload, "stepId"), stringField(payload, "ref"), stringField(payload, "target"))
	if ref == "" {
		ref = fmt.Sprintf("event:%d", ev.Seq)
	}
	msg := firstNonEmpty(stringField(payload, "error"), stringField(payload, "reason"), stringField(payload, "message"))
	switch ev.Type {
	case "artifact.quality_failed", "artifact.store_failed":
		return FailureAttribution{Type: "artifact", Ref: ref, Code: ev.Type, Message: msg}
	case "citation.missing":
		return FailureAttribution{Type: "citation", Ref: ref, Code: ev.Type, Message: msg}
	case "policy.denied":
		return FailureAttribution{Type: "policy", Ref: ref, Code: ev.Type, Message: msg}
	default:
		return FailureAttribution{Type: "event", Ref: ref, Code: ev.Type, Message: msg}
	}
}

func runSpan(run store.RunRecord) Span {
	end := run.UpdatedAt
	if run.FinishedAt != nil {
		end = *run.FinishedAt
	}
	return Span{
		ID: "run:" + run.ID, RunID: run.ID, Type: "run",
		Name: run.ScenarioName + "@" + run.ScenarioVersion, Status: run.Status,
		StartTs: ms(run.StartedAt), EndTs: ms(end), DurationMs: end.Sub(run.StartedAt).Milliseconds(),
		Attributes: map[string]any{
			"policyProfile": run.PolicyProfile,
			"spaceId":       run.SpaceID,
			"recovered":     run.Recovered,
			"repoRoot":      run.RepoRoot,
		},
	}
}

func stepSpan(runID string, step store.RunStep) Span {
	start, end := ptrMs(step.StartedAt), ptrMs(step.FinishedAt)
	return Span{
		ID: "step:" + step.StepID, ParentID: "run:" + runID, RunID: runID,
		Type: "step", Name: step.StepID, Status: step.Status,
		StartTs: start, EndTs: end, DurationMs: step.DurationMs,
		Attributes: map[string]any{
			"role": step.Role, "kind": step.Kind, "order": step.StepOrder,
			"errorCode": step.ErrorCode,
		},
	}
}

func toolSpan(runID string, tool store.ToolCall) Span {
	end := tool.CreatedAt
	if tool.CompletedAt != nil {
		end = *tool.CompletedAt
	}
	return Span{
		ID: "tool:" + tool.ID, ParentID: "step:" + tool.StepID, RunID: runID,
		Type: "tool", Name: tool.Tool, Status: tool.Status,
		StartTs: ms(tool.CreatedAt), EndTs: ms(end), DurationMs: tool.DurationMs,
		Attributes: map[string]any{
			"risk": tool.Risk, "attempt": tool.Attempt, "timeoutMs": tool.TimeoutMs,
			"argsDigest": tool.ArgsDigest,
		},
	}
}

func agentSpan(runID string, task store.AgentTask) Span {
	start := ptrMs(task.StartedAt)
	if start == 0 {
		start = ms(task.CreatedAt)
	}
	end := ptrMs(task.CompletedAt)
	return Span{
		ID: "agent:" + task.ID, ParentID: "step:" + task.StepID, RunID: runID,
		Type: "agent", Name: task.Adapter, Status: task.Status,
		StartTs: start, EndTs: end, DurationMs: task.DurationMs,
		Attributes: map[string]any{
			"agentId": task.AgentID, "sessionId": task.SessionID,
			"execGoTaskId": task.ExecGoTaskID, "actionId": task.ActionID,
			"exitCode": task.ExitCode,
		},
	}
}

func modelSpan(runID string, usage store.ModelUsage) Span {
	return Span{
		ID: "model:" + usage.ID, ParentID: "step:" + usage.StepID, RunID: runID,
		Type: "model", Name: usage.Provider + "/" + usage.Model, Status: usage.Status,
		StartTs: ms(usage.CreatedAt), EndTs: ms(usage.CreatedAt),
		Attributes: map[string]any{
			"provider": usage.Provider, "model": usage.Model,
			"inputTokens": usage.InputTokens, "outputTokens": usage.OutputTokens,
			"costMicros": usage.CostMicros,
		},
	}
}

func ms(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UTC().UnixMilli()
}

func ptrMs(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return ms(*t)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stringField(m map[string]any, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}
