package interaction_test

import (
	"encoding/json"
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/interaction"
)

func TestFoldEvents_buildsNodesAndHitUsedLinks(t *testing.T) {
	evs := []events.Envelope{
		{
			ID: "e1", RunID: "run_1", Seq: 1, TS: 100, Type: "session.turn",
			Visibility: events.VisibilityModelVisible,
			Payload:    json.RawMessage(`{"prompt":"hi","sessionId":"sess_1"}`),
		},
		{
			ID: "e2", RunID: "run_1", Seq: 2, TS: 200, Type: "memory.hit_used",
			Visibility: events.VisibilityModelVisible,
			Payload:    json.RawMessage(`{"count":2,"recordIds":["mem_a","mem_b"],"hitsByLayer":{"L1":2}}`),
		},
		{
			ID: "e3", RunID: "run_1", Seq: 3, TS: 300, Type: "metric.kpi",
			Visibility: events.VisibilityAudit,
			Payload:    json.RawMessage(`{"n":1}`),
		},
	}
	out := interaction.FoldEvents("sess_1", "th_1", "run_1", "local", evs)
	if len(out.Nodes) != 2 {
		t.Fatalf("nodes=%d want 2 (audit hidden); %+v", len(out.Nodes), out.Nodes)
	}
	if out.Nodes[0].Type != "session.turn" || out.Nodes[1].Type != "memory.hit_used" {
		t.Fatalf("nodes=%+v", out.Nodes)
	}
	if len(out.Links) != 2 {
		t.Fatalf("links=%d want 2; %+v", len(out.Links), out.Links)
	}
	for _, link := range out.Links {
		if link.SessionID != "sess_1" || link.ThreadID != "th_1" || link.LinkType != interaction.LinkHitUsed {
			t.Fatalf("link=%+v", link)
		}
		if link.MemoryID != "mem_a" && link.MemoryID != "mem_b" {
			t.Fatalf("memoryId=%q", link.MemoryID)
		}
	}
	if out.Digest == "" {
		t.Fatal("expected digest")
	}
	again := interaction.FoldEvents("sess_1", "th_1", "run_1", "local", evs)
	if again.Digest != out.Digest {
		t.Fatalf("digest not stable: %q vs %q", out.Digest, again.Digest)
	}
}

func TestFoldEvents_contextRefs(t *testing.T) {
	evs := []events.Envelope{
		{
			ID: "e1", RunID: "run_2", Seq: 1, TS: 1, Type: "memory.injected",
			Visibility: events.VisibilityModelVisible,
			Payload:    json.RawMessage(`{"count":1,"recordIds":["mem_c"]}`),
		},
		{
			ID: "e2", RunID: "run_2", Seq: 2, TS: 2, Type: "knowledge.injected",
			Visibility: events.VisibilityModelVisible,
			Payload:    json.RawMessage(`{"refs":["memory:mem_d","skill:foo"]}`),
		},
	}
	out := interaction.FoldEvents("sess_2", "th_2", "run_2", "local", evs)
	var hitTypes []string
	for _, l := range out.Links {
		hitTypes = append(hitTypes, l.LinkType+":"+l.MemoryID)
	}
	if len(out.Links) < 2 {
		t.Fatalf("links=%v", hitTypes)
	}
}
