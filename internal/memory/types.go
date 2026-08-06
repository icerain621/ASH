package memory

import "time"

const CurrentSchemaVersion = 2

// Default TTL policy for long-term layers (appendix C §4); applied by v1→v2 migration when unset.
const (
	DefaultTTLDaysL1 = BuiltinTTLDaysL1
	DefaultTTLDaysL2 = BuiltinTTLDaysL2
)

// DefaultTTLForLayer returns the catalog default ttl for L1/L2 approved records.
func DefaultTTLForLayer(layer string) *int {
	switch layer {
	case "L1":
		d := EffectiveTTLDaysL1()
		return &d
	case "L2":
		d := EffectiveTTLDaysL2()
		return &d
	default:
		return nil
	}
}

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
	SpaceID     string          `json:"spaceId,omitempty"`
}

// GovernanceHint summarizes a related memory record detected before review.
type GovernanceHint struct {
	MemoryID string `json:"memoryId"`
	Kind     string `json:"kind"` // duplicate|conflict|pending_duplicate
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// GovernanceHints are non-blocking warnings at candidate creation time.
type GovernanceHints struct {
	Duplicates []GovernanceHint `json:"duplicates,omitempty"`
	Conflicts  []GovernanceHint `json:"conflicts,omitempty"`
}

// CreateCandidateResponse returns the new candidate id.
type CreateCandidateResponse struct {
	CandidateID string           `json:"candidateId"`
	Governance  *GovernanceHints `json:"governance,omitempty"`
}

// EvidenceView is stored evidence returned in API responses.
type EvidenceView struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Digest string `json:"digest,omitempty"`
}

type EdgeView struct {
	ID         string  `json:"id"`
	FromID     string  `json:"fromId"`
	ToID       string  `json:"toId"`
	Kind       string  `json:"kind"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason,omitempty"`
	CreatedAt  int64   `json:"createdAt"`
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
	Confidence    float64        `json:"confidence"`
	CreatedAt     int64          `json:"createdAt"`
	UpdatedAt     int64          `json:"updatedAt"`
	Evidence      []EvidenceView `json:"evidence,omitempty"`
	Edges         []EdgeView     `json:"edges,omitempty"`
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
	Decision      string   `json:"decision" binding:"required"`
	Reason        string   `json:"reason" binding:"required"`
	PolicyProfile string   `json:"policyProfile" binding:"required"`
	ReviewerID    string   `json:"reviewerId,omitempty"`
	ActorID       string   `json:"actorId,omitempty"`
	RunID         string   `json:"runId,omitempty"`
	TraceID       string   `json:"traceId,omitempty"`
	Confidence    *float64 `json:"confidence,omitempty"`
	DuplicateOf   string   `json:"duplicateOf,omitempty"`
	Replaces      []string `json:"replaces,omitempty"`
	ConflictsWith []string `json:"conflictsWith,omitempty"`
}

// ReviewResponse confirms review applied.
type ReviewResponse struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}

// QueryRequest searches approved memory.
type QueryRequest struct {
	Text    string            `json:"text" binding:"required"`
	Layers  []string          `json:"layers,omitempty"`
	Scope   map[string]string `json:"scope,omitempty"`
	TopK    int               `json:"topK,omitempty"`
	RunID   string            `json:"runId,omitempty"`
	TraceID string            `json:"traceId,omitempty"`
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

// ApplyFeedbackDecayRequest lowers confidence after low-score feedback (R-04).
type ApplyFeedbackDecayRequest struct {
	SpaceID    string `json:"spaceId,omitempty"`
	MemoryID   string `json:"memoryId" binding:"required"`
	FeedbackID string `json:"feedbackId,omitempty"`
	Rating     int    `json:"rating" binding:"required"`
	ActorID    string `json:"actorId,omitempty"`
	RunID      string `json:"runId,omitempty"`
	TraceID    string `json:"traceId,omitempty"`
}

// ApplyFeedbackDecayResponse reports whether confidence changed.
type ApplyFeedbackDecayResponse struct {
	OK       bool    `json:"ok"`
	MemoryID string  `json:"memoryId,omitempty"`
	From     float64 `json:"from,omitempty"`
	To       float64 `json:"to,omitempty"`
	Adjusted bool    `json:"adjusted"`
}

func ms(t time.Time) int64 { return t.UTC().UnixMilli() }
