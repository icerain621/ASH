package memory

import "fmt"

const (
	FeedbackHitUsed       = "hit_used"
	FeedbackUsedCorrectly = "used_correctly"
	FeedbackWrongUpdate   = "wrong_update"
)

// FeedbackRequest records one memory evaluation event.
type FeedbackRequest struct {
	RunID    string
	TraceID  string
	RecordID string
	Kind     string
}

// FeedbackCounts are the three evaluation counters.
type FeedbackCounts struct {
	HitUsed       int
	UsedCorrectly int
	WrongUpdate   int
}

// RecordFeedback writes hit_used, used_correctly, or wrong_update through mem.feedback.
func (s *Service) RecordFeedback(req FeedbackRequest) error {
	switch req.Kind {
	case FeedbackHitUsed, FeedbackUsedCorrectly, FeedbackWrongUpdate:
	default:
		return fmt.Errorf("invalid feedback kind %q", req.Kind)
	}
	if req.RecordID == "" {
		return fmt.Errorf("recordId is required")
	}
	traceID, err := s.requireRunIfSet(req.RunID)
	if err != nil {
		return err
	}
	if req.TraceID != "" {
		traceID = req.TraceID
	}
	return s.emitRunEvent(req.RunID, traceID, "memory.feedback", map[string]any{
		"component": "mem.feedback",
		"recordId":  req.RecordID,
		"kind":      req.Kind,
	})
}

// TallyFeedback counts the three evaluation kinds.
func TallyFeedback(kinds []string) FeedbackCounts {
	var out FeedbackCounts
	for _, kind := range kinds {
		switch kind {
		case FeedbackHitUsed:
			out.HitUsed++
		case FeedbackUsedCorrectly:
			out.UsedCorrectly++
		case FeedbackWrongUpdate:
			out.WrongUpdate++
		}
	}
	return out
}
