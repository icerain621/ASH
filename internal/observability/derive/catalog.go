package derive

// MetricKind classifies derived Prometheus series.
type MetricKind string

const (
	MetricCounter   MetricKind = "counter"
	MetricGauge     MetricKind = "gauge"
	MetricHistogram MetricKind = "histogram"
)

// Op describes how an event updates a metric.
type Op string

const (
	OpInc      Op = "inc"
	OpDec      Op = "dec"
	OpObserve  Op = "observe"
	OpSetOne   Op = "set_one"
)

// LabelSpec binds a label name to a payload JSON field or a static value.
type LabelSpec struct {
	Name      string
	Static    string
	JSONField string
}

// Rule maps one event type to a metric update (table-driven per appendix D).
type Rule struct {
	EventType string
	Metric    string
	Kind      MetricKind
	Op        Op
	Labels    []LabelSpec
	ValueJSON string // payload field for histogram observe
}

// Catalog is the frozen TR0 event→metric mapping (appendix D §4).
func Catalog() []Rule {
	return []Rule{
		{
			EventType: "run.started",
			Metric:    "ash_run_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "status", Static: "started"}, {Name: "scenario", JSONField: "scenario.name"}},
		},
		{
			EventType: "run.started",
			Metric:    "ash_run_inflight",
			Kind:      MetricGauge,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "scenario", JSONField: "scenario.name"}},
		},
		{
			EventType: "run.finished",
			Metric:    "ash_run_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "status", Static: "finished"}, {Name: "scenario", JSONField: "_scenario"}},
		},
		{
			EventType: "run.finished",
			Metric:    "ash_run_inflight",
			Kind:      MetricGauge,
			Op:        OpDec,
			Labels:    []LabelSpec{{Name: "scenario", JSONField: "_scenario"}},
		},
		{
			EventType: "run.finished",
			Metric:    "ash_run_duration_ms",
			Kind:      MetricHistogram,
			Op:        OpObserve,
			Labels:    []LabelSpec{{Name: "scenario", JSONField: "_scenario"}},
			ValueJSON: "durationMs",
		},
		{
			EventType: "run.failed",
			Metric:    "ash_run_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "status", Static: "failed"}, {Name: "scenario", JSONField: "_scenario"}},
		},
		{
			EventType: "run.failed",
			Metric:    "ash_run_inflight",
			Kind:      MetricGauge,
			Op:        OpDec,
			Labels:    []LabelSpec{{Name: "scenario", JSONField: "_scenario"}},
		},
		{
			EventType: "run.canceled",
			Metric:    "ash_run_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "status", Static: "canceled"}, {Name: "scenario", JSONField: "_scenario"}},
		},
		{
			EventType: "run.canceled",
			Metric:    "ash_run_inflight",
			Kind:      MetricGauge,
			Op:        OpDec,
			Labels:    []LabelSpec{{Name: "scenario", JSONField: "_scenario"}},
		},
		{
			EventType: "step.finished",
			Metric:    "ash_step_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "scenario", JSONField: "_scenario"},
				{Name: "stepId", JSONField: "stepId"},
				{Name: "status", JSONField: "_step_status"},
			},
		},
		{
			EventType: "step.finished",
			Metric:    "ash_step_duration_ms",
			Kind:      MetricHistogram,
			Op:        OpObserve,
			Labels: []LabelSpec{
				{Name: "scenario", JSONField: "_scenario"},
				{Name: "stepId", JSONField: "stepId"},
				{Name: "role", JSONField: "role"},
			},
			ValueJSON: "durationMs",
		},
		{
			EventType: "tool.result",
			Metric:    "ash_tool_calls_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "tool", JSONField: "tool"},
				{Name: "ok", JSONField: "_tool_ok"},
			},
		},
		{
			EventType: "tool.result",
			Metric:    "ash_tool_duration_ms",
			Kind:      MetricHistogram,
			Op:        OpObserve,
			Labels:    []LabelSpec{{Name: "tool", JSONField: "tool"}},
			ValueJSON: "durationMs",
		},
		{
			EventType: "policy.denied",
			Metric:    "ash_policy_denied_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "target", JSONField: "target"},
				{Name: "reason", JSONField: "reason"},
			},
		},
		{
			EventType: "model.usage",
			Metric:    "ash_token_in_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "provider", JSONField: "providerId"},
				{Name: "model", JSONField: "modelId"},
			},
			ValueJSON: "inTokens",
		},
		{
			EventType: "model.usage",
			Metric:    "ash_token_out_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "provider", JSONField: "providerId"},
				{Name: "model", JSONField: "modelId"},
			},
			ValueJSON: "outTokens",
		},
		{
			EventType: "rag.results",
			Metric:    "ash_rag_latency_ms",
			Kind:      MetricHistogram,
			Op:        OpObserve,
			Labels:    []LabelSpec{{Name: "source", Static: "mixed"}},
			ValueJSON: "durationMs",
		},
		{
			EventType: "rag.results",
			Metric:    "ash_rag_citation_missing_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "scenario", JSONField: "_scenario"},
				{Name: "stepId", JSONField: "_stepId"},
			},
			ValueJSON: "_citation_missing",
		},
		{
			EventType: "rag.retrieved",
			Metric:    "ash_rag_queries_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "mode", JSONField: "retrievalMode"}},
		},
		{
			EventType: "rag.retrieved",
			Metric:    "ash_rag_fts_fallback_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "mode", Static: "chunk"}},
			ValueJSON: "_fts_fallback",
		},
		{
			EventType: "plugin.export_failed",
			Metric:    "ash_plugin_export_failures_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "plugin", JSONField: "pluginId"}},
		},
		{
			EventType: "memory.candidate_created",
			Metric:    "ash_memory_candidates_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "layer", JSONField: "layer"}},
		},
		{
			EventType: "memory.candidate_created",
			Metric:    "ash_memory_unreviewed_backlog",
			Kind:      MetricGauge,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "layer", JSONField: "layer"}},
		},
		{
			EventType: "memory.candidate_created",
			Metric:    "ash_memory_missing_evidence_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "layer", JSONField: "layer"}},
			ValueJSON: "_missing_evidence",
		},
		{
			EventType: "memory.reviewed",
			Metric:    "ash_memory_reviews_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "layer", JSONField: "layer"},
				{Name: "decision", JSONField: "decision"},
			},
		},
		{
			EventType: "memory.reviewed",
			Metric:    "ash_memory_review_latency_ms",
			Kind:      MetricHistogram,
			Op:        OpObserve,
			Labels:    []LabelSpec{{Name: "layer", JSONField: "layer"}},
			ValueJSON: "latencyMs",
		},
		{
			EventType: "memory.reviewed",
			Metric:    "ash_memory_unreviewed_backlog",
			Kind:      MetricGauge,
			Op:        OpDec,
			Labels:    []LabelSpec{{Name: "layer", JSONField: "layer"}},
		},
		{
			EventType: "memory.hit_used",
			Metric:    "ash_memory_hit_used_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "layer", JSONField: "layer"}},
			ValueJSON: "count",
		},
		{
			EventType: "memory.deprecated",
			Metric:    "ash_memory_deprecated_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "layer", JSONField: "layer"},
				{Name: "reason", JSONField: "reason"},
			},
		},
		{
			EventType: "memory.ttl_expired",
			Metric:    "ash_memory_ttl_expired_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "layer", JSONField: "layer"},
				{Name: "reason", JSONField: "reason"},
			},
		},
		{
			EventType: "memory.confidence_adjusted",
			Metric:    "ash_memory_confidence_adjusted_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "layer", JSONField: "layer"}},
		},
		{
			EventType: "memory.query",
			Metric:    "ash_memory_queries_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels:    []LabelSpec{{Name: "layersKey", JSONField: "layersKey"}},
		},
		{
			EventType: "memory.query",
			Metric:    "ash_memory_query_latency_ms",
			Kind:      MetricHistogram,
			Op:        OpObserve,
			Labels:    []LabelSpec{{Name: "layersKey", JSONField: "layersKey"}},
			ValueJSON: "latencyMs",
		},
		{
			EventType: "memory.migrated",
			Metric:    "ash_memory_migration_runs_total",
			Kind:      MetricCounter,
			Op:        OpInc,
			Labels: []LabelSpec{
				{Name: "from", JSONField: "from"},
				{Name: "to", JSONField: "to"},
				{Name: "ok", JSONField: "_migration_ok"},
			},
		},
	}
}
