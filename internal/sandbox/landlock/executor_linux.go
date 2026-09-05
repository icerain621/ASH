//go:build linux

package landlock

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

// Executor runs a program after applying a Landlock FS ruleset scoped to RepoRoot.
// Restriction is applied in a short-lived re-exec child so the parent worker is not sandboxed.
type Executor struct{}

// Dispatch applies Landlock in a child process then exec's the target program.
// If Landlock cannot be applied, returns a non-nil error (caller maps to sandbox_failed).
func (Executor) Dispatch(ctx context.Context, req sandbox.DispatchRequest) (*sandbox.DispatchResult, error) {
	if !Available() {
		return nil, fmt.Errorf("landlock unavailable on this kernel")
	}
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
		return nil, fmt.Errorf("repoRoot is required for landlock sandbox")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	program := strings.TrimSpace(req.Program)
	if program == "" {
		return &sandbox.DispatchResult{OK: false, Error: "program is required"}, nil
	}
	progPath, err := resolveProgram(program, absRoot)
	if err != nil {
		return nil, err
	}

	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("landlock re-exec: %w", err)
	}
	childArgs := append([]string{childArgv0, absRoot, progPath}, req.Args...)
	cmd := exec.CommandContext(ctx, self, childArgs...)
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
		// Child exits 127 when landlock apply fails before exec.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 127 {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("landlock apply failed: %s", msg)
		}
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

func resolveProgram(program, absRoot string) (string, error) {
	if filepath.IsAbs(program) {
		return program, nil
	}
	if strings.ContainsRune(program, os.PathSeparator) {
		p := filepath.Join(absRoot, program)
		return p, nil
	}
	if p, err := exec.LookPath(program); err == nil {
		return p, nil
	}
	return program, nil
}

func sanitizeEnv(extra []string, runID, stepID string) []string {
	base := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"LANG=" + os.Getenv("LANG"),
		"ASH_RUN_ID=" + runID,
		"ASH_STEP_ID=" + stepID,
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
