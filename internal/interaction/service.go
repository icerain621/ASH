package interaction

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

type Service struct {
	db     *store.DB
	events *events.Service
}

func NewService(db *store.DB, ev *events.Service) *Service {
	return &Service{db: db, events: ev}
}

func (s *Service) q() *gorm.DB {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.DB
}

// EnsureThread creates or returns the main thread for a run (idempotent).
func (s *Service) EnsureThread(req EnsureRequest) (*Thread, bool, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return nil, false, fmt.Errorf("runId is required")
	}
	space := firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	sessionID := strings.TrimSpace(req.SessionID)

	var row store.InteractionThread
	err := s.q().Where("run_id = ? AND kind = ?", runID, ThreadKindMain).First(&row).Error
	if err == nil {
		return threadFromRow(row), false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	now := time.Now().UTC()
	row = store.InteractionThread{
		ID: "th_" + uuid.NewString(), SpaceID: space, SessionID: sessionID, RunID: runID,
		Kind: ThreadKindMain, Status: ThreadStatusOpen, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.q().Create(&row).Error; err != nil {
		// Race: another writer may have inserted main thread.
		var again store.InteractionThread
		if e2 := s.q().Where("run_id = ? AND kind = ?", runID, ThreadKindMain).First(&again).Error; e2 == nil {
			return threadFromRow(again), false, nil
		}
		return nil, false, err
	}
	return threadFromRow(row), true, nil
}

// Fork copies a thread identity (same run/session) with parentThreadId set.
// Multiple forks are allowed. Main remains unique by EnsureThread, not this path.
func (s *Service) Fork(threadID string) (*Thread, error) {
	parent, err := s.GetThread(threadID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := store.InteractionThread{
		ID: "th_" + uuid.NewString(), SpaceID: parent.SpaceID, SessionID: parent.SessionID,
		RunID: parent.RunID, Kind: ThreadKindFork, ParentThreadID: parent.ID,
		Status: ThreadStatusOpen, HeadSeq: parent.HeadSeq, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.q().Create(&row).Error; err != nil {
		return nil, err
	}
	return threadFromRow(row), nil
}

// FoldThread loads run events and returns a pure fold projection.
func (s *Service) FoldThread(threadID string) (*FoldResult, error) {
	th, err := s.GetThread(threadID)
	if err != nil {
		return nil, err
	}
	if s.events == nil {
		return nil, fmt.Errorf("events service is not configured")
	}
	evs, err := s.events.ListAfter(th.RunID, 0, 5000)
	if err != nil {
		return nil, err
	}
	// Sealed folds are frozen at head_seq so post-seal observability events
	// (interaction.thread_sealed / replay_mismatch) do not change the digest.
	if th.Status == ThreadStatusSealed && th.HeadSeq > 0 {
		filtered := make([]events.Envelope, 0, len(evs))
		for _, ev := range evs {
			if ev.Seq <= th.HeadSeq {
				filtered = append(filtered, ev)
			}
		}
		evs = filtered
	}
	out := FoldEvents(th.SessionID, th.ID, th.RunID, th.SpaceID, evs)
	return &out, nil
}

// GetThread loads a persisted thread.
func (s *Service) GetThread(threadID string) (*Thread, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, fmt.Errorf("threadId is required")
	}
	var row store.InteractionThread
	if err := s.q().First(&row, "id = ?", threadID).Error; err != nil {
		return nil, fmt.Errorf("thread not found: %w", err)
	}
	return threadFromRow(row), nil
}

// ListThreads returns interaction_threads for a session ordered by created_at.
func (s *Service) ListThreads(sessionID string) ([]Thread, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionId is required")
	}
	var rows []store.InteractionThread
	if err := s.q().Where("session_id = ?", sessionID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Thread, 0, len(rows))
	for _, row := range rows {
		out = append(out, *threadFromRow(row))
	}
	return out, nil
}

// ListThreadsByRun returns interaction_threads for a run ordered by created_at.
func (s *Service) ListThreadsByRun(runID string) ([]Thread, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("runId is required")
	}
	var rows []store.InteractionThread
	if err := s.q().Where("run_id = ?", runID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Thread, 0, len(rows))
	for _, row := range rows {
		out = append(out, *threadFromRow(row))
	}
	return out, nil
}

// ListMemoryLinks returns fold links for a thread.
func (s *Service) ListMemoryLinks(threadID string) ([]MemoryLink, error) {
	fold, err := s.FoldThread(threadID)
	if err != nil {
		return nil, err
	}
	return fold.Links, nil
}

// ByRun resolves the main thread for a run (Ensure if missing).
func (s *Service) ByRun(runID string) (*ByRunView, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("runId is required")
	}
	space := "local"
	sessionID := ""
	var rec store.RunRecord
	if err := s.q().First(&rec, "id = ?", runID).Error; err == nil {
		space = firstNonEmpty(rec.SpaceID, "local")
	}
	// Prefer agent.session audit binding when present.
	var audit store.AuditLog
	if err := s.q().Where("event_type = ? AND run_id = ?", "agent.session", runID).
		Order("created_at DESC").First(&audit).Error; err == nil {
		sessionID = audit.ID
		if audit.SpaceID != "" {
			space = audit.SpaceID
		}
	}
	th, _, err := s.EnsureThread(EnsureRequest{SpaceID: space, SessionID: sessionID, RunID: runID})
	if err != nil {
		return nil, err
	}
	return &ByRunView{RunID: runID, Thread: *th}, nil
}

func threadFromRow(row store.InteractionThread) *Thread {
	return &Thread{
		ID: row.ID, SpaceID: row.SpaceID, SessionID: row.SessionID, RunID: row.RunID,
		Kind: row.Kind, ParentThreadID: row.ParentThreadID, Status: row.Status,
		Digest: row.Digest, HeadSeq: row.HeadSeq,
		CreatedAt: row.CreatedAt.Unix(), UpdatedAt: row.UpdatedAt.Unix(),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
