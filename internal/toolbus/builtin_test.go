package toolbus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitStatus_skipsWithoutRepo(t *testing.T) {
	bus := DefaultBus()
	res := bus.Call(Context{Inputs: map[string]any{}}, CallRequest{Tool: "git.status"})
	if !res.OK {
		t.Fatalf("expected ok: %+v", res)
	}
	clean, _ := res.Output["clean"].(bool)
	if !clean {
		t.Fatal("expected clean when no repo")
	}
}

func TestApplyPatch_writesDiff(t *testing.T) {
	dir := t.TempDir()
	bus := DefaultBus()
	res := bus.Call(Context{
		RunID:  "run_test123",
		RunDir: dir,
		Inputs: map[string]any{"issueOrSpec": "demo"},
	}, CallRequest{Tool: "apply_patch"})
	if !res.OK {
		t.Fatalf("apply_patch failed: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "artifacts", "diff.patch")); err != nil {
		t.Fatal(err)
	}
}
