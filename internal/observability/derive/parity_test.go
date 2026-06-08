package derive

import (
	"fmt"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestValidateReplayParity_runLifecycle(t *testing.T) {
	events := []Event{
		{
			RunID: "run_1", Type: "run.started",
			PayloadJSON: `{"scenario":{"name":"feature_delivery","scenarioVersion":"1.0.0"},"policyProfile":"default","inputsDigest":"d1"}`,
		},
		{
			RunID: "run_1", Type: "step.finished",
			PayloadJSON: `{"stepId":"code.implement","role":"Coder","ok":true,"durationMs":1200}`,
		},
		{
			RunID: "run_1", Type: "run.finished",
			PayloadJSON: `{"ok":true,"durationMs":5000}`,
		},
	}
	if err := ValidateReplayParity(events); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReplayParity_toolAndPolicy(t *testing.T) {
	events := []Event{
		{
			RunID: "run_1", Type: "run.started",
			PayloadJSON: `{"scenario":{"name":"hotfix","scenarioVersion":"1"},"policyProfile":"strict","inputsDigest":"d"}`,
		},
		{RunID: "run_1", Type: "tool.result", PayloadJSON: `{"tool":"git.status","ok":false,"durationMs":42}`},
		{RunID: "run_1", Type: "policy.denied", PayloadJSON: `{"target":"tool","reason":"POLICY_DENIED","action":"deny"}`},
		{RunID: "run_1", Type: "run.finished", PayloadJSON: `{"ok":false,"durationMs":100}`},
	}
	if err := ValidateReplayParity(events); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReplayParity_matchesDBEvents(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	events := []Event{
		{
			RunID: "db_parity_run", Type: "run.started",
			PayloadJSON: `{"scenario":{"name":"feature_delivery","scenarioVersion":"1"},"policyProfile":"default","inputsDigest":"d"}`,
		},
		{RunID: "db_parity_run", Type: "tool.result", PayloadJSON: `{"tool":"git.status","ok":true,"durationMs":10}`},
		{RunID: "db_parity_run", Type: "run.finished", PayloadJSON: `{"ok":true,"durationMs":200}`},
	}
	for i, ev := range events {
		row := store.RunEvent{
			ID:          fmtID("ev", i),
			RunID:       ev.RunID,
			Seq:         int64(i + 1),
			TS:          int64(1000 + i),
			Type:        ev.Type,
			Severity:    "info",
			PayloadJSON: ev.PayloadJSON,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := LoadFromDB(db.DB, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var subset []Event
	for _, ev := range loaded {
		if ev.RunID == "db_parity_run" {
			subset = append(subset, ev)
		}
	}
	if len(subset) != len(events) {
		t.Fatalf("loaded=%d want %d", len(subset), len(events))
	}
	if err := ValidateReplayParity(subset); err != nil {
		t.Fatal(err)
	}
}

func fmtID(prefix string, i int) string {
	return fmt.Sprintf("%s_%d", prefix, i)
}
