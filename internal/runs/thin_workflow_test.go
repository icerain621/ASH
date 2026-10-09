package runs

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
)

func TestThinFeatureDeliveryRunsTemplates(t *testing.T) {
	svc, _ := testRunsService(t)
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "feature_delivery", ScenarioVersion: "1.0.0"},
		Inputs:   featureDeliveryInputs(t, "thin workflow"),
	})
	if err != nil {
		t.Fatal(err)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != StatusFinished {
		t.Fatalf("status=%q want finished", sum.Status)
	}
	ev := events.NewService(svc.db)
	items, err := ev.ListAfter(created.RunID, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	var templates, packs int
	for _, item := range items {
		switch item.Type {
		case "template.finished":
			templates++
		case "context.packed":
			packs++
		case "scenario.step_deprecated":
			t.Fatalf("thin scenario should not emit deprecation: %+v", item)
		}
	}
	if templates < 3 {
		t.Fatalf("template.finished=%d want >=3", templates)
	}
	if packs < 3 {
		t.Fatalf("context.packed=%d want >=3", packs)
	}
}

func TestLegacyStepKindsEmitDeprecation(t *testing.T) {
	svc, _ := testRunsService(t)
	created, err := svc.Create(CreateRequest{
		Scenario: ScenarioRef{Name: "legacy_step_kinds", ScenarioVersion: "1.0.0"},
		Inputs: map[string]any{
			"issueOrSpec": "legacy adapt",
			"repoRoot":    repoWithEvidence(t, "legacy adapt"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sum, err := svc.Get(created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != StatusFinished {
		t.Fatalf("status=%q want finished", sum.Status)
	}
	ev := events.NewService(svc.db)
	items, err := ev.ListAfter(created.RunID, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	var deps, finished int
	for _, item := range items {
		switch item.Type {
		case "scenario.step_deprecated":
			deps++
		case "template.finished":
			finished++
		}
	}
	if deps < 3 {
		t.Fatalf("scenario.step_deprecated=%d want >=3", deps)
	}
	// Only llm is rewritten onto a template this major version.
	if finished < 1 {
		t.Fatalf("template.finished=%d want >=1", finished)
	}
}
