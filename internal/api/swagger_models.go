package api

import (
	"time"

	"github.com/ash-repwiki/ash/internal/artifacts"
	"github.com/ash-repwiki/ash/internal/artifactstore"
	"github.com/ash-repwiki/ash/internal/modelrouter"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

// APIError is the unified error shape (M0).
type APIError struct {
	Code    string `json:"code" example:"INVALID_REQUEST"`
	Message string `json:"message" example:"validation failed"`
}

// APIErrorResponse wraps APIError for OpenAPI.
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// HealthResponse for liveness/readiness probes.
type HealthResponse struct {
	Status                    string   `json:"status" example:"ok"`
	Dialect                   string   `json:"dialect,omitempty" example:"postgres"`
	Error                     string   `json:"error,omitempty"`
	SchemaMode                string   `json:"schemaMode,omitempty" example:"sql"`
	SQLMigrationVersion       uint     `json:"sqlMigrationVersion,omitempty"`
	SQLMigrationExpected      uint     `json:"sqlMigrationExpected,omitempty"`
	PostgresRLSEnabled        bool     `json:"postgresRLSEnabled,omitempty"`
	PostgresRLSPolicyCount    int64    `json:"postgresRLSPolicyCount,omitempty"`
	PostgresRLSPolicyExpected int64    `json:"postgresRLSPolicyExpected,omitempty"`
	RLSCatalogSummary         string   `json:"rlsCatalogSummary,omitempty"`
	ReadinessWarnings         []string `json:"readinessWarnings,omitempty"`
	LiveGateHints             []string `json:"liveGateHints,omitempty"`
	OtelEnabled               bool     `json:"otelEnabled,omitempty"`
	AlertsEvalInterval        string   `json:"alertsEvalInterval,omitempty" example:"5m0s"`
	MemoryTTLSweepInterval    string   `json:"memoryTTLSweepInterval,omitempty" example:"24h0m0s"`
	MetricsEventReplayEnabled bool     `json:"metricsEventReplayEnabled,omitempty"`
	RetentionEventsDays       int      `json:"retentionEventsDays,omitempty" example:"90"`
	RetentionAuditDays        int      `json:"retentionAuditDays,omitempty" example:"365"`
	RetentionArtifactsDays    int      `json:"retentionArtifactsDays,omitempty" example:"30"`
	RetentionArtifactsMaxRuns int      `json:"retentionArtifactsMaxRuns,omitempty" example:"200"`
}

type AuthSessionResponse struct {
	Token string    `json:"token"`
	User  AuthUser  `json:"user"`
	Space AuthSpace `json:"space"`
}

type AuthUser struct {
	ID          string `json:"id"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type AuthSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PasswordChangeResponse struct {
	OK bool `json:"ok"`
}

type AuthMeResponse struct {
	User        AuthUser  `json:"user"`
	Space       AuthSpace `json:"space"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
}

// RunListResponse lists runs.
type RunListResponse struct {
	Items []runs.Summary `json:"items"`
}

// RunCreateResponse is returned when a run is created (execution may still fail in-body).
type RunCreateResponse struct {
	RunID          string `json:"runId"`
	TraceID        string `json:"traceId"`
	Status         string `json:"status,omitempty"`
	ExecutionError string `json:"executionError,omitempty"`
}

type TimelineAPIResponse struct {
	Items []runs.TimelineItem `json:"items"`
}

type ToolCallListResponse struct {
	Items []store.ToolCall `json:"items"`
}

type AgentTaskListResponse struct {
	Items []store.AgentTask `json:"items"`
}

type QualityMetricListResponse struct {
	Items []store.QualityMetric `json:"items"`
}

type CheckpointListResponse struct {
	Items []store.Checkpoint `json:"items"`
}

type ModelProviderListResponse struct {
	Items []modelrouter.Provider `json:"items"`
}

type MCPToolListResponse struct {
	Items []store.MCPTool `json:"items"`
}

type AuditLogListResponse struct {
	Items []store.AuditLog `json:"items"`
}

type ApprovalRequestListResponse struct {
	Items []store.ApprovalRequest `json:"items"`
}

type AuditRetentionApplyResponse struct {
	SpaceID       string    `json:"spaceId"`
	RetentionDays int       `json:"retentionDays"`
	Cutoff        time.Time `json:"cutoff"`
	Matched       int64     `json:"matched"`
	Deleted       int64     `json:"deleted"`
	DryRun        bool      `json:"dryRun"`
}

type AuditExportListResponse struct {
	Items []store.AuditExport `json:"items"`
}

type AuditExportAccessResponse struct {
	ExportID    string `json:"exportId"`
	URI         string `json:"uri"`
	SignedURL   string `json:"signedUrl"`
	ExpiresAt   int64  `json:"expiresAt"`
	Digest      string `json:"digest"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type PluginRegistryListResponse struct {
	Items []store.PluginRegistry `json:"items"`
}

type RoleListResponse struct {
	Items []store.Role `json:"items"`
}

type MemberListResponse struct {
	Items []store.Member `json:"items"`
}

type ResourceScopeView struct {
	ID           string `json:"id"`
	SpaceID      string `json:"spaceId"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	PolicyJSON   string `json:"policyJson"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}

type ResourceScopeListResponse struct {
	Items []ResourceScopeView `json:"items"`
}

type PluginABIProfileResponse struct {
	CurrentABI           string            `json:"currentAbi"`
	SupportedABIs        []string          `json:"supportedAbis"`
	SupportedProtocols   []string          `json:"supportedProtocols"`
	GRPCEnabled          bool              `json:"grpcEnabled"`
	PluginGRPCAddr       string            `json:"pluginGrpcAddr,omitempty"`
	ProtoPackage         string            `json:"protoPackage"`
	GoPackage            string            `json:"goPackage"`
	BreakingPolicy       string            `json:"breakingPolicy"`
	ProtoFiles           []PluginProtoFile `json:"protoFiles"`
	SigningAlg           string            `json:"signingAlg,omitempty"`
	SigningRequired      bool              `json:"signingRequired"`
	SigningKeyConfigured bool              `json:"signingKeyConfigured"`
	SignCapabilityPrefix string            `json:"signCapabilityPrefix,omitempty"`
}

type PluginProtoFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Bytes  int64  `json:"bytes"`
}

type StorageProfileResponse struct {
	Database      DatabaseProfile       `json:"database"`
	ArtifactStore artifactstore.Profile `json:"artifactStore"`
	ArtifactPaths artifacts.PathProfile `json:"artifactPaths"`
}

type SecretListResponse struct {
	Items []SecretResponse `json:"items"`
}

type SecretResponse struct {
	ID            string         `json:"id"`
	SpaceID       string         `json:"spaceId"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Status        string         `json:"status"`
	Scope         map[string]any `json:"scope"`
	ValueDigest   string         `json:"valueDigest"`
	RedactedValue string         `json:"redactedValue"`
	CreatedBy     string         `json:"createdBy,omitempty"`
	UpdatedBy     string         `json:"updatedBy,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	LastUsedAt    *time.Time     `json:"lastUsedAt,omitempty"`
}

type DatabaseProfile struct {
	Dialect                string `json:"dialect"`
	URLConfigured          bool   `json:"urlConfigured"`
	DataDir                string `json:"dataDir"`
	PostgresRLSEnabled     bool   `json:"postgresRLSEnabled,omitempty"`
	PostgresRLSForce       bool   `json:"postgresRLSForce,omitempty"`
	PostgresRLSPolicyCount int64  `json:"postgresRLSPolicyCount,omitempty"`
}

// ScenarioListResponse lists loaded scenarios.
type ScenarioListResponse struct {
	Items []rules.ScenarioSummary `json:"items"`
}

// ScenarioDetailResponse returns scenario document details.
type ScenarioDetailResponse struct {
	Version  string         `json:"version" example:"ash.rules/v0.1"`
	Scenario rules.Scenario `json:"scenario"`
	Hooks    []rules.Hook   `json:"hooks,omitempty"`
	YAML     string         `json:"yaml,omitempty"`
	Valid    bool           `json:"valid,omitempty"`
}

// ValidateScenarioRequest validates DSL YAML in JSON body.
type ValidateScenarioRequest struct {
	YAML string `json:"yaml" binding:"required" example:"version: ash.rules/v0.1"`
}

// ValidationResponse is returned by scenario validate endpoints.
type ValidationResponse struct {
	OK     bool                    `json:"ok" example:"true"`
	Issues []rules.ValidationIssue `json:"issues,omitempty"`
}
