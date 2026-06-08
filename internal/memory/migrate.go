package memory

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

const (
	MemoryCatalogMetaKey = "memory_schema_catalog_version"
	MemoryToolVersion    = "ash-memory/0.1"
	legacySchemaVersion  = 0
)

// RunMigrationRequest applies pending memory schema migration steps.
type RunMigrationRequest struct {
	RunID   string `json:"runId,omitempty"`
	TraceID string `json:"traceId,omitempty"`
	DryRun  bool   `json:"dryRun,omitempty"`
}

// RunMigrationResponse summarizes one migration batch.
type RunMigrationResponse struct {
	OK             bool   `json:"ok"`
	FromVersion    int    `json:"fromVersion"`
	ToVersion      int    `json:"toVersion"`
	RecordsUpdated int    `json:"recordsUpdated"`
	MigrationID    string `json:"migrationId,omitempty"`
	AlreadyCurrent bool   `json:"alreadyCurrent,omitempty"`
	Summary        string `json:"summary,omitempty"`
}

// CatalogVersion returns the applied memory schema catalog revision.
func CatalogVersion(db *store.DB) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database is nil")
	}
	var meta store.SchemaMeta
	err := db.First(&meta, "key = ?", MemoryCatalogMetaKey).Error
	if err == gorm.ErrRecordNotFound {
		return legacySchemaVersion, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(strings.TrimSpace(meta.Value))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", MemoryCatalogMetaKey, err)
	}
	return v, nil
}

// PendingMigrationRecords counts rows needing schema backfill in a space.
func PendingMigrationRecords(db *store.DB, spaceID string) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("database is nil")
	}
	spaceID = firstNonEmpty(spaceID, "local")
	var n int64
	err := db.Model(&store.MemoryRecord{}).
		Where("space_id = ? AND (schema_version < ? OR dedupe_key = '' OR tags_json = '')", spaceID, CurrentSchemaVersion).
		Count(&n).Error
	return n, err
}

// RunMigrations applies memory schema steps up to CurrentSchemaVersion.
func (s *Service) RunMigrations(req RunMigrationRequest) (*RunMigrationResponse, error) {
	catalog, err := CatalogVersion(s.db)
	if err != nil {
		return nil, err
	}
	if catalog >= CurrentSchemaVersion {
		return &RunMigrationResponse{
			OK:             true,
			FromVersion:    catalog,
			ToVersion:      catalog,
			AlreadyCurrent: true,
			Summary:        fmt.Sprintf("catalog already at v%d", catalog),
		}, nil
	}

	traceID := req.TraceID
	if req.RunID != "" {
		traceID, err = s.validateRunRef(req.RunID, req.TraceID)
		if err != nil {
			return nil, err
		}
	}

	from := catalog
	to := from + 1
	updated, summary, err := s.applyMigrationStep(from, to, req.DryRun)
	if err != nil {
		if req.RunID != "" {
			_ = s.emitMigrated(req.RunID, traceID, from, to, false, updated, summary)
		}
		return nil, err
	}

	resp := &RunMigrationResponse{
		OK:             true,
		FromVersion:    from,
		ToVersion:      to,
		RecordsUpdated: updated,
		Summary:        summary,
	}
	if req.DryRun {
		resp.Summary = "dry-run: " + summary
		return resp, nil
	}

	migrationID := "mmig_" + uuid.NewString()
	now := time.Now().UTC()
	metaJSON, _ := json.Marshal(map[string]any{"recordsUpdated": updated})
	if err := s.gdb().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&store.MemoryMigration{
			ID:          migrationID,
			FromVersion: from,
			ToVersion:   to,
			ToolVersion: MemoryToolVersion,
			Summary:     summary,
			MetaJSON:    string(metaJSON),
			AppliedAt:   now,
		}).Error; err != nil {
			return err
		}
		var meta store.SchemaMeta
		res := tx.First(&meta, "key = ?", MemoryCatalogMetaKey)
		switch {
		case res.Error == gorm.ErrRecordNotFound:
			return tx.Create(&store.SchemaMeta{
				Key: MemoryCatalogMetaKey, Value: strconv.Itoa(to), UpdatedAt: now,
			}).Error
		case res.Error != nil:
			return res.Error
		default:
			meta.Value = strconv.Itoa(to)
			meta.UpdatedAt = now
			return tx.Save(&meta).Error
		}
	}); err != nil {
		if req.RunID != "" {
			_ = s.emitMigrated(req.RunID, traceID, from, to, false, updated, err.Error())
		}
		return nil, fmt.Errorf("record migration batch: %w", err)
	}

	resp.MigrationID = migrationID
	if req.RunID != "" {
		if err := s.emitMigrated(req.RunID, traceID, from, to, true, updated, summary); err != nil {
			return nil, fmt.Errorf("emit memory.migrated: %w", err)
		}
	}
	return resp, nil
}

func (s *Service) applyMigrationStep(from, to int, dryRun bool) (int, string, error) {
	switch {
	case from == legacySchemaVersion && to == 1:
		return s.migrateV0ToV1(dryRun)
	default:
		return 0, "", fmt.Errorf("unsupported memory migration %d→%d", from, to)
	}
}

func (s *Service) migrateV0ToV1(dryRun bool) (int, string, error) {
	var rows []store.MemoryRecord
	if err := s.gdb().
		Where("schema_version < ? OR dedupe_key = '' OR tags_json = ''", CurrentSchemaVersion).
		Find(&rows).Error; err != nil {
		return 0, "", err
	}
	if len(rows) == 0 {
		return 0, "v0→v1: no records required backfill", nil
	}
	if dryRun {
		return len(rows), fmt.Sprintf("v0→v1: would update %d record(s)", len(rows)), nil
	}
	now := time.Now().UTC()
	updated := 0
	err := s.gdb().Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			patch := map[string]any{
				"schema_version": CurrentSchemaVersion,
				"updated_at":     now,
			}
			if strings.TrimSpace(row.DedupeKey) == "" {
				patch["dedupe_key"] = dedupeKey(row.ScopeRepo, row.Title, row.Body)
			}
			if strings.TrimSpace(row.TagsJSON) == "" {
				patch["tags_json"] = "[]"
			}
			res := tx.Model(&store.MemoryRecord{}).Where("id = ?", row.ID).Updates(patch)
			if res.Error != nil {
				return res.Error
			}
			updated += int(res.RowsAffected)
		}
		return nil
	})
	if err != nil {
		return 0, "", err
	}
	return updated, fmt.Sprintf("v0→v1: backfilled %d record(s)", updated), nil
}
