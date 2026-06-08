package derive

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Event is one run_events row used for offline metric replay.
type Event struct {
	RunID       string
	Type        string
	PayloadJSON string
}

// Snapshot holds replayed metric state.
type Snapshot struct {
	Counters    map[string]float64
	Gauges      map[string]float64
	Histograms  map[string][]float64
}

// Replay applies the catalog to events in order (per-run scenario context is tracked).
func Replay(events []Event) Snapshot {
	snap := Snapshot{
		Counters:   map[string]float64{},
		Gauges:     map[string]float64{},
		Histograms: map[string][]float64{},
	}
	rules := Catalog()
	scenarioByRun := map[string]string{}
	stepByRun := map[string]string{}
	for _, ev := range events {
		payload := parsePayload(ev.PayloadJSON)
		if ev.Type == "run.started" {
			if name := stringField(payload, "scenario.name"); name != "" {
				scenarioByRun[ev.RunID] = name
			}
		}
		if ev.Type == "step.started" {
			if stepID := stringField(payload, "stepId"); stepID != "" {
				stepByRun[ev.RunID] = stepID
			}
		}
		enriched := enrichPayload(payload, scenarioByRun[ev.RunID], stepByRun[ev.RunID])
		payloads := []map[string]any{enriched}
		if ev.Type == "memory.hit_used" {
			payloads = expandHitsByLayer(enriched)
		}
		for _, p := range payloads {
			for _, rule := range rules {
				if rule.EventType != ev.Type {
					continue
				}
				applyRule(&snap, rule, p)
			}
		}
	}
	return snap
}

func expandHitsByLayer(payload map[string]any) []map[string]any {
	raw, ok := payload["hitsByLayer"].(map[string]any)
	if !ok || len(raw) == 0 {
		layer := stringField(payload, "layer")
		if layer == "" {
			layer = "mixed"
		}
		count := numberField(payload, "count")
		if count <= 0 {
			count = 1
		}
		return []map[string]any{{"layer": layer, "count": count}}
	}
	keys := make([]string, 0, len(raw))
	for layer := range raw {
		keys = append(keys, layer)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, layer := range keys {
		count := numberField(map[string]any{"n": raw[layer]}, "n")
		if count <= 0 {
			continue
		}
		out = append(out, map[string]any{"layer": layer, "count": count})
	}
	if len(out) == 0 {
		return []map[string]any{{"layer": "mixed", "count": float64(1)}}
	}
	return out
}

func enrichPayload(payload map[string]any, scenario, stepID string) map[string]any {
	out := map[string]any{}
	for k, v := range payload {
		out[k] = v
	}
	if scenario != "" {
		out["_scenario"] = scenario
	}
	if stepID != "" {
		out["_stepId"] = stepID
	}
	if ok, exists := out["ok"].(bool); exists {
		if ok {
			out["_step_status"] = "ok"
			out["_tool_ok"] = "true"
		} else {
			out["_step_status"] = "failed"
			out["_tool_ok"] = "false"
		}
	}
	if missing, ok := out["citationsMissing"].(bool); ok && missing {
		out["_citation_missing"] = float64(1)
	}
	if _, hasEvidence := out["evidenceCount"]; hasEvidence && numberField(out, "evidenceCount") == 0 {
		out["_missing_evidence"] = float64(1)
	}
	if ok, exists := out["ok"].(bool); exists {
		out["_migration_ok"] = strconv.FormatBool(ok)
	}
	if mode := stringField(out, "retrievalMode"); mode == "chunk" {
		out["_fts_fallback"] = float64(1)
	}
	return out
}

func applyRule(snap *Snapshot, rule Rule, payload map[string]any) {
	labels := renderLabels(rule.Labels, payload)
	key := metricKey(rule.Metric, labels)
	switch rule.Kind {
	case MetricCounter:
		delta := float64(1)
		if rule.ValueJSON != "" {
			delta = numberField(payload, rule.ValueJSON)
			if delta == 0 {
				return
			}
		}
		snap.Counters[key] += delta
	case MetricGauge:
		switch rule.Op {
		case OpInc:
			snap.Gauges[key]++
		case OpDec:
			snap.Gauges[key]--
		}
	case MetricHistogram:
		val := numberField(payload, rule.ValueJSON)
		if val < 0 {
			return
		}
		snap.Histograms[key] = append(snap.Histograms[key], val)
	}
}

func renderLabels(specs []LabelSpec, payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, spec := range specs {
		if spec.Static != "" {
			out[spec.Name] = spec.Static
			continue
		}
		if spec.JSONField != "" {
			if v := stringField(payload, spec.JSONField); v != "" {
				out[spec.Name] = v
			}
		}
	}
	return out
}

func metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, labels[k]))
	}
	return name + "{" + strings.Join(parts, ",") + "}"
}

func parsePayload(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func stringField(payload map[string]any, path string) string {
	v := fieldValue(payload, path)
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}

func numberField(payload map[string]any, path string) float64 {
	v := fieldValue(payload, path)
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case bool:
		if t {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func fieldValue(payload map[string]any, path string) any {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, ".")
	var cur any = payload
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[part]
		if !ok {
			return nil
		}
	}
	return cur
}

// PrometheusText renders replayed metrics as Prometheus exposition text.
func (s Snapshot) PrometheusText() string {
	var b strings.Builder
	b.WriteString("# ASH event-derived metrics (offline replay)\n")
	keys := make([]string, 0, len(s.Counters))
	for k := range s.Counters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("%s %g\n", k, s.Counters[k]))
	}
	gaugeKeys := make([]string, 0, len(s.Gauges))
	for k := range s.Gauges {
		gaugeKeys = append(gaugeKeys, k)
	}
	sort.Strings(gaugeKeys)
	for _, k := range gaugeKeys {
		b.WriteString(fmt.Sprintf("%s %g\n", k, s.Gauges[k]))
	}
	histKeys := make([]string, 0, len(s.Histograms))
	for k := range s.Histograms {
		histKeys = append(histKeys, k)
	}
	sort.Strings(histKeys)
	for _, k := range histKeys {
		vals := s.Histograms[k]
		var sum float64
		for _, v := range vals {
			sum += v
		}
		b.WriteString(fmt.Sprintf("%s_count %d\n", k, len(vals)))
		b.WriteString(fmt.Sprintf("%s_sum %g\n", k, sum))
	}
	return b.String()
}
