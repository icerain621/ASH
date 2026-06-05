package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveDualWritePostgresURLPrefersEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ASH_DUAL_WRITE_POSTGRES_URL", "postgres://env:env@127.0.0.1/ash")
	_ = SaveDualWriteConfig(dir, &DualWriteConfig{
		Enabled: true, PostgresURL: "postgres://file:file@127.0.0.1/ash",
	})
	url, source := ResolveDualWritePostgresURL(dir)
	if source != DualWriteSourceEnv || url != os.Getenv("ASH_DUAL_WRITE_POSTGRES_URL") {
		t.Fatalf("url=%q source=%q", url, source)
	}
}

func TestResolveDualWritePostgresURLFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ASH_DUAL_WRITE_POSTGRES_URL", "")
	cfg := &DualWriteConfig{
		Enabled: true, PostgresURL: "postgres://file:file@127.0.0.1/ash",
		SQLitePath: filepath.Join(dir, "ash.db"), EnabledAt: time.Now().UTC(),
	}
	if err := SaveDualWriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	url, source := ResolveDualWritePostgresURL(dir)
	if source != DualWriteSourceFile || url != cfg.PostgresURL {
		t.Fatalf("url=%q source=%q", url, source)
	}
}
