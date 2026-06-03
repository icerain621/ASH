package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterWebUIServesSPAFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("spa shell"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.Mkdir(filepath.Join(webDir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "assets", "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	r := gin.New()
	registerWebUI(r, webDir)

	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{name: "index", path: "/ui/", want: "spa shell"},
		{name: "route fallback", path: "/ui/runs", want: "spa shell"},
		{name: "asset", path: "/ui/assets/app.js", want: "console.log('ok')"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if got := w.Body.String(); !strings.Contains(got, tc.want) {
				t.Fatalf("body = %q, want contains %q", got, tc.want)
			}
		})
	}
}
