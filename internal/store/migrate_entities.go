package store

import (
	"fmt"

	"github.com/ash-repwiki/ash/internal/store/sqlmigrations"
)

// migrationEntity describes one table participating in sqlite→postgres copy.
type migrationEntity struct {
	table       string
	model       any
	pk          string
	incremental bool
}

// MigrationCatalog returns registered migration table names in copy order.
func MigrationCatalog() []string {
	ents := migrationEntities()
	out := make([]string, len(ents))
	for i, ent := range ents {
		out[i] = ent.table
	}
	return out
}

// MigrationCatalogSize returns the number of tables participating in sqlite→postgres copy.
func MigrationCatalogSize() int {
	return len(migrationEntities())
}

// VerifyMigrationSchema ensures every registered migration table exists in db.
func VerifyMigrationSchema(db *DB) error {
	if db == nil || db.DB == nil {
		return fmt.Errorf("database is nil")
	}
	if db.Dialect() == "postgres" && SQLMigrationsEnabledForDB() {
		if err := sqlmigrations.VerifyApplied(RuntimeDatabaseURL()); err != nil {
			return err
		}
	}
	for _, ent := range migrationEntities() {
		if !db.Migrator().HasTable(ent.model) {
			return fmt.Errorf("missing table %s", ent.table)
		}
	}
	return nil
}

// SQLMigrationsEnabledForDB reports whether versioned SQL migrations are active.
func SQLMigrationsEnabledForDB() bool {
	return sqlmigrations.SQLMigrationsEnabled("postgres")
}

func migrationEntities() []migrationEntity {
	return []migrationEntity{
		{table: "schema_meta", model: &SchemaMeta{}, pk: "key", incremental: true},
		{table: "users", model: &User{}, pk: "id", incremental: true},
		{table: "orgs", model: &Org{}, pk: "id", incremental: true},
		{table: "spaces", model: &Space{}, pk: "id", incremental: true},
		{table: "roles", model: &Role{}, pk: "id", incremental: true},
		{table: "members", model: &Member{}, pk: "id", incremental: true},
		{table: "resource_scopes", model: &ResourceScope{}, pk: "id", incremental: true},
		{table: "audit_policies", model: &AuditPolicy{}, pk: "space_id", incremental: true},
		{table: "runs", model: &RunRecord{}, pk: "id", incremental: true},
		{table: "run_steps", model: &RunStep{}, pk: "id", incremental: true},
		{table: "tool_calls", model: &ToolCall{}, pk: "id", incremental: true},
		{table: "agent_tasks", model: &AgentTask{}, pk: "id", incremental: true},
		{table: "artifact_index", model: &ArtifactIndex{}, pk: "id", incremental: true},
		{table: "checkpoints", model: &Checkpoint{}, pk: "id", incremental: true},
		{table: "run_events", model: &RunEvent{}, pk: "id", incremental: false},
		{table: "memory_records", model: &MemoryRecord{}, pk: "id", incremental: true},
		{table: "memory_evidence", model: &MemoryEvidence{}, pk: "id", incremental: true},
		{table: "memory_reviews", model: &MemoryReview{}, pk: "id", incremental: true},
		{table: "memory_edges", model: &MemoryEdge{}, pk: "id", incremental: true},
		{table: "memory_migrations", model: &MemoryMigration{}, pk: "id", incremental: true},
		{table: "rag_documents", model: &RAGDocument{}, pk: "id", incremental: true},
		{table: "rag_chunks", model: &RAGChunk{}, pk: "id", incremental: true},
		{table: "model_usage", model: &ModelUsage{}, pk: "id", incremental: false},
		{table: "quality_metrics", model: &QualityMetric{}, pk: "id", incremental: true},
		{table: "mcp_tools", model: &MCPTool{}, pk: "id", incremental: true},
		{table: "feedback", model: &Feedback{}, pk: "id", incremental: true},
		{table: "repo_connections", model: &RepoConnection{}, pk: "id", incremental: true},
		{table: "ci_runs", model: &CIRun{}, pk: "id", incremental: true},
		{table: "ci_jobs", model: &CIJob{}, pk: "id", incremental: true},
		{table: "ci_diagnoses", model: &CIDiagnosis{}, pk: "id", incremental: true},
		{table: "alert_rules", model: &AlertRule{}, pk: "id", incremental: true},
		{table: "alert_events", model: &AlertEvent{}, pk: "id", incremental: true},
		{table: "alert_silences", model: &AlertSilence{}, pk: "id", incremental: true},
		{table: "release_records", model: &ReleaseRecord{}, pk: "id", incremental: true},
		{table: "release_checklist_items", model: &ReleaseChecklistItem{}, pk: "id", incremental: true},
		{table: "release_gate_results", model: &ReleaseGateResult{}, pk: "id", incremental: false},
		{table: "rollback_drills", model: &RollbackDrill{}, pk: "id", incremental: true},
		{table: "secret_records", model: &SecretRecord{}, pk: "id", incremental: true},
		{table: "audit_log", model: &AuditLog{}, pk: "id", incremental: false},
		{table: "approval_requests", model: &ApprovalRequest{}, pk: "id", incremental: true},
		{table: "audit_exports", model: &AuditExport{}, pk: "id", incremental: true},
		{table: "plugin_registry", model: &PluginRegistry{}, pk: "id", incremental: true},
		{table: "improve_proposals", model: &ImproveProposal{}, pk: "id", incremental: true},
		{table: "harness_profile_versions", model: &HarnessProfileVersion{}, pk: "id", incremental: true},
		{table: "scenario_patch_drafts", model: &ScenarioPatchDraft{}, pk: "id", incremental: true},
		{table: "goal_plans", model: &GoalPlan{}, pk: "id", incremental: true},
	}
}
