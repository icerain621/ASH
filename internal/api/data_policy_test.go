package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestDataPolicyAndEventsArtifactsRetention(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	t.Setenv("ASH_RETENTION_EVENTS_DAYS", "1")
	t.Setenv("ASH_RETENTION_ARTIFACTS_DAYS", "1")
	t.Setenv("ASH_RETENTION_ARTIFACTS_MAX_RUNS", "100")

	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_data_policy", Name: "Data Policy Org", Slug: "data-policy", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_data_policy_own", OrgID: org.ID, Name: "Own", Slug: "own-dp", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_data_policy_other", OrgID: org.ID, Name: "Other", Slug: "other-dp", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_data_policy", DisplayName: "DP User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_data_policy", OrgID: org.ID, Name: "auditor", Permissions: `["audit:export"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_data_policy", OrgID: org.ID, SpaceID: ownSpace.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	getResp := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/data-policy", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("data-policy status=%d body=%s", getResp.Code, getResp.Body.String())
	}
	var policyResp DataPolicyResponse
	if err := json.Unmarshal(getResp.Body.Bytes(), &policyResp); err != nil {
		t.Fatal(err)
	}
	if policyResp.Policy.EventsDays != 1 || policyResp.Policy.ArtifactsDays != 1 {
		t.Fatalf("policy=%+v want events/artifacts 1", policyResp.Policy)
	}
	if len(policyResp.Classification) != 3 {
		t.Fatalf("classification=%v want 3 levels", policyResp.Classification)
	}

	old := now.AddDate(0, 0, -3)
	ownOld := store.RunRecord{
		ID: "run_dp_old_own", TraceID: "tr_old_own", ScenarioName: "feature_delivery", ScenarioVersion: "0.1.0",
		PolicyProfile: "default", Status: "finished", SpaceID: ownSpace.ID, ActorRole: "maintainer",
		StartedAt: old, CreatedAt: old, UpdatedAt: old,
	}
	ownNew := store.RunRecord{
		ID: "run_dp_new_own", TraceID: "tr_new_own", ScenarioName: "feature_delivery", ScenarioVersion: "0.1.0",
		PolicyProfile: "default", Status: "finished", SpaceID: ownSpace.ID, ActorRole: "maintainer",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	otherOld := store.RunRecord{
		ID: "run_dp_old_other", TraceID: "tr_old_other", ScenarioName: "feature_delivery", ScenarioVersion: "0.1.0",
		PolicyProfile: "default", Status: "finished", SpaceID: otherSpace.ID, ActorRole: "maintainer",
		StartedAt: old, CreatedAt: old, UpdatedAt: old,
	}
	for _, row := range []store.RunRecord{ownOld, ownNew, otherOld} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []store.RunEvent{
		{ID: "ev_old_own", RunID: ownOld.ID, Seq: 1, TS: old.UnixMilli(), Type: "run.started", Severity: "info", PayloadJSON: `{}`, CreatedAt: old},
		{ID: "ev_new_own", RunID: ownNew.ID, Seq: 1, TS: now.UnixMilli(), Type: "run.started", Severity: "info", PayloadJSON: `{}`, CreatedAt: now},
		{ID: "ev_old_other", RunID: otherOld.ID, Seq: 1, TS: old.UnixMilli(), Type: "run.started", Severity: "info", PayloadJSON: `{}`, CreatedAt: old},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []store.ArtifactIndex{
		{ID: "art_old_own", RunID: ownOld.ID, Type: "diff", Name: "diff.patch", URI: "fs://x", Digest: "d1", CreatedAt: old},
		{ID: "art_new_own", RunID: ownNew.ID, Type: "diff", Name: "diff.patch", URI: "fs://y", Digest: "d2", CreatedAt: now},
		{ID: "art_old_other", RunID: otherOld.ID, Type: "diff", Name: "diff.patch", URI: "fs://z", Digest: "d3", CreatedAt: old},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	dryResp := httptest.NewRecorder()
	dryReq := httptest.NewRequest(http.MethodPost, "/api/v1/events/retention/apply", bytes.NewReader([]byte(`{"dryRun":true}`)))
	dryReq.Header.Set("Authorization", "Bearer "+token)
	dryReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(dryResp, dryReq)
	if dryResp.Code != http.StatusOK {
		t.Fatalf("events dry status=%d body=%s", dryResp.Code, dryResp.Body.String())
	}
	var dry DataRetentionApplyResponse
	if err := json.Unmarshal(dryResp.Body.Bytes(), &dry); err != nil {
		t.Fatal(err)
	}
	if dry.Matched != 1 || dry.Deleted != 0 || !dry.DryRun {
		t.Fatalf("events dry=%+v want matched 1", dry)
	}

	applyResp := httptest.NewRecorder()
	applyReq := httptest.NewRequest(http.MethodPost, "/api/v1/events/retention/apply", bytes.NewReader([]byte(`{}`)))
	applyReq.Header.Set("Authorization", "Bearer "+token)
	applyReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(applyResp, applyReq)
	if applyResp.Code != http.StatusOK {
		t.Fatalf("events apply status=%d body=%s", applyResp.Code, applyResp.Body.String())
	}
	var ownOldEv, otherOldEv, ownNewEv int64
	_ = db.Model(&store.RunEvent{}).Where("id = ?", "ev_old_own").Count(&ownOldEv).Error
	_ = db.Model(&store.RunEvent{}).Where("id = ?", "ev_old_other").Count(&otherOldEv).Error
	_ = db.Model(&store.RunEvent{}).Where("id = ?", "ev_new_own").Count(&ownNewEv).Error
	if ownOldEv != 0 || otherOldEv != 1 || ownNewEv != 1 {
		t.Fatalf("events ownOld=%d otherOld=%d ownNew=%d", ownOldEv, otherOldEv, ownNewEv)
	}

	artResp := httptest.NewRecorder()
	artReq := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/retention/apply", bytes.NewReader([]byte(`{}`)))
	artReq.Header.Set("Authorization", "Bearer "+token)
	artReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(artResp, artReq)
	if artResp.Code != http.StatusOK {
		t.Fatalf("artifacts apply status=%d body=%s", artResp.Code, artResp.Body.String())
	}
	var ownOldArt, otherOldArt, ownNewArt int64
	_ = db.Model(&store.ArtifactIndex{}).Where("id = ?", "art_old_own").Count(&ownOldArt).Error
	_ = db.Model(&store.ArtifactIndex{}).Where("id = ?", "art_old_other").Count(&otherOldArt).Error
	_ = db.Model(&store.ArtifactIndex{}).Where("id = ?", "art_new_own").Count(&ownNewArt).Error
	if ownOldArt != 0 || otherOldArt != 1 || ownNewArt != 1 {
		t.Fatalf("artifacts ownOld=%d otherOld=%d ownNew=%d", ownOldArt, otherOldArt, ownNewArt)
	}

	readyz := httptest.NewRecorder()
	readyzReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(readyz, readyzReq)
	if readyz.Code != http.StatusOK {
		t.Fatalf("readyz=%d body=%s", readyz.Code, readyz.Body.String())
	}
	var health HealthResponse
	if err := json.Unmarshal(readyz.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.RetentionEventsDays != 1 || health.RetentionArtifactsDays != 1 {
		t.Fatalf("readyz retention=%+v", health)
	}
}
