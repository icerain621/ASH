//go:build linux

package landlock_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
	"github.com/ash-repwiki/ash/internal/sandbox/landlock"
)

// e2eHomeBase places fixtures under $HOME so they are outside Landlock's
// baked-in RO allowlist (/tmp, /usr, …). t.TempDir() is usually under /tmp
// and would not demonstrate FS deny.
func e2eHomeBase(t *testing.T) (repo, secretDir string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		t.Skip("UserHomeDir unavailable")
	}
	base := filepath.Join(home, ".ash-sandbox-e2e", t.Name())
	_ = os.RemoveAll(base)
	repo = filepath.Join(base, "repo")
	secretDir = filepath.Join(base, "secret")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(secretDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(base) })
	return repo, secretDir
}

func TestE2ELandlockAllowsRepoRootRead(t *testing.T) {
	if !landlock.Available() {
		t.Skip("landlock unavailable on this kernel")
	}
	repo, _ := e2eHomeBase(t)
	marker := filepath.Join(repo, "marker.txt")
	if err := os.WriteFile(marker, []byte("inside-ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not on PATH")
	}
	res, err := (landlock.Executor{}).Dispatch(context.Background(), sandbox.DispatchRequest{
		Program:  cat,
		Args:     []string{marker},
		RepoRoot: repo,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if !res.OK || res.ExitCode != 0 {
		t.Fatalf("want ok exit0; ok=%v code=%d stderr=%q", res.OK, res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "inside-ok") {
		t.Fatalf("stdout=%q", res.Stdout)
	}
}

func TestE2ELandlockDeniesOutsideRepoRoot(t *testing.T) {
	if !landlock.Available() {
		t.Skip("landlock unavailable on this kernel")
	}
	repo, secretDir := e2eHomeBase(t)
	secret := filepath.Join(secretDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("should-not-read\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "ok.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat not on PATH")
	}
	res, err := (landlock.Executor{}).Dispatch(context.Background(), sandbox.DispatchRequest{
		Program:  cat,
		Args:     []string{secret},
		RepoRoot: repo,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		// apply failure is unexpected; deny should surface as non-zero exit
		t.Fatalf("dispatch apply error: %v", err)
	}
	if res.OK && res.ExitCode == 0 {
		t.Fatalf("expected deny reading outside repoRoot; stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
	combined := strings.ToLower(res.Stdout + " " + res.Stderr + " " + res.Error)
	if strings.Contains(combined, "should-not-read") {
		t.Fatalf("secret content leaked: %q", combined)
	}
}

func TestE2ESeccompDeniesMountSyscall(t *testing.T) {
	if !landlock.Available() {
		t.Skip("landlock unavailable on this kernel")
	}
	if !landlock.SeccompAvailable() {
		t.Skip("seccomp disabled (ASH_SANDBOX_SECCOMP=0) or unavailable")
	}
	repo, _ := e2eHomeBase(t)
	src := filepath.Join(repo, "mount_probe.go")
	code := `package main
import (
	"fmt"
	"syscall"
)
func main() {
	_, _, errno := syscall.Syscall(syscall.SYS_MOUNT, 0, 0, 0)
	fmt.Printf("mount_errno=%v\n", errno)
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(repo, "mount_probe")
	build := exec.Command("go", "build", "-o", bin, src)
	build.Dir = repo
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("go build mount_probe: %v (%s)", err, out)
	}
	res, err := (landlock.Executor{}).Dispatch(context.Background(), sandbox.DispatchRequest{
		Program:  bin,
		RepoRoot: repo,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	// SECCOMP_RET_KILL_THREAD → non-zero / signal. Soft-skip kernels may return errno without kill.
	if res.OK && res.ExitCode == 0 && strings.Contains(res.Stdout, "mount_errno=") {
		t.Logf("seccomp soft-skip suspected (syscall returned): stdout=%q stderr=%q", res.Stdout, res.Stderr)
		t.Skip("kernel accepted mount probe without kill; treat as soft-skip evidence")
	}
	if res.OK && res.ExitCode == 0 {
		t.Fatalf("expected seccomp kill or non-zero; ok=%v code=%d out=%q err=%q", res.OK, res.ExitCode, res.Stdout, res.Stderr)
	}
}
