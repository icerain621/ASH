package rag

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type SymbolHit struct {
	Name string
	Kind string
	Line int
}

type SymbolIndexer interface {
	IndexFile(path string, content []byte) ([]SymbolHit, error)
	Name() string
}

type RegexIndexer struct{}

func (RegexIndexer) Name() string { return "regex" }

func (RegexIndexer) IndexFile(path string, content []byte) ([]SymbolHit, error) {
	var hits []SymbolHit
	for i, line := range splitLines(string(content)) {
		name, kind := matchSymbolLine(line)
		if name == "" {
			continue
		}
		hits = append(hits, SymbolHit{Name: name, Kind: kind, Line: i + 1})
	}
	return hits, nil
}

type CtagsIndexer struct {
	ctagsPath string
}

func NewCtagsIndexer(ctagsPath string) *CtagsIndexer {
	return &CtagsIndexer{ctagsPath: ctagsPath}
}

func (c *CtagsIndexer) Name() string { return "ctags" }

func (c *CtagsIndexer) IndexFile(path string, content []byte) ([]SymbolHit, error) {
	if c == nil || c.ctagsPath == "" {
		return nil, fmt.Errorf("ctags: no executable configured")
	}
	args := []string{"-x", "-f", "-", "--fields=+n", path}
	cmd := ctagsCommand(c.ctagsPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("ctags %s: %s", filepath.Base(path), msg)
	}
	return parseCtagsXOutput(out)
}

func ctagsCommand(ctagsPath string, args ...string) *exec.Cmd {
	if strings.HasSuffix(strings.ToLower(ctagsPath), ".sh") {
		return exec.Command("bash", append([]string{ctagsPath}, args...)...)
	}
	return exec.Command(ctagsPath, args...)
}

func parseCtagsXOutput(out []byte) ([]SymbolHit, error) {
	var hits []SymbolHit
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		lineNum, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		hits = append(hits, SymbolHit{Name: fields[0], Kind: fields[1], Line: lineNum})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return hits, nil
}

// ResolveSymbolIndexer picks a SymbolIndexer.
//
// Priority when ASH_RAG_SYMBOL_INDEXER unset:
//
//	ASH_RAG_CTAGS=0  → regex (DX16 compat)
//	ASH_RAG_CTAGS=1  → ctags if on PATH, else regex
//	ASH_RAG_LSP=1    → lsp if gopls/tsserver on PATH, else treesitter
//	auto (default)   → treesitter (pure Go; always available)
//
// ASH_RAG_SYMBOL_INDEXER=lsp|treesitter|ctags|regex forces one backend.
func ResolveSymbolIndexer() SymbolIndexer {
	switch forced := strings.ToLower(strings.TrimSpace(os.Getenv("ASH_RAG_SYMBOL_INDEXER"))); forced {
	case "lsp":
		return NewLSPIndexerFromEnv()
	case "treesitter", "tree-sitter", "ts":
		return NewTreeSitterIndexer()
	case "ctags":
		if path := findCtagsExecutable(); path != "" {
			return NewCtagsIndexer(path)
		}
		return RegexIndexer{}
	case "regex":
		return RegexIndexer{}
	}

	switch parseASHRagCtagsEnv(os.Getenv("ASH_RAG_CTAGS")) {
	case ragCtagsOff:
		return RegexIndexer{}
	case ragCtagsOn:
		if path := findCtagsExecutable(); path != "" {
			return NewCtagsIndexer(path)
		}
		return RegexIndexer{}
	default:
		if parseASHRagLSPEnv(os.Getenv(envRAGLSP)) {
			if idx := NewLSPIndexerFromEnv(); idx.Available() {
				return idx
			}
		}
		return NewTreeSitterIndexer()
	}
}

type ragCtagsMode int

const (
	ragCtagsAuto ragCtagsMode = iota
	ragCtagsOff
	ragCtagsOn
)

func parseASHRagCtagsEnv(v string) ragCtagsMode {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "auto":
		return ragCtagsAuto
	case "0", "false", "off", "no":
		return ragCtagsOff
	case "1", "true", "on", "yes":
		return ragCtagsOn
	default:
		return ragCtagsAuto
	}
}

func findCtagsExecutable() string {
	if p := strings.TrimSpace(os.Getenv("CTAGS")); p != "" {
		return p
	}
	for _, name := range []string{"ctags", "universal-ctags"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}
