package skills

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogSignAndVerify(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx28-cat-key")
	cat := Catalog{
		Schema: CatalogSchemaV1,
		Items: []CatalogItem{{
			Name: "demo", Version: "1.0.0", Publisher: "acme",
			URL: "packs/demo.zip", Digest: "sha256:abc",
		}},
	}
	cat.Signature = SignCatalogHMAC(PackSigningKey(), cat)
	if err := VerifyCatalogSignature(&cat); err != nil {
		t.Fatal(err)
	}
	cat.Signature = "00"
	if err := VerifyCatalogSignature(&cat); err == nil {
		t.Fatal("expected signature mismatch")
	}
}

func TestListCatalogFiltersAllowlist(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx28-cat-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "acme")
	t.Setenv(envCatalogURL, "")
	t.Setenv(envCatalogPath, "")

	repo := t.TempDir()
	ash := filepath.Join(repo, ".ash")
	if err := os.MkdirAll(ash, 0o755); err != nil {
		t.Fatal(err)
	}
	cat := Catalog{
		Schema: CatalogSchemaV1,
		Items: []CatalogItem{
			{Name: "a", Version: "1", Publisher: "acme", URL: "a.zip"},
			{Name: "b", Version: "1", Publisher: "other", URL: "b.zip"},
		},
	}
	raw, _ := json.Marshal(cat)
	if err := os.WriteFile(filepath.Join(ash, "skill-catalog.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := ListCatalog(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !out.OK || len(out.Items) != 1 || out.Items[0].Name != "a" {
		t.Fatalf("out=%+v", out)
	}
}

func TestInstallFromCatalogFileURL(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx28-install-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "*")
	t.Setenv("ASH_SKILL_PACK_SPACES", "*")
	t.Setenv(envCatalogURL, "")
	t.Setenv(envCatalogPath, "")

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: cat-pack\ndescription: from catalog\n---\n\n# c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipBytes, man, err := BuildPackZip(src, "local", "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	sig := SignPackHMAC(PackSigningKey(), man.Publisher, man.Name, man.Version, man.Digest)

	repo := t.TempDir()
	packs := filepath.Join(repo, ".ash", "packs")
	if err := os.MkdirAll(packs, 0o755); err != nil {
		t.Fatal(err)
	}
	packPath := filepath.Join(packs, "cat-pack.zip")
	if err := os.WriteFile(packPath, zipBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	cat := Catalog{
		Schema: CatalogSchemaV1,
		Items: []CatalogItem{{
			Name: man.Name, Version: man.Version, Publisher: man.Publisher,
			URL: "packs/cat-pack.zip", Digest: man.Digest, Signature: sig,
		}},
	}
	raw, _ := json.MarshalIndent(cat, "", "  ")
	if err := os.WriteFile(filepath.Join(repo, ".ash", "skill-catalog.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	inst, err := InstallFromCatalog(repo, "local", "cat-pack", "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if !inst.OK || inst.Name != "cat-pack" {
		t.Fatalf("%+v", inst)
	}
	if _, err := os.Stat(filepath.Join(repo, ".ash", "skills", "cat-pack", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallFromCatalogHTTP(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx28-http-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "*")
	t.Setenv("ASH_SKILL_PACK_SPACES", "*")

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: http-pack\ndescription: via http\n---\n\n# h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipBytes, man, err := BuildPackZip(src, "local", "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	sig := SignPackHMAC(PackSigningKey(), man.Publisher, man.Name, man.Version, man.Digest)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer srv.Close()

	repo := t.TempDir()
	cat := Catalog{
		Schema: CatalogSchemaV1,
		Items: []CatalogItem{{
			Name: man.Name, Version: man.Version, Publisher: man.Publisher,
			URL: srv.URL + "/pack.zip", Digest: man.Digest, Signature: sig,
		}},
	}
	raw, _ := json.Marshal(cat)
	path := filepath.Join(repo, "catalog.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCatalogPath, path)
	t.Setenv(envCatalogURL, "")

	inst, err := InstallFromCatalog(repo, "local", "http-pack", "")
	if err != nil {
		t.Fatal(err)
	}
	if !inst.OK {
		t.Fatalf("%+v", inst)
	}
}

func TestLoadCatalogMissingOK(t *testing.T) {
	t.Setenv(envCatalogURL, "")
	t.Setenv(envCatalogPath, "")
	repo := t.TempDir()
	out, err := ListCatalog(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !out.OK || len(out.Items) != 0 {
		t.Fatalf("%+v", out)
	}
}
