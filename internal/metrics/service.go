package metrics

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
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
	db *store.DB
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Overview(req OverviewRequest) (Overview, error) {
	req = normalizeRequest(req)
	out := Overview{
		SpaceID: req.SpaceID, ProjectID: req.ProjectID,
		From: req.From.Format(time.RFC3339), To: req.To.Format(time.RFC3339),
		Period: req.Period, GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	runStats, err := s.runStats(req)
	if err != nil {
		return out, err
	}
	feedbackStats, err := s.feedbackStats(req)
	if err != nil {
		return out, err
	}
	memoryStats, err := s.memoryStats(req)
	if err != nil {
		return out, err
	}
	ciStats, err := s.ciStats(req)
	if err != nil {
		return out, err
	}
	apiStats, err := s.apiStats(req)
	if err != nil {
		return out, err
	}
	queueStats, err := s.queueStats(req)
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
		unavailableCard("KPI-08", "SSE 稳定率", "当前缺少 SSE 会话成功/失败事件源，v1 不造假数据。"),
		ratioCard("KPI-09", "API 错误率", apiStats.errors, apiStats.total, "ratio"),
		durationCard("KPI-10", "队列积压时长", queueStats.totalWaitMs, queueStats.count),
	}
	out.Trends = []MetricTrend{
		{MetricID: "KPI-01", Points: s.runSuccessTrend(req)},
		{MetricID: "KPI-04", Points: s.ciFirstPassTrend(req)},
		{MetricID: "KPI-06", Points: s.lowFeedbackTrend(req)},
	}
	out.Breakdowns = []MetricBreakdown{
		{ID: "ciDiagnosis", Label: "CI 诊断根因", Items: s.ciDiagnosisBreakdown(req)},
		{ID: "feedbackRatings", Label: "反馈评分分布", Items: s.feedbackRatingBreakdown(req)},
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

func (s *Service) runStats(req OverviewRequest) (runStatsResult, error) {
	var rows []store.RunRecord
	err := s.db.Where("space_id = ? AND started_at >= ? AND started_at <= ?", req.SpaceID, req.From, req.To).Find(&rows).Error
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

func (s *Service) feedbackStats(req OverviewRequest) (feedbackStatsResult, error) {
	var rows []store.Feedback
	err := s.db.Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To).Find(&rows).Error
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

func (s *Service) memoryStats(req OverviewRequest) (memoryStatsResult, error) {
	var out memoryStatsResult
	if err := s.db.Model(&store.RunEvent{}).
		Where("created_at >= ? AND created_at <= ? AND type IN ?", req.From, req.To, []string{"memory.injected", "memory.query_failed"}).
		Count(&out.queries).Error; err != nil {
		return out, err
	}
	if err := s.db.Model(&store.AuditLog{}).
		Where("space_id = ? AND created_at >= ? AND created_at <= ? AND event_type = ?", req.SpaceID, req.From, req.To, "memory.hit_used").
		Count(&out.used).Error; err != nil {
		return out, err
	}
	if out.queries == 0 {
		if err := s.db.Model(&store.RunEvent{}).
			Where("created_at >= ? AND created_at <= ? AND type = ?", req.From, req.To, "memory.hit_used").
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

func (s *Service) ciStats(req OverviewRequest) (ciStatsResult, error) {
	var runs []store.CIRun
	q := s.db.Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To)
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
	dq := s.db.Model(&store.CIDiagnosis{}).Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To)
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

func (s *Service) apiStats(req OverviewRequest) (apiStatsResult, error) {
	var total, errors int64
	if err := s.db.Model(&store.AuditLog{}).
		Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To).
		Count(&total).Error; err != nil {
		return apiStatsResult{}, err
	}
	if err := s.db.Model(&store.AuditLog{}).
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

func (s *Service) queueStats(req OverviewRequest) (queueStatsResult, error) {
	var rows []store.RunStep
	if err := s.db.Where("created_at >= ? AND created_at <= ? AND started_at IS NOT NULL", req.From, req.To).Find(&rows).Error; err != nil {
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

func (s *Service) runSuccessTrend(req OverviewRequest) []MetricPoint {
	buckets := buckets(req)
	points := make([]MetricPoint, 0, len(buckets))
	for _, b := range buckets {
		next := advance(b, req.Period)
		var started, success int64
		_ = s.db.Model(&store.RunRecord{}).Where("space_id = ? AND started_at >= ? AND started_at < ?", req.SpaceID, b, next).Count(&started).Error
		_ = s.db.Model(&store.RunRecord{}).Where("space_id = ? AND started_at >= ? AND started_at < ? AND status = ?", req.SpaceID, b, next, "finished").Count(&success).Error
		points = append(points, point(b, ratio(success, started), started))
	}
	return points
}

func (s *Service) ciFirstPassTrend(req OverviewRequest) []MetricPoint {
	buckets := buckets(req)
	points := make([]MetricPoint, 0, len(buckets))
	for _, b := range buckets {
		next := advance(b, req.Period)
		q := s.db.Model(&store.CIRun{}).Where("space_id = ? AND created_at >= ? AND created_at < ? AND (attempt = ? OR attempt = ?)", req.SpaceID, b, next, 0, 1)
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

func (s *Service) lowFeedbackTrend(req OverviewRequest) []MetricPoint {
	buckets := buckets(req)
	points := make([]MetricPoint, 0, len(buckets))
	for _, b := range buckets {
		next := advance(b, req.Period)
		var total, low int64
		_ = s.db.Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ? AND created_at < ?", req.SpaceID, b, next).Count(&total).Error
		_ = s.db.Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ? AND created_at < ? AND rating > 0 AND rating <= 2", req.SpaceID, b, next).Count(&low).Error
		points = append(points, point(b, ratio(low, total), total))
	}
	return points
}

func (s *Service) ciDiagnosisBreakdown(req OverviewRequest) []BreakdownItem {
	q := s.db.Model(&store.CIDiagnosis{}).Where("space_id = ? AND created_at >= ? AND created_at <= ?", req.SpaceID, req.From, req.To)
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

func (s *Service) feedbackRatingBreakdown(req OverviewRequest) []BreakdownItem {
	var rows []struct {
		Rating int
		Count  int64
	}
	_ = s.db.Model(&store.Feedback{}).
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
