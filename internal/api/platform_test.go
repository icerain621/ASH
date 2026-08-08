package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/observability"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func newPlatformTestRouter(t *testing.T) (*gin.Engine, *store.DB) {
	t.Helper()
	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(db, loader).Register(r, "")
	return r, db
}

func TestRouteModelUsesFallbackAndRecordsUsage(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_MODEL_PRIMARY_STATUS", "unavailable")
	t.Setenv("ASH_MODEL_PRIMARY_MODEL", "primary-model")
	t.Setenv("ASH_MODEL_FALLBACK_STATUS", "available")
	t.Setenv("ASH_MODEL_FALLBACK_MODEL", "fallback-model")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	if err := db.Create(&store.RunRecord{
		ID: "run_test", TraceID: "trace_test",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"runId":"run_test","stepId":"arch.design","prompt":"fallback please","outputTokens":7}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/model-router/route", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Provider struct {
			ID    string `json:"id"`
			Model string `json:"model"`
		} `json:"provider"`
		Status       string `json:"status"`
		FallbackUsed bool   `json:"fallbackUsed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Provider.ID != "fallback" || resp.Provider.Model != "fallback-model" {
		t.Fatalf("provider=%+v want fallback/fallback-model", resp.Provider)
	}
	if !resp.FallbackUsed || resp.Status != "routed" {
		t.Fatalf("fallbackUsed=%v status=%q want true/routed", resp.FallbackUsed, resp.Status)
	}

	var rows []store.ModelUsage
	if err := db.Where("run_id = ? AND step_id = ?", "run_test", "arch.design").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("usage rows=%d want 1", len(rows))
	}
	if rows[0].Provider != "fallback" || rows[0].Model != "fallback-model" {
		t.Fatalf("usage provider/model=%s/%s want fallback/fallback-model", rows[0].Provider, rows[0].Model)
	}
}

func TestGetQualityMetricsRequiresRunAccessAndReturnsMetrics(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_quality", TraceID: "trace_quality",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.QualityMetric{
		ID: "qm_quality", RunID: run.ID, SpaceID: "local",
		Name: "tool_failure_rate", Value: 0.25, Unit: "ratio", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/observability/quality/"+run.ID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Items []store.QualityMetric `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Name != "tool_failure_rate" {
		t.Fatalf("items=%+v want tool_failure_rate", resp.Items)
	}
}

func TestGetWaterfallReturnsStructuredSpans(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	done := now.Add(time.Second)
	run := store.RunRecord{
		ID: "run_waterfall_api", TraceID: "trace_waterfall_api",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local",
		StartedAt: now, FinishedAt: &done, CreatedAt: now, UpdatedAt: done,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.RunStep{
		ID: "step_waterfall_api", RunID: run.ID, StepID: "arch.design", StepOrder: 1,
		Role: "Architect", Kind: "llm", Status: "finished",
		StartedAt: &now, FinishedAt: &done, DurationMs: 1000,
		CreatedAt: now, UpdatedAt: done,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.QualityMetric{
		ID: "qm_waterfall_api", RunID: run.ID, SpaceID: "local",
		Name: "steps_total", Value: 1, Unit: "count", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/observability/waterfall/"+run.ID, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp observability.Waterfall
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.RunID != run.ID || len(resp.Spans) < 2 {
		t.Fatalf("waterfall=%+v want run and step spans", resp)
	}
	if len(resp.Metrics) != 1 || resp.Metrics[0].Name != "steps_total" {
		t.Fatalf("metrics=%+v want steps_total", resp.Metrics)
	}
}

func TestCreateOrgAndSpaceProvisionGovernanceRows(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)

	orgBody := []byte(`{"name":"Product Team","slug":"product"}`)
	orgResp := httptest.NewRecorder()
	orgReq := httptest.NewRequest(http.MethodPost, "/api/v1/orgs", bytes.NewReader(orgBody))
	orgReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(orgResp, orgReq)
	if orgResp.Code != http.StatusCreated {
		t.Fatalf("org status=%d want %d body=%s", orgResp.Code, http.StatusCreated, orgResp.Body.String())
	}
	var org store.Org
	if err := json.Unmarshal(orgResp.Body.Bytes(), &org); err != nil {
		t.Fatal(err)
	}
	if org.ID == "" || org.Slug != "product" {
		t.Fatalf("org=%+v want id/product slug", org)
	}

	var roles []store.Role
	if err := db.Where("org_id = ? AND name = ?", org.ID, "admin").Find(&roles).Error; err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 {
		t.Fatalf("roles=%d want 1", len(roles))
	}
	var members []store.Member
	if err := db.Where("org_id = ? AND user_id = ?", org.ID, "dev-user").Find(&members).Error; err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("members=%d want 1", len(members))
	}

	spaceBody := []byte(`{"orgId":"` + org.ID + `","name":"Delivery Space","slug":"delivery"}`)
	spaceResp := httptest.NewRecorder()
	spaceReq := httptest.NewRequest(http.MethodPost, "/api/v1/spaces", bytes.NewReader(spaceBody))
	spaceReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(spaceResp, spaceReq)
	if spaceResp.Code != http.StatusCreated {
		t.Fatalf("space status=%d want %d body=%s", spaceResp.Code, http.StatusCreated, spaceResp.Body.String())
	}
	var space store.Space
	if err := json.Unmarshal(spaceResp.Body.Bytes(), &space); err != nil {
		t.Fatal(err)
	}
	if space.OrgID != org.ID || space.Slug != "delivery" {
		t.Fatalf("space=%+v want org/delivery", space)
	}

	var scopes []store.ResourceScope
	if err := db.Where("space_id = ?", space.ID).Find(&scopes).Error; err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 4 {
		t.Fatalf("resource scopes=%d want 4 (space + 3 scenarios)", len(scopes))
	}
	var audits []store.AuditLog
	if err := db.Where("event_type IN ?", []string{"org.created", "space.created"}).Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 2 {
		t.Fatalf("audit rows=%d want 2", len(audits))
	}
}

func TestDevLoginCanIssueTokenForSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_dev_login", Name: "Dev Org", Slug: "dev-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_dev_login", OrgID: org.ID, Name: "Dev Space", Slug: "dev", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"spaceId":"` + space.ID + `"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/dev-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
		Space struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"space"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Space.ID != space.ID || resp.Space.Name != space.Name {
		t.Fatalf("space=%+v want %s/%s", resp.Space, space.ID, space.Name)
	}
	claims, err := verifyToken(resp.Token, "dev-secret-change-me")
	if err != nil {
		t.Fatal(err)
	}
	if claims.SpaceID != space.ID || claims.Role != "admin" {
		t.Fatalf("claims=%+v want space admin", claims)
	}
}

func TestRegisterMCPToolUsesMemberPermission(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_mcp_perm", Name: "MCP Org", Slug: "mcp-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_mcp_perm", OrgID: org.ID, Name: "MCP Space", Slug: "mcp-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_mcp_perm", DisplayName: "MCP User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_mcp_perm", OrgID: org.ID, Name: "tool-maintainer", Permissions: `["mcp:write"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_mcp_perm", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"name":"repo.scan","server":"http://127.0.0.1:7331","spaceId":"` + space.ID + `"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/tools", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	var tool store.MCPTool
	if err := json.Unmarshal(w.Body.Bytes(), &tool); err != nil {
		t.Fatal(err)
	}
	if tool.SpaceID != space.ID || tool.Name != "repo.scan" {
		t.Fatalf("tool=%+v want target space repo.scan", tool)
	}
}

func TestCreateAuditExportUsesOrgMemberPermission(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_audit_perm", Name: "Audit Org", Slug: "audit-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_audit_perm", OrgID: org.ID, Name: "Audit Space", Slug: "audit-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_audit_perm", DisplayName: "Audit User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_audit_perm", OrgID: org.ID, Name: "auditor", Permissions: `["audit:export"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_audit_perm", OrgID: org.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AuditLog{
		ID: "aud_audit_perm", SpaceID: space.ID, ActorID: user.ID,
		EventType: "test.audit", PayloadJSON: `{"ok":true}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"spaceId":"` + space.ID + `"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/export", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusAccepted, w.Body.String())
	}
	var export store.AuditExport
	if err := json.Unmarshal(w.Body.Bytes(), &export); err != nil {
		t.Fatal(err)
	}
	if export.SpaceID != space.ID || export.Status != "completed" || export.URI == "" {
		t.Fatalf("export=%+v want completed target-space export", export)
	}
	if !strings.HasPrefix(export.URI, "fs://") || export.StoreKey == "" || export.Digest == "" || export.SizeBytes == 0 {
		t.Fatalf("export storage=%+v want fs uri, store key, digest and size", export)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit/exports/"+export.ID+"/access?ttlSeconds=60", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("access status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var access AuditExportAccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &access); err != nil {
		t.Fatal(err)
	}
	if access.ExportID != export.ID || access.Digest != export.Digest || access.SignedURL == "" || access.ExpiresAt <= now.UnixMilli() {
		t.Fatalf("access=%+v want signed export access", access)
	}
}

func TestListAuditLogsFiltersCurrentSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_audit_list", Name: "Audit List Org", Slug: "audit-list", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_audit_list_own", OrgID: org.ID, Name: "Own Audit", Slug: "own-audit", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_audit_list_other", OrgID: org.ID, Name: "Other Audit", Slug: "other-audit", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_audit_list", DisplayName: "Audit User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_audit_list", OrgID: org.ID, Name: "auditor", Permissions: `["audit:export"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_audit_list", OrgID: org.ID, SpaceID: ownSpace.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&store.AuditLog{
		ID: "aud_list_own", SpaceID: ownSpace.ID, EventType: "run.started",
		PayloadJSON: `{"message":"needle own"}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.AuditLog{
		ID: "aud_list_other", SpaceID: otherSpace.ID, EventType: "run.started",
		PayloadJSON: `{"message":"needle other"}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/logs?q=needle&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Items []store.AuditLog `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != "aud_list_own" {
		t.Fatalf("items=%+v want only own audit", resp.Items)
	}
}

func TestAuditPolicyAndRetentionApplyAreSpaceScoped(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_audit_policy", Name: "Audit Policy Org", Slug: "audit-policy", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_audit_policy_own", OrgID: org.ID, Name: "Own Policy", Slug: "own-policy", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_audit_policy_other", OrgID: org.ID, Name: "Other Policy", Slug: "other-policy", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_audit_policy", DisplayName: "Audit Policy User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_audit_policy", OrgID: org.ID, Name: "auditor", Permissions: `["audit:export"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_audit_policy", OrgID: org.ID, SpaceID: ownSpace.ID, UserID: user.ID, RoleID: role.ID,
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
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/audit/policy", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status=%d want %d body=%s", getResp.Code, http.StatusOK, getResp.Body.String())
	}
	var policy store.AuditPolicy
	if err := json.Unmarshal(getResp.Body.Bytes(), &policy); err != nil {
		t.Fatal(err)
	}
	if policy.SpaceID != ownSpace.ID || policy.RetentionDays != 365 {
		t.Fatalf("policy=%+v want default own 365", policy)
	}

	updateResp := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/audit/policy", bytes.NewReader([]byte(`{"retentionDays":1,"redactPayload":true}`)))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update status=%d want %d body=%s", updateResp.Code, http.StatusOK, updateResp.Body.String())
	}
	if err := json.Unmarshal(updateResp.Body.Bytes(), &policy); err != nil {
		t.Fatal(err)
	}
	if policy.RetentionDays != 1 || !policy.RedactPayload {
		t.Fatalf("policy=%+v want retention 1 redact", policy)
	}

	old := now.AddDate(0, 0, -3)
	for _, row := range []store.AuditLog{
		{ID: "aud_retention_old_own", SpaceID: ownSpace.ID, EventType: "old.own", PayloadJSON: `{}`, CreatedAt: old},
		{ID: "aud_retention_new_own", SpaceID: ownSpace.ID, EventType: "new.own", PayloadJSON: `{}`, CreatedAt: now},
		{ID: "aud_retention_old_other", SpaceID: otherSpace.ID, EventType: "old.other", PayloadJSON: `{}`, CreatedAt: old},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	dryResp := httptest.NewRecorder()
	dryReq := httptest.NewRequest(http.MethodPost, "/api/v1/audit/retention/apply", bytes.NewReader([]byte(`{"dryRun":true}`)))
	dryReq.Header.Set("Authorization", "Bearer "+token)
	dryReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(dryResp, dryReq)
	if dryResp.Code != http.StatusOK {
		t.Fatalf("dry status=%d want %d body=%s", dryResp.Code, http.StatusOK, dryResp.Body.String())
	}
	var dry struct {
		Matched int64 `json:"matched"`
		Deleted int64 `json:"deleted"`
		DryRun  bool  `json:"dryRun"`
	}
	if err := json.Unmarshal(dryResp.Body.Bytes(), &dry); err != nil {
		t.Fatal(err)
	}
	if dry.Matched != 1 || dry.Deleted != 0 || !dry.DryRun {
		t.Fatalf("dry=%+v want matched 1 deleted 0 dryRun", dry)
	}

	applyResp := httptest.NewRecorder()
	applyReq := httptest.NewRequest(http.MethodPost, "/api/v1/audit/retention/apply", bytes.NewReader([]byte(`{}`)))
	applyReq.Header.Set("Authorization", "Bearer "+token)
	applyReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(applyResp, applyReq)
	if applyResp.Code != http.StatusOK {
		t.Fatalf("apply status=%d want %d body=%s", applyResp.Code, http.StatusOK, applyResp.Body.String())
	}
	var apply struct {
		Matched int64 `json:"matched"`
		Deleted int64 `json:"deleted"`
	}
	if err := json.Unmarshal(applyResp.Body.Bytes(), &apply); err != nil {
		t.Fatal(err)
	}
	if apply.Matched != 1 || apply.Deleted != 1 {
		t.Fatalf("apply=%+v want matched/deleted 1", apply)
	}
	var ownOldCount int64
	if err := db.Model(&store.AuditLog{}).Where("id = ?", "aud_retention_old_own").Count(&ownOldCount).Error; err != nil {
		t.Fatal(err)
	}
	var otherOldCount int64
	if err := db.Model(&store.AuditLog{}).Where("id = ?", "aud_retention_old_other").Count(&otherOldCount).Error; err != nil {
		t.Fatal(err)
	}
	if ownOldCount != 0 || otherOldCount != 1 {
		t.Fatalf("ownOld=%d otherOld=%d want deleted own only", ownOldCount, otherOldCount)
	}
}

func TestRegisterPluginValidatesABIAndListsCurrentSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_plugin", Name: "Plugin Org", Slug: "plugin-org", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_plugin_own", OrgID: org.ID, Name: "Own Plugin", Slug: "own-plugin", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_plugin_other", OrgID: org.ID, Name: "Other Plugin", Slug: "other-plugin", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_plugin", DisplayName: "Plugin User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_plugin", OrgID: org.ID, Name: "plugin-maintainer", Permissions: `["plugin:*"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_plugin", OrgID: org.ID, SpaceID: ownSpace.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	otherPlugin := store.PluginRegistry{
		ID: "plg_other", SpaceID: otherSpace.ID, Name: "other", Version: "1.0.0",
		Protocol: "grpc", ABI: "ash.plugin.v1", Endpoint: "127.0.0.1:7000",
		Capabilities: `["observability.export"]`, Compatible: true, Status: "verified",
		CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user, &role, &member, &otherPlugin} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	registerBody := []byte(`{"name":"otel-exporter","version":"1.0.0","protocol":"grpc","abi":"ash.plugin.v1","endpoint":"127.0.0.1:7443","capabilities":["observability.export"]}`)
	registerResp := httptest.NewRecorder()
	registerReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins", bytes.NewReader(registerBody))
	registerReq.Header.Set("Authorization", "Bearer "+token)
	registerReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusCreated {
		t.Fatalf("register status=%d want %d body=%s", registerResp.Code, http.StatusCreated, registerResp.Body.String())
	}
	var plugin store.PluginRegistry
	if err := json.Unmarshal(registerResp.Body.Bytes(), &plugin); err != nil {
		t.Fatal(err)
	}
	if plugin.SpaceID != ownSpace.ID || !plugin.Compatible || plugin.Status != "registered" || plugin.ABI != "ash.plugin.v1" {
		t.Fatalf("plugin=%+v want compatible own registered ash.plugin.v1", plugin)
	}

	verifyResp := httptest.NewRecorder()
	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/"+plugin.ID+"/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(verifyResp, verifyReq)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("verify status=%d want %d body=%s", verifyResp.Code, http.StatusOK, verifyResp.Body.String())
	}
	if err := json.Unmarshal(verifyResp.Body.Bytes(), &plugin); err != nil {
		t.Fatal(err)
	}
	if plugin.Status != "verified" || !plugin.Compatible {
		t.Fatalf("plugin=%+v want verified compatible", plugin)
	}

	badBody := []byte(`{"name":"bad-plugin","version":"1.0.0","protocol":"grpc","abi":"ash.plugin.v2","endpoint":"127.0.0.1:7444"}`)
	badResp := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins", bytes.NewReader(badBody))
	badReq.Header.Set("Authorization", "Bearer "+token)
	badReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusCreated {
		t.Fatalf("bad status=%d want %d body=%s", badResp.Code, http.StatusCreated, badResp.Body.String())
	}
	var badPlugin store.PluginRegistry
	if err := json.Unmarshal(badResp.Body.Bytes(), &badPlugin); err != nil {
		t.Fatal(err)
	}
	if badPlugin.Compatible || badPlugin.Status != "incompatible" || badPlugin.LastError == "" {
		t.Fatalf("bad plugin=%+v want incompatible with last error", badPlugin)
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d want %d body=%s", listResp.Code, http.StatusOK, listResp.Body.String())
	}
	var list struct {
		Items []store.PluginRegistry `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	for _, item := range list.Items {
		if item.SpaceID != ownSpace.ID {
			t.Fatalf("listed cross-space plugin=%+v", item)
		}
	}
	if len(list.Items) != 2 {
		t.Fatalf("plugins=%+v want own compatible and incompatible entries", list.Items)
	}

	reportBody := []byte(`{"ok":false,"dropped":3}`)
	reportResp := httptest.NewRecorder()
	reportReq := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/"+plugin.ID+"/export-report", bytes.NewReader(reportBody))
	reportReq.Header.Set("Authorization", "Bearer "+token)
	reportReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(reportResp, reportReq)
	if reportResp.Code != http.StatusOK {
		t.Fatalf("export-report status=%d want %d body=%s", reportResp.Code, http.StatusOK, reportResp.Body.String())
	}
	if err := json.Unmarshal(reportResp.Body.Bytes(), &plugin); err != nil {
		t.Fatal(err)
	}
	if plugin.ExportErrors != 1 || plugin.DropCount != 3 || plugin.LastExportAt == nil {
		t.Fatalf("plugin=%+v want exportErrors=1 dropCount=3 lastExportAt set", plugin)
	}

	healthResp := httptest.NewRecorder()
	healthReq := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/health", nil)
	healthReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(healthResp, healthReq)
	if healthResp.Code != http.StatusOK {
		t.Fatalf("health status=%d want %d body=%s", healthResp.Code, http.StatusOK, healthResp.Body.String())
	}
	var health struct {
		PluginCount       int                    `json:"pluginCount"`
		ExportErrorsTotal int64                  `json:"exportErrorsTotal"`
		DropCountTotal    int64                  `json:"dropCountTotal"`
		Items             []store.PluginRegistry `json:"items"`
	}
	if err := json.Unmarshal(healthResp.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.PluginCount != 2 || health.ExportErrorsTotal != 1 || health.DropCountTotal != 3 {
		t.Fatalf("health=%+v want pluginCount=2 exportErrors=1 dropCount=3", health)
	}
	var auditCount int64
	if err := db.Model(&store.AuditLog{}).
		Where("space_id = ? AND event_type = ?", ownSpace.ID, "plugin.export_failed").
		Count(&auditCount).Error; err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("audit export_failed=%d want 1", auditCount)
	}
}

func TestPluginABIProfileReportsProtoDigests(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	t.Setenv("ASH_PLUGIN_GRPC_ADDR", "127.0.0.1:19090")
	r, _ := newPlatformTestRouter(t)
	token, err := signToken(tokenClaims{Sub: "plugin-maintainer", SpaceID: "local", Role: "maintainer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/abi", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp PluginABIProfileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.CurrentABI != "ash.plugin.v1" || resp.ProtoPackage != "ash.v1" {
		t.Fatalf("resp=%+v want current plugin ABI profile", resp)
	}
	if !resp.GRPCEnabled || resp.PluginGRPCAddr != "127.0.0.1:19090" {
		t.Fatalf("resp=%+v want configured plugin gRPC listener metadata", resp)
	}
	if len(resp.ProtoFiles) < 2 {
		t.Fatalf("proto files=%+v want common and plugin registry protos", resp.ProtoFiles)
	}
	byPath := map[string]PluginProtoFile{}
	for _, file := range resp.ProtoFiles {
		byPath[file.Path] = file
		if !strings.HasPrefix(file.Digest, "sha256:") || file.Bytes == 0 {
			t.Fatalf("file=%+v want digest and bytes", file)
		}
	}
	if byPath["proto/ash/v1/plugin_registry.proto"].Digest == "" || byPath["proto/ash/v1/common.proto"].Digest == "" {
		t.Fatalf("proto files=%+v missing required ABI files", resp.ProtoFiles)
	}
}

func TestPluginCompatibilityUsesSharedABIContract(t *testing.T) {
	if ok, reason := pluginCompatibility("grpc", "", "tool-runner", "1.0.0", "127.0.0.1:19090"); !ok || reason != "" {
		t.Fatalf("compatible=%v reason=%q want shared ABI default to pass", ok, reason)
	}
	if ok, reason := pluginCompatibility("grpc", "", "", "1.0.0", "127.0.0.1:19090"); ok || !strings.Contains(reason, "name and version") {
		t.Fatalf("compatible=%v reason=%q want shared name/version failure", ok, reason)
	}
	if ok, reason := pluginCompatibility("grpc", "", "tool-runner", "1.0.0", ""); ok || reason != "endpoint is required" {
		t.Fatalf("compatible=%v reason=%q want HTTP endpoint failure", ok, reason)
	}
}

func TestStorageProfileReportsDatabaseAndArtifactStore(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	t.Setenv("ASH_ARTIFACT_STORE", "filesystem")
	r, _ := newPlatformTestRouter(t)
	token, err := signToken(tokenClaims{Sub: "storage-admin", SpaceID: "local", Role: "admin"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp StorageProfileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Database.Dialect != "sqlite" || resp.Database.URLConfigured {
		t.Fatalf("database=%+v want sqlite without url", resp.Database)
	}
	if resp.ArtifactStore.Kind != "fs" || !resp.ArtifactStore.Ready {
		t.Fatalf("artifact store=%+v want ready fs", resp.ArtifactStore)
	}
	if resp.ArtifactPaths.RunsRoot == "" || resp.ArtifactPaths.DirPerm != "0755" {
		t.Fatalf("artifactPaths=%+v want runsRoot and 0755", resp.ArtifactPaths)
	}

	t.Setenv("ASH_ARTIFACT_STORE", "s3-compatible")
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/storage/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("s3 status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ArtifactStore.Kind != "s3-compatible" || resp.ArtifactStore.Ready || !resp.ArtifactStore.ObjectStore {
		t.Fatalf("artifact store=%+v want not-ready object profile", resp.ArtifactStore)
	}
}

func TestApprovalQueueIsSpaceScopedAndRejectCancelsRun(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_approval_list", Name: "Approval Org", Slug: "approval-org", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_approval_own", OrgID: org.ID, Name: "Own Approval", Slug: "own-approval", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_approval_other", OrgID: org.ID, Name: "Other Approval", Slug: "other-approval", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_approval", DisplayName: "Approval User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_approval", OrgID: org.ID, Name: "reviewer", Permissions: `["run:approve"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_approval", OrgID: org.ID, SpaceID: ownSpace.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	ownRun := store.RunRecord{
		ID: "run_approval_own", TraceID: "trace_approval_own",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "waiting_approval", SpaceID: ownSpace.ID,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	otherRun := store.RunRecord{
		ID: "run_approval_other", TraceID: "trace_approval_other",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "waiting_approval", SpaceID: otherSpace.ID,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	ownApproval := store.ApprovalRequest{
		ID: "apr_approval_own", SpaceID: ownSpace.ID, RunID: ownRun.ID, TraceID: ownRun.TraceID,
		StepID: "qa.verify", Gate: "human", Reason: "needs review", Status: "pending",
		EvidenceJSON: `{}`, CreatedAt: now, UpdatedAt: now,
	}
	otherApproval := store.ApprovalRequest{
		ID: "apr_approval_other", SpaceID: otherSpace.ID, RunID: otherRun.ID, TraceID: otherRun.TraceID,
		StepID: "qa.verify", Gate: "human", Reason: "other review", Status: "pending",
		EvidenceJSON: `{}`, CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user, &role, &member, &ownRun, &otherRun, &ownApproval, &otherApproval} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/approvals?status=pending", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d want %d body=%s", listResp.Code, http.StatusOK, listResp.Body.String())
	}
	var list struct {
		Items []store.ApprovalRequest `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != ownApproval.ID {
		t.Fatalf("approvals=%+v want only own approval", list.Items)
	}

	rejectBody := []byte(`{"reason":"not safe"}`)
	rejectResp := httptest.NewRecorder()
	rejectReq := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+ownApproval.ID+"/reject", bytes.NewReader(rejectBody))
	rejectReq.Header.Set("Authorization", "Bearer "+token)
	rejectReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rejectResp, rejectReq)
	if rejectResp.Code != http.StatusOK {
		t.Fatalf("reject status=%d want %d body=%s", rejectResp.Code, http.StatusOK, rejectResp.Body.String())
	}
	var updatedApproval store.ApprovalRequest
	if err := db.First(&updatedApproval, "id = ?", ownApproval.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedApproval.Status != "rejected" || updatedApproval.DecisionReason != "not safe" {
		t.Fatalf("approval=%+v want rejected/not safe", updatedApproval)
	}
	var updatedRun store.RunRecord
	if err := db.First(&updatedRun, "id = ?", ownRun.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedRun.Status != "canceled" {
		t.Fatalf("run status=%q want canceled", updatedRun.Status)
	}
}

func TestCancelRunUsesMemberPermission(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_run_perm", Name: "Run Org", Slug: "run-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_run_perm", OrgID: org.ID, Name: "Run Space", Slug: "run-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_run_perm", DisplayName: "Run User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_run_perm", OrgID: org.ID, Name: "operator", Permissions: `["run:cancel"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_run_perm", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	run := store.RunRecord{
		ID: "run_cancel_perm", TraceID: "trace_cancel_perm",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: space.ID,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member, &run} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		RunID  string `json:"runId"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.RunID != run.ID || resp.Status != "canceled" {
		t.Fatalf("resp=%+v want canceled run", resp)
	}
}

func TestReviewMemoryCandidateUsesMemberPermission(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_mem_perm", Name: "Memory Org", Slug: "memory-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_mem_perm", OrgID: org.ID, Name: "Memory Space", Slug: "memory-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_mem_perm", DisplayName: "Memory User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_mem_perm", OrgID: org.ID, Name: "memory-reviewer", Permissions: `["memory:review"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_mem_perm", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	candidate := store.MemoryRecord{
		ID: "mem_candidate_perm", Layer: "L0", Status: "candidate", SpaceID: space.ID,
		SchemaVersion: 1, Title: "Review me", Body: "memory body", TagsJSON: "[]",
		Sensitivity: "normal", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member, &candidate} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"decision":"approve","reason":"governed","policyProfile":"default"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/candidates/"+candidate.ID+"/review", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		OK     bool   `json:"ok"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK || resp.Status != "approved" {
		t.Fatalf("resp=%+v want approved", resp)
	}
}

func TestCreateSpaceUsesOrgMemberPermission(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_space_perm", Name: "Space Org", Slug: "space-org", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_space_perm", DisplayName: "Space User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_space_perm", OrgID: org.ID, Name: "space-admin", Permissions: `["space:write"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_space_perm", OrgID: org.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &user, &role, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: "local", Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"orgId":"` + org.ID + `","name":"Scoped Delivery","slug":"scoped-delivery"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	var space store.Space
	if err := json.Unmarshal(w.Body.Bytes(), &space); err != nil {
		t.Fatal(err)
	}
	if space.OrgID != org.ID || space.Slug != "scoped-delivery" {
		t.Fatalf("space=%+v want org scoped-delivery", space)
	}
}

func TestRoleAndSpaceMemberManagementUseScopedPermissions(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_member_api", Name: "Member API Org", Slug: "member-api", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_member_api", OrgID: org.ID, Name: "Member API Space", Slug: "member-api-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_member_api_admin", DisplayName: "Member Admin", Status: "active", CreatedAt: now, UpdatedAt: now}
	adminRole := store.Role{
		ID: "role_member_api_admin", OrgID: org.ID, Name: "iam-admin",
		Permissions: `["role:read","role:write","member:read","member:write"]`,
		CreatedAt:   now, UpdatedAt: now,
	}
	adminMember := store.Member{
		ID: "mem_member_api_admin", OrgID: org.ID, UserID: user.ID, RoleID: adminRole.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &adminRole, &adminMember} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	roleBody := []byte(`{"name":"delivery-runner","permissions":["run:create","artifact:read","run:create"]}`)
	roleResp := httptest.NewRecorder()
	roleReq := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+org.ID+"/roles", bytes.NewReader(roleBody))
	roleReq.Header.Set("Authorization", "Bearer "+token)
	roleReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(roleResp, roleReq)
	if roleResp.Code != http.StatusCreated {
		t.Fatalf("role status=%d want %d body=%s", roleResp.Code, http.StatusCreated, roleResp.Body.String())
	}
	var role store.Role
	if err := json.Unmarshal(roleResp.Body.Bytes(), &role); err != nil {
		t.Fatal(err)
	}
	if role.OrgID != org.ID || role.Name != "delivery-runner" || !strings.Contains(role.Permissions, "artifact:read") {
		t.Fatalf("role=%+v want org delivery-runner permissions", role)
	}
	if strings.Count(role.Permissions, "run:create") != 1 {
		t.Fatalf("permissions=%s want deduped run:create", role.Permissions)
	}

	listRoles := httptest.NewRecorder()
	listRolesReq := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+org.ID+"/roles", nil)
	listRolesReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(listRoles, listRolesReq)
	if listRoles.Code != http.StatusOK {
		t.Fatalf("list roles status=%d want %d body=%s", listRoles.Code, http.StatusOK, listRoles.Body.String())
	}
	var roleList RoleListResponse
	if err := json.Unmarshal(listRoles.Body.Bytes(), &roleList); err != nil {
		t.Fatal(err)
	}
	if len(roleList.Items) != 2 {
		t.Fatalf("roles=%+v want admin plus created role", roleList.Items)
	}

	memberBody := []byte(`{"userId":"user_delivery_member","email":"delivery@example.com","displayName":"Delivery Member","password":"temporary-password","roleId":"` + role.ID + `"}`)
	memberResp := httptest.NewRecorder()
	memberReq := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/"+space.ID+"/members", bytes.NewReader(memberBody))
	memberReq.Header.Set("Authorization", "Bearer "+token)
	memberReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(memberResp, memberReq)
	if memberResp.Code != http.StatusCreated {
		t.Fatalf("member status=%d want %d body=%s", memberResp.Code, http.StatusCreated, memberResp.Body.String())
	}
	var member store.Member
	if err := json.Unmarshal(memberResp.Body.Bytes(), &member); err != nil {
		t.Fatal(err)
	}
	if member.SpaceID != space.ID || member.OrgID != org.ID || member.UserID != "user_delivery_member" || member.RoleID != role.ID {
		t.Fatalf("member=%+v want target user role in space", member)
	}
	var createdUser store.User
	if err := db.First(&createdUser, "id = ?", member.UserID).Error; err != nil {
		t.Fatal(err)
	}
	if createdUser.PasswordHash == "" || createdUser.PasswordHash == "temporary-password" {
		t.Fatalf("password hash was not stored securely: %q", createdUser.PasswordHash)
	}

	listMembers := httptest.NewRecorder()
	listMembersReq := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/"+space.ID+"/members", nil)
	listMembersReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(listMembers, listMembersReq)
	if listMembers.Code != http.StatusOK {
		t.Fatalf("list members status=%d want %d body=%s", listMembers.Code, http.StatusOK, listMembers.Body.String())
	}
	var memberList MemberListResponse
	if err := json.Unmarshal(listMembers.Body.Bytes(), &memberList); err != nil {
		t.Fatal(err)
	}
	if len(memberList.Items) != 1 || memberList.Items[0].ID != member.ID {
		t.Fatalf("members=%+v want created space member only", memberList.Items)
	}

	loginBody := []byte(`{"email":"delivery@example.com","password":"temporary-password","spaceId":"` + space.ID + `"}`)
	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(login, loginReq)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d want %d body=%s", login.Code, http.StatusOK, login.Body.String())
	}
	claims, err := verifyToken(extractTokenForTest(t, login.Body.String()), "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sub != member.UserID || claims.SpaceID != space.ID {
		t.Fatalf("claims=%+v want member scoped login", claims)
	}

	var audits []store.AuditLog
	if err := db.Where("space_id = ? AND event_type IN ?", space.ID, []string{"role.created", "member.added"}).Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 2 {
		t.Fatalf("audits=%d want role.created and member.added", len(audits))
	}
}

func TestCreateSpaceMemberRejectsCrossOrgRole(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	ownOrg := store.Org{ID: "org_member_own", Name: "Own Org", Slug: "member-own", CreatedAt: now, UpdatedAt: now}
	otherOrg := store.Org{ID: "org_member_other", Name: "Other Org", Slug: "member-other", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_member_own", OrgID: ownOrg.ID, Name: "Own Space", Slug: "member-own-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_member_cross", DisplayName: "Member Cross", Status: "active", CreatedAt: now, UpdatedAt: now}
	ownRole := store.Role{
		ID: "role_member_cross_admin", OrgID: ownOrg.ID, Name: "member-admin",
		Permissions: `["member:write"]`, CreatedAt: now, UpdatedAt: now,
	}
	otherRole := store.Role{
		ID: "role_member_cross_other", OrgID: otherOrg.ID, Name: "other-runner",
		Permissions: `["run:create"]`, CreatedAt: now, UpdatedAt: now,
	}
	member := store.Member{
		ID: "mem_member_cross", OrgID: ownOrg.ID, UserID: user.ID, RoleID: ownRole.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&ownOrg, &otherOrg, &space, &user, &ownRole, &otherRole, &member} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"userId":"user_bad_member","roleId":"` + otherRole.ID + `"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/"+space.ID+"/members", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ROLE_SCOPE_MISMATCH") {
		t.Fatalf("body=%s want ROLE_SCOPE_MISMATCH", w.Body.String())
	}
}

func TestRouteModelUsesRunSpaceMemberPermission(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	t.Setenv("ASH_MODEL_PRIMARY_STATUS", "available")
	t.Setenv("ASH_MODEL_PRIMARY_MODEL", "primary-model")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_model_perm", Name: "Model Org", Slug: "model-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_model_perm", OrgID: org.ID, Name: "Model Space", Slug: "model-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_model_perm", DisplayName: "Model User", Status: "active", CreatedAt: now, UpdatedAt: now}
	role := store.Role{ID: "role_model_perm", OrgID: org.ID, Name: "model-runner", Permissions: `["model:route"]`, CreatedAt: now, UpdatedAt: now}
	member := store.Member{
		ID: "mem_model_perm", OrgID: org.ID, SpaceID: space.ID, UserID: user.ID, RoleID: role.ID,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	run := store.RunRecord{
		ID: "run_model_perm", TraceID: "trace_model_perm",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: space.ID,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &role, &member, &run} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"runId":"` + run.ID + `","stepId":"pm.plan","prompt":"route this","outputTokens":3}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/model-router/route", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var rows []store.ModelUsage
	if err := db.Where("run_id = ? AND step_id = ?", run.ID, "pm.plan").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Provider != "primary" {
		t.Fatalf("usage=%+v want primary usage row", rows)
	}
}

func TestIndexRAGRejectsUnauthorizedTargetSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_rag_perm", Name: "RAG Org", Slug: "rag-org", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_rag_own", OrgID: org.ID, Name: "Own Space", Slug: "own", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_rag_other", OrgID: org.ID, Name: "Other Space", Slug: "other", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_rag_perm", DisplayName: "RAG User", Status: "active", CreatedAt: now, UpdatedAt: now}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"repoRoot":".","spaceId":"` + otherSpace.ID + `"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rag/index", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestCreateRunRejectsUnauthorizedTargetSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_run_create_scope", Name: "Run Create Org", Slug: "run-create-org", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_run_create_own", OrgID: org.ID, Name: "Own Run Space", Slug: "own-run", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_run_create_other", OrgID: org.ID, Name: "Other Run Space", Slug: "other-run", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_run_create_scope", DisplayName: "Run Create User", Status: "active", CreatedAt: now, UpdatedAt: now}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},"inputs":{},"spaceId":"` + otherSpace.ID + `"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestMemoryQueryFiltersCurrentSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_memory_scope", Name: "Memory Scope Org", Slug: "memory-scope", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_memory_own", OrgID: org.ID, Name: "Own Memory Space", Slug: "own-memory", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_memory_other", OrgID: org.ID, Name: "Other Memory Space", Slug: "other-memory", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_memory_scope", DisplayName: "Memory Scope User", Status: "active", CreatedAt: now, UpdatedAt: now}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	ownRecord := store.MemoryRecord{
		ID: "mem_own_scope", Layer: "L0", Status: "approved", SpaceID: ownSpace.ID,
		SchemaVersion: 1, Title: "Shared marker", Body: "tenant scoped answer", TagsJSON: "[]",
		Sensitivity: "normal", Confidence: 0.9, CreatedAt: now, UpdatedAt: now,
	}
	otherRecord := store.MemoryRecord{
		ID: "mem_other_scope", Layer: "L0", Status: "approved", SpaceID: otherSpace.ID,
		SchemaVersion: 1, Title: "Shared marker", Body: "tenant scoped answer", TagsJSON: "[]",
		Sensitivity: "normal", Confidence: 0.9, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&ownRecord).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&otherRecord).Error; err != nil {
		t.Fatal(err)
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "maintainer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"text":"shared marker","topK":10}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/memory/query", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].ID != ownRecord.ID {
		t.Fatalf("items=%+v want only own space memory", resp.Items)
	}
}

func TestFeedbackRejectsUnauthorizedTargetSpace(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_feedback_scope", Name: "Feedback Org", Slug: "feedback-org", CreatedAt: now, UpdatedAt: now}
	ownSpace := store.Space{ID: "space_feedback_own", OrgID: org.ID, Name: "Own Feedback Space", Slug: "own-feedback", CreatedAt: now, UpdatedAt: now}
	otherSpace := store.Space{ID: "space_feedback_other", OrgID: org.ID, Name: "Other Feedback Space", Slug: "other-feedback", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_feedback_scope", DisplayName: "Feedback User", Status: "active", CreatedAt: now, UpdatedAt: now}
	for _, row := range []any{&org, &ownSpace, &otherSpace, &user} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: ownSpace.ID, Role: "viewer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"targetType":"run","targetId":"run_x","rating":1,"spaceId":"` + otherSpace.ID + `"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestFeedbackMemoryTargetDecaysConfidence(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	mem := store.MemoryRecord{
		ID: "mem_feedback_decay", SpaceID: "local", Layer: "L0", Status: "approved",
		SchemaVersion: 2, Title: "feedback decay", Body: "target body",
		Sensitivity: "normal", Confidence: 0.8, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&mem).Error; err != nil {
		t.Fatal(err)
	}

	createBody := []byte(`{"targetType":"memory","targetId":"mem_feedback_decay","rating":1,"category":"quality","comment":"命中后有害"}`)
	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d want 201 body=%s", createResp.Code, createResp.Body.String())
	}

	var row store.MemoryRecord
	if err := db.First(&row, "id = ?", mem.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Confidence != 0.65 {
		t.Fatalf("confidence=%v want 0.65 after rating=1 decay", row.Confidence)
	}
	var audits int64
	if err := db.Model(&store.AuditLog{}).
		Where("event_type = ? AND payload_json LIKE ?", "memory.confidence_adjusted", "%mem_feedback_decay%").
		Count(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Fatalf("audits=%d want 1", audits)
	}
}

func TestFeedbackListUpdateAndLowScoreAlert(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)

	createBody := []byte(`{"targetType":"ci_diagnosis","targetId":"ci_diag_1","rating":1,"category":"ci","comment":"失败定位不准"}`)
	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create status=%d want 201 body=%s", createResp.Code, createResp.Body.String())
	}
	var fb store.Feedback
	if err := json.Unmarshal(createResp.Body.Bytes(), &fb); err != nil {
		t.Fatal(err)
	}
	if fb.Category != "ci" || fb.Status != "open" || fb.Severity != "warn" {
		t.Fatalf("feedback=%+v want normalized fields", fb)
	}
	var alertsCount int64
	if err := db.Model(&store.AlertEvent{}).Where("target_id = ?", fb.ID).Count(&alertsCount).Error; err != nil {
		t.Fatal(err)
	}
	if alertsCount != 1 {
		t.Fatalf("alerts=%d want 1", alertsCount)
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/feedback?category=ci&status=open", nil)
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d want 200 body=%s", listResp.Code, listResp.Body.String())
	}
	var list struct {
		Items []store.Feedback `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != fb.ID {
		t.Fatalf("list=%+v want feedback", list)
	}

	patchResp := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/feedback/"+fb.ID, bytes.NewReader([]byte(`{"status":"resolved","severity":"info"}`)))
	patchReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(patchResp, patchReq)
	if patchResp.Code != http.StatusOK {
		t.Fatalf("patch status=%d want 200 body=%s", patchResp.Code, patchResp.Body.String())
	}
	var updated store.Feedback
	if err := json.Unmarshal(patchResp.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status != "resolved" || updated.Severity != "info" {
		t.Fatalf("updated=%+v want resolved/info", updated)
	}
}

func TestArtifactAccessUsesRunSpacePermissionAndAudits(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "jwt")
	t.Setenv("ASH_JWT_SECRET", "test-secret")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	org := store.Org{ID: "org_artifact_access", Name: "Artifact Org", Slug: "artifact-org", CreatedAt: now, UpdatedAt: now}
	space := store.Space{ID: "space_artifact_access", OrgID: org.ID, Name: "Artifact Space", Slug: "artifact-space", CreatedAt: now, UpdatedAt: now}
	user := store.User{ID: "user_artifact_access", DisplayName: "Artifact User", Status: "active", CreatedAt: now, UpdatedAt: now}
	run := store.RunRecord{
		ID: "run_artifact_access", TraceID: "trace_artifact_access",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: space.ID,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	artifact := store.ArtifactIndex{
		ID: "artidx_access", RunID: run.ID, StepID: "ship.package",
		Type: "release_notes", Name: "release_notes.md", URI: "fs:///tmp/release_notes.md",
		Digest: "sha256:test", ContentType: "text/markdown", SizeBytes: 42, CreatedAt: now,
	}
	checkpoint := store.Checkpoint{
		ID: "ckpt_access", RunID: run.ID, StepID: "qa.verify",
		SnapshotDigest: "sha256:checkpoint", URI: "fs:///tmp/checkpoint.json",
		StoreKey:    "runs/" + run.ID + "/checkpoints/ckpt_access.json",
		ContentType: "application/json", SizeBytes: 64, Strategy: "per_step", CreatedAt: now,
	}
	for _, row := range []any{&org, &space, &user, &run, &artifact, &checkpoint} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	token, err := signToken(tokenClaims{Sub: user.ID, SpaceID: space.ID, Role: "maintainer"}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID+"/artifacts/"+artifact.Name+"/access?ttlSeconds=60", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		RunID     string `json:"runId"`
		Name      string `json:"name"`
		SignedURL string `json:"signedUrl"`
		ExpiresAt int64  `json:"expiresAt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.RunID != run.ID || resp.Name != artifact.Name || resp.SignedURL == "" || resp.ExpiresAt <= now.UnixMilli() {
		t.Fatalf("resp=%+v want signed artifact access", resp)
	}
	var audits []store.AuditLog
	if err := db.Where("run_id = ? AND event_type = ?", run.ID, "artifact.access_url_issued").Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].SpaceID != space.ID {
		t.Fatalf("audits=%+v want one audit in run space", audits)
	}

	ckptListResp := httptest.NewRecorder()
	ckptListReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID+"/checkpoints", nil)
	ckptListReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(ckptListResp, ckptListReq)
	if ckptListResp.Code != http.StatusOK {
		t.Fatalf("checkpoint list status=%d want %d body=%s", ckptListResp.Code, http.StatusOK, ckptListResp.Body.String())
	}
	var checkpointList struct {
		Items []store.Checkpoint `json:"items"`
	}
	if err := json.Unmarshal(ckptListResp.Body.Bytes(), &checkpointList); err != nil {
		t.Fatal(err)
	}
	if len(checkpointList.Items) != 1 || checkpointList.Items[0].ID != checkpoint.ID || checkpointList.Items[0].RunID != run.ID {
		t.Fatalf("checkpoint list=%+v want run checkpoint", checkpointList.Items)
	}

	ckptResp := httptest.NewRecorder()
	ckptReq := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID+"/checkpoints/"+checkpoint.ID+"/access?ttlSeconds=60", nil)
	ckptReq.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(ckptResp, ckptReq)
	if ckptResp.Code != http.StatusOK {
		t.Fatalf("checkpoint status=%d want %d body=%s", ckptResp.Code, http.StatusOK, ckptResp.Body.String())
	}
	var checkpointAccess struct {
		RunID          string `json:"runId"`
		CheckpointID   string `json:"checkpointId"`
		SignedURL      string `json:"signedUrl"`
		SnapshotDigest string `json:"snapshotDigest"`
	}
	if err := json.Unmarshal(ckptResp.Body.Bytes(), &checkpointAccess); err != nil {
		t.Fatal(err)
	}
	if checkpointAccess.RunID != run.ID || checkpointAccess.CheckpointID != checkpoint.ID ||
		checkpointAccess.SignedURL == "" || checkpointAccess.SnapshotDigest != checkpoint.SnapshotDigest {
		t.Fatalf("checkpoint access=%+v want signed checkpoint access", checkpointAccess)
	}
	audits = nil
	if err := db.Where("run_id = ? AND event_type = ?", run.ID, "checkpoint.access_url_issued").Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].SpaceID != space.ID {
		t.Fatalf("checkpoint audits=%+v want one audit in run space", audits)
	}
}
