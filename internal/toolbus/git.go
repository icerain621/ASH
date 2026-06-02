package toolbus

import (
	"fmt"
	"os/exec"
	"strings"
)

func registerGitTools(r *Registry) {
	r.Register("git.status", RiskSafe, gitStatus)
	r.Register("git.diff", RiskSafe, gitDiff)
	r.Register("git.checkout", RiskMedium, gitCheckout)
}

func gitStatus(ctx Context, _ map[string]any) (map[string]any, error) {
	root := ctx.RepoRoot
	if root == "" {
		if v, ok := ctx.Inputs["repoRoot"].(string); ok {
			root = v
		}
	}
	if root == "" {
		return map[string]any{"clean": true, "skipped": true}, nil
	}
	out, err := exec.Command("git", "-C", root, "status", "--porcelain").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	clean := len(strings.TrimSpace(string(out))) == 0
	return map[string]any{"clean": clean, "porcelain": strings.TrimSpace(string(out))}, nil
}

func gitDiff(ctx Context, _ map[string]any) (map[string]any, error) {
	root := ctx.RepoRoot
	if root == "" {
		return map[string]any{"diff": ""}, nil
	}
	out, err := exec.Command("git", "-C", root, "diff", "HEAD").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return map[string]any{"diff": string(out)}, nil
}

func gitCheckout(ctx Context, args map[string]any) (map[string]any, error) {
	root := ctx.RepoRoot
	if root == "" {
		return map[string]any{"ok": true, "skipped": true}, nil
	}
	branch, _ := args["branch"].(string)
	if branch == "" {
		if nb, ok := args["newBranchFrom"].(string); ok && nb != "" {
			suffix := ctx.RunID
			if strings.HasPrefix(suffix, "run_") {
				suffix = suffix[4:]
			}
			if len(suffix) > 8 {
				suffix = suffix[:8]
			}
			branch = "ash/" + suffix
			out, err := exec.Command("git", "-C", root, "checkout", "-b", branch, nb).CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("git checkout -b: %w (%s)", err, strings.TrimSpace(string(out)))
			}
			return map[string]any{"branch": branch, "from": nb}, nil
		}
		return map[string]any{"ok": true, "note": "no branch args"}, nil
	}
	out, err := exec.Command("git", "-C", root, "checkout", branch).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git checkout: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return map[string]any{"branch": branch}, nil
}
