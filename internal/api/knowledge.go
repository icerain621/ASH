package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/knowledge"
)

type wikiListAPIResponse struct {
	Items []knowledge.WikiPage `json:"items"`
	Query string               `json:"query,omitempty"`
}

// GetRepoProfile godoc
// @Summary Build ephemeral repo profile
// @Tags knowledge
// @Produce json
// @Param repoRoot query string false "repository root" default(.)
// @Success 200 {object} knowledge.RepoProfile
// @Router /api/v1/repos/profile [get]
func (h *Handler) getRepoProfile(c *gin.Context) {
	root := c.DefaultQuery("repoRoot", ".")
	view, err := h.knowledge.Profile(root)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("REPO_PROFILE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, view)
}

// ListWikiPages godoc
// @Summary List ephemeral wiki page projections
// @Tags knowledge
// @Produce json
// @Param repoRoot query string false "repository root"
// @Param q query string false "search query"
// @Param limit query int false "max items"
// @Success 200 {object} wikiListAPIResponse
// @Router /api/v1/wiki/pages [get]
func (h *Handler) listWikiPages(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	out, err := h.knowledge.ListWikiPages(currentSpace(c), c.Query("repoRoot"), c.Query("q"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("WIKI_LIST_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, out)
}

// GetWikiPage godoc
// @Summary Get a wiki page projection
// @Tags knowledge
// @Produce json
// @Param pageId path string true "page id"
// @Param repoRoot query string false "repository root"
// @Success 200 {object} knowledge.WikiPage
// @Router /api/v1/wiki/pages/{pageId} [get]
func (h *Handler) getWikiPage(c *gin.Context) {
	view, err := h.knowledge.GetWikiPage(currentSpace(c), c.Param("pageId"), c.Query("repoRoot"))
	if err != nil {
		msg := err.Error()
		code := http.StatusNotFound
		if strings.Contains(msg, "required") {
			code = http.StatusBadRequest
		}
		c.JSON(code, errorBody("WIKI_GET_FAILED", msg))
		return
	}
	c.JSON(http.StatusOK, view)
}
