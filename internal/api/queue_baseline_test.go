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

func TestTTLQueueConsumeBaseline(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_MEMORY_TTL_REVIEW_DAYS", "7")
	r, db := newPlatformTestRouter(t)
	space := "space_queue_baseline"
	now := time.Now().UTC()

	expiredTTL := 1
	expiredID := "mem_queue_expired"
	createdExpired := now.Add(-72 * time.Hour)
	if err := db.Create(&store.MemoryRecord{
		ID: expiredID, Layer: "L2", Status: "approved", SpaceID: space,
		SchemaVersion: 2, Title: "expired", Body: "body", TTLDays: &expiredTTL,
		CreatedAt: createdExpired, UpdatedAt: createdExpired,
	}).Error; err != nil {
		t.Fatal(err)
	}

	queueBefore := getTTLQueue(t, r, space)
	if queueBefore.ExpiredPendingCount != 1 {
		t.Fatalf("before sweep queue=%+v want 1 expired pending", queueBefore)
	}

	start := time.Now()
	sweepResp := httptest.NewRecorder()
	sweepReq := httptest.NewRequest(http.MethodPost, "/api/v1/memory/ttl-sweep", bytes.NewReader([]byte(`{}`)))
	sweepReq.Header.Set("Content-Type", "application/json")
	sweepReq.Header.Set("X-ASH-Space-ID", space)
	r.ServeHTTP(sweepResp, sweepReq)
	if sweepResp.Code != http.StatusOK {
		t.Fatalf("ttl-sweep status=%d body=%s", sweepResp.Code, sweepResp.Body.String())
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("ttl-sweep took %s want < 500ms", elapsed)
	}

	queueAfter := getTTLQueue(t, r, space)
	if queueAfter.ExpiredPendingCount != 0 {
		t.Fatalf("after sweep queue=%+v want 0 expired pending", queueAfter)
	}
}

func getTTLQueue(t *testing.T, r http.Handler, space string) memory.TTLQueueResponse {
	t.Helper()
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
	return queue
}
