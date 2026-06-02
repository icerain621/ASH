package toolbus

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func registerBuiltinTools(r *Registry) {
	registerGitTools(r)
	r.Register("apply_patch", RiskMedium, applyPatch)
	r.Register("test.run", RiskSafe, testRun)
}

// DefaultRegistry returns M0 native tools.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	registerBuiltinTools(r)
	return r
}

// DefaultBus returns a bus with M0 tools registered.
func DefaultBus() *Bus {
	return NewBus(DefaultRegistry())
}

func applyPatch(ctx Context, args map[string]any) (map[string]any, error) {
	artDir := filepath.Join(ctx.RunDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		return nil, err
	}
	patch := fmt.Sprintf("# M0 stub patch for run %s\n# issue: %v\n", ctx.RunID, ctx.Inputs["issueOrSpec"])
	if p, ok := args["patch"].(string); ok && p != "" {
		patch = p
	}
	path := filepath.Join(artDir, "diff.patch")
	if err := os.WriteFile(path, []byte(normalizeLF(patch)), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "bytes": len(patch)}, nil
}

func testRun(ctx Context, args map[string]any) (map[string]any, error) {
	root := ctx.RepoRoot
	if root == "" {
		if v, ok := ctx.Inputs["repoRoot"].(string); ok {
			root = v
		}
	}
	scope, _ := args["scope"].(string)
	report := map[string]any{
		"scope":     scope,
		"startedAt": time.Now().UTC().UnixMilli(),
		"ok":        true,
		"tests":     []any{},
		"summary":   map[string]any{"passed": 0, "failed": 0, "skipped": 0},
	}

	if root != "" {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil && scope == "changed" {
			cmd := exec.Command("go", "test", "./...", "-count=1", "-json")
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			report["rawLog"] = string(out)
			if err != nil {
				report["ok"] = false
				report["error"] = err.Error()
			} else {
				passed := strings.Count(string(out), `"Action":"pass"`)
				report["summary"] = map[string]any{"passed": passed, "failed": 0, "skipped": 0}
			}
		}
	}

	artDir := filepath.Join(ctx.RunDir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		return nil, err
	}
	b, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(artDir, "test_report.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": path, "ok": report["ok"]}, nil
}

func normalizeLF(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
