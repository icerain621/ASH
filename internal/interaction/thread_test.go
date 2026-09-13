package interaction_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/interaction"
)

func TestEnsureMainThread_createsOnce(t *testing.T) {
	meta := map[string]any{}
	id1, meta1, created := interaction.EnsureMainThread(meta, "sess_1", "run_1")
	if !created || id1 == "" || meta1["threadId"] != id1 {
		t.Fatalf("first ensure id=%q created=%v meta=%v", id1, created, meta1)
	}
	if meta1["threadKind"] != "main" {
		t.Fatalf("kind=%v", meta1["threadKind"])
	}
	id2, _, created2 := interaction.EnsureMainThread(meta1, "sess_1", "run_1")
	if created2 || id2 != id1 {
		t.Fatalf("second ensure id=%q created=%v want same %q", id2, created2, id1)
	}
}

func TestEnsureMainThread_nilMeta(t *testing.T) {
	id, meta, created := interaction.EnsureMainThread(nil, "sess_2", "run_2")
	if !created || id == "" || meta == nil || meta["threadId"] != id {
		t.Fatalf("id=%q created=%v meta=%v", id, created, meta)
	}
}
