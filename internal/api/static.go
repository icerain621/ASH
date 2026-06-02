package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// corsMiddleware allows local UI dev (same-origin /ui or Vite proxy).
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Last-Event-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// registerWebUI serves console from frontend/dist (Vite build) or frontend/public fallback.
func registerWebUI(r *gin.Engine, webDir string) {
	if webDir == "" {
		return
	}
	if !filepath.IsAbs(webDir) {
		if wd, err := os.Getwd(); err == nil {
			webDir = filepath.Join(wd, webDir)
		}
	}
	if st, err := os.Stat(webDir); err != nil || !st.IsDir() {
		return
	}
	fs := http.Dir(webDir)
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui/")
	})
	r.StaticFS("/ui", fs)
}
