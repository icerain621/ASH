package artifacts

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunsRootOverrideAndEnsureLayout(t *testing.T) {
	data := t.TempDir()
	override := filepath.Join(t.TempDir(), "custom-runs")
	t.Setenv("ASH_RUNS_DIR", override)
	if got := RunsRoot(data); got != filepath.Clean(override) {
		t.Fatalf("RunsRoot=%q want %q", got, override)
	}
	runDir := RunDir(data, "run_abc")
	if err := EnsureRunLayout(runDir); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"artifacts", "checkpoints", "audit"} {
		st, err := os.Stat(filepath.Join(runDir, sub))
		if err != nil || !st.IsDir() {
			t.Fatalf("missing %s: %v", sub, err)
		}
	}
	prof := DescribePaths(data)
	if !prof.RunsDirOverride || prof.Platform != runtime.GOOS {
		t.Fatalf("profile=%+v", prof)
	}
}

func TestEnforceArtifactsBudget(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASH_ARTIFACTS_MAX_BYTES", "2")
	if err := enforceArtifactsBudget(dir); err == nil {
		t.Fatal("expected budget error")
	}
	t.Setenv("ASH_ARTIFACTS_MAX_BYTES", "100")
	if err := enforceArtifactsBudget(dir); err != nil {
		t.Fatal(err)
	}
}
