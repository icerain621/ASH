package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestGetSpaceAuditReport(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	rows := []store.AuditLog{
		{ID: "a1", SpaceID: "local", EventType: "gate.approved", PayloadJSON: "{}", CreatedAt: now},
		{ID: "a2", SpaceID: "local", EventType: "policy.denied", PayloadJSON: "{}", CreatedAt: now},
		{ID: "a3", SpaceID: "local", EventType: "hook.pre_tool_use", PayloadJSON: "{}", CreatedAt: now},
		{ID: "a4", SpaceID: "local", EventType: "run.spawn", PayloadJSON: "{}", CreatedAt: now},
		{ID: "a5", SpaceID: "local", EventType: "run.started", PayloadJSON: "{}", CreatedAt: now.Add(-40 * 24 * time.Hour)},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/spaces/local/audit-report?window=7d", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp AuditReport
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Window != "7d" || resp.Total != 4 {
		t.Fatalf("window/total %+v", resp)
	}
	if resp.Buckets.Approve != 1 || resp.Buckets.Deny != 1 || resp.Buckets.Hook != 1 || resp.Buckets.Spawn != 1 {
		t.Fatalf("buckets=%+v", resp.Buckets)
	}

	bad := httptest.NewRecorder()
	r.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "/api/v1/spaces/local/audit-report?window=1y", nil))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("bad window status=%d", bad.Code)
	}
}
