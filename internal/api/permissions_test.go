package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/authz"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestPermissionMatrixForSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_matrix", Name: "Matrix Org", Slug: "matrix-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_matrix", OrgID: org.ID, Name: "Matrix Space", Slug: "matrix", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	if err := authz.SeedScenarioScopes(db, space.ID, now); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/"+space.ID+"/permissions/matrix", nil)
	req.Header.Set("X-ASH-Space-ID", space.ID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp authz.MatrixResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.BuiltinRoles) < 5 {
		t.Fatalf("builtin roles=%d want >=5", len(resp.BuiltinRoles))
	}
	if len(resp.ScenarioTools) < 3 {
		t.Fatalf("scenario tools=%d want >=3", len(resp.ScenarioTools))
	}
}

func TestUpdateSpaceResourceScopePolicy(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := store.Space{ID: "space_scope_update", OrgID: "org_scope", Name: "Scope Space", Slug: "scope", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	if err := authz.SeedScenarioScopes(db, space.ID, now); err != nil {
		t.Fatal(err)
	}
	var scope store.ResourceScope
	if err := db.Where("space_id = ? AND resource_id = ?", space.ID, "feature_delivery@1.0.0").First(&scope).Error; err != nil {
		t.Fatal(err)
	}
	customPolicy := `{"toolMatrix":{"reviewer":{"allow":["git.status","apply_patch"],"deny":[],"denyMode":"block"}}}`
	body, _ := json.Marshal(map[string]string{"policyJson": customPolicy})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/spaces/"+space.ID+"/resource-scopes/"+scope.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ASH-Space-ID", space.ID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	loaded, err := authz.LoadScenarioPolicy(db, space.ID, "feature_delivery", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := authz.EvaluateScenarioTool(loaded, "reviewer", "apply_patch")
	if !ok {
		t.Fatal("updated policy should allow reviewer apply_patch")
	}
	var audits []store.AuditLog
	if err := db.Where("space_id = ? AND event_type = ?", space.ID, "scope.policy_updated").Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 {
		t.Fatalf("audit rows=%d want 1", len(audits))
	}
	if !strings.Contains(audits[0].PayloadJSON, scope.ID) {
		t.Fatalf("audit payload=%s", audits[0].PayloadJSON)
	}
}
