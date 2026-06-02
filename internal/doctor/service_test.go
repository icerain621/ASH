package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
	"github.com/ash-repwiki/ash/internal/toolbus"
)

func TestTR0Suite(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	scenariosDir := filepath.Join("..", "..", "scenarios")
	if _, err := os.Stat(scenariosDir); err != nil {
		scenariosDir = filepath.Join("scenarios")
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	ev := events.NewService(db)
	runsSvc := runs.NewService(db, ev, loader, toolbus.DefaultBus())
	svc := NewService(runsSvc, ev, loader, dir)

	rep, err := svc.RunSuite("TR0")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Fail > 0 {
		for _, r := range rep.Results {
			if r.Status != "pass" {
				t.Errorf("%s: %s", r.ID, r.Message)
			}
		}
		t.Fatalf("TR0 failed: pass=%d fail=%d", rep.Summary.Pass, rep.Summary.Fail)
	}
}
