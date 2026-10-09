package memory

import (
	"fmt"
	"strings"
)

// ProposeRequest is the mem.propose write path: always creates a candidate.
type ProposeRequest struct {
	CreateCandidateRequest
}

// Propose requires at least one non-trajectory evidence item, then creates a candidate.
func (s *Service) Propose(req ProposeRequest) (*CreateCandidateResponse, error) {
	if err := requireProposeEvidence(req.Evidence); err != nil {
		return nil, err
	}
	return s.CreateCandidate(req.CreateCandidateRequest)
}

func requireProposeEvidence(evidence []EvidenceInput) error {
	if len(evidence) == 0 {
		return fmt.Errorf("mem.propose requires evidence")
	}
	for _, ev := range evidence {
		kind := strings.ToLower(strings.TrimSpace(ev.Kind))
		if kind == "trajectory" || kind == "trace" || kind == "run_transcript" || kind == "transcript" {
			return fmt.Errorf("mem.propose rejects trajectory evidence kind %q", ev.Kind)
		}
		if strings.TrimSpace(ev.Ref) == "" {
			return fmt.Errorf("evidence ref is required")
		}
	}
	return nil
}
