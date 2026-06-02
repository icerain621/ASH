package store

import "time"

// RunRecord indexes a delivery run for listing and replay.
type RunRecord struct {
	ID               string     `gorm:"primaryKey;size:64"`
	TraceID          string     `gorm:"index;size:64;not null"`
	ScenarioName     string     `gorm:"size:128;not null"`
	ScenarioVersion  string     `gorm:"size:64;not null"`
	PolicyProfile    string     `gorm:"size:64;not null;default:default"`
	Status           string     `gorm:"size:32;not null;index"`
	InputsDigest     string     `gorm:"size:128"`
	RepoRoot         string     `gorm:"size:512"`
	StartedAt        time.Time  `gorm:"not null"`
	FinishedAt       *time.Time `gorm:"index"`
	Recovered        bool       `gorm:"not null;default:false"`
	ErrorCode        string     `gorm:"size:64"`
	ErrorMessage     string     `gorm:"size:1024"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (RunRecord) TableName() string { return "runs" }

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
	SchemaVersion int    `gorm:"not null;default:1"`
	Title         string `gorm:"size:512;not null"`
	Body          string `gorm:"type:text;not null"`
	ScopeRepo     string `gorm:"size:512;index"`
	ScopeTeam     string `gorm:"size:128"`
	TagsJSON      string `gorm:"type:text;not null;default:'[]'"`
	TTLDays       *int
	Sensitivity   string `gorm:"size:32;not null;default:normal"`
	DedupeKey     string `gorm:"size:128;index"`
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

type AuditLog struct {
	ID          string `gorm:"primaryKey;size:64"`
	TraceID     string `gorm:"index;size:64"`
	RunID       string `gorm:"index;size:64"`
	ActorID     string `gorm:"size:128"`
	EventType   string `gorm:"size:128;not null;index"`
	PayloadJSON string `gorm:"type:text;not null"`
	CreatedAt   time.Time
}

func (AuditLog) TableName() string { return "audit_log" }

type SchemaMeta struct {
	Key       string `gorm:"primaryKey;size:64"`
	Value     string `gorm:"size:256;not null"`
	UpdatedAt time.Time
}

func (SchemaMeta) TableName() string { return "schema_meta" }
