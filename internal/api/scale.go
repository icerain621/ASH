package api

import (
	"net/http"

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
	LastMigrationSyncAtMs      *int64  `json:"lastMigrationSyncAtMs,omitempty"`
	LastMigrationSyncError     string  `json:"lastMigrationSyncError,omitempty"`
	LastMigrationSyncErrorAtMs *int64  `json:"lastMigrationSyncErrorAtMs,omitempty"`
	PostgresRLSEnabled    bool   `json:"postgresRLSEnabled"`
	PostgresRLSForce      bool   `json:"postgresRLSForce"`
	PostgresRLSPolicyCount int64 `json:"postgresRLSPolicyCount,omitempty"`
	PostgresAppURLConfigured bool   `json:"postgresAppUrlConfigured,omitempty"`
	WorkerConnectionRole     string `json:"workerConnectionRole,omitempty"`
	RuntimeDSNHint           string `json:"runtimeDsnHint,omitempty"`
	DualWriteShadowURLHint   string `json:"dualWriteShadowUrlHint,omitempty"`
	SchemaMode               string `json:"schemaMode,omitempty"`
	SQLMigrationsEnabled     bool   `json:"sqlMigrationsEnabled,omitempty"`
	AutoMigrateEnabled       bool   `json:"autoMigrateEnabled,omitempty"`
	SQLMigrationVersion      uint   `json:"sqlMigrationVersion,omitempty"`
	SQLMigrationExpected     uint   `json:"sqlMigrationExpected,omitempty"`
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
	db := h.dbFor(c)
	var memApproved int64
	_ = db.Model(&store.MemoryRecord{}).
		Where("space_id = ? AND status = ?", space, "approved").Count(&memApproved).Error
	var ragDocs, ragChunks int64
	_ = db.Model(&store.RAGDocument{}).Where("space_id = ?", space).Count(&ragDocs).Error
	_ = db.Model(&store.RAGChunk{}).Where("space_id = ?", space).Count(&ragChunks).Error
	var usageRows int64
	var costTotal int64
	_ = db.Model(&store.ModelUsage{}).Count(&usageRows).Error
	_ = db.Model(&store.ModelUsage{}).Select("COALESCE(SUM(cost_micros),0)").Scan(&costTotal).Error
	var qmRows int64
	_ = db.Model(&store.QualityMetric{}).Where("space_id = ?", space).Count(&qmRows).Error
	var auditRows int64
	_ = db.Model(&store.AuditLog{}).Where("space_id = ?", space).Count(&auditRows).Error
	dbProfile, _ := store.DatabaseProfile(h.db.DataDir(), store.RuntimeDatabaseURL())
	if dbProfile.PostgresRLSEnabled && h.db.Dialect() == "postgres" {
		if n, err := store.CountPostgresRLSPolicies(h.db); err == nil {
			dbProfile.PostgresRLSPolicyCount = n
		}
	}
	migSnap, _ := store.MigrationSnapshotFor(h.db.DataDir())
	var lastSyncMs, lastSyncErrMs *int64
	if migSnap.LastSyncAt != nil {
		ms := migSnap.LastSyncAt.UnixMilli()
		lastSyncMs = &ms
	}
	if migSnap.LastSyncErrorAt != nil {
		ms := migSnap.LastSyncErrorAt.UnixMilli()
		lastSyncErrMs = &ms
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
		LastMigrationSyncAtMs:      lastSyncMs,
		LastMigrationSyncError:     migSnap.LastSyncError,
		LastMigrationSyncErrorAtMs: lastSyncErrMs,
		PostgresRLSEnabled:     dbProfile.PostgresRLSEnabled,
		PostgresRLSForce:       dbProfile.PostgresRLSForce,
		PostgresRLSPolicyCount:   dbProfile.PostgresRLSPolicyCount,
		PostgresAppURLConfigured: dbProfile.PostgresAppURL,
		WorkerConnectionRole:       store.WorkerConnectionRole(),
		RuntimeDSNHint:             dbProfile.DSNHint,
		DualWriteShadowURLHint:     migSnap.DualWriteShadowURLHint,
		SchemaMode:                 dbProfile.SchemaMode,
		SQLMigrationsEnabled:       dbProfile.SQLMigrationsEnabled,
		AutoMigrateEnabled:         dbProfile.AutoMigrateEnabled,
		SQLMigrationVersion:        dbProfile.SQLMigrationVersion,
		SQLMigrationExpected:       dbProfile.SQLMigrationExpected,
	})
}
