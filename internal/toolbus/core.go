package toolbus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func registerCoreTools(r *Registry) {
	r.Register("read", RiskSafe, coreRead)
	r.Register("write", RiskMedium, coreWrite)
	r.Register("edit", RiskMedium, coreEdit)
	r.Register("bash", RiskDanger, coreBash)
	r.Register("rag.query", RiskSafe, coreRAGQuery)
}

func coreRead(ctx Context, args map[string]any) (map[string]any, error) {
	path, err := resolveInRoot(ctx.RepoRoot, argString(args, "path"))
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": argString(args, "path"), "content": string(body)}, nil
}

func coreWrite(ctx Context, args map[string]any) (map[string]any, error) {
	path, err := resolveInRoot(ctx.RepoRoot, argString(args, "path"))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	content := argString(args, "content")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": argString(args, "path"), "bytes": len(content)}, nil
}

func coreEdit(ctx Context, args map[string]any) (map[string]any, error) {
	path, err := resolveInRoot(ctx.RepoRoot, argString(args, "path"))
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	old := argString(args, "old")
	if old == "" || !strings.Contains(string(body), old) {
		return nil, fmt.Errorf("edit old text not found")
	}
	next := strings.Replace(string(body), old, argString(args, "new"), 1)
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return nil, err
	}
	return map[string]any{"path": argString(args, "path"), "edited": true}, nil
}

func coreBash(ctx Context, args map[string]any) (map[string]any, error) {
	if approved, _ := args["approved"].(bool); !approved {
		return nil, fmt.Errorf("danger tool bash requires approval")
	}
	command := argString(args, "command")
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}
	root, err := filepath.Abs(ctx.RepoRoot)
	if err != nil || root == "" {
		return nil, fmt.Errorf("repo root required")
	}
	out, err := runBash(root, command)
	if err != nil {
		return map[string]any{"output": out}, err
	}
	return map[string]any{"output": out}, nil
}

func coreRAGQuery(ctx Context, args map[string]any) (map[string]any, error) {
	if ctx.RAGQuery == nil {
		return nil, fmt.Errorf("rag.query is not bound for this run")
	}
	text := argString(args, "text")
	if text == "" {
		text = argString(args, "query")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is required")
	}
	topK := 8
	if v, ok := args["topK"].(int); ok && v > 0 {
		topK = v
	} else if v, ok := args["topK"].(float64); ok && v > 0 {
		topK = int(v)
	}
	return ctx.RAGQuery(text, topK)
}

var runBash = func(dir, command string) (string, error) {
	cmd := exec.Command("bash", "-lc", command)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func resolveInRoot(root, rel string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("repo root required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel = strings.TrimSpace(rel)
	if rel == "" || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative")
	}
	full := filepath.Clean(filepath.Join(root, rel))
	back, err := filepath.Rel(root, full)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repo root")
	}
	return full, nil
}

func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, _ := args[key].(string)
	return v
}
