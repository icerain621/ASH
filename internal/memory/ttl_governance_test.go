package memory

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestTTLQueueAndSweep(t *testing.T) {
	t.Setenv("ASH_MEMORY_TTL_REVIEW_DAYS", "7")
	svc, _, _ := newTestMemory(t)
	now := time.Now().UTC()
	space := "local"

	reviewTTL := 10
	reviewID := "mem_ttl_review"
	expiresAt := now.Add(5 * 24 * time.Hour)
	createdReview := expiresAt.Add(-time.Duration(reviewTTL) * 24 * time.Hour)
	if err := svc.gdb().Create(&store.MemoryRecord{
		ID: reviewID, Layer: "L1", Status: "approved", SpaceID: space,
		SchemaVersion: 2, Title: "review soon", Body: "body", TTLDays: &reviewTTL,
		CreatedAt: createdReview, UpdatedAt: createdReview,
	}).Error; err != nil {
		t.Fatal(err)
	}

	expiredTTL := 1
	expiredID := "mem_ttl_expired"
	createdExpired := now.Add(-48 * time.Hour)
	if err := svc.gdb().Create(&store.MemoryRecord{
		ID: expiredID, Layer: "L2", Status: "approved", SpaceID: space,
		SchemaVersion: 2, Title: "expired", Body: "body", TTLDays: &expiredTTL,
		CreatedAt: createdExpired, UpdatedAt: createdExpired,
	}).Error; err != nil {
		t.Fatal(err)
	}

	queue, err := TTLQueue(svc.db, space, 10)
	if err != nil {
		t.Fatal(err)
	}
	if queue.ReviewDueCount != 1 || queue.ExpiredPendingCount != 1 {
		t.Fatalf("queue=%+v want 1 review 1 expired", queue)
	}
	if len(queue.ReviewDue) != 1 || queue.ReviewDue[0].RecordID != reviewID {
		t.Fatalf("reviewDue=%+v", queue.ReviewDue)
	}

	dry, err := svc.SweepTTL(SweepTTLRequest{SpaceID: space, DryRun: true})
	if err != nil || dry.Deprecated != 1 || dry.ReviewDue != 1 {
		t.Fatalf("dry=%+v err=%v", dry, err)
	}
	var stillApproved store.MemoryRecord
	if err := svc.gdb().First(&stillApproved, "id = ?", expiredID).Error; err != nil {
		t.Fatal(err)
	}
	if stillApproved.Status != "approved" {
		t.Fatalf("dry-run changed status=%q", stillApproved.Status)
	}

	sweep, err := svc.SweepTTL(SweepTTLRequest{SpaceID: space, ActorID: "tester"})
	if err != nil || sweep.Deprecated != 1 {
		t.Fatalf("sweep=%+v err=%v", sweep, err)
	}
	var deprecated store.MemoryRecord
	if err := svc.gdb().First(&deprecated, "id = ?", expiredID).Error; err != nil {
		t.Fatal(err)
	}
	if deprecated.Status != "deprecated" {
		t.Fatalf("status=%q want deprecated", deprecated.Status)
	}
	q, err := svc.Query(QueryRequest{Text: "expired", TopK: 5})
	if err != nil || len(q.Items) != 0 {
		t.Fatalf("query=%+v want empty after sweep", q.Items)
	}
	var ttlAudits int64
	if err := svc.db.Model(&store.AuditLog{}).
		Where("event_type = ? AND payload_json LIKE ?", "memory.ttl_expired", "%"+expiredID+"%").
		Count(&ttlAudits).Error; err != nil {
		t.Fatal(err)
	}
	if ttlAudits != 1 {
		t.Fatalf("ttl audits=%d want 1", ttlAudits)
	}
}

func TestClassifyTTLStates(t *testing.T) {
	t.Setenv("ASH_MEMORY_TTL_REVIEW_DAYS", "7")
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	ttl := 30
	row := store.MemoryRecord{
		Status: "approved", TTLDays: &ttl,
		CreatedAt: now.Add(-23 * 24 * time.Hour),
	}
	if classifyTTL(row, now) != TTLStateReviewDue {
		t.Fatalf("want review_due for 7 days left")
	}
	row.CreatedAt = now.Add(-31 * 24 * time.Hour)
	if classifyTTL(row, now) != TTLStateExpired {
		t.Fatalf("want expired")
	}
	row.CreatedAt = now.Add(-10 * 24 * time.Hour)
	if classifyTTL(row, now) != TTLStateActive {
		t.Fatalf("want active for 20 days left")
	}
}
