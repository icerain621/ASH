package runs

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/store"
)

func (s *Service) requestApproval(rec *store.RunRecord, step *store.RunStep, gate, risk, reason string, evidence map[string]any) {
	if rec == nil || rec.ID == "" || step == nil || step.StepID == "" {
		return
	}
	now := time.Now().UTC()
	evidenceJSON := "{}"
	if evidence != nil {
		if b, err := json.Marshal(evidence); err == nil {
			evidenceJSON = string(b)
		}
	}
	row := store.ApprovalRequest{
		ID: "apr_" + uuid.NewString(), SpaceID: firstNonEmpty(rec.SpaceID, "local"),
		RunID: rec.ID, TraceID: rec.TraceID, StepID: step.StepID,
		Gate: gate, Risk: risk, Reason: reason, Status: "pending",
		RequestedBy: "ash-runner", EvidenceJSON: evidenceJSON,
		CreatedAt: now, UpdatedAt: now,
	}
	var existing store.ApprovalRequest
	err := s.db.Where("run_id = ? AND step_id = ? AND status = ?", rec.ID, step.StepID, "pending").First(&existing).Error
	if err == nil {
		existing.Gate = gate
		existing.Risk = risk
		existing.Reason = reason
		existing.EvidenceJSON = evidenceJSON
		existing.UpdatedAt = now
		_ = s.db.Save(&existing).Error
		row = existing
	} else {
		_ = s.db.Create(&row).Error
	}
	_, _ = s.events.Append(rec.ID, rec.TraceID, "approval.requested", "warn", map[string]any{
		"approvalId": row.ID, "stepId": step.StepID, "gate": gate, "risk": risk, "reason": reason,
	})
	_ = s.writeAudit(rec.ID, rec.TraceID, "approval.requested", map[string]any{
		"approvalId": row.ID, "stepId": step.StepID, "gate": gate, "risk": risk, "reason": reason,
	})
}

func (s *Service) decidePendingApproval(runID, stepID, status, actorID, reason string) {
	now := time.Now().UTC()
	q := s.db.Model(&store.ApprovalRequest{}).Where("run_id = ? AND status = ?", runID, "pending")
	if stepID != "" {
		q = q.Where("step_id = ?", stepID)
	}
	_ = q.Updates(map[string]any{
		"status":          status,
		"decided_by":      actorID,
		"decision_reason": reason,
		"decided_at":      &now,
		"updated_at":      now,
	}).Error
}
