package runs

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/spacepolicy"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestApplySubRunPolicy(t *testing.T) {
	if got := applySubRunDepth(4, spacepolicy.SubRunPolicy{}); got != 4 {
		t.Fatalf("harness depth %d", got)
	}
	if got := applySubRunDepth(4, spacepolicy.SubRunPolicy{MaxDepth: 1}); got != 1 {
		t.Fatalf("policy override %d", got)
	}
	got := applySubRunTools([]string{"read", "grep", "bash"}, spacepolicy.SubRunPolicy{AllowedTools: []string{"read"}})
	if len(got) != 1 || got[0] != "read" {
		t.Fatalf("tools=%v", got)
	}
	bad := applySubRunTools([]string{"bash"}, spacepolicy.SubRunPolicy{AllowedTools: []string{"bash", "read"}})
	if err := validateSubRunAllowlist(bad); err == nil {
		t.Fatal("dangerous tool must stay rejected")
	}
}

func TestSpawnHonorsSpaceSubRunMaxDepth(t *testing.T) {
	svc, _ := testRunsService(t)
	space := "sp_subrun"
	now := time.Now().UTC()
	if err := svc.gdb().Create(&store.SpacePolicyPack{
		SpaceID: space, CitationMode: "optional", BodyJSON: `{"subRun":{"maxDepth":1}}`,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	parentID := space + "_parent"
	if err := svc.gdb().Create(&store.RunRecord{
		ID: parentID, TraceID: "trc_sub", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: StatusRunning, SpaceID: space, Depth: 1,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := svc.Spawn(parentID, SpawnRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "child"},
		Reason:   "test",
	})
	if err == nil || !strings.Contains(err.Error(), "exceeds maxDepth 1") {
		t.Fatalf("want depth reject, got %v", err)
	}
}
