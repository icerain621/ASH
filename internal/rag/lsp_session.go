package rag

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	envRAGLSPSession     = "ASH_RAG_LSP_SESSION"
	envRAGLSPIdleSec     = "ASH_RAG_LSP_IDLE_SEC"
	envRAGLSPMaxOpenDocs = "ASH_RAG_LSP_MAX_OPEN_DOCS"
	defaultLSPIdle       = 30 * time.Second
	defaultLSPMaxOpenDocs = 64
)

var sharedLSPSessionPool = newLSPSessionPool()

type lspSessionPool struct {
	mu       sync.Mutex
	sessions map[string]*lspSession
}

func newLSPSessionPool() *lspSessionPool {
	return &lspSessionPool{sessions: make(map[string]*lspSession)}
}

func lspSessionEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envRAGLSPSession))) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func lspIdleDuration() time.Duration {
	v := strings.TrimSpace(os.Getenv(envRAGLSPIdleSec))
	if v == "" {
		return defaultLSPIdle
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultLSPIdle
	}
	return time.Duration(n) * time.Second
}

func lspMaxOpenDocs() int {
	v := strings.TrimSpace(os.Getenv(envRAGLSPMaxOpenDocs))
	if v == "" {
		return defaultLSPMaxOpenDocs
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 4 {
		return defaultLSPMaxOpenDocs
	}
	if n > 512 {
		return 512
	}
	return n
}

func lspSessionKey(rootAbs, server string) string {
	return filepath.Clean(rootAbs) + "|" + server
}

func (p *lspSessionPool) documentSymbol(ctx context.Context, rootAbs, server, absPath, langID, text string) ([]SymbolHit, error) {
	var hits []SymbolHit
	err := p.withSession(ctx, rootAbs, server, func(s *lspSession) error {
		var err error
		hits, err = s.documentSymbolLocked(ctx, absPath, langID, text)
		return err
	})
	return hits, err
}

func (p *lspSessionPool) hover(ctx context.Context, rootAbs, server, absPath, langID, text string, line, character int) (*lspHoverResult, error) {
	var out *lspHoverResult
	err := p.withSession(ctx, rootAbs, server, func(s *lspSession) error {
		var err error
		out, err = s.hoverLocked(ctx, absPath, langID, text, line, character)
		return err
	})
	return out, err
}

func (p *lspSessionPool) definition(ctx context.Context, rootAbs, server, absPath, langID, text string, line, character int) ([]lspLocationResult, error) {
	var out []lspLocationResult
	err := p.withSession(ctx, rootAbs, server, func(s *lspSession) error {
		var err error
		out, err = s.definitionLocked(ctx, absPath, langID, text, line, character)
		return err
	})
	return out, err
}

func (p *lspSessionPool) references(ctx context.Context, rootAbs, server, absPath, langID, text string, line, character, limit int) ([]lspLocationResult, error) {
	var out []lspLocationResult
	err := p.withSession(ctx, rootAbs, server, func(s *lspSession) error {
		var err error
		out, err = s.referencesLocked(ctx, absPath, langID, text, line, character, limit)
		return err
	})
	return out, err
}

func (p *lspSessionPool) withSession(ctx context.Context, rootAbs, server string, fn func(*lspSession) error) error {
	if p == nil {
		return fmt.Errorf("lsp session pool is nil")
	}
	rootAbs, err := filepath.Abs(rootAbs)
	if err != nil {
		return err
	}
	s, err := p.getOrStart(ctx, rootAbs, server)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dead {
		return fmt.Errorf("lsp session dead")
	}
	if err := fn(s); err != nil {
		s.closeLocked()
		p.mu.Lock()
		if cur := p.sessions[s.key]; cur == s {
			delete(p.sessions, s.key)
		}
		p.mu.Unlock()
		return err
	}
	return nil
}

func (p *lspSessionPool) getOrStart(ctx context.Context, rootAbs, server string) (*lspSession, error) {
	key := lspSessionKey(rootAbs, server)
	p.mu.Lock()
	p.reclaimExpiredLocked()
	if s := p.sessions[key]; s != nil && !s.dead {
		p.mu.Unlock()
		return s, nil
	}
	p.mu.Unlock()

	s, err := startLSPSession(ctx, key, rootAbs, server)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.reclaimExpiredLocked()
	if existing := p.sessions[key]; existing != nil && !existing.dead {
		go s.forceKill()
		return existing, nil
	}
	p.sessions[key] = s
	return s, nil
}

func (p *lspSessionPool) reclaimExpiredLocked() {
	idle := lspIdleDuration()
	now := time.Now()
	for key, s := range p.sessions {
		if s == nil {
			delete(p.sessions, key)
			continue
		}
		s.mu.Lock()
		expired := s.dead || now.Sub(s.lastUsed) >= idle
		if expired && !s.dead {
			s.closeLocked()
		}
		dead := s.dead
		s.mu.Unlock()
		if dead {
			delete(p.sessions, key)
		}
	}
}

func (p *lspSessionPool) closeRoot(rootAbs string) {
	if p == nil || strings.TrimSpace(rootAbs) == "" {
		return
	}
	rootAbs, err := filepath.Abs(rootAbs)
	if err != nil {
		return
	}
	rootAbs = filepath.Clean(rootAbs)
	prefix := rootAbs + "|"
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, s := range p.sessions {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if s == nil {
			delete(p.sessions, key)
			continue
		}
		s.mu.Lock()
		s.closeLocked()
		s.mu.Unlock()
		delete(p.sessions, key)
	}
}

func (p *lspSessionPool) closeAll() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, s := range p.sessions {
		if s != nil {
			s.mu.Lock()
			s.closeLocked()
			s.mu.Unlock()
		}
		delete(p.sessions, key)
	}
}

type lspSession struct {
	key      string
	rootAbs  string
	server   string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	client   *lspClient
	stderr   *bytes.Buffer
	opened   map[string]int
	lastUsed time.Time
	mu       sync.Mutex
	dead     bool
}

func startLSPSession(ctx context.Context, key, rootAbs, server string) (*lspSession, error) {
	args := []string{}
	base := strings.ToLower(filepath.Base(server))
	if strings.Contains(base, "typescript-language-server") {
		args = append(args, "--stdio")
	}
	cmd := exec.CommandContext(context.WithoutCancel(ctx), server, args...)
	cmd.Dir = rootAbs
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("lsp start %s: %w", filepath.Base(server), err)
	}

	s := &lspSession{
		key:      key,
		rootAbs:  rootAbs,
		server:   server,
		cmd:      cmd,
		stdin:    stdin,
		client:   &lspClient{w: stdin, r: bufio.NewReader(stdout)},
		stderr:   &stderr,
		opened:   map[string]int{},
		lastUsed: time.Now(),
	}

	rootURI := pathToURI(rootAbs)
	initCtx, cancel := context.WithTimeout(ctx, lspRequestTimeoutDuration())
	defer cancel()
	if _, err := s.client.call(initCtx, "initialize", map[string]any{
		"processId": os.Getpid(),
		"rootUri":   rootURI,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"documentSymbol": map[string]any{
					"hierarchicalDocumentSymbolSupport": true,
				},
				"hover": map[string]any{
					"contentFormat": []string{"markdown", "plaintext"},
				},
				"definition": map[string]any{},
				"references": map[string]any{},
			},
		},
		"workspaceFolders": []map[string]any{{
			"uri":  rootURI,
			"name": filepath.Base(rootAbs),
		}},
	}); err != nil {
		s.forceKill()
		return nil, fmt.Errorf("lsp initialize: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	if err := s.client.notify("initialized", map[string]any{}); err != nil {
		s.forceKill()
		return nil, err
	}
	return s, nil
}

func (s *lspSession) ensureOpenLocked(absPath, langID, text string) (string, error) {
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", err
	}
	docURI := pathToURI(absPath)
	if _, ok := s.opened[docURI]; !ok {
		for len(s.opened) >= lspMaxOpenDocs() {
			if !s.evictOneOpenedLocked(docURI) {
				break
			}
		}
	}
	if ver, ok := s.opened[docURI]; ok {
		ver++
		s.opened[docURI] = ver
		if err := s.client.notify("textDocument/didChange", map[string]any{
			"textDocument": map[string]any{
				"uri":     docURI,
				"version": ver,
			},
			"contentChanges": []map[string]any{{"text": text}},
		}); err != nil {
			return "", err
		}
		return docURI, nil
	}
	s.opened[docURI] = 1
	if err := s.client.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        docURI,
			"languageId": langID,
			"version":    1,
			"text":       text,
		},
	}); err != nil {
		return "", err
	}
	return docURI, nil
}

func (s *lspSession) evictOneOpenedLocked(keepURI string) bool {
	for uri := range s.opened {
		if uri == keepURI {
			continue
		}
		_ = s.client.notify("textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		})
		delete(s.opened, uri)
		return true
	}
	return false
}

func (s *lspSession) documentSymbolLocked(ctx context.Context, absPath, langID, text string) ([]SymbolHit, error) {
	docURI, err := s.ensureOpenLocked(absPath, langID, text)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, lspRequestTimeoutDuration())
	defer cancel()
	raw, err := s.client.call(callCtx, "textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": docURI},
	})
	if err != nil {
		return nil, fmt.Errorf("lsp documentSymbol: %w (%s)", err, strings.TrimSpace(s.stderr.String()))
	}
	hits, err := parseDocumentSymbolResult(raw)
	if err != nil {
		return nil, err
	}
	s.lastUsed = time.Now()
	return hits, nil
}

type lspHoverResult struct {
	Contents string
	Kind     string
	Range    *lspRange
}

type lspLocationResult struct {
	URI       string
	Line      int // 1-based
	Character int // 0-based
}

func (s *lspSession) hoverLocked(ctx context.Context, absPath, langID, text string, line, character int) (*lspHoverResult, error) {
	docURI, err := s.ensureOpenLocked(absPath, langID, text)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, lspRequestTimeoutDuration())
	defer cancel()
	raw, err := s.client.call(callCtx, "textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": docURI},
		"position":     map[string]int{"line": line, "character": character},
	})
	if err != nil {
		return nil, fmt.Errorf("lsp hover: %w (%s)", err, strings.TrimSpace(s.stderr.String()))
	}
	out, err := parseHoverResult(raw)
	if err != nil {
		return nil, err
	}
	s.lastUsed = time.Now()
	return out, nil
}

func (s *lspSession) definitionLocked(ctx context.Context, absPath, langID, text string, line, character int) ([]lspLocationResult, error) {
	docURI, err := s.ensureOpenLocked(absPath, langID, text)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, lspRequestTimeoutDuration())
	defer cancel()
	raw, err := s.client.call(callCtx, "textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": docURI},
		"position":     map[string]int{"line": line, "character": character},
	})
	if err != nil {
		return nil, fmt.Errorf("lsp definition: %w (%s)", err, strings.TrimSpace(s.stderr.String()))
	}
	out, err := parseDefinitionResult(raw)
	if err != nil {
		return nil, err
	}
	s.lastUsed = time.Now()
	return out, nil
}

func (s *lspSession) referencesLocked(ctx context.Context, absPath, langID, text string, line, character, limit int) ([]lspLocationResult, error) {
	docURI, err := s.ensureOpenLocked(absPath, langID, text)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	callCtx, cancel := context.WithTimeout(ctx, lspRequestTimeoutDuration())
	defer cancel()
	raw, err := s.client.call(callCtx, "textDocument/references", map[string]any{
		"textDocument": map[string]any{"uri": docURI},
		"position":     map[string]int{"line": line, "character": character},
		"context":      map[string]any{"includeDeclaration": true},
	})
	if err != nil {
		return nil, fmt.Errorf("lsp references: %w (%s)", err, strings.TrimSpace(s.stderr.String()))
	}
	out, err := parseDefinitionResult(raw) // Location | Location[] same shape
	if err != nil {
		return nil, err
	}
	if len(out) > limit {
		out = out[:limit]
	}
	s.lastUsed = time.Now()
	return out, nil
}

func parseHoverResult(raw json.RawMessage) (*lspHoverResult, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return &lspHoverResult{}, nil
	}
	var full struct {
		Contents json.RawMessage `json:"contents"`
		Range    *lspRange       `json:"range"`
	}
	if err := json.Unmarshal(raw, &full); err == nil && len(full.Contents) > 0 {
		text, kind := parseMarkedContent(full.Contents)
		return &lspHoverResult{Contents: text, Kind: kind, Range: full.Range}, nil
	}
	// Bare MarkupContent / MarkedString
	text, kind := parseMarkedContent(raw)
	return &lspHoverResult{Contents: text, Kind: kind}, nil
}

func parseMarkedContent(raw json.RawMessage) (text, kind string) {
	var mc struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &mc); err == nil && (mc.Value != "" || mc.Kind != "") {
		return mc.Value, firstNonEmpty(mc.Kind, "markdown")
	}
	var ms struct {
		Language string `json:"language"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(raw, &ms); err == nil && ms.Value != "" {
		return ms.Value, "plaintext"
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		parts := make([]string, 0, len(arr))
		for _, item := range arr {
			t, _ := parseMarkedContent(item)
			if t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n\n"), "markdown"
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, "plaintext"
	}
	return strings.TrimSpace(string(raw)), "plaintext"
}

func parseDefinitionResult(raw json.RawMessage) ([]lspLocationResult, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var one lspLocWire
	if err := json.Unmarshal(raw, &one); err == nil && one.URI != "" {
		return []lspLocationResult{one.toResult()}, nil
	}
	var many []lspLocWire
	if err := json.Unmarshal(raw, &many); err == nil {
		out := make([]lspLocationResult, 0, len(many))
		for _, loc := range many {
			if loc.URI == "" && loc.TargetURI == "" {
				continue
			}
			out = append(out, loc.toResult())
		}
		return out, nil
	}
	return nil, fmt.Errorf("definition decode: unexpected payload")
}

type lspLocWire struct {
	URI       string   `json:"uri"`
	Range     lspRange `json:"range"`
	TargetURI string   `json:"targetUri"`
	TargetSel lspRange `json:"targetSelectionRange"`
	TargetRange lspRange `json:"targetRange"`
}

func (l lspLocWire) toResult() lspLocationResult {
	uri := l.URI
	rng := l.Range
	if uri == "" {
		uri = l.TargetURI
		rng = l.TargetSel
		if rng.Start.Line == 0 && rng.Start.Character == 0 && l.TargetRange.Start.Line != 0 {
			rng = l.TargetRange
		}
	}
	return lspLocationResult{
		URI:       uri,
		Line:      rng.Start.Line + 1,
		Character: rng.Start.Character,
	}
}

func (s *lspSession) closeLocked() {
	if s == nil || s.dead {
		return
	}
	s.dead = true
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = s.client.call(ctx, "shutdown", nil)
	_ = s.client.notify("exit", nil)
	_ = s.stdin.Close()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
}

func (s *lspSession) forceKill() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dead = true
	_ = s.stdin.Close()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
}
