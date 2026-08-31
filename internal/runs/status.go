package runs

import (
	"errors"
	"fmt"

	"github.com/ash-repwiki/ash/internal/store"
)

// Run lifecycle statuses (R-05).
const (
	StatusRunning          = "running"
	StatusWaitingApproval  = "waiting_approval"
	StatusFinished         = "finished"
	StatusFailed           = "failed"
	StatusCanceled         = "canceled"
)

var (
	ErrIllegalStatusTransition = errors.New("illegal run status transition")
	ErrRunCanceled             = errors.New("run canceled")
)

func isTerminalRunStatus(status string) bool {
	switch status {
	case StatusFinished, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

func canTransition(from, to string) bool {
	if from == to {
		return false
	}
	switch from {
	case StatusRunning:
		switch to {
		case StatusWaitingApproval, StatusFinished, StatusFailed, StatusCanceled:
			return true
		}
	case StatusWaitingApproval:
		switch to {
		case StatusRunning, StatusCanceled:
			return true
		}
	case StatusFailed:
		return to == StatusRunning
	}
	return false
}

func applyRunStatus(rec *store.RunRecord, to string) error {
	if rec == nil {
		return fmt.Errorf("%w: nil run record", ErrIllegalStatusTransition)
	}
	if !canTransition(rec.Status, to) {
		return fmt.Errorf("%w: %s -> %s", ErrIllegalStatusTransition, rec.Status, to)
	}
	rec.Status = to
	return nil
}

func (s *Service) refreshRunStatus(rec *store.RunRecord) error {
	if s == nil || rec == nil || rec.ID == "" {
		return fmt.Errorf("refresh run status: invalid args")
	}
	var row store.RunRecord
	if err := s.gdb().Select("id", "status").First(&row, "id = ?", rec.ID).Error; err != nil {
		return err
	}
	rec.Status = row.Status
	return nil
}

// trySetRunStatus reloads status from DB then applies a legal transition.
// Canceled / other illegal targets return ErrRunCanceled or ErrIllegalStatusTransition.
func (s *Service) trySetRunStatus(rec *store.RunRecord, to string) error {
	if err := s.refreshRunStatus(rec); err != nil {
		return err
	}
	if rec.Status == StatusCanceled {
		return ErrRunCanceled
	}
	return applyRunStatus(rec, to)
}

// observeCanceled reloads status and returns ErrRunCanceled when the run was canceled mid-flight.
func (s *Service) observeCanceled(rec *store.RunRecord) error {
	if err := s.refreshRunStatus(rec); err != nil {
		return err
	}
	if rec.Status == StatusCanceled {
		return ErrRunCanceled
	}
	return nil
}

func canApprove(status string) bool {
	return status == StatusWaitingApproval
}

func canResume(status string) bool {
	return status == StatusFailed
}

// ApplyStatusTransition applies a legal in-memory status change (e.g. waker cancel).
func ApplyStatusTransition(rec *store.RunRecord, to string) error {
	return applyRunStatus(rec, to)
}
