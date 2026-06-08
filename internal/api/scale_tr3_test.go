package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestScaleReadiness(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := "space_scale_readiness"
	_ = db.Create(&store.MemoryRecord{
		ID: "mem_scale_1", Layer: "L1", Status: "approved", SpaceID: space,
		SchemaVersion: memory.CurrentSchemaVersion, Title: "t", Body: "b",
		CreatedAt: now, UpdatedAt: now,
	}).Error

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scale/readiness", nil)
	req.Header.Set("X-ASH-Space-ID", space)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp ScaleReadinessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.SpaceID != space {
		t.Fatalf("spaceId=%q want %q", resp.SpaceID, space)
	}
	if resp.MemorySchemaVersion != memory.CurrentSchemaVersion {
		t.Fatalf("schema=%d want %d", resp.MemorySchemaVersion, memory.CurrentSchemaVersion)
	}
	if resp.MemoryApprovedCount < 1 {
		t.Fatalf("memoryApproved=%d want >=1", resp.MemoryApprovedCount)
	}
	if resp.MigrationTableCount < 25 {
		t.Fatalf("migrationTableCount=%d want >=25", resp.MigrationTableCount)
	}
	if resp.DatabaseDialect == "" {
		t.Fatal("databaseDialect is empty")
	}
	if resp.WorkerConnectionRole != "sqlite" {
		t.Fatalf("workerConnectionRole=%q want sqlite", resp.WorkerConnectionRole)
	}
}

func TestScaleReadinessSchemaSqlDualWriteWarning(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	if err := store.SaveDualWriteConfig(db.DataDir(), &store.DualWriteConfig{
		Enabled:     true,
		PostgresURL: "postgres://ash:ash@127.0.0.1:5432/ash_shadow?sslmode=disable",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASH_SCHEMA_MODE", "sql")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scale/readiness", nil)
	req.Header.Set("X-ASH-Space-ID", "local")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp ScaleReadinessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.ReadinessWarnings) == 0 {
		t.Fatalf("expected readinessWarnings, got %+v", resp)
	}
}

func TestScaleReadinessMigrationSyncError(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	errAt := time.Now().UTC().Add(-2 * time.Minute)
	if err := store.SaveSyncState(db.DataDir(), &store.SyncState{
		LastError:   "verify row counts: orgs mismatch",
		LastErrorAt: &errAt,
		UpdatedAt:   errAt,
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scale/readiness", nil)
	req.Header.Set("X-ASH-Space-ID", "local")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp ScaleReadinessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.LastMigrationSyncError != "verify row counts: orgs mismatch" {
		t.Fatalf("lastMigrationSyncError=%q", resp.LastMigrationSyncError)
	}
	if resp.LastMigrationSyncErrorAtMs == nil {
		t.Fatal("expected lastMigrationSyncErrorAtMs")
	}
	if *resp.LastMigrationSyncErrorAtMs != errAt.UnixMilli() {
		t.Fatalf("lastMigrationSyncErrorAtMs=%d want %d", *resp.LastMigrationSyncErrorAtMs, errAt.UnixMilli())
	}
}

func TestRunProvenance(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	runID := "run_prov_test"
	traceID := "trace_prov_test"
	_ = db.Create(&store.RunRecord{
		ID: runID, TraceID: traceID,
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "completed", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error
	_ = db.Create(&store.RunEvent{
		ID: "ev_1", RunID: runID, Seq: 1, TS: now.UnixMilli(),
		Type: "run.started", PayloadJSON: "{}", CreatedAt: now,
	}).Error
	_ = db.Create(&store.ToolCall{
		ID: "tc_1", RunID: runID, StepID: "step1", Tool: "git.status",
		Risk: "low", Status: "success", CreatedAt: now,
	}).Error

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+runID+"/provenance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp ProvenanceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.TraceID != traceID || resp.RunID != runID {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.Events < 1 || resp.ToolCalls < 1 {
		t.Fatalf("events=%d toolCalls=%d want >=1", resp.Events, resp.ToolCalls)
	}
}
