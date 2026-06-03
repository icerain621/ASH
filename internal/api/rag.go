package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rag"
)

// IndexRAG godoc
// @Summary Index repository for retrieval
// @Tags rag
// @Accept json
// @Produce json
// @Param body body rag.IndexRequest true "index request"
// @Success 200 {object} rag.IndexResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/rag/index [post]
func (h *Handler) indexRAG(c *gin.Context) {
	var req rag.IndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requirePermission(c, permRAGIndex, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runs.RAG().Index(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RAG_INDEX_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// QueryRAG godoc
// @Summary Query repository retrieval index
// @Tags rag
// @Accept json
// @Produce json
// @Param body body rag.QueryRequest true "query request"
// @Success 200 {object} rag.QueryResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/rag/query [post]
func (h *Handler) queryRAG(c *gin.Context) {
	var req rag.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requirePermission(c, permRAGQuery, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runs.RAG().Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RAG_QUERY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}
