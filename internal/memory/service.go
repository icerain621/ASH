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
	spaceID := firstNonEmpty(req.SpaceID, "local")

	var out *CreateCandidateResponse
	err = s.db.Transaction(func(tx *gorm.DB) error {
		rec := store.MemoryRecord{
			ID:            id,
			Layer:         req.Layer,
			Status:        "candidate",
			SpaceID:       spaceID,
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
			"memoryId":  id,
			"layer":     req.Layer,
			"title":     req.Title,
			"dedupeKey": dedupe,
		})
		audit := store.AuditLog{
			ID:          "aud_" + uuid.NewString(),
			SpaceID:     spaceID,
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
	return s.ListCandidatesForSpace("local", layer, status, repo, limit, offset)
}

func (s *Service) ListCandidatesForSpace(spaceID, layer, status, repo string, limit, offset int) (*ListCandidatesResponse, error) {
	if status == "" {
		status = "candidate"
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	q := s.db.Model(&store.MemoryRecord{}).Where("space_id = ?", firstNonEmpty(spaceID, "local"))
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
	if req.Confidence != nil && (*req.Confidence < 0 || *req.Confidence > 1) {
		return nil, fmt.Errorf("confidence must be between 0 and 1")
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
	var governanceEdges []store.MemoryEdge
	err = s.db.Transaction(func(tx *gorm.DB) error {
		rec.Status = newStatus
		if req.Decision == "approve" {
			rec.Confidence = reviewConfidence(req, rec)
		}
		rec.UpdatedAt = now
		edges, err := s.applyGovernance(tx, rec, req, now)
		if err != nil {
			return err
		}
		governanceEdges = edges
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
			"confidence":    rec.Confidence,
			"edges":         edgeAuditPayload(governanceEdges),
		})
		return tx.Create(&store.AuditLog{
			ID:          "aud_" + uuid.NewString(),
			SpaceID:     rec.SpaceID,
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
	for _, edge := range governanceEdges {
		if err := s.emitGovernanceEdge(req.RunID, traceID, edge); err != nil {
			return nil, fmt.Errorf("emit memory edge event: %w", err)
		}
	}
	return &ReviewResponse{OK: true, Status: newStatus}, nil
}

func (s *Service) Query(req QueryRequest) (*QueryResponse, error) {
	return s.QueryForSpace("local", req)
}

func (s *Service) QueryForSpace(spaceID string, req QueryRequest) (*QueryResponse, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, fmt.Errorf("text is required")
	}
	topK := req.TopK
	if topK <= 0 || topK > 50 {
		topK = 10
	}

	q := s.db.Where("status = ? AND space_id = ?", "approved", firstNonEmpty(spaceID, "local"))
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
	if err := q.Order("updated_at desc").Limit(topK * 3).Find(&rows).Error; err != nil {
		return nil, err
	}
	rows, err := s.filterQueryableMemory(rows, topK, time.Now().UTC())
	if err != nil {
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
	spaceID, err := s.runSpaceID(req.RunID)
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
		SpaceID:     spaceID,
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

func (s *Service) runSpaceID(runID string) (string, error) {
	var rec store.RunRecord
	if err := s.db.Select("space_id").First(&rec, "id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrRunNotFound
		}
		return "", err
	}
	return firstNonEmpty(rec.SpaceID, "local"), nil
}

func (s *Service) Get(id string) (*RecordView, error) {
	return s.GetForSpace("", id)
}

func (s *Service) GetForSpace(spaceID, id string) (*RecordView, error) {
	var rec store.MemoryRecord
	q := s.db.DB
	if spaceID != "" {
		q = q.Where("space_id = ?", spaceID)
	}
	if err := q.First(&rec, "id = ?", id).Error; err != nil {
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
	var edgeRows []store.MemoryEdge
	if err := s.db.Where("from_id IN ? OR to_id IN ?", ids, ids).Order("created_at asc").Find(&edgeRows).Error; err != nil {
		return nil, err
	}
	edgesByMem := map[string][]EdgeView{}
	for _, edge := range edgeRows {
		view := EdgeView{
			ID: edge.ID, FromID: edge.FromID, ToID: edge.ToID, Kind: edge.Kind,
			Confidence: edge.Confidence, Reason: edge.Reason, CreatedAt: ms(edge.CreatedAt),
		}
		edgesByMem[edge.FromID] = append(edgesByMem[edge.FromID], view)
		if edge.ToID != edge.FromID {
			edgesByMem[edge.ToID] = append(edgesByMem[edge.ToID], view)
		}
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
			Confidence:    r.Confidence,
			CreatedAt:     ms(r.CreatedAt),
			UpdatedAt:     ms(r.UpdatedAt),
			Evidence:      byMem[r.ID],
			Edges:         edgesByMem[r.ID],
		})
	}
	return out, nil
}

func (s *Service) applyGovernance(tx *gorm.DB, rec store.MemoryRecord, req ReviewRequest, now time.Time) ([]store.MemoryEdge, error) {
	if req.Decision != "approve" {
		return nil, nil
	}

	var edges []store.MemoryEdge
	addEdge := func(kind, toID, reason string, confidence float64) error {
		if toID == "" || toID == rec.ID {
			return nil
		}
		if err := ensureMemoryExists(tx, toID); err != nil {
			return err
		}
		edge := store.MemoryEdge{
			ID: "medge_" + uuid.NewString(), SpaceID: rec.SpaceID,
			FromID: rec.ID, ToID: toID, Kind: kind,
			Confidence: confidence, Reason: firstNonEmpty(reason, req.Reason), CreatedAt: now,
		}
		if err := tx.Create(&edge).Error; err != nil {
			return err
		}
		edges = append(edges, edge)
		return nil
	}

	for _, duplicateID := range uniqueStrings(req.DuplicateOf) {
		if err := addEdge("duplicate", duplicateID, "reviewer marked duplicate", rec.Confidence); err != nil {
			return nil, err
		}
	}

	autoDuplicates, err := s.findApprovedDuplicates(tx, rec)
	if err != nil {
		return nil, err
	}
	for _, duplicateID := range autoDuplicates {
		if err := addEdge("duplicate", duplicateID, "dedupe key matched approved memory", 1); err != nil {
			return nil, err
		}
	}

	autoConflicts, err := s.findApprovedTitleConflicts(tx, rec)
	if err != nil {
		return nil, err
	}
	for _, conflictID := range autoConflicts {
		if err := addEdge("conflict", conflictID, "title matched approved memory with different body", rec.Confidence); err != nil {
			return nil, err
		}
	}

	for _, replacedID := range uniqueStrings(req.Replaces...) {
		if err := addEdge("replaces", replacedID, "reviewer marked replacement", rec.Confidence); err != nil {
			return nil, err
		}
		if err := tx.Model(&store.MemoryRecord{}).Where("id = ?", replacedID).Updates(map[string]any{
			"status":     "deprecated",
			"updated_at": now,
		}).Error; err != nil {
			return nil, err
		}
	}

	for _, conflictID := range uniqueStrings(req.ConflictsWith...) {
		if err := addEdge("conflict", conflictID, "reviewer marked conflict", rec.Confidence); err != nil {
			return nil, err
		}
	}
	return edges, nil
}

func ensureMemoryExists(tx *gorm.DB, id string) error {
	var count int64
	if err := tx.Model(&store.MemoryRecord{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("memory edge target %q not found", id)
	}
	return nil
}

func (s *Service) findApprovedDuplicates(tx *gorm.DB, rec store.MemoryRecord) ([]string, error) {
	if rec.DedupeKey == "" {
		return nil, nil
	}
	var rows []store.MemoryRecord
	if err := tx.Where("id <> ? AND status = ? AND dedupe_key = ?", rec.ID, "approved", rec.DedupeKey).
		Order("updated_at desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ID)
	}
	return out, nil
}

func (s *Service) findApprovedTitleConflicts(tx *gorm.DB, rec store.MemoryRecord) ([]string, error) {
	title := strings.ToLower(strings.TrimSpace(rec.Title))
	if title == "" {
		return nil, nil
	}
	q := tx.Where("id <> ? AND status = ? AND space_id = ? AND scope_repo = ? AND LOWER(title) = ?",
		rec.ID, "approved", rec.SpaceID, rec.ScopeRepo, title)
	if rec.DedupeKey != "" {
		q = q.Where("dedupe_key <> ?", rec.DedupeKey)
	}
	var rows []store.MemoryRecord
	if err := q.Order("updated_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ID)
	}
	return out, nil
}

func (s *Service) filterQueryableMemory(rows []store.MemoryRecord, topK int, now time.Time) ([]store.MemoryRecord, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	var edges []store.MemoryEdge
	if err := s.db.Where("from_id IN ? AND kind = ?", ids, "duplicate").Find(&edges).Error; err != nil {
		return nil, err
	}
	duplicated := map[string]bool{}
	for _, edge := range edges {
		duplicated[edge.FromID] = true
	}

	out := make([]store.MemoryRecord, 0, len(rows))
	for _, row := range rows {
		if duplicated[row.ID] || memoryExpired(row, now) {
			continue
		}
		out = append(out, row)
		if len(out) >= topK {
			break
		}
	}
	return out, nil
}

func memoryExpired(row store.MemoryRecord, now time.Time) bool {
	if row.TTLDays == nil || *row.TTLDays <= 0 {
		return false
	}
	return row.CreatedAt.Add(time.Duration(*row.TTLDays) * 24 * time.Hour).Before(now)
}

func reviewConfidence(req ReviewRequest, rec store.MemoryRecord) float64 {
	if req.Confidence != nil {
		return *req.Confidence
	}
	if rec.Confidence > 0 {
		return rec.Confidence
	}
	if rec.Layer == "L0" {
		return 0.6
	}
	return 0.85
}

func edgeAuditPayload(edges []store.MemoryEdge) []map[string]any {
	out := make([]map[string]any, 0, len(edges))
	for _, edge := range edges {
		out = append(out, map[string]any{
			"edgeId": edge.ID, "fromId": edge.FromID, "toId": edge.ToID,
			"kind": edge.Kind, "confidence": edge.Confidence,
		})
	}
	return out
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

func uniqueStrings(vals ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(vals))
	for _, val := range vals {
		val = strings.TrimSpace(val)
		if val == "" || seen[val] {
			continue
		}
		seen[val] = true
		out = append(out, val)
	}
	return out
}
