package interaction_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/interaction"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestSealReplayCompare(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := interaction.NewService(db, ev)

	th, _, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_seal", RunID: "run_seal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append("run_seal", "tr", "session.turn", "info", map[string]any{"prompt": "a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append("run_seal", "tr", "memory.hit_used", "info", map[string]any{
		"recordIds": []string{"mem_s"}, "count": 1,
	}); err != nil {
		t.Fatal(err)
	}

	sealed, err := svc.Seal(th.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sealed.Status != interaction.ThreadStatusSealed || sealed.Digest == "" {
		t.Fatalf("sealed=%+v", sealed)
	}

	replay, err := svc.Replay(th.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.OK || replay.Digest != sealed.Digest || len(replay.Nodes) < 2 {
		t.Fatalf("replay=%+v", replay)
	}

	// Second thread with different events for compare.
	thB, _, err := svc.EnsureThread(interaction.EnsureRequest{
		SpaceID: "local", SessionID: "sess_b", RunID: "run_seal_b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append("run_seal_b", "tr", "session.turn", "info", map[string]any{"prompt": "b"}); err != nil {
		t.Fatal(err)
	}
	diff, err := svc.Compare(th.ID, thB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff.LeftDigest == "" || diff.RightDigest == "" || diff.LeftDigest == diff.RightDigest {
		t.Fatalf("compare=%+v", diff)
	}
	if len(diff.NodesAdded)+len(diff.NodesRemoved)+len(diff.NodesChanged) == 0 &&
		len(diff.LinksAdded)+len(diff.LinksRemoved) == 0 {
		t.Fatalf("expected some node/link diff: %+v", diff)
	}
}

func TestCompareJSONEmptyDimsAreArraysNotNull(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := interaction.NewService(db, ev)
	thA, _, err := svc.EnsureThread(interaction.EnsureRequest{SpaceID: "local", RunID: "run_cmp_a"})
	if err != nil {
		t.Fatal(err)
	}
	thB, _, err := svc.EnsureThread(interaction.EnsureRequest{SpaceID: "local", RunID: "run_cmp_b"})
	if err != nil {
		t.Fatal(err)
	}
	diff, err := svc.Compare(thA.ID, thB.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(diff)
	if err != nil {
		t.Fatal(err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"nodesAdded", "nodesRemoved", "nodesChanged", "linksAdded", "linksRemoved"} {
		v, ok := asMap[key]
		if !ok {
			t.Fatalf("missing %s in %s", key, raw)
		}
		if v == nil {
			t.Fatalf("%s marshaled as null; want [] in %s", key, raw)
		}
		arr, ok := v.([]any)
		if !ok {
			t.Fatalf("%s=%T want array in %s", key, v, raw)
		}
		_ = arr
	}
}

func TestReplayDigestMismatch(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := interaction.NewService(db, ev)
	th, _, err := svc.EnsureThread(interaction.EnsureRequest{SpaceID: "local", RunID: "run_mm"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Append("run_mm", "tr", "session.turn", "info", map[string]any{"prompt": "x"}); err != nil {
		t.Fatal(err)
	}
	sealed, err := svc.Seal(th.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Tamper sealed digest in DB.
	if err := db.Model(&store.InteractionThread{}).Where("id = ?", th.ID).
		Update("digest", sealed.Digest+"_tampered").Error; err != nil {
		t.Fatal(err)
	}
	_, err = svc.Replay(th.ID)
	if !errors.Is(err, interaction.ErrReplayDigestMismatch) {
		t.Fatalf("err=%v want ErrReplayDigestMismatch", err)
	}
}
