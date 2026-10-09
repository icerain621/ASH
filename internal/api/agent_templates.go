package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/agenttpl"
)

// ListAgentTemplates godoc
// @Summary List built-in agent templates
// @Tags agents
// @Produce json
// @Success 200 {object} agenttpl.ListResponse
// @Router /api/v1/agent-templates [get]
func (h *Handler) listAgentTemplates(c *gin.Context) {
	out, err := agenttpl.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AGENT_TEMPLATE_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}
