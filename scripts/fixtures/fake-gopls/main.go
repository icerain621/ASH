package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Minimal LSP stub for DX29 tests: initialize + documentSymbol → one Function.
func main() {
	br := bufio.NewReader(os.Stdin)
	for {
		msg, err := readLSP(br)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "fake-gopls read: %v\n", err)
			os.Exit(1)
		}
		var envelope struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}
		switch envelope.Method {
		case "initialize":
			appendEventLog("initialize")
			writeLSP(map[string]any{
				"jsonrpc": "2.0",
				"id":      rawID(envelope.ID),
				"result": map[string]any{
					"capabilities": map[string]any{
						"documentSymbolProvider": true,
						"hoverProvider":          true,
						"definitionProvider":     true,
						"referencesProvider":     true,
					},
					"serverInfo": map[string]any{"name": "fake-gopls", "version": "dx31"},
				},
			})
		case "initialized", "textDocument/didOpen", "textDocument/didChange":
			// notifications — no response
		case "textDocument/documentSymbol":
			appendEventLog("documentSymbol")
			writeLSP(map[string]any{
				"jsonrpc": "2.0",
				"id":      rawID(envelope.ID),
				"result": []map[string]any{{
					"name":   "FixtureLSPFunc",
					"kind":   12, // Function
					"detail": "func",
					"range": map[string]any{
						"start": map[string]int{"line": 6, "character": 0},
						"end":   map[string]int{"line": 6, "character": 20},
					},
					"selectionRange": map[string]any{
						"start": map[string]int{"line": 6, "character": 5},
						"end":   map[string]int{"line": 6, "character": 19},
					},
				}},
			})
		case "textDocument/hover":
			appendEventLog("hover")
			writeLSP(map[string]any{
				"jsonrpc": "2.0",
				"id":      rawID(envelope.ID),
				"result": map[string]any{
					"contents": map[string]any{
						"kind":  "markdown",
						"value": "**FixtureLSPFunc** function",
					},
					"range": map[string]any{
						"start": map[string]int{"line": 6, "character": 5},
						"end":   map[string]int{"line": 6, "character": 19},
					},
				},
			})
		case "textDocument/definition":
			appendEventLog("definition")
			uri := extractURI(envelope.Params)
			if uri == "" {
				uri = "file:///fixture.go"
			}
			writeLSP(map[string]any{
				"jsonrpc": "2.0",
				"id":      rawID(envelope.ID),
				"result": map[string]any{
					"uri": uri,
					"range": map[string]any{
						"start": map[string]int{"line": 6, "character": 5},
						"end":   map[string]int{"line": 6, "character": 19},
					},
				},
			})
		case "textDocument/references":
			appendEventLog("references")
			uri := extractURI(envelope.Params)
			if uri == "" {
				uri = "file:///fixture.go"
			}
			writeLSP(map[string]any{
				"jsonrpc": "2.0",
				"id":      rawID(envelope.ID),
				"result": []map[string]any{
					{
						"uri": uri,
						"range": map[string]any{
							"start": map[string]int{"line": 6, "character": 5},
							"end":   map[string]int{"line": 6, "character": 19},
						},
					},
					{
						"uri": uri,
						"range": map[string]any{
							"start": map[string]int{"line": 10, "character": 1},
							"end":   map[string]int{"line": 10, "character": 8},
						},
					},
				},
			})
		case "shutdown":
			writeLSP(map[string]any{
				"jsonrpc": "2.0",
				"id":      rawID(envelope.ID),
				"result":  nil,
			})
		case "exit":
			return
		default:
			if len(envelope.ID) > 0 && string(envelope.ID) != "null" {
				writeLSP(map[string]any{
					"jsonrpc": "2.0",
					"id":      rawID(envelope.ID),
					"error":   map[string]any{"code": -32601, "message": "method not found: " + envelope.Method},
				})
			}
		}
	}
}

func appendEventLog(event string) {
	path := strings.TrimSpace(os.Getenv("ASH_FAKE_GOPLS_EVENT_LOG"))
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s\n", event)
}

func extractURI(params json.RawMessage) string {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	_ = json.Unmarshal(params, &p)
	return p.TextDocument.URI
}

func rawID(id json.RawMessage) any {
	if len(id) == 0 || string(id) == "null" {
		return nil
	}
	var n float64
	if err := json.Unmarshal(id, &n); err == nil {
		return n
	}
	var s string
	if err := json.Unmarshal(id, &s); err == nil {
		return s
	}
	return json.RawMessage(id)
}

func readLSP(br *bufio.Reader) ([]byte, error) {
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

func writeLSP(v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(raw))
	buf.Write(raw)
	_, _ = os.Stdout.Write(buf.Bytes())
}
