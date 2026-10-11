package review

import (
	"fmt"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestSubmitAndDecide(t *testing.T) {
	svc := NewService()
	item, err := svc.Submit(KindMemoryRecord, "mem_1", "consolidate")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != StatusPending {
		t.Fatalf("%+v", item)
	}
	if len(svc.List(KindMemoryRecord)) != 1 {
		t.Fatal("want one pending")
	}
	decided, err := svc.Decide(item.ID, "approve", "ok")
	if err != nil {
		t.Fatal(err)
	}
	if decided.Status != StatusApproved {
		t.Fatalf("%+v", decided)
	}
	if len(svc.List("")) != 0 {
		t.Fatal("queue should be empty")
	}
}

func TestRejectUnknownKind(t *testing.T) {
	svc := NewService()
	if _, err := svc.Submit(Kind("nope"), "x", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadMemoryEvalCountsFromEvents(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_eval", SpaceID: "local", TraceID: "tr", Status: "succeeded",
		ScenarioName: "x", ScenarioVersion: "1", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	for i, payload := range []string{
		`{"kind":"hit_used","recordId":"m1"}`,
		`{"kind":"used_correctly","recordId":"m1"}`,
		`{"kind":"wrong_update","recordId":"m1"}`,
	} {
		if err := db.Create(&store.RunEvent{
			ID: fmt.Sprintf("ev_fb_%d", i), RunID: run.ID, Seq: int64(i + 1), TS: now.UnixMilli(),
			Type: "memory.feedback", PayloadJSON: payload, CreatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&store.RunEvent{
		ID: "ev_hit", RunID: run.ID, Seq: 4, TS: now.UnixMilli(),
		Type: "memory.hit_used", PayloadJSON: `{"recordId":"m1"}`, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	got, err := LoadMemoryEvalCounts(db, "local", 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.Remembered != 2 || got.UsedCorrectly != 1 || got.WrongUpdate != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestMemoryEvalCounts(t *testing.T) {
	got := MemoryEvalCounts([]string{"hit_used", "hit_used", "used_correctly", "wrong_update"})
	if got.Remembered != 2 || got.UsedCorrectly != 1 || got.WrongUpdate != 1 {
		t.Fatalf("%+v", got)
	}
}
