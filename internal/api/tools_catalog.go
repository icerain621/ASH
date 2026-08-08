package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/toolbus"
)

type toolRiskCatalogResponse struct {
	Items []toolbus.ToolRiskEntry `json:"items"`
	Doc   string                  `json:"docRef"`
}

// ListToolRiskCatalog godoc
// @Summary List built-in tool risk catalog (dangerous-ops)
// @Tags tools
// @Produce json
// @Success 200 {object} toolRiskCatalogResponse
// @Router /api/v1/tools/risk-catalog [get]
func (h *Handler) listToolRiskCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, toolRiskCatalogResponse{
		Items: toolbus.DefaultCatalog(),
		Doc:   "doc/design/ARCH-架构与技术选型.md §安全",
	})
}
