package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/rules"
)

// ListScenarios godoc
// @Summary List loaded scenarios
// @Tags scenarios
// @Produce json
// @Success 200 {object} ScenarioListResponse
// @Router /api/v1/scenarios [get]
func (h *Handler) listScenarios(c *gin.Context) {
	c.JSON(http.StatusOK, ScenarioListResponse{Items: h.scenarios.List()})
}

// GetScenario godoc
// @Summary Get scenario by name and version
// @Tags scenarios
// @Produce json
// @Param name path string true "scenario name"
// @Param version path string true "scenario version"
// @Success 200 {object} ScenarioDetailResponse
// @Failure 404 {object} APIErrorResponse
// @Router /api/v1/scenarios/{name}/{version} [get]
func (h *Handler) getScenario(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")
	doc, err := h.scenarios.Get(name, version)
	if err != nil {
		c.JSON(http.StatusNotFound, errorBody("SCENARIO_NOT_FOUND", err.Error()))
		return
	}
	raw, err := h.scenarios.RawYAML(name, version)
	if err != nil {
		c.JSON(http.StatusOK, ScenarioDetailResponse{
			Scenario: doc.Scenario,
			Hooks:    doc.Hooks,
			Version:  doc.Version,
		})
		return
	}
	c.JSON(http.StatusOK, ScenarioDetailResponse{
		Version:  doc.Version,
		Scenario: doc.Scenario,
		Hooks:    doc.Hooks,
		YAML:     string(raw),
		Valid:    true,
	})
}

// ValidateScenario godoc
// @Summary Validate scenario DSL (JSON body)
// @Tags scenarios
// @Accept json
// @Produce json
// @Param body body ValidateScenarioRequest true "YAML text"
// @Success 200 {object} ValidationResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/scenarios/validate [post]
func (h *Handler) validateScenario(c *gin.Context) {
	var body ValidateScenarioRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	res := h.scenarios.ValidateYAML([]byte(body.YAML))
	c.JSON(http.StatusOK, ValidationResponse{
		OK:     res.OK,
		Issues: res.Issues,
	})
}

// ValidateScenarioRaw godoc
// @Summary Validate scenario DSL (raw YAML body)
// @Tags scenarios
// @Accept application/x-yaml
// @Accept text/plain
// @Produce json
// @Param body body string true "YAML document"
// @Success 200 {object} ValidationResponse
// @Failure 400 {object} APIErrorResponse
// @Router /api/v1/scenarios/validate/raw [post]
func (h *Handler) validateScenarioRaw(c *gin.Context) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorBody("INVALID_REQUEST", err.Error()))
		return
	}
	res := rules.ParseAndValidate(raw)
	c.JSON(http.StatusOK, ValidationResponse{
		OK:     res.OK,
		Issues: res.Issues,
	})
}
