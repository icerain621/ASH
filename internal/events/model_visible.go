package events

// DeriveModelVisible returns append-only events that may enter LLM context fold.
// Only visibility=model_visible (after NormalizeVisibility) is included.
// ui_only and audit stay out of the model path (DSH-aligned surface fold).
//
// Wiring note (GV03): call sites that assemble agent/LLM context should prefer
// DeriveModelVisible(runID) over raw ListAfter. Harness/agentexec currently have
// no central message fold yet; this is the extension point for BE-39.
func (s *Service) DeriveModelVisible(runID string) ([]Envelope, error) {
	items, err := s.ListAfter(runID, 0, 5000)
	if err != nil {
		return nil, err
	}
	out := make([]Envelope, 0, len(items))
	for _, env := range items {
		if NormalizeVisibility(env.Type, env.Visibility) != VisibilityModelVisible {
			continue
		}
		out = append(out, env)
	}
	return out, nil
}
