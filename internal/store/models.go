package store

import "time"

// RunRecord indexes a delivery run for listing and replay.
type RunRecord struct {
	ID              string     `gorm:"primaryKey;size:64"`
	TraceID         string     `gorm:"index;size:64;not null"`
	ScenarioName    string     `gorm:"size:128;not null"`
	ScenarioVersion string     `gorm:"size:64;not null"`
	PolicyProfile   string     `gorm:"size:64;not null;default:default"`
	Status          string     `gorm:"size:32;not null;index"`
	SpaceID         string     `gorm:"size:64;not null;default:local;index"`
	ActorRole       string     `gorm:"size:64;not null;default:maintainer"`
	InputsDigest    string     `gorm:"size:128"`
	RepoRoot        string     `gorm:"size:512"`
	StartedAt       time.Time  `gorm:"not null"`
	FinishedAt      *time.Time `gorm:"index"`
	Recovered       bool       `gorm:"not null;default:false"`
	ErrorCode       string     `gorm:"size:64"`
	ErrorMessage    string     `gorm:"size:1024"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (RunRecord) TableName() string { return "runs" }

// RunStep persists step-level state for timeline, resume, and gate rendering.
type RunStep struct {
	ID           string     `gorm:"primaryKey;size:64"`
	RunID        string     `gorm:"index:idx_run_steps_run_order,priority:1;size:64;not null"`
	StepID       string     `gorm:"size:128;not null"`
	StepOrder    int        `gorm:"index:idx_run_steps_run_order,priority:2;not null"`
	Role         string     `gorm:"size:64"`
	Kind         string     `gorm:"size:64;not null"`
	Status       string     `gorm:"size:32;not null;index"`
	StartedAt    *time.Time `gorm:"index"`
	FinishedAt   *time.Time `gorm:"index"`
	DurationMs   int64
	ErrorCode    string `gorm:"size:64"`
	ErrorMessage string `gorm:"size:1024"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (RunStep) TableName() string { return "run_steps" }

// ToolCall persists native ToolBus calls with audit-friendly metadata.
type ToolCall struct {
	ID          string `gorm:"primaryKey;size:64"`
	RunID       string `gorm:"index;size:64;not null"`
	TraceID     string `gorm:"index;size:64"`
	StepID      string `gorm:"index;size:128"`
	Tool        string `gorm:"size:128;not null;index"`
	Risk        string `gorm:"size:32;not null"`
	Status      string `gorm:"size:32;not null;index"`
	ArgsDigest  string `gorm:"size:128"`
	OutputJSON  string `gorm:"type:text;not null;default:'{}'"`
	Error       string `gorm:"type:text"`
	DurationMs  int64
	Attempt     int `gorm:"not null;default:1"`
	TimeoutMs   int64
	CreatedAt   time.Time
	CompletedAt *time.Time
}

func (ToolCall) TableName() string { return "tool_calls" }

// AgentTask tracks external agent execution through ExecGo/Codex.
type AgentTask struct {
	ID            string `gorm:"primaryKey;size:64"`
	RunID         string `gorm:"index;size:64;not null"`
	TraceID       string `gorm:"index;size:64"`
	StepID        string `gorm:"index;size:128"`
	Adapter       string `gorm:"size:64;not null"`
	AgentID       string `gorm:"size:128"`
	SessionID     string `gorm:"size:128"`
	ActionID      string `gorm:"size:128;index"`
	ExecGoTaskID  string `gorm:"size:128;index"`
	Status        string `gorm:"size:32;not null;index"`
	PromptDigest  string `gorm:"size:128"`
	StdoutSummary string `gorm:"type:text"`
	StderrSummary string `gorm:"type:text"`
	ExitCode      *int
	ErrorCode     string `gorm:"size:64"`
	ErrorMessage  string `gorm:"type:text"`
	DurationMs    int64
	TimeoutMs     int64
	CreatedAt     time.Time
	StartedAt     *time.Time
	CompletedAt   *time.Time
}

func (AgentTask) TableName() string { return "agent_tasks" }

// ArtifactIndex makes artifact discovery queryable without reading every manifest.
type ArtifactIndex struct {
	ID          string `gorm:"primaryKey;size:64"`
	RunID       string `gorm:"index;size:64;not null"`
	StepID      string `gorm:"index;size:128"`
	Type        string `gorm:"size:64;not null;index"`
	Name        string `gorm:"size:256"`
	URI         string `gorm:"size:1024;not null"`
	StoreKey    string `gorm:"size:1024"`
	Digest      string `gorm:"size:128;not null;index"`
	ContentType string `gorm:"size:128"`
	SizeBytes   int64
	EventRange  string `gorm:"size:128"`
	CreatedAt   time.Time
}

func (ArtifactIndex) TableName() string { return "artifact_index" }

// Checkpoint stores recoverable per-step snapshots.
type Checkpoint struct {
	ID             string `gorm:"primaryKey;size:64"`
	RunID          string `gorm:"index;size:64;not null"`
	StepID         string `gorm:"index;size:128;not null"`
	SnapshotDigest string `gorm:"size:128;not null"`
	URI            string `gorm:"size:1024"`
	StoreKey       string `gorm:"size:1024"`
	ContentType    string `gorm:"size:128"`
	SizeBytes      int64
	Strategy       string `gorm:"size:64"`
	CreatedAt      time.Time
}

func (Checkpoint) TableName() string { return "checkpoints" }

// RunEvent persists the event stream (source of truth for SSE/replay).
type RunEvent struct {
	ID          string `gorm:"primaryKey;size:64"`
	RunID       string `gorm:"index:idx_run_seq,priority:1;uniqueIndex:uniq_run_seq;size:64;not null"`
	Seq         int64  `gorm:"index:idx_run_seq,priority:2;uniqueIndex:uniq_run_seq;not null"`
	TS          int64  `gorm:"not null"`
	Type        string `gorm:"size:128;not null;index"`
	Severity    string `gorm:"size:16;not null;default:info"`
	PayloadJSON string `gorm:"type:text;not null"`
	CreatedAt   time.Time
}

func (RunEvent) TableName() string { return "run_events" }

// MemoryRecord implements layered memory (appendix C subset for M0).
type MemoryRecord struct {
	ID            string `gorm:"primaryKey;size:64"`
	Layer         string `gorm:"size:8;not null;index:idx_memory_layer_status,priority:1"`
	Status        string `gorm:"size:32;not null;index:idx_memory_layer_status,priority:2"`
	SpaceID       string `gorm:"size:64;not null;default:local;index"`
	SchemaVersion int    `gorm:"not null;default:1"`
	Title         string `gorm:"size:512;not null"`
	Body          string `gorm:"type:text;not null"`
	ScopeRepo     string `gorm:"size:512;index"`
	ScopeTeam     string `gorm:"size:128"`
	TagsJSON      string `gorm:"type:text;not null;default:'[]'"`
	TTLDays       *int
	Sensitivity   string  `gorm:"size:32;not null;default:normal"`
	DedupeKey     string  `gorm:"size:128;index"`
	Confidence    float64 `gorm:"not null;default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (MemoryRecord) TableName() string { return "memory_records" }

type MemoryEvidence struct {
	ID        string `gorm:"primaryKey;size:64"`
	MemoryID  string `gorm:"index;size:64;not null"`
	Kind      string `gorm:"size:16;not null"`
	Ref       string `gorm:"size:1024;not null"`
	Digest    string `gorm:"size:128"`
	MetaJSON  string `gorm:"type:text;not null;default:'{}'"`
	CreatedAt time.Time
}

func (MemoryEvidence) TableName() string { return "memory_evidence" }

type MemoryReview struct {
	ID            string `gorm:"primaryKey;size:64"`
	MemoryID      string `gorm:"index;size:64;not null"`
	Decision      string `gorm:"size:32;not null"`
	ReviewerID    string `gorm:"size:128"`
	Reason        string `gorm:"type:text;not null"`
	PolicyProfile string `gorm:"size:64;not null"`
	CreatedAt     time.Time
}

func (MemoryReview) TableName() string { return "memory_reviews" }

// MemoryEdge describes governance relationships between memory records.
type MemoryEdge struct {
	ID         string  `gorm:"primaryKey;size:64"`
	SpaceID    string  `gorm:"size:64;not null;default:local;index"`
	FromID     string  `gorm:"index;size:64;not null"`
	ToID       string  `gorm:"index;size:64;not null"`
	Kind       string  `gorm:"size:32;not null;index"` // duplicate|conflict|replaces|supports
	Confidence float64 `gorm:"not null;default:0"`
	Reason     string  `gorm:"type:text"`
	CreatedAt  time.Time
}

func (MemoryEdge) TableName() string { return "memory_edges" }

// MemoryMigration records one applied memory schema migration batch (appendix C).
type MemoryMigration struct {
	ID          string `gorm:"primaryKey;size:64"`
	FromVersion int    `gorm:"not null"`
	ToVersion   int    `gorm:"not null"`
	ToolVersion string `gorm:"size:64;not null"`
	Summary     string `gorm:"type:text;not null"`
	MetaJSON    string `gorm:"type:text;not null;default:'{}'"`
	AppliedAt   time.Time
}

func (MemoryMigration) TableName() string { return "memory_migrations" }

// RAGDocument indexes a repo file for FTS/BM25-style retrieval.
type RAGDocument struct {
	ID        string `gorm:"primaryKey;size:64"`
	SpaceID   string `gorm:"size:64;not null;default:local;index"`
	RepoRoot  string `gorm:"size:512;not null;index"`
	Path      string `gorm:"size:1024;not null;index"`
	Digest    string `gorm:"size:128;not null;index"`
	SizeBytes int64
	UpdatedAt time.Time
	CreatedAt time.Time
}

func (RAGDocument) TableName() string { return "rag_documents" }

// RAGChunk stores small file chunks with line ranges and citation refs.
type RAGChunk struct {
	ID         string `gorm:"primaryKey;size:64"`
	DocumentID string `gorm:"index;size:64;not null"`
	SpaceID    string `gorm:"size:64;not null;default:local;index"`
	RepoRoot   string `gorm:"size:512;not null;index"`
	Path       string `gorm:"size:1024;not null;index"`
	Symbol     string `gorm:"size:256;index"`
	StartLine  int
	EndLine    int
	Text       string `gorm:"type:text;not null"`
	Digest     string `gorm:"size:128;not null;index"`
	CreatedAt  time.Time
}

func (RAGChunk) TableName() string { return "rag_chunks" }

// ModelUsage is the M1 cost/accounting ledger for non-coding model calls.
type ModelUsage struct {
	ID           string `gorm:"primaryKey;size:64"`
	RunID        string `gorm:"index;size:64"`
	StepID       string `gorm:"index;size:128"`
	Provider     string `gorm:"size:64;not null"`
	Model        string `gorm:"size:128;not null"`
	InputTokens  int64
	OutputTokens int64
	CostMicros   int64
	Status       string `gorm:"size:32;not null;index"`
	CreatedAt    time.Time
}

func (ModelUsage) TableName() string { return "model_usage" }

// QualityMetric records M1 quality baseline values.
type QualityMetric struct {
	ID        string `gorm:"primaryKey;size:64"`
	RunID     string `gorm:"index;size:64"`
	SpaceID   string `gorm:"size:64;not null;default:local;index"`
	Name      string `gorm:"size:128;not null;index"`
	Value     float64
	Unit      string `gorm:"size:32"`
	CreatedAt time.Time
}

func (QualityMetric) TableName() string { return "quality_metrics" }

// MCPTool tracks registered MCP tools and isolation state.
type MCPTool struct {
	ID         string `gorm:"primaryKey;size:64"`
	SpaceID    string `gorm:"size:64;not null;default:local;index"`
	Name       string `gorm:"size:128;not null;index"`
	Server     string `gorm:"size:256;not null"`
	SchemaJSON string `gorm:"type:text;not null;default:'{}'"`
	Risk       string `gorm:"size:32;not null"`
	Status     string `gorm:"size:32;not null;index"`
	LastError  string `gorm:"type:text"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (MCPTool) TableName() string { return "mcp_tools" }

// Feedback records user feedback over runs, artifacts, and memory hits.
type Feedback struct {
	ID         string `gorm:"primaryKey;size:64"`
	SpaceID    string `gorm:"size:64;not null;default:local;index"`
	TargetType string `gorm:"size:64;not null;index"`
	TargetID   string `gorm:"size:128;not null;index"`
	RunID      string `gorm:"size:64;index"` // optional; uniqueness with target when set
	Rating     int
	Category   string `gorm:"size:64;not null;default:general;index"`
	Status     string `gorm:"size:32;not null;default:open;index"`
	Severity   string `gorm:"size:32;not null;default:info;index"`
	Source     string `gorm:"size:64;not null;default:ui;index"`
	Comment    string `gorm:"type:text"`
	ActorID    string `gorm:"size:128"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (Feedback) TableName() string { return "feedback" }

// RepoConnection stores a space-scoped external repository integration.
type RepoConnection struct {
	ID            string     `json:"id" gorm:"primaryKey;size:64"`
	SpaceID       string     `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	Provider      string     `json:"provider" gorm:"size:32;not null;index"`
	Owner         string     `json:"owner" gorm:"size:128;not null;index"`
	Repo          string     `json:"repo" gorm:"size:128;not null;index"`
	DefaultBranch string     `json:"defaultBranch" gorm:"size:128;not null;default:main"`
	SecretID      string     `json:"secretId" gorm:"size:64;not null;index"`
	Status        string     `json:"status" gorm:"size:32;not null;default:active;index"`
	LastSyncAt    *time.Time `json:"lastSyncAt,omitempty"`
	CreatedBy     string     `json:"createdBy" gorm:"size:128"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (RepoConnection) TableName() string { return "repo_connections" }

// CIRun stores a GitHub Actions workflow run snapshot for KPI and diagnosis.
type CIRun struct {
	ID            string     `json:"id" gorm:"primaryKey;size:64"`
	SpaceID       string     `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	ConnectionID  string     `json:"connectionId" gorm:"size:64;not null;index"`
	ProviderRunID string     `json:"providerRunId" gorm:"size:128;not null;index"`
	Workflow      string     `json:"workflow" gorm:"size:256;index"`
	Status        string     `json:"status" gorm:"size:32;not null;index"`
	Conclusion    string     `json:"conclusion" gorm:"size:32;index"`
	Attempt       int        `json:"attempt" gorm:"not null;default:1;index"`
	CommitSHA     string     `json:"commitSha" gorm:"size:64;index"`
	Branch        string     `json:"branch" gorm:"size:256;index"`
	RunURL        string     `json:"runUrl" gorm:"size:1024"`
	StartedAt     *time.Time `json:"startedAt,omitempty" gorm:"index"`
	CompletedAt   *time.Time `json:"completedAt,omitempty" gorm:"index"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (CIRun) TableName() string { return "ci_runs" }

// CIJob stores a GitHub Actions job snapshot and optional log digest.
type CIJob struct {
	ID            string     `json:"id" gorm:"primaryKey;size:64"`
	SpaceID       string     `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	ConnectionID  string     `json:"connectionId" gorm:"size:64;not null;index"`
	CIRunID       string     `json:"ciRunId" gorm:"size:64;not null;index"`
	ProviderJobID string     `json:"providerJobId" gorm:"size:128;not null;index"`
	Name          string     `json:"name" gorm:"size:256;index"`
	Status        string     `json:"status" gorm:"size:32;not null;index"`
	Conclusion    string     `json:"conclusion" gorm:"size:32;index"`
	Attempt       int        `json:"attempt" gorm:"not null;default:1;index"`
	LogDigest     string     `json:"logDigest" gorm:"size:128;index"`
	StartedAt     *time.Time `json:"startedAt,omitempty" gorm:"index"`
	CompletedAt   *time.Time `json:"completedAt,omitempty" gorm:"index"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (CIJob) TableName() string { return "ci_jobs" }

// CIDiagnosis records deterministic diagnosis output for a CI failure.
type CIDiagnosis struct {
	ID                 string     `json:"id" gorm:"primaryKey;size:64"`
	SpaceID            string     `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	ConnectionID       string     `json:"connectionId" gorm:"size:64;index"`
	CIRunID            string     `json:"ciRunId" gorm:"size:64;index"`
	CIJobID            string     `json:"ciJobId" gorm:"size:64;index"`
	Status             string     `json:"status" gorm:"size:32;not null;index"`
	RootCause          string     `json:"rootCause" gorm:"size:128;not null;index"`
	FixSuggestionsJSON string     `json:"fixSuggestionsJson" gorm:"type:text;not null;default:'[]'"`
	EvidenceRefsJSON   string     `json:"evidenceRefsJson" gorm:"type:text;not null;default:'[]'"`
	Confidence         float64    `json:"confidence"`
	Adopted            bool       `json:"adopted" gorm:"not null;default:false;index"`
	DecisionStatus     string     `json:"decisionStatus" gorm:"size:32;not null;default:pending;index"`
	DecisionReason     string     `json:"decisionReason" gorm:"type:text"`
	DecidedBy          string     `json:"decidedBy" gorm:"size:128"`
	DecidedAt          *time.Time `json:"decidedAt,omitempty" gorm:"index"`
	LogDigest          string     `json:"logDigest" gorm:"size:128;index"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (CIDiagnosis) TableName() string { return "ci_diagnoses" }

type AlertRule struct {
	ID            string    `json:"id" gorm:"primaryKey;size:64"`
	SpaceID       string    `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	Name          string    `json:"name" gorm:"size:128;not null;index"`
	Metric        string    `json:"metric" gorm:"size:128;not null;index"`
	Condition     string    `json:"condition" gorm:"size:16;not null;default:gt"`
	Threshold     float64   `json:"threshold"`
	WindowMinutes int       `json:"windowMinutes" gorm:"not null;default:60"`
	Severity      string    `json:"severity" gorm:"size:32;not null;default:warn;index"`
	Enabled       bool      `json:"enabled" gorm:"not null;default:true;index"`
	Description   string    `json:"description" gorm:"size:512"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (AlertRule) TableName() string { return "alert_rules" }

type AlertEvent struct {
	ID               string     `json:"id" gorm:"primaryKey;size:64"`
	SpaceID          string     `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	RuleID           string     `json:"ruleId" gorm:"size:64;index"`
	RuleName         string     `json:"ruleName" gorm:"size:128;index"`
	Severity         string     `json:"severity" gorm:"size:32;not null;index"`
	Status           string     `json:"status" gorm:"size:32;not null;default:active;index"`
	TargetType       string     `json:"targetType" gorm:"size:64;index"`
	TargetID         string     `json:"targetId" gorm:"size:128;index"`
	Fingerprint      string     `json:"fingerprint" gorm:"size:128;index"`
	Message          string     `json:"message" gorm:"type:text;not null"`
	EvidenceRefsJSON string     `json:"evidenceRefsJson" gorm:"type:text;not null;default:'[]'"`
	TriggeredAt      time.Time  `json:"triggeredAt" gorm:"index"`
	ResolvedAt       *time.Time `json:"resolvedAt,omitempty" gorm:"index"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (AlertEvent) TableName() string { return "alert_events" }

type AlertSilence struct {
	ID        string    `json:"id" gorm:"primaryKey;size:64"`
	SpaceID   string    `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	RuleID    string    `json:"ruleId" gorm:"size:64;index"`
	Reason    string    `json:"reason" gorm:"type:text;not null"`
	CreatedBy string    `json:"createdBy" gorm:"size:128"`
	StartsAt  time.Time `json:"startsAt" gorm:"index"`
	EndsAt    time.Time `json:"endsAt" gorm:"index"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (AlertSilence) TableName() string { return "alert_silences" }

type ReleaseRecord struct {
	ID               string    `json:"id" gorm:"primaryKey;size:64"`
	SpaceID          string    `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	Version          string    `json:"version" gorm:"size:128;not null;index"`
	Title            string    `json:"title" gorm:"size:256;not null"`
	Status           string    `json:"status" gorm:"size:32;not null;default:draft;index"`
	CanaryStrategy   string    `json:"canaryStrategy" gorm:"type:text"`
	GateStatus       string    `json:"gateStatus" gorm:"size:32;not null;default:pending;index"`
	EvidenceRefsJSON string    `json:"evidenceRefsJson" gorm:"type:text;not null;default:'[]'"`
	CreatedBy        string    `json:"createdBy" gorm:"size:128"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (ReleaseRecord) TableName() string { return "release_records" }

type ReleaseChecklistItem struct {
	ID          string    `json:"id" gorm:"primaryKey;size:64"`
	SpaceID     string    `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	ReleaseID   string    `json:"releaseId" gorm:"size:64;not null;index"`
	ItemKey     string    `json:"itemKey" gorm:"size:128;not null;index"`
	Label       string    `json:"label" gorm:"size:512;not null"`
	Status      string    `json:"status" gorm:"size:32;not null;default:pending;index"`
	EvidenceRef string    `json:"evidenceRef" gorm:"size:1024"`
	UpdatedBy   string    `json:"updatedBy" gorm:"size:128"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (ReleaseChecklistItem) TableName() string { return "release_checklist_items" }

type ReleaseGateResult struct {
	ID               string    `json:"id" gorm:"primaryKey;size:64"`
	SpaceID          string    `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	ReleaseID        string    `json:"releaseId" gorm:"size:64;not null;index"`
	GateKey          string    `json:"gateKey" gorm:"size:128;not null;index"`
	Status           string    `json:"status" gorm:"size:32;not null;index"`
	Message          string    `json:"message" gorm:"type:text;not null"`
	EvidenceRefsJSON string    `json:"evidenceRefsJson" gorm:"type:text;not null;default:'[]'"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (ReleaseGateResult) TableName() string { return "release_gate_results" }

type RollbackDrill struct {
	ID               string    `json:"id" gorm:"primaryKey;size:64"`
	SpaceID          string    `json:"spaceId" gorm:"size:64;not null;default:local;index"`
	ReleaseID        string    `json:"releaseId" gorm:"size:64;not null;index"`
	Scenario         string    `json:"scenario" gorm:"size:256;not null"`
	Status           string    `json:"status" gorm:"size:32;not null;index"`
	DurationMs       int64     `json:"durationMs"`
	EvidenceRefsJSON string    `json:"evidenceRefsJson" gorm:"type:text;not null;default:'[]'"`
	Notes            string    `json:"notes" gorm:"type:text"`
	CreatedBy        string    `json:"createdBy" gorm:"size:128"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (RollbackDrill) TableName() string { return "rollback_drills" }

type SecretRecord struct {
	ID              string `gorm:"primaryKey;size:64"`
	SpaceID         string `gorm:"size:64;not null;default:local;uniqueIndex:uniq_secret_space_name,priority:1"`
	Name            string `gorm:"size:128;not null;uniqueIndex:uniq_secret_space_name,priority:2"`
	Description     string `gorm:"size:512"`
	Status          string `gorm:"size:32;not null;default:active;index"`
	ScopeJSON       string `gorm:"type:text;not null;default:'{}'"`
	ValueCiphertext string `gorm:"type:text;not null"`
	ValueDigest     string `gorm:"size:128;not null;index"`
	CreatedBy       string `gorm:"size:128"`
	UpdatedBy       string `gorm:"size:128"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastUsedAt      *time.Time
}

func (SecretRecord) TableName() string { return "secret_records" }

type AuditLog struct {
	ID          string `gorm:"primaryKey;size:64"`
	SpaceID     string `gorm:"size:64;not null;default:local;index"`
	TraceID     string `gorm:"index;size:64"`
	RunID       string `gorm:"index;size:64"`
	ActorID     string `gorm:"size:128"`
	EventType   string `gorm:"size:128;not null;index"`
	PayloadJSON string `gorm:"type:text;not null"`
	CreatedAt   time.Time
}

func (AuditLog) TableName() string { return "audit_log" }

type ApprovalRequest struct {
	ID             string `gorm:"primaryKey;size:64"`
	SpaceID        string `gorm:"size:64;not null;default:local;index"`
	RunID          string `gorm:"index;size:64;not null"`
	TraceID        string `gorm:"index;size:64"`
	StepID         string `gorm:"index;size:128;not null"`
	Gate           string `gorm:"size:64;not null;index"`
	Risk           string `gorm:"size:32"`
	Reason         string `gorm:"type:text;not null"`
	Status         string `gorm:"size:32;not null;default:pending;index"`
	RequestedBy    string `gorm:"size:128"`
	DecidedBy      string `gorm:"size:128"`
	DecisionReason string `gorm:"type:text"`
	EvidenceJSON   string `gorm:"type:text;not null;default:'{}'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DecidedAt      *time.Time
}

func (ApprovalRequest) TableName() string { return "approval_requests" }

type User struct {
	ID           string `gorm:"primaryKey;size:64"`
	Email        string `gorm:"size:256;uniqueIndex"`
	DisplayName  string `gorm:"size:256"`
	PasswordHash string `gorm:"size:256"`
	Status       string `gorm:"size:32;not null;default:active;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string { return "users" }

type Org struct {
	ID        string `gorm:"primaryKey;size:64"`
	Name      string `gorm:"size:256;not null"`
	Slug      string `gorm:"size:128;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Org) TableName() string { return "orgs" }

type Space struct {
	ID        string `gorm:"primaryKey;size:64"`
	OrgID     string `gorm:"index;size:64;not null"`
	Name      string `gorm:"size:256;not null"`
	Slug      string `gorm:"size:128;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Space) TableName() string { return "spaces" }

type Member struct {
	ID        string `gorm:"primaryKey;size:64"`
	OrgID     string `gorm:"index;size:64;not null"`
	SpaceID   string `gorm:"index;size:64"`
	UserID    string `gorm:"index;size:64;not null"`
	RoleID    string `gorm:"index;size:64;not null"`
	Status    string `gorm:"size:32;not null;default:active;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Member) TableName() string { return "members" }

type Role struct {
	ID          string `gorm:"primaryKey;size:64"`
	OrgID       string `gorm:"index;size:64"`
	Name        string `gorm:"size:128;not null;index"`
	Permissions string `gorm:"type:text;not null;default:'[]'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Role) TableName() string { return "roles" }

type ResourceScope struct {
	ID           string `gorm:"primaryKey;size:64"`
	SpaceID      string `gorm:"index;size:64;not null"`
	ResourceType string `gorm:"size:64;not null;index"`
	ResourceID   string `gorm:"size:128;not null;index"`
	PolicyJSON   string `gorm:"type:text;not null;default:'{}'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (ResourceScope) TableName() string { return "resource_scopes" }

type AuditExport struct {
	ID          string     `json:"id" gorm:"primaryKey;size:64"`
	SpaceID     string     `json:"spaceId" gorm:"index;size:64;not null"`
	Status      string     `json:"status" gorm:"size:32;not null;index"`
	URI         string     `json:"uri" gorm:"size:1024"`
	StoreKey    string     `json:"storeKey" gorm:"size:1024"`
	Digest      string     `json:"digest" gorm:"size:128;index"`
	ContentType string     `json:"contentType" gorm:"size:128"`
	SizeBytes   int64      `json:"sizeBytes"`
	RequestedBy string     `json:"requestedBy" gorm:"size:128"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

func (AuditExport) TableName() string { return "audit_exports" }

type AuditPolicy struct {
	SpaceID       string `gorm:"primaryKey;size:64"`
	RetentionDays int    `gorm:"not null;default:365"`
	RedactPayload bool   `gorm:"not null;default:false"`
	Locked        bool   `gorm:"not null;default:false"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (AuditPolicy) TableName() string { return "audit_policies" }

type PluginRegistry struct {
	ID           string `gorm:"primaryKey;size:64"`
	SpaceID      string `gorm:"index;size:64;not null;default:local"`
	Name         string `gorm:"size:128;not null;index"`
	Version      string `gorm:"size:64;not null"`
	Protocol     string `gorm:"size:32;not null;default:grpc;index"`
	ABI          string `gorm:"size:64;not null;default:ash.plugin.v1;index"`
	Endpoint     string `gorm:"size:512"`
	Capabilities string `gorm:"type:text;not null;default:'[]'"`
	Compatible   bool   `gorm:"not null;default:false"`
	Status        string `gorm:"size:32;not null;index"`
	LastError     string `gorm:"type:text"`
	LastExportAt  *time.Time
	ExportErrors  int64 `gorm:"not null;default:0"`
	DropCount     int64 `gorm:"not null;default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (PluginRegistry) TableName() string { return "plugin_registry" }

// ImproveProposal tracks self-iteration experiments (M1).
type ImproveProposal struct {
	ID              string `gorm:"primaryKey;size:64"`
	SpaceID         string `gorm:"index;size:64;not null;default:local"`
	Title           string `gorm:"size:256;not null"`
	Description     string `gorm:"type:text"`
	BaselineRunID   string `gorm:"index;size:64"`
	ExperimentRunID string `gorm:"index;size:64"`
	Status          string `gorm:"size:32;not null;index"` // draft|experimenting|canary|promoted|rolled_back
	ChangeSummary   string `gorm:"type:text"`
	CanaryPercent   int    `gorm:"not null;default:0"`
	CompareJSON     string `gorm:"type:text;not null;default:'{}'"`
	ActorID         string `gorm:"size:128"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (ImproveProposal) TableName() string { return "improve_proposals" }

// HarnessProfileVersion stores versioned agent harness configs (v2).
type HarnessProfileVersion struct {
	ID            string     `gorm:"primaryKey;size:64"`
	SpaceID       string     `gorm:"index;size:64;not null;uniqueIndex:uidx_harness_space_name_ver"`
	Name          string     `gorm:"size:128;not null;uniqueIndex:uidx_harness_space_name_ver"`
	Version       int        `gorm:"not null;uniqueIndex:uidx_harness_space_name_ver"`
	Status        string     `gorm:"size:32;not null;index"` // draft|in_review|active|archived
	SpecJSON      string     `gorm:"type:text;not null"`
	ParentVersion *int       `gorm:""`
	CreatedBy     string     `gorm:"size:128"`
	PromotedBy    string     `gorm:"size:128"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
	PromotedAt    *time.Time `gorm:""`
}

func (HarnessProfileVersion) TableName() string { return "harness_profile_versions" }

// ScenarioPatchDraft stores orchestration scenario DSL patch drafts (v2 DZ).
type ScenarioPatchDraft struct {
	ID            string     `gorm:"primaryKey;size:64"`
	SpaceID       string     `gorm:"index;size:64;not null"`
	ScenarioName  string     `gorm:"size:128;not null;index"`
	FromVersion   string     `gorm:"size:64"`
	ToVersion     string     `gorm:"size:64"`
	Title         string     `gorm:"size:256;not null"`
	DiffText      string     `gorm:"type:text;not null"`
	Status        string     `gorm:"size:32;not null;index"` // draft|in_review|approved|rejected|archived
	CreatedBy     string     `gorm:"size:128"`
	DecidedBy     string     `gorm:"size:128"`
	DecisionNote  string     `gorm:"type:text"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
	DecidedAt     *time.Time `gorm:""`
}

func (ScenarioPatchDraft) TableName() string { return "scenario_patch_drafts" }

// GoalPlan stores NL goal → scenario plan drafts awaiting approval (v2 DJ).
type GoalPlan struct {
	ID              string    `gorm:"primaryKey;size:64"`
	SpaceID         string    `gorm:"index;size:64;not null"`
	Goal            string    `gorm:"type:text;not null"`
	ScenarioName    string    `gorm:"size:128;not null;index"`
	ScenarioVersion string    `gorm:"size:64;not null"`
	PolicyProfile   string    `gorm:"size:64;not null;default:default"`
	RouteReason     string    `gorm:"size:128"`
	InputsJSON      string    `gorm:"type:text;not null;default:'{}'"`
	StepsJSON       string    `gorm:"type:text;not null;default:'[]'"`
	Status          string    `gorm:"size:32;not null;index"` // draft|approved|rejected|started
	RunID           string    `gorm:"size:64;index"`
	CreatedBy       string    `gorm:"size:128"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (GoalPlan) TableName() string { return "goal_plans" }

type SchemaMeta struct {
	Key       string `gorm:"primaryKey;size:64"`
	Value     string `gorm:"size:256;not null"`
	UpdatedAt time.Time
}

func (SchemaMeta) TableName() string { return "schema_meta" }
