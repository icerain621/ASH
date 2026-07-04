package openapicheck

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractSchemasMatchSwagger(t *testing.T) {
	root := repoRoot(t)
	contractPath := filepath.Join(root, "doc/api/openapi-ash-v1.yaml")
	swaggerPath := filepath.Join(root, "internal/api/docs/swagger.yaml")

	pairs := []struct {
		contract string
		swagger  string
		minProps int
	}{
		{"ScaleReadinessResponse", "internal_api.ScaleReadinessResponse", 37},
		{"ProvenanceResponse", "internal_api.ProvenanceResponse", 8},
		{"SecretScanResponse", "internal_api.SecretScanResponse", 4},
		{"ComplianceExportResponse", "internal_api.ComplianceExportResponse", 3},
		{"PluginHealthSummary", "internal_api.PluginHealthSummary", 5},
		{"RAGProfileResponse", "github_com_ash-repwiki_ash_internal_rag.Profile", 6},
		{"OtelStatusResponse", "github_com_ash-repwiki_ash_internal_observability_otel.Status", 3},
		{"HealthResponse", "internal_api.HealthResponse", 11},
		{"RunSummaryResponse", "github_com_ash-repwiki_ash_internal_runs.Summary", 8},
		{"RunListResponse", "internal_api.RunListResponse", 1},
		{"RunCreateResponse", "github_com_ash-repwiki_ash_internal_runs.CreateResponse", 2},
		{"DoctorRunResponse", "internal_api.doctorRunResponse", 1},
		{"DoctorReportResponse", "github_com_ash-repwiki_ash_internal_doctor.Report", 4},
		{"ArtifactsManifestResponse", "internal_api.artifactsManifestResponse", 2},
		{"TimelineAPIResponse", "internal_api.TimelineAPIResponse", 1},
		{"StorageProfileResponse", "internal_api.StorageProfileResponse", 2},
		{"ValidationResponse", "internal_api.ValidationResponse", 1},
		{"ArtifactAccessResponse", "internal_api.ArtifactAccessResponse", 6},
		{"ScenarioListResponse", "internal_api.ScenarioListResponse", 1},
		{"ScenarioDetailResponse", "internal_api.ScenarioDetailResponse", 4},
		{"ToolCallListResponse", "internal_api.ToolCallListResponse", 1},
		{"AgentTaskListResponse", "internal_api.AgentTaskListResponse", 1},
		{"QualityMetricListResponse", "internal_api.QualityMetricListResponse", 1},
		{"PermissionMatrixResponse", "github_com_ash-repwiki_ash_internal_authz.MatrixResponse", 6},
		{"MetricsOverviewResponse", "github_com_ash-repwiki_ash_internal_metrics.Overview", 8},
		{"DiagnosisResponse", "github_com_ash-repwiki_ash_internal_ci.DiagnosisResponse", 15},
		{"CIDiagnosisListResponse", "internal_api.CIDiagnosisListResponse", 1},
		{"CIRunListResponse", "internal_api.CIRunListResponse", 1},
		{"CIJobListResponse", "internal_api.CIJobListResponse", 1},
		{"FeedbackListResponse", "internal_api.FeedbackListResponse", 1},
		{"PluginABIProfileResponse", "internal_api.PluginABIProfileResponse", 8},
		{"MCPToolListResponse", "internal_api.MCPToolListResponse", 1},
		{"ModelProviderListResponse", "internal_api.ModelProviderListResponse", 1},
		{"ModelRouterDecision", "github_com_ash-repwiki_ash_internal_modelrouter.Decision", 6},
		{"PluginRegistryListResponse", "internal_api.PluginRegistryListResponse", 1},
		{"ImproveListProposalsResponse", "github_com_ash-repwiki_ash_internal_improve.ListProposalsResponse", 1},
		{"ImproveProposalView", "github_com_ash-repwiki_ash_internal_improve.ProposalView", 8},
		{"ImproveStartExperimentResponse", "github_com_ash-repwiki_ash_internal_improve.StartExperimentResponse", 2},
		{"ImproveStatusResponse", "github_com_ash-repwiki_ash_internal_improve.StatusResponse", 2},
		{"MemoryListCandidatesResponse", "github_com_ash-repwiki_ash_internal_memory.ListCandidatesResponse", 3},
		{"MemoryCreateCandidateResponse", "github_com_ash-repwiki_ash_internal_memory.CreateCandidateResponse", 1},
		{"MemoryQueryResponse", "github_com_ash-repwiki_ash_internal_memory.QueryResponse", 1},
		{"MemoryReviewResponse", "github_com_ash-repwiki_ash_internal_memory.ReviewResponse", 2},
		{"MemoryHitUsedResponse", "github_com_ash-repwiki_ash_internal_memory.HitUsedResponse", 1},
		{"MemoryRunMigrationResponse", "github_com_ash-repwiki_ash_internal_memory.RunMigrationResponse", 6},
		{"MemoryTTLQueueResponse", "github_com_ash-repwiki_ash_internal_memory.TTLQueueResponse", 4},
		{"MemorySweepTTLResponse", "github_com_ash-repwiki_ash_internal_memory.SweepTTLResponse", 5},
		{"MemoryRecordView", "github_com_ash-repwiki_ash_internal_memory.RecordView", 15},
		{"MemberListResponse", "internal_api.MemberListResponse", 1},
		{"RoleListResponse", "internal_api.RoleListResponse", 1},
		{"ResourceScopeListResponse", "internal_api.ResourceScopeListResponse", 1},
		{"AlertRuleListResponse", "internal_api.AlertRuleListResponse", 1},
		{"AlertEventListResponse", "internal_api.AlertEventListResponse", 1},
		{"AlertEvaluationResult", "github_com_ash-repwiki_ash_internal_alerts.EvaluationResult", 3},
		{"WaterfallResponse", "github_com_ash-repwiki_ash_internal_observability.Waterfall", 6},
		{"TraceViewResponse", "github_com_ash-repwiki_ash_internal_alerts.TraceView", 6},
		{"ReleaseListResponse", "internal_api.ReleaseListResponse", 1},
		{"ReleaseChecklistResponse", "internal_api.ReleaseChecklistResponse", 1},
		{"ReleaseGateEvaluation", "github_com_ash-repwiki_ash_internal_releases.GateEvaluation", 5},
		{"ApprovalRequestListResponse", "internal_api.ApprovalRequestListResponse", 1},
		{"ApproveResponse", "github_com_ash-repwiki_ash_internal_runs.ApproveResponse", 2},
		{"AuditExportListResponse", "internal_api.AuditExportListResponse", 1},
		{"AuditExportAccessResponse", "internal_api.AuditExportAccessResponse", 6},
		{"AuditLogListResponse", "internal_api.AuditLogListResponse", 1},
		{"AuditRetentionApplyResponse", "internal_api.AuditRetentionApplyResponse", 6},
		{"AuthSessionResponse", "internal_api.AuthSessionResponse", 2},
		{"AuthMeResponse", "internal_api.AuthMeResponse", 3},
		{"PasswordChangeResponse", "internal_api.PasswordChangeResponse", 1},
		{"RepoConnectionListResponse", "internal_api.RepoConnectionListResponse", 1},
		{"SecretListResponse", "internal_api.SecretListResponse", 1},
		{"SecretResponse", "internal_api.SecretResponse", 10},
		{"RAGIndexResponse", "github_com_ash-repwiki_ash_internal_rag.IndexResponse", 2},
		{"RAGQueryResponse", "github_com_ash-repwiki_ash_internal_rag.QueryResponse", 2},
		{"RunReplayResponse", "github_com_ash-repwiki_ash_internal_runs.ReplayResponse", 2},
		{"RunResumeResponse", "github_com_ash-repwiki_ash_internal_runs.ResumeResponse", 2},
		{"CheckpointListResponse", "internal_api.CheckpointListResponse", 1},
		{"CheckpointAccessResponse", "internal_api.CheckpointAccessResponse", 6},
	}

	for _, pair := range pairs {
		pair := pair
		t.Run(pair.contract, func(t *testing.T) {
			contract, err := SchemaPropertyNames(contractPath, pair.contract)
			if err != nil {
				t.Fatal(err)
			}
			swagger, err := SwaggerDefinitionPropertyNames(swaggerPath, pair.swagger)
			if err != nil {
				t.Fatal(err)
			}
			missingInContract, missingInSwagger := DiffPropertyNames(contract, swagger)
			if len(missingInContract) > 0 || len(missingInSwagger) > 0 {
				t.Fatalf("%s drift:\n  contract only: %s\n  swagger only: %s",
					pair.contract,
					strings.Join(missingInSwagger, ", "),
					strings.Join(missingInContract, ", "))
			}
			if len(contract) < pair.minProps {
				t.Fatalf("%s properties=%d want >= %d", pair.contract, len(contract), pair.minProps)
			}
		})
	}
}

func TestNestedContractSchemasMatchSwagger(t *testing.T) {
	root := repoRoot(t)
	contractPath := filepath.Join(root, "doc/api/openapi-ash-v1.yaml")
	swaggerPath := filepath.Join(root, "internal/api/docs/swagger.yaml")

	pairs := []struct {
		contract string
		swagger  string
	}{
		{"ProvenanceLink", "internal_api.ProvenanceLink"},
		{"ScenarioRef", "github_com_ash-repwiki_ash_internal_runs.ScenarioRef"},
		{"LeakFinding", "github_com_ash-repwiki_ash_internal_security.LeakFinding"},
		{"RepoRef", "github_com_ash-repwiki_ash_internal_runs.RepoRef"},
		{"DoctorEvidence", "github_com_ash-repwiki_ash_internal_doctor.Evidence"},
		{"DoctorCaseResult", "github_com_ash-repwiki_ash_internal_doctor.CaseResult"},
		{"DoctorRunRequest", "internal_api.doctorRunRequest"},
		{"TimelineItem", "github_com_ash-repwiki_ash_internal_runs.TimelineItem"},
		{"DatabaseProfile", "internal_api.DatabaseProfile"},
		{"ArtifactStoreProfile", "github_com_ash-repwiki_ash_internal_artifactstore.Profile"},
		{"ValidationIssue", "github_com_ash-repwiki_ash_internal_rules.ValidationIssue"},
		{"ValidateScenarioRequest", "internal_api.ValidateScenarioRequest"},
		{"ScenarioSummaryItem", "github_com_ash-repwiki_ash_internal_rules.ScenarioSummary"},
		{"ToolCallItem", "github_com_ash-repwiki_ash_internal_store.ToolCall"},
		{"AgentTaskItem", "github_com_ash-repwiki_ash_internal_store.AgentTask"},
		{"QualityMetricItem", "github_com_ash-repwiki_ash_internal_store.QualityMetric"},
		{"PermissionDefItem", "github_com_ash-repwiki_ash_internal_authz.PermissionDef"},
		{"BuiltinRoleItem", "github_com_ash-repwiki_ash_internal_authz.BuiltinRole"},
		{"OrgRoleRow", "github_com_ash-repwiki_ash_internal_authz.OrgRoleRow"},
		{"ScenarioMatrixRow", "github_com_ash-repwiki_ash_internal_authz.ScenarioMatrixRow"},
		{"ToolRule", "github_com_ash-repwiki_ash_internal_authz.ToolRule"},
		{"MetricCard", "github_com_ash-repwiki_ash_internal_metrics.MetricCard"},
		{"MetricTrend", "github_com_ash-repwiki_ash_internal_metrics.MetricTrend"},
		{"MetricBreakdown", "github_com_ash-repwiki_ash_internal_metrics.MetricBreakdown"},
		{"DataQualityNote", "github_com_ash-repwiki_ash_internal_metrics.DataQualityNote"},
		{"MetricPoint", "github_com_ash-repwiki_ash_internal_metrics.MetricPoint"},
		{"BreakdownItem", "github_com_ash-repwiki_ash_internal_metrics.BreakdownItem"},
		{"CIRunItem", "github_com_ash-repwiki_ash_internal_store.CIRun"},
		{"CIJobItem", "github_com_ash-repwiki_ash_internal_store.CIJob"},
		{"FeedbackItem", "github_com_ash-repwiki_ash_internal_store.Feedback"},
		{"PluginProtoFile", "internal_api.PluginProtoFile"},
		{"DiagnoseCIFailureRequest", "internal_api.diagnoseCIFailureRequest"},
		{"DecideCIDiagnosisRequest", "internal_api.decideCIDiagnosisRequest"},
		{"MCPToolItem", "github_com_ash-repwiki_ash_internal_store.MCPTool"},
		{"ModelProviderItem", "github_com_ash-repwiki_ash_internal_modelrouter.Provider"},
		{"ModelRouterRequest", "github_com_ash-repwiki_ash_internal_modelrouter.Request"},
		{"PluginRegistryItem", "github_com_ash-repwiki_ash_internal_store.PluginRegistry"},
		{"ImproveArtifactCompare", "github_com_ash-repwiki_ash_internal_improve.ArtifactCompare"},
		{"ImproveCreateProposalRequest", "github_com_ash-repwiki_ash_internal_improve.CreateProposalRequest"},
		{"ImproveCanaryRequest", "github_com_ash-repwiki_ash_internal_improve.CanaryRequest"},
		{"MemoryEvidenceInput", "github_com_ash-repwiki_ash_internal_memory.EvidenceInput"},
		{"MemoryEvidenceView", "github_com_ash-repwiki_ash_internal_memory.EvidenceView"},
		{"MemoryEdgeView", "github_com_ash-repwiki_ash_internal_memory.EdgeView"},
		{"MemoryGovernanceHint", "github_com_ash-repwiki_ash_internal_memory.GovernanceHint"},
		{"MemoryGovernanceHints", "github_com_ash-repwiki_ash_internal_memory.GovernanceHints"},
		{"MemoryCreateCandidateRequest", "github_com_ash-repwiki_ash_internal_memory.CreateCandidateRequest"},
		{"MemoryReviewRequest", "github_com_ash-repwiki_ash_internal_memory.ReviewRequest"},
		{"MemoryHitUsedRequest", "github_com_ash-repwiki_ash_internal_memory.HitUsedRequest"},
		{"MemoryQueryRequest", "github_com_ash-repwiki_ash_internal_memory.QueryRequest"},
		{"MemoryRunMigrationRequest", "github_com_ash-repwiki_ash_internal_memory.RunMigrationRequest"},
		{"MemorySweepTTLRequest", "github_com_ash-repwiki_ash_internal_memory.SweepTTLRequest"},
		{"MemoryTTLQueueItem", "github_com_ash-repwiki_ash_internal_memory.TTLQueueItem"},
		{"OrgItem", "github_com_ash-repwiki_ash_internal_store.Org"},
		{"CreateOrgRequest", "internal_api.createOrgRequest"},
		{"SpaceItem", "github_com_ash-repwiki_ash_internal_store.Space"},
		{"CreateSpaceRequest", "internal_api.createSpaceRequest"},
		{"MemberItem", "github_com_ash-repwiki_ash_internal_store.Member"},
		{"CreateMemberRequest", "internal_api.createMemberRequest"},
		{"RoleItem", "github_com_ash-repwiki_ash_internal_store.Role"},
		{"CreateRoleRequest", "internal_api.createRoleRequest"},
		{"ResourceScopeView", "internal_api.ResourceScopeView"},
		{"UpdateResourceScopeRequest", "internal_api.updateResourceScopeRequest"},
		{"AlertRuleItem", "github_com_ash-repwiki_ash_internal_store.AlertRule"},
		{"AlertRuleInput", "github_com_ash-repwiki_ash_internal_alerts.RuleInput"},
		{"PutAlertRulesRequest", "internal_api.putAlertRulesRequest"},
		{"AlertEventItem", "github_com_ash-repwiki_ash_internal_store.AlertEvent"},
		{"AlertRuleEvaluation", "github_com_ash-repwiki_ash_internal_alerts.RuleEvaluation"},
		{"WaterfallSpan", "github_com_ash-repwiki_ash_internal_observability.Span"},
		{"WaterfallMetric", "github_com_ash-repwiki_ash_internal_observability.Metric"},
		{"FailureAttributionItem", "github_com_ash-repwiki_ash_internal_observability.FailureAttribution"},
		{"PluginExportReportRequest", "internal_api.pluginExportReportRequest"},
		{"ReleaseRecordItem", "github_com_ash-repwiki_ash_internal_store.ReleaseRecord"},
		{"CreateReleaseRequest", "internal_api.createReleaseRequest"},
		{"ReleaseChecklistItem", "github_com_ash-repwiki_ash_internal_store.ReleaseChecklistItem"},
		{"ChecklistUpdateItem", "github_com_ash-repwiki_ash_internal_releases.ChecklistUpdate"},
		{"PatchReleaseChecklistRequest", "internal_api.patchReleaseChecklistRequest"},
		{"ReleaseGateResultItem", "github_com_ash-repwiki_ash_internal_store.ReleaseGateResult"},
		{"RollbackDrillItem", "github_com_ash-repwiki_ash_internal_store.RollbackDrill"},
		{"CreateRollbackDrillRequest", "internal_api.createRollbackDrillRequest"},
		{"ApprovalRequestItem", "github_com_ash-repwiki_ash_internal_store.ApprovalRequest"},
		{"ApproveRequest", "github_com_ash-repwiki_ash_internal_runs.ApproveRequest"},
		{"RejectApprovalRequest", "internal_api.rejectApprovalRequest"},
		{"CancelResponse", "github_com_ash-repwiki_ash_internal_runs.CancelResponse"},
		{"AuditExportItem", "github_com_ash-repwiki_ash_internal_store.AuditExport"},
		{"AuditLogItem", "github_com_ash-repwiki_ash_internal_store.AuditLog"},
		{"AuditPolicyItem", "github_com_ash-repwiki_ash_internal_store.AuditPolicy"},
		{"UpdateAuditPolicyRequest", "internal_api.updateAuditPolicyRequest"},
		{"ApplyAuditRetentionRequest", "internal_api.applyAuditRetentionRequest"},
		{"AuthUser", "internal_api.AuthUser"},
		{"AuthSpace", "internal_api.AuthSpace"},
		{"AuthLoginRequest", "internal_api.loginRequest"},
		{"ChangePasswordRequest", "internal_api.changePasswordRequest"},
		{"RepoConnectionItem", "github_com_ash-repwiki_ash_internal_store.RepoConnection"},
		{"CreateRepoConnectionRequest", "internal_api.createRepoConnectionRequest"},
		{"CreateSecretRequest", "internal_api.createSecretRequest"},
		{"RotateSecretRequest", "internal_api.rotateSecretRequest"},
		{"RAGIndexRequest", "github_com_ash-repwiki_ash_internal_rag.IndexRequest"},
		{"RAGQueryRequest", "github_com_ash-repwiki_ash_internal_rag.QueryRequest"},
		{"RAGHit", "github_com_ash-repwiki_ash_internal_rag.Hit"},
		{"RunReplayRequest", "github_com_ash-repwiki_ash_internal_runs.ReplayRequest"},
		{"CheckpointItem", "github_com_ash-repwiki_ash_internal_store.Checkpoint"},
	}

	for _, pair := range pairs {
		pair := pair
		t.Run(pair.contract, func(t *testing.T) {
			assertSchemaParity(t, contractPath, swaggerPath, pair.contract, pair.swagger)
		})
	}
}

func assertSchemaParity(t *testing.T, contractPath, swaggerPath, contractName, swaggerName string) {
	t.Helper()
	contract, err := SchemaPropertyNames(contractPath, contractName)
	if err != nil {
		t.Fatal(err)
	}
	swagger, err := SwaggerDefinitionPropertyNames(swaggerPath, swaggerName)
	if err != nil {
		t.Fatal(err)
	}
	missingInContract, missingInSwagger := DiffPropertyNames(contract, swagger)
	if len(missingInContract) > 0 || len(missingInSwagger) > 0 {
		t.Fatalf("%s drift:\n  contract only: %s\n  swagger only: %s",
			contractName,
			strings.Join(missingInSwagger, ", "),
			strings.Join(missingInContract, ", "))
	}
	if len(contract) == 0 {
		t.Fatalf("%s has no properties", contractName)
	}
}

// FormatSchemaDrift is a helper for error messages in future emit tooling.
func FormatSchemaDrift(contractName string, missingInContract, missingInSwagger []string) string {
	return fmt.Sprintf("%s: contract-only [%s] swagger-only [%s]",
		contractName,
		strings.Join(missingInSwagger, ", "),
		strings.Join(missingInContract, ", "))
}
