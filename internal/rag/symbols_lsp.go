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
	"sync/atomic"
	"time"
)

const (
	envRAGLSP       = "ASH_RAG_LSP"
	envRAGLSPGopls  = "ASH_RAG_LSP_GOPLS"
	envRAGLSPTS     = "ASH_RAG_LSP_TSSERVER"
	envRAGLSPTimeout = "ASH_RAG_LSP_TIMEOUT_SEC"
	defaultLSPTimeout = 20 * time.Second
)

func lspRequestTimeoutDuration() time.Duration {
	v := strings.TrimSpace(os.Getenv(envRAGLSPTimeout))
	if v == "" {
		return defaultLSPTimeout
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultLSPTimeout
	}
	if n > 300 {
		n = 300
	}
	return time.Duration(n) * time.Second
}

// LSPIndexer extracts symbols via language servers (documentSymbol).
// Default (DX31): workspace-scoped short-lived sessions with idle reclaim.
// Escape hatch ASH_RAG_LSP_SESSION=0 keeps DX29 per-file one-shot.
// Not an IDE daemon — sessions exit on Close / idle timeout.
type LSPIndexer struct {
	goplsPath     string
	tsPath        string
	workspaceRoot string
	pool          *lspSessionPool
}

func NewLSPIndexer(goplsPath, tsPath string) *LSPIndexer {
	return &LSPIndexer{
		goplsPath: strings.TrimSpace(goplsPath),
		tsPath:    strings.TrimSpace(tsPath),
		pool:      sharedLSPSessionPool,
	}
}

func NewLSPIndexerFromEnv() *LSPIndexer {
	return NewLSPIndexer(findGoplsExecutable(), findTypescriptLanguageServer())
}

func (l *LSPIndexer) Name() string { return "lsp" }

// SetWorkspaceRoot pins the LSP rootUri for session reuse (RebuildSymbols).
func (l *LSPIndexer) SetWorkspaceRoot(root string) {
	if l == nil {
		return
	}
	l.workspaceRoot = strings.TrimSpace(root)
}

// Close shuts down pooled sessions for this indexer's workspace root.
func (l *LSPIndexer) Close() error {
	if l == nil || l.pool == nil {
		return nil
	}
	root := l.workspaceRoot
	if root == "" {
		return nil
	}
	l.pool.closeRoot(root)
	return nil
}

// Available reports whether at least one supported language server binary is on PATH.
func (l *LSPIndexer) Available() bool {
	if l == nil {
		return false
	}
	return l.goplsPath != "" || l.tsPath != ""
}

func (l *LSPIndexer) IndexFile(path string, content []byte) ([]SymbolHit, error) {
	if l == nil {
		return nil, fmt.Errorf("lsp indexer is nil")
	}
	ext := strings.ToLower(filepath.Ext(path))
	server, langID, err := l.resolveServer(ext)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root := l.workspaceRoot
	if root == "" {
		root = filepath.Dir(abs)
	}
	ctx, cancel := context.WithTimeout(context.Background(), lspRequestTimeoutDuration())
	defer cancel()
	if lspSessionEnabled() {
		pool := l.pool
		if pool == nil {
			pool = sharedLSPSessionPool
		}
		return pool.documentSymbol(ctx, root, server, abs, langID, string(content))
	}
	return runDocumentSymbol(ctx, server, abs, langID, string(content))
}

func (l *LSPIndexer) resolveServer(ext string) (bin, langID string, err error) {
	switch ext {
	case ".go":
		if l.goplsPath == "" {
			return "", "", fmt.Errorf("lsp: gopls not found")
		}
		return l.goplsPath, "go", nil
	case ".ts", ".tsx":
		if l.tsPath == "" {
			return "", "", fmt.Errorf("lsp: typescript-language-server not found")
		}
		lang := "typescript"
		if ext == ".tsx" {
			lang = "typescriptreact"
		}
		return l.tsPath, lang, nil
	case ".js", ".jsx":
		if l.tsPath == "" {
			return "", "", fmt.Errorf("lsp: typescript-language-server not found")
		}
		lang := "javascript"
		if ext == ".jsx" {
			lang = "javascriptreact"
		}
		return l.tsPath, lang, nil
	default:
		return "", "", fmt.Errorf("lsp: unsupported language %s", ext)
	}
}

func findGoplsExecutable() string {
	if p := strings.TrimSpace(os.Getenv(envRAGLSPGopls)); p != "" {
		return p
	}
	if path, err := exec.LookPath("gopls"); err == nil {
		return path
	}
	return ""
}

func findTypescriptLanguageServer() string {
	if p := strings.TrimSpace(os.Getenv(envRAGLSPTS)); p != "" {
		return p
	}
	for _, name := range []string{"typescript-language-server", "typescript-language-server.cmd"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func parseASHRagLSPEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on", "yes", "auto":
		return true
	default:
		return false
	}
}

func pathToURI(abs string) string {
	abs = filepath.ToSlash(abs)
	if !strings.HasPrefix(abs, "/") {
		// Windows drive path → file:///C:/...
		return "file:///" + abs
	}
	return "file://" + abs
}

func runDocumentSymbol(ctx context.Context, server, absPath, langID, text string) ([]SymbolHit, error) {
	args := []string{}
	base := strings.ToLower(filepath.Base(server))
	if strings.Contains(base, "typescript-language-server") {
		args = append(args, "--stdio")
	}
	cmd := exec.CommandContext(ctx, server, args...)
	cmd.Dir = filepath.Dir(absPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("lsp start %s: %w", filepath.Base(server), err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	client := &lspClient{w: stdin, r: bufio.NewReader(stdout)}
	rootURI := pathToURI(filepath.Dir(absPath))
	docURI := pathToURI(absPath)

	if _, err := client.call(ctx, "initialize", map[string]any{
		"processId": os.Getpid(),
		"rootUri":   rootURI,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"documentSymbol": map[string]any{
					"hierarchicalDocumentSymbolSupport": true,
				},
			},
		},
		"workspaceFolders": []map[string]any{{
			"uri":  rootURI,
			"name": filepath.Base(filepath.Dir(absPath)),
		}},
	}); err != nil {
		return nil, fmt.Errorf("lsp initialize: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	if err := client.notify("initialized", map[string]any{}); err != nil {
		return nil, err
	}
	if err := client.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        docURI,
			"languageId": langID,
			"version":    1,
			"text":       text,
		},
	}); err != nil {
		return nil, err
	}
	raw, err := client.call(ctx, "textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": docURI},
	})
	if err != nil {
		return nil, fmt.Errorf("lsp documentSymbol: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	hits, err := parseDocumentSymbolResult(raw)
	if err != nil {
		return nil, err
	}
	_, _ = client.call(ctx, "shutdown", nil)
	_ = client.notify("exit", nil)
	return hits, nil
}

type lspClient struct {
	w   io.Writer
	r   *bufio.Reader
	seq atomic.Int64
}

func (c *lspClient) nextID() int64 {
	return c.seq.Add(1)
}

func (c *lspClient) write(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(raw))
	buf.Write(raw)
	_, err = c.w.Write(buf.Bytes())
	return err
}

func (c *lspClient) notify(method string, params any) error {
	msg := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		msg["params"] = params
	}
	return c.write(msg)
}

func (c *lspClient) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID()
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if err := c.write(msg); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		body, err := readLSPMessage(c.r)
		if err != nil {
			return nil, err
		}
		var resp struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}
		if resp.Method != "" && len(resp.ID) == 0 {
			continue // server notification
		}
		if !lspIDMatches(resp.ID, id) {
			continue
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc %s: %s", method, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

func lspIDMatches(raw json.RawMessage, want int64) bool {
	if len(raw) == 0 {
		return false
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return int64(n) == want
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s == strconv.FormatInt(want, 10)
	}
	return false
}

func readLSPMessage(br *bufio.Reader) ([]byte, error) {
	headers := map[string]string{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
		}
	}
	n, err := strconv.Atoi(headers["content-length"])
	if err != nil || n < 0 {
		return nil, fmt.Errorf("bad Content-Length")
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(br, body); err != nil {
		return nil, err
	}
	return body, nil
}

func parseDocumentSymbolResult(raw json.RawMessage) ([]SymbolHit, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	// Prefer DocumentSymbol[] (hierarchical).
	var docs []lspDocumentSymbol
	if err := json.Unmarshal(raw, &docs); err == nil && (len(docs) > 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("[]"))) {
		var hits []SymbolHit
		var walk func([]lspDocumentSymbol)
		walk = func(syms []lspDocumentSymbol) {
			for _, s := range syms {
				if name := strings.TrimSpace(s.Name); name != "" {
					line := s.SelectionRange.Start.Line + 1
					if line <= 0 {
						line = s.Range.Start.Line + 1
					}
					hits = append(hits, SymbolHit{Name: name, Kind: lspKindName(s.Kind), Line: line})
				}
				if len(s.Children) > 0 {
					walk(s.Children)
				}
			}
		}
		walk(docs)
		return hits, nil
	}
	var infos []lspSymbolInformation
	if err := json.Unmarshal(raw, &infos); err != nil {
		return nil, fmt.Errorf("documentSymbol decode: %w", err)
	}
	hits := make([]SymbolHit, 0, len(infos))
	for _, s := range infos {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			continue
		}
		hits = append(hits, SymbolHit{
			Name: name,
			Kind: lspKindName(s.Kind),
			Line: s.Location.Range.Start.Line + 1,
		})
	}
	return hits, nil
}

type lspDocumentSymbol struct {
	Name           string              `json:"name"`
	Kind           int                 `json:"kind"`
	Range          lspRange            `json:"range"`
	SelectionRange lspRange            `json:"selectionRange"`
	Children       []lspDocumentSymbol `json:"children"`
}

type lspSymbolInformation struct {
	Name     string `json:"name"`
	Kind     int    `json:"kind"`
	Location struct {
		URI   string   `json:"uri"`
		Range lspRange `json:"range"`
	} `json:"location"`
}

type lspRange struct {
	Start lspPos `json:"start"`
	End   lspPos `json:"end"`
}

type lspPos struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

func lspKindName(kind int) string {
	switch kind {
	case 5:
		return "class"
	case 6:
		return "method"
	case 8:
		return "field"
	case 9:
		return "constructor"
	case 10:
		return "enum"
	case 11:
		return "interface"
	case 12:
		return "func"
	case 13:
		return "var"
	case 14:
		return "const"
	case 23:
		return "type"
	default:
		return "symbol"
	}
}