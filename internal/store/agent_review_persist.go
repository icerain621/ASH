package store

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SaveAgentTemplateCandidate upserts a candidate template row for a space.
func (db *DB) SaveAgentTemplateCandidate(spaceID, templateID, version, manifestJSON, createdBy string) (*AgentTemplateVersion, error) {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, fmt.Errorf("templateId is required")
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "1.0.0"
	}
	now := time.Now().UTC()
	var row AgentTemplateVersion
	err := db.Where("space_id = ? AND template_id = ? AND version = ?", spaceID, templateID, version).First(&row).Error
	if err == nil {
		if row.Status == "approved" {
			return nil, fmt.Errorf("template %s is already approved", templateID)
		}
		row.ManifestJSON = manifestJSON
		row.Status = "candidate"
		row.UpdatedAt = now
		if err := db.Save(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row = AgentTemplateVersion{
		ID: "atpl_" + uuid.NewString(), SpaceID: spaceID, TemplateID: templateID, Version: version,
		Status: "candidate", ManifestJSON: manifestJSON, CreatedBy: strings.TrimSpace(createdBy),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ApproveAgentTemplate marks a candidate template approved.
func (db *DB) ApproveAgentTemplate(spaceID, templateID, actor string) (*AgentTemplateVersion, error) {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	var row AgentTemplateVersion
	if err := db.Where("space_id = ? AND template_id = ? AND status = ?", spaceID, strings.TrimSpace(templateID), "candidate").
		Order("updated_at desc").First(&row).Error; err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row.Status = "approved"
	row.ApprovedBy = strings.TrimSpace(actor)
	row.ApprovedAt = &now
	row.UpdatedAt = now
	if err := db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetApprovedAgentTemplate returns the latest approved custom template manifest JSON.
func (db *DB) GetApprovedAgentTemplate(spaceID, templateID string) (*AgentTemplateVersion, error) {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	var row AgentTemplateVersion
	if err := db.Where("space_id = ? AND template_id = ? AND status = ?", spaceID, strings.TrimSpace(templateID), "approved").
		Order("updated_at desc").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// CreateReviewItem inserts a pending review queue row.
func (db *DB) CreateReviewItem(spaceID, kind, targetID, reason string) (*ReviewItemRow, error) {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	kind = strings.TrimSpace(kind)
	targetID = strings.TrimSpace(targetID)
	if kind == "" || targetID == "" {
		return nil, fmt.Errorf("kind and targetId are required")
	}
	now := time.Now().UTC()
	row := ReviewItemRow{
		ID: "rev_" + uuid.NewString(), SpaceID: spaceID, Kind: kind, TargetID: targetID,
		Status: "pending", Reason: strings.TrimSpace(reason), CreatedAt: now,
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// DecideReviewItem approves or rejects a pending review row.
func (db *DB) DecideReviewItem(id, decision, actor, reason string) (*ReviewItemRow, error) {
	var row ReviewItemRow
	if err := db.First(&row, "id = ?", strings.TrimSpace(id)).Error; err != nil {
		return nil, err
	}
	if row.Status != "pending" {
		return nil, fmt.Errorf("review item %s status %q", id, row.Status)
	}
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "approve":
		row.Status = "approved"
	case "reject":
		row.Status = "rejected"
	default:
		return nil, fmt.Errorf("decision must be approve|reject")
	}
	if r := strings.TrimSpace(reason); r != "" {
		row.Reason = r
	}
	now := time.Now().UTC()
	row.DecidedAt = &now
	row.DecidedBy = strings.TrimSpace(actor)
	if err := db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
