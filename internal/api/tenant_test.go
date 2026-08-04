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
	for _, row := range []any{&run, &approval} {
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
		{"stream", http.MethodGet, "/api/v1/runs/" + run.ID + "/stream", ""},
		{"provenance", http.MethodGet, "/api/v1/runs/" + run.ID + "/provenance", ""},
		{"artifacts", http.MethodGet, "/api/v1/runs/" + run.ID + "/artifacts", ""},
		{"timeline", http.MethodGet, "/api/v1/runs/" + run.ID + "/timeline", ""},
		{"approve", http.MethodPost, "/api/v1/approvals/" + approval.ID + "/approve", `{}`},
		{"reject", http.MethodPost, "/api/v1/approvals/" + approval.ID + "/reject", `{"reason":"x"}`},
		{"createRelease", http.MethodPost, "/api/v1/releases", `{"spaceId":"space_other","version":"9.9.9","title":"x"}`},
		{"createRepo", http.MethodPost, "/api/v1/repo/connections", `{"spaceId":"space_other","provider":"github","owner":"o","repo":"r","secretId":"sec_missing"}`},
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
