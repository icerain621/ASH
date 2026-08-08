package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/config"
	"github.com/ash-repwiki/ash/internal/store"
)

type DataPolicyResponse struct {
	Policy         config.DataPolicy `json:"policy"`
	Classification []string          `json:"classification"`
	DocRef         string            `json:"docRef"`
}

type dataRetentionApplyRequest struct {
	SpaceID string `json:"spaceId,omitempty"`
	DryRun  bool   `json:"dryRun,omitempty"`
}

type DataRetentionApplyResponse struct {
	SpaceID       string    `json:"spaceId"`
	Domain        string    `json:"domain"`
	RetentionDays int       `json:"retentionDays"`
	MaxRuns       int       `json:"maxRuns,omitempty"`
	Cutoff        time.Time `json:"cutoff"`
	Matched       int64     `json:"matched"`
	Deleted       int64     `json:"deleted"`
	FilesRemoved  int64     `json:"filesRemoved,omitempty"`
	DryRun        bool      `json:"dryRun"`
}

// GetDataPolicy godoc
// @Summary Get effective data classification and retention policy
// @Tags compliance
// @Produce json
// @Success 200 {object} DataPolicyResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/data-policy [get]
func (h *Handler) getDataPolicy(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	c.JSON(http.StatusOK, DataPolicyResponse{
		Policy:         config.LoadDataPolicy(),
		Classification: []string{config.SensitivityNormal, config.SensitivityRestricted, config.SensitivitySecret},
		DocRef:         "doc/appendices/J-数据分级与保留期.md",
	})
}

// ApplyEventsRetention godoc
// @Summary Apply run_events retention for current space
// @Tags compliance
// @Accept json
// @Produce json
// @Param body body dataRetentionApplyRequest false "retention request"
// @Success 200 {object} DataRetentionApplyResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/events/retention/apply [post]
func (h *Handler) applyEventsRetention(c *gin.Context) {
	var req dataRetentionApplyRequest
	_ = c.ShouldBindJSON(&req)
	space := currentSpace(c)
	if strings.TrimSpace(req.SpaceID) != "" {
		if !h.requireTargetSpace(c, req.SpaceID) {
			return
		}
		space = strings.TrimSpace(req.SpaceID)
	}
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	days := config.EffectiveRetentionEventsDays()
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	db := h.dbFor(c)
	oldRuns := db.Model(&store.RunRecord{}).Select("id").Where("space_id = ? AND created_at < ?", space, cutoff)
	q := db.Model(&store.RunEvent{}).Where("run_id IN (?)", oldRuns)
	var matched int64
	if err := q.Count(&matched).Error; err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("EVENTS_RETENTION_COUNT_FAILED", err.Error()))
		return
	}
	deleted := int64(0)
	if !req.DryRun && matched > 0 {
		res := db.Where("run_id IN (?)", oldRuns).Delete(&store.RunEvent{})
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, errorBody("EVENTS_RETENTION_APPLY_FAILED", res.Error.Error()))
			return
		}
		deleted = res.RowsAffected
	}
	resp := DataRetentionApplyResponse{
		SpaceID: space, Domain: "events", RetentionDays: days, Cutoff: cutoff,
		Matched: matched, Deleted: deleted, DryRun: req.DryRun,
	}
	_ = db.Create(auditRow(space, currentActor(c), "events.retention_applied", resp)).Error
	c.JSON(http.StatusOK, resp)
}

// ApplyArtifactsRetention godoc
// @Summary Apply artifact_index retention for current space
// @Tags compliance
// @Accept json
// @Produce json
// @Param body body dataRetentionApplyRequest false "retention request"
// @Success 200 {object} DataRetentionApplyResponse
// @Failure 403 {object} APIErrorResponse
// @Failure 500 {object} APIErrorResponse
// @Router /api/v1/artifacts/retention/apply [post]
func (h *Handler) applyArtifactsRetention(c *gin.Context) {
	var req dataRetentionApplyRequest
	_ = c.ShouldBindJSON(&req)
	space := currentSpace(c)
	if strings.TrimSpace(req.SpaceID) != "" {
		if !h.requireTargetSpace(c, req.SpaceID) {
			return
		}
		space = strings.TrimSpace(req.SpaceID)
	}
	if !h.requirePermission(c, permAuditExport, space) {
		return
	}
	days := config.EffectiveRetentionArtifactsDays()
	maxRuns := config.EffectiveRetentionArtifactsMaxRuns()
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	db := h.dbFor(c)

	expireIDs, err := artifactRetentionRunIDs(db, space, cutoff, maxRuns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("ARTIFACTS_RETENTION_COUNT_FAILED", err.Error()))
		return
	}
	var matched int64
	if len(expireIDs) > 0 {
		if err := db.Model(&store.ArtifactIndex{}).Where("run_id IN ?", expireIDs).Count(&matched).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("ARTIFACTS_RETENTION_COUNT_FAILED", err.Error()))
			return
		}
	}
	deleted := int64(0)
	filesRemoved := int64(0)
	if !req.DryRun && matched > 0 {
		var rows []store.ArtifactIndex
		if err := db.Where("run_id IN ?", expireIDs).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errorBody("ARTIFACTS_RETENTION_APPLY_FAILED", err.Error()))
			return
		}
		res := db.Where("run_id IN ?", expireIDs).Delete(&store.ArtifactIndex{})
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, errorBody("ARTIFACTS_RETENTION_APPLY_FAILED", res.Error.Error()))
			return
		}
		deleted = res.RowsAffected
		filesRemoved = removeArtifactFiles(h.db.DataDir(), rows)
	}
	resp := DataRetentionApplyResponse{
		SpaceID: space, Domain: "artifacts", RetentionDays: days, MaxRuns: maxRuns, Cutoff: cutoff,
		Matched: matched, Deleted: deleted, FilesRemoved: filesRemoved, DryRun: req.DryRun,
	}
	_ = db.Create(auditRow(space, currentActor(c), "artifacts.retention_applied", resp)).Error
	c.JSON(http.StatusOK, resp)
}

func artifactRetentionRunIDs(db *gorm.DB, space string, cutoff time.Time, maxRuns int) ([]string, error) {
	var byAge []string
	if err := db.Model(&store.RunRecord{}).
		Where("space_id = ? AND created_at < ?", space, cutoff).
		Pluck("id", &byAge).Error; err != nil {
		return nil, err
	}
	var keep []string
	if err := db.Model(&store.RunRecord{}).
		Where("space_id = ?", space).
		Order("created_at desc").
		Limit(maxRuns).
		Pluck("id", &keep).Error; err != nil {
		return nil, err
	}
	keepSet := make(map[string]struct{}, len(keep))
	for _, id := range keep {
		keepSet[id] = struct{}{}
	}
	var all []string
	if err := db.Model(&store.RunRecord{}).Where("space_id = ?", space).Pluck("id", &all).Error; err != nil {
		return nil, err
	}
	byMax := make([]string, 0)
	for _, id := range all {
		if _, ok := keepSet[id]; !ok {
			byMax = append(byMax, id)
		}
	}
	seen := make(map[string]struct{}, len(byAge)+len(byMax))
	out := make([]string, 0, len(byAge)+len(byMax))
	for _, id := range append(byAge, byMax...) {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func removeArtifactFiles(dataDir string, rows []store.ArtifactIndex) int64 {
	root := filepath.Join(dataDir, "object_store")
	var n int64
	for _, row := range rows {
		key := strings.TrimSpace(row.StoreKey)
		if key == "" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(key))
		if err := os.Remove(path); err == nil {
			n++
		}
	}
	return n
}
