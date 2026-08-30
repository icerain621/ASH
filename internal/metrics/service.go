package metrics

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

// Scenario stability (R-02): among scenarios with enough repeats, fraction that
// meet the success-rate floor. Thresholds are product defaults for console KPI-11.
const (
	scenarioStabilityMinSamples = 2
	scenarioStabilityRateFloor  = 0.85
)

type OverviewRequest struct {
	SpaceID   string
	ProjectID string
	From      time.Time
	To        time.Time
	Period    string
}

type Overview struct {
	SpaceID     string            `json:"spaceId"`
	ProjectID   string            `json:"projectId,omitempty"`
	From        string            `json:"from"`
	To          string            `json:"to"`
	Period      string            `json:"period"`
	Summary     []MetricCard      `json:"summary"`
	Trends      []MetricTrend     `json:"trends"`
	Breakdowns  []MetricBreakdown `json:"breakdowns"`
	DataQuality []DataQualityNote `json:"dataQuality"`
	GeneratedAt string            `json:"generatedAt"`
}

type MetricCard struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Status      string  `json:"status"`
	Numerator   int64   `json:"numerator,omitempty"`
	Denominator int64   `json:"denominator,omitempty"`
	Description string  `json:"description,omitempty"`
}

type MetricTrend struct {
	MetricID string        `json:"metricId"`
	Points   []MetricPoint `json:"points"`
}

type MetricPoint struct {
	PeriodStart string  `json:"periodStart"`
	Value       float64 `json:"value"`
	Status      string  `json:"status"`
}

type MetricBreakdown struct {
	ID    string          `json:"id"`
	Label string          `json:"label"`
	Items []BreakdownItem `json:"items"`
}

type BreakdownItem struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type DataQualityNote struct {
	MetricID string `json:"metricId"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type Service struct {
	db  *store.DB
	ctx context.Context
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

// WithContext returns a shallow copy bound to ctx for Postgres RLS session vars.
func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	return &Service{db: s.db, ctx: ctx}
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

func (s *Service) Overview(req OverviewRequest) (Overview, error) {
	return s.overview(req)
}

func (s *Service) OverviewContext(ctx context.Context, req OverviewRequest) (Overview, error) {
	return s.WithContext(ctx).overview(req)
}

func (s *Service) overview(req OverviewRequest) (Overview, error) {
	gdb := s.gdb()
	req = normalizeRequest(req)
	out := Overview{
		SpaceID: req.SpaceID, ProjectID: req.ProjectID,
		From: req.From.Format(time.RFC3339), To: req.To.Format(time.RFC3339),
		Period: req.Period, GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	runStats, err := s.runStats(gdb, req)
	if err != nil {
		return out, err
	}
	feedbackStats, err := s.feedbackStats(gdb, req)
	if err != nil {
		return out, err
	}
	memoryStats, err := s.memoryStats(gdb, req)
	if err != nil {
		return out, err
	}
	ciStats, err := s.ciStats(gdb, req)
	if err != nil {
		return out, err
	}
	apiStats, err := s.apiStats(gdb, req)
	if err != nil {
		return out, err
	}
	queueStats, err := s.queueStats(gdb, req)
	if err != nil {
		return out, err
	}
	sseStats, err := s.sseStats(gdb, req)
	if err != nil {
		return out, err
	}
	scenarioCard, scenarioBreakdown, err := s.scenarioStability(gdb, req)
	if err != nil {
		return out, err
	}
	sandboxStats, err := s.sandboxCoverageStats(gdb, req)
	if err != nil {
		return out, err
	}
	evolveBacklog, err := s.evolveBacklogStats(gdb, req)
	if err != nil {
		return out, err
	}
	improveRollback, err := s.improveRollbackStats(gdb, req)
	if err != nil {
		return out, err
	}

	out.Summary = []MetricCard{
		ratioCard("KPI-01", "任务成功率", runStats.success, runStats.started, "ratio"),
		durationCard("KPI-02", "平均任务时长", runStats.totalDurationMs, runStats.completed),
		ratioCard("KPI-03", "AI 建议采纳率", feedbackStats.accepted, feedbackStats.totalSuggestions, "ratio"),
		ratioCard("KPI-04", "CI 一次通过率", ciStats.firstPass, ciStats.firstAttempt, "ratio"),
		ratioCard("KPI-05", "CI 诊断采纳率", ciStats.adoptedDiagnosis, ciStats.totalDiagnosis, "ratio"),
		ratioCard("KPI-06", "低分反馈率", feedbackStats.lowScore, feedbackStats.totalFeedback, "ratio"),
		ratioCard("KPI-07", "Memory 命中率", memoryStats.used, memoryStats.queries, "ratio"),
		sseStabilityCard(sseStats),
		ratioCard("KPI-09", "API 错误率", apiStats.errors, apiStats.total, "ratio"),
		durationCard("KPI-10", "队列积压时长", queueStats.totalWaitMs, queueStats.count),
		scenarioCard,
		evolveBacklogCard(evolveBacklog),
		improveRollbackCard(improveRollback),
		sandboxCoverageCard(sandboxStats),
	}
	out.Trends = []MetricTrend{
		{MetricID: "KPI-01", Points: s.runSuccessTrend(gdb, req)},
		{MetricID: "KPI-04", Points: s.ciFirstPassTrend(gdb, req)},
		{MetricID: "KPI-06", Points: s.lowFeedbackTrend(gdb, req)},
	}
	out.Breakdowns = []MetricBreakdown{
		scenarioBreakdown,
		{ID: "ciDiagnosis", Label: "CI 诊断根因", Items: s.ciDiagnosisBreakdown(gdb, req)},
		{ID: "feedbackRatings", Label: "反馈评分分布", Items: s.feedbackRatingBreakdown(gdb, req)},
		{ID: "memoryEvents", Label: "Memory 事件", Items: []BreakdownItem{
			{Key: "queries", Label: "查询", Value: float64(memoryStats.queries), Unit: "count"},
			{Key: "used", Label: "命中使用", Value: float64(memoryStats.used), Unit: "count"},
		}},
	}
	out.DataQuality = dataQuality(out.Summary)
	if req.ProjectID != "" {
		out.DataQuality = append(out.DataQuality, DataQualityNote{
			MetricID: "projectId", Status: "partial",
			Message: "projectId v1 仅用于 CI repo connection 过滤；通用 run/feedback 仍按 space 聚合。",
		})
	}
	return out, nil
}

type runStatsResult struct {
	started         int64
	success         int64
	completed       int64
	totalDurationMs int64
}

func (s *Service) runStats(gdb *gorm.DB, req OverviewRequest) (runStatsResult, error) {
	var rows []store.RunRecord
	err := gdb.Where("space_id = ? AND started_at >= ? AND started_at <= ?", req.SpaceID, req.From, req.To).Find(&rows).Error
	if err != nil {
		return runStatsResult{}, err
	}
	var out runStatsResult
	for _, row := range rows {
		out.started++
		if row.Status == "finished" {
			out.success++
		}
		if row.FinishedAt != nil {
			out.completed++
			out.totalDurationMs += row.FinishedAt.Sub(row.StartedAt).Milliseconds()
		}
	}
	return out, nil
}

type feedbackStatsResult struct {
	totalFeedback    int64
	lowScore         int64
	totalSuggestions int64
	accepted         int64
}

func (s *Service) feedbackStats(gdb *gorm.DB, req OverviewRequest) (feedbackStatsResult, error) {
	var rows []store.Feedback
	err := gdb.Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To).Find(&rows).Error
	if err != nil {
		return feedbackStatsResult{}, err
	}
	var out feedbackStatsResult
	for _, row := range rows {
		out.totalFeedback++
		if row.Rating > 0 && row.Rating <= 2 {
			out.lowScore++
		}
		if strings.Contains(strings.ToLower(row.TargetType), "suggestion") || strings.Contains(strings.ToLower(row.Comment), "suggestion") {
			out.totalSuggestions++
			if row.Rating >= 4 || strings.Contains(strings.ToLower(row.Comment), "accepted") || strings.Contains(row.Comment, "采纳") {
				out.accepted++
			}
		}
	}
	return out, nil
}

type memoryStatsResult struct {
	queries int64
	used    int64
}

func runIDsInSpace(gdb *gorm.DB, spaceID string) *gorm.DB {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		return nil
	}
	return gdb.Model(&store.RunRecord{}).Select("id").Where("space_id = ?", spaceID)
}

func (s *Service) runEventsQuery(gdb *gorm.DB, req OverviewRequest) *gorm.DB {
	q := gdb.Model(&store.RunEvent{}).Where("created_at >= ? AND created_at <= ?", req.From, req.To)
	if sub := runIDsInSpace(gdb, req.SpaceID); sub != nil {
		q = q.Where("run_id IN (?)", sub)
	}
	return q
}

func (s *Service) runStepsQuery(gdb *gorm.DB, req OverviewRequest) *gorm.DB {
	q := gdb.Model(&store.RunStep{}).Where("created_at >= ? AND created_at <= ?", req.From, req.To)
	if sub := runIDsInSpace(gdb, req.SpaceID); sub != nil {
		q = q.Where("run_id IN (?)", sub)
	}
	return q
}

func (s *Service) memoryStats(gdb *gorm.DB, req OverviewRequest) (memoryStatsResult, error) {
	var out memoryStatsResult
	if err := s.runEventsQuery(gdb, req).
		Where("type IN ?", []string{"memory.injected", "memory.query_failed"}).
		Count(&out.queries).Error; err != nil {
		return out, err
	}
	if err := gdb.Model(&store.AuditLog{}).
		Where("space_id = ? AND created_at >= ? AND created_at <= ? AND event_type = ?", req.SpaceID, req.From, req.To, "memory.hit_used").
		Count(&out.used).Error; err != nil {
		return out, err
	}
	if out.queries == 0 {
		if err := s.runEventsQuery(gdb, req).
			Where("type = ?", "memory.hit_used").
			Count(&out.used).Error; err != nil {
			return out, err
		}
		out.queries = out.used
	}
	return out, nil
}

type ciStatsResult struct {
	firstAttempt     int64
	firstPass        int64
	totalDiagnosis   int64
	adoptedDiagnosis int64
}

func (s *Service) ciStats(gdb *gorm.DB, req OverviewRequest) (ciStatsResult, error) {
	var runs []store.CIRun
	q := gdb.Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To)
	if req.ProjectID != "" {
		q = q.Where("connection_id = ?", req.ProjectID)
	}
	if err := q.Find(&runs).Error; err != nil {
		return ciStatsResult{}, err
	}
	var out ciStatsResult
	for _, row := range runs {
		if row.Attempt == 0 || row.Attempt == 1 {
			out.firstAttempt++
			if normalizeConclusion(row.Conclusion, row.Status) == "success" {
				out.firstPass++
			}
		}
	}
	dq := gdb.Model(&store.CIDiagnosis{}).Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To)
	if req.ProjectID != "" {
		dq = dq.Where("connection_id = ?", req.ProjectID)
	}
	if err := dq.Count(&out.totalDiagnosis).Error; err != nil {
		return out, err
	}
	if err := dq.Where("adopted = ?", true).Count(&out.adoptedDiagnosis).Error; err != nil {
		return out, err
	}
	return out, nil
}

type apiStatsResult struct {
	total  int64
	errors int64
}

func (s *Service) apiStats(gdb *gorm.DB, req OverviewRequest) (apiStatsResult, error) {
	var total, errors int64
	if err := gdb.Model(&store.AuditLog{}).
		Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To).
		Count(&total).Error; err != nil {
		return apiStatsResult{}, err
	}
	if err := gdb.Model(&store.AuditLog{}).
		Where("space_id = ? AND created_at >= ? AND created_at <= ? AND event_type LIKE ?", req.SpaceID, req.From, req.To, "%.failed%").
		Count(&errors).Error; err != nil {
		return apiStatsResult{}, err
	}
	return apiStatsResult{total: total, errors: errors}, nil
}

type queueStatsResult struct {
	totalWaitMs int64
	count       int64
}

type sseStatsResult struct {
	opened int64
	closed int64
	failed int64
}

func (s *Service) sseStats(gdb *gorm.DB, req OverviewRequest) (sseStatsResult, error) {
	var out sseStatsResult
	count := func(eventType string, dest *int64) error {
		return gdb.Model(&store.AuditLog{}).
			Where("space_id = ? AND created_at >= ? AND created_at <= ? AND event_type = ?", req.SpaceID, req.From, req.To, eventType).
			Count(dest).Error
	}
	if err := count("stream.session_opened", &out.opened); err != nil {
		return out, err
	}
	if err := count("stream.session_closed", &out.closed); err != nil {
		return out, err
	}
	if err := count("stream.session_failed", &out.failed); err != nil {
		return out, err
	}
	return out, nil
}

func sseStabilityCard(stats sseStatsResult) MetricCard {
	// KPI-08: successful_sse_sessions / total_sse_sessions among terminal outcomes.
	// successful = stream.session_closed; failed = stream.session_failed; in-flight opened ignored.
	terminal := stats.closed + stats.failed
	if terminal == 0 && stats.opened == 0 {
		return unavailableCard("KPI-08", "SSE 稳定率", "当前时间窗口尚无 SSE 会话审计事件；连接 /runs/{id}/stream 后将自动采集。")
	}
	if terminal == 0 {
		return MetricCard{
			ID: "KPI-08", Label: "SSE 稳定率", Unit: "ratio", Status: "empty",
			Numerator: 0, Denominator: 0,
			Description: fmt.Sprintf(
				"有 %d 个已打开会话尚无终态（closed/failed）；待客户端断开或失败后再计入 KPI-08。",
				stats.opened,
			),
		}
	}
	card := ratioCard("KPI-08", "SSE 稳定率", stats.closed, terminal, "ratio")
	card.Description = fmt.Sprintf(
		"成功关闭 / (成功关闭 + 失败)；opened=%d closed=%d failed=%d。",
		stats.opened, stats.closed, stats.failed,
	)
	return card
}

type sandboxCoverageResult struct {
	dangerTotal    int64
	dangerIsolated int64
}

func (s *Service) sandboxCoverageStats(gdb *gorm.DB, req OverviewRequest) (sandboxCoverageResult, error) {
	var out sandboxCoverageResult
	var rows []store.RunEvent
	if err := s.runEventsQuery(gdb, req).
		Where("type = ?", "harness.tool.routed").
		Find(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		var payload map[string]any
		if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
			continue
		}
		risk, _ := payload["risk"].(string)
		if strings.ToLower(strings.TrimSpace(risk)) != "danger" {
			continue
		}
		out.dangerTotal++
		mode, _ := payload["sandboxMode"].(string)
		if strings.EqualFold(strings.TrimSpace(mode), "isolated") {
			out.dangerIsolated++
		}
	}
	return out, nil
}

func sandboxCoverageCard(stats sandboxCoverageResult) MetricCard {
	// KPI-19: danger tool routes with sandboxMode=isolated / all danger routes.
	if stats.dangerTotal == 0 {
		return MetricCard{
			ID: "KPI-19", Label: "danger 沙盒覆盖率", Unit: "ratio", Status: "empty",
			Description: "当前窗口尚无 risk=danger 的 harness.tool.routed 事件。",
		}
	}
	card := ratioCard("KPI-19", "danger 沙盒覆盖率", stats.dangerIsolated, stats.dangerTotal, "ratio")
	if card.Value < 1 {
		card.Description = fmt.Sprintf(
			"目标 100%%；isolated=%d / danger=%d。hotfix/security 与 danger 工具须 ≥ isolated（DX2）。",
			stats.dangerIsolated, stats.dangerTotal,
		)
	} else {
		card.Description = "danger 工具路由均达到 isolated（KPI-19）。"
	}
	return card
}

const evolveBacklogTarget = int64(50)

type evolveBacklogResult struct {
	harnessInReview int64
	patchInReview   int64
	staleOver7d     int64
}

func (s *Service) evolveBacklogStats(gdb *gorm.DB, req OverviewRequest) (evolveBacklogResult, error) {
	var out evolveBacklogResult
	space := req.SpaceID
	if err := gdb.Model(&store.HarnessProfileVersion{}).
		Where("space_id = ? AND status = ?", space, "in_review").
		Count(&out.harnessInReview).Error; err != nil {
		return out, err
	}
	if err := gdb.Model(&store.ScenarioPatchDraft{}).
		Where("space_id = ? AND status = ?", space, "in_review").
		Count(&out.patchInReview).Error; err != nil {
		return out, err
	}
	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)
	var staleH, staleP int64
	if err := gdb.Model(&store.HarnessProfileVersion{}).
		Where("space_id = ? AND status = ? AND created_at < ?", space, "in_review", cutoff).
		Count(&staleH).Error; err != nil {
		return out, err
	}
	if err := gdb.Model(&store.ScenarioPatchDraft{}).
		Where("space_id = ? AND status = ? AND created_at < ?", space, "in_review", cutoff).
		Count(&staleP).Error; err != nil {
		return out, err
	}
	out.staleOver7d = staleH + staleP
	return out, nil
}

func evolveBacklogCard(stats evolveBacklogResult) MetricCard {
	// KPI-17: orchestration review backlog (harness + scenario_patch in_review).
	backlog := stats.harnessInReview + stats.patchInReview
	status := "ok"
	if backlog >= evolveBacklogTarget {
		status = "warn"
	}
	desc := fmt.Sprintf(
		"目标 < %d/space；harness=%d patch=%d；>7天未清=%d。",
		evolveBacklogTarget, stats.harnessInReview, stats.patchInReview, stats.staleOver7d,
	)
	if backlog == 0 {
		return MetricCard{
			ID: "KPI-17", Label: "编排评审积压", Unit: "count", Status: "ok",
			Numerator: 0, Denominator: evolveBacklogTarget, Description: desc + " 队列为空。",
		}
	}
	return MetricCard{
		ID: "KPI-17", Label: "编排评审积压", Value: float64(backlog), Unit: "count", Status: status,
		Numerator: backlog, Denominator: evolveBacklogTarget, Description: desc,
	}
}

type improveRollbackResult struct {
	promoted   int64
	rolledBack int64
}

func (s *Service) improveRollbackStats(gdb *gorm.DB, req OverviewRequest) (improveRollbackResult, error) {
	var out improveRollbackResult
	// Window filter on updated_at so recent promotes/rollbacks are visible.
	if err := gdb.Model(&store.ImproveProposal{}).
		Where("space_id = ? AND updated_at >= ? AND updated_at <= ? AND status = ?",
			req.SpaceID, req.From, req.To, "promoted").
		Count(&out.promoted).Error; err != nil {
		return out, err
	}
	if err := gdb.Model(&store.ImproveProposal{}).
		Where("space_id = ? AND updated_at >= ? AND updated_at <= ? AND status = ?",
			req.SpaceID, req.From, req.To, "rolled_back").
		Count(&out.rolledBack).Error; err != nil {
		return out, err
	}
	return out, nil
}

func improveRollbackCard(stats improveRollbackResult) MetricCard {
	// KPI-18: rolled_back / (promoted + rolled_back); target < 2%.
	denom := stats.promoted + stats.rolledBack
	if denom == 0 {
		return MetricCard{
			ID: "KPI-18", Label: "Improve 误升格回滚率", Unit: "ratio", Status: "empty",
			Description: "当前窗口尚无 promoted/rolled_back 的 Improve 提案。",
		}
	}
	card := ratioCard("KPI-18", "Improve 误升格回滚率", stats.rolledBack, denom, "ratio")
	if card.Value >= 0.02 {
		card.Status = "warn"
		card.Description = fmt.Sprintf(
			"目标 < 2%%；rolled_back=%d / (promoted+rolled_back)=%d。",
			stats.rolledBack, denom,
		)
	} else {
		card.Description = fmt.Sprintf(
			"目标 < 2%%；rolled_back=%d / (promoted+rolled_back)=%d。",
			stats.rolledBack, denom,
		)
	}
	return card
}

type scenarioBucket struct {
	started  int64
	finished int64
	lowScore int64
}

func scenarioKey(name, version string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" {
		name = "unknown"
	}
	if version == "" {
		return name
	}
	return name + "@" + version
}

// scenarioStability aggregates per-scenario repeat success (R-02) for KPI-11 + breakdown.
func (s *Service) scenarioStability(gdb *gorm.DB, req OverviewRequest) (MetricCard, MetricBreakdown, error) {
	breakdown := MetricBreakdown{ID: "scenarioStability", Label: "场景可重复性 (R-02)"}
	var runs []store.RunRecord
	if err := gdb.Where("space_id = ? AND started_at >= ? AND started_at <= ?", req.SpaceID, req.From, req.To).
		Find(&runs).Error; err != nil {
		return MetricCard{}, breakdown, err
	}
	buckets := map[string]*scenarioBucket{}
	runScenario := map[string]string{}
	for _, row := range runs {
		key := scenarioKey(row.ScenarioName, row.ScenarioVersion)
		b := buckets[key]
		if b == nil {
			b = &scenarioBucket{}
			buckets[key] = b
		}
		b.started++
		if row.Status == "finished" {
			b.finished++
		}
		runScenario[row.ID] = key
	}

	var feedback []store.Feedback
	if err := gdb.Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To).
		Find(&feedback).Error; err != nil {
		return MetricCard{}, breakdown, err
	}
	for _, row := range feedback {
		if row.Rating <= 0 || row.Rating > 2 {
			continue
		}
		target := strings.ToLower(strings.TrimSpace(row.TargetType))
		if target != "run" && target != "runs" {
			continue
		}
		if key, ok := runScenario[row.TargetID]; ok {
			if b := buckets[key]; b != nil {
				b.lowScore++
			}
		}
	}

	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var stable, eligible int64
	items := make([]BreakdownItem, 0, len(keys)*3)
	for _, key := range keys {
		b := buckets[key]
		rate := ratio(b.finished, b.started)
		items = append(items,
			BreakdownItem{Key: key, Label: key + " 成功率", Value: rate, Unit: "ratio"},
			BreakdownItem{Key: key + ":n", Label: key + " 样本", Value: float64(b.started), Unit: "count"},
			BreakdownItem{Key: key + ":low", Label: key + " 低分反馈", Value: float64(b.lowScore), Unit: "count"},
		)
		if b.started >= scenarioStabilityMinSamples {
			eligible++
			if rate >= scenarioStabilityRateFloor {
				stable++
			}
		}
	}
	breakdown.Items = items

	card := ratioCard("KPI-11", "场景稳定率", stable, eligible, "ratio")
	card.Description = fmt.Sprintf(
		"R-02：同场景样本≥%d 且成功率≥%.0f%% 的场景占比；分解见 scenarioStability。",
		scenarioStabilityMinSamples, scenarioStabilityRateFloor*100,
	)
	if eligible <= 0 {
		card = MetricCard{
			ID: "KPI-11", Label: "场景稳定率", Unit: "ratio", Status: "empty",
			Description: fmt.Sprintf(
				"当前窗口没有场景达到可重复样本门槛（≥%d 次运行）；单次运行不计入 KPI-11。",
				scenarioStabilityMinSamples,
			),
		}
	}
	return card, breakdown, nil
}

func (s *Service) queueStats(gdb *gorm.DB, req OverviewRequest) (queueStatsResult, error) {
	var rows []store.RunStep
	if err := s.runStepsQuery(gdb, req).Where("started_at IS NOT NULL").Find(&rows).Error; err != nil {
		return queueStatsResult{}, err
	}
	var out queueStatsResult
	for _, row := range rows {
		if row.StartedAt == nil {
			continue
		}
		wait := row.StartedAt.Sub(row.CreatedAt).Milliseconds()
		if wait > 0 {
			out.totalWaitMs += wait
			out.count++
		}
	}
	return out, nil
}

func (s *Service) runSuccessTrend(gdb *gorm.DB, req OverviewRequest) []MetricPoint {
	buckets := buckets(req)
	points := make([]MetricPoint, 0, len(buckets))
	for _, b := range buckets {
		next := advance(b, req.Period)
		var started, success int64
		_ = gdb.Model(&store.RunRecord{}).Where("space_id = ? AND started_at >= ? AND started_at < ?", req.SpaceID, b, next).Count(&started).Error
		_ = gdb.Model(&store.RunRecord{}).Where("space_id = ? AND started_at >= ? AND started_at < ? AND status = ?", req.SpaceID, b, next, "finished").Count(&success).Error
		points = append(points, point(b, ratio(success, started), started))
	}
	return points
}

func (s *Service) ciFirstPassTrend(gdb *gorm.DB, req OverviewRequest) []MetricPoint {
	buckets := buckets(req)
	points := make([]MetricPoint, 0, len(buckets))
	for _, b := range buckets {
		next := advance(b, req.Period)
		q := gdb.Model(&store.CIRun{}).Where("space_id = ? AND created_at >= ? AND created_at < ? AND (attempt = ? OR attempt = ?)", req.SpaceID, b, next, 0, 1)
		if req.ProjectID != "" {
			q = q.Where("connection_id = ?", req.ProjectID)
		}
		var rows []store.CIRun
		_ = q.Find(&rows).Error
		var pass int64
		for _, row := range rows {
			if normalizeConclusion(row.Conclusion, row.Status) == "success" {
				pass++
			}
		}
		points = append(points, point(b, ratio(pass, int64(len(rows))), int64(len(rows))))
	}
	return points
}

func (s *Service) lowFeedbackTrend(gdb *gorm.DB, req OverviewRequest) []MetricPoint {
	buckets := buckets(req)
	points := make([]MetricPoint, 0, len(buckets))
	for _, b := range buckets {
		next := advance(b, req.Period)
		var total, low int64
		_ = gdb.Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ? AND created_at < ?", req.SpaceID, b, next).Count(&total).Error
		_ = gdb.Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ? AND created_at < ? AND rating > 0 AND rating <= 2", req.SpaceID, b, next).Count(&low).Error
		points = append(points, point(b, ratio(low, total), total))
	}
	return points
}

func (s *Service) ciDiagnosisBreakdown(gdb *gorm.DB, req OverviewRequest) []BreakdownItem {
	q := gdb.Model(&store.CIDiagnosis{}).Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To)
	if req.ProjectID != "" {
		q = q.Where("connection_id = ?", req.ProjectID)
	}
	var rows []struct {
		RootCause string
		Count     int64
	}
	_ = q.Select("root_cause, COUNT(*) as count").Group("root_cause").Order("count desc").Scan(&rows).Error
	out := make([]BreakdownItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, BreakdownItem{Key: row.RootCause, Label: row.RootCause, Value: float64(row.Count), Unit: "count"})
	}
	return out
}

func (s *Service) feedbackRatingBreakdown(gdb *gorm.DB, req OverviewRequest) []BreakdownItem {
	var rows []struct {
		Rating int
		Count  int64
	}
	_ = gdb.Model(&store.Feedback{}).
		Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To).
		Select("rating, COUNT(*) as count").Group("rating").Order("rating asc").Scan(&rows).Error
	out := make([]BreakdownItem, 0, len(rows))
	for _, row := range rows {
		label := "未评分"
		if row.Rating > 0 {
			label = fmt.Sprintf("%d 星", row.Rating)
		}
		out = append(out, BreakdownItem{Key: fmt.Sprintf("%d", row.Rating), Label: label, Value: float64(row.Count), Unit: "count"})
	}
	return out
}

func normalizeRequest(req OverviewRequest) OverviewRequest {
	req.SpaceID = strings.TrimSpace(req.SpaceID)
	if req.SpaceID == "" {
		req.SpaceID = "local"
	}
	req.Period = strings.ToLower(strings.TrimSpace(req.Period))
	if req.Period != "week" {
		req.Period = "day"
	}
	now := time.Now().UTC()
	if req.To.IsZero() {
		req.To = now
	}
	if req.From.IsZero() {
		req.From = req.To.AddDate(0, 0, -7)
	}
	if req.From.After(req.To) {
		req.From, req.To = req.To, req.From
	}
	return req
}

func ratioCard(id, label string, numerator, denominator int64, unit string) MetricCard {
	if denominator <= 0 {
		return MetricCard{ID: id, Label: label, Unit: unit, Status: "empty", Description: "当前时间窗口没有可聚合数据。"}
	}
	return MetricCard{
		ID: id, Label: label, Value: ratio(numerator, denominator), Unit: unit, Status: "ok",
		Numerator: numerator, Denominator: denominator,
	}
}

func durationCard(id, label string, totalMs, count int64) MetricCard {
	if count <= 0 {
		return MetricCard{ID: id, Label: label, Unit: "ms", Status: "empty", Description: "当前时间窗口没有完成样本。"}
	}
	return MetricCard{ID: id, Label: label, Value: round(float64(totalMs) / float64(count)), Unit: "ms", Status: "ok", Numerator: totalMs, Denominator: count}
}

func unavailableCard(id, label, message string) MetricCard {
	return MetricCard{ID: id, Label: label, Unit: "ratio", Status: "unavailable", Description: message}
}

func dataQuality(cards []MetricCard) []DataQualityNote {
	notes := []DataQualityNote{}
	for _, card := range cards {
		if card.Status == "ok" {
			continue
		}
		notes = append(notes, DataQualityNote{MetricID: card.ID, Status: card.Status, Message: card.Description})
	}
	return notes
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return round(float64(numerator) / float64(denominator))
}

func round(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func point(start time.Time, value float64, denominator int64) MetricPoint {
	status := "ok"
	if denominator <= 0 {
		status = "empty"
	}
	return MetricPoint{PeriodStart: start.Format(time.RFC3339), Value: value, Status: status}
}

func buckets(req OverviewRequest) []time.Time {
	start := truncate(req.From, req.Period)
	out := []time.Time{}
	for !start.After(req.To) {
		out = append(out, start)
		start = advance(start, req.Period)
	}
	return out
}

func truncate(t time.Time, period string) time.Time {
	t = t.UTC()
	switch period {
	case "week":
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		base := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return base.AddDate(0, 0, -(weekday - 1))
	default:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func advance(t time.Time, period string) time.Time {
	if period == "week" {
		return t.AddDate(0, 0, 7)
	}
	return t.AddDate(0, 0, 1)
}

func normalizeConclusion(conclusion, status string) string {
	conclusion = strings.ToLower(strings.TrimSpace(conclusion))
	if conclusion != "" {
		return conclusion
	}
	return strings.ToLower(strings.TrimSpace(status))
}

func Strings(raw string) []string {
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
