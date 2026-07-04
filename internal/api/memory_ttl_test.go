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

func TestMemoryTTLQueueAndSweepAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_MEMORY_TTL_REVIEW_DAYS", "7")
	r, db := newPlatformTestRouter(t)
	space := "space_ttl_api"
	now := time.Now().UTC()

	reviewTTL := 10
	reviewID := "mem_api_review"
	expiresReview := now.Add(5 * 24 * time.Hour)
	createdReview := expiresReview.Add(-time.Duration(reviewTTL) * 24 * time.Hour)
	if err := db.Create(&store.MemoryRecord{
		ID: reviewID, Layer: "L1", Status: "approved", SpaceID: space,
		SchemaVersion: 2, Title: "review soon", Body: "body", TTLDays: &reviewTTL,
		CreatedAt: createdReview, UpdatedAt: createdReview,
	}).Error; err != nil {
		t.Fatal(err)
	}

	expiredTTL := 1
	expiredID := "mem_api_expired"
	createdExpired := now.Add(-48 * time.Hour)
	if err := db.Create(&store.MemoryRecord{
		ID: expiredID, Layer: "L2", Status: "approved", SpaceID: space,
		SchemaVersion: 2, Title: "expired", Body: "body", TTLDays: &expiredTTL,
		CreatedAt: createdExpired, UpdatedAt: createdExpired,
	}).Error; err != nil {
		t.Fatal(err)
	}

	queueResp := httptest.NewRecorder()
	queueReq := httptest.NewRequest(http.MethodGet, "/api/v1/memory/ttl-queue?limit=10", nil)
	queueReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(queueResp, queueReq)
	if queueResp.Code != http.StatusOK {
		t.Fatalf("ttl-queue status=%d body=%s", queueResp.Code, queueResp.Body.String())
	}
	var queue memory.TTLQueueResponse
	if err := json.Unmarshal(queueResp.Body.Bytes(), &queue); err != nil {
		t.Fatal(err)
	}
	if queue.ReviewDueCount != 1 || queue.ExpiredPendingCount != 1 {
		t.Fatalf("queue=%+v want 1 review 1 expired", queue)
	}
	if len(queue.ReviewDue) != 1 || queue.ReviewDue[0].RecordID != reviewID {
		t.Fatalf("reviewDue=%+v", queue.ReviewDue)
	}

	sweepResp := httptest.NewRecorder()
	sweepReq := httptest.NewRequest(http.MethodPost, "/api/v1/memory/ttl-sweep", bytes.NewReader([]byte(`{}`)))
	sweepReq.Header.Set("Content-Type", "application/json")
	sweepReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(sweepResp, sweepReq)
	if sweepResp.Code != http.StatusOK {
		t.Fatalf("ttl-sweep status=%d body=%s", sweepResp.Code, sweepResp.Body.String())
	}
	var sweep memory.SweepTTLResponse
	if err := json.Unmarshal(sweepResp.Body.Bytes(), &sweep); err != nil {
		t.Fatal(err)
	}
	if !sweep.OK || sweep.Deprecated != 1 {
		t.Fatalf("sweep=%+v want deprecated=1", sweep)
	}
	var deprecated store.MemoryRecord
	if err := db.First(&deprecated, "id = ?", expiredID).Error; err != nil {
		t.Fatal(err)
	}
	if deprecated.Status != "deprecated" {
		t.Fatalf("status=%q want deprecated", deprecated.Status)
	}
}
