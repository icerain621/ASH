package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/config"
)

// RuntimePreflight godoc
// @Summary Check runtime readiness for agent execution
// @Tags runtime
// @Produce json
// @Param agent query string false "agent executor"
// @Success 200 {object} agentexec.PreflightResult
// @Router /api/v1/runtime/preflight [get]
func (h *Handler) runtimePreflight(c *gin.Context) {
	cfg := config.Load()
	agent := strings.TrimSpace(c.Query("agent"))
	if agent == "" {
		agent = cfg.AgentExecutor
	}
	switch agent {
	case "static":
		c.JSON(http.StatusOK, agentexec.StaticPreflight(cfg.AgentExecutor))
	default:
		c.JSON(http.StatusOK, agentexec.ExecGoCodexPreflight(c.Request.Context(), cfg.AgentExecutor))
	}
}
