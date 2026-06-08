package ci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ash-repwiki/ash/internal/store"
)

type SecretResolver func(spaceID, secretID string) (string, error)

type Provider interface {
	ListWorkflowRuns(ctx context.Context, conn store.RepoConnection, token string, limit int) ([]store.CIRun, error)
	GetRunJobs(ctx context.Context, conn store.RepoConnection, token, providerRunID string) ([]store.CIJob, error)
	GetJobLogs(ctx context.Context, conn store.RepoConnection, token, providerJobID string) (string, error)
}

type Service struct {
	db             *store.DB
	providers      map[string]Provider
	secretResolver SecretResolver
	ctx            context.Context
}

type CreateConnectionRequest struct {
	SpaceID       string
	Provider      string
	Owner         string
	Repo          string
	DefaultBranch string
	SecretID      string
	CreatedBy     string
}

type DiagnoseRequest struct {
	SpaceID      string
	ConnectionID string
	RunID        string
	JobID        string
	LogText      string
}

type ConnectionTestResponse struct {
	OK        bool   `json:"ok"`
	Provider  string `json:"provider"`
	Message   string `json:"message,omitempty"`
	CheckedAt string `json:"checkedAt"`
}

type DiagnosisResponse struct {
	ID             string   `json:"id"`
	SpaceID        string   `json:"spaceId"`
	ConnectionID   string   `json:"connectionId,omitempty"`
	RunID          string   `json:"runId,omitempty"`
	JobID          string   `json:"jobId,omitempty"`
	Status         string   `json:"status"`
	RootCause      string   `json:"rootCause"`
	FixSuggestions []string `json:"fixSuggestions"`
	EvidenceRefs   []string `json:"evidenceRefs"`
	Confidence     float64  `json:"confidence"`
	Adopted        bool     `json:"adopted"`
	DecisionStatus string   `json:"decisionStatus"`
	DecisionReason string   `json:"decisionReason,omitempty"`
	DecidedBy      string   `json:"decidedBy,omitempty"`
	DecidedAt      string   `json:"decidedAt,omitempty"`
	LogDigest      string   `json:"logDigest,omitempty"`
	CreatedAt      string   `json:"createdAt"`
}

type ListDiagnosesRequest struct {
	SpaceID        string
	ConnectionID   string
	RunID          string
	JobID          string
	DecisionStatus string
	Limit          int
}

type DecideDiagnosisRequest struct {
	SpaceID     string
	DiagnosisID string
	Decision    string
	Reason      string
	ActorID     string
}

type PatternRule struct {
	Cause       string
	Confidence  float64
	Suggestions []string
	Patterns    []*regexp.Regexp
}

func NewService(db *store.DB, resolver SecretResolver) *Service {
	return &Service{
		db:             db,
		secretResolver: resolver,
		providers: map[string]Provider{
			"github": GitHubProvider{Client: http.DefaultClient},
		},
	}
}

func (s *Service) WithProvider(name string, provider Provider) *Service {
	if s.providers == nil {
		s.providers = map[string]Provider{}
	}
	s.providers[normalizeProvider(name)] = provider
	return s
}

// WithContext returns a shallow copy bound to ctx for Postgres RLS session vars.
func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	return &Service{
		db: s.db, providers: s.providers, secretResolver: s.secretResolver, ctx: ctx,
	}
}

func (s *Service) q(callCtx context.Context) *gorm.DB {
	ctx := callCtx
	if ctx == nil {
		ctx = s.ctx
	}
	if ctx != nil {
		return s.db.WithContext(ctx)
	}
	return s.db.DB
}

func (s *Service) CreateConnection(req CreateConnectionRequest) (store.RepoConnection, error) {
	provider := normalizeProvider(req.Provider)
	if provider == "" {
		provider = "github"
	}
	if provider != "github" {
		return store.RepoConnection{}, fmt.Errorf("unsupported repo provider %q", req.Provider)
	}
	owner := strings.TrimSpace(req.Owner)
	repo := strings.TrimSpace(req.Repo)
	if owner == "" || repo == "" {
		return store.RepoConnection{}, fmt.Errorf("owner and repo are required")
	}
	if strings.TrimSpace(req.SecretID) == "" {
		return store.RepoConnection{}, fmt.Errorf("secretId is required; plaintext tokens are not accepted")
	}
	now := time.Now().UTC()
	row := store.RepoConnection{
		ID:            "repo_conn_" + uuid.NewString(),
		SpaceID:       firstNonEmpty(req.SpaceID, "local"),
		Provider:      provider,
		Owner:         owner,
		Repo:          repo,
		DefaultBranch: firstNonEmpty(strings.TrimSpace(req.DefaultBranch), "main"),
		SecretID:      strings.TrimSpace(req.SecretID),
		Status:        "active",
		CreatedBy:     strings.TrimSpace(req.CreatedBy),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.q(nil).Create(&row).Error; err != nil {
		return store.RepoConnection{}, err
	}
	return row, nil
}

func (s *Service) ListConnections(spaceID string) ([]store.RepoConnection, error) {
	var rows []store.RepoConnection
	err := s.q(nil).Where("space_id = ?", firstNonEmpty(spaceID, "local")).Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (s *Service) TestConnection(ctx context.Context, spaceID, connectionID string) (ConnectionTestResponse, error) {
	spaceID = firstNonEmpty(spaceID, "local")
	conn, err := s.connection(spaceID, connectionID)
	if err != nil {
		return ConnectionTestResponse{}, err
	}
	providerName := normalizeProvider(conn.Provider)
	provider, ok := s.providers[providerName]
	if !ok || provider == nil {
		return ConnectionTestResponse{}, fmt.Errorf("provider %q is not configured", conn.Provider)
	}
	token, err := s.resolveSecret(conn.SpaceID, conn.SecretID)
	if err != nil {
		return ConnectionTestResponse{}, err
	}
	_, err = provider.ListWorkflowRuns(ctx, conn, token, 1)
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		return ConnectionTestResponse{OK: false, Provider: providerName, Message: err.Error(), CheckedAt: checkedAt}, nil
	}
	return ConnectionTestResponse{OK: true, Provider: providerName, Message: "connection ok", CheckedAt: checkedAt}, nil
}

func (s *Service) ListRuns(ctx context.Context, spaceID, connectionID string, limit int, sync bool) ([]store.CIRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	spaceID = firstNonEmpty(spaceID, "local")
	if sync && strings.TrimSpace(connectionID) != "" {
		if err := s.SyncRuns(ctx, spaceID, connectionID, limit); err != nil {
			return nil, err
		}
	}
	q := s.q(ctx).Where("space_id = ?", spaceID)
	if strings.TrimSpace(connectionID) != "" {
		q = q.Where("connection_id = ?", strings.TrimSpace(connectionID))
	}
	var rows []store.CIRun
	err := q.Order("created_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Service) ListJobs(ctx context.Context, spaceID, runID string, limit int, sync bool) ([]store.CIJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	spaceID = firstNonEmpty(spaceID, "local")
	runID = strings.TrimSpace(runID)
	if sync {
		if runID == "" {
			return nil, fmt.Errorf("runId is required when sync=true")
		}
		if err := s.SyncJobs(ctx, spaceID, runID); err != nil {
			return nil, err
		}
	}
	q := s.q(ctx).Where("space_id = ?", spaceID)
	if runID != "" {
		q = q.Where("ci_run_id = ?", runID)
	}
	var rows []store.CIJob
	err := q.Order("created_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Service) SyncJobs(ctx context.Context, spaceID, runID string) error {
	spaceID = firstNonEmpty(spaceID, "local")
	var run store.CIRun
	if err := s.q(ctx).First(&run, "id = ? AND space_id = ?", strings.TrimSpace(runID), spaceID).Error; err != nil {
		return fmt.Errorf("ci run not found: %w", err)
	}
	conn, err := s.connection(spaceID, run.ConnectionID)
	if err != nil {
		return err
	}
	provider, ok := s.providers[normalizeProvider(conn.Provider)]
	if !ok || provider == nil {
		return fmt.Errorf("provider %q is not configured", conn.Provider)
	}
	token, err := s.resolveSecret(conn.SpaceID, conn.SecretID)
	if err != nil {
		return err
	}
	jobs, err := provider.GetRunJobs(ctx, conn, token, run.ProviderRunID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, row := range jobs {
		row.SpaceID = conn.SpaceID
		row.ConnectionID = conn.ID
		row.CIRunID = run.ID
		row.UpdatedAt = now
		if row.CreatedAt.IsZero() {
			row.CreatedAt = now
		}
		if row.Attempt == 0 {
			row.Attempt = run.Attempt
		}
		if row.ID == "" {
			row.ID = "ci_job_" + uuid.NewString()
		}
		if err := s.q(ctx).Where("connection_id = ? AND provider_job_id = ?", conn.ID, row.ProviderJobID).
			Assign(row).FirstOrCreate(&store.CIJob{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SyncRuns(ctx context.Context, spaceID, connectionID string, limit int) error {
	conn, err := s.connection(spaceID, connectionID)
	if err != nil {
		return err
	}
	provider, ok := s.providers[normalizeProvider(conn.Provider)]
	if !ok || provider == nil {
		return fmt.Errorf("provider %q is not configured", conn.Provider)
	}
	token, err := s.resolveSecret(conn.SpaceID, conn.SecretID)
	if err != nil {
		return err
	}
	runs, err := provider.ListWorkflowRuns(ctx, conn, token, limit)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, row := range runs {
		row.SpaceID = conn.SpaceID
		row.ConnectionID = conn.ID
		row.UpdatedAt = now
		if row.CreatedAt.IsZero() {
			row.CreatedAt = now
		}
		if row.ID == "" {
			row.ID = "ci_run_" + uuid.NewString()
		}
		if err := s.q(ctx).Where("connection_id = ? AND provider_run_id = ?", conn.ID, row.ProviderRunID).
			Assign(row).FirstOrCreate(&store.CIRun{}).Error; err != nil {
			return err
		}
	}
	conn.LastSyncAt = &now
	conn.UpdatedAt = now
	return s.q(ctx).Save(&conn).Error
}

func (s *Service) Diagnose(ctx context.Context, req DiagnoseRequest) (DiagnosisResponse, error) {
	spaceID := firstNonEmpty(req.SpaceID, "local")
	logText := req.LogText
	connID := strings.TrimSpace(req.ConnectionID)
	runID := strings.TrimSpace(req.RunID)
	jobID := strings.TrimSpace(req.JobID)
	var conn store.RepoConnection
	if connID != "" {
		var err error
		conn, err = s.connection(spaceID, connID)
		if err != nil {
			return DiagnosisResponse{}, err
		}
	}
	var run store.CIRun
	if runID != "" {
		if err := s.q(ctx).First(&run, "id = ? AND space_id = ?", runID, spaceID).Error; err != nil {
			return DiagnosisResponse{}, fmt.Errorf("ci run not found: %w", err)
		}
		if connID == "" {
			connID = run.ConnectionID
		}
	}
	var job store.CIJob
	if jobID != "" {
		if err := s.q(ctx).First(&job, "id = ? AND space_id = ?", jobID, spaceID).Error; err != nil {
			return DiagnosisResponse{}, fmt.Errorf("ci job not found: %w", err)
		}
		if connID == "" {
			connID = job.ConnectionID
		}
		if runID == "" {
			runID = job.CIRunID
		}
	}
	if conn.ID == "" && connID != "" {
		var err error
		conn, err = s.connection(spaceID, connID)
		if err != nil {
			return DiagnosisResponse{}, err
		}
	}
	if strings.TrimSpace(logText) == "" && job.ProviderJobID != "" && conn.ID != "" {
		provider, ok := s.providers[normalizeProvider(conn.Provider)]
		if !ok || provider == nil {
			return DiagnosisResponse{}, fmt.Errorf("provider %q is not configured", conn.Provider)
		}
		token, err := s.resolveSecret(conn.SpaceID, conn.SecretID)
		if err != nil {
			return DiagnosisResponse{}, err
		}
		fetched, err := provider.GetJobLogs(ctx, conn, token, job.ProviderJobID)
		if err != nil {
			return DiagnosisResponse{}, err
		}
		logText = fetched
		if digest := digestText(logText); digest != "" && job.ID != "" {
			_ = s.q(ctx).Model(&store.CIJob{}).
				Where("id = ? AND space_id = ?", job.ID, spaceID).
				Updates(map[string]any{"log_digest": digest, "updated_at": time.Now().UTC()}).Error
		}
	}
	if strings.TrimSpace(logText) == "" {
		return DiagnosisResponse{}, fmt.Errorf("logText is required unless a connected job can provide logs")
	}
	out := DiagnoseLog(logText)
	now := time.Now().UTC()
	row := store.CIDiagnosis{
		ID:                 "ci_diag_" + uuid.NewString(),
		SpaceID:            spaceID,
		ConnectionID:       connID,
		CIRunID:            runID,
		CIJobID:            jobID,
		Status:             out.Status,
		RootCause:          out.RootCause,
		FixSuggestionsJSON: mustJSON(out.FixSuggestions),
		EvidenceRefsJSON:   mustJSON(out.EvidenceRefs),
		Confidence:         out.Confidence,
		DecisionStatus:     "pending",
		LogDigest:          out.LogDigest,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.q(ctx).Create(&row).Error; err != nil {
		return DiagnosisResponse{}, err
	}
	out.ID = row.ID
	out.SpaceID = row.SpaceID
	out.ConnectionID = row.ConnectionID
	out.RunID = row.CIRunID
	out.JobID = row.CIJobID
	out.Adopted = row.Adopted
	out.DecisionStatus = row.DecisionStatus
	out.DecisionReason = row.DecisionReason
	out.DecidedBy = row.DecidedBy
	if row.DecidedAt != nil {
		out.DecidedAt = row.DecidedAt.Format(time.RFC3339)
	}
	out.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	return out, nil
}

func (s *Service) ListDiagnoses(req ListDiagnosesRequest) ([]DiagnosisResponse, error) {
	spaceID := firstNonEmpty(req.SpaceID, "local")
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := s.q(nil).Where("space_id = ?", spaceID)
	if strings.TrimSpace(req.ConnectionID) != "" {
		q = q.Where("connection_id = ?", strings.TrimSpace(req.ConnectionID))
	}
	if strings.TrimSpace(req.RunID) != "" {
		q = q.Where("ci_run_id = ?", strings.TrimSpace(req.RunID))
	}
	if strings.TrimSpace(req.JobID) != "" {
		q = q.Where("ci_job_id = ?", strings.TrimSpace(req.JobID))
	}
	if strings.TrimSpace(req.DecisionStatus) != "" {
		q = q.Where("decision_status = ?", strings.TrimSpace(req.DecisionStatus))
	}
	var rows []store.CIDiagnosis
	if err := q.Order("created_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DiagnosisResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, diagnosisFromRow(row))
	}
	return out, nil
}

func (s *Service) DecideDiagnosis(req DecideDiagnosisRequest) (DiagnosisResponse, error) {
	decision := strings.TrimSpace(strings.ToLower(req.Decision))
	if decision != "adopted" && decision != "dismissed" {
		return DiagnosisResponse{}, fmt.Errorf("decision must be adopted or dismissed")
	}
	spaceID := firstNonEmpty(req.SpaceID, "local")
	var row store.CIDiagnosis
	if err := s.q(nil).First(&row, "id = ? AND space_id = ?", strings.TrimSpace(req.DiagnosisID), spaceID).Error; err != nil {
		return DiagnosisResponse{}, fmt.Errorf("ci diagnosis not found: %w", err)
	}
	now := time.Now().UTC()
	row.DecisionStatus = decision
	row.Adopted = decision == "adopted"
	row.DecisionReason = strings.TrimSpace(req.Reason)
	row.DecidedBy = strings.TrimSpace(req.ActorID)
	row.DecidedAt = &now
	row.UpdatedAt = now
	if err := s.q(nil).Save(&row).Error; err != nil {
		return DiagnosisResponse{}, err
	}
	return diagnosisFromRow(row), nil
}

func diagnosisFromRow(row store.CIDiagnosis) DiagnosisResponse {
	out := DiagnosisResponse{
		ID:             row.ID,
		SpaceID:        row.SpaceID,
		ConnectionID:   row.ConnectionID,
		RunID:          row.CIRunID,
		JobID:          row.CIJobID,
		Status:         row.Status,
		RootCause:      row.RootCause,
		FixSuggestions: parseJSONArray(row.FixSuggestionsJSON),
		EvidenceRefs:   parseJSONArray(row.EvidenceRefsJSON),
		Confidence:     row.Confidence,
		Adopted:        row.Adopted,
		DecisionStatus: firstNonEmpty(row.DecisionStatus, "pending"),
		DecisionReason: row.DecisionReason,
		DecidedBy:      row.DecidedBy,
		LogDigest:      row.LogDigest,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
	}
	if row.DecidedAt != nil {
		out.DecidedAt = row.DecidedAt.Format(time.RFC3339)
	}
	return out
}

func DiagnoseLog(logText string) DiagnosisResponse {
	lines := strings.Split(logText, "\n")
	for _, rule := range diagnosisRules() {
		refs := matchingRefs(lines, rule.Patterns)
		if len(refs) > 0 {
			return DiagnosisResponse{
				Status:         "diagnosed",
				RootCause:      rule.Cause,
				FixSuggestions: append([]string{}, rule.Suggestions...),
				EvidenceRefs:   refs,
				Confidence:     rule.Confidence,
				LogDigest:      digestText(logText),
			}
		}
	}
	return DiagnosisResponse{
		Status:         "diagnosed",
		RootCause:      "unknown_ci_failure",
		FixSuggestions: []string{"查看失败 job 的完整日志，优先定位首个非零退出码、首个 FAIL 段或依赖安装错误。"},
		EvidenceRefs:   firstRefs(lines),
		Confidence:     0.35,
		LogDigest:      digestText(logText),
	}
}

func diagnosisRules() []PatternRule {
	return []PatternRule{
		{
			Cause:      "github_token_or_permission_failure",
			Confidence: 0.9,
			Suggestions: []string{
				"检查 repo connection 绑定的 secretId 是否仍为 active。",
				"确认 GitHub token 具备 actions:read、contents:read 权限，私有仓库还需要 repo 访问权。",
			},
			Patterns: rex(`(?i)bad credentials`, `(?i)resource not accessible`, `(?i)401 unauthorized`, `(?i)403 forbidden`, `(?i)permission denied`),
		},
		{
			Cause:      "docker_or_postgres_unavailable",
			Confidence: 0.88,
			Suggestions: []string{
				"确认 runner 可启动 Docker service，或将 Postgres e2e 放到支持 Docker 的 workflow。",
				"检查 ASH_DATABASE_URL、ASH_MIGRATE_POSTGRES_URL 与 Postgres healthcheck 输出。",
			},
			Patterns: rex(`(?i)cannot connect to the docker daemon`, `(?i)docker: command not found`, `(?i)postgres.*connection refused`, `(?i)database system is starting`),
		},
		{
			Cause:      "go_compile_failure",
			Confidence: 0.86,
			Suggestions: []string{
				"从首个 Go 编译错误开始修复，优先处理 undefined、类型不匹配和导入缺失。",
				"本地运行 go test ./... 复现，并确认生成文件没有漏提交。",
			},
			Patterns: rex(`(?m)^# .*`, `(?i)undefined:`, `(?i)cannot find package`, `(?i)expected .* before`, `(?i)build failed`),
		},
		{
			Cause:      "test_failure",
			Confidence: 0.84,
			Suggestions: []string{
				"定位第一个 --- FAIL 测试用例，确认是行为回归还是断言样本需要更新。",
				"本地使用相同 -run 过滤条件复现，并保留失败输出作为修复证据。",
			},
			Patterns: rex(`(?m)^--- FAIL:`, `(?m)^FAIL\s`, `(?i)t\.fatalf`, `(?i)panic:`),
		},
		{
			Cause:      "dependency_resolution_failure",
			Confidence: 0.8,
			Suggestions: []string{
				"检查 go.mod/go.sum 或前端 lockfile 是否与代码同步提交。",
				"确认 CI runner 可以访问依赖源；必要时配置 GOPROXY 或 npm registry。",
			},
			Patterns: rex(`(?i)checksum mismatch`, `(?i)missing go.sum entry`, `(?i)no required module provides package`, `(?i)npm ERR!`, `(?i)module lookup disabled`),
		},
		{
			Cause:      "timeout",
			Confidence: 0.78,
			Suggestions: []string{
				"检查测试是否等待外部服务或端口，必要时增加 readiness 轮询而不是固定 sleep。",
				"缩小 CI 任务范围，分离 nightly e2e 与 PR 快速门禁。",
			},
			Patterns: rex(`(?i)timed out`, `(?i)timeout exceeded`, `(?i)context deadline exceeded`, `(?i)deadline exceeded`),
		},
	}
}

func matchingRefs(lines []string, patterns []*regexp.Regexp) []string {
	refs := []string{}
	for i, line := range lines {
		for _, p := range patterns {
			if p.MatchString(line) {
				refs = append(refs, fmt.Sprintf("log:L%d:%s", i+1, trimLine(line)))
				break
			}
		}
		if len(refs) >= 5 {
			break
		}
	}
	return refs
}

func firstRefs(lines []string) []string {
	out := []string{}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, fmt.Sprintf("log:L%d:%s", i+1, trimLine(line)))
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func rex(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, regexp.MustCompile(p))
	}
	return out
}

func (s *Service) connection(spaceID, connectionID string) (store.RepoConnection, error) {
	var row store.RepoConnection
	err := s.q(nil).First(&row, "id = ? AND space_id = ?", connectionID, firstNonEmpty(spaceID, "local")).Error
	if err == gorm.ErrRecordNotFound {
		return row, fmt.Errorf("repo connection not found")
	}
	return row, err
}

func (s *Service) resolveSecret(spaceID, secretID string) (string, error) {
	if s.secretResolver == nil {
		return "", fmt.Errorf("secret resolver is not configured")
	}
	return s.secretResolver(spaceID, secretID)
}

func normalizeProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "github_actions" || provider == "github-actions" {
		return "github"
	}
	return provider
}

func digestText(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func trimLine(line string) string {
	line = strings.TrimSpace(line)
	if len(line) > 160 {
		return line[:160]
	}
	return line
}

func mustJSON(values []string) string {
	raw, _ := json.Marshal(values)
	return string(raw)
}

func parseJSONArray(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(firstNonEmpty(raw, "[]")), &values); err != nil {
		return []string{}
	}
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type GitHubProvider struct {
	Client *http.Client
}

func (p GitHubProvider) ListWorkflowRuns(ctx context.Context, conn store.RepoConnection, token string, limit int) ([]store.CIRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs?per_page=%d", url.PathEscape(conn.Owner), url.PathEscape(conn.Repo), limit)
	var payload struct {
		WorkflowRuns []struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			RunAttempt int    `json:"run_attempt"`
			HTMLURL    string `json:"html_url"`
			HeadBranch string `json:"head_branch"`
			HeadSHA    string `json:"head_sha"`
			CreatedAt  string `json:"created_at"`
			UpdatedAt  string `json:"updated_at"`
			RunStarted string `json:"run_started_at"`
		} `json:"workflow_runs"`
	}
	if err := p.getJSON(ctx, endpoint, token, &payload); err != nil {
		return nil, err
	}
	rows := make([]store.CIRun, 0, len(payload.WorkflowRuns))
	now := time.Now().UTC()
	for _, item := range payload.WorkflowRuns {
		started := parseGithubTime(firstNonEmpty(item.RunStarted, item.CreatedAt))
		completed := parseGithubTime(item.UpdatedAt)
		rows = append(rows, store.CIRun{
			ID:            "ci_run_" + uuid.NewString(),
			ProviderRunID: strconv.FormatInt(item.ID, 10),
			Workflow:      item.Name,
			Status:        item.Status,
			Conclusion:    item.Conclusion,
			Attempt:       item.RunAttempt,
			CommitSHA:     item.HeadSHA,
			Branch:        item.HeadBranch,
			RunURL:        item.HTMLURL,
			StartedAt:     started,
			CompletedAt:   completed,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return rows, nil
}

func (p GitHubProvider) GetRunJobs(ctx context.Context, conn store.RepoConnection, token, providerRunID string) ([]store.CIJob, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%s/jobs", url.PathEscape(conn.Owner), url.PathEscape(conn.Repo), url.PathEscape(providerRunID))
	var payload struct {
		Jobs []struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Status      string `json:"status"`
			Conclusion  string `json:"conclusion"`
			StartedAt   string `json:"started_at"`
			CompletedAt string `json:"completed_at"`
			HTMLURL     string `json:"html_url"`
		} `json:"jobs"`
	}
	if err := p.getJSON(ctx, endpoint, token, &payload); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	rows := make([]store.CIJob, 0, len(payload.Jobs))
	for _, item := range payload.Jobs {
		rows = append(rows, store.CIJob{
			ID:            "ci_job_" + uuid.NewString(),
			ProviderJobID: strconv.FormatInt(item.ID, 10),
			Name:          item.Name,
			Status:        item.Status,
			Conclusion:    item.Conclusion,
			StartedAt:     parseGithubTime(item.StartedAt),
			CompletedAt:   parseGithubTime(item.CompletedAt),
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return rows, nil
}

func (p GitHubProvider) GetJobLogs(ctx context.Context, conn store.RepoConnection, token, providerJobID string) (string, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/jobs/%s/logs", url.PathEscape(conn.Owner), url.PathEscape(conn.Repo), url.PathEscape(providerJobID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	addGitHubHeaders(req, token)
	client := p.client()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github logs returned %s", resp.Status)
	}
	raw, err := io.ReadAll(resp.Body)
	return string(raw), err
}

func (p GitHubProvider) getJSON(ctx context.Context, endpoint, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	addGitHubHeaders(req, token)
	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github API returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (p GitHubProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

func addGitHubHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
}

func parseGithubTime(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &t
}
