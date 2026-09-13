package evolve

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/scenariopatch"
	"github.com/ash-repwiki/ash/internal/scoring"
	"github.com/ash-repwiki/ash/internal/spacepolicy"
	"github.com/ash-repwiki/ash/internal/store"
)

type Item struct {
	ID         string  `json:"id"`
	Queue      string  `json:"queue"`
	TargetType string  `json:"targetType"`
	TargetID   string  `json:"targetId"`
	Title      string  `json:"title"`
	Summary    string  `json:"summary,omitempty"`
	Diff       string  `json:"diff,omitempty"`
	Status     string  `json:"status"`
	SpaceID    string  `json:"spaceId"`
	CreatedAt  int64   `json:"createdAt"`
	AssigneeID string  `json:"assigneeId,omitempty"`
	SlaBreach  bool    `json:"slaBreach,omitempty"`
	AgeHours   float64 `json:"ageHours,omitempty"`
}

type ListResponse struct {
	Items []Item `json:"items"`
	Queue string `json:"queue,omitempty"`
}

// DecideConfig toggles rubric enforcement.
type DecideConfig struct {
	RequireRubric bool // when true, approve on memory/harness requires rubric
}

type DecideRequest struct {
	Decision      string             `json:"decision"`
	Reason        string             `json:"reason"`
	PolicyProfile string             `json:"policyProfile,omitempty"`
	ActorID       string             `json:"actorId,omitempty"`
	RunID         string             `json:"runId,omitempty"`
	Rubric        *scoring.ReviewRubric `json:"rubric,omitempty"`
	RubricMap     map[string]int     `json:"-"` // alternate map form (tests)
}

type DecideResponse struct {
	ID            string  `json:"id"`
	Queue         string  `json:"queue"`
	TargetType    string  `json:"targetType"`
	TargetID      string  `json:"targetId"`
	Decision      string  `json:"decision"`
	Status        string  `json:"status"`
	ScoreEventID  string  `json:"scoreEventId,omitempty"`
	Composite     float64 `json:"composite,omitempty"`
	ImproveDraftID string `json:"improveDraftId,omitempty"`
}

type Service struct {
	db       *store.DB
	memory   *memory.Service
	harness  *harness.Service
	patches  *scenariopatch.Service
	scoring  *scoring.Service
	policy   *spacepolicy.Service
	cfg      DecideConfig
}

func NewService(db *store.DB, mem *memory.Service, har *harness.Service, patches *scenariopatch.Service) *Service {
	return &Service{
		db: db, memory: mem, harness: har, patches: patches,
		scoring: scoring.NewService(db),
		policy:  spacepolicy.NewService(db),
		cfg:     DecideConfig{RequireRubric: true},
	}
}

func (s *Service) WithScoring(sc *scoring.Service) *Service {
	if s == nil {
		return nil
	}
	out := *s
	if sc != nil {
		out.scoring = sc
	}
	return &out
}

func (s *Service) WithPolicy(p *spacepolicy.Service) *Service {
	if s == nil {
		return nil
	}
	out := *s
	if p != nil {
		out.policy = p
	}
	return &out
}

func (s *Service) WithDecideConfig(cfg DecideConfig) *Service {
	if s == nil {
		return nil
	}
	out := *s
	out.cfg = cfg
	return &out
}

func (s *Service) WithContext(db *store.DB, mem *memory.Service, har *harness.Service, patches *scenariopatch.Service) *Service {
	if s == nil {
		return nil
	}
	out := *s
	if db != nil {
		out.db = db
		out.scoring = scoring.NewService(db)
		out.policy = spacepolicy.NewService(db)
	}
	if mem != nil {
		out.memory = mem
	}
	if har != nil {
		out.harness = har
	}
	if patches != nil {
		out.patches = patches
	}
	return &out
}

func (s *Service) ListQueue(spaceID, queue string, limit int) (*ListResponse, error) {
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := strings.ToLower(strings.TrimSpace(queue))
	var items []Item
	switch q {
	case "", "all":
		memItems, err := s.listMemory(space, limit)
		if err != nil {
			return nil, err
		}
		orchItems, err := s.listOrchestration(space, limit)
		if err != nil {
			return nil, err
		}
		items = append(memItems, orchItems...)
	case QueueMemory:
		var err error
		items, err = s.listMemory(space, limit)
		if err != nil {
			return nil, err
		}
	case QueueOrchestration:
		var err error
		items, err = s.listOrchestration(space, limit)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("queue must be memory|orchestration|all")
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return &ListResponse{Items: items, Queue: q}, nil
}

func (s *Service) listMemory(spaceID string, limit int) ([]Item, error) {
	if s.memory == nil {
		return nil, nil
	}
	resp, err := s.memory.ListCandidatesForSpace(spaceID, "", "candidate", "", limit, 0)
	if err != nil {
		return nil, err
	}
	hours := s.reviewSLAHours(spaceID)
	now := time.Now().UTC()
	out := make([]Item, 0, len(resp.Items))
	for _, it := range resp.Items {
		status := StatusPending
		if first, ok := s.loadPendingApprover("memory", it.ID); ok && first != "" {
			status = StatusPendingSecond
		}
		item := Item{
			ID:         ItemID("memory", it.ID),
			Queue:      QueueMemory,
			TargetType: "memory",
			TargetID:   it.ID,
			Title:      firstNonEmpty(it.Title, it.ID),
			Summary:    truncate(it.Body, 200),
			Status:     status,
			SpaceID:    spaceID,
			CreatedAt:  it.CreatedAt,
		}
		if aid, ok := s.loadAssignee("memory", it.ID); ok {
			item.AssigneeID = aid
		}
		applySLABreach(&item, hours, now)
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) listOrchestration(spaceID string, limit int) ([]Item, error) {
	hours := s.reviewSLAHours(spaceID)
	now := time.Now().UTC()
	out := make([]Item, 0, limit)
	if s.harness != nil {
		for _, st := range []string{harness.StatusInReview, harness.StatusPendingSecond} {
			views, err := s.harness.List(spaceID, st, "")
			if err != nil {
				return nil, err
			}
			for _, v := range views {
				if len(out) >= limit {
					break
				}
				status := StatusPending
				if v.Status == harness.StatusPendingSecond {
					status = StatusPendingSecond
				}
				item := Item{
					ID:         ItemID("harness_profile", v.ID),
					Queue:      QueueOrchestration,
					TargetType: "harness_profile",
					TargetID:   v.ID,
					Title:      fmt.Sprintf("%s@v%d", v.Name, v.Version),
					Summary:    "harness profile awaiting orchestration review",
					Status:     status,
					SpaceID:    v.SpaceID,
					CreatedAt:  v.UpdatedAt,
				}
				if aid, ok := s.loadAssignee("harness_profile", v.ID); ok {
					item.AssigneeID = aid
				}
				applySLABreach(&item, hours, now)
				out = append(out, item)
			}
		}
	}
	if s.patches != nil && len(out) < limit {
		patches, err := s.patches.List(spaceID, scenariopatch.StatusInReview)
		if err != nil {
			return nil, err
		}
		for _, p := range patches {
			if len(out) >= limit {
				break
			}
			item := Item{
				ID:         ItemID("scenario_patch", p.ID),
				Queue:      QueueOrchestration,
				TargetType: "scenario_patch",
				TargetID:   p.ID,
				Title:      p.Title,
				Summary:    fmt.Sprintf("%s %s→%s", p.ScenarioName, p.FromVersion, p.ToVersion),
				Diff:       p.DiffText,
				Status:     StatusPending,
				SpaceID:    p.SpaceID,
				CreatedAt:  p.UpdatedAt,
			}
			if aid, ok := s.loadAssignee("scenario_patch", p.ID); ok {
				item.AssigneeID = aid
			}
			applySLABreach(&item, hours, now)
			out = append(out, item)
		}
	}
	return out, nil
}

// reviewSLAHours mirrors waker/review_sla.go: EffectivePolicy ReviewSLAHours, else 72h.
func (s *Service) reviewSLAHours(spaceID string) int {
	hours := 72
	if s.policy != nil {
		if eff, err := s.policy.EffectivePolicy(spaceID); err == nil && eff != nil && eff.ReviewSLAHours > 0 {
			hours = eff.ReviewSLAHours
		}
	}
	return hours
}

// applySLABreach sets AgeHours and SlaBreach from the item age anchor (CreatedAt unix ms).
// Memory uses CreatedAt; harness/patch list paths already store UpdatedAt in CreatedAt (waker semantics).
func applySLABreach(item *Item, hours int, now time.Time) {
	if item == nil || item.CreatedAt <= 0 || hours <= 0 {
		return
	}
	anchor := time.UnixMilli(item.CreatedAt).UTC()
	item.AgeHours = now.Sub(anchor).Hours()
	cutoff := now.Add(-time.Duration(hours) * time.Hour)
	if anchor.Before(cutoff) {
		item.SlaBreach = true
	}
}

// Assign sets the review queue assignee for a pending item (persisted via audit_log).
func (s *Service) Assign(spaceID, itemID, actorID, assigneeID string) error {
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}
	targetType, targetID, ok := ParseItemID(itemID)
	if !ok {
		return fmt.Errorf("invalid review id")
	}
	assigneeID = strings.TrimSpace(assigneeID)
	if assigneeID == "" {
		return fmt.Errorf("assigneeId is required")
	}
	q, err := s.ListQueue(space, "all", 100)
	if err != nil {
		return err
	}
	found := false
	for _, it := range q.Items {
		if it.ID == itemID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("review item not in pending queue")
	}
	s.saveAssignee(space, targetType, targetID, assigneeID, actorID)
	return nil
}

func (s *Service) Decide(spaceID, itemID string, req DecideRequest) (*DecideResponse, error) {
	decision := strings.ToLower(strings.TrimSpace(req.Decision))
	if decision != DecisionApprove && decision != DecisionReject {
		return nil, fmt.Errorf("decision must be approve|reject")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf("reason is required")
	}
	tt, tid, ok := ParseItemID(itemID)
	if !ok {
		return nil, fmt.Errorf("invalid review item id")
	}
	space := strings.TrimSpace(spaceID)
	if space == "" {
		space = "local"
	}

	_, scoreID, composite, err := s.maybeRecordRubric(space, tt, tid, req)
	if err != nil {
		return nil, err
	}

	var resp *DecideResponse
	switch tt {
	case "memory":
		resp, err = s.decideMemory(space, tid, decision, req)
	case "harness_profile":
		resp, err = s.decideHarness(space, tid, decision, req)
	case "scenario_patch":
		resp, err = s.decideScenarioPatch(space, tid, decision, req)
	default:
		return nil, fmt.Errorf("unsupported targetType %q", tt)
	}
	if err != nil {
		return nil, err
	}
	resp.ScoreEventID = scoreID
	resp.Composite = composite

	if scoreID != "" && (decision == DecisionReject || composite < scoring.LowCompositeThreshold) {
		if draftID := s.maybeDraftImprove(space, tt, tid, req.RunID, scoreID, composite, req.ActorID); draftID != "" {
			resp.ImproveDraftID = draftID
		}
	}
	return resp, nil
}

func (s *Service) maybeRecordRubric(space, targetType, targetID string, req DecideRequest) (scoring.ReviewRubric, string, float64, error) {
	var rubric scoring.ReviewRubric
	has := false
	if req.Rubric != nil {
		rubric = *req.Rubric
		has = true
	} else if len(req.RubricMap) > 0 {
		r, err := scoring.RubricFromMap(req.RubricMap)
		if err != nil {
			return scoring.ReviewRubric{}, "", 0, err
		}
		rubric = r
		has = true
	}
	need := s.cfg.RequireRubric &&
		strings.EqualFold(req.Decision, DecisionApprove) &&
		(targetType == "memory" || targetType == "harness_profile")
	if need && !has {
		return scoring.ReviewRubric{}, "", 0, fmt.Errorf("rubric is required for approve on %s", targetType)
	}
	if !has {
		return scoring.ReviewRubric{}, "", 0, nil
	}
	if err := rubric.Validate(); err != nil {
		return scoring.ReviewRubric{}, "", 0, err
	}
	if s.scoring == nil {
		return scoring.ReviewRubric{}, "", 0, fmt.Errorf("scoring service unavailable")
	}
	ev, err := s.scoring.RecordScore(space, targetType, targetID, req.RunID, rubric, req.ActorID, req.Reason)
	if err != nil {
		return scoring.ReviewRubric{}, "", 0, err
	}
	return rubric, ev.ID, ev.Composite, nil
}

func (s *Service) maybeDraftImprove(space, targetType, targetID, runID, scoreEventID string, composite float64, actor string) string {
	if composite >= scoring.LowCompositeThreshold && strings.TrimSpace(scoreEventID) == "" {
		return ""
	}
	if s.db == nil {
		return ""
	}
	now := time.Now().UTC()
	baseline := strings.TrimSpace(runID)
	row := store.ImproveProposal{
		ID: "imp_" + uuid.NewString(), SpaceID: space,
		Title:         fmt.Sprintf("Low score on %s:%s (%.2f)", targetType, targetID, composite),
		Description:   fmt.Sprintf("Auto-draft from review score_event %s (composite=%.2f).", scoreEventID, composite),
		BaselineRunID: baseline,
		Status:        "draft",
		ChangeSummary: "source=low_score",
		Source:        "low_score",
		ScoreEventID:  scoreEventID,
		ActorID:       actor,
		CompareJSON:   "{}",
		CreatedAt:     now, UpdatedAt: now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return ""
	}
	return row.ID
}

func (s *Service) requiresMultiSign(spaceID string) bool {
	if s.policy == nil || s.db == nil {
		return false
	}
	var sp store.Space
	if err := s.db.First(&sp, "id = ?", spaceID).Error; err != nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(sp.Kind), "team") {
		return false
	}
	eff, err := s.policy.EffectivePolicy(spaceID)
	if err != nil || eff == nil {
		return false
	}
	return eff.MultiSign
}

func (s *Service) decideScenarioPatch(spaceID, patchID, decision string, req DecideRequest) (*DecideResponse, error) {
	if s.patches == nil {
		return nil, fmt.Errorf("scenario patch service unavailable")
	}
	view, err := s.patches.Get(patchID)
	if err != nil {
		return nil, err
	}
	if view.SpaceID != "" && spaceID != "" && view.SpaceID != spaceID {
		return nil, fmt.Errorf("patch space mismatch")
	}
	out, err := s.patches.Decide(patchID, decision, req.ActorID, req.Reason)
	if err != nil {
		return nil, err
	}
	status := StatusApproved
	if decision == DecisionReject {
		status = StatusRejected
	}
	_ = out
	return &DecideResponse{
		ID: ItemID("scenario_patch", patchID), Queue: QueueOrchestration,
		TargetType: "scenario_patch", TargetID: patchID,
		Decision: decision, Status: status,
	}, nil
}

func (s *Service) decideMemory(spaceID, candidateID, decision string, req DecideRequest) (*DecideResponse, error) {
	if s.memory == nil {
		return nil, fmt.Errorf("memory service unavailable")
	}
	actor := strings.TrimSpace(req.ActorID)

	if decision == DecisionApprove && s.requiresMultiSign(spaceID) {
		if first, ok := s.loadPendingApprover("memory", candidateID); ok {
			if first != "" && first == actor {
				return nil, fmt.Errorf("second approver must differ from first approver")
			}
			s.clearPendingApprover("memory", candidateID)
		} else {
			s.savePendingApprover("memory", candidateID, actor)
			return &DecideResponse{
				ID: ItemID("memory", candidateID), Queue: QueueMemory,
				TargetType: "memory", TargetID: candidateID,
				Decision: decision, Status: StatusPendingSecond,
			}, nil
		}
	}

	policy := strings.TrimSpace(req.PolicyProfile)
	if policy == "" {
		policy = "default"
	}
	resp, err := s.memory.Review(candidateID, memory.ReviewRequest{
		Decision:      decision,
		Reason:        req.Reason,
		PolicyProfile: policy,
		ReviewerID:    actor,
		ActorID:       actor,
	})
	if err != nil {
		return nil, err
	}
	status := StatusApproved
	if decision == DecisionReject {
		status = StatusRejected
		s.clearPendingApprover("memory", candidateID)
	}
	_ = spaceID
	_ = resp
	return &DecideResponse{
		ID: ItemID("memory", candidateID), Queue: QueueMemory,
		TargetType: "memory", TargetID: candidateID,
		Decision: decision, Status: status,
	}, nil
}

func (s *Service) decideHarness(spaceID, profileID, decision string, req DecideRequest) (*DecideResponse, error) {
	if s.harness == nil {
		return nil, fmt.Errorf("harness service unavailable")
	}
	view, err := s.harness.Get(profileID)
	if err != nil {
		return nil, err
	}
	if view.SpaceID != "" && spaceID != "" && view.SpaceID != spaceID {
		return nil, fmt.Errorf("profile space mismatch")
	}
	actor := strings.TrimSpace(req.ActorID)

	if decision == DecisionApprove {
		if view.Status == harness.StatusPendingSecond {
			if strings.TrimSpace(view.PromotedBy) != "" && view.PromotedBy == actor {
				return nil, fmt.Errorf("second approver must differ from first approver")
			}
			if _, err := s.harness.Promote(profileID, actor); err != nil {
				return nil, err
			}
			return &DecideResponse{
				ID: ItemID("harness_profile", profileID), Queue: QueueOrchestration,
				TargetType: "harness_profile", TargetID: profileID,
				Decision: decision, Status: StatusApproved,
			}, nil
		}
		if s.requiresMultiSign(spaceID) {
			if _, err := s.harness.MarkPendingSecond(profileID, actor); err != nil {
				return nil, err
			}
			return &DecideResponse{
				ID: ItemID("harness_profile", profileID), Queue: QueueOrchestration,
				TargetType: "harness_profile", TargetID: profileID,
				Decision: decision, Status: StatusPendingSecond,
			}, nil
		}
		if _, err := s.harness.Promote(profileID, actor); err != nil {
			return nil, err
		}
		return &DecideResponse{
			ID: ItemID("harness_profile", profileID), Queue: QueueOrchestration,
			TargetType: "harness_profile", TargetID: profileID,
			Decision: decision, Status: StatusApproved,
		}, nil
	}
	if _, err := s.harness.Reject(profileID, actor, req.Reason); err != nil {
		return nil, err
	}
	return &DecideResponse{
		ID: ItemID("harness_profile", profileID), Queue: QueueOrchestration,
		TargetType: "harness_profile", TargetID: profileID,
		Decision: decision, Status: StatusRejected,
	}, nil
}

func assigneeAuditKey(targetType, targetID string) string {
	return "review-assignee:" + targetType + ":" + targetID
}

func (s *Service) writeAudit(spaceID, actorID, kind string, payload map[string]string) {
	if s.db == nil {
		return
	}
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	b, _ := json.Marshal(payload)
	_ = s.db.Create(&store.AuditLog{
		ID: "aud_" + uuid.NewString(), SpaceID: spaceID,
		ActorID: actorID, EventType: kind,
		PayloadJSON: string(b), CreatedAt: time.Now().UTC(),
	}).Error
}

func (s *Service) saveAssignee(spaceID, targetType, targetID, assigneeID, actorID string) {
	if s.db == nil || assigneeID == "" {
		return
	}
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		spaceID = "local"
	}
	// Replace prior assignment rows for this target (latest-wins via load).
	_ = s.db.Where("event_type = ? AND payload_json LIKE ?", "review.assigned", "%\"targetId\":\""+targetID+"\"%").
		Delete(&store.AuditLog{}).Error
	s.writeAudit(spaceID, actorID, "review.assigned", map[string]string{
		"targetType": targetType, "targetId": targetID, "assigneeId": assigneeID,
		"key": assigneeAuditKey(targetType, targetID),
	})
}

func (s *Service) loadAssignee(targetType, targetID string) (assigneeID string, ok bool) {
	if s.db == nil {
		return "", false
	}
	var row store.AuditLog
	err := s.db.Where("event_type = ? AND payload_json LIKE ?", "review.assigned", "%\"targetId\":\""+targetID+"\"%").
		Order("created_at desc").First(&row).Error
	if err != nil {
		return "", false
	}
	if !strings.Contains(row.PayloadJSON, `"targetType":"`+targetType+`"`) {
		return "", false
	}
	const marker = `"assigneeId":"`
	i := strings.Index(row.PayloadJSON, marker)
	if i < 0 {
		return "", true
	}
	rest := row.PayloadJSON[i+len(marker):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return "", true
	}
	return rest[:j], true
}

func (s *Service) savePendingApprover(targetType, targetID, actor string) {
	if s.db == nil || actor == "" {
		return
	}
	now := time.Now().UTC()
	_ = s.db.Create(&store.AuditLog{
		ID: "aud_" + uuid.NewString(), SpaceID: "local",
		ActorID: actor, EventType: "review.pending_second",
		PayloadJSON: fmt.Sprintf(`{"targetType":%q,"targetId":%q,"firstApprover":%q}`, targetType, targetID, actor),
		CreatedAt:   now,
	}).Error
}

func (s *Service) loadPendingApprover(targetType, targetID string) (string, bool) {
	if s.db == nil {
		return "", false
	}
	var row store.AuditLog
	err := s.db.Where("event_type = ? AND payload_json LIKE ?", "review.pending_second", "%\"targetId\":\""+targetID+"\"%").
		Order("created_at desc").First(&row).Error
	if err != nil {
		return "", false
	}
	if !strings.Contains(row.PayloadJSON, `"targetType":"`+targetType+`"`) {
		return "", false
	}
	if strings.TrimSpace(row.ActorID) != "" {
		return row.ActorID, true
	}
	return "", true
}

func (s *Service) clearPendingApprover(targetType, targetID string) {
	if s.db == nil {
		return
	}
	_ = s.db.Where("event_type = ? AND payload_json LIKE ?", "review.pending_second", "%\"targetId\":\""+targetID+"\"%").
		Delete(&store.AuditLog{}).Error
	_ = targetType
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// NowUnixMilli is for tests.
func NowUnixMilli() int64 { return time.Now().UTC().UnixMilli() }
