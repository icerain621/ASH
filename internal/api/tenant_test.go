package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestGetMemoryRecordRejectsCrossSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	mem := store.MemoryRecord{
		ID: "mem_cross_space", Layer: "L1", Status: "approved", SpaceID: "space_other",
		SchemaVersion: memory.CurrentSchemaVersion, Title: "t", Body: "b",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&mem).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/memory/records/"+mem.ID, nil)
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestGetRunRejectsCrossSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_cross_space", TraceID: "trace_cross", ScenarioName: "feature_delivery",
		ScenarioVersion: "1.0.0", PolicyProfile: "default", Status: "finished",
		SpaceID: "space_other", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID, nil)
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestSpaceMembersRejectsCrossSpaceParam(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	other := store.Space{
		ID: "space_members_other", OrgID: "org_x", Name: "Other", Slug: "other",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/"+other.ID+"/members", nil)
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestCreateRunRejectsCrossSpaceBody(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)
	body, _ := json.Marshal(map[string]any{
		"spaceId": "space_other",
		"scenario": map[string]any{
			"name": "feature_delivery", "scenarioVersion": "1.0.0",
		},
		"inputs": map[string]any{"issueOrSpec": "cross space", "repoRoot": "."},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

func TestRotateSecretRejectsCrossSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	row := store.SecretRecord{
		ID: "sec_cross", SpaceID: "space_other", Name: "TOKEN",
		Status: "active", ScopeJSON: "{}", ValueCiphertext: "x", ValueDigest: "d",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"value":"new-secret-value"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/"+row.ID+"/rotate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", "space_home")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
	}
}

// TestCrossSpaceAPIRegression covers high-risk read/write paths for R-08 (越权).
func TestCrossSpaceAPIRegression(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	otherSpace := store.Space{
		ID: "space_other", OrgID: "org_cross_reg", Name: "Other", Slug: "other-reg",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&otherSpace).Error; err != nil {
		t.Fatal(err)
	}
	run := store.RunRecord{
		ID: "run_cross_reg", TraceID: "trace_cross_reg", ScenarioName: "feature_delivery",
		ScenarioVersion: "1.0.0", PolicyProfile: "default", Status: "waiting_approval",
		SpaceID: "space_other", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	approval := store.ApprovalRequest{
		ID: "apr_cross_reg", SpaceID: "space_other", RunID: run.ID, TraceID: run.TraceID,
		StepID: "qa.verify", Gate: "human", Reason: "cross", Status: "pending",
		EvidenceJSON: `{}`, CreatedAt: now, UpdatedAt: now,
	}
	secret := store.SecretRecord{
		ID: "sec_cross_reg", SpaceID: "space_other", Name: "CROSS_REG_TOKEN",
		Status: "active", ScopeJSON: "{}", ValueCiphertext: "x", ValueDigest: "d",
		CreatedAt: now, UpdatedAt: now,
	}
	mem := store.MemoryRecord{
		ID: "mem_cross_reg", Layer: "L1", Status: "candidate", SpaceID: "space_other",
		SchemaVersion: 1, Title: "cross", Body: "body", TagsJSON: "[]", Confidence: 0.9,
		Sensitivity: "normal", CreatedAt: now, UpdatedAt: now,
	}
	diag := store.CIDiagnosis{
		ID: "diag_cross_reg", SpaceID: "space_other", Status: "ready",
		RootCause: "fixture", FixSuggestionsJSON: "[]", EvidenceRefsJSON: "[]",
		DecisionStatus: "pending", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&run, &approval, &secret, &mem, &diag} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"getRun", http.MethodGet, "/api/v1/runs/" + run.ID, ""},
		{"stream", http.MethodGet, "/api/v1/runs/" + run.ID + "/stream", ""},
		{"provenance", http.MethodGet, "/api/v1/runs/" + run.ID + "/provenance", ""},
		{"artifacts", http.MethodGet, "/api/v1/runs/" + run.ID + "/artifacts", ""},
		{"artifactAccess", http.MethodGet, "/api/v1/runs/" + run.ID + "/artifacts/bundle.zip/access", ""},
		{"checkpointAccess", http.MethodGet, "/api/v1/runs/" + run.ID + "/checkpoints/cp_missing/access", ""},
		{"checkpoints", http.MethodGet, "/api/v1/runs/" + run.ID + "/checkpoints", ""},
		{"timeline", http.MethodGet, "/api/v1/runs/" + run.ID + "/timeline", ""},
		{"toolCalls", http.MethodGet, "/api/v1/runs/" + run.ID + "/tool-calls", ""},
		{"agentTasks", http.MethodGet, "/api/v1/runs/" + run.ID + "/agent-tasks", ""},
		{"qualityMetrics", http.MethodGet, "/api/v1/runs/" + run.ID + "/quality-metrics", ""},
		{"obsQuality", http.MethodGet, "/api/v1/observability/quality/" + run.ID, ""},
		{"waterfall", http.MethodGet, "/api/v1/observability/waterfall/" + run.ID, ""},
		{"runCancel", http.MethodPost, "/api/v1/runs/" + run.ID + "/cancel", ""},
		{"runResume", http.MethodPost, "/api/v1/runs/" + run.ID + "/resume", ""},
		{"runReplay", http.MethodPost, "/api/v1/runs/" + run.ID + "/replay", `{"mode":"exact"}`},
		{"runApprove", http.MethodPost, "/api/v1/runs/" + run.ID + "/approve", `{"actorId":"t","reason":"x"}`},
		{"approve", http.MethodPost, "/api/v1/approvals/" + approval.ID + "/approve", `{}`},
		{"reject", http.MethodPost, "/api/v1/approvals/" + approval.ID + "/reject", `{"reason":"x"}`},
		{"createRelease", http.MethodPost, "/api/v1/releases", `{"spaceId":"space_other","version":"9.9.9","title":"x"}`},
		{"createRepo", http.MethodPost, "/api/v1/repo/connections", `{"spaceId":"space_other","provider":"github","owner":"o","repo":"r","secretId":"sec_missing"}`},
		{"createSecret", http.MethodPost, "/api/v1/secrets", `{"spaceId":"space_other","name":"X_TOKEN","value":"secret-value-here"}`},
		{"rotateSecret", http.MethodPost, "/api/v1/secrets/" + secret.ID + "/rotate", `{"value":"new-secret-value"}`},
		{"deleteSecret", http.MethodDelete, "/api/v1/secrets/" + secret.ID, ""},
		{"reviewMemory", http.MethodPost, "/api/v1/memory/candidates/" + mem.ID + "/review", `{"decision":"approve","reason":"x","policyProfile":"default"}`},
		{"getMemory", http.MethodGet, "/api/v1/memory/records/" + mem.ID, ""},
		{"createFeedback", http.MethodPost, "/api/v1/feedback", `{"spaceId":"space_other","targetType":"run","targetId":"` + run.ID + `","rating":1}`},
		{"ragQuery", http.MethodPost, "/api/v1/rag/query", `{"spaceId":"space_other","text":"cross","topK":3}`},
		{"ragIndex", http.MethodPost, "/api/v1/rag/index", `{"spaceId":"space_other","repoRoot":"."}`},
		{"adoptDiagnosis", http.MethodPost, "/api/v1/ci/diagnoses/" + diag.ID + "/adopt", `{"reason":"x"}`},
		{"dismissDiagnosis", http.MethodPost, "/api/v1/ci/diagnoses/" + diag.ID + "/dismiss", `{"reason":"x"}`},
		{"auditExport", http.MethodPost, "/api/v1/audit/export", `{"spaceId":"space_other"}`},
		{"registerPlugin", http.MethodPost, "/api/v1/plugins", `{"spaceId":"space_other","name":"p","version":"1.0.0","endpoint":"http://127.0.0.1:9"}`},
		{"eventsRetention", http.MethodPost, "/api/v1/events/retention/apply", `{"spaceId":"space_other","dryRun":true}`},
		{"artifactsRetention", http.MethodPost, "/api/v1/artifacts/retention/apply", `{"spaceId":"space_other","dryRun":true}`},
		{"spaceMembers", http.MethodGet, "/api/v1/spaces/space_other/members", ""},
		{"spaceScopes", http.MethodGet, "/api/v1/spaces/space_other/resource-scopes", ""},
		{"spaceMatrix", http.MethodGet, "/api/v1/spaces/space_other/permissions/matrix", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tc.body != "" {
				bodyReader = bytes.NewReader([]byte(tc.body))
			} else {
				bodyReader = bytes.NewReader(nil)
			}
			req := httptest.NewRequest(tc.method, tc.path, bodyReader)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("X-ASH-Space-ID", "space_home")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s want 403", w.Code, w.Body.String())
			}
		})
	}
}
