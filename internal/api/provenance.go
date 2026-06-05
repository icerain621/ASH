package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/runs"
	"github.com/ash-repwiki/ash/internal/store"
)

type ProvenanceResponse struct {
	RunID      string              `json:"runId"`
	TraceID    string              `json:"traceId"`
	Scenario   runs.ScenarioRef    `json:"scenario"`
	Status     string              `json:"status"`
	ToolCalls  int                 `json:"toolCalls"`
	AgentTasks int                 `json:"agentTasks"`
	Artifacts  int                 `json:"artifacts"`
	Events     int                 `json:"events"`
	ModelUsage int                 `json:"modelUsage"`
	Links      []ProvenanceLink    `json:"links"`
}

type ProvenanceLink struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// GetRunProvenance godoc
// @Summary Delivery provenance chain for a run
// @Tags runs
// @Produce json
// @Param runId path string true "run id"
// @Success 200 {object} ProvenanceResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/runs/{runId}/provenance [get]
func (h *Handler) getRunProvenance(c *gin.Context) {
	runID := c.Param("runId")
	if !h.requireRunPermission(c, runID, permArtifactRead) {
		return
	}
	var rec store.RunRecord
	if err := h.db.First(&rec, "id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, errorBody("RUN_NOT_FOUND", runs.ErrRunNotFound.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorBody("RUN_LOOKUP_FAILED", err.Error()))
		return
	}
	var toolCount, agentCount, eventCount, usageCount int64
	_ = h.db.Model(&store.ToolCall{}).Where("run_id = ?", runID).Count(&toolCount).Error
	_ = h.db.Model(&store.AgentTask{}).Where("run_id = ?", runID).Count(&agentCount).Error
	_ = h.db.Model(&store.RunEvent{}).Where("run_id = ?", runID).Count(&eventCount).Error
	_ = h.db.Model(&store.ModelUsage{}).Where("run_id = ?", runID).Count(&usageCount).Error
	manifest, _ := h.runs.Artifacts(runID)
	artifactCount := 0
	if manifest != nil {
		artifactCount = len(manifest.Artifacts)
	}
	links := []ProvenanceLink{
		{Kind: "run", Ref: runID},
		{Kind: "trace", Ref: rec.TraceID},
		{Kind: "events", Ref: "run_events:count=" + itoa(int(eventCount))},
	}
	if artifactCount > 0 {
		links = append(links, ProvenanceLink{Kind: "artifacts", Ref: "manifest:" + runID})
	}
	if toolCount > 0 {
		links = append(links, ProvenanceLink{Kind: "toolCalls", Ref: "count=" + itoa(int(toolCount))})
	}
	c.JSON(http.StatusOK, ProvenanceResponse{
		RunID:   runID,
		TraceID: rec.TraceID,
		Scenario: runs.ScenarioRef{
			Name: rec.ScenarioName, ScenarioVersion: rec.ScenarioVersion,
		},
		Status:     rec.Status,
		ToolCalls:  int(toolCount),
		AgentTasks: int(agentCount),
		Artifacts:  artifactCount,
		Events:     int(eventCount),
		ModelUsage: int(usageCount),
		Links:      links,
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
