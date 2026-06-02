package memory

import "time"

const CurrentSchemaVersion = 1

var (
	validLayers       = map[string]struct{}{"L0": {}, "L1": {}, "L2": {}, "L3": {}}
	validStatuses     = map[string]struct{}{"candidate": {}, "approved": {}, "rejected": {}, "deprecated": {}}
	validSensitivity  = map[string]struct{}{"normal": {}, "restricted": {}, "secret": {}}
	validEvidenceKind = map[string]struct{}{"file": {}, "pr": {}, "ci": {}, "url": {}}
	validDecisions    = map[string]struct{}{"approve": {}, "reject": {}, "deprecate": {}}
)

// EvidenceInput is required for L1+ candidates.
type EvidenceInput struct {
	Kind   string         `json:"kind" binding:"required"`
	Ref    string         `json:"ref" binding:"required"`
	Digest string         `json:"digest,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

// CreateCandidateRequest creates a memory candidate.
type CreateCandidateRequest struct {
	Layer       string          `json:"layer" binding:"required"`
	Title       string          `json:"title" binding:"required"`
	Body        string          `json:"body" binding:"required"`
	ScopeRepo   string          `json:"scopeRepo,omitempty"`
	ScopeTeam   string          `json:"scopeTeam,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	TTLDays     *int            `json:"ttlDays,omitempty"`
	Sensitivity string          `json:"sensitivity,omitempty"`
	Evidence    []EvidenceInput `json:"evidence,omitempty"`
	RunID       string          `json:"runId,omitempty"`
	TraceID     string          `json:"traceId,omitempty"`
	ActorID     string          `json:"actorId,omitempty"`
}

// CreateCandidateResponse returns the new candidate id.
type CreateCandidateResponse struct {
	CandidateID string `json:"candidateId"`
}

// EvidenceView is stored evidence returned in API responses.
type EvidenceView struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Digest string `json:"digest,omitempty"`
}

// RecordView is a memory record with optional evidence.
type RecordView struct {
	ID            string         `json:"id"`
	Layer         string         `json:"layer"`
	Status        string         `json:"status"`
	SchemaVersion int            `json:"schemaVersion"`
	Title         string         `json:"title"`
	Body          string         `json:"body"`
	ScopeRepo     string         `json:"scopeRepo,omitempty"`
	ScopeTeam     string         `json:"scopeTeam,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	TTLDays       *int           `json:"ttlDays,omitempty"`
	Sensitivity   string         `json:"sensitivity"`
	DedupeKey     string         `json:"dedupeKey,omitempty"`
	CreatedAt     int64          `json:"createdAt"`
	UpdatedAt     int64          `json:"updatedAt"`
	Evidence      []EvidenceView `json:"evidence,omitempty"`
}

// ListCandidatesResponse paginates candidates.
type ListCandidatesResponse struct {
	Items  []RecordView `json:"items"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
	Total  int64        `json:"total"`
}

// ReviewRequest records a review decision.
type ReviewRequest struct {
	Decision      string `json:"decision" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	PolicyProfile string `json:"policyProfile" binding:"required"`
	ReviewerID    string `json:"reviewerId,omitempty"`
	ActorID       string `json:"actorId,omitempty"`
	RunID         string `json:"runId,omitempty"`
	TraceID       string `json:"traceId,omitempty"`
}

// ReviewResponse confirms review applied.
type ReviewResponse struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}

// QueryRequest searches approved memory.
type QueryRequest struct {
	Text   string            `json:"text" binding:"required"`
	Layers []string          `json:"layers,omitempty"`
	Scope  map[string]string `json:"scope,omitempty"`
	TopK   int               `json:"topK,omitempty"`
}

// QueryResponse returns matching records.
type QueryResponse struct {
	Items []RecordView `json:"items"`
}

// HitUsedRequest records memory usage in a run.
type HitUsedRequest struct {
	RunID     string   `json:"runId" binding:"required"`
	RecordIDs []string `json:"recordIds" binding:"required"`
	TraceID   string   `json:"traceId,omitempty"`
	ActorID   string   `json:"actorId,omitempty"`
}

// HitUsedResponse confirms audit write.
type HitUsedResponse struct {
	OK bool `json:"ok"`
}

func ms(t time.Time) int64 { return t.UTC().UnixMilli() }
