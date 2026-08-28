package docker

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

const defaultImage = "ash-sandbox-runner:dev"

// Executor runs commands inside a short-lived Docker container.
type Executor struct {
	Image string
}

func (e Executor) image() string {
	if strings.TrimSpace(e.Image) != "" {
		return e.Image
	}
	if v := strings.TrimSpace(os.Getenv("ASH_SANDBOX_IMAGE")); v != "" {
		return v
	}
	return defaultImage
}

func (e Executor) Dispatch(ctx context.Context, req sandbox.DispatchRequest) (*sandbox.DispatchResult, error) {
	if !sandbox.DockerAvailable() {
		return &sandbox.DispatchResult{OK: false, Error: "docker unavailable or ASH_SKIP_SANDBOX set"}, nil
	}
	if req.Timeout <= 0 {
		req.Timeout = 60 * time.Second
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	root := strings.TrimSpace(req.RepoRoot)
	if root == "" {
		return &sandbox.DispatchResult{OK: false, Error: "repoRoot is required for docker sandbox"}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	program := strings.TrimSpace(req.Program)
	if program == "" {
		return &sandbox.DispatchResult{OK: false, Error: "program is required"}, nil
	}

	mountMode := "rw"
	if req.SandboxMode == sandbox.ModeReadOnly {
		mountMode = "ro"
	}
	args := []string{
		"run", "--rm",
		"--network", "none",
		"--memory", "256m",
		"-v", absRoot + ":/workspace:" + mountMode,
		"-w", "/workspace",
		"-e", "ASH_RUN_ID=" + req.RunID,
		"-e", "ASH_STEP_ID=" + req.StepID,
		e.image(),
		program,
	}
	args = append(args, req.Args...)

	cmd := exec.CommandContext(ctx, "docker", args...)
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
		if strings.Contains(res.Stderr, "Unable to find image") || strings.Contains(res.Error, "Unable to find image") {
			res.Error = fmt.Sprintf("image %s missing: %s", e.image(), res.Error)
		}
		return res, nil
	}
	res.OK = true
	return res, nil
}

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 4000 {
		return s
	}
	return s[:4000] + "\n...<truncated>"
}
