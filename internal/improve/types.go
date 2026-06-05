package improve

import "time"

type CreateProposalRequest struct {
	Title         string `json:"title" binding:"required"`
	Description   string `json:"description"`
	BaselineRunID string `json:"baselineRunId" binding:"required"`
	ChangeSummary string `json:"changeSummary"`
	SpaceID       string `json:"spaceId,omitempty"`
	ActorID       string `json:"actorId,omitempty"`
}

type ProposalView struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Description     string          `json:"description,omitempty"`
	BaselineRunID   string          `json:"baselineRunId"`
	ExperimentRunID string          `json:"experimentRunId,omitempty"`
	Status          string          `json:"status"`
	ChangeSummary   string          `json:"changeSummary,omitempty"`
	CanaryPercent   int             `json:"canaryPercent"`
	Compare         *ArtifactCompare `json:"compare,omitempty"`
	CreatedAt       int64           `json:"createdAt"`
	UpdatedAt       int64           `json:"updatedAt"`
}

type ArtifactCompare struct {
	BaselineRunID   string            `json:"baselineRunId"`
	ExperimentRunID string            `json:"experimentRunId"`
	Matched         int               `json:"matched"`
	Changed         int               `json:"changed"`
	Missing         int               `json:"missing"`
	ByType          map[string]string `json:"byType,omitempty"`
}

type ListProposalsResponse struct {
	Items []ProposalView `json:"items"`
}

type StartExperimentResponse struct {
	ProposalID      string          `json:"proposalId"`
	ExperimentRunID string          `json:"experimentRunId"`
	Compare         *ArtifactCompare `json:"compare,omitempty"`
}

type CanaryRequest struct {
	Percent int    `json:"percent" binding:"required"`
	ActorID string `json:"actorId,omitempty"`
}

type StatusResponse struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}

func ms(t time.Time) int64 { return t.UTC().UnixMilli() }
