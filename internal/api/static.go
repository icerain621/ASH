package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	webDir = filepath.Clean(webDir)
	if st, err := os.Stat(webDir); err != nil || !st.IsDir() {
		return
	}
	indexPath := filepath.Join(webDir, "index.html")
	if st, err := os.Stat(indexPath); err != nil || st.IsDir() {
		return
	}
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui/")
	})
	r.GET("/ui", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/ui/")
	})
	r.GET("/ui/*filepath", func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		if rel != "" {
			target := filepath.Join(webDir, filepath.Clean(rel))
			if relToWeb, err := filepath.Rel(webDir, target); err == nil && relToWeb != ".." && !strings.HasPrefix(relToWeb, ".."+string(filepath.Separator)) {
				if st, err := os.Stat(target); err == nil && !st.IsDir() {
					http.ServeFile(c.Writer, c.Request, target)
					return
				}
			}
		}
		http.ServeFile(c.Writer, c.Request, indexPath)
	})
}
