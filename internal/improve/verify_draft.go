package improve

import "fmt"

// DraftFromVerifyFailure creates a draft improve proposal after a verify step fails.
func (s *Service) DraftFromVerifyFailure(spaceID, runID, stepID, detail string) (string, error) {
	title := fmt.Sprintf("Verify failed on %s", stepID)
	desc := fmt.Sprintf("Automatic draft from verify step %q.\n\n%s", stepID, detail)
	view, err := s.Create(CreateProposalRequest{
		Title:         title,
		Description:   desc,
		BaselineRunID: runID,
		ChangeSummary: "verify.onFail=improve",
		SpaceID:       spaceID,
		ActorID:       "ash-runner",
	})
	if err != nil {
		return "", err
	}
	return view.ID, nil
}
