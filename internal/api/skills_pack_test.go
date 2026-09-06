package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/skills"
)

func TestSkillPackVerifyAndInstallAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx22-api-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "*")
	t.Setenv("ASH_SKILL_PACK_SPACES", "*")

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: api-pack\ndescription: from api\n---\n\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipBytes, man, err := skills.BuildPackZip(src, "local", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	sig := skills.SignPackHMAC(skills.PackSigningKey(), man.Publisher, man.Name, man.Version, man.Digest)

	r, _ := newPlatformTestRouter(t)
	repo := t.TempDir()

	body, _ := json.Marshal(map[string]string{
		"repoRoot":   repo,
		"spaceId":    "local",
		"packBase64": base64.StdEncoding.EncodeToString(zipBytes),
		"signature":  sig,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skills/packs/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/skills/packs/install", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("install status=%d body=%s", w2.Code, w2.Body.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".ash", "skills", "api-pack", "SKILL.md")); err != nil {
		t.Fatal(err)
	}

	// bad signature
	bad, _ := json.Marshal(map[string]string{
		"packBase64": base64.StdEncoding.EncodeToString(zipBytes),
		"signature":  "00",
	})
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/skills/packs/install", bytes.NewReader(bad))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	if w3.Code == http.StatusOK {
		t.Fatal("bad signature should fail")
	}
}

func TestSkillCatalogListAndInstallAPI(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx28-api-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "*")
	t.Setenv("ASH_SKILL_PACK_SPACES", "*")
	t.Setenv("ASH_SKILL_CATALOG_URL", "")
	t.Setenv("ASH_SKILL_CATALOG_PATH", "")

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: catalog-api\ndescription: catalog api\n---\n\n# y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipBytes, man, err := skills.BuildPackZip(src, "local", "2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	sig := skills.SignPackHMAC(skills.PackSigningKey(), man.Publisher, man.Name, man.Version, man.Digest)

	repo := t.TempDir()
	packs := filepath.Join(repo, ".ash", "packs")
	if err := os.MkdirAll(packs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packs, "catalog-api.zip"), zipBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	cat := skills.Catalog{
		Schema: skills.CatalogSchemaV1,
		Items: []skills.CatalogItem{{
			Name: man.Name, Version: man.Version, Publisher: man.Publisher,
			URL: "packs/catalog-api.zip", Digest: man.Digest, Signature: sig,
		}},
	}
	raw, _ := json.Marshal(cat)
	if err := os.WriteFile(filepath.Join(repo, ".ash", "skill-catalog.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	r, _ := newPlatformTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/catalog?repoRoot="+repo, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", w.Code, w.Body.String())
	}
	var listed skills.CatalogListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if !listed.OK || len(listed.Items) != 1 {
		t.Fatalf("listed=%+v", listed)
	}

	body, _ := json.Marshal(map[string]string{
		"repoRoot": repo,
		"spaceId":  "local",
		"name":     "catalog-api",
		"version":  "2.0.0",
	})
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/skills/catalog/install", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("install status=%d body=%s", w2.Code, w2.Body.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".ash", "skills", "catalog-api", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}
