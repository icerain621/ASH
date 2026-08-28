package process

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

// Executor runs commands with cwd jailed to repoRoot.
type Executor struct{}

func (Executor) Dispatch(ctx context.Context, req sandbox.DispatchRequest) (*sandbox.DispatchResult, error) {
	if req.Timeout <= 0 {
		req.Timeout = 30 * time.Second
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	root := strings.TrimSpace(req.RepoRoot)
	if root == "" {
		root = os.TempDir()
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	program := strings.TrimSpace(req.Program)
	if program == "" {
		return &sandbox.DispatchResult{OK: false, Error: "program is required"}, nil
	}
	// Reject absolute program paths outside root (path jail for scripts).
	if filepath.IsAbs(program) {
		ok, err := sandbox.PathWithinRoot(absRoot, program)
		if err != nil {
			return nil, err
		}
		if !ok {
			return &sandbox.DispatchResult{OK: false, Error: "program path outside repoRoot"}, nil
		}
	}
	for _, a := range req.Args {
		if filepath.IsAbs(a) || strings.Contains(a, "..") {
			ok, err := sandbox.PathWithinRoot(absRoot, a)
			if err != nil {
				return nil, err
			}
			if !ok {
				return &sandbox.DispatchResult{OK: false, Error: fmt.Sprintf("arg path outside repoRoot: %s", a)}, nil
			}
		}
	}

	cmd := exec.CommandContext(ctx, program, req.Args...)
	cmd.Dir = absRoot
	cmd.Env = sanitizeEnv(req.Env, req.RunID, req.StepID)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	res := &sandbox.DispatchResult{
		Stdout: truncate(stdout.String()),
		Stderr: truncate(stderr.String()),
	}
	if err != nil {
		res.OK = false
		res.Error = err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res, nil
	}
	res.OK = true
	res.ExitCode = 0
	return res, nil
}

func sanitizeEnv(extra []string, runID, stepID string) []string {
	base := []string{
		"PATH=" + os.Getenv("PATH"),
		"ASH_RUN_ID=" + runID,
		"ASH_STEP_ID=" + stepID,
	}
	// Windows needs SystemRoot for many builtins.
	if v := os.Getenv("SystemRoot"); v != "" {
		base = append(base, "SystemRoot="+v)
	}
	if v := os.Getenv("SYSTEMROOT"); v != "" {
		base = append(base, "SYSTEMROOT="+v)
	}
	if v := os.Getenv("ComSpec"); v != "" {
		base = append(base, "ComSpec="+v)
	}
	out := append([]string{}, base...)
	for _, e := range extra {
		if strings.HasPrefix(e, "ASH_") || strings.HasPrefix(e, "PATH=") {
			out = append(out, e)
		}
	}
	return out
}

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 4000 {
		return s
	}
	return s[:4000] + "\n...<truncated>"
}
