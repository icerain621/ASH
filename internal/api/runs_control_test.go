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

func TestRunControlNegativeStatusCodes(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_ctrl_neg", TraceID: "trace_ctrl_neg",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "running", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	canceled := store.RunRecord{
		ID: "run_ctrl_canceled", TraceID: "trace_ctrl_canceled",
		ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "canceled", SpaceID: "local",
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	for _, row := range []store.RunRecord{run, canceled} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name: "resume running", method: http.MethodPost, path: "/api/v1/runs/" + run.ID + "/resume",
			wantStatus: http.StatusConflict, wantCode: "RUN_NOT_RESUMABLE",
		},
		{
			name: "approve running", method: http.MethodPost, path: "/api/v1/runs/" + run.ID + "/approve",
			body: `{"actorId":"t","reason":"x"}`, wantStatus: http.StatusConflict, wantCode: "RUN_NOT_APPROVABLE",
		},
		{
			name: "replay running", method: http.MethodPost, path: "/api/v1/runs/" + run.ID + "/replay",
			body: `{"mode":"exact"}`, wantStatus: http.StatusConflict, wantCode: "RUN_NOT_REPLAYABLE",
		},
		{
			name: "resume canceled", method: http.MethodPost, path: "/api/v1/runs/" + canceled.ID + "/resume",
			wantStatus: http.StatusConflict, wantCode: "RUN_NOT_RESUMABLE",
		},
		{
			name: "approve canceled", method: http.MethodPost, path: "/api/v1/runs/" + canceled.ID + "/approve",
			body: `{"actorId":"t","reason":"x"}`, wantStatus: http.StatusConflict, wantCode: "RUN_NOT_APPROVABLE",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Reader
			if tc.body != "" {
				body = bytes.NewReader([]byte(tc.body))
			} else {
				body = bytes.NewReader(nil)
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, body)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("status=%d want %d body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Error.Code != tc.wantCode {
				t.Fatalf("code=%q want %q body=%s", payload.Error.Code, tc.wantCode, w.Body.String())
			}
		})
	}
}

func TestScaleReadinessRunBacklogCounts(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, db := newPlatformTestRouter(t)
	now := time.Now().UTC()
	space := "local"
	rows := []store.RunRecord{
		{ID: "run_backlog_1", TraceID: "t1", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			PolicyProfile: "default", Status: "running", SpaceID: space, StartedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "run_backlog_2", TraceID: "t2", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			PolicyProfile: "default", Status: "waiting_approval", SpaceID: space, StartedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "run_backlog_3", TraceID: "t3", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
			PolicyProfile: "default", Status: "finished", SpaceID: space, StartedAt: now, CreatedAt: now, UpdatedAt: now},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scale/readiness", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var resp ScaleReadinessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.RunRunningCount < 1 || resp.RunWaitingApprovalCount < 1 {
		t.Fatalf("backlog running=%d waiting=%d want >=1 each", resp.RunRunningCount, resp.RunWaitingApprovalCount)
	}
	if resp.RunInflightCount != resp.RunRunningCount+resp.RunWaitingApprovalCount {
		t.Fatalf("inflight=%d want running+waiting", resp.RunInflightCount)
	}
}
