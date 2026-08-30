package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/spacerules"
)

func TestSpaceRulesCRUDImportExportPreview(t *testing.T) {
	t.Setenv("ASH_AUTH_MODE", "dev")
	r, _ := newPlatformTestRouter(t)

	body, _ := json.Marshal(spacerules.PutRequest{
		Document: spacerules.Document{
			Version: 1,
			Route: map[string][]string{
				"security_patch": {"sec-token"},
				"hotfix":         {"hot-token"},
			},
			Defaults: spacerules.Defaults{PolicyProfile: "strict"},
		},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/spaces/local/rules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", w.Code, w.Body.String())
	}

	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, httptest.NewRequest(http.MethodGet, "/api/v1/spaces/local/rules", nil))
	if wGet.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", wGet.Code, wGet.Body.String())
	}

	dir := t.TempDir()
	expBody, _ := json.Marshal(spacerules.SyncRequest{RepoRoot: dir})
	wExp := httptest.NewRecorder()
	reqExp := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/local/rules/export", bytes.NewReader(expBody))
	reqExp.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wExp, reqExp)
	if wExp.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", wExp.Code, wExp.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, spacerules.RelativeFilePath)); err != nil {
		t.Fatal(err)
	}

	prevBody, _ := json.Marshal(spacerules.PreviewRequest{Goal: "please sec-token now", RepoRoot: "."})
	wPrev := httptest.NewRecorder()
	reqPrev := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/local/rules/preview", bytes.NewReader(prevBody))
	reqPrev.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wPrev, reqPrev)
	if wPrev.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", wPrev.Code, wPrev.Body.String())
	}
	var prev spacerules.PreviewResponse
	if err := json.Unmarshal(wPrev.Body.Bytes(), &prev); err != nil {
		t.Fatal(err)
	}
	if prev.ScenarioName != "security_patch" {
		t.Fatalf("preview=%+v", prev)
	}
}
