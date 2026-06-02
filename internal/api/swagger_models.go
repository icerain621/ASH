package api

import (
	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/runs"
)

// APIError is the unified error shape (M0).
type APIError struct {
	Code    string `json:"code" example:"INVALID_REQUEST"`
	Message string `json:"message" example:"validation failed"`
}

// APIErrorResponse wraps APIError for OpenAPI.
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// HealthResponse for liveness/readiness probes.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
	Error  string `json:"error,omitempty"`
}

// RunListResponse lists runs.
type RunListResponse struct {
	Items []runs.Summary `json:"items"`
}

// ScenarioListResponse lists loaded scenarios.
type ScenarioListResponse struct {
	Items []rules.ScenarioSummary `json:"items"`
}

// ScenarioDetailResponse returns scenario document details.
type ScenarioDetailResponse struct {
	Version  string          `json:"version" example:"ash.rules/v0.1"`
	Scenario rules.Scenario  `json:"scenario"`
	Hooks    []rules.Hook    `json:"hooks,omitempty"`
	YAML     string          `json:"yaml,omitempty"`
	Valid    bool            `json:"valid,omitempty"`
}

// ValidateScenarioRequest validates DSL YAML in JSON body.
type ValidateScenarioRequest struct {
	YAML string `json:"yaml" binding:"required" example:"version: ash.rules/v0.1"`
}

// ValidationResponse is returned by scenario validate endpoints.
type ValidationResponse struct {
	OK     bool                    `json:"ok" example:"true"`
	Issues []rules.ValidationIssue `json:"issues,omitempty"`
}
