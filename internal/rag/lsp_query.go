package rag

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ash-repwiki/ash/internal/store"
)

var (
	ErrLSPUnavailable = errors.New("lsp: language server unavailable for path")
	ErrLSPUnsupported = errors.New("lsp: unsupported language")
)

// LSPPositionQuery is the shared body for hover / definition (RAG internal surface).
type LSPPositionQuery struct {
	RepoRoot  string `json:"repoRoot" binding:"required"`
	Path      string `json:"path" binding:"required"`
	Line      int    `json:"line" binding:"required"` // 1-based
	Character int    `json:"character"`               // 0-based
	SpaceID   string `json:"spaceId,omitempty"`
	Text      string `json:"text,omitempty"` // optional; otherwise read from disk
}

type LSPHoverResponse struct {
	Contents string         `json:"contents"`
	Kind     string         `json:"kind,omitempty"`
	Range    *LSPRangeView  `json:"range,omitempty"`
	Server   string         `json:"server"`
	Path     string         `json:"path"`
}

type LSPDefinitionResponse struct {
	Locations []LSPLocationView `json:"locations"`
	Server    string            `json:"server"`
	Path      string            `json:"path"`
}

type LSPReferencesRequest struct {
	LSPPositionQuery
	Limit int `json:"limit,omitempty"` // default 20, max 50
}

type LSPReferencesResponse struct {
	Locations []LSPLocationView `json:"locations"`
	Server    string            `json:"server,omitempty"`
	Path      string            `json:"path"`
	Source    string            `json:"source"` // lsp|symbol_table
	Truncated bool              `json:"truncated,omitempty"`
}

type LSPLocationView struct {
	Path      string `json:"path"`
	URI       string `json:"uri,omitempty"`
	Line      int    `json:"line"`      // 1-based
	Character int    `json:"character"` // 0-based
}

type LSPRangeView struct {
	StartLine      int `json:"startLine"`
	StartCharacter int `json:"startCharacter"`
	EndLine        int `json:"endLine"`
	EndCharacter   int `json:"endCharacter"`
}

// Hover resolves textDocument/hover via the workspace LSP session pool.
func (s *Service) Hover(req LSPPositionQuery) (*LSPHoverResponse, error) {
	prep, err := prepareLSPQuery(req)
	if err != nil {
		return nil, err
	}
	defer prep.close()

	ctx, cancel := context.WithTimeout(context.Background(), lspRequestTimeoutDuration())
	defer cancel()
	raw, err := prep.pool.hover(ctx, prep.rootAbs, prep.server, prep.fileAbs, prep.langID, prep.text, prep.line0, prep.character)
	if err != nil {
		return nil, err
	}
	out := &LSPHoverResponse{
		Server: filepath.Base(prep.server),
		Path:   prep.relPath,
	}
	if raw != nil {
		out.Contents = raw.Contents
		out.Kind = raw.Kind
		if raw.Range != nil {
			out.Range = &LSPRangeView{
				StartLine:      raw.Range.Start.Line + 1,
				StartCharacter: raw.Range.Start.Character,
				EndLine:        raw.Range.End.Line + 1,
				EndCharacter:   raw.Range.End.Character,
			}
		}
	}
	return out, nil
}

// Definition resolves textDocument/definition via the workspace LSP session pool.
func (s *Service) Definition(req LSPPositionQuery) (*LSPDefinitionResponse, error) {
	prep, err := prepareLSPQuery(req)
	if err != nil {
		return nil, err
	}
	defer prep.close()

	ctx, cancel := context.WithTimeout(context.Background(), lspRequestTimeoutDuration())
	defer cancel()
	locs, err := prep.pool.definition(ctx, prep.rootAbs, prep.server, prep.fileAbs, prep.langID, prep.text, prep.line0, prep.character)
	if err != nil {
		return nil, err
	}
	out := &LSPDefinitionResponse{
		Server:    filepath.Base(prep.server),
		Path:      prep.relPath,
		Locations: make([]LSPLocationView, 0, len(locs)),
	}
	for _, loc := range locs {
		view := LSPLocationView{
			URI:       loc.URI,
			Line:      loc.Line,
			Character: loc.Character,
		}
		if p, err := uriToPath(loc.URI); err == nil {
			if rel, err := filepath.Rel(prep.rootAbs, p); err == nil && !strings.HasPrefix(rel, "..") {
				view.Path = filepath.ToSlash(rel)
			} else {
				view.Path = filepath.ToSlash(p)
			}
		}
		out.Locations = append(out.Locations, view)
	}
	return out, nil
}

// References resolves textDocument/references (bounded). Falls back to symbol-table
// same-name rows when the language server is unavailable.
func (s *Service) References(req LSPReferencesRequest) (*LSPReferencesResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	prep, err := prepareLSPQuery(req.LSPPositionQuery)
	if err != nil {
		if errors.Is(err, ErrLSPUnavailable) || errors.Is(err, ErrLSPUnsupported) {
			return s.referencesFromSymbolTable(req, limit, err)
		}
		return nil, err
	}
	defer prep.close()

	ctx, cancel := context.WithTimeout(context.Background(), lspRequestTimeoutDuration())
	defer cancel()
	locs, err := prep.pool.references(ctx, prep.rootAbs, prep.server, prep.fileAbs, prep.langID, prep.text, prep.line0, prep.character, limit+1)
	if err != nil {
		return s.referencesFromSymbolTable(req, limit, err)
	}
	truncated := len(locs) > limit
	if truncated {
		locs = locs[:limit]
	}
	out := &LSPReferencesResponse{
		Server:    filepath.Base(prep.server),
		Path:      prep.relPath,
		Source:    "lsp",
		Truncated: truncated,
		Locations: make([]LSPLocationView, 0, len(locs)),
	}
	for _, loc := range locs {
		out.Locations = append(out.Locations, locationViewFromResult(prep.rootAbs, loc))
	}
	return out, nil
}

func locationViewFromResult(rootAbs string, loc lspLocationResult) LSPLocationView {
	view := LSPLocationView{
		URI:       loc.URI,
		Line:      loc.Line,
		Character: loc.Character,
	}
	if p, err := uriToPath(loc.URI); err == nil {
		if rel, err := filepath.Rel(rootAbs, p); err == nil && !strings.HasPrefix(rel, "..") {
			view.Path = filepath.ToSlash(rel)
		} else {
			view.Path = filepath.ToSlash(p)
		}
	}
	return view
}

func (s *Service) referencesFromSymbolTable(req LSPReferencesRequest, limit int, cause error) (*LSPReferencesResponse, error) {
	rootAbs, err := AbsRepoRoot(req.RepoRoot)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(req.Path)
	var fileAbs string
	if filepath.IsAbs(path) {
		fileAbs = filepath.Clean(path)
	} else {
		fileAbs = filepath.Join(rootAbs, filepath.FromSlash(path))
	}
	rel, relErr := filepath.Rel(rootAbs, fileAbs)
	if relErr != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	symbol := ""
	space := firstNonEmpty(req.SpaceID, "local")
	if gdb := s.gdb(); gdb != nil && rel != "" {
		var row store.RAGSymbol
		if err := gdb.Where("space_id = ? AND repo_root = ? AND path = ? AND line = ?", space, rootAbs, rel, req.Line).
			First(&row).Error; err == nil {
			symbol = row.Name
		}
	}
	if symbol == "" {
		return nil, fmt.Errorf("%w: %v", ErrLSPUnavailable, cause)
	}
	hits := s.symbolTableRefs(space, rootAbs, symbol, limit+1)
	truncated := len(hits) > limit
	if truncated {
		hits = hits[:limit]
	}
	out := &LSPReferencesResponse{
		Path:      rel,
		Source:    "symbol_table",
		Truncated: truncated,
		Locations: make([]LSPLocationView, 0, len(hits)),
	}
	for _, h := range hits {
		out.Locations = append(out.Locations, LSPLocationView{
			Path: h.Path,
			Line: h.StartLine,
		})
	}
	return out, nil
}

func (s *Service) symbolTableRefs(space, repoRoot, symbol string, limit int) []Hit {
	if s.gdb() == nil || strings.TrimSpace(symbol) == "" {
		return nil
	}
	q := s.gdb().Where("space_id = ? AND LOWER(name) = ?", space, strings.ToLower(symbol))
	if repoRoot != "" {
		q = q.Where("repo_root = ?", repoRoot)
	}
	var rows []store.RAGSymbol
	if err := q.Order("path ASC, line ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil
	}
	hits := make([]Hit, 0, len(rows))
	for _, row := range rows {
		hits = append(hits, Hit{
			Ref: makeRef(row.Path, row.Name, row.Line, row.Line), Path: row.Path,
			Symbol: row.Name, StartLine: row.Line, EndLine: row.Line,
			Digest: row.Digest, Snippet: "ref " + row.Name,
		})
	}
	return hits
}

type lspQueryPrep struct {
	rootAbs   string
	fileAbs   string
	relPath   string
	server    string
	langID    string
	text      string
	line0     int
	character int
	pool      *lspSessionPool
	ownPool   bool
}

func (p *lspQueryPrep) close() {
	if p == nil || p.pool == nil {
		return
	}
	// Keep shared sessions warm; only tear down test/private pools via indexer Close.
	if p.ownPool {
		p.pool.closeRoot(p.rootAbs)
	}
}

func prepareLSPQuery(req LSPPositionQuery) (*lspQueryPrep, error) {
	if req.Line < 1 {
		return nil, fmt.Errorf("line must be >= 1")
	}
	if req.Character < 0 {
		return nil, fmt.Errorf("character must be >= 0")
	}
	rootAbs, err := AbsRepoRoot(req.RepoRoot)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	var fileAbs string
	if filepath.IsAbs(path) {
		fileAbs = filepath.Clean(path)
	} else {
		fileAbs = filepath.Join(rootAbs, filepath.FromSlash(path))
	}
	fileAbs, err = filepath.Abs(fileAbs)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(rootAbs, fileAbs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("path escapes repoRoot")
	}
	rel = filepath.ToSlash(rel)

	text := req.Text
	if text == "" {
		b, err := os.ReadFile(fileAbs)
		if err != nil {
			return nil, err
		}
		text = string(b)
	}

	idx := NewLSPIndexerFromEnv()
	server, langID, err := idx.resolveServer(strings.ToLower(filepath.Ext(fileAbs)))
	if err != nil {
		if strings.Contains(err.Error(), "unsupported") {
			return nil, fmt.Errorf("%w: %v", ErrLSPUnsupported, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrLSPUnavailable, err)
	}
	if server == "" {
		return nil, ErrLSPUnavailable
	}
	if _, err := os.Stat(server); err != nil {
		return nil, fmt.Errorf("%w: binary %s: %v", ErrLSPUnavailable, filepath.Base(server), err)
	}

	return &lspQueryPrep{
		rootAbs:   rootAbs,
		fileAbs:   fileAbs,
		relPath:   rel,
		server:    server,
		langID:    langID,
		text:      text,
		line0:     req.Line - 1,
		character: req.Character,
		pool:      sharedLSPSessionPool,
	}, nil
}

func uriToPath(uri string) (string, error) {
	uri = strings.TrimSpace(uri)
	if !strings.HasPrefix(uri, "file://") {
		return "", fmt.Errorf("not a file uri")
	}
	path := strings.TrimPrefix(uri, "file://")
	if strings.HasPrefix(path, "/") && len(path) >= 3 && path[2] == ':' {
		// file:///C:/... → /C:/... → C:/...
		path = path[1:]
	}
	return filepath.FromSlash(path), nil
}
