package knowledge

import (
	"fmt"
	"strings"

	"github.com/ash-repwiki/ash/internal/memory"
	"github.com/ash-repwiki/ash/internal/store"
)

// WikiPage is a read-only projection over approved memory (and optional notes).
type WikiPage struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Layer      string   `json:"layer,omitempty"`
	Source     string   `json:"source"` // memory|synthetic
	MemoryID   string   `json:"memoryId,omitempty"`
	ContextRef string   `json:"contextRef"`
	Tags       []string `json:"tags,omitempty"`
}

type WikiListResponse struct {
	Items []WikiPage `json:"items"`
	Query string     `json:"query,omitempty"`
}

// Service builds ephemeral knowledge views (no persistence).
type Service struct {
	db     *store.DB
	memory *memory.Service
}

func NewService(db *store.DB, mem *memory.Service) *Service {
	return &Service{db: db, memory: mem}
}

func (s *Service) Profile(repoRoot string) (*RepoProfile, error) {
	return BuildProfile(repoRoot)
}

func (s *Service) ListWikiPages(spaceID, repoRoot, query string, limit int) (*WikiListResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	spaceID = firstNonEmpty(spaceID, "local")
	q := strings.TrimSpace(query)
	if q == "" {
		q = "architecture overview testing deploy"
	}
	var items []WikiPage
	if strings.TrimSpace(repoRoot) != "" {
		if prof, err := BuildProfile(repoRoot); err == nil {
			items = append(items, WikiPage{
				ID: "wiki_profile_overview", Title: "Repo Profile Overview",
				Body: prof.Summary + "\n\nTest commands:\n- " + strings.Join(prof.TestCommands, "\n- "),
				Source: "synthetic", ContextRef: "wiki:wiki_profile_overview",
				Tags: prof.Languages,
			})
		}
	}
	if s.memory != nil {
		// Ephemeral wiki: project approved memory by query text (repoRoot only drives synthetic overview).
		req := memory.QueryRequest{Text: q, TopK: limit}
		resp, err := s.memory.QueryForSpace(spaceID, req)
		if err == nil && resp != nil {
			for _, it := range resp.Items {
				items = append(items, WikiPage{
					ID: "wiki_" + it.ID, Title: it.Title, Body: truncate(it.Body, 1200),
					Layer: it.Layer, Source: "memory", MemoryID: it.ID,
					ContextRef: "wiki:wiki_" + it.ID, Tags: it.Tags,
				})
			}
		}
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return &WikiListResponse{Items: items, Query: q}, nil
}

func (s *Service) GetWikiPage(spaceID, pageID, repoRoot string) (*WikiPage, error) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return nil, fmt.Errorf("pageId required")
	}
	if pageID == "wiki_profile_overview" {
		if strings.TrimSpace(repoRoot) != "" {
			if prof, err := BuildProfile(repoRoot); err == nil {
				return &WikiPage{
					ID: pageID, Title: "Repo Profile Overview", Source: "synthetic",
					Body: prof.Summary + "\n\nTest commands:\n- " + strings.Join(prof.TestCommands, "\n- "),
					ContextRef: "wiki:" + pageID, Tags: prof.Languages,
				}, nil
			}
		}
		return &WikiPage{
			ID: pageID, Title: "Repo Profile Overview", Source: "synthetic",
			Body: "Provide repoRoot to regenerate overview.", ContextRef: "wiki:" + pageID,
		}, nil
	}
	if s.memory == nil {
		return nil, fmt.Errorf("memory service unavailable")
	}
	memID := strings.TrimPrefix(pageID, "wiki_")
	rec, err := s.memory.GetForSpace(firstNonEmpty(spaceID, ""), memID)
	if err != nil {
		return nil, err
	}
	if rec.Status != "approved" {
		return nil, fmt.Errorf("wiki page source is not approved")
	}
	return &WikiPage{
		ID: "wiki_" + rec.ID, Title: rec.Title, Body: rec.Body, Layer: rec.Layer,
		Source: "memory", MemoryID: rec.ID, ContextRef: "wiki:wiki_" + rec.ID, Tags: rec.Tags,
	}, nil
}

// ContextRefsForRun returns profile + top wiki refs for injection.
func (s *Service) ContextRefsForRun(spaceID, repoRoot, issue string) []string {
	var refs []string
	seen := map[string]struct{}{}
	add := func(r string) {
		if r == "" {
			return
		}
		if _, ok := seen[r]; ok {
			return
		}
		seen[r] = struct{}{}
		refs = append(refs, r)
	}
	if strings.TrimSpace(repoRoot) != "" {
		if p, err := BuildProfile(repoRoot); err == nil {
			add(p.ContextRef)
		}
	}
	list, err := s.ListWikiPages(spaceID, repoRoot, issue, 5)
	if err == nil {
		for _, p := range list.Items {
			add(p.ContextRef)
		}
	}
	return refs
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
