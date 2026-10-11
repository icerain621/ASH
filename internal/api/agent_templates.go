package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/agenttpl"
)

// ListAgentTemplates godoc
// @Summary List built-in and custom agent templates with review status
// @Tags agents
// @Produce json
// @Success 200 {object} agenttpl.EntryListResponse
// @Router /api/v1/agent-templates [get]
func (h *Handler) listAgentTemplates(c *gin.Context) {
	out, err := agenttpl.ListEntries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("AGENT_TEMPLATE_LIST_FAILED", err.Error()))
		return
	}
	if h.db != nil {
		rows, err := h.db.ListAgentTemplateVersions(currentSpace(c))
		if err == nil {
			seen := map[string]struct{}{}
			for _, item := range out.Items {
				seen[item.Manifest.ID] = struct{}{}
			}
			for _, row := range rows {
				if _, ok := seen[row.TemplateID]; ok {
					continue
				}
				m, err := agenttpl.ManifestFromJSON(row.ManifestJSON)
				if err != nil {
					m = agenttpl.Manifest{ID: row.TemplateID, Version: row.Version}
				}
				out.Items = append(out.Items, agenttpl.Entry{Manifest: m, Status: row.Status})
				seen[row.TemplateID] = struct{}{}
			}
		}
	}
	c.JSON(http.StatusOK, out)
}
