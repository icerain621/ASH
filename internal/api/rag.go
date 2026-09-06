package api

import (
	"errors"
	"net/http"
	"strings"

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
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permRAGIndex, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runsFor(c).RAG().Index(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RAG_INDEX_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RebuildRAGSymbols godoc
// @Summary Rebuild path/symbol Hybrid index
// @Tags rag
// @Accept json
// @Produce json
// @Param body body rag.RebuildSymbolsRequest true "rebuild request"
// @Success 200 {object} rag.RebuildSymbolsResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/rag/symbols/rebuild [post]
func (h *Handler) rebuildRAGSymbols(c *gin.Context) {
	var req rag.RebuildSymbolsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permRAGIndex, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runsFor(c).RAG().RebuildSymbols(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("RAG_SYMBOLS_REBUILD_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RAGLSPHover godoc
// @Summary LSP hover at a file position (RAG internal)
// @Tags rag
// @Accept json
// @Produce json
// @Param body body rag.LSPPositionQuery true "hover request"
// @Success 200 {object} rag.LSPHoverResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/rag/lsp/hover [post]
func (h *Handler) ragLSPHover(c *gin.Context) {
	var req rag.LSPPositionQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permRAGQuery, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runsFor(c).RAG().Hover(req)
	if err != nil {
		writeLSPQueryError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RAGLSPDefinition godoc
// @Summary LSP definition at a file position (RAG internal)
// @Tags rag
// @Accept json
// @Produce json
// @Param body body rag.LSPPositionQuery true "definition request"
// @Success 200 {object} rag.LSPDefinitionResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/rag/lsp/definition [post]
func (h *Handler) ragLSPDefinition(c *gin.Context) {
	var req rag.LSPPositionQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permRAGQuery, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runsFor(c).RAG().Definition(req)
	if err != nil {
		writeLSPQueryError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RAGLSPReferences godoc
// @Summary LSP references at a file position (RAG internal, bounded)
// @Tags rag
// @Accept json
// @Produce json
// @Param body body rag.LSPReferencesRequest true "references request"
// @Success 200 {object} rag.LSPReferencesResponse
// @Failure 400 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/rag/lsp/references [post]
func (h *Handler) ragLSPReferences(c *gin.Context) {
	var req rag.LSPReferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	space := firstNonEmptyAPI(req.SpaceID, currentSpace(c))
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permRAGQuery, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runsFor(c).RAG().References(req)
	if err != nil {
		writeLSPQueryError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func writeLSPQueryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, rag.ErrLSPUnsupported), errors.Is(err, rag.ErrLSPUnavailable):
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
	default:
		msg := err.Error()
		if strings.Contains(msg, "line must") || strings.Contains(msg, "character must") ||
			strings.Contains(msg, "path escapes") || strings.Contains(msg, "path is required") {
			c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", msg))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("RAG_LSP_FAILED", msg))
	}
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
	if !h.requireTargetSpace(c, space) {
		return
	}
	if !h.requirePermission(c, permRAGQuery, space) {
		return
	}
	req.SpaceID = space
	resp, err := h.runsFor(c).RAG().Query(req)
	if err != nil {
		if errors.Is(err, rag.ErrInvalidPrefer) {
			c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("RAG_QUERY_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetRAGProfile godoc
// @Summary RAG retrieval profile for current space
// @Tags rag
// @Produce json
// @Success 200 {object} rag.Profile
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/rag/profile [get]
func (h *Handler) getRAGProfile(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permRAGQuery, space) {
		return
	}
	c.JSON(http.StatusOK, h.runsFor(c).RAG().Profile(space))
}
