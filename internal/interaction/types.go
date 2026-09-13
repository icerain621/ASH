package interaction

import "encoding/json"

const (
	LinkHitUsed      = "hit_used"
	LinkContextRef   = "context_ref"
	LinkCitation     = "citation"
	LinkCandidateOut = "candidate_out"

	ThreadStatusOpen   = "open"
	ThreadStatusSealed = "sealed"
)

// TimelineNode is a pure projection of a run event onto a thread.
type TimelineNode struct {
	ID         string          `json:"id"`
	Seq        int64           `json:"seq"`
	TS         int64           `json:"ts"`
	Type       string          `json:"type"`
	Visibility string          `json:"visibility"`
	Severity   string          `json:"severity,omitempty"`
	StepID     string          `json:"stepId,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

// MemoryLink associates a memory/asset with a thread event.
type MemoryLink struct {
	ID        string `json:"id"`
	SpaceID   string `json:"spaceId"`
	SessionID string `json:"sessionId"`
	ThreadID  string `json:"threadId"`
	RunID     string `json:"runId"`
	EventSeq  int64  `json:"eventSeq"`
	EventID   string `json:"eventId,omitempty"`
	StepID    string `json:"stepId,omitempty"`
	MemoryID  string `json:"memoryId"`
	LinkType  string `json:"linkType"`
	Layer     string `json:"layer,omitempty"`
	TS        int64  `json:"ts"`
	Digest    string `json:"digest"`
}

// Thread is the persisted interaction thread (main per run for GV04).
type Thread struct {
	ID        string `json:"id"`
	SpaceID   string `json:"spaceId"`
	SessionID string `json:"sessionId,omitempty"`
	RunID     string `json:"runId"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	HeadSeq   int64  `json:"headSeq,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// FoldResult is the pure FoldThread output.
type FoldResult struct {
	SessionID string         `json:"sessionId,omitempty"`
	ThreadID  string         `json:"threadId"`
	RunID     string         `json:"runId"`
	SpaceID   string         `json:"spaceId"`
	Nodes     []TimelineNode `json:"nodes"`
	Links     []MemoryLink   `json:"links"`
	Digest    string         `json:"digest"`
	HeadSeq   int64          `json:"headSeq"`
}

// EnsureRequest creates or returns the main thread for a run/session.
type EnsureRequest struct {
	SpaceID   string `json:"spaceId"`
	SessionID string `json:"sessionId"`
	RunID     string `json:"runId"`
}

// ByRunView resolves run → thread (+ optional fold summary).
type ByRunView struct {
	RunID  string `json:"runId"`
	Thread Thread `json:"thread"`
}
