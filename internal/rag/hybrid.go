package rag

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ash-repwiki/ash/internal/store"
)

const rrfK = 60

func preferBoost(prefer, lane string) float64 {
	if prefer == "" || prefer == lane {
		if prefer == lane {
			return 2.0
		}
		return 1.0
	}
	return 1.0
}

func rrfMerge(lanes map[string][]Hit, prefer string, topK int) []Hit {
	scores := map[string]float64{}
	best := map[string]Hit{}
	for lane, hits := range lanes {
		boost := preferBoost(prefer, lane)
		for rank, h := range hits {
			key := h.Ref
			if key == "" {
				key = makeRef(h.Path, h.Symbol, h.StartLine, h.EndLine)
			}
			scores[key] += boost * (1.0 / float64(rrfK+rank+1))
			if prev, ok := best[key]; !ok || h.Score >= prev.Score {
				cp := h
				best[key] = cp
			}
		}
	}
	out := make([]Hit, 0, len(best))
	for key, h := range best {
		h.Score = scores[key]
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > topK {
		out = out[:topK]
	}
	return out
}

func validatePrefer(prefer string) error {
	switch prefer {
	case "", "path", "symbol", "text":
		return nil
	default:
		return fmt.Errorf("invalid prefer %q: must be empty or one of path, symbol, text", prefer)
	}
}

func (s *Service) hybridCounts(space, repoRoot string) (pathCount, symbolCount int64) {
	gdb := s.gdb()
	if gdb == nil {
		return 0, 0
	}
	q := gdb.Where("space_id = ?", space)
	if repoRoot != "" {
		q = q.Where("repo_root = ?", repoRoot)
	}
	_ = q.Model(&store.RAGPathEntry{}).Count(&pathCount).Error
	q2 := gdb.Where("space_id = ?", space)
	if repoRoot != "" {
		q2 = q2.Where("repo_root = ?", repoRoot)
	}
	_ = q2.Model(&store.RAGSymbol{}).Count(&symbolCount).Error
	return pathCount, symbolCount
}

func (s *Service) queryTextLane(req QueryRequest, terms []string, topK int, space string) ([]Hit, string, error) {
	if hits, ok := s.queryFTS(req, terms, topK, space); ok {
		return hits, RetrievalModeFTS, nil
	}
	q := s.gdb().Where("space_id = ?", space)
	if req.RepoRoot != "" {
		if abs, err := AbsRepoRoot(req.RepoRoot); err == nil {
			q = q.Where("repo_root = ?", abs)
		}
	}
	for _, term := range terms {
		q = q.Where("LOWER(text) LIKE ? OR LOWER(path) LIKE ? OR LOWER(symbol) LIKE ?",
			"%"+term+"%", "%"+term+"%", "%"+term+"%")
	}
	var rows []store.RAGChunk
	if err := q.Limit(topK * 4).Find(&rows).Error; err != nil {
		return nil, RetrievalModeChunk, err
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
	return hits, RetrievalModeChunk, nil
}

func (s *Service) queryPathLane(space, repoRoot string, terms []string, limit int) []Hit {
	q := s.gdb().Where("space_id = ?", space)
	if repoRoot != "" {
		q = q.Where("repo_root = ?", repoRoot)
	}
	for _, term := range terms {
		q = q.Where("LOWER(basename) = ? OR LOWER(path) LIKE ?", term, "%"+term+"%")
	}
	var rows []store.RAGPathEntry
	if err := q.Limit(limit).Find(&rows).Error; err != nil {
		return nil
	}
	hits := make([]Hit, 0, len(rows))
	for i, row := range rows {
		score := float64(len(rows) - i)
		for _, term := range terms {
			if strings.EqualFold(row.Basename, term) {
				score += 10
				break
			}
		}
		hits = append(hits, Hit{
			Ref: makeRef(row.Path, "", 0, 0), Path: row.Path,
			Digest: row.Digest, Score: score, Snippet: row.Path,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	return hits
}

func (s *Service) querySymbolLane(space, repoRoot string, terms []string, limit int) []Hit {
	q := s.gdb().Where("space_id = ?", space)
	if repoRoot != "" {
		q = q.Where("repo_root = ?", repoRoot)
	}
	for _, term := range terms {
		q = q.Where("LOWER(name) = ? OR LOWER(name) LIKE ?", term, "%"+term+"%")
	}
	var rows []store.RAGSymbol
	if err := q.Order("line ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil
	}
	hits := make([]Hit, 0, len(rows))
	for i, row := range rows {
		score := float64(len(rows) - i)
		if len(terms) > 0 && strings.EqualFold(row.Name, terms[0]) {
			score += 10
		}
		snippet := strings.TrimSpace(row.Kind + " " + row.Name)
		hits = append(hits, Hit{
			Ref: makeRef(row.Path, row.Name, row.Line, row.Line), Path: row.Path,
			Symbol: row.Name, StartLine: row.Line, EndLine: row.Line,
			Digest: row.Digest, Score: score, Snippet: snippet,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	return hits
}
