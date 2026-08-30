package spacerules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestBuiltinPickAndRoundTripFile(t *testing.T) {
	doc := BuiltinDocument()
	name, reason := PickScenario("fix CVE-2024", doc)
	if name != "security_patch" || reason == "" {
		t.Fatalf("got %s %s", name, reason)
	}
	name, reason = PickScenario("紧急热修线上", doc)
	if name != "hotfix" {
		t.Fatalf("got %s %s", name, reason)
	}

	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	put, err := svc.Put("local", PutRequest{Document: Document{
		Version: 1,
		Route: map[string][]string{
			"security_patch": {"custom-sec"},
			"hotfix":         {"custom-hot"},
		},
		Defaults: Defaults{PolicyProfile: "strict"},
	}, UpdatedBy: "tester"})
	if err != nil {
		t.Fatal(err)
	}
	if put.Builtin || put.Document.Defaults.PolicyProfile != "strict" {
		t.Fatalf("%+v", put)
	}

	dir := t.TempDir()
	if _, err := svc.ExportToFile("local", SyncRequest{RepoRoot: dir}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, RelativeFilePath)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	_ = db.Where("space_id = ?", "local").Delete(&store.SpaceRule{}).Error
	imp, err := svc.ImportFromFile("local", SyncRequest{RepoRoot: dir, UpdatedBy: "importer"})
	if err != nil {
		t.Fatal(err)
	}
	if imp.Source != "file" || imp.Document.Defaults.PolicyProfile != "strict" {
		t.Fatalf("%+v", imp)
	}
	n, r := PickScenario("please custom-sec now", imp.Document)
	if n != "security_patch" || !strings.Contains(r, "custom-sec") {
		t.Fatalf("%s %s", n, r)
	}
}
