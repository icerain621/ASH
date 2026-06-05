package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
)

type ScaleReadinessResponse struct {
	SpaceID              string `json:"spaceId"`
	MemorySchemaVersion  int    `json:"memorySchemaVersion"`
	MemoryApprovedCount  int64  `json:"memoryApprovedCount"`
	RAGDocumentCount     int64  `json:"ragDocumentCount"`
	RAGChunkCount        int64  `json:"ragChunkCount"`
	ModelUsageRows       int64  `json:"modelUsageRows"`
	ModelCostMicrosTotal int64  `json:"modelCostMicrosTotal"`
	QualityMetricRows    int64  `json:"qualityMetricRows"`
	AuditLogRows         int64  `json:"auditLogRows"`
	DatabaseDialect       string `json:"databaseDialect"`
	PostgresConfigured    bool   `json:"postgresConfigured"`
	MigrationReady        bool   `json:"migrationReady"`
	SQLitePath            string `json:"sqlitePath,omitempty"`
	MigrationTableCount   int    `json:"migrationTableCount"`
	DualWriteEnabled      bool   `json:"dualWriteEnabled"`
	DualWriteRuntime      bool   `json:"dualWriteRuntime"`
	DualWriteSource       string `json:"dualWriteSource,omitempty"`
	LastMigrationSyncAtMs *int64 `json:"lastMigrationSyncAtMs,omitempty"`
}

// ScaleReadiness godoc
// @Summary TR3 scale readiness snapshot
// @Tags scale
// @Produce json
// @Success 200 {object} ScaleReadinessResponse
// @Failure 403 {object} APIErrorResponse
// @Router /api/v1/scale/readiness [get]
func (h *Handler) scaleReadiness(c *gin.Context) {
	space := currentSpace(c)
	if !h.requirePermission(c, permMemoryRead, space) {
		return
	}
	var memApproved int64
	_ = h.db.Model(&store.MemoryRecord{}).
		Where("space_id = ? AND status = ?", space, "approved").Count(&memApproved).Error
	var ragDocs, ragChunks int64
	_ = h.db.Model(&store.RAGDocument{}).Where("space_id = ?", space).Count(&ragDocs).Error
	_ = h.db.Model(&store.RAGChunk{}).Where("space_id = ?", space).Count(&ragChunks).Error
	var usageRows int64
	var costTotal int64
	_ = h.db.Model(&store.ModelUsage{}).Count(&usageRows).Error
	_ = h.db.Model(&store.ModelUsage{}).Select("COALESCE(SUM(cost_micros),0)").Scan(&costTotal).Error
	var qmRows int64
	_ = h.db.Model(&store.QualityMetric{}).Where("space_id = ?", space).Count(&qmRows).Error
	var auditRows int64
	_ = h.db.Model(&store.AuditLog{}).Where("space_id = ?", space).Count(&auditRows).Error
	dbProfile, _ := store.DatabaseProfile(h.db.DataDir(), os.Getenv("ASH_DATABASE_URL"))
	migSnap, _ := store.MigrationSnapshotFor(h.db.DataDir())
	var lastSyncMs *int64
	if migSnap.LastSyncAt != nil {
		ms := migSnap.LastSyncAt.UnixMilli()
		lastSyncMs = &ms
	}

	c.JSON(http.StatusOK, ScaleReadinessResponse{
		SpaceID:              space,
		MemorySchemaVersion:  memory.CurrentSchemaVersion,
		MemoryApprovedCount:  memApproved,
		RAGDocumentCount:     ragDocs,
		RAGChunkCount:        ragChunks,
		ModelUsageRows:       usageRows,
		ModelCostMicrosTotal: costTotal,
		QualityMetricRows:    qmRows,
		AuditLogRows:         auditRows,
		DatabaseDialect:       dbProfile.Dialect,
		PostgresConfigured:    dbProfile.PostgresConfigured,
		MigrationReady:        dbProfile.MigrationReady,
		SQLitePath:            migSnap.SQLitePath,
		MigrationTableCount:   migSnap.MigrationTableCount,
		DualWriteEnabled:      migSnap.DualWriteEnabled,
		DualWriteRuntime:      migSnap.DualWriteRuntime,
		DualWriteSource:       string(migSnap.DualWriteSource),
		LastMigrationSyncAtMs: lastSyncMs,
	})
}
