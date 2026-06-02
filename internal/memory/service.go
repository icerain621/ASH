package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

var ErrNotFound = errors.New("memory not found")

func (s *Service) CreateCandidate(req CreateCandidateRequest) (*CreateCandidateResponse, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}
	traceID, err := s.validateRunRef(req.RunID, req.TraceID)
	if err != nil {
		return nil, err
	}

	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, err
	}
	sensitivity := req.Sensitivity
	if sensitivity == "" {
		sensitivity = "normal"
	}
	dedupe := dedupeKey(req.ScopeRepo, req.Title, req.Body)
	now := time.Now().UTC()
	id := "mem_" + uuid.NewString()

	var out *CreateCandidateResponse
	err = s.db.Transaction(func(tx *gorm.DB) error {
		rec := store.MemoryRecord{
			ID:            id,
			Layer:         req.Layer,
			Status:        "candidate",
			SchemaVersion: CurrentSchemaVersion,
			Title:         req.Title,
			Body:          req.Body,
			ScopeRepo:     req.ScopeRepo,
			ScopeTeam:     req.ScopeTeam,
			TagsJSON:      string(tagsJSON),
			TTLDays:       req.TTLDays,
			Sensitivity:   sensitivity,
			DedupeKey:     dedupe,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}

		for _, ev := range req.Evidence {
			metaJSON := "{}"
			if ev.Meta != nil {
				b, _ := json.Marshal(ev.Meta)
				metaJSON = string(b)
			}
			row := store.MemoryEvidence{
				ID:        "mev_" + uuid.NewString(),
				MemoryID:  id,
				Kind:      ev.Kind,
				Ref:       ev.Ref,
				Digest:    ev.Digest,
				MetaJSON:  metaJSON,
				CreatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		payload, _ := json.Marshal(map[string]any{
			"memoryId": id,
			"layer":    req.Layer,
			"title":    req.Title,
			"dedupeKey": dedupe,
		})
		audit := store.AuditLog{
			ID:          "aud_" + uuid.NewString(),
			TraceID:     req.TraceID,
			RunID:       req.RunID,
			ActorID:     req.ActorID,
			EventType:   "memory.candidate_created",
			PayloadJSON: string(payload),
			CreatedAt:   now,
		}
		if err := tx.Create(&audit).Error; err != nil {
			return err
		}
		out = &CreateCandidateResponse{CandidateID: id}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("create candidate: %w", err)
	}
	if err := s.emitCandidateCreated(req.RunID, traceID, id, req.Layer, sensitivity, len(req.Evidence)); err != nil {
		return nil, fmt.Errorf("emit memory event: %w", err)
	}
	if req.RunID != "" && req.Layer != "L0" {
		_ = s.emitRunEvent(req.RunID, traceID, "memory.review_requested", map[string]any{
			"candidateId": id,
			"layer":       req.Layer,
		})
	}
	return out, nil
}

func (s *Service) ListCandidates(layer, status, repo string, limit, offset int) (*ListCandidatesResponse, error) {
	if status == "" {
		status = "candidate"
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	q := s.db.Model(&store.MemoryRecord{})
	if layer != "" {
		q = q.Where("layer = ?", layer)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if repo != "" {
		q = q.Where("scope_repo = ?", repo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []store.MemoryRecord
	if err := q.Order("created_at desc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}

	items, err := s.attachEvidence(rows)
	if err != nil {
		return nil, err
	}
	return &ListCandidatesResponse{Items: items, Limit: limit, Offset: offset, Total: total}, nil
}

func (s *Service) Review(candidateID string, req ReviewRequest) (*ReviewResponse, error) {
	if _, ok := validDecisions[req.Decision]; !ok {
		return nil, fmt.Errorf("invalid decision %q", req.Decision)
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf("reason is required")
	}
	if req.PolicyProfile == "" {
		return nil, fmt.Errorf("policyProfile is required")
	}
	traceID, err := s.validateRunRef(req.RunID, req.TraceID)
	if err != nil {
		return nil, err
	}

	var rec store.MemoryRecord
	if err := s.db.First(&rec, "id = ?", candidateID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	newStatus, err := decisionToStatus(req.Decision, rec.Status)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	err = s.db.Transaction(func(tx *gorm.DB) error {
		rec.Status = newStatus
		rec.UpdatedAt = now
		if err := tx.Save(&rec).Error; err != nil {
			return err
		}
		rev := store.MemoryReview{
			ID:            "mrv_" + uuid.NewString(),
			MemoryID:      candidateID,
			Decision:      req.Decision,
			ReviewerID:    req.ReviewerID,
			Reason:        req.Reason,
			PolicyProfile: req.PolicyProfile,
			CreatedAt:     now,
		}
		if err := tx.Create(&rev).Error; err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{
			"memoryId":      candidateID,
			"decision":      req.Decision,
			"status":        newStatus,
			"policyProfile": req.PolicyProfile,
		})
		return tx.Create(&store.AuditLog{
			ID:          "aud_" + uuid.NewString(),
			TraceID:     req.TraceID,
			RunID:       req.RunID,
			ActorID:     firstNonEmpty(req.ActorID, req.ReviewerID),
			EventType:   "memory.reviewed",
			PayloadJSON: string(payload),
			CreatedAt:   now,
		}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("review: %w", err)
	}
	if err := s.emitReviewed(req.RunID, traceID, candidateID, req.Decision, req.Reason, req.PolicyProfile); err != nil {
		return nil, fmt.Errorf("emit memory event: %w", err)
	}
	return &ReviewResponse{OK: true, Status: newStatus}, nil
}

func (s *Service) Query(req QueryRequest) (*QueryResponse, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, fmt.Errorf("text is required")
	}
	topK := req.TopK
	if topK <= 0 || topK > 50 {
		topK = 10
	}

	q := s.db.Where("status = ?", "approved")
	if len(req.Layers) > 0 {
		q = q.Where("layer IN ?", req.Layers)
	}
	if req.Scope != nil {
		if repo := req.Scope["repo"]; repo != "" {
			q = q.Where("scope_repo = ?", repo)
		}
		if team := req.Scope["team"]; team != "" {
			q = q.Where("scope_team = ?", team)
		}
	}
	like := "%" + strings.ToLower(req.Text) + "%"
	q = q.Where("LOWER(title) LIKE ? OR LOWER(body) LIKE ?", like, like)

	var rows []store.MemoryRecord
	if err := q.Order("updated_at desc").Limit(topK).Find(&rows).Error; err != nil {
		return nil, err
	}
	items, err := s.attachEvidence(rows)
	if err != nil {
		return nil, err
	}
	return &QueryResponse{Items: items}, nil
}

func (s *Service) HitUsed(req HitUsedRequest) (*HitUsedResponse, error) {
	if req.RunID == "" {
		return nil, fmt.Errorf("runId is required")
	}
	if len(req.RecordIDs) == 0 {
		return nil, fmt.Errorf("recordIds is required")
	}
	traceID, err := s.validateRunRef(req.RunID, req.TraceID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{
		"runId":     req.RunID,
		"recordIds": req.RecordIDs,
	})
	if err := s.db.Create(&store.AuditLog{
		ID:          "aud_" + uuid.NewString(),
		TraceID:     traceID,
		RunID:       req.RunID,
		ActorID:     req.ActorID,
		EventType:   "memory.hit_used",
		PayloadJSON: string(payload),
		CreatedAt:   now,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.emitHitUsed(req.RunID, traceID, req.RecordIDs); err != nil {
		return nil, fmt.Errorf("emit memory event: %w", err)
	}
	return &HitUsedResponse{OK: true}, nil
}

func (s *Service) Get(id string) (*RecordView, error) {
	var rec store.MemoryRecord
	if err := s.db.First(&rec, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	items, err := s.attachEvidence([]store.MemoryRecord{rec})
	if err != nil || len(items) == 0 {
		return nil, err
	}
	return &items[0], nil
}

func (s *Service) attachEvidence(rows []store.MemoryRecord) ([]RecordView, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	var evRows []store.MemoryEvidence
	if err := s.db.Where("memory_id IN ?", ids).Find(&evRows).Error; err != nil {
		return nil, err
	}
	byMem := map[string][]EvidenceView{}
	for _, e := range evRows {
		byMem[e.MemoryID] = append(byMem[e.MemoryID], EvidenceView{
			ID: e.ID, Kind: e.Kind, Ref: e.Ref, Digest: e.Digest,
		})
	}

	out := make([]RecordView, 0, len(rows))
	for _, r := range rows {
		var tags []string
		_ = json.Unmarshal([]byte(r.TagsJSON), &tags)
		out = append(out, RecordView{
			ID:            r.ID,
			Layer:         r.Layer,
			Status:        r.Status,
			SchemaVersion: r.SchemaVersion,
			Title:         r.Title,
			Body:          r.Body,
			ScopeRepo:     r.ScopeRepo,
			ScopeTeam:     r.ScopeTeam,
			Tags:          tags,
			TTLDays:       r.TTLDays,
			Sensitivity:   r.Sensitivity,
			DedupeKey:     r.DedupeKey,
			CreatedAt:     ms(r.CreatedAt),
			UpdatedAt:     ms(r.UpdatedAt),
			Evidence:      byMem[r.ID],
		})
	}
	return out, nil
}

func validateCreate(req CreateCandidateRequest) error {
	if _, ok := validLayers[req.Layer]; !ok {
		return fmt.Errorf("invalid layer %q", req.Layer)
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		return fmt.Errorf("title and body are required")
	}
	sens := req.Sensitivity
	if sens == "" {
		sens = "normal"
	}
	if _, ok := validSensitivity[sens]; !ok {
		return fmt.Errorf("invalid sensitivity %q", sens)
	}
	for _, ev := range req.Evidence {
		if _, ok := validEvidenceKind[ev.Kind]; !ok {
			return fmt.Errorf("invalid evidence kind %q", ev.Kind)
		}
		if strings.TrimSpace(ev.Ref) == "" {
			return fmt.Errorf("evidence ref is required")
		}
	}
	if req.Layer != "L0" && len(req.Evidence) == 0 {
		return fmt.Errorf("layer %s requires at least one evidence item", req.Layer)
	}
	return nil
}

func decisionToStatus(decision, current string) (string, error) {
	switch decision {
	case "approve":
		if current != "candidate" {
			return "", fmt.Errorf("approve only allowed for candidate, got %q", current)
		}
		return "approved", nil
	case "reject":
		if current != "candidate" {
			return "", fmt.Errorf("reject only allowed for candidate, got %q", current)
		}
		return "rejected", nil
	case "deprecate":
		if current != "approved" && current != "candidate" {
			return "", fmt.Errorf("deprecate not allowed for status %q", current)
		}
		return "deprecated", nil
	default:
		return "", fmt.Errorf("unknown decision %q", decision)
	}
}

func dedupeKey(repo, title, body string) string {
	norm := strings.ToLower(strings.TrimSpace(title)) + "|" + strings.ToLower(strings.TrimSpace(body)) + "|" + repo
	h := sha256.Sum256([]byte(norm))
	return "sha256:" + hex.EncodeToString(h[:])
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
