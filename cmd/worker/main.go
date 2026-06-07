//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/worker/main.go -o internal/api/docs --parseDependency --parseInternal

// ASH Worker HTTP API.
//
// @title ASH Worker API
// @version 0.1
// @description Delivery-oriented coding robot worker API (M0). See docs/appendices/G-OpenAPI-端点清单(M0).md.
//
// @contact.name ASH
// @license.name Proprietary
//
// @host localhost:8080
// @BasePath /
//
// @tag.name health description Health and readiness probes
// @tag.name runs description Run lifecycle and event stream
// @tag.name scenarios description Rules / scenario DSL
// @tag.name memory description Memory governance (candidate/review/query)
// @tag.name compliance description TR2 compliance scan and audit export
// @tag.name scale description TR3 scale readiness
// @tag.name improve description M1 self-improvement proposals
// @tag.name permissions description M2 RBAC and scenario tool matrix
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/api"
	_ "github.com/ash-repwiki/ash/internal/api/docs" // swag OpenAPI
	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/pluginabi"
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

func main() {
	cfg := config.Load()

	db, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if db.Dialect() == "sqlite" {
		if shadowURL, source := store.ResolveDualWritePostgresURL(cfg.DataDir); shadowURL != "" {
			log.Printf("dual-write active (source=%s, target=postgres)", source)
		}
	}

	scenariosDir := cfg.ScenariosDir
	if !filepath.IsAbs(scenariosDir) {
		if wd, err := os.Getwd(); err == nil {
			scenariosDir = filepath.Join(wd, scenariosDir)
		}
	}
	loader := rules.NewLoader(scenariosDir)
	if err := loader.LoadDir(); err != nil {
		log.Fatalf("load scenarios: %v", err)
	}
	log.Printf("loaded %d scenario(s) from %s", len(loader.List()), scenariosDir)

	gin.SetMode(gin.DebugMode)
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	h := api.NewHandler(db, loader)
	h.Register(r, cfg.WebDir)

	if cfg.PluginGRPCAddr != "" {
		startPluginGRPC(cfg.PluginGRPCAddr, db)
	}

	log.Printf("ASH worker listening on %s (data dir: %s)", cfg.HTTPAddr, cfg.DataDir)
	log.Printf("Swagger UI: http://%s/docs", trimLeadingColon(cfg.HTTPAddr))
	log.Printf("Web console: http://%s/ui/", trimLeadingColon(cfg.HTTPAddr))
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func startPluginGRPC(addr string, db *store.DB) {
	rt, err := pluginabi.StartRegistryServer(addr, db)
	if err != nil {
		log.Fatalf("plugin grpc listen %s: %v", addr, err)
	}
	go func() {
		if err := <-rt.Done(); err != nil {
			log.Fatalf("plugin grpc server: %v", err)
		}
	}()
	log.Printf("Plugin registry gRPC listening on %s", rt.Addr)
}

func trimLeadingColon(addr string) string {
	if len(addr) > 0 && addr[0] == ':' {
		return "localhost" + addr
	}
	return addr
}
