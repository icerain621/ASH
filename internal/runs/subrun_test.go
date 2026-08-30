package runs

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestValidateSubRunAllowlist(t *testing.T) {
	if err := validateSubRunAllowlist([]string{"read", "grep"}); err != nil {
		t.Fatal(err)
	}
	if err := validateSubRunAllowlist([]string{"write"}); err == nil {
		t.Fatal("expected write denied")
	}
	if maxDepthFromSpec(nil) != 2 {
		t.Fatal("default max depth")
	}
	if maxDepthFromSpec(&harness.SubRunSpec{MaxDepth: 3}) != 3 {
		t.Fatal("custom max depth")
	}
}

func TestBuildTreeAndDepthGate(t *testing.T) {
	root := store.RunRecord{ID: "run_root", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0", Status: "finished"}
	child := store.RunRecord{ID: "run_child", ParentRunID: "run_root", RootRunID: "run_root", Depth: 1, ScenarioName: "hotfix", ScenarioVersion: "1.0.0", Status: "finished"}
	byParent := map[string][]store.RunRecord{"run_root": {child}}
	node := buildTreeNode(root, byParent)
	if node.Summary.RunID != "run_root" || len(node.Children) != 1 {
		t.Fatalf("%+v", node)
	}

	db := store.OpenTest(t, t.TempDir())
	loader := rules.NewLoader(filepath.Join("..", "..", "scenarios"))
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, events.NewService(db), loader, toolbus.DefaultBus())
	now := time.Now().UTC()
	parent := store.RunRecord{
		ID: "run_p", TraceID: "trc_p", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: "finished", SpaceID: "local", ActorRole: "maintainer",
		Depth: 2, StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	_, err := svc.Spawn("run_p", SpawnRequest{
		Scenario: ScenarioRef{Name: "hotfix", ScenarioVersion: "1.0.0"},
		Inputs:   map[string]any{"issueOrSpec": "x", "repoRoot": "."},
	})
	if err == nil || err.Error() == "" {
		t.Fatal("expected depth rejection")
	}
	if !allowlistPermits(nil, "write") || allowlistPermits([]string{"read"}, "write") {
		t.Fatal("allowlist semantics")
	}
}
