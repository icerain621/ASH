package runs

// ImproveDrafter creates an improve proposal draft after verify failures (wired from API layer).
type ImproveDrafter interface {
	DraftFromVerifyFailure(spaceID, runID, stepID, detail string) (proposalID string, err error)
}

func (s *Service) SetImproveDrafter(d ImproveDrafter) {
	if s != nil {
		s.improve = d
	}
}
