package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/observability/derive"
	"github.com/ash-repwiki/ash/internal/store"
)

type Service struct {
	db  *store.DB
	ctx context.Context
}

type RuleInput struct {
	ID            string  `json:"id,omitempty"`
	Name          string  `json:"name"`
	Metric        string  `json:"metric"`
	Condition     string  `json:"condition,omitempty"`
	Threshold     float64 `json:"threshold"`
	WindowMinutes int     `json:"windowMinutes,omitempty"`
	Severity      string  `json:"severity,omitempty"`
	Enabled       bool    `json:"enabled"`
	Description   string  `json:"description,omitempty"`
}

type EvaluationResult struct {
	SpaceID     string             `json:"spaceId"`
	EvaluatedAt string             `json:"evaluatedAt"`
	Results     []RuleEvaluation   `json:"results"`
	Events      []store.AlertEvent `json:"events"`
}

type RuleEvaluation struct {
	RuleID       string   `json:"ruleId"`
	RuleName     string   `json:"ruleName"`
	Metric       string   `json:"metric"`
	Status       string   `json:"status"`
	Message      string   `json:"message"`
	Value        float64  `json:"value"`
	Threshold    float64  `json:"threshold"`
	EvidenceRefs []string `json:"evidenceRefs"`
}

type TraceView struct {
	SpaceID    string            `json:"spaceId"`
	TraceID    string            `json:"traceId"`
	Runs       []store.RunRecord `json:"runs"`
	Events     []store.RunEvent  `json:"events"`
	ToolCalls  []store.ToolCall  `json:"toolCalls"`
	AgentTasks []store.AgentTask `json:"agentTasks"`
	AuditLogs  []store.AuditLog  `json:"auditLogs"`
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

func (s *Service) ListEvents(spaceID, status string, limit int) ([]store.AlertEvent, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := s.gdb().Where("space_id = ?", spaceID)
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []store.AlertEvent
	err := q.Order("triggered_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Service) ListRules(spaceID string) ([]store.AlertRule, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	if err := s.ensureDefaultRules(spaceID); err != nil {
		return nil, err
	}
	var rows []store.AlertRule
	err := s.gdb().Where("space_id = ?", spaceID).Order("created_at asc").Find(&rows).Error
	return rows, err
}

func (s *Service) PutRules(spaceID string, inputs []RuleInput) ([]store.AlertRule, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	if err := s.ensureDefaultRules(spaceID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for _, input := range inputs {
		metric := strings.TrimSpace(input.Metric)
		if metric == "" {
			return nil, fmt.Errorf("metric is required")
		}
		row := store.AlertRule{
			SpaceID:       spaceID,
			Name:          firstNonEmpty(strings.TrimSpace(input.Name), metric),
			Metric:        metric,
			Condition:     firstNonEmpty(strings.TrimSpace(input.Condition), "gt"),
			Threshold:     input.Threshold,
			WindowMinutes: input.WindowMinutes,
			Severity:      firstNonEmpty(strings.TrimSpace(input.Severity), "warn"),
			Enabled:       input.Enabled,
			Description:   strings.TrimSpace(input.Description),
			UpdatedAt:     now,
		}
		if row.WindowMinutes <= 0 {
			row.WindowMinutes = 60
		}
		if strings.TrimSpace(input.ID) != "" {
			row.ID = strings.TrimSpace(input.ID)
			if err := s.gdb().Model(&store.AlertRule{}).Where("id = ? AND space_id = ?", row.ID, spaceID).Updates(map[string]any{
				"name": row.Name, "metric": row.Metric, "condition": row.Condition,
				"threshold": row.Threshold, "window_minutes": row.WindowMinutes,
				"severity": row.Severity, "enabled": row.Enabled,
				"description": row.Description, "updated_at": row.UpdatedAt,
			}).Error; err != nil {
				return nil, err
			}
			continue
		}
		row.ID = "alert_rule_" + uuid.NewString()
		row.CreatedAt = now
		if err := s.gdb().Where("space_id = ? AND metric = ?", spaceID, row.Metric).
			Assign(row).FirstOrCreate(&store.AlertRule{}).Error; err != nil {
			return nil, err
		}
	}
	return s.ListRules(spaceID)
}

func (s *Service) Evaluate(spaceID string) (EvaluationResult, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	rules, err := s.ListRules(spaceID)
	if err != nil {
		return EvaluationResult{}, err
	}
	now := time.Now().UTC()
	out := EvaluationResult{SpaceID: spaceID, EvaluatedAt: now.Format(time.RFC3339)}
	for _, rule := range rules {
		value, available, refs, message, err := s.metricValue(rule, now)
		if err != nil {
			return out, err
		}
		ev := RuleEvaluation{
			RuleID: rule.ID, RuleName: rule.Name, Metric: rule.Metric,
			Status: "ok", Message: message, Value: value, Threshold: rule.Threshold,
			EvidenceRefs: refs,
		}
		if !available {
			ev.Status = "unavailable"
		} else if rule.Enabled && compare(rule.Condition, value, rule.Threshold) {
			ev.Status = "alert"
			alert, err := s.upsertAlert(rule, value, refs, message, now)
			if err != nil {
				return out, err
			}
			out.Events = append(out.Events, alert)
		}
		out.Results = append(out.Results, ev)
	}
	return out, nil
}

func (s *Service) RecordLowFeedback(row store.Feedback) (store.AlertEvent, error) {
	if row.Rating <= 0 || row.Rating > 2 {
		return store.AlertEvent{}, nil
	}
	now := time.Now().UTC()
	event := store.AlertEvent{
		ID: "alert_evt_" + uuid.NewString(), SpaceID: firstNonEmpty(row.SpaceID, "local"),
		RuleName: "低分反馈", Severity: firstNonEmpty(row.Severity, "warn"), Status: "active",
		TargetType: "feedback", TargetID: row.ID, Fingerprint: "feedback_low_score:" + row.ID,
		Message:          fmt.Sprintf("收到低分反馈：%s/%s rating=%d", row.TargetType, row.TargetID, row.Rating),
		EvidenceRefsJSON: mustJSON([]string{"feedback:" + row.ID}),
		TriggeredAt:      now, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.gdb().Create(&event).Error; err != nil {
		return store.AlertEvent{}, err
	}
	return event, nil
}

func (s *Service) PrometheusText() (string, error) {
	return s.PrometheusTextWith(PrometheusOptions{})
}

// PrometheusTextWith emits Prometheus text; SpaceID empty = deployment-global aggregate.
func (s *Service) PrometheusTextWith(opts PrometheusOptions) (string, error) {
	opts = opts.normalized()
	lbl := opts.labelSuffix()
	var b strings.Builder
	writeHelp := func(name, help, typ string) {
		b.WriteString("# HELP " + name + " " + help + "\n")
		b.WriteString("# TYPE " + name + " " + typ + "\n")
	}
	if opts.scoped() {
		b.WriteString("# ASH metrics scope: space_id=" + label(opts.SpaceID) + "\n")
	} else {
		b.WriteString("# ASH metrics scope: global (all spaces; Postgres RLS bypass on /metrics scrape)\n")
	}
	writeHelp("run_total", "ASH runs by status.", "counter")
	for _, item := range s.countBy("runs", "status", opts) {
		b.WriteString(fmt.Sprintf("run_total{status=%q%s} %d\n", label(item.Key), lbl, item.Count))
	}
	var completed []store.RunRecord
	_ = s.scopedRunsQuery(opts).Where("finished_at IS NOT NULL").Find(&completed).Error
	var durationCount int64
	var durationMs int64
	for _, row := range completed {
		if row.FinishedAt == nil {
			continue
		}
		durationCount++
		durationMs += row.FinishedAt.Sub(row.StartedAt).Milliseconds()
	}
	writeHelp("run_duration_seconds", "ASH completed run duration.", "summary")
	if opts.scoped() {
		b.WriteString(fmt.Sprintf("run_duration_seconds{space_id=%q}_count %d\n", label(opts.SpaceID), durationCount))
		b.WriteString(fmt.Sprintf("run_duration_seconds{space_id=%q}_sum %.3f\n", label(opts.SpaceID), float64(durationMs)/1000))
	} else {
		b.WriteString(fmt.Sprintf("run_duration_seconds_count %d\n", durationCount))
		b.WriteString(fmt.Sprintf("run_duration_seconds_sum %.3f\n", float64(durationMs)/1000))
	}

	writeHelp("tool_calls_total", "ASH tool calls by status.", "counter")
	for _, item := range s.countBy("tool_calls", "status", opts) {
		b.WriteString(fmt.Sprintf("tool_calls_total{status=%q%s} %d\n", label(item.Key), lbl, item.Count))
	}
	agentQ := s.gdb().Model(&store.AgentTask{}).Where("status NOT IN ?", []string{"succeeded", "success", "completed", "done"})
	agentQ = s.applyPrometheusScope(agentQ, "agent_tasks", opts)
	var agentFailures int64
	_ = agentQ.Count(&agentFailures).Error
	writeHelp("agent_failures_total", "ASH agent task failures.", "counter")
	if opts.scoped() {
		b.WriteString(fmt.Sprintf("agent_failures_total{space_id=%q} %d\n", label(opts.SpaceID), agentFailures))
	} else {
		b.WriteString(fmt.Sprintf("agent_failures_total %d\n", agentFailures))
	}

	writeHelp("ci_diagnoses_total", "ASH CI diagnoses by root cause and decision.", "counter")
	diagQ := s.gdb().Model(&store.CIDiagnosis{}).
		Select("root_cause, decision_status, COUNT(*) as count").
		Group("root_cause, decision_status")
	if opts.scoped() {
		diagQ = diagQ.Where("space_id = ?", opts.SpaceID)
	}
	var diagRows []struct {
		RootCause      string
		DecisionStatus string
		Count          int64
	}
	_ = diagQ.Scan(&diagRows).Error
	for _, row := range diagRows {
		b.WriteString(fmt.Sprintf("ci_diagnoses_total{root_cause=%q,decision_status=%q%s} %d\n",
			label(row.RootCause), label(firstNonEmpty(row.DecisionStatus, "pending")), lbl, row.Count))
	}
	feedbackQ := s.gdb().Model(&store.Feedback{}).Where("rating > 0 AND rating <= 2")
	if opts.scoped() {
		feedbackQ = feedbackQ.Where("space_id = ?", opts.SpaceID)
	}
	var lowFeedback int64
	_ = feedbackQ.Count(&lowFeedback).Error
	writeHelp("feedback_low_score_total", "ASH low-score feedback records.", "counter")
	if opts.scoped() {
		b.WriteString(fmt.Sprintf("feedback_low_score_total{space_id=%q} %d\n", label(opts.SpaceID), lowFeedback))
	} else {
		b.WriteString(fmt.Sprintf("feedback_low_score_total %d\n", lowFeedback))
	}
	alertQ := s.gdb().Model(&store.AlertEvent{}).Where("status = ?", "active")
	if opts.scoped() {
		alertQ = alertQ.Where("space_id = ?", opts.SpaceID)
	}
	var activeAlerts int64
	_ = alertQ.Count(&activeAlerts).Error
	writeHelp("alerts_active", "ASH active alert events.", "gauge")
	if opts.scoped() {
		b.WriteString(fmt.Sprintf("alerts_active{space_id=%q} %d\n", label(opts.SpaceID), activeAlerts))
	} else {
		b.WriteString(fmt.Sprintf("alerts_active %d\n", activeAlerts))
	}
	if os.Getenv("ASH_METRICS_EVENT_REPLAY") == "1" {
		events, err := derive.LoadFromDB(s.gdb(), derive.LoadOptions{SpaceID: opts.SpaceID})
		if err == nil && len(events) > 0 {
			b.WriteString("\n")
			b.WriteString(derive.Replay(events).PrometheusText())
		}
	}
	return b.String(), nil
}

func (s *Service) Trace(spaceID, traceID string) (TraceView, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return TraceView{}, fmt.Errorf("traceId is required")
	}
	out := TraceView{SpaceID: spaceID, TraceID: traceID}
	if err := s.gdb().Where("space_id = ? AND trace_id = ?", spaceID, traceID).Order("created_at asc").Find(&out.Runs).Error; err != nil {
		return out, err
	}
	runIDs := make([]string, 0, len(out.Runs))
	for _, run := range out.Runs {
		runIDs = append(runIDs, run.ID)
	}
	if len(runIDs) > 0 {
		_ = s.gdb().Where("run_id IN ?", runIDs).Order("created_at asc").Find(&out.Events).Error
		_ = s.gdb().Where("trace_id = ? OR run_id IN ?", traceID, runIDs).Order("created_at asc").Find(&out.ToolCalls).Error
		_ = s.gdb().Where("trace_id = ? OR run_id IN ?", traceID, runIDs).Order("created_at asc").Find(&out.AgentTasks).Error
		_ = s.gdb().Where("space_id = ? AND (trace_id = ? OR run_id IN ?)", spaceID, traceID, runIDs).Order("created_at asc").Find(&out.AuditLogs).Error
	} else {
		_ = s.gdb().Where("trace_id = ?", traceID).Order("created_at asc").Find(&out.ToolCalls).Error
		_ = s.gdb().Where("trace_id = ?", traceID).Order("created_at asc").Find(&out.AgentTasks).Error
		_ = s.gdb().Where("space_id = ? AND trace_id = ?", spaceID, traceID).Order("created_at asc").Find(&out.AuditLogs).Error
	}
	return out, nil
}

func (s *Service) metricValue(rule store.AlertRule, now time.Time) (float64, bool, []string, string, error) {
	window := now.Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	switch rule.Metric {
	case "run_failure_rate":
		var total, failed int64
		if err := s.gdb().Model(&store.RunRecord{}).Where("space_id = ? AND started_at >= ?", rule.SpaceID, window).Count(&total).Error; err != nil {
			return 0, false, nil, "", err
		}
		if err := s.gdb().Model(&store.RunRecord{}).
			Where("space_id = ? AND started_at >= ? AND status IN ?", rule.SpaceID, window, []string{"failed", "error", "canceled"}).
			Count(&failed).Error; err != nil {
			return 0, false, nil, "", err
		}
		return ratio(failed, total), total > 0, []string{fmt.Sprintf("runs:failed=%d,total=%d", failed, total)}, "运行失败率超过阈值", nil
	case "api_error_rate":
		var total, failed int64
		if err := s.gdb().Model(&store.AuditLog{}).Where("space_id = ? AND created_at >= ?", rule.SpaceID, window).Count(&total).Error; err != nil {
			return 0, false, nil, "", err
		}
		if err := s.gdb().Model(&store.AuditLog{}).Where("space_id = ? AND created_at >= ? AND event_type LIKE ?", rule.SpaceID, window, "%failed%").Count(&failed).Error; err != nil {
			return 0, false, nil, "", err
		}
		return ratio(failed, total), total > 0, []string{fmt.Sprintf("audit:failed=%d,total=%d", failed, total)}, "API/平台失败事件率超过阈值", nil
	case "queue_backlog_minutes":
		cutoff := now.Add(-time.Duration(rule.Threshold) * time.Minute)
		var rows []store.RunStep
		if err := s.gdb().Where("created_at <= ? AND started_at IS NULL AND status IN ?", cutoff, []string{"pending", "queued", "running"}).Limit(5).Find(&rows).Error; err != nil {
			return 0, false, nil, "", err
		}
		refs := make([]string, 0, len(rows))
		for _, row := range rows {
			refs = append(refs, "run_step:"+row.ID)
		}
		return float64(len(rows)), true, refs, "存在长时间未开始的运行步骤", nil
	case "low_feedback_rate":
		var total, low int64
		if err := s.gdb().Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ?", rule.SpaceID, window).Count(&total).Error; err != nil {
			return 0, false, nil, "", err
		}
		if err := s.gdb().Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ? AND rating > 0 AND rating <= 2", rule.SpaceID, window).Count(&low).Error; err != nil {
			return 0, false, nil, "", err
		}
		return ratio(low, total), total > 0, []string{fmt.Sprintf("feedback:low=%d,total=%d", low, total)}, "低分反馈率超过阈值", nil
	case "postgres_live_gate":
		return s.liveGateValue(rule.SpaceID, window, "postgres.e2e_completed", "Postgres live gate 最近窗口无通过证据")
	case "execgo_live_gate":
		return s.liveGateValue(rule.SpaceID, window, "execgo.live_smoke", "ExecGo live gate 最近窗口无通过证据")
	default:
		return 0, false, nil, "未知指标，规则暂不可评估", nil
	}
}

func (s *Service) liveGateValue(spaceID string, since time.Time, eventType, missing string) (float64, bool, []string, string, error) {
	var row store.AuditLog
	err := s.gdb().Where("space_id = ? AND event_type = ? AND created_at >= ?", spaceID, eventType, since).
		Order("created_at desc").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return 0, false, nil, missing, nil
	}
	if err != nil {
		return 0, false, nil, "", err
	}
	if strings.Contains(strings.ToLower(row.PayloadJSON), "fail") || strings.Contains(strings.ToLower(row.PayloadJSON), "block") {
		return 1, true, []string{"audit:" + row.ID}, "live gate 最近一次失败", nil
	}
	return 0, true, []string{"audit:" + row.ID}, "live gate 最近一次通过", nil
}

func (s *Service) upsertAlert(rule store.AlertRule, value float64, refs []string, message string, now time.Time) (store.AlertEvent, error) {
	fingerprint := fmt.Sprintf("%s:%s", rule.SpaceID, rule.Metric)
	var existing store.AlertEvent
	err := s.gdb().First(&existing, "space_id = ? AND fingerprint = ? AND status = ?", rule.SpaceID, fingerprint, "active").Error
	if err == nil {
		existing.Message = fmt.Sprintf("%s value=%.4f threshold=%.4f", message, value, rule.Threshold)
		existing.EvidenceRefsJSON = mustJSON(refs)
		existing.UpdatedAt = now
		return existing, s.gdb().Save(&existing).Error
	}
	if err != gorm.ErrRecordNotFound {
		return store.AlertEvent{}, err
	}
	row := store.AlertEvent{
		ID: "alert_evt_" + uuid.NewString(), SpaceID: rule.SpaceID,
		RuleID: rule.ID, RuleName: rule.Name, Severity: rule.Severity, Status: "active",
		TargetType: "metric", TargetID: rule.Metric, Fingerprint: fingerprint,
		Message:          fmt.Sprintf("%s value=%.4f threshold=%.4f", message, value, rule.Threshold),
		EvidenceRefsJSON: mustJSON(refs), TriggeredAt: now, CreatedAt: now, UpdatedAt: now,
	}
	return row, s.gdb().Create(&row).Error
}

func (s *Service) ensureDefaultRules(spaceID string) error {
	now := time.Now().UTC()
	for _, rule := range defaultRules(spaceID, now) {
		if err := s.gdb().Where("space_id = ? AND metric = ?", spaceID, rule.Metric).
			Assign(map[string]any{
				"name": rule.Name, "condition": rule.Condition, "threshold": rule.Threshold,
				"window_minutes": rule.WindowMinutes, "severity": rule.Severity,
				"description": rule.Description, "updated_at": now,
			}).FirstOrCreate(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}

func defaultRules(spaceID string, now time.Time) []store.AlertRule {
	return []store.AlertRule{
		rule(spaceID, "运行失败率", "run_failure_rate", 0.3, 60, "critical", "最近窗口内 failed/error/canceled 运行占比超过阈值", now),
		rule(spaceID, "API 错误率", "api_error_rate", 0.2, 60, "warn", "基于 audit failed 事件估算 API/平台错误率", now),
		rule(spaceID, "队列积压", "queue_backlog_minutes", 30, 60, "warn", "存在超过阈值分钟数仍未开始的步骤", now),
		rule(spaceID, "低分反馈率", "low_feedback_rate", 0.35, 60, "warn", "最近窗口低分反馈占比超过阈值", now),
		rule(spaceID, "Postgres live gate", "postgres_live_gate", 0, 1440, "warn", "读取 Postgres e2e live gate 审计证据", now),
		rule(spaceID, "ExecGo live gate", "execgo_live_gate", 0, 1440, "warn", "读取 ExecGo live smoke 审计证据", now),
	}
}

func rule(spaceID, name, metric string, threshold float64, window int, severity, desc string, now time.Time) store.AlertRule {
	return store.AlertRule{
		ID: "alert_rule_" + uuid.NewString(), SpaceID: spaceID, Name: name, Metric: metric,
		Condition: "gt", Threshold: threshold, WindowMinutes: window, Severity: severity,
		Enabled: true, Description: desc, CreatedAt: now, UpdatedAt: now,
	}
}

type countRow struct {
	Key   string
	Count int64
}

func compare(condition string, value, threshold float64) bool {
	switch strings.ToLower(strings.TrimSpace(condition)) {
	case "gte", ">=":
		return value >= threshold
	case "lt", "<":
		return value < threshold
	case "lte", "<=":
		return value <= threshold
	default:
		return value > threshold
	}
}

func ratio(num, den int64) float64 {
	if den <= 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func mustJSON(values []string) string {
	raw, _ := json.Marshal(values)
	return string(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func label(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}
