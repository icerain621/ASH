package derive

import (
	"fmt"
	"strings"
)

// ValidateReplayParity checks replayed ash_* counters against independent per-type event tallies.
func ValidateReplayParity(events []Event) error {
	if len(events) == 0 {
		return fmt.Errorf("no events to validate")
	}
	snap := Replay(events)
	typeCounts := map[string]int{}
	for _, ev := range events {
		typeCounts[ev.Type]++
	}
	if err := checkCounterSum(snap, "ash_run_total", `status="started"`, typeCounts["run.started"]); err != nil {
		return err
	}
	if err := checkCounterSum(snap, "ash_run_total", `status="finished"`, typeCounts["run.finished"]); err != nil {
		return err
	}
	if err := checkCounterSum(snap, "ash_run_total", `status="failed"`, typeCounts["run.failed"]); err != nil {
		return err
	}
	if err := checkCounterSum(snap, "ash_run_total", `status="canceled"`, typeCounts["run.canceled"]); err != nil {
		return err
	}
	if err := checkCounterSum(snap, "ash_tool_calls_total", "", typeCounts["tool.result"]); err != nil {
		return err
	}
	if err := checkCounterSum(snap, "ash_step_total", "", typeCounts["step.finished"]); err != nil {
		return err
	}
	if err := checkCounterSum(snap, "ash_policy_denied_total", "", typeCounts["policy.denied"]); err != nil {
		return err
	}
	if err := checkInflightGauges(events, snap); err != nil {
		return err
	}
	return nil
}

func checkCounterSum(snap Snapshot, prefix, labelNeedle string, want int) error {
	if want == 0 {
		return nil
	}
	var got float64
	for key, val := range snap.Counters {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if labelNeedle != "" && !strings.Contains(key, labelNeedle) {
			continue
		}
		got += val
	}
	if got != float64(want) {
		return fmt.Errorf("%s%s: replay=%g want=%d", prefix, labelSuffix(labelNeedle), got, want)
	}
	return nil
}

func labelSuffix(needle string) string {
	if needle == "" {
		return ""
	}
	return " (" + needle + ")"
}

func checkInflightGauges(events []Event, snap Snapshot) error {
	scenarioByRun := map[string]string{}
	started := map[string]int{}
	finished := map[string]int{}
	failed := map[string]int{}
	canceled := map[string]int{}
	for _, ev := range events {
		payload := parsePayload(ev.PayloadJSON)
		if ev.Type == "run.started" {
			if name := stringField(payload, "scenario.name"); name != "" {
				scenarioByRun[ev.RunID] = name
				started[name]++
			}
			continue
		}
		scenario := scenarioByRun[ev.RunID]
		if scenario == "" {
			continue
		}
		switch ev.Type {
		case "run.finished":
			finished[scenario]++
		case "run.failed":
			failed[scenario]++
		case "run.canceled":
			canceled[scenario]++
		}
	}
	for scenario, startN := range started {
		key := fmt.Sprintf(`ash_run_inflight{scenario=%q}`, scenario)
		want := float64(startN - finished[scenario] - failed[scenario] - canceled[scenario])
		got := snap.Gauges[key]
		if got != want {
			return fmt.Errorf("inflight %s: replay=%g want=%g", scenario, got, want)
		}
	}
	return nil
}
