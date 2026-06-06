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
	LogDigest      string   `json:"logDigest,omitempty"`
	CreatedAt      string   `json:"createdAt"`
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
	if err := s.db.Create(&row).Error; err != nil {
		return store.RepoConnection{}, err
	}
	return row, nil
}

func (s *Service) ListConnections(spaceID string) ([]store.RepoConnection, error) {
	var rows []store.RepoConnection
	err := s.db.Where("space_id = ?", firstNonEmpty(spaceID, "local")).Order("created_at desc").Find(&rows).Error
	return rows, err
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
	q := s.db.Where("space_id = ?", spaceID)
	if strings.TrimSpace(connectionID) != "" {
		q = q.Where("connection_id = ?", strings.TrimSpace(connectionID))
	}
	var rows []store.CIRun
	err := q.Order("created_at desc").Limit(limit).Find(&rows).Error
	return rows, err
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
		if err := s.db.Where("connection_id = ? AND provider_run_id = ?", conn.ID, row.ProviderRunID).
			Assign(row).FirstOrCreate(&store.CIRun{}).Error; err != nil {
			return err
		}
	}
	conn.LastSyncAt = &now
	conn.UpdatedAt = now
	return s.db.Save(&conn).Error
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
		if err := s.db.First(&run, "id = ? AND space_id = ?", runID, spaceID).Error; err != nil {
			return DiagnosisResponse{}, fmt.Errorf("ci run not found: %w", err)
		}
		if connID == "" {
			connID = run.ConnectionID
		}
	}
	var job store.CIJob
	if jobID != "" {
		if err := s.db.First(&job, "id = ? AND space_id = ?", jobID, spaceID).Error; err != nil {
			return DiagnosisResponse{}, fmt.Errorf("ci job not found: %w", err)
		}
		if connID == "" {
			connID = job.ConnectionID
		}
		if runID == "" {
			runID = job.CIRunID
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
		LogDigest:          out.LogDigest,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return DiagnosisResponse{}, err
	}
	out.ID = row.ID
	out.SpaceID = row.SpaceID
	out.ConnectionID = row.ConnectionID
	out.RunID = row.CIRunID
	out.JobID = row.CIJobID
	out.CreatedAt = row.CreatedAt.Format(time.RFC3339)
	return out, nil
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
	err := s.db.First(&row, "id = ? AND space_id = ?", connectionID, firstNonEmpty(spaceID, "local")).Error
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
