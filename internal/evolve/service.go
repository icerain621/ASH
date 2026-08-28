package evolve

import (
	"fmt"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/harness"
	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/scenariopatch"
	"github.com/ash-repwiki/ash/internal/store"
)

type Item struct {
	ID         string `json:"id"`
	Queue      string `json:"queue"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Title      string `json:"title"`
	Summary    string `json:"summary,omitempty"`
	Diff       string `json:"diff,omitempty"`
	Status     string `json:"status"`
	SpaceID    string `json:"spaceId"`
	CreatedAt  int64  `json:"createdAt"`
}

type ListResponse struct {
	Items []Item `json:"items"`
	Queue string `json:"queue,omitempty"`
}

type DecideRequest struct {
	Decision      string `json:"decision"`
	Reason        string `json:"reason"`
	PolicyProfile string `json:"policyProfile,omitempty"`
	ActorID       string `json:"actorId,omitempty"`
}

type DecideResponse struct {
	ID         string `json:"id"`
	Queue      string `json:"queue"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Decision   string `json:"decision"`
	Status     string `json:"status"`
}

type Service struct {
	db      *store.DB
	memory  *memory.Service
	harness *harness.Service
	patches *scenariopatch.Service
}

func NewService(db *store.DB, mem *memory.Service, har *harness.Service, patches *scenariopatch.Service) *Service {
	return &Service{db: db, memory: mem, harness: har, patches: patches}
}

func (s *Service) WithContext(db *store.DB, mem *memory.Service, har *harness.Service, patches *scenariopatch.Service) *Service {
	if s == nil {
		return nil
	}
	out := *s
	if db != nil {
		out.db = db
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
	out := make([]Item, 0, len(resp.Items))
	for _, it := range resp.Items {
		out = append(out, Item{
			ID:         ItemID("memory", it.ID),
			Queue:      QueueMemory,
			TargetType: "memory",
			TargetID:   it.ID,
			Title:      firstNonEmpty(it.Title, it.ID),
			Summary:    truncate(it.Body, 200),
			Status:     StatusPending,
			SpaceID:    spaceID,
			CreatedAt:  it.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) listOrchestration(spaceID string, limit int) ([]Item, error) {
	out := make([]Item, 0, limit)
	if s.harness != nil {
		views, err := s.harness.List(spaceID, harness.StatusInReview, "")
		if err != nil {
			return nil, err
		}
		for _, v := range views {
			if len(out) >= limit {
				break
			}
			out = append(out, Item{
				ID:         ItemID("harness_profile", v.ID),
				Queue:      QueueOrchestration,
				TargetType: "harness_profile",
				TargetID:   v.ID,
				Title:      fmt.Sprintf("%s@v%d", v.Name, v.Version),
				Summary:    "harness profile awaiting orchestration review",
				Status:     StatusPending,
				SpaceID:    v.SpaceID,
				CreatedAt:  v.UpdatedAt,
			})
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
			out = append(out, Item{
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
			})
		}
	}
	return out, nil
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
	switch tt {
	case "memory":
		return s.decideMemory(space, tid, decision, req)
	case "harness_profile":
		return s.decideHarness(space, tid, decision, req)
	case "scenario_patch":
		return s.decideScenarioPatch(space, tid, decision, req)
	default:
		return nil, fmt.Errorf("unsupported targetType %q", tt)
	}
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
	policy := strings.TrimSpace(req.PolicyProfile)
	if policy == "" {
		policy = "default"
	}
	resp, err := s.memory.Review(candidateID, memory.ReviewRequest{
		Decision:      decision,
		Reason:        req.Reason,
		PolicyProfile: policy,
		ReviewerID:    req.ActorID,
		ActorID:       req.ActorID,
	})
	if err != nil {
		return nil, err
	}
	status := StatusApproved
	if decision == DecisionReject {
		status = StatusRejected
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
	if decision == DecisionApprove {
		if _, err := s.harness.Promote(profileID, req.ActorID); err != nil {
			return nil, err
		}
		return &DecideResponse{
			ID: ItemID("harness_profile", profileID), Queue: QueueOrchestration,
			TargetType: "harness_profile", TargetID: profileID,
			Decision: decision, Status: StatusApproved,
		}, nil
	}
	if _, err := s.harness.Reject(profileID, req.ActorID, req.Reason); err != nil {
		return nil, err
	}
	return &DecideResponse{
		ID: ItemID("harness_profile", profileID), Queue: QueueOrchestration,
		TargetType: "harness_profile", TargetID: profileID,
		Decision: decision, Status: StatusRejected,
	}, nil
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
