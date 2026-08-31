package agentexec

import (
	"encoding/json"
	"fmt"
	"strings"
)

const ACPTaskSchemaV1 = "ash.acp.task.v1"

// ACPTaskV1 is the outbound ACP task contract (Sprint DX4).
type ACPTaskV1 struct {
	Schema    string `json:"schema"`
	AgentID   string `json:"agentId"`
	RunID     string `json:"runId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	TraceID   string `json:"traceId,omitempty"`
	StepID    string `json:"stepId,omitempty"`
	Role      string `json:"role,omitempty"`
	RepoRoot  string `json:"repoRoot,omitempty"`
	Prompt    string `json:"prompt"`
	Issue     string `json:"issue,omitempty"`
	TimeoutMs int64  `json:"timeoutMs,omitempty"`
}

// ACPTaskResultV1 is the inbound ACP task result contract.
type ACPTaskResultV1 struct {
	Schema  string         `json:"schema,omitempty"`
	OK      bool           `json:"ok"`
	TaskID  string         `json:"taskId"`
	Status  string         `json:"status,omitempty"`
	Message string         `json:"message,omitempty"`
	Output  map[string]any `json:"output,omitempty"`
}

// NewACPTaskV1 builds a validated outbound task from an executor request.
func NewACPTaskV1(agentID string, req Request) (ACPTaskV1, error) {
	timeout := req.TimeoutMs
	if timeout <= 0 {
		timeout = 120000
	}
	task := ACPTaskV1{
		Schema:    ACPTaskSchemaV1,
		AgentID:   firstNonEmpty(agentID, "ash-acp"),
		RunID:     strings.TrimSpace(req.RunID),
		SessionID: firstNonEmpty(metaString(req.Metadata, "sessionId"), req.RunID),
		TraceID:   strings.TrimSpace(req.TraceID),
		StepID:    strings.TrimSpace(req.StepID),
		Role:      strings.TrimSpace(req.Role),
		RepoRoot:  strings.TrimSpace(req.RepoRoot),
		Prompt:    req.Prompt,
		Issue:     req.Issue,
		TimeoutMs: timeout,
	}
	if err := task.Validate(); err != nil {
		return ACPTaskV1{}, err
	}
	return task, nil
}

// Validate checks required outbound fields.
func (t ACPTaskV1) Validate() error {
	if t.Schema != ACPTaskSchemaV1 {
		return fmt.Errorf("%w: schema must be %s", ErrAgentOutputInvalid, ACPTaskSchemaV1)
	}
	if strings.TrimSpace(t.AgentID) == "" {
		return fmt.Errorf("%w: agentId is required", ErrAgentOutputInvalid)
	}
	if strings.TrimSpace(t.Prompt) == "" && strings.TrimSpace(t.Issue) == "" {
		return fmt.Errorf("%w: prompt or issue is required", ErrAgentOutputInvalid)
	}
	if t.TimeoutMs < 0 {
		return fmt.Errorf("%w: timeoutMs must be >= 0", ErrAgentOutputInvalid)
	}
	return nil
}

// ParseACPTaskResultV1 decodes and validates an ACP task response body.
func ParseACPTaskResultV1(raw []byte) (ACPTaskResultV1, error) {
	var out ACPTaskResultV1
	if err := json.Unmarshal(raw, &out); err != nil {
		return ACPTaskResultV1{}, fmt.Errorf("%w: acp task decode: %v", ErrAgentOutputInvalid, err)
	}
	if err := out.Validate(); err != nil {
		return ACPTaskResultV1{}, err
	}
	return out, nil
}

// Validate checks inbound result shape. Empty schema is allowed (legacy); if set must match v1.
func (r ACPTaskResultV1) Validate() error {
	if s := strings.TrimSpace(r.Schema); s != "" && s != ACPTaskSchemaV1 {
		return fmt.Errorf("%w: unexpected result schema %q", ErrAgentOutputInvalid, s)
	}
	if strings.TrimSpace(r.TaskID) == "" && !r.OK {
		// failed responses may omit taskId; still require a message or status
		if strings.TrimSpace(r.Message) == "" && strings.TrimSpace(r.Status) == "" {
			return fmt.Errorf("%w: failed result needs message or status", ErrAgentOutputInvalid)
		}
	}
	status := strings.ToLower(strings.TrimSpace(r.Status))
	switch status {
	case "", "success", "failed", "running", "canceled", "cancelled":
	default:
		return fmt.Errorf("%w: invalid status %q", ErrAgentOutputInvalid, r.Status)
	}
	return nil
}

// EffectiveStatus normalizes status from ok/status fields.
func (r ACPTaskResultV1) EffectiveStatus() string {
	status := strings.TrimSpace(r.Status)
	if status != "" {
		return status
	}
	if r.OK {
		return "success"
	}
	return "failed"
}
