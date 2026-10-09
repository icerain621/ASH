package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/agenttpl"
	"github.com/ash-repwiki/ash/internal/contextpack"
	"github.com/ash-repwiki/ash/internal/plugins"
	"github.com/ash-repwiki/ash/internal/review"
	"github.com/ash-repwiki/ash/internal/review/passk"
)

// processReview is the in-process queue for agent/template/component gates (G02).
var processReview = review.NewService()

type submitAgentTemplateRequest struct {
	agenttpl.Manifest
}

type approveAgentTemplateResponse struct {
	Entry agenttpl.Entry `json:"entry"`
}

type agentComponentListResponse struct {
	Items []plugins.Component `json:"items"`
}

type registerAgentComponentRequest struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type submitAgentReviewRequest struct {
	Kind     string `json:"kind"`
	TargetID string `json:"targetId"`
	Reason   string `json:"reason"`
}

type decideAgentReviewRequest struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type passKRequest struct {
	K      int    `json:"k"`
	Target string `json:"target"`
}

type contextPackPreviewRequest struct {
	Issue    string              `json:"issue"`
	RAGRefs  []string            `json:"ragRefs"`
	Memories []contextpack.Hit   `json:"memories"`
	Memory   []string            `json:"memory"`
	Skills   []contextpack.SkillCard `json:"skills"`
}

// SubmitAgentTemplateCandidate godoc
// @Summary Submit a custom agent template as candidate
// @Tags agents
// @Accept json
// @Produce json
// @Param body body submitAgentTemplateRequest true "template manifest"
// @Success 200 {object} agenttpl.Entry
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/agent-templates/candidates [post]
func (h *Handler) submitAgentTemplateCandidate(c *gin.Context) {
	var req submitAgentTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	entry, err := agenttpl.SubmitCandidate(req.Manifest)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("AGENT_TEMPLATE_CANDIDATE_FAILED", err.Error()))
		return
	}
	_, _ = processReview.Submit(review.KindAgentTemplate, entry.Manifest.ID, "candidate submitted")
	c.JSON(http.StatusOK, entry)
}

// ApproveAgentTemplate godoc
// @Summary Approve a candidate agent template for production binding
// @Tags agents
// @Produce json
// @Param templateId path string true "template id"
// @Success 200 {object} approveAgentTemplateResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/agent-templates/{templateId}/approve [post]
func (h *Handler) approveAgentTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("templateId"))
	entry, err := agenttpl.ApproveTemplate(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("AGENT_TEMPLATE_APPROVE_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, approveAgentTemplateResponse{Entry: entry})
}

// ListAgentComponents godoc
// @Summary List in-process agent/memory plugin components
// @Tags agents
// @Produce json
// @Success 200 {object} agentComponentListResponse
// @Router /api/v1/agent-components [get]
func (h *Handler) listAgentComponents(c *gin.Context) {
	c.JSON(http.StatusOK, agentComponentListResponse{Items: plugins.Global().List()})
}

// RegisterAgentComponent godoc
// @Summary Register an in-process plugin component (defaults to candidate)
// @Tags agents
// @Accept json
// @Produce json
// @Param body body registerAgentComponentRequest true "component"
// @Success 200 {object} plugins.Component
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/agent-components [post]
func (h *Handler) registerAgentComponent(c *gin.Context) {
	var req registerAgentComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	comp := plugins.Component{ID: req.ID, Kind: req.Kind, Status: req.Status}
	if err := plugins.Global().Register(comp); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("AGENT_COMPONENT_REGISTER_FAILED", err.Error()))
		return
	}
	got, _ := plugins.Global().Get(req.ID)
	c.JSON(http.StatusOK, got)
}

// ApproveAgentComponent godoc
// @Summary Approve a candidate plugin component for production binding
// @Tags agents
// @Produce json
// @Param componentId path string true "component id"
// @Success 200 {object} plugins.Component
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/agent-components/{componentId}/approve [post]
func (h *Handler) approveAgentComponent(c *gin.Context) {
	id := strings.TrimSpace(c.Param("componentId"))
	if err := plugins.Global().Approve(id); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("AGENT_COMPONENT_APPROVE_FAILED", err.Error()))
		return
	}
	got, _ := plugins.Global().Get(id)
	c.JSON(http.StatusOK, got)
}

// SubmitAgentReview godoc
// @Summary Enqueue an in-process review item (template/harness/memory/artifact/run)
// @Tags reviews
// @Accept json
// @Produce json
// @Param body body submitAgentReviewRequest true "review target"
// @Success 200 {object} review.Item
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/agent-reviews [post]
func (h *Handler) submitAgentReview(c *gin.Context) {
	var req submitAgentReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	item, err := processReview.Submit(review.Kind(req.Kind), req.TargetID, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("AGENT_REVIEW_SUBMIT_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, item)
}

// DecideAgentReview godoc
// @Summary Decide an in-process review item; template/component approvals side-effect
// @Tags reviews
// @Accept json
// @Produce json
// @Param reviewId path string true "review id"
// @Param body body decideAgentReviewRequest true "decision"
// @Success 200 {object} review.Item
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/agent-reviews/{reviewId}/decide [post]
func (h *Handler) decideAgentReview(c *gin.Context) {
	var req decideAgentReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	item, err := processReview.Decide(c.Param("reviewId"), req.Decision, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("AGENT_REVIEW_DECIDE_FAILED", err.Error()))
		return
	}
	if item.Status == review.StatusApproved && item.Kind == review.KindAgentTemplate {
		_, _ = agenttpl.ApproveTemplate(item.TargetID)
	}
	c.JSON(http.StatusOK, item)
}

// RunPassK godoc
// @Summary Run Pass^k sampling for a frozen review check (not Doctor)
// @Tags reviews
// @Accept json
// @Produce json
// @Param body body passKRequest true "k and target"
// @Success 200 {object} passk.Result
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/passk/run [post]
func (h *Handler) runPassK(c *gin.Context) {
	var req passKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	target := strings.TrimSpace(req.Target)
	result, err := passk.Run(passk.Config{
		K: req.K,
		Check: func(i int) error {
			_ = i
			if target == "" {
				return nil
			}
			if _, err := agenttpl.GetProduction(target); err != nil {
				return err
			}
			return nil
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("PASSK_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, result)
}

// PreviewContextPack godoc
// @Summary Preview a Context Pack (issue + RAG + memory)
// @Tags agents
// @Accept json
// @Produce json
// @Param body body contextPackPreviewRequest true "pack input"
// @Success 200 {object} contextpack.Pack
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/context-pack/preview [post]
func (h *Handler) previewContextPack(c *gin.Context) {
	var req contextPackPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	pack, err := contextpack.Build(contextpack.Input{
		Issue: req.Issue, RAGRefs: req.RAGRefs, Memories: req.Memories,
		Memory: req.Memory, Skills: req.Skills,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("CONTEXT_PACK_PREVIEW_FAILED", err.Error()))
		return
	}
	c.JSON(http.StatusOK, pack)
}
