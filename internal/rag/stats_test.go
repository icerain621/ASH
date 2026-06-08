package rag

import (
	"fmt"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestCountChunkFallbackQueries(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_rag_fb", TraceID: "tr_rag", SpaceID: "local",
		ScenarioName: "x", ScenarioVersion: "1", PolicyProfile: "default",
		Status: "finished", ActorRole: "maintainer", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	chunkEvt := store.RunEvent{
		ID: "evt_chunk", RunID: run.ID, Seq: 1, TS: now.UnixMilli(),
		Type: "rag.retrieved", Severity: "info",
		PayloadJSON: `{"retrievalMode":"chunk","hits":1}`,
		CreatedAt: now,
	}
	ftsEvt := store.RunEvent{
		ID: "evt_fts", RunID: run.ID, Seq: 2, TS: now.UnixMilli(),
		Type: "rag.retrieved", Severity: "info",
		PayloadJSON: `{"retrievalMode":"fts","hits":2}`,
		CreatedAt: now,
	}
	for _, row := range []any{&run, &chunkEvt, &ftsEvt} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if n := CountChunkFallbackQueries(db.DB, "local"); n != 1 {
		t.Fatalf("fallback count=%d want 1", n)
	}
}

func TestFallbackRateInWindow(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	run := store.RunRecord{
		ID: "run_rag_rate", TraceID: "tr_rate", SpaceID: "local",
		ScenarioName: "x", ScenarioVersion: "1", PolicyProfile: "default",
		Status: "finished", ActorRole: "maintainer", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}
	for i, payload := range []string{
		`{"retrievalMode":"chunk","hits":1}`,
		`{"retrievalMode":"fts","hits":1}`,
		`{"retrievalMode":"fts","hits":1}`,
	} {
		evt := store.RunEvent{
			ID: fmt.Sprintf("evt_rate_%d", i), RunID: run.ID, Seq: int64(i + 1), TS: now.UnixMilli(),
			Type: "rag.retrieved", Severity: "info", PayloadJSON: payload, CreatedAt: now,
		}
		if err := db.Create(&evt).Error; err != nil {
			t.Fatal(err)
		}
	}
	rate, total, chunk := FallbackRateInWindow(db.DB, "local", now.Add(-time.Hour))
	if total != 3 || chunk != 1 {
		t.Fatalf("total=%d chunk=%d want 3/1", total, chunk)
	}
	if rate < 0.33 || rate > 0.34 {
		t.Fatalf("rate=%v want ~0.33", rate)
	}
}
