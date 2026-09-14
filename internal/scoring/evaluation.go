package scoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
	"gorm.io/gorm"
)

// EvaluationRequest scopes a space evaluation window.
type EvaluationRequest struct {
	SpaceID string
	From    time.Time
	To      time.Time
}

// Evaluation is the space-level dual-core scoreboard (BE-30).
type Evaluation struct {
	SpaceID     string               `json:"spaceId"`
	From        string               `json:"from"`
	To          string               `json:"to"`
	GeneratedAt string               `json:"generatedAt"`
	Dimensions  []DimensionScore     `json:"dimensions"`
	Health      EvaluationHealth     `json:"health"`
	Scenarios   []ScenarioEvaluation `json:"scenarios"`
	DataQuality []DataQualityNote    `json:"dataQuality"`
}

// DimensionScore is one of quality|safety|efficiency|governance.
type DimensionScore struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Score   float64        `json:"score"`
	Status  string         `json:"status"`
	Signals []SignalMetric `json:"signals"`
}

// SignalMetric is a contributing raw signal.
type SignalMetric struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Numerator   int64   `json:"numerator,omitempty"`
	Denominator int64   `json:"denominator,omitempty"`
}

// EvaluationHealth covers seal / replay / citation / memory-link health.
type EvaluationHealth struct {
	ThreadSealRate       SignalMetric     `json:"threadSealRate"`
	ReplayMismatchRate   SignalMetric     `json:"replayMismatchRate"`
	CitationMissingTotal int64            `json:"citationMissingTotal"`
	MemoryLinksByType    map[string]int64 `json:"memoryLinksByType"`
}

// ScenarioEvaluation is per-scenario weighted ranking.
type ScenarioEvaluation struct {
	Scenario   string           `json:"scenario"`
	Version    string           `json:"version,omitempty"`
	ProfileID  string           `json:"profileId"`
	RunCount   int64            `json:"runCount"`
	RankScore  float64          `json:"rankScore"`
	Status     string           `json:"status"`
	Dimensions []DimensionScore `json:"dimensions"`
	GateHints  []string         `json:"gateHints,omitempty"`
}

// DataQualityNote explains proxy/partial scoring.
type DataQualityNote struct {
	MetricID string `json:"metricId"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// Service aggregates evaluation from runs / events / threads.
type Service struct {
	db       *store.DB
	ctx      context.Context
	profiles []ScenarioEvalProfile
}

func NewService(db *store.DB) *Service {
	path := ResolveProfilesPath()
	profiles, _ := LoadProfiles(path)
	if len(profiles) == 0 {
		profiles = DefaultProfiles()
	}
	return &Service{db: db, profiles: profiles}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	out := *s
	out.ctx = ctx
	return &out
}

func (s *Service) WithProfiles(profiles []ScenarioEvalProfile) *Service {
	if s == nil {
		return nil
	}
	out := *s
	if len(profiles) > 0 {
		out.profiles = profiles
	}
	return &out
}

func (s *Service) gdb() *gorm.DB {
	if s == nil || s.db == nil {
		return nil
	}
	if s.ctx != nil {
		return s.db.WithContext(s.ctx)
	}
	return s.db.DB
}

// Evaluate builds space evaluation for the window.
func (s *Service) Evaluate(req EvaluationRequest) (Evaluation, error) {
	req = normalizeEvalRequest(req)
	out := Evaluation{
		SpaceID: req.SpaceID,
		From:    req.From.Format(time.RFC3339), To: req.To.Format(time.RFC3339),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Health:      EvaluationHealth{MemoryLinksByType: map[string]int64{}},
	}
	gdb := s.gdb()
	if gdb == nil {
		return out, nil
	}

	var runs []store.RunRecord
	if err := gdb.Where("space_id = ? AND started_at >= ? AND started_at <= ?", req.SpaceID, req.From, req.To).
		Find(&runs).Error; err != nil {
		return out, err
	}
	runIDs := make([]string, 0, len(runs))
	for _, r := range runs {
		runIDs = append(runIDs, r.ID)
	}

	agg, err := s.aggregateSignals(gdb, req.SpaceID, runIDs, runs)
	if err != nil {
		return out, err
	}
	scoreAgg, err := s.aggregateScoreEvents(gdb, req)
	if err != nil {
		return out, err
	}
	out.Dimensions = buildDimensions(agg)
	applyScoreEventBlend(&out.Dimensions, scoreAgg)
	out.Health = agg.health
	out.Scenarios = s.buildScenarios(gdb, runs)
	out.DataQuality = scoreEventDataQuality(scoreAgg)
	if len(runs) == 0 {
		out.DataQuality = append(out.DataQuality, DataQualityNote{
			MetricID: "runs", Status: "empty", Message: "no runs in evaluation window",
		})
	}
	return out, nil
}

// scoreEventAgg summarizes non-voided rubric scores in the evaluation window.
type scoreEventAgg struct {
	total, active, voided int64
	avgComposite          float64 // 1–5 mean of non-voided composites
}

func (s *Service) aggregateScoreEvents(gdb *gorm.DB, req EvaluationRequest) (scoreEventAgg, error) {
	var out scoreEventAgg
	if gdb == nil {
		return out, nil
	}
	voided, err := VoidedIDs(gdb, req.SpaceID)
	if err != nil {
		return out, err
	}
	var rows []store.ScoreEvent
	if err := gdb.Where("space_id = ? AND created_at >= ? AND created_at <= ?",
		req.SpaceID, req.From, req.To).Find(&rows).Error; err != nil {
		return out, err
	}
	var sum float64
	for _, row := range rows {
		out.total++
		if _, ok := voided[row.ID]; ok {
			out.voided++
			continue
		}
		out.active++
		sum += row.Composite
	}
	if out.active > 0 {
		out.avgComposite = sum / float64(out.active)
	}
	return out, nil
}

func applyScoreEventBlend(dims *[]DimensionScore, agg scoreEventAgg) {
	if dims == nil || agg.active <= 0 {
		return
	}
	rubric01 := clamp01((agg.avgComposite - 1) / 4)
	sig := SignalMetric{
		ID: "rubric_composite_avg", Label: "量纲合成分均值（排除作废）",
		Value: round4(agg.avgComposite), Unit: "score",
		Numerator: agg.active, Denominator: agg.total,
	}
	for i := range *dims {
		if (*dims)[i].ID != "quality" {
			continue
		}
		blended := clamp01(0.5*(*dims)[i].Score + 0.5*rubric01)
		(*dims)[i].Score = round4(blended)
		(*dims)[i].Signals = append((*dims)[i].Signals, sig)
		if (*dims)[i].Status == "empty" {
			(*dims)[i].Status = "ok"
		}
		return
	}
}

func scoreEventDataQuality(agg scoreEventAgg) []DataQualityNote {
	switch {
	case agg.active > 0:
		return []DataQualityNote{{
			MetricID: "rubric",
			Status:   "ok",
			Message: fmt.Sprintf(
				"score_events active=%d voided_excluded=%d avg_composite=%.2f (blended into quality)",
				agg.active, agg.voided, agg.avgComposite,
			),
		}}
	case agg.voided > 0:
		return []DataQualityNote{{
			MetricID: "rubric",
			Status:   "empty",
			Message: fmt.Sprintf(
				"all score_events voided (%d excluded); composite from run/event/thread proxies",
				agg.voided,
			),
		}}
	default:
		return []DataQualityNote{{
			MetricID: "rubric",
			Status:   "partial",
			Message:  "score_events absent; composite from run/event/thread proxies (GV06 MVP)",
		}}
	}
}

type signalAgg struct {
	started, success, completed int64
	totalDurationMs             int64
	lowFeedback, totalFeedback  int64
	policyDenied                int64
	citationMissing             int64
	memoryHitUsed               int64
	memoryQueries               int64
	threadsTotal, threadsSealed int64
	replayMismatch, sealedObs   int64
	memoryLinksByType           map[string]int64
	health                      EvaluationHealth
}

func (s *Service) aggregateSignals(gdb *gorm.DB, spaceID string, runIDs []string, runs []store.RunRecord) (signalAgg, error) {
	out := signalAgg{memoryLinksByType: map[string]int64{}}
	for _, r := range runs {
		out.started++
		if r.Status == "finished" {
			out.success++
		}
		if r.FinishedAt != nil {
			out.completed++
			out.totalDurationMs += r.FinishedAt.Sub(r.StartedAt).Milliseconds()
		}
	}

	var feedback []store.Feedback
	if err := gdb.Where("space_id = ?", spaceID).Find(&feedback).Error; err != nil {
		return out, err
	}
	for _, f := range feedback {
		if !inRunSet(runIDs, f.RunID) && f.RunID != "" {
			continue
		}
		out.totalFeedback++
		if f.Rating > 0 && f.Rating <= 2 {
			out.lowFeedback++
		}
	}

	if len(runIDs) > 0 {
		var evs []store.RunEvent
		if err := gdb.Where("run_id IN ? AND type IN ?", runIDs, []string{
			"policy.denied", "citation.missing", "memory.hit_used", "memory.query",
			"memory.injected", "memory.candidate", "knowledge.injected", "skills.injected",
			"interaction.thread_sealed", "interaction.replay_mismatch",
		}).Find(&evs).Error; err != nil {
			return out, err
		}
		for _, ev := range evs {
			switch ev.Type {
			case "policy.denied":
				out.policyDenied++
			case "citation.missing":
				out.citationMissing++
			case "memory.hit_used":
				out.memoryHitUsed += memoryLinkCount(ev.Type, ev.PayloadJSON)
				out.memoryLinksByType["hit_used"] += memoryLinkCount(ev.Type, ev.PayloadJSON)
			case "memory.query":
				out.memoryQueries++
			case "memory.injected", "knowledge.injected", "skills.injected":
				n := memoryLinkCount(ev.Type, ev.PayloadJSON)
				out.memoryLinksByType["context_ref"] += n
			case "memory.candidate":
				n := memoryLinkCount(ev.Type, ev.PayloadJSON)
				out.memoryLinksByType["candidate_out"] += n
			case "interaction.thread_sealed":
				out.sealedObs++
			case "interaction.replay_mismatch":
				out.replayMismatch++
			}
		}
	}

	var threads []store.InteractionThread
	q := gdb.Where("space_id = ?", spaceID)
	if len(runIDs) > 0 {
		q = q.Where("run_id IN ?", runIDs)
	} else {
		q = q.Where("1 = 0")
	}
	if err := q.Find(&threads).Error; err != nil {
		return out, err
	}
	for _, th := range threads {
		out.threadsTotal++
		if th.Status == "sealed" {
			out.threadsSealed++
		}
	}

	out.health = EvaluationHealth{
		ThreadSealRate:       ratioSignal("thread_seal_rate", "Thread 封印率", out.threadsSealed, out.threadsTotal),
		ReplayMismatchRate:   ratioSignal("replay_mismatch_rate", "Replay mismatch 率", out.replayMismatch, max64(out.sealedObs, out.threadsSealed)),
		CitationMissingTotal: out.citationMissing,
		MemoryLinksByType:    out.memoryLinksByType,
	}
	return out, nil
}

func (s *Service) buildScenarios(gdb *gorm.DB, runs []store.RunRecord) []ScenarioEvaluation {
	_ = gdb
	byKey := map[string][]store.RunRecord{}
	for _, r := range runs {
		key := r.ScenarioName + "@" + r.ScenarioVersion
		byKey[key] = append(byKey[key], r)
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]ScenarioEvaluation, 0, len(keys))
	for _, key := range keys {
		group := byKey[key]
		name := group[0].ScenarioName
		version := group[0].ScenarioVersion
		profile := profileForScenario(s.profiles, name)
		agg := signalAgg{memoryLinksByType: map[string]int64{}}
		for _, r := range group {
			agg.started++
			if r.Status == "finished" {
				agg.success++
			}
			if r.FinishedAt != nil {
				agg.completed++
				agg.totalDurationMs += r.FinishedAt.Sub(r.StartedAt).Milliseconds()
			}
		}
		// Per-scenario light event counts for the group's run IDs.
		ids := make([]string, 0, len(group))
		for _, r := range group {
			ids = append(ids, r.ID)
		}
		if len(ids) > 0 && s.gdb() != nil {
			var evs []store.RunEvent
			_ = s.gdb().Where("run_id IN ? AND type IN ?", ids, []string{
				"policy.denied", "citation.missing", "memory.hit_used", "memory.query",
				"interaction.thread_sealed", "interaction.replay_mismatch",
			}).Find(&evs)
			for _, ev := range evs {
				switch ev.Type {
				case "policy.denied":
					agg.policyDenied++
				case "citation.missing":
					agg.citationMissing++
				case "memory.hit_used":
					agg.memoryHitUsed += memoryLinkCount(ev.Type, ev.PayloadJSON)
				case "memory.query":
					agg.memoryQueries++
				case "interaction.thread_sealed":
					agg.sealedObs++
				case "interaction.replay_mismatch":
					agg.replayMismatch++
				}
			}
			var threads []store.InteractionThread
			_ = s.gdb().Where("run_id IN ?", ids).Find(&threads)
			for _, th := range threads {
				agg.threadsTotal++
				if th.Status == "sealed" {
					agg.threadsSealed++
				}
			}
		}
		dims := buildDimensions(agg)
		rank := weightedRank(dims, profile.Weights)
		status := "ok"
		if agg.started == 0 {
			status = "empty"
		}
		out = append(out, ScenarioEvaluation{
			Scenario: name, Version: version, ProfileID: profile.ID,
			RunCount: agg.started, RankScore: round4(rank), Status: status,
			Dimensions: dims, GateHints: profile.GateHints,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RankScore == out[j].RankScore {
			return out[i].Scenario < out[j].Scenario
		}
		return out[i].RankScore > out[j].RankScore
	})
	return out
}

func buildDimensions(agg signalAgg) []DimensionScore {
	success := ratioSignal("run_success_rate", "任务成功率", agg.success, agg.started)
	lowFeedback := ratioSignal("low_feedback_rate", "低分反馈率", agg.lowFeedback, agg.totalFeedback)
	qualityScore := success.Value
	if lowFeedback.Denominator > 0 {
		qualityScore = clamp01(0.7*success.Value + 0.3*(1-lowFeedback.Value))
	}

	policyRate := ratioSignal("policy_denied_rate", "策略拒绝率", agg.policyDenied, max64(agg.started, 1))
	citationRate := ratioSignal("citation_missing_rate", "缺引用率", agg.citationMissing, max64(agg.started, 1))
	safetyScore := clamp01(1 - 0.6*policyRate.Value - 0.4*citationRate.Value)

	hitRate := ratioSignal("memory_hit_rate", "Memory 命中使用率", agg.memoryHitUsed, max64(agg.memoryQueries, agg.memoryHitUsed))
	effScore := hitRate.Value
	if hitRate.Denominator == 0 && agg.completed > 0 {
		avgMs := float64(agg.totalDurationMs) / float64(agg.completed)
		// Soft efficiency: under 10 minutes → 1.0, over 60 minutes → 0.
		effScore = clamp01(1 - (avgMs-10*60*1000)/(50*60*1000))
		hitRate = SignalMetric{ID: "avg_duration_efficiency", Label: "时长效率代理", Value: round4(effScore), Unit: "ratio"}
	}

	seal := ratioSignal("thread_seal_rate", "Thread 封印率", agg.threadsSealed, agg.threadsTotal)
	govScore := seal.Value
	if seal.Denominator == 0 {
		govScore = 0
	}

	return []DimensionScore{
		dim("quality", "Quality", qualityScore, []SignalMetric{success, invertSignal(lowFeedback, "low_feedback_health", "低分反馈健康度")}),
		dim("safety", "Safety", safetyScore, []SignalMetric{invertSignal(policyRate, "policy_health", "策略健康度"), invertSignal(citationRate, "citation_health", "引用健康度")}),
		dim("efficiency", "Efficiency", effScore, []SignalMetric{hitRate}),
		dim("governance", "Governance", govScore, []SignalMetric{seal}),
	}
}

func dim(id, label string, score float64, signals []SignalMetric) DimensionScore {
	status := "ok"
	if len(signals) == 0 || (signals[0].Denominator == 0 && score == 0) {
		status = "empty"
	}
	return DimensionScore{ID: id, Label: label, Score: round4(clamp01(score)), Status: status, Signals: signals}
}

func ratioSignal(id, label string, num, den int64) SignalMetric {
	m := SignalMetric{ID: id, Label: label, Unit: "ratio", Numerator: num, Denominator: den}
	if den <= 0 {
		m.Value = 0
		return m
	}
	m.Value = round4(float64(num) / float64(den))
	return m
}

func invertSignal(m SignalMetric, id, label string) SignalMetric {
	out := m
	out.ID = id
	out.Label = label
	if m.Denominator <= 0 {
		out.Value = 0
		return out
	}
	out.Value = round4(1 - m.Value)
	return out
}

func memoryLinkCount(eventType, payloadJSON string) int64 {
	payload := map[string]any{}
	_ = json.Unmarshal([]byte(payloadJSON), &payload)
	switch eventType {
	case "memory.hit_used", "memory.injected":
		return int64(len(anyStringSlice(payload["recordIds"])))
	case "memory.candidate":
		n := int64(0)
		if strField(payload, "candidateId") != "" {
			n++
		}
		if strField(payload, "recordId") != "" {
			n++
		}
		return n
	case "knowledge.injected", "skills.injected":
		n := int64(0)
		for _, ref := range anyStringSlice(payload["refs"]) {
			if strings.HasPrefix(strings.TrimSpace(ref), "memory:") {
				n++
			}
		}
		return n
	default:
		return 0
	}
}

func anyStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func strField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func inRunSet(ids []string, runID string) bool {
	if runID == "" {
		return true
	}
	for _, id := range ids {
		if id == runID {
			return true
		}
	}
	return false
}

func normalizeEvalRequest(req EvaluationRequest) EvaluationRequest {
	req.SpaceID = firstNonEmpty(strings.TrimSpace(req.SpaceID), "local")
	if req.To.IsZero() {
		req.To = time.Now().UTC()
	}
	if req.From.IsZero() {
		req.From = req.To.Add(-7 * 24 * time.Hour)
	}
	return req
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
