package runs

import "testing"

func TestM1ScenariosExecuteWithStaticAgent(t *testing.T) {
	cases := []struct {
		name      string
		agentStep string
		issue     string
	}{
		{name: "hotfix", agentStep: "code.fix", issue: "hotfix service crash"},
		{name: "security_patch", agentStep: "code.patch", issue: "security patch vulnerable dependency"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := testRunsService(t)
			repo := repoWithEvidence(t, tc.issue)
			created, err := svc.Create(CreateRequest{
				Scenario: ScenarioRef{Name: tc.name, ScenarioVersion: "1.0.0"},
				Inputs: map[string]any{
					"issueOrSpec": tc.issue,
					"repoRoot":    repo,
				},
				Repo: &RepoRef{Root: repo},
			})
			if err != nil {
				t.Fatal(err)
			}

			sum, err := svc.Get(created.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if sum.Status != "finished" {
				t.Fatalf("status=%q want finished", sum.Status)
			}

			manifest, err := svc.Artifacts(created.RunID)
			if err != nil {
				t.Fatal(err)
			}
			types := map[string]int{}
			for _, art := range manifest.Artifacts {
				types[art.Type]++
			}
			for _, typ := range []string{"diff", "test_report", "release_notes", "rollback_plan"} {
				if types[typ] == 0 {
					t.Fatalf("missing required artifact type %q in %+v", typ, manifest.Artifacts)
				}
			}
			if types["step_output"] == 0 {
				t.Fatalf("missing step_output artifact in %+v", manifest.Artifacts)
			}

			tasks, err := svc.AgentTasks(created.RunID)
			if err != nil {
				t.Fatal(err)
			}
			foundAgent := false
			for _, task := range tasks {
				if task.StepID == tc.agentStep && task.Status == "success" {
					foundAgent = true
				}
			}
			if !foundAgent {
				t.Fatalf("missing successful agent task for step %s: %+v", tc.agentStep, tasks)
			}

			evs, err := svc.Events().ListAfter(created.RunID, 0, 500)
			if err != nil {
				t.Fatal(err)
			}
			foundCitation := false
			for _, ev := range evs {
				if ev.Type == "citation.bound" {
					foundCitation = true
					break
				}
			}
			if !foundCitation {
				t.Fatal("expected citation.bound event")
			}
		})
	}
}
