package releases

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

type Service struct {
	db *store.DB
}

type CreateRequest struct {
	SpaceID        string
	Version        string
	Title          string
	CanaryStrategy string
	CreatedBy      string
}

type ChecklistUpdate struct {
	ID          string `json:"id,omitempty"`
	ItemKey     string `json:"itemKey,omitempty"`
	Status      string `json:"status,omitempty"`
	EvidenceRef string `json:"evidenceRef,omitempty"`
}

type GateEvaluation struct {
	ReleaseID    string                    `json:"releaseId"`
	SpaceID      string                    `json:"spaceId"`
	Overall      string                    `json:"overall"`
	Results      []store.ReleaseGateResult `json:"results"`
	EvidenceRefs []string                  `json:"evidenceRefs"`
	EvaluatedAt  string                    `json:"evaluatedAt"`
}

type RollbackDrillRequest struct {
	SpaceID      string
	ReleaseID    string
	Scenario     string
	Status       string
	DurationMs   int64
	EvidenceRefs []string
	Notes        string
	CreatedBy    string
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(req CreateRequest) (store.ReleaseRecord, error) {
	spaceID := firstNonEmpty(req.SpaceID, "local")
	version := strings.TrimSpace(req.Version)
	if version == "" {
		return store.ReleaseRecord{}, fmt.Errorf("version is required")
	}
	now := time.Now().UTC()
	row := store.ReleaseRecord{
		ID: "rel_" + uuid.NewString(), SpaceID: spaceID, Version: version,
		Title:  firstNonEmpty(strings.TrimSpace(req.Title), version),
		Status: "draft", CanaryStrategy: strings.TrimSpace(req.CanaryStrategy),
		GateStatus: "pending", EvidenceRefsJSON: "[]", CreatedBy: strings.TrimSpace(req.CreatedBy),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return store.ReleaseRecord{}, err
	}
	if err := s.initializeChecklist(row); err != nil {
		return store.ReleaseRecord{}, err
	}
	return row, nil
}

func (s *Service) List(spaceID string, limit int) ([]store.ReleaseRecord, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []store.ReleaseRecord
	err := s.db.Where("space_id = ?", spaceID).Order("created_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Service) Checklist(spaceID, releaseID string) ([]store.ReleaseChecklistItem, error) {
	if err := s.requireRelease(spaceID, releaseID); err != nil {
		return nil, err
	}
	var rows []store.ReleaseChecklistItem
	err := s.db.Where("space_id = ? AND release_id = ?", firstNonEmpty(spaceID, "local"), releaseID).
		Order("item_key asc").Find(&rows).Error
	return rows, err
}

func (s *Service) PatchChecklist(spaceID, releaseID, actor string, updates []ChecklistUpdate) ([]store.ReleaseChecklistItem, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	if err := s.requireRelease(spaceID, releaseID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for _, update := range updates {
		q := s.db.Model(&store.ReleaseChecklistItem{}).Where("space_id = ? AND release_id = ?", spaceID, releaseID)
		if strings.TrimSpace(update.ID) != "" {
			q = q.Where("id = ?", strings.TrimSpace(update.ID))
		} else if strings.TrimSpace(update.ItemKey) != "" {
			q = q.Where("item_key = ?", strings.TrimSpace(update.ItemKey))
		} else {
			return nil, fmt.Errorf("checklist update requires id or itemKey")
		}
		fields := map[string]any{"updated_at": now, "updated_by": strings.TrimSpace(actor)}
		if strings.TrimSpace(update.Status) != "" {
			fields["status"] = normalizeStatus(update.Status, "pending")
		}
		if strings.TrimSpace(update.EvidenceRef) != "" {
			fields["evidence_ref"] = strings.TrimSpace(update.EvidenceRef)
		}
		if err := q.Updates(fields).Error; err != nil {
			return nil, err
		}
	}
	return s.Checklist(spaceID, releaseID)
}

func (s *Service) EvaluateGate(spaceID, releaseID string) (GateEvaluation, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	if err := s.requireRelease(spaceID, releaseID); err != nil {
		return GateEvaluation{}, err
	}
	now := time.Now().UTC()
	checks := []gateCheck{
		s.ciWorkflowGate(spaceID),
		s.doctorGate(spaceID, "M3"),
		s.doctorGate(spaceID, "ALL"),
		s.auditEvidenceGate(spaceID, "postgres.e2e_completed", "postgres_e2e", "Postgres e2e 迁移验证"),
		s.activeAlertsGate(spaceID),
		s.kpiFeedbackGate(spaceID),
		s.auditEvidenceGate(spaceID, "execgo.live_smoke", "execgo_live_smoke", "ExecGo live smoke"),
	}
	results := make([]store.ReleaseGateResult, 0, len(checks))
	evidence := []string{}
	overall := "pass"
	for _, check := range checks {
		if check.Status == "block" {
			overall = "block"
		} else if check.Status == "warn" && overall != "block" {
			overall = "warn"
		}
		evidence = append(evidence, check.EvidenceRefs...)
		results = append(results, store.ReleaseGateResult{
			ID: "rel_gate_" + uuid.NewString(), SpaceID: spaceID, ReleaseID: releaseID,
			GateKey: check.Key, Status: check.Status, Message: check.Message,
			EvidenceRefsJSON: mustJSON(check.EvidenceRefs), CreatedAt: now,
		})
	}
	if err := s.db.Create(&results).Error; err != nil {
		return GateEvaluation{}, err
	}
	if err := s.db.Model(&store.ReleaseRecord{}).Where("id = ? AND space_id = ?", releaseID, spaceID).
		Updates(map[string]any{"gate_status": overall, "evidence_refs_json": mustJSON(evidence), "updated_at": now}).Error; err != nil {
		return GateEvaluation{}, err
	}
	return GateEvaluation{
		ReleaseID: releaseID, SpaceID: spaceID, Overall: overall,
		Results: results, EvidenceRefs: evidence, EvaluatedAt: now.Format(time.RFC3339),
	}, nil
}

func (s *Service) CreateRollbackDrill(req RollbackDrillRequest) (store.RollbackDrill, error) {
	spaceID := firstNonEmpty(req.SpaceID, "local")
	if err := s.requireRelease(spaceID, req.ReleaseID); err != nil {
		return store.RollbackDrill{}, err
	}
	if strings.TrimSpace(req.Scenario) == "" {
		return store.RollbackDrill{}, fmt.Errorf("scenario is required")
	}
	now := time.Now().UTC()
	row := store.RollbackDrill{
		ID: "rollback_" + uuid.NewString(), SpaceID: spaceID, ReleaseID: req.ReleaseID,
		Scenario: strings.TrimSpace(req.Scenario), Status: normalizeStatus(req.Status, "recorded"),
		DurationMs: req.DurationMs, EvidenceRefsJSON: mustJSON(req.EvidenceRefs),
		Notes: strings.TrimSpace(req.Notes), CreatedBy: strings.TrimSpace(req.CreatedBy),
		CreatedAt: now, UpdatedAt: now,
	}
	return row, s.db.Create(&row).Error
}

func (s *Service) initializeChecklist(rel store.ReleaseRecord) error {
	now := time.Now().UTC()
	items := defaultChecklistItems()
	rows := make([]store.ReleaseChecklistItem, 0, len(items))
	for i, label := range items {
		rows = append(rows, store.ReleaseChecklistItem{
			ID: "rel_item_" + uuid.NewString(), SpaceID: rel.SpaceID, ReleaseID: rel.ID,
			ItemKey: fmt.Sprintf("mvp-%02d", i+1), Label: label, Status: "pending",
			CreatedAt: now, UpdatedAt: now,
		})
	}
	return s.db.Create(&rows).Error
}

func defaultChecklistItems() []string {
	return []string{
		"本次发布目标、范围、不包含范围已在评审中确认",
		"PRD、API、数据库文档已同步至最新版本",
		"P0 功能全部开发完成，前后端主链路可用",
		"后端单元测试、集成测试和回归测试通过",
		"严重缺陷 P0/P1 为 0",
		"关键接口、队列积压与 SSE 稳定性在可控范围",
		"JWT、RBAC、高风险审批与审计日志已验证",
		"数据库迁移、备份和回滚脚本已验证",
		"核心指标、告警规则和仪表盘已配置并测试",
		"发布窗口、值班人员、版本号与变更说明已确认",
		"灰度策略、观察指标和回滚触发条件已定义",
		"回滚流程演练完成并记录耗时",
		"T+0/T+1 上线后验证计划已明确",
	}
}

type gateCheck struct {
	Key          string
	Status       string
	Message      string
	EvidenceRefs []string
}

func (s *Service) ciWorkflowGate(spaceID string) gateCheck {
	var row store.CIRun
	err := s.db.Where("space_id = ?", spaceID).Order("created_at desc").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return gateCheck{Key: "ci_workflow", Status: "warn", Message: "暂无 CI workflow run 证据", EvidenceRefs: []string{"ci:missing"}}
	}
	if err != nil {
		return gateCheck{Key: "ci_workflow", Status: "block", Message: err.Error()}
	}
	status := "warn"
	msg := "最近 CI workflow 尚未成功完成"
	if normalizeConclusion(row.Conclusion, row.Status) == "success" {
		status, msg = "pass", "最近 CI workflow 成功"
	} else if normalizeConclusion(row.Conclusion, row.Status) == "failure" {
		status, msg = "block", "最近 CI workflow 失败"
	}
	return gateCheck{Key: "ci_workflow", Status: status, Message: msg, EvidenceRefs: []string{"ci_run:" + row.ID}}
}

func (s *Service) doctorGate(spaceID, suite string) gateCheck {
	var row store.AuditLog
	pattern := fmt.Sprintf("%%%q:%q%%", "suite", suite)
	err := s.db.Where("space_id = ? AND event_type = ? AND payload_json LIKE ?", spaceID, "doctor.suite_completed", pattern).
		Order("created_at desc").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return gateCheck{Key: "doctor_" + strings.ToLower(suite), Status: "warn", Message: "暂无 Doctor " + suite + " 通过证据", EvidenceRefs: []string{"doctor:" + suite + ":missing"}}
	}
	if err != nil {
		return gateCheck{Key: "doctor_" + strings.ToLower(suite), Status: "block", Message: err.Error()}
	}
	status := "pass"
	msg := "Doctor " + suite + " 最近一次通过"
	if strings.Contains(strings.ToLower(row.PayloadJSON), `"fail":`) && !strings.Contains(strings.ToLower(row.PayloadJSON), `"fail":0`) {
		status, msg = "block", "Doctor "+suite+" 最近一次存在失败项"
	}
	return gateCheck{Key: "doctor_" + strings.ToLower(suite), Status: status, Message: msg, EvidenceRefs: []string{"audit:" + row.ID}}
}

func (s *Service) auditEvidenceGate(spaceID, eventType, key, label string) gateCheck {
	var row store.AuditLog
	err := s.db.Where("space_id = ? AND event_type = ?", spaceID, eventType).Order("created_at desc").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return gateCheck{Key: key, Status: "warn", Message: "暂无 " + label + " 证据", EvidenceRefs: []string{key + ":missing"}}
	}
	if err != nil {
		return gateCheck{Key: key, Status: "block", Message: err.Error()}
	}
	status := "pass"
	msg := label + " 最近一次通过"
	payload := strings.ToLower(row.PayloadJSON)
	if strings.Contains(payload, "fail") || strings.Contains(payload, "block") {
		status, msg = "block", label+" 最近一次失败"
	}
	return gateCheck{Key: key, Status: status, Message: msg, EvidenceRefs: []string{"audit:" + row.ID}}
}

func (s *Service) activeAlertsGate(spaceID string) gateCheck {
	var critical, warn int64
	_ = s.db.Model(&store.AlertEvent{}).Where("space_id = ? AND status = ? AND severity IN ?", spaceID, "active", []string{"critical", "error"}).Count(&critical).Error
	_ = s.db.Model(&store.AlertEvent{}).Where("space_id = ? AND status = ? AND severity NOT IN ?", spaceID, "active", []string{"critical", "error"}).Count(&warn).Error
	if critical > 0 {
		return gateCheck{Key: "active_alerts", Status: "block", Message: fmt.Sprintf("存在 %d 个 critical/error active alert", critical), EvidenceRefs: []string{"alerts:critical"}}
	}
	if warn > 0 {
		return gateCheck{Key: "active_alerts", Status: "warn", Message: fmt.Sprintf("存在 %d 个非阻断 active alert", warn), EvidenceRefs: []string{"alerts:warn"}}
	}
	return gateCheck{Key: "active_alerts", Status: "pass", Message: "无 active alert", EvidenceRefs: []string{"alerts:none"}}
}

func (s *Service) kpiFeedbackGate(spaceID string) gateCheck {
	since := time.Now().UTC().AddDate(0, 0, -7)
	var total, low int64
	_ = s.db.Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ?", spaceID, since).Count(&total).Error
	_ = s.db.Model(&store.Feedback{}).Where("space_id = ? AND created_at >= ? AND rating > 0 AND rating <= 2", spaceID, since).Count(&low).Error
	if total == 0 {
		return gateCheck{Key: "kpi_feedback", Status: "warn", Message: "最近 7 天暂无反馈 KPI 样本", EvidenceRefs: []string{"kpi:feedback:empty"}}
	}
	rate := float64(low) / float64(total)
	if rate > 0.5 {
		return gateCheck{Key: "kpi_feedback", Status: "block", Message: fmt.Sprintf("低分反馈率 %.2f 超过阻断阈值", rate), EvidenceRefs: []string{"feedback:low_rate"}}
	}
	if rate > 0.35 {
		return gateCheck{Key: "kpi_feedback", Status: "warn", Message: fmt.Sprintf("低分反馈率 %.2f 超过观察阈值", rate), EvidenceRefs: []string{"feedback:low_rate"}}
	}
	return gateCheck{Key: "kpi_feedback", Status: "pass", Message: fmt.Sprintf("低分反馈率 %.2f 在阈值内", rate), EvidenceRefs: []string{"feedback:low_rate"}}
}

func (s *Service) requireRelease(spaceID, releaseID string) error {
	var row store.ReleaseRecord
	err := s.db.First(&row, "id = ? AND space_id = ?", strings.TrimSpace(releaseID), firstNonEmpty(spaceID, "local")).Error
	if err == gorm.ErrRecordNotFound {
		return fmt.Errorf("release not found")
	}
	return err
}

func normalizeConclusion(conclusion, status string) string {
	conclusion = strings.ToLower(strings.TrimSpace(conclusion))
	if conclusion != "" {
		return conclusion
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "completed" || status == "success" {
		return "success"
	}
	return status
}

func normalizeStatus(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
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
