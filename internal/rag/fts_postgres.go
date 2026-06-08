package rag

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (s *Service) ensurePostgresFTS() error {
	if s.db == nil || s.db.Dialect() != "postgres" {
		return fmt.Errorf("postgres fts requires postgres dialect")
	}
	var exists bool
	err := s.gdb().Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'rag_chunks'
			  AND column_name = 'search_vector'
		)`).Scan(&exists).Error
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("rag_chunks.search_vector missing; apply SQL revision 17")
	}
	return nil
}

func (s *Service) queryPostgresFTS(req QueryRequest, terms []string, topK int, space string) ([]Hit, bool) {
	if err := s.ensurePostgresFTS(); err != nil {
		return nil, false
	}
	queryText := strings.TrimSpace(strings.Join(terms, " "))
	if queryText == "" {
		return []Hit{}, true
	}
	where := "c.space_id = ? AND c.search_vector @@ plainto_tsquery('simple', ?)"
	args := []any{space, queryText}
	if req.RepoRoot != "" {
		if abs, err := filepath.Abs(req.RepoRoot); err == nil {
			where += " AND c.repo_root = ?"
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
	queryArgs := append([]any{queryText}, args...)
	err := s.gdb().Raw(`
		SELECT c.id AS chunk_id, c.path, c.symbol, c.start_line, c.end_line, c.digest, c.text,
			ts_rank(c.search_vector, plainto_tsquery('simple', ?)) AS rank
		FROM rag_chunks c
		WHERE `+where+`
		ORDER BY rank DESC
		LIMIT ?`, queryArgs...).Scan(&rows).Error
	if err != nil {
		return nil, false
	}
	if len(rows) == 0 {
		return nil, false
	}
	hits := make([]Hit, 0, len(rows))
	for _, row := range rows {
		score := row.Rank
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
