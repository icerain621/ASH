package remote

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/sandbox"
)

func TestConfigFromEnvDisabledByDefault(t *testing.T) {
	t.Setenv(envRemoteEnabled, "")
	cfg := ConfigFromEnv()
	if cfg.Enabled {
		t.Fatal("expected remote disabled by default")
	}
	if Resolve() != nil {
		t.Fatal("Resolve want nil when disabled")
	}
	st := ProbeStatus()
	if st.Available || st.Enabled {
		t.Fatalf("status=%+v", st)
	}
}

func TestMockBackendAvailableAndDispatch(t *testing.T) {
	t.Setenv(envRemoteEnabled, "1")
	t.Setenv(envRemoteBackend, BackendMock)
	be := Resolve()
	if be == nil || !be.Available() || be.Name() != BackendMock {
		t.Fatalf("backend=%v", be)
	}
	st := ProbeStatus()
	if !st.Available || st.Backend != BackendMock {
		t.Fatalf("status=%+v", st)
	}

	dir := t.TempDir()
	prog := "echo"
	args := []string{"remote-ok"}
	if runtime.GOOS == "windows" {
		prog = "cmd"
		args = []string{"/C", "echo remote-ok"}
	}
	res, err := be.Dispatch(context.Background(), sandbox.DispatchRequest{
		Program:  prog,
		Args:     args,
		RepoRoot: dir,
		Timeout:  5 * time.Second,
		RunID:    "run_test",
		StepID:   "step_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.OK {
		t.Fatalf("res=%+v", res)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestE2BUnavailableWithoutCreds(t *testing.T) {
	t.Setenv(envRemoteEnabled, "1")
	t.Setenv(envRemoteBackend, BackendE2B)
	t.Setenv(envRemoteURL, "")
	t.Setenv(envRemoteAPIKey, "")
	be := Resolve()
	if be == nil || be.Available() {
		t.Fatalf("e2b without creds should be unavailable, got %+v available=%v", be, be != nil && be.Available())
	}
	_, err := be.Dispatch(context.Background(), sandbox.DispatchRequest{Program: "true"})
	if err == nil {
		t.Fatal("expected dispatch error")
	}
}

func TestUnknownBackendUnavailable(t *testing.T) {
	cfg := Config{Enabled: true, Backend: "weird", Timeout: time.Second}
	be := ResolveWith(cfg)
	if be.Available() {
		t.Fatal("expected unavailable")
	}
}
