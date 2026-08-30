package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/agentexec"
	"github.com/ash-repwiki/ash/internal/opsenv"
	"github.com/ash-repwiki/ash/internal/runs"
)

// AgentProviderStatus mirrors runs.AgentProviderStatus for swagger.
type AgentProviderStatus struct {
	Pinned           bool                         `json:"pinned"`
	PinnedAdapter    string                       `json:"pinnedAdapter,omitempty"`
	HarnessKind      string                       `json:"harnessKind,omitempty"`
	Selection        runs.ProviderSelectionDTO    `json:"selection"`
	ExecGo           agentexec.ProbeReport        `json:"execGo"`
	LiveGateHints    []string                     `json:"liveGateHints,omitempty"`
	ExecGoE2EEnabled bool                         `json:"execGoE2EEnabled"`
	LiveSmokeHint    string                       `json:"liveSmokeHint,omitempty"`
}

// GetAgentProviderStatus godoc
// @Summary Probe agent provider / ExecGo readiness
// @Tags providers
// @Produce json
// @Param spaceId query string false "space id" default(local)
// @Success 200 {object} AgentProviderStatus
// @Router /api/v1/providers/agent [get]
func (h *Handler) getAgentProviderStatus(c *gin.Context) {
	spaceID := c.DefaultQuery("spaceId", currentSpace(c))
	if spaceID == "" {
		spaceID = "local"
	}
	status := h.runs.WithContext(c.Request.Context()).ProviderStatus(
		spaceID,
		opsenv.LiveGateHints(),
		os.Getenv("ASH_EXECGO_E2E") == "1",
	)
	c.JSON(http.StatusOK, AgentProviderStatus{
		Pinned:           status.Pinned,
		PinnedAdapter:    status.PinnedAdapter,
		HarnessKind:      status.HarnessKind,
		Selection:        status.Selection,
		ExecGo:           status.ExecGo,
		LiveGateHints:    status.LiveGateHints,
		ExecGoE2EEnabled: status.ExecGoE2EEnabled,
		LiveSmokeHint:    status.LiveSmokeHint,
	})
}
