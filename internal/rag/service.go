package rag

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ash-repwiki/ash/internal/store"
)

type Service struct {
	db *store.DB
}

func NewService(db *store.DB) *Service {
	return &Service{db: db}
}

type IndexRequest struct {
	RepoRoot string `json:"repoRoot" binding:"required"`
	SpaceID  string `json:"spaceId,omitempty"`
}

type IndexResponse struct {
	Documents int `json:"documents"`
	Chunks    int `json:"chunks"`
}

type QueryRequest struct {
	RepoRoot string `json:"repoRoot,omitempty"`
	Text     string `json:"text" binding:"required"`
	TopK     int    `json:"topK,omitempty"`
	SpaceID  string `json:"spaceId,omitempty"`
}

type Hit struct {
	Ref       string  `json:"ref"`
	Path      string  `json:"path"`
	Symbol    string  `json:"symbol,omitempty"`
	StartLine int     `json:"startLine"`
	EndLine   int     `json:"endLine"`
	Digest    string  `json:"digest"`
	Score     float64 `json:"score"`
	Snippet   string  `json:"snippet"`
}

type QueryResponse struct {
	Items []Hit `json:"items"`
}

// AbsRepoRoot returns the canonical absolute path used when persisting RAG rows.
func AbsRepoRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("repoRoot is required")
	}
	return filepath.Abs(root)
}

func (s *Service) Index(req IndexRequest) (*IndexResponse, error) {
	abs, err := AbsRepoRoot(req.RepoRoot)
	if err != nil {
		return nil, err
	}
	space := firstNonEmpty(req.SpaceID, "local")

	var docs, chunks int
	err = filepath.WalkDir(abs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isIndexable(path) {
			return nil
		}
		docChunks, err := s.indexFile(space, abs, path)
		if err != nil {
			return nil
		}
		docs++
		chunks += docChunks
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &IndexResponse{Documents: docs, Chunks: chunks}, nil
}

func (s *Service) Query(req QueryRequest) (*QueryResponse, error) {
	text := strings.TrimSpace(strings.ToLower(req.Text))
	if text == "" {
		return nil, fmt.Errorf("text is required")
	}
	topK := req.TopK
	if topK <= 0 || topK > 50 {
		topK = 8
	}
	space := firstNonEmpty(req.SpaceID, "local")

	q := s.db.Where("space_id = ?", space)
	if req.RepoRoot != "" {
		if abs, err := AbsRepoRoot(req.RepoRoot); err == nil {
			q = q.Where("repo_root = ?", abs)
		}
	}
	var rows []store.RAGChunk
	terms := queryTerms(text)
	if len(terms) == 0 {
		return &QueryResponse{Items: []Hit{}}, nil
	}
	if hits, ok := s.queryFTS(req, terms, topK, space); ok {
		return &QueryResponse{Items: hits}, nil
	}
	likeQ := q
	for _, term := range terms {
		likeQ = likeQ.Where("LOWER(text) LIKE ? OR LOWER(path) LIKE ? OR LOWER(symbol) LIKE ?", "%"+term+"%", "%"+term+"%", "%"+term+"%")
	}
	if err := likeQ.Limit(topK * 4).Find(&rows).Error; err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(rows))
	for _, row := range rows {
		score := scoreChunk(row, terms)
		hits = append(hits, hitFromChunk(row, score))
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return &QueryResponse{Items: hits}, nil
}

func (s *Service) indexFile(space, root, path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil || info.Size() > 512*1024 {
		return 0, nil
	}
	b, err := os.ReadFile(path)
	if err != nil || looksBinary(b) {
		return 0, nil
	}
	rel, _ := filepath.Rel(root, path)
	rel = filepath.ToSlash(rel)
	digest := digestBytes(b)
	now := time.Now().UTC()
	ftsAvailable := s.ensureFTS() == nil

	var doc store.RAGDocument
	err = s.db.Transaction(func(tx *gorm.DB) error {
		_ = tx.Where("space_id = ? AND repo_root = ? AND path = ?", space, root, rel).Delete(&store.RAGDocument{}).Error
		_ = tx.Where("space_id = ? AND repo_root = ? AND path = ?", space, root, rel).Delete(&store.RAGChunk{}).Error
		if ftsAvailable {
			if err := tx.Exec("DELETE FROM rag_chunks_fts WHERE space_id = ? AND repo_root = ? AND path = ?", space, root, rel).Error; err != nil {
				return err
			}
		}
		doc = store.RAGDocument{
			ID: "ragdoc_" + uuid.NewString(), SpaceID: space, RepoRoot: root, Path: rel,
			Digest: digest, SizeBytes: info.Size(), CreatedAt: now, UpdatedAt: now,
		}
		return tx.Create(&doc).Error
	})
	if err != nil {
		return 0, err
	}

	lines := splitLines(string(b))
	chunkCount := 0
	for start := 0; start < len(lines); start += 40 {
		end := start + 40
		if end > len(lines) {
			end = len(lines)
		}
		text := strings.Join(lines[start:end], "\n")
		if strings.TrimSpace(text) == "" {
			continue
		}
		symbol := chunkSymbol(lines, start, end)
		chunk := store.RAGChunk{
			ID: "ragchk_" + uuid.NewString(), DocumentID: doc.ID, SpaceID: space,
			RepoRoot: root, Path: rel, Symbol: symbol, StartLine: start + 1, EndLine: end,
			Text: text, Digest: digestString(fmt.Sprintf("%s:%d:%s", digest, start+1, text)),
			CreatedAt: now,
		}
		if err := s.db.Create(&chunk).Error; err != nil {
			return chunkCount, err
		}
		if ftsAvailable {
			if err := s.insertFTS(chunk); err != nil {
				return chunkCount, err
			}
		}
		chunkCount++
	}
	return chunkCount, nil
}

func (s *Service) ensureFTS() error {
	if s.db.Dialect() != "sqlite" {
		return fmt.Errorf("fts is only available for sqlite")
	}
	db := s.db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	return db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS rag_chunks_fts USING fts5(
		chunk_id UNINDEXED,
		space_id UNINDEXED,
		repo_root UNINDEXED,
		path,
		symbol,
		text,
		digest UNINDEXED,
		start_line UNINDEXED,
		end_line UNINDEXED
	)`).Error
}

func (s *Service) insertFTS(chunk store.RAGChunk) error {
	return s.db.Exec(`INSERT INTO rag_chunks_fts
		(chunk_id, space_id, repo_root, path, symbol, text, digest, start_line, end_line)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chunk.ID, chunk.SpaceID, chunk.RepoRoot, chunk.Path, chunk.Symbol, chunk.Text,
		chunk.Digest, chunk.StartLine, chunk.EndLine,
	).Error
}

func (s *Service) queryFTS(req QueryRequest, terms []string, topK int, space string) ([]Hit, bool) {
	if err := s.ensureFTS(); err != nil {
		return nil, false
	}
	match := ftsMatch(terms)
	if match == "" {
		return []Hit{}, true
	}
	args := []any{match, space}
	where := "rag_chunks_fts MATCH ? AND space_id = ?"
	if req.RepoRoot != "" {
		abs, err := filepath.Abs(req.RepoRoot)
		if err == nil {
			where += " AND repo_root = ?"
			args = append(args, abs)
		}
	}
	args = append(args, topK)

	var rows []struct {
		ChunkID   string  `gorm:"column:chunk_id"`
		Path      string  `gorm:"column:path"`
		Symbol    string  `gorm:"column:symbol"`
		StartLine int     `gorm:"column:start_line"`
		EndLine   int     `gorm:"column:end_line"`
		Digest    string  `gorm:"column:digest"`
		Text      string  `gorm:"column:text"`
		Rank      float64 `gorm:"column:rank"`
	}
	err := s.db.Raw(`SELECT chunk_id, path, symbol, start_line, end_line, digest, text, bm25(rag_chunks_fts) AS rank
		FROM rag_chunks_fts
		WHERE `+where+`
		ORDER BY rank ASC
		LIMIT ?`, args...).Scan(&rows).Error
	if err != nil {
		return nil, false
	}
	if len(rows) == 0 {
		return nil, false
	}
	hits := make([]Hit, 0, len(rows))
	for _, row := range rows {
		score := -row.Rank
		if score == 0 {
			score = 1
		}
		hits = append(hits, Hit{
			Ref:  makeRef(row.Path, row.Symbol, row.StartLine, row.EndLine),
			Path: row.Path, Symbol: row.Symbol, StartLine: row.StartLine, EndLine: row.EndLine,
			Digest: row.Digest, Score: score, Snippet: trimSnippet(row.Text),
		})
	}
	return hits, true
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "build", "vendor", ".cache", ".next", "tmp":
		return true
	default:
		return strings.HasPrefix(name, ".") && name != ".github"
	}
}

func isIndexable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".md", ".yaml", ".yml", ".json", ".sql", ".txt", ".css", ".html":
		return true
	default:
		return false
	}
}

func looksBinary(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 1024), 1024*1024)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if len(lines) == 0 {
		return []string{s}
	}
	return lines
}

func queryTerms(text string) []string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return !(r == '_' || r == '-' || r == '/' || r == '.' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	seen := map[string]bool{}
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if len(p) < 2 || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func ftsMatch(terms []string) string {
	quoted := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		quoted = append(quoted, `"`+strings.ReplaceAll(term, `"`, `""`)+`"`)
	}
	return strings.Join(quoted, " OR ")
}

func hitFromChunk(row store.RAGChunk, score float64) Hit {
	return Hit{
		Ref:  makeRef(row.Path, row.Symbol, row.StartLine, row.EndLine),
		Path: row.Path, Symbol: row.Symbol, StartLine: row.StartLine, EndLine: row.EndLine,
		Digest: row.Digest, Score: score, Snippet: trimSnippet(row.Text),
	}
}

func makeRef(path, symbol string, startLine, endLine int) string {
	if symbol != "" {
		return fmt.Sprintf("%s#%s:%d-%d", path, symbol, startLine, endLine)
	}
	return fmt.Sprintf("%s:%d-%d", path, startLine, endLine)
}

func scoreChunk(row store.RAGChunk, terms []string) float64 {
	text := strings.ToLower(row.Path + "\n" + row.Symbol + "\n" + row.Text)
	var score float64
	for _, term := range terms {
		score += float64(strings.Count(text, term))
		if strings.Contains(strings.ToLower(row.Path), term) {
			score += 2
		}
		if strings.Contains(strings.ToLower(row.Symbol), term) {
			score += 3
		}
	}
	return score
}

var symbolPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s*func\s+(?:\([^)]+\)\s*)?([A-Za-z_][A-Za-z0-9_]*)`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:class|interface|type|struct)\s+([A-Za-z_][A-Za-z0-9_]*)`),
}

func chunkSymbol(lines []string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	for i := start; i < end; i++ {
		if symbol := lineSymbol(lines[i]); symbol != "" {
			return symbol
		}
	}
	lookback := start - 20
	if lookback < 0 {
		lookback = 0
	}
	for i := start - 1; i >= lookback; i-- {
		if symbol := lineSymbol(lines[i]); symbol != "" {
			return symbol
		}
	}
	return ""
}

func lineSymbol(line string) string {
	for _, pattern := range symbolPatterns {
		if m := pattern.FindStringSubmatch(line); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func trimSnippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 500 {
		return s
	}
	return s[:500] + "\n..."
}

func digestBytes(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

func digestString(s string) string {
	return digestBytes([]byte(s))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
